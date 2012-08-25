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
    wl "goanysync/log"
    "log/syslog"
    "math"
    "os"
    "os/exec"
    "path"
    "path/filepath"
    "regexp"
    "strings"
    "syscall"
    "time"
)

// Global logger
var LOG *wl.Log

const (
    VOLATILE_BASE_PREFIX = "goanysync-"
    VOLATILE_BASE_RE     = VOLATILE_BASE_PREFIX + "[0-9]+-[0-9]+"
    VOLATILE_BASE        = VOLATILE_BASE_PREFIX + "%d-%d"
    BACKUP_POSTFIX       = "-backup_goanysync"
)

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
    if fi, uid, gid, err = getFileInfo(s); err != nil {
        return
    }

    if !fi.IsDir() {
        err = errors.New("Sync source path was not a directory: " + s)
        return
    }
    return
}   // }}}

// getFileInfo returns given files FileInfo, user id and group id and possibly
// an error.
func getFileInfo(fn string) (fi os.FileInfo, uid uint, gid uint, err error) { // {{{
    if fi, err = os.Stat(fn); err != nil {
        return
    }

    if uid, gid, err = getFileUserAndGroupId(fi); err != nil {
        return
    }
    return
}   // }}}

// Generate regex to identify base volatile paths.
func getVolatileBasePathRe(tmpfs string) (re string) { // {{{
    return path.Join(tmpfs, VOLATILE_BASE_RE)
}   // }}}

// Generate backup path for sync source
func getBackupPath(syncSource string) string { // {{{
    return syncSource + BACKUP_POSTFIX
}   // }}}

// pathNameGen generates volatile and backup path names and a regex string for
// matching volatile path name.
func pathNameGen(s string, tmpfs string, uid, gid uint) (volatilePath, backupPath, volatilePathRe string) { // {{{
    //volatilePrefix := path.Join(tmpfs, VOLATILE_BASE_PREFIX)

    volatileBasePathRe := getVolatileBasePathRe(tmpfs)

    //volatileBasePathRe := fmt.Sprintf("%s[0-9]+-[0-9]+", volatilePrefix)
    volatilePathRe = path.Join(volatileBasePathRe, s)

    volatileBasePath := path.Join(tmpfs, fmt.Sprintf(VOLATILE_BASE, uid, gid))
    //volatileBasePath := fmt.Sprintf("%s%d-%d", volatilePrefix, uid, gid)
    volatilePath = path.Join(volatileBasePath, s)

    backupPath = getBackupPath(s)
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
        LOG.Emerg("releaseLock: %s", err)
        panic(err)
    }
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

    // If process efective user id is root then add additional checks
    if os.Geteuid() == 0 {
        var uid, _ uint
        if uid, _, err = getFileUserAndGroupId(fi); err != nil {
            return
        }

        if uid != 0 {
            err = errors.New("Lock file parent dir was not root owned.")
            return
        }

        if fi.Mode().Perm()&0022 != 0 {
            err = errors.New("Lock file parent dir did not have right permissions: others than the owner had write permissions.")
            return
        }
    }
    return
}   // }}}

