package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync/atomic"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
	w.WriteHeader(200)
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())))
	w.WriteHeader(200)
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(200)
}

func validate_chirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	ch := chirp{}
	err := decoder.Decode(&ch)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(ch.Body) > 140 {
		w.WriteHeader(400)
		w.Write([]byte("{\"error\": \"Chirp is too long\"}"))
		return
	}
	re := regexp.MustCompile(`(?i)kerfuffle|(?i)sharbert|(?i)fornax`)
	cleaned_body := re.ReplaceAllString(ch.Body, "****")
	w.WriteHeader(200)
	w.Write([]byte("{\"cleaned_body\": \"" + cleaned_body + "\"}"))
}

func main() {
	port := "8080"
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthz)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.reset)
	mux.HandleFunc("POST /api/validate_chirp", validate_chirp)
	srvr := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	srvr.ListenAndServe()
}
