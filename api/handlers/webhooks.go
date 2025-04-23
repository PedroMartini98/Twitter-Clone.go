package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/auth"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/utils"
	"github.com/google/uuid"
)

type webhookHandler struct {
	dbQueries *database.Queries
	polkaKey  string
}

func NewWeebHookHandler(dbQueries *database.Queries, polkaKey string) *webhookHandler {
	return &webhookHandler{
		dbQueries: dbQueries,
		polkaKey:  polkaKey}
}

func (h *webhookHandler) Polka(w http.ResponseWriter, r *http.Request) {

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Failed to get header apiKey")
		return
	}

	if apiKey != h.polkaKey {
		utils.RespondWithError(w, http.StatusUnauthorized, "ApiKey doesn't match the server's")
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
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(req.Data.UserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid  json:user_id")
		return
	}

	_, err = h.dbQueries.UpgradeUserToChirpyRed(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to upgrade user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
