package finish

import (
	"fmt"
	"log"
)

// Logger is the interface expected by Finisher.Log.
// It allows using any logger which implements the Infof() and Errorf() methods.
type Logger interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
}

// default logger

type defaultLogger struct{}

func (d *defaultLogger) Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (d *defaultLogger) Errorf(format string, args ...interface{}) {
	d.Infof(format, args...)
}

// stdout logger

type stdoutLogger struct{}

func (s *stdoutLogger) Infof(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (s *stdoutLogger) Errorf(format string, args ...interface{}) {
	s.Infof(format, args...)
}
