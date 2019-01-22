// Package finish provides graceful shutdown of servers.
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

// DefaultTimeout is used if Finisher.Timeout is not set.
const DefaultTimeout = 10 * time.Second

var (
	// DefaultLogger is used if Finisher.Logger is not set. It uses the Go standard log package.
	DefaultLogger = &defaultLogger{}
	// StdoutLogger can be used as a simple logger which writes to stdout via the fmt standard package.
	StdoutLogger = &stdoutLogger{}
	// DefaultSignals is used if Finisher.Signals is not set.
	// The default shutdown signals are:
	//   - SIGINT (triggered by pressing Control-C)
	//   - SIGTERM (sent by `kill $pid` or e.g. systemd stop)
	DefaultSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
)

// A Server is a type which can be shutdown.
//
// This is the interface expected by Add() which allows registering any server which implements the Shutdown() method.
type Server interface {
	Shutdown(ctx context.Context) error
}

type serverKeeper struct {
	srv     Server
	name    string
	timeout time.Duration
}

// Finisher implements graceful shutdown of servers.
type Finisher struct {
	// Timeout is the maximum amount of time to wait for still running server
	// requests to finish when the shutdown signal was received for each server.
	// It defaults to DefaultTimeout which is 10 seconds.
	//
	// The timeout can be overridden on a per server basis with passing the
	// WithTimeout() option to Add() while adding the server.
	Timeout time.Duration

	// Log can be set to change where finish logs to.
	// It defaults to DefaultLogger which uses the standard Go log package.
	Log Logger

	// Signals can be used to change which signals finish catches to initiate
	// the shutdown.
	// It defaults to DefaultSignals which contains SIGINT and SIGTERM.
	Signals []os.Signal

	mutex   sync.Mutex
	keepers []*serverKeeper
	manSig  chan interface{}
}

// New creates a Finisher. This is a convenience constructor if no changes to the default configuration are needed.
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

// Add a server for graceful shutdown.
//
// Options can be passed as the second argument to change the behavior for this server:
//
// To give the server a specific name instead of just "server #<num>":
// 	fin.Add(srv, finish.WithName("internal server"))
//
// To override the timeout, configured in Finisher, for this specific server:
// 	fin.Add(srv, finish.WithTimeout(5*time.Second))
//
// To do both at the same time:
// 	fin.Add(srv, finish.WithName("internal server"), finish.WithTimeout(5*time.Second))
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

// Wait blocks until one of the shutdown signals is received and then closes all servers with a timeout.
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

// Trigger the shutdown signal manually. This is probably only useful for testing.
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
