package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/auth"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/models"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/utils"
)

type AuthHandler struct {
	dbQueries *database.Queries
	jwtSecret string
}

func NewAuthHandler(dbQueries *database.Queries, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		dbQueries: dbQueries,
		jwtSecret: jwtSecret,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {

	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	var req requestBody
	err := decoder.Decode(&req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Error deconding json")
		return
	}

	dbUser, err := h.dbQueries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	err = auth.CheckPasswordHash(dbUser.HashedPassword, req.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	token, err := auth.MakeJWT(dbUser.ID, h.jwtSecret, 1*time.Hour)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create jwt")
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create refreshToken")
		return
	}

	user := models.User{
		ID:           dbUser.ID,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
		Email:        dbUser.Email,
		Token:        token,
		RefreshToken: refreshToken,
		IsChirpyRed:  dbUser.IsChirpyRed,
	}

	_, err = h.dbQueries.StoreRefreshToken(r.Context(), database.StoreRefreshTokenParams{
		Token:  refreshToken,
		UserID: dbUser.ID,
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to store refresh token")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, user)

}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}

	userID, err := h.dbQueries.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	accessToken, err := auth.MakeJWT(userID, h.jwtSecret, 1*time.Hour)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create acess token")
		return
	}
	reponse := struct {
		Token string `json:"token"`
	}{
		Token: accessToken,
	}

	utils.RespondWithJSON(w, http.StatusOK, reponse)
}

func (h *AuthHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}

	_, err = h.dbQueries.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to revoke refresh token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
