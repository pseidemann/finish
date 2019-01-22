package main

import (
	"fmt"
	"log"
	"net/http"
	"syscall"
	"time"

	"github.com/pseidemann/finish"
)

func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		fmt.Fprintln(w, "world")
	})

	srv := &http.Server{Addr: "localhost:8080"}

	fin := &finish.Finisher{Signals: append(finish.DefaultSignals, syscall.SIGHUP)}
	fin.Add(srv)

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	fin.Wait()
}
