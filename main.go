package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type errorResponse struct {
	Error string
}

type successResponse struct {
	Valid bool
}

type chirpResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) middlewareMetricsInc(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cfg.fileserverHits.Add(1)
		nextHandler.ServeHTTP(w, r)

	})
}
func cleanProfane(chirp string) string {
	profaneWords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}
	words := strings.Split(chirp, " ")
	for i, w := range words {
		wordLower := strings.ToLower(w)
		// Check if the word exactly matches a profane word (no punctuation)
		if profaneWords[wordLower] {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Erro marshaling data:%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func main() {

	errENV := godotenv.Load()
	if errENV != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL not found in .env")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM not found in .env")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      dbQueries,
		platform:       platform,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {

		type Chirp struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}

		type requestBody struct {
			Body   string    `json:"body"`
			UserID uuid.UUID `json:"user_id"`
		}

		decoder := json.NewDecoder(r.Body)
		decodeData := requestBody{}
		err := decoder.Decode(&decodeData)
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid request body"})
			return
		}
		if len(decodeData.Body) > 140 {
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "chirps can only be 140 characters long"})
			return
		}

		censoredBody := cleanProfane(decodeData.Body)
		dbChirp, err := apiCfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
			Body:   censoredBody,
			UserID: decodeData.UserID,
		})
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create chirp"})
			return
		}

		chirp := Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		}
		respondWithJSON(w, http.StatusOK, chirp)

	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {

		type requestBody struct {
			Email string `json:"email"`
		}

		decoder := json.NewDecoder(r.Body)
		var req requestBody
		if err := decoder.Decode(&req); err != nil {
			respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		dbUser, err := apiCfg.dbQueries.CreateUser(r.Context(), req.Email)
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
			return
		}

		user := User{
			ID:        dbUser.ID,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
			Email:     dbUser.Email,
		}

		respondWithJSON(w, http.StatusCreated, user)

	})

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		htmlTemplate := `
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

		fmt.Fprintf(w, htmlTemplate, apiCfg.fileserverHits.Load())

	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {

		if apiCfg.platform != "dev" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err := apiCfg.dbQueries.DeleteAllUsers(r.Context())
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete all users"})
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Users deleted"))
	})

	server := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	log.Printf("Server starting on %s", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
