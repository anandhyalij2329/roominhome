package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"room-rental/internal/models"
)

// Meta returns dropdown options for property forms
func (a *API) Meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.MetaOptions{
		PropertyTypes: []string{"room", "home", "pg", "shop"},
		PreferredTenants: []string{
			"family", "bachelor", "couple", "student", "anyone", "company",
		},
		SharingTypes: []string{
			"private", "1_sharing", "2_sharing", "3_sharing", "4_sharing",
		},
		GenderOptions: []string{"male", "female", "any"},
		Furnishing:    []string{"furnished", "semi_furnished", "unfurnished"},
		ShopCategories: []string{
			"retail", "office", "warehouse", "showroom", "clinic", "other",
		},
		FoodTypes: []string{"veg", "non_veg", "both"},
		SuggestedAmenities: []string{
			"wifi", "ac", "geyser", "washing_machine", "fridge", "tv",
			"power_backup", "lift", "security", "cctv", "water_24x7",
			"attached_bathroom", "balcony", "kitchen", "meals", "housekeeping",
			"parking", "garden", "gym", "society",
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(n int) bool {
	return n != 0
}

func encodeJSONList[T any](items []T) string {
	if items == nil {
		items = []T{}
	}
	b, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func decodePreferred(s string) []models.PreferredTenant {
	var out []models.PreferredTenant
	if s == "" {
		return []models.PreferredTenant{}
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return []models.PreferredTenant{}
	}
	return out
}

func decodeAmenities(s string) []string {
	var out []string
	if s == "" {
		return []string{}
	}
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		return []string{}
	}
	return out
}

func removePropertyUploadDir(uploadDir, propertyID string) error {
	return os.RemoveAll(filepath.Join(uploadDir, propertyID))
}
