package utils

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func CleanProfane(chirp string) string {
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

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
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

func RespondWithError(w http.ResponseWriter, code int, msg string) {

	RespondWithJSON(w, code, map[string]string{"error": msg})

}
