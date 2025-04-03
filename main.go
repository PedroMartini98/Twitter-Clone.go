package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
	apiCfg := &apiConfig{}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))

	})

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {

		type chirp struct {
			Body string
		}

		decoder := json.NewDecoder(r.Body)
		decodeData := chirp{}
		err := decoder.Decode(&decodeData)
		if err != nil {
			log.Printf("Error decoding chirp: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(decodeData.Body) > 140 {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("The chirp can only be 140 characters long")
			return
		}

		censoredBody := cleanProfane(decodeData.Body)
		response := chirpResponse{CleanedBody: censoredBody}
		respondWithJSON(w, http.StatusOK, response)
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

		apiCfg.fileserverHits.Store(0)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Counter reset to 0"))
	})

	server := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	log.Printf("Server starting on %s", server.Addr)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
