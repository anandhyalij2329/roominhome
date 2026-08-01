package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"room-rental/internal/middleware"
	"room-rental/internal/models"
)

type locationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
	Locality  string  `json:"locality"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Pincode   string  `json:"pincode"`
	Landmark  string  `json:"landmark"`
}

// UpdateLocation PUT /api/properties/{id}/location
func (a *API) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prop, err := a.fetchProperty(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if prop.OwnerID != middleware.UserID(r) {
		writeError(w, http.StatusForbidden, "only the owner can update location")
		return
	}

	var req locationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if msg := validateLatLng(req.Latitude, req.Longitude); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	if strings.TrimSpace(req.Address) != "" {
		prop.Address = strings.TrimSpace(req.Address)
	}
	if strings.TrimSpace(req.Locality) != "" {
		prop.Locality = strings.TrimSpace(req.Locality)
	}
	if strings.TrimSpace(req.City) != "" {
		prop.City = strings.TrimSpace(req.City)
	}
	if strings.TrimSpace(req.State) != "" {
		prop.State = strings.TrimSpace(req.State)
	}
	if strings.TrimSpace(req.Pincode) != "" {
		prop.Pincode = strings.TrimSpace(req.Pincode)
	}
	if strings.TrimSpace(req.Landmark) != "" {
		prop.Landmark = strings.TrimSpace(req.Landmark)
	}
	prop.Latitude = req.Latitude
	prop.Longitude = req.Longitude
	prop.UpdatedAt = time.Now().UTC()

	_, err = a.DB.Exec(
		`UPDATE properties SET latitude=?, longitude=?, address=?, locality=?, city=?, state=?, pincode=?, landmark=?, updated_at=?
		 WHERE id=?`,
		prop.Latitude, prop.Longitude, prop.Address, prop.Locality, prop.City, prop.State, prop.Pincode, prop.Landmark,
		prop.UpdatedAt, id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update location")
		return
	}

	a.attachMediaAndMap(&prop)
	writeJSON(w, http.StatusOK, prop)
}

// PropertyMap GET /api/properties/{id}/map
func (a *API) PropertyMap(w http.ResponseWriter, r *http.Request) {
	prop, err := a.fetchProperty(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if prop.Latitude == 0 && prop.Longitude == 0 {
		writeError(w, http.StatusBadRequest, "property has no map coordinates yet")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"property_id":     prop.ID,
		"title":           prop.Title,
		"latitude":        prop.Latitude,
		"longitude":       prop.Longitude,
		"address":         prop.Address,
		"locality":        prop.Locality,
		"city":            prop.City,
		"map_url":         openStreetMapURL(prop.Latitude, prop.Longitude),
		"google_map_url":  googleMapsURL(prop.Latitude, prop.Longitude),
		"embed_osm_url":   osmEmbedURL(prop.Latitude, prop.Longitude),
	})
}

// NearbyProperties GET /api/map/nearby?lat=&lng=&radius_km=5
func (a *API) NearbyProperties(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, err1 := strconv.ParseFloat(q.Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(q.Get("lng"), 64)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "lat and lng query params are required")
		return
	}
	if msg := validateLatLng(lat, lng); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	radiusKm := 5.0
	if v := q.Get("radius_km"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 && n <= 100 {
			radiusKm = n
		}
	}
	propType := strings.TrimSpace(q.Get("type"))
	status := q.Get("status")
	if status == "" {
		status = string(models.StatusAvailable)
	}

	// Bounding box (~1 degree lat ≈ 111 km)
	delta := radiusKm / 111.0
	lngDelta := radiusKm / (111.0 * math.Cos(lat*math.Pi/180))
	if lngDelta < 0.001 {
		lngDelta = delta
	}

	query := propertySelect + ` WHERE latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?`
	args := []any{lat - delta, lat + delta, lng - lngDelta, lng + lngDelta}
	if status != "all" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if propType != "" {
		query += ` AND type = ?`
		args = append(args, propType)
	}

	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not search nearby")
		return
	}

	list := make([]models.Property, 0)
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "could not read properties")
			return
		}
		dist := haversineKm(lat, lng, p.Latitude, p.Longitude)
		if dist > radiusKm {
			continue
		}
		p.DistanceKm = math.Round(dist*100) / 100
		list = append(list, p)
	}
	rows.Close()

	for i := range list {
		a.attachMediaAndMap(&list[i])
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].DistanceKm < list[j].DistanceKm
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"center":     map[string]float64{"lat": lat, "lng": lng},
		"radius_km":  radiusKm,
		"count":      len(list),
		"properties": list,
	})
}

func validateLatLng(lat, lng float64) string {
	if lat < -90 || lat > 90 {
		return "latitude must be between -90 and 90"
	}
	if lng < -180 || lng > 180 {
		return "longitude must be between -180 and 180"
	}
	if lat == 0 && lng == 0 {
		return "latitude and longitude are required (pick location on map)"
	}
	return ""
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earth = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earth * math.Asin(math.Sqrt(a))
}

func osmEmbedURL(lat, lng float64) string {
	delta := 0.01
	return "https://www.openstreetmap.org/export/embed.html?bbox=" +
		strconv.FormatFloat(lng-delta, 'f', 6, 64) + "%2C" +
		strconv.FormatFloat(lat-delta, 'f', 6, 64) + "%2C" +
		strconv.FormatFloat(lng+delta, 'f', 6, 64) + "%2C" +
		strconv.FormatFloat(lat+delta, 'f', 6, 64) +
		"&layer=mapnik&marker=" +
		strconv.FormatFloat(lat, 'f', 6, 64) + "%2C" +
		strconv.FormatFloat(lng, 'f', 6, 64)
}
