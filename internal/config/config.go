package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL     string
	Platform  string
	JwtSecret string
	PolkaKey  string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("Failed to load env:%v", err)
	}

	config := &Config{
		DBURL:     os.Getenv("DB_URL"),
		Platform:  os.Getenv("PLATFORM"),
		JwtSecret: os.Getenv("JWT_SECRET"),
		PolkaKey:  os.Getenv("POLKA_KEY"),
	}

	if config.DBURL == "" {
		return nil, fmt.Errorf("DB_URL not found in .env")
	}

	if config.Platform == "" {
		return nil, fmt.Errorf("PLATFORM not found in .env")
	}

	if config.JwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET not found in .env")
	}

	if config.PolkaKey == "" {
		return nil, fmt.Errorf("POLKA_KEY not found in .env")
	}
	return config, nil
}
