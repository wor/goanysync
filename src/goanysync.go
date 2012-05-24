// vim: set sts=4 ts=4 sw=4 et foldmethod=marker:
// Formated with: gofmt -w=true -tabwidth=4 -tabs=false

// main package of goanysync program by Esa Määttä <esa.maatta AT iki DOT fi>.
// Inspired by anything-sync-daemon written by graysky <graysky AT archlinux DOT us>
// Should be drop-in-replacement functionally wise, though doesn't use exactly same
// config file syntax.
package main


import (
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "os/exec"
    "path"
    "regexp"
    "syscall"
)


// Checks whether given file name exists.
func exists(fn string) bool { // {{{
    if _, err := os.Stat(fn); err != nil {
        return !os.IsNotExist(err)
    }
    return true
} // }}}

// Returns owner user and group ids from given FileInfo
func getFileUserAndGroupId(fi os.FileInfo) (uid uint, gid uint, err error) { // {{{
    if st, ok := fi.Sys().(*syscall.Stat_t); ok {
        return uint(st.Uid), uint(st.Gid), nil
    }
    err = errors.New("Stat failed on: " + fi.Name())
    return
} // }}}

// Checks whether given path name "s" is valid source for sync. Returns
// necessary information for sync/unsync function about "s".
func isValidSource(s string) (fi os.FileInfo, uid uint, gid uint, err error) { // {{{
    if fi, err = os.Stat(s); err != nil {
        return
    }

    if !fi.IsDir() {
        err = errors.New("Sync source path was not a directory: " + s)
        return
    }

    if uid, gid, err = getFileUserAndGroupId(fi); err != nil {
        return
    }

    return
} // }}}

// Generates volatile and backup path names and a regex string for matching
// volatile path name.
func pathNameGen(s string, tmpfs string, uid uint, gid uint) (volatilePath string, backupPath string, volatilePathRe string) { // {{{
    volatilePrefix := path.Join(tmpfs, "goanysync-")
    const backupPostfix  string = "-backup_goanysync"

    volatileBasePathRe := fmt.Sprintf("%s[0-9]+:[0-9]+", volatilePrefix)
    volatilePathRe = path.Join(volatileBasePathRe, s)

    volatileBasePath := fmt.Sprintf("%s%d:%d", volatilePrefix, uid, gid)
    volatilePath = path.Join(volatileBasePath, s)

    backupPath = s + backupPostfix
    return
} // }}}

// --------------------------------------------------------------------------

// Checks if any sync sources where synced but not finally unsynced. Restores
// such sources from backup path to orginal state.
func checkAndFix(tmpfs string, syncSources *[]string) { // {{{
    for _, s := range *syncSources {
        _, backupPath, volatilePathRe := pathNameGen(s, tmpfs, 0, 0)

        vpMatch := func(p string, s string) bool {
            var match bool
            var err error
            if match, err = regexp.MatchString(p, s); err != nil {
                panic("Regexp matching error: " + err.Error())
            }
            return match
        }
        // Check if sync has already been called but tmpfs copy has been
        // deleted. This happens for example if computer boots before unsync is
        // called. In this case the 's' path is a broken symlink to the
        // volatilePath and the backupPath exists.
        if target, err := os.Readlink(s); err == nil && vpMatch(volatilePathRe, target) && !exists(target) && exists(backupPath) {
            os.Remove(s)
            os.Rename(backupPath, s)
        }
    }
    return
} // }}}

