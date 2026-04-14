package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	handler := newHandler()
	log.Printf("specd listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler))
}
