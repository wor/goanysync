// Copyright (C) 2012 Esa Määttä <esa.maatta@iki.fi>
// This file is released under the GNU GPL, version 3 or a later revision.
// For further details see the COPYING file.

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
    "log/syslog"
    "os"
    "os/exec"
    "path"
    "regexp"
    "syscall"
    "time"
    "strings"
)

// global logger
var LOG *syslog.Writer

// should I be chatty?
var VERBOSE bool


var ERR = func (msg string) (error, string) {return LOG.Err(msg), "ERR" }
var INFO = func (msg string) (error, string) {return LOG.Info(msg), "INFO" }
var CRIT = func (msg string) (error, string) {return LOG.Crit(msg), "CRIT" }
var DEBUG = func (msg string) (error, string) {return LOG.Debug(msg), "DEBUG" }

// The Logger() takes one of the previously defined logging methods and
// wraps the msg directly to the regular logger mechanism, if VERBOSE is not set.
// Otherwise Logger() prints the messages to the stdout in order to be verbose
func Logger(method func(msg string) (error, string) , wrap_msg string) error {
	ret, funcname := method(wrap_msg)
	if VERBOSE {
		lines := strings.Split(strings.TrimSpace(wrap_msg), "\n")
		i := 0
        for i<len(lines) {
            out := fmt.Sprintf("%7s %s", "["+funcname+"]", strings.TrimSpace(lines[i]))
			fmt.Println(out)
            i++
        }
	}
	return ret
}

// mkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits perm are used
// for all directories that mkdirAll creates. Also given uid and gid are set. If
// path is already a directory, mkdirAll does nothing and returns nil.
//
// This function is a copy of os.MkdirAll with uid and gid setting.
//
// TODO: this version should check and ensure that given perm is set for all
//       path parts
func mkdirAll(path string, perm os.FileMode, uid uint, gid uint) error { // {{{
    // If path exists, stop with success or error.
    dir, err := os.Stat(path)
    if err == nil {
        if dir.IsDir() {
            return nil
        }
        return &os.PathError{"mkdir", path, syscall.ENOTDIR}
    }

    // Doesn't already exist; make sure parent does.
    i := len(path)
    for i > 0 && os.IsPathSeparator(path[i-1]) { // Skip trailing path separator.
        i--
    }

    j := i
    for j > 0 && !os.IsPathSeparator(path[j-1]) { // Scan backward over element.
        j--
    }

    if j > 1 {
        // Create parent
        err = mkdirAll(path[0:j-1], perm, uid, gid)
        if err != nil {
            return err
        }
    }

    // Now parent exists, try to create.
    err = os.Mkdir(path, perm)
    if err != nil {
        // Handle arguments like "foo/." by
        // double-checking that directory doesn't exist.
        dir, err1 := os.Lstat(path)
        if err1 == nil && dir.IsDir() {
            return nil
        }
        return err
    }
    // Change user and group id
    if err1 := os.Chown(path, int(uid), int(gid)); err1 != nil {
        return err1
    }
    return nil
}   // }}}

// exists checks whether given file name exists.
func exists(fn string) bool { // {{{
    if _, err := os.Stat(fn); err != nil {
        return !os.IsNotExist(err)
    }
    return true
}   // }}}

// getFileUserAndGroupId returns owner user and group ids from given FileInfo.
func getFileUserAndGroupId(fi os.FileInfo) (uid uint, gid uint, err error) { // {{{
    if st, ok := fi.Sys().(*syscall.Stat_t); ok {
        return uint(st.Uid), uint(st.Gid), nil
    }
    err = errors.New("Stat failed on: " + fi.Name())
    return
}   // }}}

// isValidSource checks whether given path name "s" is valid source for sync.
// Returns necessary information for sync/unsync function about "s".
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
}   // }}}

