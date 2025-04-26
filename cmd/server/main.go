package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/PedroMartini98/Twitter-Clone.go.git/api/handlers"
	"github.com/PedroMartini98/Twitter-Clone.go.git/api/middleware"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/config"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	_ "github.com/lib/pq"
)

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

	mux := http.NewServeMux()

	metricsMiddleware := middleware.NewMetricsMiddleware()
	JwtMiddleware := middleware.NewJwtMiddleware(cfg.JwtSecret)

	adminHandler := handlers.NewAdminHandler(dbQueries, cfg.Platform, metricsMiddleware)
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
	mux.HandleFunc("POST /api/chirps", JwtMiddleware.Authenticate(chirpHandler.CreateChirp))

	mux.HandleFunc("DELETE /api/chirps/{chirpID}", JwtMiddleware.Authenticate(chirpHandler.DeleteChirp))

	mux.HandleFunc("GET /api/chirps", chirpHandler.GetChirps)

	mux.HandleFunc("GET /api/chirps/{chirpID}", chirpHandler.GetChirpByID)

	// User routes
	mux.HandleFunc("POST /api/users", userHandler.CreateUser)

	mux.HandleFunc("PUT /api/users", JwtMiddleware.Authenticate(userHandler.UpdateUser))

	// Webhooks routes
	mux.HandleFunc("POST /api/polka/webhooks", webhookHandler.Polka)

	// Me recuso a criar um handler pra isso
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	})

	mux.Handle("/app/", metricsMiddleware.IncHits(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + cfg.Port,
	}
	log.Printf("Server starting on %s", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
