package config

import (
	"errors"
	"os"
)

type Config struct {
	DBURL     string
	JWTSecret string
	Platform  string
	PolkaKey  string
}

func LoadConfig() (*Config, error) {

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, errors.New("DB_URL not found in enviroment")
	}

	JWTSecret := os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		return nil, errors.New("JWT_SECRET not found in enviroment")
	}

	Platform := os.Getenv("PLATFORM")
	if Platform == "" {
		return nil, errors.New("PLATFORM not found in enviroment")
	}

	PolkaKey := os.Getenv("POLKA_KEY")
	if PolkaKey == "" {
		return nil, errors.New("POLKA_KEY not found in enviroment")
	}

	return &Config{
		Platform:  Platform,
		PolkaKey:  PolkaKey,
		JWTSecret: JWTSecret,
		DBURL:     dbURL,
	}, nil

}
