package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/auth"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/models"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/utils"
)

type UserHandler struct {
	dbQueries *database.Queries
	jwtSecret string
}

func NewUserHandler(dbQueries *database.Queries, jwtSecret string) *UserHandler {
	return &UserHandler{
		dbQueries: dbQueries,
		jwtSecret: jwtSecret,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {

	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	var req requestBody
	if err := decoder.Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hashPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	dbUser, err := h.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email: req.Email, HashedPassword: hashPassword})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	user := models.User{
		ID:          dbUser.ID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
		// não colocar a senha de propósito por segurança
	}

	utils.RespondWithJSON(w, http.StatusCreated, user)

}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		NewEmail    string `json:"email"`
		NewPassword string `json:"password"`
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

	var req requestBody
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to hash the password")
		return
	}

	updatedUser, err := h.dbQueries.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userID,
		Email:          req.NewEmail,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update the user")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, updatedUser)
}
