package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path)) // nolint gas
	})

	http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		log.Fatal("Forced failure")
	})

	log.Println("Starting server")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
