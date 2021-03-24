// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
// package log provides some simple information level print helpers wrapped
// up in a lightweight struct that can be embedded in other objects
package log

import (
	"fmt"
	"os"
)

type Logger struct {
	// The verbosity level of this logger. -1 means quiet mode,
	// 0 (the default) means normal mode, and 1 means verbose mode.
	level int
}

func NewLogger(level int) *Logger {
	return &Logger{level: level}
}

// Print `output` at a normal verbosity level
func (l *Logger) Info(output string) {
	if l.level >= 0 {
		fmt.Print(output)
	}
}

// Print `output` at a normal verbosity level, formatting the output
// using the standard formatting codes from `fmt`.
func (l *Logger) Infof(format string, a ...interface{}) {
	l.Info(fmt.Sprintf(format, a...))
}

func (l *Logger) Warn(output string) {
	if l.level >= -1 {
		fmt.Fprint(os.Stderr, output)
	}
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.Warn(fmt.Sprintf("WARN: "+format, a...))
}