// pathNameGen generates volatile and backup path names and a regex string for
// matching volatile path name.
func pathNameGen(s string, tmpfs string, uid, gid uint) (volatilePath, backupPath, volatilePathRe string) { // {{{
    volatilePrefix := path.Join(tmpfs, "goanysync-")
    const backupPostfix string = "-backup_goanysync"

    volatileBasePathRe := fmt.Sprintf("%s[0-9]+:[0-9]+", volatilePrefix)
    volatilePathRe = path.Join(volatileBasePathRe, s)

    volatileBasePath := fmt.Sprintf("%s%d:%d", volatilePrefix, uid, gid)
    volatilePath = path.Join(volatileBasePath, s)

    backupPath = s + backupPostfix
    return
}   // }}}

// getLock acquires the file lock.
func getLock(lockName string) (bool, error) { // {{{
    err := os.Mkdir(lockName, 0600)
    if err != nil {
        if os.IsExist(err) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}   // }}}

// releaseLock releases the file lock.
func releaseLock(lockName string) { // {{{
    if err := os.Remove(lockName); err != nil {
        lmsg := fmt.Sprintf("releaseLock error: %s\n... This should not happen, panicing..", err)
        LOG.Crit(lmsg)
        panic(err)
    }
}   // }}}

// --------------------------------------------------------------------------

// checkAndFix checks if any sync sources where synced but not finally unsynced.
// Restores such sources from backup path to original state.
func checkAndFix(tmpfs string, syncSources *[]string) { // {{{
    Logger(DEBUG, "Checking for inconsistencies...")
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
}   // }}}

// initSync does initial preparation for syncing and if preparations already
// done it does nothing so it should be safe to call in any case. Initial
// preparation incorporates following acts: 1. Replacement of given paths in
// syncSources with symlinks to directories under given tmpfs path. 2. Creation
// of a backup directory for every syncSource path.
func initSync(tmpfs string, syncSources *[]string, syncerBin string) { // {{{
	Logger(DEBUG, "Starting initial sync run...")
    for _, s := range *syncSources {
        var (
            fi        os.FileInfo
            uid, gid  uint
            err       error
        )

        // Create initial tmpfs base dir right permissions
        if err := os.Mkdir(tmpfs, 0777); err != nil {
            if os.IsExist(err) {
                if err := os.Chmod(tmpfs, 0777); err != nil {
                    lmsg := fmt.Sprintf("initSync error: Changing permissions of tmpfs dir '%s' failed...: %s", tmpfs, err)
                    LOG.Err(lmsg)
                    return
                }
            } else {
                lmsg := fmt.Sprintf("initSync error: Creation of tmpfs dir '%s' failed...: %s", tmpfs, err)
                LOG.Err(lmsg)
                return
            }
        }

        if fi, uid, gid, err = isValidSource(s); err != nil {
            lmsg := fmt.Sprintf("initSync error: %s\n... Skipping path: %s", err, s)
            Logger(ERR, lmsg)
            continue
        }

        // Volatile dirs name is based on orginal dir's name, uid and gid
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // First check if our target directory in tmpfs is ready.
        // We must ensure that the original owner of the source directory can
        // read the tmpfs volatile target dir, so we use the originals
        // permissions.
        if err := mkdirAll(volatilePath, fi.Mode(), uid, gid); err != nil { // {{{
            lmsg := fmt.Sprintf("initSync error (volatile path creation): %s\n... Skipping path: %s", err, s)
            Logger(ERR, lmsg)
            continue
        }   // }}}

        // Second check if we need to create initial backup and initial sync to
        // volatile
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            // trying to rename the target path
            err2 := os.Rename(s, backupPath)
            if err2 != nil {
                lmsg := fmt.Sprintf("could not rename target path: %s\n... Skipping path: %s", err2, s)
                Logger(ERR, lmsg)
                continue
            }
            // create symlink from original path to volatile path
            if linkError := os.Symlink(volatilePath, s); linkError != nil {
                lmsg := fmt.Sprintf("initSync error (symlink): %s\n... Skipping path: %s", err, s)
                Logger(ERR, lmsg)
                os.Rename(backupPath, s)
                continue
            }
            // Let's do initial sync to volatile
            cmd := exec.Command(syncerBin, "-a", "--delete", backupPath+"/", s)
            if err := cmd.Run(); err != nil {
                lmsg := fmt.Sprintf("initSync error (volatile): %s\n... With command: %s\n... Skipping path: %s", err, cmd, s)
                Logger(ERR, lmsg)
                os.Rename(backupPath, s)
            }
            continue
        } else {
            lmsg := fmt.Sprintf("initSync info: sync path was already initialized: %s\n", s)
            Logger(DEBUG, lmsg)
        }   // }}}
    }
    return
}   // }}}

