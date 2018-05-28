package finish_test

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pseidemann/finish"
)

func Example() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "world")
	})

	srv := &http.Server{Addr: "localhost:8080"}

	fin := &finish.Finisher{Log: finish.StdoutLogger}
	fin.Add(srv)

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		// Simulate user pressing Control-C to make example work
		fin.Trigger()
	}()

	fin.Wait()
	// Output:
	// finish: shutdown signal received
	// finish: shutting down server ...
	// finish: server closed
}
