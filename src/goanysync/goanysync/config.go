// Copyright (C) 2012 Esa Määttä <esa.maatta@iki.fi>
// This file is released under the GNU GPL, version 3 or a later revision.
// For further details see the COPYING file.

package main

import (
    "errors"
    "fmt"
    "goanysync/config"
    "os"
    "os/exec"
    "strings"
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

    // ---------------------------------------
    // Read the config files TMPFS option
    if _, ok := c.Data["TMPFS"]; !ok {
        err = errors.New("No TMPFS defined.")
        return
    }
    tmpfsPath := strings.TrimSpace(*c.Data["TMPFS"])
    //tmpfsPath = strings.TrimSpace(tmpfsPath)

    if len(tmpfsPath) < 1 {
        err = errors.New("Empty TMPFS path defined.")
        return
    }

    // this again only checks if _anyone_ can write to tmpfsPath -> TODO
    tmpfsStat, oerr := os.Stat(tmpfsPath)
    if oerr == nil {
        if tmpfsPerm := os.FileMode.Perm(tmpfsStat.Mode()); tmpfsPerm&0222 == 0 {
            fmsg := fmt.Sprintf("The tmpfsPath is not writable. (%s)", tmpfsPath)
            err = errors.New(fmsg)
            return
        }
    }

    // ---------------------------------------
    // Read the config files RSYNC_BIN option.
    // If no RSYNC_BIN option is defined in the config file default to "rsync".
    var syncerBin string = "rsync"
    if _, ok := c.Data["RSYNC_BIN"]; ok {
        syncerBin = *c.Data["RSYNC_BIN"]
    }
    syncerBin = strings.TrimSpace(syncerBin)
    // If RSYNC_BIN option is defined but with empty value then issue error.
    if len(syncerBin) < 1 {
        err = errors.New("Empty RSYNC_BIN path defined.")
        return
    }

    // Check that syncerBin is executable and found from PATH if it's relative.
    if _, oerr := exec.LookPath(syncerBin); oerr != nil {
        fmsg := fmt.Sprintf("Could not find the sync-binary. (%s) - %s", syncerBin, oerr)
        err = errors.New(fmsg)
        return
    }

    // ---------------------------------------
    // Read the config files WHATTOSYNC option
    if _, ok := c.Data["WHATTOSYNC"]; !ok {
        err = errors.New("No WHATTOSYNC defined.")
        return
    }
    syncPaths := strings.TrimSpace(*c.Data["WHATTOSYNC"])

    if len(syncPaths) < 1 {
        err = errors.New("Empty WHATTOSYNC paths defined.")
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
