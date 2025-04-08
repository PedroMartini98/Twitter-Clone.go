package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func CheckPasswordHash(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {

	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(expiresIn).Unix(),
		"iat": time.Now().Unix(),
		"iss": "chirpy",
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := newToken.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil

}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, jwt.ErrTokenNotValidYet
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	sub, ok := (*claims)["sub"].(string)
	if !ok {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}

	return userID, nil

}

func GetBearerToken(headers http.Header) (string, error) {

	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", errors.New("invalid Authorization header: missing Bearer prefix")
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if tokenString == "" {
		return "", errors.New("invalid Authorization header: empty token")
	}
	return tokenString, nil

}
func MakeRefreshToken() (string, error) {

	refreshToken := make([]byte, 32)
	_, err := rand.Read(refreshToken)
	if err != nil {
		return "", errors.New("Error generating code")
	}
	return hex.EncodeToString(refreshToken), nil
}
