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

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/auth"
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

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
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
		log.Printf("Error marshaling data:%s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return

	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
func respondWithError(w http.ResponseWriter, code int, msg string) {

	respondWithJSON(w, code, map[string]string{"error": msg})

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

	//j

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {

		type requestBody struct {
			Body   string    `json:"body"`
			UserID uuid.UUID `json:"user_id"`
		}

		decoder := json.NewDecoder(r.Body)
		decodeData := requestBody{}
		err := decoder.Decode(&decodeData)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "invalid request body")
			return
		}
		if len(decodeData.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "chirps can only be 140 characters long")
			return
		}

		censoredBody := cleanProfane(decodeData.Body)
		dbChirp, err := apiCfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
			Body:   censoredBody,
			UserID: decodeData.UserID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "failed to create chirp")
			return
		}

		chirp := Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		}
		respondWithJSON(w, http.StatusCreated, chirp)

	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {

		dbChirps, err := apiCfg.dbQueries.GetAllChirps(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch chirps")
			return
		}
		log.Printf("dbChirps: %+v", dbChirps)
		chirps := make([]Chirp, len(dbChirps))
		for i, chirp := range dbChirps {
			chirps[i] = Chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}
		}
		log.Printf("chirps: %+v", chirps)

		respondWithJSON(w, http.StatusOK, chirps)
	})

	mux.HandleFunc("GET /api/chirps/{chirpId}", func(w http.ResponseWriter, r *http.Request) {

		chirpIdNotValidated := r.PathValue("chirpId")
		chirpId, err := uuid.Parse(chirpIdNotValidated)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Please enter valid id format")
			return
		}

		chirpInDB, err := apiCfg.dbQueries.GetChirpByID(r.Context(), chirpId)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Failed to get chirp id")
			return
		}

		chirp := Chirp{
			ID:        chirpInDB.ID,
			CreatedAt: chirpInDB.CreatedAt,
			UpdatedAt: chirpInDB.UpdatedAt,
			Body:      chirpInDB.Body,
			UserID:    chirpInDB.UserID,
		}
		respondWithJSON(w, http.StatusOK, chirp)

	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {

		type requestBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		decoder := json.NewDecoder(r.Body)
		var req requestBody
		if err := decoder.Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		hashPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		dbUser, err := apiCfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email: req.Email, HashedPassword: hashPassword})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		user := User{
			ID:        dbUser.ID,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
			Email:     dbUser.Email,
			// não colocar a senha de propósito por segurança
		}

		respondWithJSON(w, http.StatusCreated, user)

	})

	// k
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {

		type requestBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		decoder := json.NewDecoder(r.Body)
		var req requestBody
		err := decoder.Decode(&req)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Error deconding json")
			return
		}

		dbUser, err := apiCfg.dbQueries.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "User not found")
			return
		}

		err = auth.CheckPasswordHash(dbUser.HashedPassword, req.Password)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid password")
			return
		}

		user := User{
			ID:        dbUser.ID,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
			Email:     dbUser.Email,
		}

		respondWithJSON(w, http.StatusOK, user)

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
			respondWithError(w, http.StatusInternalServerError, "Failed to delete all users")
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
