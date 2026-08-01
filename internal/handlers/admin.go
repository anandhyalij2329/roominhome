package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"room-rental/internal/middleware"
	"room-rental/internal/models"
)

type adminStats struct {
	PropertiesTotal  int            `json:"properties_total"`
	PropertiesByType map[string]int `json:"properties_by_type"`
	PropertiesToday  int            `json:"properties_today"`
	PropertiesWeek   int            `json:"properties_week"`
	UsersTotal       int            `json:"users_total"`
	UsersByRole      map[string]int `json:"users_by_role"`
	Owners           int            `json:"owners"`
	Seekers          int            `json:"seekers"`
}

type adminUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Listings  int       `json:"listings"`
}

type adminProperty struct {
	models.Property
	OwnerName  string `json:"owner_name"`
	OwnerEmail string `json:"owner_email"`
}

// AdminStats GET /api/admin/stats
func (a *API) AdminStats(w http.ResponseWriter, r *http.Request) {
	stats := adminStats{
		PropertiesByType: map[string]int{"room": 0, "home": 0, "pg": 0, "shop": 0},
		UsersByRole:      map[string]int{"owner": 0, "seeker": 0, "admin": 0},
	}

	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM properties`).Scan(&stats.PropertiesTotal)
	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM properties WHERE date(created_at) = date('now')`).Scan(&stats.PropertiesToday)
	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM properties WHERE created_at >= datetime('now', '-7 days')`).Scan(&stats.PropertiesWeek)
	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&stats.UsersTotal)

	rows, err := a.DB.Query(`SELECT type, COUNT(*) FROM properties GROUP BY type`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var c int
			if rows.Scan(&t, &c) == nil {
				stats.PropertiesByType[t] = c
			}
		}
	}

	rows2, err := a.DB.Query(`SELECT role, COUNT(*) FROM users GROUP BY role`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var role string
			var c int
			if rows2.Scan(&role, &c) == nil {
				stats.UsersByRole[role] = c
				if role == "owner" {
					stats.Owners = c
				}
				if role == "seeker" {
					stats.Seekers = c
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, stats)
}

// AdminListUsers GET /api/admin/users
func (a *API) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	roleFilter := strings.TrimSpace(r.URL.Query().Get("role"))
	q := `SELECT u.id, u.name, u.email, u.role, u.phone, u.created_at,
		(SELECT COUNT(*) FROM properties p WHERE p.owner_id = u.id) AS listings
		FROM users u`
	args := []any{}
	if roleFilter != "" {
		q += ` WHERE u.role = ?`
		args = append(args, roleFilter)
	}
	q += ` ORDER BY u.created_at DESC`

	rows, err := a.DB.Query(q, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	defer rows.Close()

	list := []adminUser{}
	for rows.Next() {
		var u adminUser
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Phone, &u.CreatedAt, &u.Listings); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read users")
			return
		}
		list = append(list, u)
	}
	writeJSON(w, http.StatusOK, list)
}

// AdminListProperties GET /api/admin/properties?type=&q=
func (a *API) AdminListProperties(w http.ResponseWriter, r *http.Request) {
	ptype := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	query := propertySelect + ` WHERE 1=1`
	args := []any{}
	if ptype != "" {
		query += ` AND type = ?`
		args = append(args, ptype)
	}
	if q != "" {
		like := "%" + q + "%"
		query += ` AND (title LIKE ? OR city LIKE ? OR locality LIKE ? OR contact_phone LIKE ?)`
		args = append(args, like, like, like, like)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list properties")
		return
	}

	list := make([]adminProperty, 0)
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "could not read properties")
			return
		}
		list = append(list, adminProperty{Property: p})
	}
	rows.Close()

	ownerCache := map[string][2]string{}
	for i := range list {
		a.attachMediaAndMap(&list[i].Property)
		oid := list[i].OwnerID
		if cached, ok := ownerCache[oid]; ok {
			list[i].OwnerName, list[i].OwnerEmail = cached[0], cached[1]
			continue
		}
		var name, email string
		_ = a.DB.QueryRow(`SELECT name, email FROM users WHERE id = ?`, oid).Scan(&name, &email)
		ownerCache[oid] = [2]string{name, email}
		list[i].OwnerName, list[i].OwnerEmail = name, email
	}
	writeJSON(w, http.StatusOK, list)
}

// AdminDeleteProperty DELETE /api/admin/properties/{id}
func (a *API) AdminDeleteProperty(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := a.fetchProperty(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load property")
		return
	}
	_, _ = a.DB.Exec(`DELETE FROM bookings WHERE property_id = ?`, id)
	_, _ = a.DB.Exec(`DELETE FROM media WHERE property_id = ?`, id)
	_, err = a.DB.Exec(`DELETE FROM properties WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete property")
		return
	}
	_ = removePropertyUploadDir(a.UploadDir, id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "property deleted"})
}

// AdminDeleteUser DELETE /api/admin/users/{id}
func (a *API) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == middleware.UserID(r) {
		writeError(w, http.StatusBadRequest, "cannot delete your own admin account")
		return
	}
	var role string
	err := a.DB.QueryRow(`SELECT role FROM users WHERE id = ?`, id).Scan(&role)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load user")
		return
	}
	if role == "admin" {
		writeError(w, http.StatusForbidden, "cannot delete another admin")
		return
	}

	propRows, err := a.DB.Query(`SELECT id FROM properties WHERE owner_id = ?`, id)
	if err == nil {
		var ids []string
		for propRows.Next() {
			var pid string
			_ = propRows.Scan(&pid)
			ids = append(ids, pid)
		}
		propRows.Close()
		for _, pid := range ids {
			_, _ = a.DB.Exec(`DELETE FROM bookings WHERE property_id = ?`, pid)
			_, _ = a.DB.Exec(`DELETE FROM media WHERE property_id = ?`, pid)
			_, _ = a.DB.Exec(`DELETE FROM properties WHERE id = ?`, pid)
			_ = removePropertyUploadDir(a.UploadDir, pid)
		}
	}
	_, _ = a.DB.Exec(`DELETE FROM bookings WHERE seeker_id = ?`, id)
	_, err = a.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
