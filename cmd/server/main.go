package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
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
	jwtSecret      string
	polkaKey       string
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
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
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

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not found in .env")
	}

	polkaKey := os.Getenv("POLKA_KEY")
	if polkaKey == "" {
		log.Fatal("POLKA_KEY not found in .env")
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
		jwtSecret:      jwtSecret,
		polkaKey:       polkaKey,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {

		type requestBody struct {
			Body string `json:"body"`
		}

		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}

		userID, err := auth.ValidateJWT(tokenString, apiCfg.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Expired or invalid jwt token")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decodeData := requestBody{}
		err = decoder.Decode(&decodeData)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if len(decodeData.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "chirps can only be 140 characters long")
			return
		}

		censoredBody := cleanProfane(decodeData.Body)
		dbChirp, err := apiCfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
			Body:   censoredBody,
			UserID: userID,
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

		authorQuery := r.URL.Query().Get("author_id")
		sortQuery := r.URL.Query().Get("sort")

		if authorQuery != "" {
			authorID, err := uuid.Parse(authorQuery)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid id format")
				return
			}
			dbChirpsByAuthor, err := apiCfg.dbQueries.GetChirpsByAuthor(r.Context(), authorID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to fetch author chirps")
				return
			}

			resultChirps := make([]Chirp, len(dbChirpsByAuthor))

			for i, chirp := range dbChirpsByAuthor {
				resultChirps[i] = Chirp{
					ID:        chirp.ID,
					CreatedAt: chirp.CreatedAt,
					UpdatedAt: chirp.UpdatedAt,
					Body:      chirp.Body,
					UserID:    chirp.UserID,
				}

			}

			if sortQuery == "desc" {
				sort.Slice(resultChirps, func(i, j int) bool {
					return resultChirps[i].CreatedAt.After(resultChirps[j].CreatedAt)
				})
			} else {
				sort.Slice(resultChirps, func(i, j int) bool {
					return resultChirps[i].CreatedAt.Before(resultChirps[j].CreatedAt)
				})
			}

			respondWithJSON(w, http.StatusOK, resultChirps)
			return
		}

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

		if sortQuery == "desc" {
			sort.Slice(chirps, func(i, j int) bool {
				return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
			})
		} else {
			sort.Slice(chirps, func(i, j int) bool {
				return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
			})
		}
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

	mux.HandleFunc("DELETE /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {

		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization token")
			return
		}

		userID, err := auth.ValidateJWT(tokenString, jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Expired or invalid jwt token")
			return
		}

		chirpNotValidated := r.PathValue("chirpID")

		chirpID, err := uuid.Parse(chirpNotValidated)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid id format")
			return
		}

		chirpValidated, err := apiCfg.dbQueries.GetChirpByID(r.Context(), chirpID)
		if err != nil {
			if err == sql.ErrNoRows {
				respondWithError(w, http.StatusNotFound, "chirp not found")
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Failed to get chirp by id")
			return
		}

		if chirpValidated.UserID != userID {
			respondWithError(w, http.StatusForbidden, "Only the owner of the chirp may delete it")
			return
		}

		err = apiCfg.dbQueries.DeleteChirp(r.Context(), database.DeleteChirpParams{
			ID:     chirpValidated.ID,
			UserID: userID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
		}

		w.WriteHeader(http.StatusNoContent)

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
			ID:          dbUser.ID,
			CreatedAt:   dbUser.CreatedAt,
			UpdatedAt:   dbUser.UpdatedAt,
			Email:       dbUser.Email,
			IsChirpyRed: dbUser.IsChirpyRed,
			// não colocar a senha de propósito por segurança
		}

		respondWithJSON(w, http.StatusCreated, user)

	})

	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		type requestBody struct {
			NewEmail    string `json:"email"`
			NewPassword string `json:"password"`
		}
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}

		userID, err := auth.ValidateJWT(tokenString, jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Expired or invalid jwt token")
			return
		}

		var req requestBody
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "failed to hash the password")
			return
		}

		updatedUser, err := apiCfg.dbQueries.UpdateUser(r.Context(), database.UpdateUserParams{
			ID:             userID,
			Email:          req.NewEmail,
			HashedPassword: hashedPassword,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to update the user")
			return
		}

		respondWithJSON(w, http.StatusOK, updatedUser)
	})

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

		token, err := auth.MakeJWT(dbUser.ID, jwtSecret, 1*time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create jwt")
			return
		}

		refreshToken, err := auth.MakeRefreshToken()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create refreshToken")
			return
		}

		user := User{
			ID:           dbUser.ID,
			CreatedAt:    dbUser.CreatedAt,
			UpdatedAt:    dbUser.UpdatedAt,
			Email:        dbUser.Email,
			Token:        token,
			RefreshToken: refreshToken,
			IsChirpyRed:  dbUser.IsChirpyRed,
		}

		_, err = apiCfg.dbQueries.StoreRefreshToken(r.Context(), database.StoreRefreshTokenParams{
			Token:  refreshToken,
			UserID: dbUser.ID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to store refresh token")
			return
		}

		respondWithJSON(w, http.StatusOK, user)

	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}

		userID, err := apiCfg.dbQueries.GetUserFromRefreshToken(r.Context(), refreshToken)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
			return
		}

		acessToken, err := auth.MakeJWT(userID, apiCfg.jwtSecret, 1*time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create acess token")
			return
		}
		reponse := struct {
			Token string `json:"token"`
		}{
			Token: acessToken,
		}

		respondWithJSON(w, http.StatusOK, reponse)
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}

		_, err = apiCfg.dbQueries.RevokeRefreshToken(r.Context(), refreshToken)
		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Failed to revoke refresh token")
			return
		}
		w.WriteHeader(http.StatusNoContent)
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

	mux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {

		apiKey, err := auth.GetAPIKey(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Failed to get header apiKey")
			return
		}

		if apiKey != apiCfg.polkaKey {
			respondWithError(w, http.StatusUnauthorized, "ApiKey doesn't match the server's")
			return
		}

		type webhookRequest struct {
			Event string `json:"event"`
			Data  struct {
				UserID string `json:"user_id"`
			} `json:"data"`
		}

		var req webhookRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Event != "user.upgraded" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		userID, err := uuid.Parse(req.Data.UserID)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid  json:user_id")
			return
		}

		_, err = apiCfg.dbQueries.UpgradeUserToChirpyRed(r.Context(), userID)
		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Failed to upgrade user")
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
