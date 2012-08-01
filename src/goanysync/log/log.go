// Copyright (C) 2012 Esa Määttä <esa.maatta@iki.fi>
// This file is released under the GNU GPL, version 3 or a later revision.
// For further details see the COPYING file.

package log

import (
    "os"
    "log"
    "log/syslog"
)

const (
    DEFAULT_LOG_LEVEL = syslog.LOG_WARNING
)

type Log struct {
    conlog1 *log.Logger
    conlog2 *log.Logger
    syslog *log.Logger
    p syslog.Priority
}

// New creates a new Log and returns pointer to it.
func New(prefix string, p syslog.Priority) (*Log, error) {
    l := new(Log)
    consoleFlags := log.Ldate|log.Ltime|log.Lshortfile
    l.conlog1 = log.New(os.Stdout, prefix + ": ", consoleFlags)
    l.conlog2 = log.New(os.Stderr, prefix + ": ", consoleFlags)
    l.p = p

    var err error
    if l.syslog, err = syslog.NewLogger(syslog.LOG_INFO, log.Lshortfile); err != nil {
        return nil, err
    }
    return l, nil
}

func (self *Log) pMsg(p syslog.Priority, prefix string, format string, v ...interface{}) {
    if p <= self.p {
        // If increased log level print also to console
        if self.p > DEFAULT_LOG_LEVEL {
            format = prefix + ": " + format
            self.conlog2.Printf(format, v...)
        }
        self.syslog.Printf(format, v...)
    }
}

func (self *Log) SetPriority(p syslog.Priority) {
    self.p = p
}

func (self *Log) Crit(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_CRIT, "Critical", format, v...)
}

func (self *Log) Err(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_ERR, "Error", format, v...)
}

func (self *Log) Warn(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_WARNING, "Warning", format, v...)
}

func (self *Log) Info(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_INFO, "Info", format, v...)
}

func (self *Log) Debug(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_DEBUG, "Debug", format, v...)
}