// Checks if volatile TMPFS path contains paths not specified in syncSources.
// Returns first such path found.
func checkVolatileForExtra(tmpfs string, syncSources *[]string, onlyFirst bool) (ok bool, extraPaths *[]string, extraBackupPaths *[]string, err error) { // {{{
    volatileBasePathRe := getVolatileBasePathRe(tmpfs)
    vbpRE := regexp.MustCompile(volatileBasePathRe)
    foundExtraSyncSources := make([]string, 0, 100)
    foundPathsWithBackups := make([]string, 0, 100)
    stopError := errors.New("Stopped filewalk normally.")

    // Helper function to remove tmpfs path prefix from given path
    trimTmpfsPrefix := func(path string) string {
        loc := vbpRE.FindStringIndex(path)
        if loc == nil {
            panicMsg := fmt.Sprintf("trimTmpfsPrefix: '%s' regex did not match '%s'\n", volatileBasePathRe, path)
            panic(panicMsg)
        }
        return path[loc[1]:]
    }

    // Path walker function for checking existing backup paths and symlinked
    // targets.
    wfBackupLinkChecker := func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() {
            return nil
        }
        fullPath := path
        path = trimTmpfsPrefix(path)
        backupPath := getBackupPath(path)

        // If backup path exists and path is a symlink which points to tmpfs
        // path or is a broken symlink. This check could be more comprehensive
        // but it should now cover the most usual cases.
        if target, err := os.Readlink(path); exists(backupPath) && err == nil && (target == fullPath || !exists(target)) {
            foundPathsWithBackups = append(foundPathsWithBackups, fullPath)
            return filepath.SkipDir
        }
        return nil
    }

    // Path walker function
    wf := func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() {
            return nil
        }
        match, rerr := regexp.MatchString(volatileBasePathRe, path)
        if rerr != nil {
            panic("checkVolatileForExtra: Regexp matching error: " + rerr.Error())
        }
        if match {
            fullPath := path
            path = trimTmpfsPrefix(path)
            failCount := 0
            for _, ss := range *syncSources {
                if path == ss {
                    return filepath.SkipDir
                } else if !strings.HasPrefix(path, ss) && !strings.HasPrefix(ss, path) {
                    failCount++
                } else {
                    break // path was part of a sync source path
                }
            }
            if failCount >= len(*syncSources) {
                err := filepath.Walk(fullPath, wfBackupLinkChecker)
                if err != nil {
                    LOG.Debug("checkVolatileForExtra: walk returned error_: %s\n", err)
                }
                foundExtraSyncSources = append(foundExtraSyncSources, fullPath)
                if onlyFirst {
                    return stopError
                }
                return filepath.SkipDir
            }
        }
        return nil
    }

    err = filepath.Walk(tmpfs, wf)
    if err != nil && err != stopError {
        LOG.Debug("checkVolatileForExtra: walk returned error: %s\n", err)
    } else {
        err = nil
    }
    // Set other return values
    ok = len(foundExtraSyncSources) == 0 && len(foundPathsWithBackups) == 0
    extraPaths = &foundExtraSyncSources
    extraBackupPaths = &foundPathsWithBackups
    return
}   // }}}

// checkVolatile checks volatile TMPFS path for extra paths not in sync
// sources. It doesn't do anything for an empty tmpfs path.
func checkVolatile(tmpfsPath string, syncPaths *[]string) (ok bool) { // {{{
    if !exists(tmpfsPath) {
        return true
    }
    if ok, extraPaths, extraBackupPaths, err := checkVolatileForExtra(tmpfsPath, syncPaths, true); !ok || err != nil {
        if err != nil {
            LOG.Err("Volatile (TMPFS) directory checker returned an error: %s\n", err)
        } else {
            foundBackupSubdirs := ""
            if len(*extraBackupPaths) > 0 {
                for i, s := range *extraBackupPaths {
                    foundBackupSubdirs = foundBackupSubdirs + s
                    if i < len(*extraBackupPaths) {
                        foundBackupSubdirs = foundBackupSubdirs + ", "
                    }
                }
            }
            LOG.Err("TMPFS path contained volatile path(s) not in WHATTOSYNC: %s: %s\n", (*extraPaths)[0], foundBackupSubdirs)
        }
        return false
    }
    return true
}   // }}}

// --------------------------------------------------------------------------

// info shows currently used space and what and where data is stored and
// synced. Also it tells if there is extra paths in the TMPFS directory which
// are not in current WHATTOSYNC path list.
func info(copts *ConfigOptions) { // {{{
    var ( // {{{
        target     string
        uid, gid   uint
        err        error
        bgRed      = "\x1b[41m"
        reset      = "\x1b[0m"
        colorStart string
        colorEnd   string
        totalSize  int64
    )   // }}}

    fmt.Printf("Current base TMPFS path is: %s\n", copts.tmpfsPath)
    fmt.Printf("Sync path info:\n")
    for i, s := range copts.syncPaths {
        if _, uid, gid, err = isValidSource(s); err != nil {
            fmt.Printf("  %s\n", err)
            continue
        }
        ss, backupPath, _ := pathNameGen(s, copts.tmpfsPath, uid, gid)

        colorStart, colorEnd = "", ""
        targetStr := " -> not a symlink."
        if target, err = os.Readlink(s); err == nil {
            targetStr = " -> " + target
        }
        if target != ss {
            colorStart, colorEnd = bgRed, reset
        }
        fmt.Printf("%d. Sync path: %s%s%s%s\n", i, colorStart, s, targetStr, colorEnd)

        var size int64
        colorStart, colorEnd = "", ""
        if !exists(ss) {
            colorStart, colorEnd = bgRed, reset
        } else {
            wf := func(path string, info os.FileInfo, err error) error {
                if err == nil {
                    size = size + info.Size()
                }
                return nil
            }
            err = filepath.Walk(ss, wf)
            // Convert size to MB rounding up
            size = int64(math.Floor(float64(size)/(1024*1024) + 0.5))
            totalSize = totalSize + size
        }
        fmt.Printf("  tmpfs path  : %s%s%s\n", colorStart, ss, colorEnd)
        if size != 0 {
            fmt.Printf("  tmpfs size  : %dM\n", size)
        }

        colorStart, colorEnd = "", ""
        if !exists(backupPath) {
            colorStart = bgRed
            colorEnd = reset
        }
        fmt.Printf("  backup path : %s%s%s\n", colorStart, backupPath, colorEnd)
    }
    fmt.Printf("---------- Total space of TMPFS used: %dM\n", totalSize)

    if ok, extraPaths, extraBackupPaths, err := checkVolatileForExtra(copts.tmpfsPath, &copts.syncPaths, false); !ok || err != nil {
        if err != nil {
            fmt.Printf("TMPFS directory checker returned an error: %s\n", err)
        } else {
            fmt.Println()
            if len(*extraPaths) > 0 {
                fmt.Printf("TMPFS contained paths which were not in WHATTOSYNC paths:\n\n")
            }
            for _, s := range *extraPaths {
                fmt.Printf("  %s\n", s)
            }
            if len(*extraBackupPaths) > 0 {
                fmt.Printf("\nAlso found paths which matching target backup paths existed. The target path was a broken symlink or a symlink to the TMPFS. You should carefully check these paths and remove them from TMPFS afterwards.\n\n")
            }
            for _, s := range *extraBackupPaths {
                fmt.Printf("  %s\n", s)
            }
        }
    }

}   // }}}

