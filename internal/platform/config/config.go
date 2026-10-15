package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppHost string
	AppPort string
	DB      DB
	JWT     JWT
	CORS    CORS
	Storage Storage
}

type CORS struct {
	AllowedOrigins []string
}

// Storage configures the MinIO (S3-compatible) object storage used for file uploads. It is
// optional: with an empty Endpoint, file upload is switched off.
type Storage struct {
	Endpoint  string // host:port the API server uses to reach MinIO
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool

	// PublicEndpoint is the host:port clients use to reach MinIO. Presigned URLs are signed for
	// this host, so it must be an address the browser can open. Defaults to Endpoint.
	PublicEndpoint string
	PublicUseSSL   bool
}

func (s Storage) Enabled() bool { return s.Endpoint != "" }

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
		AppHost: getEnv("APP_HOST", "0.0.0.0"),
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
		CORS: CORS{
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
		},
		Storage: loadStorage(),
	}
}

func loadStorage() Storage {
	endpoint := getEnv("MINIO_ENDPOINT", "")
	useSSL := getEnvBool("MINIO_USE_SSL", false)
	return Storage{
		Endpoint:       endpoint,
		AccessKey:      getEnv("MINIO_ACCESS_KEY", ""),
		SecretKey:      getEnv("MINIO_SECRET_KEY", ""),
		Bucket:         getEnv("MINIO_BUCKET", ""),
		Region:         getEnv("MINIO_REGION", "us-east-1"),
		UseSSL:         useSSL,
		PublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", endpoint),
		PublicUseSSL:   getEnvBool("MINIO_PUBLIC_USE_SSL", useSSL),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
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

func splitCSV(v string) []string {
	var out []string
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func (c *Config) Validate() error {
	for _, origin := range c.CORS.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must list explicit origins, wildcard \"*\" is not allowed")
		}
	}

	if c.Storage.Enabled() {
		st := c.Storage
		if st.AccessKey == "" || st.SecretKey == "" || st.Bucket == "" {
			return fmt.Errorf("MINIO_ACCESS_KEY, MINIO_SECRET_KEY and MINIO_BUCKET are required when MINIO_ENDPOINT is set")
		}
	}

	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be set and at least 32 characters long")
	}
	return nil
}
