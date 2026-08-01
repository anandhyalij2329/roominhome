package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"room-rental/internal/middleware"
	"room-rental/internal/models"
)

type bookingRequest struct {
	PropertyID string `json:"property_id"`
	StartDate  string `json:"start_date"` // YYYY-MM-DD
	EndDate    string `json:"end_date"`   // YYYY-MM-DD
	Message    string `json:"message"`
}

type bookingStatusRequest struct {
	Status string `json:"status"`
}

func (a *API) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req bookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	start, err1 := time.Parse("2006-01-02", req.StartDate)
	end, err2 := time.Parse("2006-01-02", req.EndDate)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "start_date and end_date must be YYYY-MM-DD")
		return
	}
	if !end.After(start) {
		writeError(w, http.StatusBadRequest, "end_date must be after start_date")
		return
	}

	prop, err := a.fetchProperty(req.PropertyID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load property")
		return
	}
	if prop.Status != models.StatusAvailable {
		writeError(w, http.StatusBadRequest, "property is not available")
		return
	}
	if prop.OwnerID == middleware.UserID(r) {
		writeError(w, http.StatusBadRequest, "cannot book your own property")
		return
	}
	if start.Before(prop.AvailableFrom.Truncate(24*time.Hour)) || end.After(prop.AvailableUntil.Add(24*time.Hour)) {
		writeError(w, http.StatusBadRequest, "booking dates must be within property availability window")
		return
	}

	now := time.Now().UTC()
	booking := models.Booking{
		ID:            uuid.NewString(),
		PropertyID:    prop.ID,
		SeekerID:      middleware.UserID(r),
		StartDate:     start,
		EndDate:       end,
		Message:       strings.TrimSpace(req.Message),
		Status:        models.BookingPending,
		CreatedAt:     now,
		UpdatedAt:     now,
		PropertyTitle: prop.Title,
	}

	_, err = a.DB.Exec(
		`INSERT INTO bookings (id, property_id, seeker_id, start_date, end_date, message, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		booking.ID, booking.PropertyID, booking.SeekerID, booking.StartDate, booking.EndDate,
		booking.Message, booking.Status, booking.CreatedAt, booking.UpdatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create booking")
		return
	}
	writeJSON(w, http.StatusCreated, booking)
}

func (a *API) MyBookings(w http.ResponseWriter, r *http.Request) {
	role := middleware.Role(r)
	userID := middleware.UserID(r)

	var rows *sql.Rows
	var err error

	if role == string(models.RoleOwner) {
		rows, err = a.DB.Query(
			`SELECT b.id, b.property_id, b.seeker_id, b.start_date, b.end_date, b.message, b.status,
				b.created_at, b.updated_at, p.title, u.name
			 FROM bookings b
			 JOIN properties p ON p.id = b.property_id
			 JOIN users u ON u.id = b.seeker_id
			 WHERE p.owner_id = ?
			 ORDER BY b.created_at DESC`, userID,
		)
	} else {
		rows, err = a.DB.Query(
			`SELECT b.id, b.property_id, b.seeker_id, b.start_date, b.end_date, b.message, b.status,
				b.created_at, b.updated_at, p.title, u.name
			 FROM bookings b
			 JOIN properties p ON p.id = b.property_id
			 JOIN users u ON u.id = b.seeker_id
			 WHERE b.seeker_id = ?
			 ORDER BY b.created_at DESC`, userID,
		)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list bookings")
		return
	}
	defer rows.Close()

	bookings := make([]models.Booking, 0)
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(
			&b.ID, &b.PropertyID, &b.SeekerID, &b.StartDate, &b.EndDate, &b.Message, &b.Status,
			&b.CreatedAt, &b.UpdatedAt, &b.PropertyTitle, &b.SeekerName,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read bookings")
			return
		}
		bookings = append(bookings, b)
	}
	writeJSON(w, http.StatusOK, bookings)
}

func (a *API) UpdateBookingStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req bookingStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	status := models.BookingStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	role := middleware.Role(r)
	userID := middleware.UserID(r)

	var booking models.Booking
	var ownerID string
	err := a.DB.QueryRow(
		`SELECT b.id, b.property_id, b.seeker_id, b.start_date, b.end_date, b.message, b.status,
			b.created_at, b.updated_at, p.owner_id
		 FROM bookings b JOIN properties p ON p.id = b.property_id WHERE b.id = ?`, id,
	).Scan(
		&booking.ID, &booking.PropertyID, &booking.SeekerID, &booking.StartDate, &booking.EndDate,
		&booking.Message, &booking.Status, &booking.CreatedAt, &booking.UpdatedAt, &ownerID,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load booking")
		return
	}

	switch {
	case role == string(models.RoleOwner) && ownerID == userID:
		if status != models.BookingApproved && status != models.BookingRejected {
			writeError(w, http.StatusBadRequest, "owner can set approved or rejected")
			return
		}
	case role == string(models.RoleSeeker) && booking.SeekerID == userID:
		if status != models.BookingCancelled {
			writeError(w, http.StatusBadRequest, "seeker can only cancel")
			return
		}
	default:
		writeError(w, http.StatusForbidden, "not allowed to update this booking")
		return
	}

	now := time.Now().UTC()
	_, err = a.DB.Exec(`UPDATE bookings SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update booking")
		return
	}

	if status == models.BookingApproved {
		_, _ = a.DB.Exec(`UPDATE properties SET status = ?, updated_at = ? WHERE id = ?`,
			models.StatusRented, now, booking.PropertyID)
	}

	booking.Status = status
	booking.UpdatedAt = now
	writeJSON(w, http.StatusOK, booking)
}
