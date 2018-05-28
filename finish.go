// Package finish provides gracious shutdown of servers.
package finish

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// DefaultTimeout is used if no timeout is given for a server.
const DefaultTimeout = 10 * time.Second

var (
	// DefaultLogger is used if Finisher.Logger is not set. It uses the Go standard log package.
	DefaultLogger = &defaultLogger{}
	// StdoutLogger can be used as a simple logger which writes to stdout via the fmt standard package.
	StdoutLogger = &stdoutLogger{}
	// DefaultSignals is used if Finisher.Signals is not set.
	DefaultSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
)

// A Server is a type which can be shutdown.
//
// This interface is expected by Add() and allows registering any server which
// implements a Shutdown() method.
type Server interface {
	Shutdown(ctx context.Context) error
}

type serverKeeper struct {
	srv     Server
	name    string
	timeout time.Duration
}

// Finisher implements gracious shutdown of servers.
type Finisher struct {
	Timeout time.Duration
	Log     Logger
	Signals []os.Signal

	mutex   sync.Mutex
	keepers []*serverKeeper
	manSig  chan interface{}
}

// New creates a Finisher.
func New() *Finisher {
	return &Finisher{}
}

func (f *Finisher) signals() []os.Signal {
	if f.Signals != nil {
		return f.Signals
	}
	return DefaultSignals
}

func (f *Finisher) log() Logger {
	if f.Log != nil {
		return f.Log
	}
	return DefaultLogger
}

func (f *Finisher) timeout() time.Duration {
	if f.Timeout != 0 {
		return f.Timeout
	}
	return DefaultTimeout
}

func (f *Finisher) getManSig() chan interface{} {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.manSig == nil {
		f.manSig = make(chan interface{}, 1)
	}
	return f.manSig
}

// Add a server for gracious shutdown.
func (f *Finisher) Add(srv Server, opts ...Option) {
	keeper := &serverKeeper{
		srv:     srv,
		timeout: f.timeout(),
	}

	for _, opt := range opts {
		if err := opt(keeper); err != nil {
			panic(err)
		}
	}

	f.keepers = append(f.keepers, keeper)
}

// Wait blocks until the shutdown signal is received and then closes all servers with a timeout.
//
// The default shutdown signals are:
//  - SIGINT (triggered by pressing Control-C)
//  - SIGTERM (sent by `kill $pid` or e.g. systemd stop)
func (f *Finisher) Wait() {
	f.updateNames()

	signals := f.signals()
	stop := make(chan os.Signal, len(signals))
	signal.Notify(stop, signals...)

	// wait for signal
	select {
	case sig := <-stop:
		if sig == syscall.SIGINT {
			// fix prints after "^C"
			fmt.Println("")
		}
	case <-f.getManSig():
		// Trigger() was called
	}

	f.log().Infof("finish: shutdown signal received")

	for _, keeper := range f.keepers {
		ctx, cancel := context.WithTimeout(context.Background(), keeper.timeout)
		defer cancel()
		f.log().Infof("finish: shutting down %s ...", keeper.name)
		err := keeper.srv.Shutdown(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				f.log().Errorf("finish: shutdown timeout for %s", keeper.name)
			} else {
				f.log().Errorf("finish: error while shutting down %s: %s", keeper.name, err)
			}
		} else {
			f.log().Infof("finish: %s closed", keeper.name)
		}
	}
}

// Trigger the shutdown signal manually.
func (f *Finisher) Trigger() {
	f.getManSig() <- nil
}

func (f *Finisher) updateNames() {
	if len(f.keepers) == 1 && f.keepers[0].name == "" {
		f.keepers[0].name = "server"
		return
	}

	for i, keeper := range f.keepers {
		if keeper.name == "" {
			keeper.name = fmt.Sprintf("server #%d", i+1)
		}
	}
}