// checkAndFix checks if any sync sources where synced but not finally unsynced.
// Restores such sources from backup path to original state.
func checkAndFix(tmpfs string, syncSources *[]string) { // {{{
    LOG.Debug("checkAndFix: Checking for inconsistencies...")
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
    LOG.Debug("checkAndFix: ...completed check.")
    return
}   // }}}

// initSync does initial preparation for syncing and if preparations already
// done it does nothing so it should be safe to call in any case. Initial
// preparation incorporates following acts: 1. Replacement of given paths in
// syncSources with symlinks to directories under given tmpfs path. 2. Creation
// of a backup directory for every syncSource path.
func initSync(tmpfs string, syncSources *[]string, syncerBin string) error { // {{{
    LOG.Debug("initSync: Starting initial sync run...")
    for _, s := range *syncSources {
        var (
            fi       os.FileInfo
            uid, gid uint
            err      error
        )

        // Create initial tmpfs base dir
        if err := os.Mkdir(tmpfs, 0711); err != nil && !os.IsExist(err) {
            emsg := fmt.Sprintf("initSync: Creation of tmpfs dir '%s' failed...: %s", tmpfs, err)
            return errors.New(emsg)
        }

        // Base tmpfs dir needs at least 0111 (+x) for every user
        // (Mkdir uses umask so we need to chmod.)
        d, serr := os.Stat(tmpfs)
        if serr != nil {
            emsg := fmt.Sprintf("initSync: tmpfs path '%s' access error: %s", tmpfs, serr)
            return errors.New(emsg)
        }
        if m := d.Mode(); m&0111 != 0111 {
            if err := os.Chmod(tmpfs, m|0111); err != nil {
                emsg := fmt.Sprintf("initSync: Changing permissions of tmpfs dir '%s' failed...: %s", tmpfs, err)
                return errors.New(emsg)
            }
            lmsg := fmt.Sprintf("initSync: Changed '%s' permissions from '%s' -> '%s'.", tmpfs, m, m|0111)
            LOG.Info(lmsg)
        }

        if fi, uid, gid, err = isValidSource(s); err != nil {
            LOG.Warn("initSync: %s", err)
            LOG.Warn("initSync: Skipping sync source: %s", s)
            continue
        }

        // Volatile dirs name is based on orginal dir's name, uid and gid
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // First check if our target directory in tmpfs is ready.
        // We must ensure that the original owner of the source directory can
        // read the tmpfs volatile target dir, so we use the originals
        // permissions.
        if err := mkdirAll(volatilePath, fi.Mode(), uid, gid); err != nil { // {{{
            LOG.Warn("initSync (volatile path creation): %s", err)
            LOG.Warn("initSync: Skipping sync source: %s", s)
            continue
        }   // }}}

        // Second check if we need to create initial backup and initial sync to
        // volatile
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            // trying to rename the target path
            if err := os.Rename(s, backupPath); err != nil {
                LOG.Warn("initSync: could not rename target path: %s", err)
                LOG.Warn("initSync: Skipping sync source: %s", s)
                continue
            }
            // create symlink from original path to volatile path
            if linkError := os.Symlink(volatilePath, s); linkError != nil {
                LOG.Warn("initSync (symlink): %s", err)
                LOG.Warn("initSync: Skipping sync source: %s", s)
                os.Rename(backupPath, s)
                continue
            }
            // Let's do initial sync to volatile
            cmd := exec.Command(syncerBin, "-a", "--delete", backupPath+"/", s)
            if err := cmd.Run(); err != nil {
                LOG.Warn("initSync (volatile): '%s' => with command: %s", err, cmd)
                LOG.Warn("initSync: Skipping sync source: %s", s)
                os.Rename(backupPath, s)
            }
            continue
        } else {
            LOG.Debug("initSync: sync path was already initialized: %s", s)
        }   // }}}
    }
    LOG.Debug("initSync: ...completed without errors.")
    return nil
}   // }}}

