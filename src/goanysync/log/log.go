// Copyright (C) 2012 Esa Määttä <esa.maatta@iki.fi>
// This file is released under the GNU GPL, version 3 or a later revision.
// For further details see the COPYING file.

package log

import (
    "log"
    "log/syslog"
    "os"
)

const (
    DEFAULT_LOG_LEVEL = syslog.LOG_WARNING
)

var LOG_LEVELS = map[syslog.Priority] string {
    syslog.LOG_EMERG: "Emergency",
    syslog.LOG_ALERT: "Alert",
    syslog.LOG_CRIT: "Critical",
    syslog.LOG_ERR: "Error",
    syslog.LOG_WARNING: "Warning",
    syslog.LOG_NOTICE: "Notice",
    syslog.LOG_INFO: "Info",
    syslog.LOG_DEBUG: "Debug",
}

type Log struct {
    conlog1 *log.Logger
    conlog2 *log.Logger
    syslog  *log.Logger
    sp      syslog.Priority // syslog priority
    cp      syslog.Priority // console log priority
}

// New creates a new Log and returns pointer to it.
func New(prefix string, sp syslog.Priority, cp syslog.Priority) (*Log, error) {
    l := new(Log)
    consoleFlags := log.Ldate | log.Ltime | log.Lshortfile
    l.conlog1 = log.New(os.Stdout, prefix+": ", consoleFlags)
    l.conlog2 = log.New(os.Stderr, prefix+": ", consoleFlags)
    l.sp = sp
    l.cp = cp

    var err error
    if l.syslog, err = syslog.NewLogger(syslog.LOG_INFO, log.Lshortfile); err != nil {
        return nil, err
    }
    return l, nil
}

func (self *Log) pMsg(p syslog.Priority, prefix string, format string, v ...interface{}) {
    // Print to syslog
    if p <= self.sp {
        self.syslog.Printf(format, v...)
    }
    // Print to console log
    if p <= self.cp {
        // Append line ending if needed
        if len(format) == 0 || format[len(format)-1] != '\n' {
            format = format + "\n"
        }
        format = prefix + ": " + format
        self.conlog2.Printf(format, v...)
    }
}

func (self *Log) SetSyslogPriority(p syslog.Priority) {
    self.sp = p
}

func (self *Log) SetConsoleLogPriority(p syslog.Priority) {
    self.cp = p
}

func (self *Log) Emerg(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_EMERG, LOG_LEVELS[syslog.LOG_EMERG], format, v...)
}

func (self *Log) Alert(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_ALERT, LOG_LEVELS[syslog.LOG_ALERT], format, v...)
}

func (self *Log) Crit(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_CRIT, LOG_LEVELS[syslog.LOG_CRIT], format, v...)
}

func (self *Log) Err(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_ERR, LOG_LEVELS[syslog.LOG_ERR], format, v...)
}

func (self *Log) Warn(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_WARNING, LOG_LEVELS[syslog.LOG_WARNING], format, v...)
}

func (self *Log) Notice(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_NOTICE, LOG_LEVELS[syslog.LOG_NOTICE], format, v...)
}

func (self *Log) Info(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_INFO, LOG_LEVELS[syslog.LOG_INFO], format, v...)
}

func (self *Log) Debug(format string, v ...interface{}) {
    self.pMsg(syslog.LOG_DEBUG, LOG_LEVELS[syslog.LOG_DEBUG], format, v...)
}
