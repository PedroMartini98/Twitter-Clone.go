package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/PedroMartini98/Twitter-Clone.go.git/internal/auth"
	"github.com/google/uuid"
)

type JwtMiddleware struct {
	JWTSecret string
}

func NewJwtMiddleware(JwtSecret string) *JwtMiddleware {
	return &JwtMiddleware{
		JWTSecret: JwtSecret,
	}
}

func (m *JwtMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.GetBearerToken(r.Header)
		if err != nil {
			http.Error(w, "Unauthorized: Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		userID, err := auth.ValidateJWT(tokenString, m.JWTSecret)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = contextWithUserID(ctx, userID)

		next.ServeHTTP(w, r.WithContext(ctx))

	}
}

type userIDKey struct{}

func contextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user ID not found in context")
	}
	if userID == uuid.Nil {
		return uuid.Nil, errors.New("invalid user ID in context")
	}
	return userID, nil
}