// sync syncs content from tmpfs paths to backup paths. It expects that initSync
// has been called for the syncSources.
func sync(tmpfs string, syncSources *[]string, syncerBin string) { // {{{
    LOG.Debug("sync: Starting...")
    for _, s := range *syncSources {
        var (
            uid, gid uint
            err      error
        )

        if _, uid, gid, err = isValidSource(s); err != nil {
            LOG.Warn("sync: %s", err)
            LOG.Warn("sync: Skipping sync source: %s", s)
            continue
        }

        // Volatile dirs name is based on orginal dir's name, uid and gid
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // Volatile path must exists
        if !exists(volatilePath) {
            // syncInit failed or not called for the sync path
            LOG.Warn("sync (volatile path did not exist): %s", volatilePath)
            LOG.Warn("sync: Skipping sync source: %s", s)
            continue
        }

        // Target must be a symlink to the volatile path
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            LOG.Warn("sync (volatile path was not linked): %s", err)
            LOG.Warn("sync: Skipping sync source: %s", s)
            continue
        }   // }}}

        // Backup path must exists
        if !exists(backupPath) {
            // syncInit failed or not called for the sync path
            LOG.Warn("sync (backup path did not exist): %s", backupPath)
            LOG.Warn("sync: Skipping sync source: %s", s)
            continue
        }

        // Everything was ok, so we just sync from volatile tmpfs to backup
        cmd := exec.Command(syncerBin, "-a", "--delete", s+"/", backupPath)
        if err := cmd.Run(); err != nil { // {{{
            LOG.Err("sync (backup): '%s' >= with command: %s", err, cmd)
            LOG.Err("Sync: backup failed for sync source: %s", s)
            continue
        }   // }}}

        LOG.Debug("sync: synced dir '%s'.", s)
    }
    LOG.Debug("sync: ...completed.")
    return
}   // }}}

// unsync removes symbolic linkin to tmpfs and restores original from backup.
func unsync(tmpfs string, syncSources *[]string, removeVolatile bool) { // {{{
    LOG.Debug("unsync: Starting...")
    for _, s := range *syncSources {
        var (
            uid, gid uint
            err      error
        )
        if _, uid, gid, err = isValidSource(s); err != nil {
            LOG.Warn("unsync: %s", err)
            LOG.Warn("unsync: Skipping sync source: %s", s)
            continue
        }
        volatilePath, backupPath, _ := pathNameGen(s, tmpfs, uid, gid)

        // Check that backup path exists and is a directory
        if fi, err := os.Stat(backupPath); err != nil || !fi.IsDir() { // {{{
            LOG.Warn("unsync (backup): %s", err)
            LOG.Warn("unsync: Skipping sync source: %s", s)
            continue
        }   // }}}

        // Check that "s" was symlink to the volatile path
        if target, err := os.Readlink(s); err != nil || target != volatilePath { // {{{
            LOG.Warn("unsync (volatile): %s", err)
            LOG.Warn("unsync: Skipping sync source: %s", s)
            continue
        }   // }}}

        // Remove the link and replace it with backup
        os.Remove(s) // TODO: how we should react to an error from this?
        if err := os.Rename(backupPath, s); err != nil {
            LOG.Err("unsync: While trying to rename backup '%s' to '%s': %s", backupPath, s, err)
            continue
        }

        // Removing volatile after unsync makes checking that everything is
        // synced back to disk easier.
        if removeVolatile {
            if err := os.RemoveAll(volatilePath); err != nil {
                LOG.Err("unsync: While trying to remove volatile path: %s", err)
            }
            // Remove empty parents until base TMPFS dir
            volatileParent := path.Clean(volatilePath)
            cleanTmpfs := path.Clean(tmpfs)
            for rerr := error(nil); rerr == nil; rerr = os.Remove(volatileParent) {
                volatileParent = path.Dir(volatileParent)
                if cleanTmpfs == volatileParent {
                    break
                }
            }
        }
    }
    LOG.Debug("unsync: ...completed.")
    return
}   // }}}

