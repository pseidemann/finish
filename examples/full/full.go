package main

import (
	"fmt"
	"log"
	"net/http"
	"syscall"
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

	fin := &finish.Finisher{
		Timeout: 30 * time.Second,
		Log:     logrus.StandardLogger(),
		Signals: append(finish.DefaultSignals, syscall.SIGHUP),
	}
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
