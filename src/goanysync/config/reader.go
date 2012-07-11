// Copyright (C) 2012 Esa Määttä <esa.maatta@iki.fi>
// This file is released under the GNU GPL, version 3 or a later revision.
// For further details see the COPYING file.

package config

import (
    "io"
    "errors"
    "bufio"
    "os"
    "strings"
    "unicode"
)

type Config struct {
    //comment   string
    //separator string

    // option -> value
    Data map[string] *string
}

func Read(file string) (*Config, error) {
    // Initialize Config type
    c := new(Config)
    c.Data = make(map[string]*string)

    f, err := os.Open(file)
    if err != nil {
        return nil, err
    }

    const (
        COMMENT = '#'
        OPTION  = "="
    )

    // Read the config file and store option values to the created config
    br := bufio.NewReader(f);
    for {
        line, err := br.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                return c, nil
            }
            return nil, err
        }

        line = strings.TrimSpace(line)

        // Skip empty and comment lines
        if line == "" || line[0] == COMMENT {
            continue
        }

        // Parse option line
        // TODO: maybe allow multiline options
        optionLine := strings.SplitN(line, OPTION, 2)

        if len(optionLine) != 2 {
            return nil, errors.New("Could not parse line: " + line)
        }

        optionName := strings.TrimRightFunc(optionLine[0], unicode.IsSpace)
        if optionName == "" {
            return nil, errors.New("Could not parse line: " + line)
        }

        // TODO: Strip comments
        optionValue := strings.TrimLeftFunc(optionLine[1], unicode.IsSpace)

        // Add parsed option to the config
        c.Data[optionName] = &optionValue
    }

    return c, nil
}