// --------------------------------------------------------------------------

// runMain is a main function which returns programs exit value.
func runMain() int {
    var err error
    LOG, err = wl.New("goanysync", syslog.Priority(0), wl.DEFAULT_LOG_LEVEL)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: Logger initialization failed with error: %s\n", err)
        return 1
    }

    // Check that at least one argument given
    if len(os.Args) < 2 {
        LOG.Err("No command given.")
        return 1
    }
    configFilePath := flag.String("c", "/etc/goanysync.conf", "Config file.")
    verbose := flag.Bool("v", false, "Be more verbose with console messages.")
    syslogLogLevel := flag.Int("sl", int(wl.DEFAULT_LOG_LEVEL), "Set syslog log level.")
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage of %s %s:\n", os.Args[0], "[options] <command>")
        fmt.Fprintf(os.Stderr, "  Commands:\n")
        fmt.Fprintf(os.Stderr, "   initsync\tReplaces sync directories with symlinks to tmpfs while syncing orginal content there.\n")
        fmt.Fprintf(os.Stderr, "   sync\t\tSyncs content from tmpfs to the backup.\n")
        fmt.Fprintf(os.Stderr, "   unsync\tRestores orginal state of sync directories.\n")
        fmt.Fprintf(os.Stderr, "   check\tChecks if sync was called without unsync before tmpfs was cleared.\n")
        fmt.Fprintf(os.Stderr, "   start\tAlias for running check and initsync.\n")
        fmt.Fprintf(os.Stderr, "   stop\t\tAlias for running sync and unsync.\n")
        fmt.Fprintf(os.Stderr, "   info\t\tGives information about current sync status.\n")
        fmt.Fprintf(os.Stderr, "  Options:\n")
        flag.PrintDefaults()
        if *verbose {
            fmt.Fprintf(os.Stderr, "  Log levels:\n")
            for i := 0; i < len(wl.LOG_LEVELS); i++ {
                fmt.Fprintf(os.Stderr, "    %d: %s\n", i, wl.LOG_LEVELS[syslog.Priority(i)])
            }
        }
    }
    flag.Parse()

    LOG.SetSyslogPriority(syslog.Priority(*syslogLogLevel))
    if *verbose {
        LOG.SetConsoleLogPriority(syslog.LOG_DEBUG)
    }

    // Read config file
    copts, err := ReadConfigFile(*configFilePath)
    if err != nil {
        LOG.Err("Config file: %s", err)
        return 1
    }

    if *verbose {
        copts.Print()
    }

    // For now do not allow synchronous calls at all.
    // Check that lock files base path
    if err = checkLockFileDir(path.Dir(copts.lockfile)); err != nil {
        LOG.Err("Lock file path: %s", err)
        return 1
    }

    // Locking to prevent synchronous operations
    for ok, err := getLock(copts.lockfile); !ok; ok, err = getLock(copts.lockfile) {
        if err != nil {
            LOG.Err("Lock file: %s", err)
            return 1
        }
        // TODO: specify max wait time
        // TODO: use inotify when go provides an interface for it
        time.Sleep(time.Millisecond * 100)
    }
    // If os.Exit() is called remember to remove the lock file, manually.
    defer releaseLock(copts.lockfile)

    switch flag.Arg(0) {
    case "info":
        info(copts)
    case "check":
        checkAndFix(copts.tmpfsPath, &copts.syncPaths)
    case "initsync":
        if err := initSync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin); err != nil {
            LOG.Err(err.Error())
            return 1
        }
    case "sync":
        sync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin)
    case "unsync":
        unsync(copts.tmpfsPath, &copts.syncPaths, true)
    case "start":
        // Check that given TMPFS path does not contain any extra paths which
        // are not in syncPaths and might not be synced back
        if ok := checkVolatile(copts.tmpfsPath, &copts.syncPaths); !ok {
            return 1
        }
        checkAndFix(copts.tmpfsPath, &copts.syncPaths)
        if err := initSync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin); err != nil {
            LOG.Err(err.Error())
            return 1
        }
    case "stop":
        sync(copts.tmpfsPath, &copts.syncPaths, copts.syncerBin)
        unsync(copts.tmpfsPath, &copts.syncPaths, true)
        // If not all volatile paths were synced back issue a warning
        // XXX: does this really do above?
        if ok := checkVolatile(copts.tmpfsPath, &copts.syncPaths); !ok {
            return 1
        }
    default:
        LOG.Err("Invalid command provided", err)
        flag.Usage()
        return 1
    }
    return 0
}

func main() {
    os.Exit(runMain())
}

// vim: set sts=4 ts=4 sw=4 et foldmethod=marker:
