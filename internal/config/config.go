package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DBPath        string
	JWTSecret     string
	UploadDir     string
	PublicBaseURL string // e.g. http://localhost:8080
	MaxPhotoMB    int64
	MaxVideoMB    int64
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		DBPath:        getEnv("DB_PATH", "room_rental.db"),
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		UploadDir:     getEnv("UPLOAD_DIR", "uploads"),
		PublicBaseURL: getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		MaxPhotoMB:    int64(getEnvInt("MAX_PHOTO_MB", 5)),
		MaxVideoMB:    int64(getEnvInt("MAX_VIDEO_MB", 50)),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