// sync syncs content from tmpfs paths to backup paths. It expects that initSync
// has been called for the syncSources.
func sync(tmpfs string, syncSources *[]string, syncerBin string) { // {{{
    Logger(DEBUG, "Starting sync...")
    for _, s := range *syncSources {
        var (
            uid, gid uint
            err      error
        )

        if _, uid, gid, err = isValidSource(s); err != nil {
            lmsg := fmt.Sprintf("sync error: %s\n... Skipping path: %s", err, s)
            Logger(ERR, lmsg)
            continue
        }

        // Volatile dirs name is based on orginal dir's name, uid and gid
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // Volatile path must exists
        if !exists(volatilePath) {
            // syncInit failed or not called for the sync path
            lmsg := fmt.Sprintf("sync error (volatile path did not exist): %s\n... Skipping path: %s", volatilePath, s)
            Logger(ERR, lmsg)
            continue
        }

        // Target must be a symlink to the volatile path
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            lmsg := fmt.Sprintf("sync error (volatile path was not linked): %s\n... Skipping path: %s", err, s)
            Logger(ERR, lmsg)
            continue
        }   // }}}

        // Backup path must exists
        if !exists(backupPath) {
            // syncInit failed or not called for the sync path
            lmsg := fmt.Sprintf("sync error (backup path did not exist): %s\n... Skipping path: %s", backupPath, s)
            Logger(ERR, lmsg)
            continue
        }

        // Everything was ok, so we just sync from volatile tmpfs to backup
        cmd := exec.Command(syncerBin, "-a", "--delete", s+"/", backupPath)
        if err := cmd.Run(); err != nil { // {{{
            lmsg := fmt.Sprintf("sync error (backup): %s\n... With command: %s\n... Sync to backup failed for: %s", err, cmd, s)
            Logger(ERR, lmsg)
            continue
        }   // }}}

        lmsg := fmt.Sprintf("sync: synced dir '%s'.", s)
        Logger(DEBUG, lmsg)
    }
    return
}   // }}}

