package finish_test

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pseidemann/finish"
)

func Example() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		fmt.Fprintln(w, "world")
	})

	srv := &http.Server{Addr: "localhost:8080"}

	fin := finish.New()
	fin.Add(srv)

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	fin.Wait()
}
