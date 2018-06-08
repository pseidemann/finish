# finish

[![GoDoc](https://godoc.org/github.com/pseidemann/finish?status.svg)](https://godoc.org/github.com/pseidemann/finish)
[![Go Report Card](https://goreportcard.com/badge/github.com/pseidemann/finish)](https://goreportcard.com/report/github.com/pseidemann/finish)
[![Build Status](https://travis-ci.org/pseidemann/finish.svg?branch=master)](https://travis-ci.org/pseidemann/finish)

Package finish provides gracious shutdown of servers.

It uses `http.Server`'s built-in [`Shutdown()`](https://golang.org/pkg/net/http/#Server.Shutdown)
method and therefore requires Go 1.8+.


## Quick Start

```sh
# assume the following code in simple.go file
$ cat simple.go
```

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pseidemann/finish"
)

func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		fmt.Fprintln(w, "world")
	})

	srv := &http.Server{Addr: "localhost:8080"}

	fin := finish.New()
	fin.Add(srv)

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	fin.Wait()
}
```

```sh
# run simple.go
$ go run simple.go
```

```sh
# now do a GET
$ curl localhost:8080/hello
# it will print "world" after 5 seconds
```

If you terminate the server with pressing `Ctrl+C` or `kill`, while `/hello` is
loading, finish will wait until the request was handled, before the server gets
killed.

The output will look like this:
```
2038/01/19 03:14:08 finish: shutdown signal received
2038/01/19 03:14:08 finish: shutting down server ...
2038/01/19 03:14:11 finish: server closed
```


## Full Example

This example uses a custom router [httprouter](https://github.com/julienschmidt/httprouter),
a custom logger [logrus](https://github.com/sirupsen/logrus), options for `Add()`
and multiple servers.

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/pseidemann/finish"
	"github.com/sirupsen/logrus"
)

func main() {
	routerPub := httprouter.New()
	routerPub.HandlerFunc("GET", "/hello", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		fmt.Fprintln(w, "world")
	})

	routerInt := httprouter.New()
	routerInt.HandlerFunc("GET", "/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	srvPub := &http.Server{Addr: "localhost:8080", Handler: routerPub}
	srvInt := &http.Server{Addr: "localhost:3000", Handler: routerInt}

	fin := &finish.Finisher{Log: logrus.StandardLogger(), Timeout: 30 * time.Second}
	fin.Add(srvPub, finish.WithName("public server"))
	fin.Add(srvInt, finish.WithName("internal server"), finish.WithTimeout(5*time.Second))

	go func() {
		logrus.Infof("starting public server at %s", srvPub.Addr)
		err := srvPub.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		logrus.Infof("starting internal server at %s", srvInt.Addr)
		err := srvInt.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	fin.Wait()
}
```
