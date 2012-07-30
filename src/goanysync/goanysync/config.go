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
    "path"
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

    if len(tmpfsPath) < 1 {
        err = errors.New("Empty TMPFS path defined.")
        return
    }

    if !path.IsAbs(tmpfsPath) {
        err = errors.New("TMPFS path must be absolute.")
        return
    }

    // TMPFS dir does not have to exists as init creates it, but all it's parent
    // dirs must.
    // Check that every TMPFS parent dir has excutable bit set for all users.
    for p := path.Dir(tmpfsPath); p != string(os.PathSeparator); p = path.Dir(p) {
        d, serr := os.Stat(p)
        if serr != nil {
            fmsg := fmt.Sprintf("The TMPFS parent path '%s' access error: %s", p, serr)
            err = errors.New(fmsg)
            return
        }
        if m := d.Mode(); m&0111 != 0111 {
            fmsg := fmt.Sprintf("The TMPFS parent path '%s' did not have executable bit set for all users.", p)
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
