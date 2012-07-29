// Copyright (C) 2012 Esa Määttä <esa.maatta@iki.fi>
// This file is released under the GNU GPL, version 3 or a later revision.
// For further details see the COPYING file.

package main

import (
    "errors"
    "fmt"
    "goanysync/config"
    "strings"
    "os/exec"
    "os"
)

// configOptions to be read from the config file.
type ConfigOptions struct {
    tmpfsPath string
    syncPaths []string
    syncerBin string
}

func (self *ConfigOptions) Print() {
    const indent string = "  "
    fmt.Println("Config options:")
    fmt.Println(indent, "TMPFS:", self.tmpfsPath)
    fmt.Println(indent, "RSYNC_BIN:", self.syncerBin)
    fmt.Println(indent, "WHATTOSYNC:")
    for i, v := range self.syncPaths {
        fmt.Printf("%s%s %d: %s\n", indent, indent, i, v)
    }
}

// readConfigFile reads config file and checks that necessary information was
// given. After this it returns the read options in configOptions struct.
func ReadConfigFile(cfp string) (copts *ConfigOptions, err error) {
    var c *config.Config
    c, err = config.Read(cfp)
    if err != nil {
        return
    }

    // Read the config file
    //tmpfsPath, _ := c.String("DEFAULT", "TMPFS")
    //syncerBin, _ := c.String("DEFAULT", "RSYNC_BIN")
    //syncPaths, _ := c.String("DEFAULT", "WHATTOSYNC")

    _, tmpfs_ok := c.Data["TMPFS"]
    _, syncbin_ok := c.Data["RSYNC_BIN"]
    _, paths_ok := c.Data["WHATTOSYNC"]

    if ! tmpfs_ok {
        err = errors.New("No TMPFS defined.")
        return
    }
    tmpfsPath := *c.Data["TMPFS"]

    var syncerBin string = "rsync"
    if ! syncbin_ok {
        //err = errors.New("No RSYNC_BIN defined, assuming 'rsync'")
        syncerBin = "rsync"
    } else {
        syncerBin = *c.Data["RSYNC_BIN"]
    }

    if ! paths_ok {
        err = errors.New("No WHATTOSYNC defined.")
        return
    }
    syncPaths := *c.Data["WHATTOSYNC"]

    tmpfsPath = strings.TrimSpace(tmpfsPath)
    syncerBin = strings.TrimSpace(syncerBin)
    syncPaths = strings.TrimSpace(syncPaths)

    // Check that given options are valid to some degree
    if len(tmpfsPath) < 1 {
        err = errors.New("Empty TMPFS path defined.")
        return
    }
    if len(syncPaths) < 1 {
        err = errors.New("Empty WHATTOSYNC paths defined.")
        return
    }
    // check if binary is inside path
    sync_path, oerr := exec.LookPath(syncerBin)
    if oerr != nil {
        fmsg := fmt.Sprintf("Could not find the sync-binary. (%s) - %s", syncerBin, oerr)
        err = errors.New(fmsg)
        return
    }
    // this only checks if _anyone_ can execute syncerBin -> TODO
    file_stat, oerr := os.Stat(sync_path)
    if oerr != nil {
        fmsg := fmt.Sprintf("Could not execute stat() on sync-binary. (%s) - %s", syncerBin, oerr)
        err = errors.New(fmsg)
        return
    }
    syncerBin = sync_path
    bin_perms := os.FileMode.Perm(file_stat.Mode())
    if bin_perms & 0111 == 0 {
        fmsg := fmt.Sprintf("The sync-binary is not executable. (%s)", syncerBin)
        err = errors.New(fmsg)
        return
    }
    // this again only checks if _anyone_ can write to tmpfsPath -> TODO
    tmpfs_stat, oerr := os.Stat(tmpfsPath)
    if oerr != nil {
        fmsg := fmt.Sprintf("Could not execute stat() on tmpfsPath. (%s) - %s", tmpfsPath, oerr)
        err = errors.New(fmsg)
        return
    }
    tmpfs_perms := os.FileMode.Perm(tmpfs_stat.Mode())
    if tmpfs_perms & 0222 == 0 {
        fmsg := fmt.Sprintf("The tmpfsPath is not writable. (%s)", tmpfsPath)
        err = errors.New(fmsg)
        return
    }

    // Parse WHATTOSYNC comma separated list of paths
    // XXX: if path names contain commas then though luck for now
    fieldFunc := func(r rune) bool {
        return r == ','
    }
    paths := strings.FieldsFunc(syncPaths, fieldFunc)
    if len(paths) < 1 {
        err = errors.New("Empty WHATTOSYNC paths defined.")
        return
    }
    for i, v := range paths {
        paths[i] = strings.TrimSpace(v)
    }

    copts = &ConfigOptions{tmpfsPath, paths, syncerBin}
    return
}

// vim: set sts=4 ts=4 sw=4 et foldmethod=marker:
