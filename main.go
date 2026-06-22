package main

import (
	"net/http"
)

func handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
	w.WriteHeader(200)
}

func main() {
	port := "8080"
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", handle)
	srvr := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	srvr.ListenAndServe()
}
