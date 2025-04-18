package postgres

import (
	"context"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/database"
	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/model"
	"github.com/google/uuid"
)

type userRepository struct {
	queries *database.Queries
}

func NewUserRepository(queries *database.Queries) UserRepository {
	return &userRepository{
		queries: queries,
	}
}

func (r *userRepository) CreateUser(ctx context.Context, email, hashedPassword string) (model.User, error) {
	dbUser, err := r.queries.CreateUser(ctx, database.CreateUserParams{
		Email:          email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return model.User{}, err
	}

	return model.User{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		IsChirpyRed: dbUser.IsChirpyRed,
	}, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return model.User{}, err
	}

	return model.User{ID: dbUser.ID,
		Email:       dbUser.Email,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		IsChirpyRed: dbUser.IsChirpyRed}, nil

}

func (r *userRepository) UpdateUser(ctx context.Context, userID uuid.UUID, email, hashedPassword string) (model.User, error) {
	dbUser, err := r.queries.UpdateUser(ctx, database.UpdateUserParams{ID: userID, Email: email, HashedPassword: hashedPassword})
	if err != nil {
		return model.User{}, err
	}

	return model.User{
		ID:          dbUser.ID,
		Email:       email,
		UpdatedAt:   dbUser.UpdatedAt,
		CreatedAt:   dbUser.CreatedAt,
		IsChirpyRed: dbUser.IsChirpyRed,
	}, nil
}

func (r *userRepository) UpgradeToChirpyRed(ctx context.Context, userID uuid.UUID) error {
	_, err := r.queries.UpgradeUserToChirpyRed(ctx, userID)
	if err != nil {
		return err
	}

	return nil
}
