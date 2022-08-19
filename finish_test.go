package finish

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

var errTest = errors.New("test error")

type testServer struct {
	shutdown bool
	wait     time.Duration
}

func (t *testServer) Shutdown(ctx context.Context) error {
	wait := time.NewTimer(t.wait)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait.C:
		// server finished fake busy work
	}

	t.shutdown = true

	return nil
}

type logRecorder struct {
	infos  []string
	errors []string
}

func (l *logRecorder) Infof(format string, args ...interface{}) {
	l.infos = append(l.infos, fmt.Sprintf(format, args...))
}

func (l *logRecorder) Errorf(format string, args ...interface{}) {
	l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

func captureStdout(f func()) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	before := os.Stdout
	os.Stdout = writer

	f()

	if err := writer.Close(); err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		panic(err)
	}

	if err := reader.Close(); err != nil {
		panic(err)
	}

	os.Stdout = before

	return buf.String()
}

func captureLog(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr)
	return buf.String()
}

func Test(t *testing.T) {
	srv := &testServer{wait: time.Second}
	log := &logRecorder{}

	fin := &Finisher{Log: log}
	fin.Add(srv)

	keeper := fin.keepers[0]

	if keeper.srv != srv {
		t.Error("expected server to be added")
	}

	if keeper.name != "" {
		t.Error("expected name to be empty")
	}

	if keeper.timeout != DefaultTimeout {
		t.Error("expected timeout to be the default")
	}

	go fin.Trigger()

	if srv.shutdown {
		t.Error("expected server not to be shutdown yet")
	}

	fin.Wait()

	if !srv.shutdown {
		t.Error("expected server to be shutdown")
	}

	if keeper.name != "server" {
		t.Error("expected name to be 'server'")
	}

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
		"finish: server closed",
	}) {
		t.Error("wrong log output")
	}

	if log.errors != nil {
		t.Error("expected no error logs")
	}
}

func TestDefaultLogger(t *testing.T) {
	srv := &testServer{wait: 2 * time.Second}

	fin := &Finisher{Timeout: time.Second}
	fin.Add(srv)

	go fin.Trigger()

	log := captureLog(func() {
		fin.Wait()
	})

	// using Contains() because the default logger contains timestamps

	if !strings.Contains(log, "finish: shutdown signal received") {
		t.Error("missing log")
	}

	if !strings.Contains(log, "finish: shutting down server ...") {
		t.Error("missing log")
	}

	// trigger error to get coverage for defaultLogger's Errorf()
	if !strings.Contains(log, "finish: shutdown timeout for server") {
		t.Error("missing log")
	}
}

func TestStdoutLogger(t *testing.T) {
	srv := &testServer{wait: 2 * time.Second}

	fin := &Finisher{Timeout: time.Second, Log: StdoutLogger}
	fin.Add(srv)

	go fin.Trigger()

	stdout := captureStdout(func() {
		fin.Wait()
	})

	if stdout != "finish: shutdown signal received\n"+
		"finish: shutting down server ...\n"+
		"finish: shutdown timeout for server\n" {
		t.Error("wrong log")
	}
}

func TestSettingName(t *testing.T) {
	srv := &testServer{}
	log := &logRecorder{}

	fin := &Finisher{Log: log}
	fin.Add(srv, WithName("foobar"))

	keeper := fin.keepers[0]

	if keeper.name != "foobar" {
		t.Error("expected name to be set")
	}

	go fin.Trigger()

	fin.Wait()

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down foobar ...",
		"finish: foobar closed",
	}) {
		t.Error("wrong log output")
	}

	if log.errors != nil {
		t.Error("expected no error logs")
	}
}

func TestUpdateNames(t *testing.T) {
	srv := &testServer{}

	fin := New()
	fin.Add(srv, WithName("foobar"))
	fin.Add(srv)

	fin.updateNames()

	if fin.keepers[0].name != "foobar" {
		t.Error("wrong name")
	}

	if fin.keepers[1].name != "server #2" {
		t.Error("wrong name")
	}
}

func TestGlobalTimeout(t *testing.T) {
	srv := &testServer{}

	fin := &Finisher{Timeout: 21 * time.Second}
	fin.Add(srv)

	keeper := fin.keepers[0]

	if keeper.timeout != 21*time.Second {
		t.Error("expected timeout to be changed")
	}
}

func TestOverridingTimeout(t *testing.T) {
	srv := &testServer{}

	fin := New()
	fin.Add(srv, WithTimeout(42*time.Second))

	keeper := fin.keepers[0]

	if keeper.timeout != 42*time.Second {
		t.Error("expected timeout to be set")
	}
}

func TestOptionError(t *testing.T) {
	testOpt := func(keeper *serverKeeper) error {
		return errTest
	}

	srv := &testServer{}

	fin := New()
	func() {
		defer func() {
			err := recover()
			if err != errTest {
				t.Error("expected Add() to panic")
			}
		}()

		fin.Add(srv, testOpt)
	}()
}

func TestSlowServer(t *testing.T) {
	srv := &testServer{wait: 2 * time.Second}
	log := &logRecorder{}

	fin := &Finisher{Log: log}
	fin.Add(srv, WithTimeout(time.Second))

	go fin.Trigger()

	fin.Wait()

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
	}) {
		t.Error("wrong log output")
	}

	if !reflect.DeepEqual(log.errors, []string{
		"finish: shutdown timeout for server",
	}) {
		t.Error("wrong error log output")
	}
}

type testServerErr struct{}

func (t *testServerErr) Shutdown(ctx context.Context) error {
	return errTest
}

func TestServerError(t *testing.T) {
	srv := &testServerErr{}
	log := &logRecorder{}

	fin := &Finisher{Log: log}
	fin.Add(srv, WithTimeout(time.Second))

	go fin.Trigger()

	fin.Wait()

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
	}) {
		t.Error("wrong log output")
	}

	if !reflect.DeepEqual(log.errors, []string{
		"finish: error while shutting down server: test error",
	}) {
		t.Error("wrong error log output")
	}
}

func TestSigIntPrint(t *testing.T) {
	srv := &testServer{}
	log := &logRecorder{}

	fin := &Finisher{Log: log}
	fin.Add(srv)

	go func() {
		// sleep so Wait() can actually catch the signal
		time.Sleep(time.Second)
		// trigger signal
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			panic(err)
		}
		p.Signal(syscall.SIGINT)
	}()

	stdout := captureStdout(func() {
		fin.Wait()
	})

	if stdout != "\n" {
		t.Error("expected newline to be printed to stdout")
	}

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
		"finish: server closed",
	}) {
		t.Error("wrong log output")
	}

	if log.errors != nil {
		t.Error("expected no error logs")
	}
}

func TestCustomSignal(t *testing.T) {
	srv := &testServer{}
	log := &logRecorder{}

	mySignal := syscall.SIGUSR1

	fin := &Finisher{Log: log, Signals: []os.Signal{mySignal}}
	fin.Add(srv)

	go func() {
		// sleep so Wait() can actually catch the signal
		time.Sleep(time.Second)
		// trigger custom signal
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			panic(err)
		}
		p.Signal(mySignal)
	}()

	fin.Wait()

	if !reflect.DeepEqual(log.infos, []string{
		"finish: shutdown signal received",
		"finish: shutting down server ...",
		"finish: server closed",
	}) {
		t.Error("wrong log output")
	}

	if log.errors != nil {
		t.Error("expected no error logs")
	}
}