// sync replaces given paths in syncSources with symlinks to directories
// under given tmpfs path. Also it creates a backup directory for every
// syncSource path. If sync is called consecutively for same syncSources it
// syncs content from tmpfs paths to backup paths.
func sync(tmpfs string, syncSources *[]string, syncerBin string) { // {{{
    for _, s := range *syncSources {
        var (
            fi       os.FileInfo
            uid, gid uint
            err      error
        )

        if fi, uid, gid, err = isValidSource(s); err != nil {
            log.Printf("sync error: %s\n... Skipping path: %s", err, s)
            continue
        }

        // Volatile dirs name is based on orginal dir's name, uid and gid
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // First check if our target directory in tmpfs is ready.
        // We must ensure that the orginal owner of the source directory can
        // read the tmpfs volatile target dir, so we use the orginals
        // permissions.
        if err := os.MkdirAll(volatilePath, fi.Mode()); err != nil { // {{{
            log.Printf("sync error (volatile path creation): %s\n... Skipping path: %s", err, s)
            continue
        }   // }}}

        // Second check if we need to create initial backup and initial sync to
        // volatile
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            // TODO: don't ignore errors
            os.Rename(s, backupPath)
            if linkError := os.Symlink(volatilePath, s); linkError != nil {
                log.Printf("sync error (symlink): %s\n... Skipping path: %s", err, s)
                os.Rename(backupPath, s)
                continue
            }
            // Let's do initial sync to volatile
            cmd := exec.Command(syncerBin, "-a", backupPath + "/", s)
            if err := cmd.Run(); err != nil {
                log.Printf("sync error (volatile): %s\n... With command: %s\n... Skipping path: %s", err, cmd, s)
                os.Rename(backupPath, s)
            }
            continue
        }   // }}}

        // Everything was ready so we just sync from volatile tmpfs to backup
        cmd := exec.Command(syncerBin, "-a", "--delete", s + "/", backupPath)
        if err := cmd.Run(); err != nil { // {{{
            log.Printf("sync error (backup): %s\n... With command: %s\n... Sync to backup failed for: %s", err, cmd, s)
            continue
        }   // }}}
    }
    return
}   // }}}

// unsync removes symbolic linkin to tmpfs and restores orginal from backup
func unsync(tmpfs string, syncSources *[]string, removeVolatile bool) { // {{{
    for _, s := range *syncSources {
        var (
            uid, gid uint
            err      error
        )
        if _, uid, gid, err = isValidSource(s); err != nil {
            log.Printf("unsync error: %s\n... Skipping path: %s", err, s)
            continue
        }
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // Check that backup path exists and is a directory
        if fi, err := os.Stat(backupPath); err != nil || !fi.IsDir() { // {{{
            log.Printf("unsync error (backup): %s\n... Skipping path: %s", err, s)
            continue
        }   // }}}

        // Check that "s" was symlink to the volatile path
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            log.Printf("unsync error (volatile): %s\n... Skipping path: %s", err, s)
            continue
        }   // }}}

        // Remove the link and replace it with backup
        // TODO: don't ignore errors
        os.Remove(s)
        os.Rename(backupPath, s)

        // XXX: Is there any reason to remove volatile target? Any other than
        // saving space?
        if removeVolatile {
            os.RemoveAll(volatilePath)
        }
    }
    return
} // }}}

func main() {
    const errorMessage string = "Error: use this program through rc.d wrapper (addvisable), or provide valid command."
    // Check that at least one argment given
    if len(os.Args) < 2 {
        log.Fatalln(errorMessage)
    }
    configFilePath := flag.String("c", "/etc/goanysync.conf", "Config file.")
    verbose := *flag.Bool("v", false, "Be more verbose.")
    // TODO: write command infos
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage of %s %s:\n", os.Args[0], "[options] <command>")
        fmt.Fprintf(os.Stderr, "  Commands:\n")
        fmt.Fprintf(os.Stderr, "   sync\n")
        fmt.Fprintf(os.Stderr, "   unsync\n")
        fmt.Fprintf(os.Stderr, "   check\n")
        fmt.Fprintf(os.Stderr, "  Options:\n")
        flag.PrintDefaults()
    }
    flag.Parse()

    copts, err := ReadConfigFile(*configFilePath)
    if err != nil {
        log.Fatalln("Config file error:", err)
    }

    if verbose {
        copts.Print()
    }

    switch flag.Arg(0) {
    case "check":
        checkAndFix(copts.tmpfsPath, &copts.syncPaths)
    case "sync":
        sync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin)
    case "unsync":
        unsync(copts.tmpfsPath, &copts.syncPaths, false)
    default:
        log.Fatalln(errorMessage)
    }
    return
}
