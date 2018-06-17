package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path)) // nolint gas
	})

	log.Println("Starting server")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
