package finish

import (
	"fmt"
	"log"
)

// Logger is the interface expected by [Finisher].Log.
//
// It allows using any loggers which implement the Infof() and Errorf() methods.
type Logger interface {
	Infof(format string, v ...any)
	Errorf(format string, v ...any)
}

// default logger

type defaultLogger struct{}

func (l *defaultLogger) Infof(format string, v ...any) {
	log.Printf(format, v...)
}

func (l *defaultLogger) Errorf(format string, v ...any) {
	l.Infof(format, v...)
}

// stdout logger

type stdoutLogger struct{}

func (l *stdoutLogger) Infof(format string, v ...any) {
	fmt.Printf(format+"\n", v...)
}

func (l *stdoutLogger) Errorf(format string, v ...any) {
	l.Infof(format, v...)
}
