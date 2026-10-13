package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
	DB      DB
	JWT     JWT
}

type JWT struct {
	Secret    string
	AccessTTL time.Duration
}

type DB struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	TimeZone string
}

// DSN returns the PostgreSQL connection string.
func (d DB) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.TimeZone,
	)
}

// Load reads configuration from the environment, using a .env file if present.
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppPort: getEnv("APP_PORT", "8080"),
		DB: DB{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "simrs"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			TimeZone: getEnv("DB_TIMEZONE", "Asia/Jakarta"),
		},
		JWT: JWT{
			Secret:    getEnv("JWT_SECRET", ""),
			AccessTTL: time.Duration(getEnvInt("JWT_ACCESS_TTL_HOURS", 8)) * time.Hour,
		},
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}

func (c *Config) Validate() error {
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be set and at least 32 characters long")
	}
	return nil
}
