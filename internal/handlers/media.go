package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"room-rental/internal/middleware"
	"room-rental/internal/models"
)

var photoExts = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".png": "image/png", ".webp": "image/webp", ".gif": "image/gif",
}

var videoExts = map[string]string{
	".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime",
}

const (
	maxPhotosPerProperty = 20
	maxVideosPerProperty = 5
)

// UploadMedia accepts multipart form field "files" (multiple) or "file" (single).
// POST /api/properties/{id}/media
func (a *API) UploadMedia(w http.ResponseWriter, r *http.Request) {
	propertyID := r.PathValue("id")
	prop, err := a.fetchProperty(propertyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if prop.OwnerID != middleware.UserID(r) {
		writeError(w, http.StatusForbidden, "only the owner can upload media")
		return
	}

	maxBytes := a.MaxVideoBytes + (10 << 20) // buffer above largest allowed file
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form or file too large")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		files = r.MultipartForm.File["file"]
	}
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, `use form field "files" or "file"`)
		return
	}

	photoCount, videoCount, err := a.countMedia(propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check media limits")
		return
	}

	uploaded := make([]models.Media, 0, len(files))
	for _, fh := range files {
		media, errMsg := a.saveOneMedia(propertyID, fh, &photoCount, &videoCount)
		if errMsg != "" {
			writeError(w, http.StatusBadRequest, errMsg)
			return
		}
		uploaded = append(uploaded, media)
	}

	writeJSON(w, http.StatusCreated, uploaded)
}

