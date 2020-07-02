package log

import (
	"fmt"
	"os"
)

type Level int

const (
	LevelDebug = Level(-1)
	LevelInfo  = Level(0)
)

type Log struct {
	Level Level
}

func (l Log) Debug(format string, args ...interface{}) {
	if LevelDebug >= l.Level {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", format), args...)
	}
}

func (l Log) Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", format), args...)
}

func (l Log) Fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", format), args...)
	os.Exit(1)
}
