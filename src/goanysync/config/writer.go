// Copyright (C) 2012 Esa Määttä <esa.maatta@iki.fi>
// This file is released under the GNU GPL, version 3 or a later revision.
// For further details see the COPYING file.

package config

import (
    "bufio"
    "fmt"
    "os"
)

// Write writes Config struct to the given file.
func Write(c *Config, fn string) error { // {{{
    f, err := os.Create(fn)
    if err != nil {
        return err
    }

    // Write given config struct with one option, value pair per line
    bw := bufio.NewWriter(f)
    for option, value := range c.Data {
        line := fmt.Sprintf("%s %s %s\n", option, OPTION, *value)
        if _, err := bw.WriteString(line); err != nil {
            return err
        }
    }
    return nil
}   // }}}

// vim: set sts=4 ts=4 sw=4 et foldmethod=marker:
