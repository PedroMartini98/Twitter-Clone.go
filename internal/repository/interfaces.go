package repository

import (
	"context"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/model"
	"github.com/google/uuid"
)

type ChirpRepository interface {
	CreateChirp(ctx context.Context, body string, userID uuid.UUID) (*model.Chirp, error)
	GetAllChirps(ctx context.Context) ([]model.Chirp, error)
	GetAllChirpsByAuthor(ctx context.Context, authorID uuid.UUID) ([]model.Chirp, error)
	GetChripByID(ctx context.Context, chirpID uuid.UUID) (*model.Chirp, error)
	DeleteChirp(ctx context.Context, chirpID uuid.UUID, authorID uuid.UUID) error
}

type UserRepository interface {
	CreateUser(ctx context.Context, email string, hashedPassword string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUser(ctx context.Context, email string, hashedPassword string, userID uuid.UUID) (*model.User, error)
	UpgradeUserToChirpyRed(ctx context.Context, userID uuid.UUID) error
	DeleteAllUsers(ctx context.Context) error
	StoreRefreshToken(ctx context.Context, token string, userID uuid.UUID) error
	GetUserFromRefreshToken(ctx context.Context, token string) (uuid.UUID, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}
