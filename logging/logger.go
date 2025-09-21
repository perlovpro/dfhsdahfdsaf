package logging

import (
	"fmt"
	"time"
)

type Logger struct {
	verbose bool
}

func New(verbose bool) *Logger {
	return &Logger{verbose: verbose}
}

func (l *Logger) Println(format string, args ...any) {
	if !l.verbose {
		return
	}
	prefix := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s\n", prefix, fmt.Sprintf(format, args...))
}

func (l *Logger) Info(format string, args ...any) {
	prefix := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s\n", prefix, fmt.Sprintf(format, args...))
}

func (l *Logger) Error(format string, args ...any) {
	prefix := time.Now().Format("15:04:05")
	fmt.Printf("[%s] ERROR: %s\n", prefix, fmt.Sprintf(format, args...))
}
