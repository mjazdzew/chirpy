package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
	"github.com/mjazdzew/chirpy/internal/database"

	_ "github.com/lib/pq"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
	w.WriteHeader(200)
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
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
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}
	cfg.fileserverHits.Store(0)
	err := cfg.dbQueries.DeleteAllUsers(r.Context())
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
}

func (cfg *apiConfig) chirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body   string `json:"body"`
		UserId string `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	r_ch := chirp{}
	err := decoder.Decode(&r_ch)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(r_ch.Body) > 140 {
		w.WriteHeader(400)
		w.Write([]byte("{\"error\": \"Chirp is too long\"}"))
		return
	}
	re := regexp.MustCompile(`(?i)kerfuffle|(?i)sharbert|(?i)fornax`)
	cleaned_body := re.ReplaceAllString(r_ch.Body, "****")

	db_chirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{Body: cleaned_body, UserID: r_ch.UserId})
	if err != nil {
		w.WriteHeader(500)
		return
	}

	type Chirp struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    string    `json:"user_id"`
	}

	ch := Chirp{
		ID:        db_chirp.ID,
		CreatedAt: db_chirp.CreatedAt,
		UpdatedAt: db_chirp.UpdatedAt,
		Body:      db_chirp.Body,
		UserID:    db_chirp.UserID,
	}
	data, err := json.Marshal(ch)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(data)
}

func (cfg *apiConfig) create_user(w http.ResponseWriter, r *http.Request) {
	type email struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	e := email{}
	err := decoder.Decode(&e)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	db_user, err := cfg.dbQueries.CreateUser(r.Context(), e.Email)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	type user struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	u := user{
		ID:        db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email:     db_user.Email,
	}

	data, err := json.Marshal(u)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(201)
	w.Write(data)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, _ := sql.Open("postgres", dbURL)
	port := "8080"
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      database.New(db),
		platform:       os.Getenv("PLATFORM"),
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthz)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.reset)
	mux.HandleFunc("POST /api/users", apiCfg.create_user)
	mux.HandleFunc("POST /api/chirps", apiCfg.chirp)
	srvr := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	srvr.ListenAndServe()
}
