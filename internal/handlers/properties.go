package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"room-rental/internal/middleware"
	"room-rental/internal/models"
)

type propertyRequest struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`

	Address   string  `json:"address"`
	Locality  string  `json:"locality"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Pincode   string  `json:"pincode"`
	Landmark  string  `json:"landmark"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	Rent           float64 `json:"rent"`
	Deposit        float64 `json:"deposit"`
	AvailableFrom  string  `json:"available_from"`  // YYYY-MM-DD
	AvailableUntil string  `json:"available_until"` // YYYY-MM-DD

	ParkingTwoWheeler  bool `json:"parking_two_wheeler"`
	ParkingFourWheeler bool `json:"parking_four_wheeler"`

	PreferredTenants []string `json:"preferred_tenants"`
	Amenities        []string `json:"amenities"`
	ContactPhone     string   `json:"contact_phone"`
	OwnerName        string   `json:"owner_name"`

	Bedrooms         int     `json:"bedrooms"`
	Bathrooms        int     `json:"bathrooms"`
	AreaSqFt         float64 `json:"area_sq_ft"`
	Furnishing       string  `json:"furnishing"`
	SharingType      string  `json:"sharing_type"`
	GenderPreference string  `json:"gender_preference"`
	Floor            int     `json:"floor"`
	TotalFloors      int     `json:"total_floors"`
	BHK              int     `json:"bhk"`

	MealsIncluded bool   `json:"meals_included"`
	FoodType      string `json:"food_type"`

	ShopCategory string  `json:"shop_category"`
	FrontageFt   float64 `json:"frontage_ft"`
	PowerBackup  bool    `json:"power_backup"`
	Washroom     bool    `json:"washroom"`
}

const propertySelect = `SELECT id, owner_id, type, title, description, status,
	address, locality, city, state, pincode, landmark, latitude, longitude,
	rent, deposit, available_from, available_until,
	parking_two_wheeler, parking_four_wheeler,
	preferred_tenants, amenities, contact_phone,
	bedrooms, bathrooms, area_sq_ft, furnishing, sharing_type, gender_preference,
	floor, total_floors, bhk, meals_included, food_type,
	shop_category, frontage_ft, power_backup, washroom,
	created_at, updated_at FROM properties`

func (a *API) ListProperties(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	propType := strings.TrimSpace(q.Get("type"))
	city := strings.TrimSpace(q.Get("city"))
	locality := strings.TrimSpace(q.Get("locality"))
	pincode := strings.TrimSpace(q.Get("pincode"))
	tenant := strings.TrimSpace(q.Get("preferred_tenant"))
	sharing := strings.TrimSpace(q.Get("sharing_type"))
	shopCat := strings.TrimSpace(q.Get("shop_category"))
	status := q.Get("status")
	if status == "" {
		status = string(models.StatusAvailable)
	}
	minRent, _ := strconv.ParseFloat(q.Get("min_rent"), 64)
	maxRent, _ := strconv.ParseFloat(q.Get("max_rent"), 64)
	parking2 := q.Get("parking_two_wheeler")
	parking4 := q.Get("parking_four_wheeler")

	query := propertySelect + ` WHERE 1=1`
	args := []any{}

	if status != "all" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if propType != "" {
		query += ` AND type = ?`
		args = append(args, propType)
	}
	if city != "" {
		query += ` AND LOWER(city) = LOWER(?)`
		args = append(args, city)
	}
	if locality != "" {
		query += ` AND LOWER(locality) LIKE LOWER(?)`
		args = append(args, "%"+locality+"%")
	}
	if pincode != "" {
		query += ` AND pincode = ?`
		args = append(args, pincode)
	}
	if sharing != "" {
		query += ` AND sharing_type = ?`
		args = append(args, sharing)
	}
	if shopCat != "" {
		query += ` AND shop_category = ?`
		args = append(args, shopCat)
	}
	if tenant != "" {
		query += ` AND preferred_tenants LIKE ?`
		args = append(args, "%\""+tenant+"\"%")
	}
	if minRent > 0 {
		query += ` AND rent >= ?`
		args = append(args, minRent)
	}
	if maxRent > 0 {
		query += ` AND rent <= ?`
		args = append(args, maxRent)
	}
	if parking2 == "true" || parking2 == "1" {
		query += ` AND parking_two_wheeler = 1`
	}
	if parking4 == "true" || parking4 == "1" {
		query += ` AND parking_four_wheeler = 1`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list properties")
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
		list = append(list, p)
	}
	rows.Close()

	for i := range list {
		a.attachMediaAndMap(&list[i])
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) GetProperty(w http.ResponseWriter, r *http.Request) {
	p, err := a.fetchProperty(r.PathValue("id"))
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load property")
		return
	}
	a.attachMediaAndMap(&p)
	writeJSON(w, http.StatusOK, p)
}

func (a *API) CreateProperty(w http.ResponseWriter, r *http.Request) {
	var req propertyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if name := strings.TrimSpace(req.OwnerName); name != "" {
		_, _ = a.DB.Exec(`UPDATE users SET name = ? WHERE id = ?`, name, middleware.UserID(r))
	}
	if phone := strings.TrimSpace(req.ContactPhone); phone != "" {
		_, _ = a.DB.Exec(`UPDATE users SET phone = ? WHERE id = ?`, phone, middleware.UserID(r))
	}

	p, msg := buildPropertyFromRequest(req, middleware.UserID(r), "")
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if err := a.insertProperty(p); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create property")
		return
	}
	a.attachMediaAndMap(&p)
	writeJSON(w, http.StatusCreated, p)
}

func (a *API) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := a.fetchProperty(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load property")
		return
	}
	if existing.OwnerID != middleware.UserID(r) {
		writeError(w, http.StatusForbidden, "only the owner can update this property")
		return
	}

	var req propertyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	p, msg := buildPropertyFromRequest(req, existing.OwnerID, id)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	p.CreatedAt = existing.CreatedAt

	if err := a.updateProperty(p); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update property")
		return
	}
	a.attachMediaAndMap(&p)
	writeJSON(w, http.StatusOK, p)
}

func (a *API) DeleteProperty(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := a.fetchProperty(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "property not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load property")
		return
	}
	if existing.OwnerID != middleware.UserID(r) && middleware.Role(r) != "admin" {
		writeError(w, http.StatusForbidden, "only the owner can delete this property")
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

func (a *API) MyProperties(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(propertySelect+` WHERE owner_id = ? ORDER BY created_at DESC`, middleware.UserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list properties")
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
		list = append(list, p)
	}
	rows.Close()

	for i := range list {
		a.attachMediaAndMap(&list[i])
	}
	writeJSON(w, http.StatusOK, list)
}

func buildPropertyFromRequest(req propertyRequest, ownerID, existingID string) (models.Property, string) {
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	switch models.PropertyType(req.Type) {
	case models.PropertyRoom, models.PropertyHome, models.PropertyPG, models.PropertyShop:
	default:
		return models.Property{}, "type must be room, home, pg or shop"
	}

	if strings.TrimSpace(req.Title) == "" {
		return models.Property{}, "title is required"
	}
	if strings.TrimSpace(req.Address) == "" || strings.TrimSpace(req.City) == "" {
		return models.Property{}, "address and city are required"
	}
	if strings.TrimSpace(req.Locality) == "" {
		return models.Property{}, "locality (area) is required"
	}
	if req.Rent <= 0 {
		return models.Property{}, "rent must be greater than 0"
	}
	if msg := validateLatLng(req.Latitude, req.Longitude); msg != "" {
		return models.Property{}, msg
	}

	from, err1 := time.Parse("2006-01-02", req.AvailableFrom)
	until, err2 := time.Parse("2006-01-02", req.AvailableUntil)
	if err1 != nil || err2 != nil {
		return models.Property{}, "available_from and available_until must be YYYY-MM-DD"
	}
	if !until.After(from) && !until.Equal(from) {
		return models.Property{}, "available_until must be on or after available_from"
	}

	tenants, msg := normalizePreferredTenants(req.PreferredTenants, models.PropertyType(req.Type))
	if msg != "" {
		return models.Property{}, msg
	}

	sharing := models.SharingType(strings.ToLower(strings.TrimSpace(req.SharingType)))
	furnishing := models.Furnishing(strings.ToLower(strings.TrimSpace(req.Furnishing)))
	gender := models.GenderPreference(strings.ToLower(strings.TrimSpace(req.GenderPreference)))
	if gender == "" {
		gender = models.GenderAny
	}

	switch models.PropertyType(req.Type) {
	case models.PropertyPG:
		if !validSharing(sharing) {
			return models.Property{}, "pg requires sharing_type: private, 1_sharing, 2_sharing, 3_sharing or 4_sharing"
		}
		if gender != models.GenderMale && gender != models.GenderFemale && gender != models.GenderAny {
			return models.Property{}, "gender_preference must be male, female or any"
		}
		food := strings.ToLower(strings.TrimSpace(req.FoodType))
		if food != "" && food != "veg" && food != "non_veg" && food != "both" {
			return models.Property{}, "food_type must be veg, non_veg or both"
		}
		req.FoodType = food
	case models.PropertyRoom:
		if sharing != "" && !validSharing(sharing) {
			return models.Property{}, "sharing_type must be private, 1_sharing, 2_sharing, 3_sharing or 4_sharing"
		}
		if sharing == "" {
			sharing = models.SharingPrivate
		}
	case models.PropertyHome:
		if req.BHK <= 0 && req.Bedrooms <= 0 {
			return models.Property{}, "home requires bhk or bedrooms"
		}
		if furnishing != "" && !validFurnishing(furnishing) {
			return models.Property{}, "furnishing must be furnished, semi_furnished or unfurnished"
		}
	case models.PropertyShop:
		cat := models.ShopCategory(strings.ToLower(strings.TrimSpace(req.ShopCategory)))
		if !validShopCategory(cat) {
			return models.Property{}, "shop_category must be retail, office, warehouse, showroom, clinic or other"
		}
		req.ShopCategory = string(cat)
		if req.AreaSqFt <= 0 {
			return models.Property{}, "shop requires area_sq_ft"
		}
	}

	if furnishing != "" && !validFurnishing(furnishing) {
		return models.Property{}, "furnishing must be furnished, semi_furnished or unfurnished"
	}

	status := models.PropertyStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	if status != models.StatusAvailable && status != models.StatusUnavailable && status != models.StatusRented {
		status = models.StatusAvailable
	}

	amenities := cleanAmenities(req.Amenities)
	now := time.Now().UTC()
	id := existingID
	if id == "" {
		id = uuid.NewString()
	}

	p := models.Property{
		ID:                 id,
		OwnerID:            ownerID,
		Type:               models.PropertyType(req.Type),
		Title:              strings.TrimSpace(req.Title),
		Description:        strings.TrimSpace(req.Description),
		Status:             status,
		Address:            strings.TrimSpace(req.Address),
		Locality:           strings.TrimSpace(req.Locality),
		City:               strings.TrimSpace(req.City),
		State:              strings.TrimSpace(req.State),
		Pincode:            strings.TrimSpace(req.Pincode),
		Landmark:           strings.TrimSpace(req.Landmark),
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		Rent:               req.Rent,
		Deposit:            req.Deposit,
		AvailableFrom:      from,
		AvailableUntil:     until,
		ParkingTwoWheeler:  req.ParkingTwoWheeler,
		ParkingFourWheeler: req.ParkingFourWheeler,
		PreferredTenants:   tenants,
		Amenities:          amenities,
		ContactPhone:       strings.TrimSpace(req.ContactPhone),
		Bedrooms:           req.Bedrooms,
		Bathrooms:          req.Bathrooms,
		AreaSqFt:           req.AreaSqFt,
		Furnishing:         furnishing,
		SharingType:        sharing,
		GenderPreference:   gender,
		Floor:              req.Floor,
		TotalFloors:        req.TotalFloors,
		BHK:                req.BHK,
		MealsIncluded:      req.MealsIncluded,
		FoodType:           strings.ToLower(strings.TrimSpace(req.FoodType)),
		ShopCategory:       models.ShopCategory(req.ShopCategory),
		FrontageFt:         req.FrontageFt,
		PowerBackup:        req.PowerBackup,
		Washroom:           req.Washroom,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return p, ""
}

func normalizePreferredTenants(raw []string, ptype models.PropertyType) ([]models.PreferredTenant, string) {
	allowed := map[string]bool{
		"family": true, "bachelor": true, "couple": true,
		"student": true, "anyone": true, "company": true,
	}
	out := make([]models.PreferredTenant, 0, len(raw))
	seen := map[string]bool{}
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if !allowed[t] {
			return nil, "preferred_tenants must be from: family, bachelor, couple, student, anyone, company"
		}
		if ptype == models.PropertyShop && t != "company" && t != "anyone" {
			// shops usually prefer company/anyone; other values still allowed
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, models.PreferredTenant(t))
		}
	}
	if len(out) == 0 {
		if ptype == models.PropertyShop {
			out = []models.PreferredTenant{models.TenantAnyone, models.TenantCompany}
		} else {
			out = []models.PreferredTenant{models.TenantAnyone}
		}
	}
	return out, ""
}

func validSharing(s models.SharingType) bool {
	switch s {
	case models.SharingPrivate, models.SharingSingle, models.SharingDouble, models.SharingTriple, models.SharingQuad:
		return true
	}
	return false
}

func validFurnishing(f models.Furnishing) bool {
	switch f {
	case models.FurnishingFurnished, models.FurnishingSemi, models.FurnishingUnfurnished:
		return true
	}
	return false
}

func validShopCategory(c models.ShopCategory) bool {
	switch c {
	case models.ShopRetail, models.ShopOffice, models.ShopWarehouse, models.ShopShowroom, models.ShopClinic, models.ShopOther:
		return true
	}
	return false
}

func cleanAmenities(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, a := range items {
		a = strings.ToLower(strings.TrimSpace(a))
		a = strings.ReplaceAll(a, " ", "_")
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

func (a *API) insertProperty(p models.Property) error {
	_, err := a.DB.Exec(
		`INSERT INTO properties (
			id, owner_id, type, title, description, status,
			address, locality, city, state, pincode, landmark, latitude, longitude,
			rent, deposit, available_from, available_until,
			parking_two_wheeler, parking_four_wheeler,
			preferred_tenants, amenities, contact_phone,
			bedrooms, bathrooms, area_sq_ft, furnishing, sharing_type, gender_preference,
			floor, total_floors, bhk, meals_included, food_type,
			shop_category, frontage_ft, power_backup, washroom,
			created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.OwnerID, p.Type, p.Title, p.Description, p.Status,
		p.Address, p.Locality, p.City, p.State, p.Pincode, p.Landmark, p.Latitude, p.Longitude,
		p.Rent, p.Deposit, p.AvailableFrom, p.AvailableUntil,
		boolToInt(p.ParkingTwoWheeler), boolToInt(p.ParkingFourWheeler),
		encodeJSONList(p.PreferredTenants), encodeJSONList(p.Amenities), p.ContactPhone,
		p.Bedrooms, p.Bathrooms, p.AreaSqFt, p.Furnishing, p.SharingType, p.GenderPreference,
		p.Floor, p.TotalFloors, p.BHK, boolToInt(p.MealsIncluded), p.FoodType,
		p.ShopCategory, p.FrontageFt, boolToInt(p.PowerBackup), boolToInt(p.Washroom),
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (a *API) updateProperty(p models.Property) error {
	_, err := a.DB.Exec(
		`UPDATE properties SET
			type=?, title=?, description=?, status=?,
			address=?, locality=?, city=?, state=?, pincode=?, landmark=?, latitude=?, longitude=?,
			rent=?, deposit=?, available_from=?, available_until=?,
			parking_two_wheeler=?, parking_four_wheeler=?,
			preferred_tenants=?, amenities=?, contact_phone=?,
			bedrooms=?, bathrooms=?, area_sq_ft=?, furnishing=?, sharing_type=?, gender_preference=?,
			floor=?, total_floors=?, bhk=?, meals_included=?, food_type=?,
			shop_category=?, frontage_ft=?, power_backup=?, washroom=?,
			updated_at=?
		WHERE id=?`,
		p.Type, p.Title, p.Description, p.Status,
		p.Address, p.Locality, p.City, p.State, p.Pincode, p.Landmark, p.Latitude, p.Longitude,
		p.Rent, p.Deposit, p.AvailableFrom, p.AvailableUntil,
		boolToInt(p.ParkingTwoWheeler), boolToInt(p.ParkingFourWheeler),
		encodeJSONList(p.PreferredTenants), encodeJSONList(p.Amenities), p.ContactPhone,
		p.Bedrooms, p.Bathrooms, p.AreaSqFt, p.Furnishing, p.SharingType, p.GenderPreference,
		p.Floor, p.TotalFloors, p.BHK, boolToInt(p.MealsIncluded), p.FoodType,
		p.ShopCategory, p.FrontageFt, boolToInt(p.PowerBackup), boolToInt(p.Washroom),
		p.UpdatedAt, p.ID,
	)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProperty(row rowScanner) (models.Property, error) {
	var p models.Property
	var prefJSON, amenJSON string
	var park2, park4, meals, power, wash int
	err := row.Scan(
		&p.ID, &p.OwnerID, &p.Type, &p.Title, &p.Description, &p.Status,
		&p.Address, &p.Locality, &p.City, &p.State, &p.Pincode, &p.Landmark, &p.Latitude, &p.Longitude,
		&p.Rent, &p.Deposit, &p.AvailableFrom, &p.AvailableUntil,
		&park2, &park4, &prefJSON, &amenJSON, &p.ContactPhone,
		&p.Bedrooms, &p.Bathrooms, &p.AreaSqFt, &p.Furnishing, &p.SharingType, &p.GenderPreference,
		&p.Floor, &p.TotalFloors, &p.BHK, &meals, &p.FoodType,
		&p.ShopCategory, &p.FrontageFt, &power, &wash,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.ParkingTwoWheeler = intToBool(park2)
	p.ParkingFourWheeler = intToBool(park4)
	p.MealsIncluded = intToBool(meals)
	p.PowerBackup = intToBool(power)
	p.Washroom = intToBool(wash)
	p.PreferredTenants = decodePreferred(prefJSON)
	p.Amenities = decodeAmenities(amenJSON)
	return p, nil
}

func (a *API) fetchProperty(id string) (models.Property, error) {
	return scanProperty(a.DB.QueryRow(propertySelect+` WHERE id = ?`, id))
}

func getUserID(r *http.Request) string {
	return middleware.UserID(r)
}
