package main

import (
	"net/http"
)

func main() {
	port := "8080"
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))
	srvr := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	srvr.ListenAndServe()
}
