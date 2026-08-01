package models

import "time"

type Role string

const (
	RoleOwner  Role = "owner"  // lists properties
	RoleSeeker Role = "seeker" // searches and books rentals
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	Phone        string    `json:"phone,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// PropertyType — room / home / pg / shop
type PropertyType string

const (
	PropertyRoom PropertyType = "room"
	PropertyHome PropertyType = "home"
	PropertyPG   PropertyType = "pg"
	PropertyShop PropertyType = "shop"
)

type PropertyStatus string

const (
	StatusAvailable   PropertyStatus = "available"
	StatusUnavailable PropertyStatus = "unavailable"
	StatusRented      PropertyStatus = "rented"
)

// PreferredTenant — who the owner wants to rent to
type PreferredTenant string

const (
	TenantFamily   PreferredTenant = "family"
	TenantBachelor PreferredTenant = "bachelor"
	TenantCouple   PreferredTenant = "couple"
	TenantStudent  PreferredTenant = "student"
	TenantAnyone   PreferredTenant = "anyone"
	TenantCompany  PreferredTenant = "company" // useful for shops
)

// SharingType — PG / room sharing options
type SharingType string

const (
	SharingSingle  SharingType = "1_sharing"
	SharingDouble  SharingType = "2_sharing"
	SharingTriple  SharingType = "3_sharing"
	SharingQuad    SharingType = "4_sharing"
	SharingPrivate SharingType = "private"
)

type GenderPreference string

const (
	GenderMale   GenderPreference = "male"
	GenderFemale GenderPreference = "female"
	GenderAny    GenderPreference = "any"
)

type Furnishing string

const (
	FurnishingFurnished   Furnishing = "furnished"
	FurnishingSemi        Furnishing = "semi_furnished"
	FurnishingUnfurnished Furnishing = "unfurnished"
)

type ShopCategory string

const (
	ShopRetail    ShopCategory = "retail"
	ShopOffice    ShopCategory = "office"
	ShopWarehouse ShopCategory = "warehouse"
	ShopShowroom  ShopCategory = "showroom"
	ShopClinic    ShopCategory = "clinic"
	ShopOther     ShopCategory = "other"
)

// Property — shared model for all listing types
type Property struct {
	ID          string         `json:"id"`
	OwnerID     string         `json:"owner_id"`
	Type        PropertyType   `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      PropertyStatus `json:"status"`

	// Location
	Address   string  `json:"address"`
	Locality  string  `json:"locality"` // area / street / society
	City      string  `json:"city"`
	State     string  `json:"state"`
	Pincode   string  `json:"pincode"`
	Landmark  string  `json:"landmark,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`

	// Rent & availability
	Rent           float64   `json:"rent"`
	Deposit        float64   `json:"deposit"`
	AvailableFrom  time.Time `json:"available_from"`
	AvailableUntil time.Time `json:"available_until"` // available until this date

	// Parking
	ParkingTwoWheeler  bool `json:"parking_two_wheeler"`
	ParkingFourWheeler bool `json:"parking_four_wheeler"`

	// Who can rent (family, bachelor, ...)
	PreferredTenants []PreferredTenant `json:"preferred_tenants"`

	// Owner-defined amenities / services
	Amenities []string `json:"amenities"`

	ContactPhone string `json:"contact_phone,omitempty"`
	OwnerName    string `json:"owner_name,omitempty"`
	OwnerPhone   string `json:"owner_phone,omitempty"`

	// ---- Room / Home / PG common residential fields ----
	Bedrooms         int              `json:"bedrooms,omitempty"`
	Bathrooms        int              `json:"bathrooms,omitempty"`
	AreaSqFt         float64          `json:"area_sq_ft,omitempty"`
	Furnishing       Furnishing       `json:"furnishing,omitempty"`
	SharingType      SharingType      `json:"sharing_type,omitempty"` // 1_sharing, 2_sharing, private
	GenderPreference GenderPreference `json:"gender_preference,omitempty"`
	Floor            int              `json:"floor,omitempty"`
	TotalFloors      int              `json:"total_floors,omitempty"`
	BHK              int              `json:"bhk,omitempty"` // for homes

	// ---- PG specific ----
	MealsIncluded bool   `json:"meals_included,omitempty"`
	FoodType      string `json:"food_type,omitempty"` // veg | non_veg | both

	// ---- Shop specific ----
	ShopCategory ShopCategory `json:"shop_category,omitempty"`
	FrontageFt   float64      `json:"frontage_ft,omitempty"`
	PowerBackup  bool         `json:"power_backup,omitempty"`
	Washroom     bool         `json:"washroom,omitempty"`

	// Media + map helpers (filled when listing/detail)
	Media         []Media `json:"media,omitempty"`
	MapURL        string  `json:"map_url,omitempty"`        // OpenStreetMap view
	GoogleMapURL  string  `json:"google_map_url,omitempty"` // Google Maps pin
	DirectionsURL string  `json:"directions_url,omitempty"` // Google Maps directions
	DistanceKm    float64 `json:"distance_km,omitempty"`    // nearby search

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MediaType string

const (
	MediaPhoto MediaType = "photo"
	MediaVideo MediaType = "video"
)

type Media struct {
	ID         string    `json:"id"`
	PropertyID string    `json:"property_id"`
	Type       MediaType `json:"type"` // photo | video
	URL        string    `json:"url"`
	FileName   string    `json:"file_name"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	IsCover    bool      `json:"is_cover"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
}

type BookingStatus string

const (
	BookingPending   BookingStatus = "pending"
	BookingApproved  BookingStatus = "approved"
	BookingRejected  BookingStatus = "rejected"
	BookingCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID            string        `json:"id"`
	PropertyID    string        `json:"property_id"`
	SeekerID      string        `json:"seeker_id"`
	StartDate     time.Time     `json:"start_date"`
	EndDate       time.Time     `json:"end_date"`
	Message       string        `json:"message,omitempty"`
	Status        BookingStatus `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	PropertyTitle string        `json:"property_title,omitempty"`
	SeekerName    string        `json:"seeker_name,omitempty"`
}

// MetaOptions — available options for frontend dropdowns
type MetaOptions struct {
	PropertyTypes      []string `json:"property_types"`
	PreferredTenants   []string `json:"preferred_tenants"`
	SharingTypes       []string `json:"sharing_types"`
	GenderOptions      []string `json:"gender_options"`
	Furnishing         []string `json:"furnishing"`
	ShopCategories     []string `json:"shop_categories"`
	FoodTypes          []string `json:"food_types"`
	SuggestedAmenities []string `json:"suggested_amenities"`
}
