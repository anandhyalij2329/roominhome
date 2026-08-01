package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"room-rental/internal/config"
	"room-rental/internal/database"
	"room-rental/internal/handlers"
	"room-rental/internal/middleware"
	"room-rental/internal/seed"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("upload dir: %v", err)
	}

	db, err := database.Connect(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if n, err := seed.SeedIfEmpty(db); err != nil {
		log.Printf("seed warning: %v", err)
	} else if n > 0 {
		log.Printf("seeded %d demo properties (empty database)", n)
	}
	api := &handlers.API{
		DB:            db,
		JWTSecret:     cfg.JWTSecret,
		UploadDir:     cfg.UploadDir,
		PublicBaseURL: cfg.PublicBaseURL,
		MaxPhotoBytes: cfg.MaxPhotoMB << 20,
		MaxVideoBytes: cfg.MaxVideoMB << 20,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("GET /api/meta", api.Meta)

	mux.HandleFunc("POST /api/auth/register", api.Register)
	mux.HandleFunc("POST /api/auth/login", api.Login)

	auth := middleware.Auth(cfg.JWTSecret)
	ownerOnly := middleware.RequireRole("owner")
	seekerOnly := middleware.RequireRole("seeker")

	mux.Handle("GET /api/me", auth(http.HandlerFunc(api.Me)))

	mux.HandleFunc("GET /api/properties", api.ListProperties)
	mux.HandleFunc("GET /api/properties/{id}", api.GetProperty)
	mux.Handle("POST /api/properties", auth(ownerOnly(http.HandlerFunc(api.CreateProperty))))
	mux.Handle("PUT /api/properties/{id}", auth(ownerOnly(http.HandlerFunc(api.UpdateProperty))))
	mux.Handle("DELETE /api/properties/{id}", auth(ownerOnly(http.HandlerFunc(api.DeleteProperty))))
	mux.Handle("GET /api/my/properties", auth(ownerOnly(http.HandlerFunc(api.MyProperties))))

	mux.HandleFunc("GET /api/properties/{id}/media", api.ListMedia)
	mux.Handle("POST /api/properties/{id}/media", auth(ownerOnly(http.HandlerFunc(api.UploadMedia))))
	mux.Handle("DELETE /api/properties/{id}/media/{mediaId}", auth(ownerOnly(http.HandlerFunc(api.DeleteMedia))))
	mux.Handle("POST /api/properties/{id}/media/{mediaId}/cover", auth(ownerOnly(http.HandlerFunc(api.SetCoverPhoto))))

	mux.HandleFunc("GET /api/map/nearby", api.NearbyProperties)
	mux.HandleFunc("GET /api/properties/{id}/map", api.PropertyMap)
	mux.Handle("PUT /api/properties/{id}/location", auth(ownerOnly(http.HandlerFunc(api.UpdateLocation))))

	mux.Handle("POST /api/bookings", auth(seekerOnly(http.HandlerFunc(api.CreateBooking))))
	mux.Handle("GET /api/bookings", auth(http.HandlerFunc(api.MyBookings)))
	mux.Handle("PATCH /api/bookings/{id}/status", auth(http.HandlerFunc(api.UpdateBookingStatus)))

	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir)))
	mux.Handle("GET /uploads/", fileServer)

	webDir := findWebDir()
	if webDir != "" {
		mux.Handle("/", spaFileServer(webDir))
	}

	handler := middleware.CORS(apiAwareJSON(mux))

	addr := ":" + cfg.Port
	fmt.Printf("RoomInHome running on http://localhost%s\n", addr)
	if webDir != "" {
		fmt.Printf("UI: http://localhost%s/  (web dir: %s)\n", addr, webDir)
	}
	fmt.Printf("Uploads: %s  |  Public URL: %s\n", cfg.UploadDir, cfg.PublicBaseURL)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func apiAwareJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
		}
		next.ServeHTTP(w, r)
	})
}

func findWebDir() string {
	candidates := []string{"web", filepath.Join("..", "web"), filepath.Join("..", "..", "web")}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func spaFileServer(webDir string) http.Handler {
	fs := http.FileServer(http.Dir(webDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path
		if path == "/" {
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		clean := filepath.Clean(strings.TrimPrefix(path, "/"))
		full := filepath.Join(webDir, clean)
		rel, err := filepath.Rel(webDir, full)
		if err != nil || strings.HasPrefix(rel, "..") {
			http.NotFound(w, r)
			return
		}
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			http.ServeFile(w, r, full)
			return
		}
		// let FileServer handle directories / 404
		fs.ServeHTTP(w, r)
	})
}
