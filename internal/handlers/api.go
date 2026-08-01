package handlers

import "database/sql"

// API holds shared dependencies for HTTP handlers.
type API struct {
	DB            *sql.DB
	JWTSecret     string
	UploadDir     string
	PublicBaseURL string
	MaxPhotoBytes int64
	MaxVideoBytes int64
}
