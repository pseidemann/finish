package finish

import (
	"fmt"
	"log"
)

// Logger is the interface expected by [Finisher].Log.
//
// It allows using any loggers which implement the Infof() and Errorf() methods.
type Logger interface {
	Infof(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// default logger

type defaultLogger struct{}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.Infof(format, v...)
}

// stdout logger

type stdoutLogger struct{}

func (l *stdoutLogger) Infof(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

func (l *stdoutLogger) Errorf(format string, v ...interface{}) {
	l.Infof(format, v...)
}
