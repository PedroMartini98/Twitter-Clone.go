package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/PedroMartini98/Twitter-Clone.go.git/api/handlers"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/auth"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/config"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/utils"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cfg.fileserverHits.Add(1)
		nextHandler.ServeHTTP(w, r)

	})
}
func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config file")
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      dbQueries,
	}

	mux := http.NewServeMux()

	adminHandler := handlers.NewAdminHandler(dbQueries, cfg.Platform)
	authHandler := handlers.NewAuthHandler(dbQueries, cfg.JwtSecret)
	chirpHandler := handlers.NewChirpHandler(dbQueries, cfg.JwtSecret)
	userHandler := handlers.NewUserHandler(dbQueries, cfg.JwtSecret)
	webhookHandler := handlers.NewWeebHookHandler(dbQueries, cfg.PolkaKey)

	// Admin routes
	mux.HandleFunc("GET /admin/metrics", adminHandler.GetMetricts)

	mux.HandleFunc("POST /admin/reset", adminHandler.ResetServer)

	// Auth routes
	mux.HandleFunc("POST /api/login", authHandler.Login)

	mux.HandleFunc("POST /api/refresh", authHandler.RefreshToken)

	mux.HandleFunc("POST /api/revoke", authHandler.RevokeToken)

	//Chirp routes
	mux.HandleFunc("POST /api/chirps", chirpHandler.CreateChirp)

	mux.HandleFunc("DELETE /api/chirps/{chirpID}", chirpHandler.DeleteChirp)

	mux.HandleFunc("GET /api/chirps", chirpHandler.GetChirps)

	mux.HandleFunc("GET /api/chirps/{chirpId}", chirpHandler.GetChirpByID)

	// User routes
	mux.HandleFunc("POST /api/users", userHandler.CreateUser)

	mux.HandleFunc("PUT /api/users", userHandler.UpdateUser)

	// Webhooks routes
	mux.HandleFunc("POST /api/polka/webhooks", webhookHandler.Polka)

	// Acho desnecessario criar um handler pra isso
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	})

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

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