// unsync removes symbolic linkin to tmpfs and restores original from backup.
func unsync(tmpfs string, syncSources *[]string, removeVolatile bool) { // {{{
    Logger(DEBUG, "Starting unsync...")
    for _, s := range *syncSources {
        var (
            uid, gid uint
            err      error
        )
        if _, uid, gid, err = isValidSource(s); err != nil {
            lmsg := fmt.Sprintf("unsync error: %s\n... Skipping path: %s", err, s)
            Logger(ERR, lmsg)
            continue
        }
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // Check that backup path exists and is a directory
        if fi, err := os.Stat(backupPath); err != nil || !fi.IsDir() { // {{{
            lmsg := fmt.Sprintf("unsync error (backup): %s\n... Skipping path: %s", err, s)
            Logger(ERR, lmsg)
            continue
        }   // }}}

        // Check that "s" was symlink to the volatile path
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            lmsg := fmt.Sprintf("unsync error (volatile): %s\n... Skipping path: %s", err, s)
            Logger(ERR, lmsg)
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
}   // }}}

// checkLockFileDir checks if directory which contains the lock file exists and
// has right permissions and owner.
func checkLockFileDir(dir string) (err error) { // {{{
    var fi os.FileInfo

    if fi, err = os.Stat(dir); err != nil {
        return
    }

    if !fi.IsDir() {
        err = errors.New("Lock files parent dir was not a directory: " + dir)
        return
    }

    if fi.Mode().Perm() != 0755 {
        err = errors.New("Lock file parent dir did not have right permissions != 755")
        return
    }

    var uid uint
    if uid, _, err = getFileUserAndGroupId(fi); err != nil {
        return
    }

    if uid != 0 {
        err = errors.New("Lock file parent dir was not root owned.")
        return
    }
    return
}   // }}}

// runMain is a main function which returns programs exit value.
func runMain() int {
    var err error
    LOG, err = syslog.New(syslog.LOG_INFO, "goanysync")
    if err != nil {
        log.Println("Error: Syslog logger initialization failed with error:", err)
        return 1
    }
    defer LOG.Close()

    const errorMessage string = "Error: invalid command provided."
    // Check that at least one argument given
    if len(os.Args) < 2 {
        log.Println(errorMessage)
        return 1
    }
    configFilePath := flag.String("c", "/etc/goanysync.conf", "Config file.")
    verbose := flag.Bool("v", false, "Be more verbose.")
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage of %s %s:\n", os.Args[0], "[options] <command>")
        fmt.Fprintf(os.Stderr, "  Commands:\n")
        fmt.Fprintf(os.Stderr, "   initsync\tReplaces sync directories with symlinks to tmpfs while syncing orginal content there.\n")
        fmt.Fprintf(os.Stderr, "   sync\t\tSyncs content from tmpfs to the backup.\n")
        fmt.Fprintf(os.Stderr, "   unsync\tRestores orginal state of sync directories.\n")
        fmt.Fprintf(os.Stderr, "   check\tChecks if sync was called without unsync before tmpfs was cleared.\n")
        fmt.Fprintf(os.Stderr, "   start\tAlias for running check and initsync.\n")
        fmt.Fprintf(os.Stderr, "   stop\t\tAlias for running sync and unsync.\n")
        fmt.Fprintf(os.Stderr, "  Options:\n")
        flag.PrintDefaults()
    }
    flag.Parse()

    // Read config file
    copts, err := ReadConfigFile(*configFilePath)
    if err != nil {
        Logger(CRIT, "Config file error: " + err.Error())
        return 1
    }

    if *verbose {
        copts.Print()
        VERBOSE = true
    }

    // For now do not allow synchronous calls at all.
    // "/run/goanysync" is path is configured in tmpfiles.d and should be only
    // root writable.
    const processLockFile string = "/run/goanysync/process.lock"
    // Check that lock files base path
    if err = checkLockFileDir(path.Dir(processLockFile)); err != nil {
        Logger(CRIT, "Lock file path error: " + err.Error())
        return 1
    }

    // Locking to prevent synchronous operations
    for ok, err := getLock(processLockFile); !ok; ok, err = getLock(processLockFile) {
        if err != nil {
            Logger(CRIT, "Lock file error: " + err.Error())
            return 1
        }
        // TODO: specify max wait time
        // TODO: use inotify when go provides an interface for it
        time.Sleep(time.Millisecond * 100)
    }
    // If os.Exit() is called remember to remove the lock file, manually.
    defer releaseLock(processLockFile)

    switch flag.Arg(0) {
    case "check":
        checkAndFix(copts.tmpfsPath, &copts.syncPaths)
    case "initsync":
        initSync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin)
    case "sync":
        sync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin)
    case "unsync":
        unsync(copts.tmpfsPath, &copts.syncPaths, false)
    case "start":
        checkAndFix(copts.tmpfsPath, &copts.syncPaths)
        initSync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin)
    case "stop":
        sync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin)
        unsync(copts.tmpfsPath, &copts.syncPaths, false)
    default:
        log.Println(errorMessage)
        fmt.Println()
        flag.Usage()
        return 1
    }
    return 0
}

func main() {
    os.Exit(runMain())
}

// vim: set sts=4 ts=4 sw=4 et foldmethod=marker:
