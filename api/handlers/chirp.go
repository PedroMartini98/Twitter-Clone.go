package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/auth"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/models"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/utils"
	"github.com/google/uuid"
)

type ChirpHandler struct {
	dbQueries *database.Queries
	jwtSecret string
}

func NewChirpHandler(dbQueries *database.Queries, jwtSecret string) *ChirpHandler {
	return &ChirpHandler{
		dbQueries: dbQueries,
		jwtSecret: jwtSecret,
	}
}

func (h *ChirpHandler) CreateChirp(w http.ResponseWriter, r *http.Request) {

	type requestBody struct {
		Body string `json:"body"`
	}

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}

	userID, err := auth.ValidateJWT(tokenString, h.jwtSecret)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Expired or invalid jwt token")
		return
	}

	decoder := json.NewDecoder(r.Body)
	decodeData := requestBody{}
	err = decoder.Decode(&decodeData)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(decodeData.Body) > 140 {
		utils.RespondWithError(w, http.StatusBadRequest, "chirps can only be 140 characters long")
		return
	}

	censoredBody := utils.CleanProfane(decodeData.Body)
	dbChirp, err := h.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   censoredBody,
		UserID: userID,
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create chirp")
		return
	}

	chirp := models.Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
	utils.RespondWithJSON(w, http.StatusCreated, chirp)

}

func (h *ChirpHandler) DeleteChirp(w http.ResponseWriter, r *http.Request) {

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization token")
		return
	}

	userID, err := auth.ValidateJWT(tokenString, h.jwtSecret)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Expired or invalid jwt token")
		return
	}

	chirpNotValidated := r.PathValue("chirpID")

	chirpID, err := uuid.Parse(chirpNotValidated)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid id format")
		return
	}

	chirpValidated, err := h.dbQueries.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(w, http.StatusNotFound, "chirp not found")
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get chirp by id")
		return
	}

	if chirpValidated.UserID != userID {
		utils.RespondWithError(w, http.StatusForbidden, "Only the owner of the chirp may delete it")
		return
	}

	err = h.dbQueries.DeleteChirp(r.Context(), database.DeleteChirpParams{
		ID:     chirpValidated.ID,
		UserID: userID,
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
	}

	w.WriteHeader(http.StatusNoContent)

}

func (h *ChirpHandler) GetChirps(w http.ResponseWriter, r *http.Request) {

	authorQuery := r.URL.Query().Get("author_id")
	sortQuery := r.URL.Query().Get("sort")

	var dbChirps []database.Chirp
	var err error

	if authorQuery != "" {
		authorID, err := uuid.Parse(authorQuery)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid id format")
			return
		}
		dbChirps, err = h.dbQueries.GetChirpsByAuthor(r.Context(), authorID)
	} else {
		dbChirps, err = h.dbQueries.GetAllChirps(r.Context())
	}
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch chirps")
		return
	}
	chirps := make([]models.Chirp, len(dbChirps))
	for i, chirp := range dbChirps {
		chirps[i] = models.Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
	}

	if sortQuery == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
		})
	} else {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
		})
	}
	utils.RespondWithJSON(w, http.StatusOK, chirps)
}

func (h *ChirpHandler) GetChirpByID(w http.ResponseWriter, r *http.Request) {

	chirpIdNotValidated := r.PathValue("chirpId")
	chirpId, err := uuid.Parse(chirpIdNotValidated)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Please enter valid id format")
		return
	}

	chirpInDB, err := h.dbQueries.GetChirpByID(r.Context(), chirpId)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Failed to get chirp id")
		return
	}

	chirp := models.Chirp{
		ID:        chirpInDB.ID,
		CreatedAt: chirpInDB.CreatedAt,
		UpdatedAt: chirpInDB.UpdatedAt,
		Body:      chirpInDB.Body,
		UserID:    chirpInDB.UserID,
	}
	utils.RespondWithJSON(w, http.StatusOK, chirp)

}