func (a *API) saveOneMedia(propertyID string, fh *multipart.FileHeader, photoCount, videoCount *int) (models.Media, string) {
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	mime, isPhoto := photoExts[ext]
	if !isPhoto {
		var ok bool
		mime, ok = videoExts[ext]
		if !ok {
			return models.Media{}, "unsupported file type (use jpg/png/webp/gif or mp4/webm/mov)"
		}
	}

	mediaType := models.MediaPhoto
	maxSize := a.MaxPhotoBytes
	if !isPhoto {
		mediaType = models.MediaVideo
		maxSize = a.MaxVideoBytes
		if *videoCount >= maxVideosPerProperty {
			return models.Media{}, fmt.Sprintf("max %d videos per property", maxVideosPerProperty)
		}
	} else if *photoCount >= maxPhotosPerProperty {
		return models.Media{}, fmt.Sprintf("max %d photos per property", maxPhotosPerProperty)
	}

	if fh.Size > maxSize {
		mb := maxSize / (1 << 20)
		return models.Media{}, fmt.Sprintf("%s exceeds max size of %d MB", mediaType, mb)
	}

	src, err := fh.Open()
	if err != nil {
		return models.Media{}, "could not open uploaded file"
	}
	defer src.Close()

	id := uuid.NewString()
	safeName := id + ext
	subdir := string(mediaType) + "s" // photos / videos
	dir := filepath.Join(a.UploadDir, propertyID, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return models.Media{}, "could not create upload directory"
	}
	relPath := filepath.ToSlash(filepath.Join(propertyID, subdir, safeName))
	absPath := filepath.Join(a.UploadDir, filepath.FromSlash(relPath))

	dst, err := os.Create(absPath)
	if err != nil {
		return models.Media{}, "could not save file"
	}
	written, err := io.Copy(dst, src)
	dst.Close()
	if err != nil {
		_ = os.Remove(absPath)
		return models.Media{}, "could not write file"
	}

	isCover := false
	if mediaType == models.MediaPhoto && *photoCount == 0 {
		isCover = true
	}
	sortOrder := *photoCount + *videoCount
	now := time.Now().UTC()

	_, err = a.DB.Exec(
		`INSERT INTO media (id, property_id, type, file_path, file_name, mime_type, size_bytes, is_cover, sort_order, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, propertyID, mediaType, relPath, fh.Filename, mime, written, boolToInt(isCover), sortOrder, now,
	)
	if err != nil {
		_ = os.Remove(absPath)
		return models.Media{}, "could not save media record"
	}

	if mediaType == models.MediaPhoto {
		*photoCount++
	} else {
		*videoCount++
	}

	return models.Media{
		ID:         id,
		PropertyID: propertyID,
		Type:       mediaType,
		URL:        a.mediaPublicURL(relPath),
		FileName:   fh.Filename,
		MimeType:   mime,
		SizeBytes:  written,
		IsCover:    isCover,
		SortOrder:  sortOrder,
		CreatedAt:  now,
	}, ""
}

// ListMedia GET /api/properties/{id}/media
func (a *API) ListMedia(w http.ResponseWriter, r *http.Request) {
	propertyID := r.PathValue("id")
	if _, err := a.fetchProperty(propertyID); err != nil {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	list, err := a.loadMedia(propertyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list media")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// DeleteMedia DELETE /api/properties/{id}/media/{mediaId}
func (a *API) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	propertyID := r.PathValue("id")
	mediaID := r.PathValue("mediaId")

	prop, err := a.fetchProperty(propertyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if prop.OwnerID != middleware.UserID(r) {
		writeError(w, http.StatusForbidden, "only the owner can delete media")
		return
	}

	var filePath string
	var isCover int
	err = a.DB.QueryRow(
		`SELECT file_path, is_cover FROM media WHERE id = ? AND property_id = ?`,
		mediaID, propertyID,
	).Scan(&filePath, &isCover)
	if err != nil {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}

	_, _ = a.DB.Exec(`DELETE FROM media WHERE id = ?`, mediaID)
	_ = os.Remove(filepath.Join(a.UploadDir, filepath.FromSlash(filePath)))

	if isCover == 1 {
		_, _ = a.DB.Exec(
			`UPDATE media SET is_cover = 1 WHERE id = (
				SELECT id FROM media WHERE property_id = ? AND type = 'photo' ORDER BY sort_order LIMIT 1
			)`, propertyID,
		)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "media deleted"})
}

// SetCoverPhoto POST /api/properties/{id}/media/{mediaId}/cover
func (a *API) SetCoverPhoto(w http.ResponseWriter, r *http.Request) {
	propertyID := r.PathValue("id")
	mediaID := r.PathValue("mediaId")

	prop, err := a.fetchProperty(propertyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if prop.OwnerID != middleware.UserID(r) {
		writeError(w, http.StatusForbidden, "only the owner can set cover")
		return
	}

	var mediaType string
	err = a.DB.QueryRow(
		`SELECT type FROM media WHERE id = ? AND property_id = ?`, mediaID, propertyID,
	).Scan(&mediaType)
	if err != nil {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}
	if mediaType != string(models.MediaPhoto) {
		writeError(w, http.StatusBadRequest, "only a photo can be cover")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update cover")
		return
	}
	defer tx.Rollback()
	_, _ = tx.Exec(`UPDATE media SET is_cover = 0 WHERE property_id = ?`, propertyID)
	_, err = tx.Exec(`UPDATE media SET is_cover = 1 WHERE id = ?`, mediaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not set cover")
		return
	}
	_ = tx.Commit()
	writeJSON(w, http.StatusOK, map[string]string{"message": "cover updated"})
}

func (a *API) countMedia(propertyID string) (photos, videos int, err error) {
	err = a.DB.QueryRow(
		`SELECT
			COALESCE(SUM(CASE WHEN type='photo' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type='video' THEN 1 ELSE 0 END), 0)
		 FROM media WHERE property_id = ?`, propertyID,
	).Scan(&photos, &videos)
	return
}

func (a *API) loadMedia(propertyID string) ([]models.Media, error) {
	rows, err := a.DB.Query(
		`SELECT id, property_id, type, file_path, file_name, mime_type, size_bytes, is_cover, sort_order, created_at
		 FROM media WHERE property_id = ? ORDER BY is_cover DESC, sort_order ASC, created_at ASC`,
		propertyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]models.Media, 0)
	for rows.Next() {
		var m models.Media
		var path string
		var cover int
		if err := rows.Scan(
			&m.ID, &m.PropertyID, &m.Type, &path, &m.FileName, &m.MimeType, &m.SizeBytes,
			&cover, &m.SortOrder, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.IsCover = cover == 1
		m.URL = a.mediaPublicURL(path)
		list = append(list, m)
	}
	return list, nil
}

func (a *API) mediaPublicURL(relPath string) string {
	if strings.HasPrefix(relPath, "http://") || strings.HasPrefix(relPath, "https://") {
		return relPath
	}
	base := strings.TrimRight(a.PublicBaseURL, "/")
	return base + "/uploads/" + strings.TrimLeft(filepath.ToSlash(relPath), "/")
}

func (a *API) attachMediaAndMap(p *models.Property) {
	media, err := a.loadMedia(p.ID)
	if err != nil {
		media = []models.Media{}
	}
	p.Media = media
	if p.Latitude != 0 || p.Longitude != 0 {
		p.MapURL = openStreetMapURL(p.Latitude, p.Longitude)
		p.GoogleMapURL = googleMapsURL(p.Latitude, p.Longitude)
		p.DirectionsURL = googleDirectionsURL(p.Latitude, p.Longitude)
	}
	if p.OwnerID != "" {
		var name, phone string
		_ = a.DB.QueryRow(`SELECT name, phone FROM users WHERE id = ?`, p.OwnerID).Scan(&name, &phone)
		p.OwnerName = name
		p.OwnerPhone = phone
		if p.ContactPhone == "" {
			p.ContactPhone = phone
		}
	}
}

func openStreetMapURL(lat, lng float64) string {
	return fmt.Sprintf("https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=16/%.6f/%.6f", lat, lng, lat, lng)
}

func googleMapsURL(lat, lng float64) string {
	return fmt.Sprintf("https://www.google.com/maps?q=%.6f,%.6f", lat, lng)
}

func googleDirectionsURL(lat, lng float64) string {
	return fmt.Sprintf("https://www.google.com/maps/dir/?api=1&destination=%.6f,%.6f", lat, lng)
}
