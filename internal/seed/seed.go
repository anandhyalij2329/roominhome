package seed

import (
	"database/sql"
	"fmt"
	"time"

	"room-rental/internal/auth"
)

// SeedIfEmpty inserts demo listings when the database has no properties.
func SeedIfEmpty(db *sql.DB) (int, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM properties`).Scan(&count); err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, nil
	}
	return Run(db)
}

// Run clears seed data and inserts demo users + 40 properties.
func Run(db *sql.DB) (int, error) {
	ownerHash, err := auth.HashPassword("owner123")
	if err != nil {
		return 0, err
	}
	seekerHash, err := auth.HashPassword("seeker123")
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	ownerID := "seed-owner-1"
	seekerID := "seed-seeker-1"

	if _, err := db.Exec(`DELETE FROM bookings`); err != nil {
		return 0, err
	}
	if _, err := db.Exec(`DELETE FROM media`); err != nil {
		return 0, err
	}
	if _, err := db.Exec(`DELETE FROM properties`); err != nil {
		return 0, err
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id LIKE 'seed-%' OR email IN (?, ?)`,
		"owner@roominhome.test", "seeker@roominhome.test"); err != nil {
		return 0, err
	}

	if err := exec(db, `INSERT INTO users (id, name, email, password_hash, role, phone, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ownerID, "Suresh Patil", "owner@roominhome.test", ownerHash, "owner", "9876543210", now); err != nil {
		return 0, err
	}
	if err := exec(db, `INSERT INTO users (id, name, email, password_hash, role, phone, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		seekerID, "Ananya Deshmukh", "seeker@roominhome.test", seekerHash, "seeker", "9123456780", now); err != nil {
		return 0, err
	}

	areas := []struct {
		Locality, Address, Pincode, Landmark string
		Lat, Lng                             float64
	}{
		{"Baner", "12 Baner Road", "411045", "Near Balewadi High Street", 18.5590, 73.7868},
		{"Shivajinagar", "22 FC Road", "411005", "Near Fergusson College", 18.5204, 73.8567},
		{"Kothrud", "45 Paud Road", "411038", "Near Karve Statue", 18.5074, 73.8077},
		{"Wakad", "8 Wakad Bridge", "411057", "Near Dange Chowk", 18.5970, 73.7630},
		{"Hadapsar", "19 Magarpatta Road", "411028", "Near Magarpatta", 18.5089, 73.9260},
		{"Viman Nagar", "31 Nagar Road", "411014", "Near Phoenix Mall", 18.5679, 73.9143},
		{"Kalyani Nagar", "7 North Main Road", "411006", "Lane 5", 18.5480, 73.9010},
		{"Deccan", "14 JM Road", "411004", "Near Goodluck Chowk", 18.5160, 73.8410},
		{"Hinjewadi", "Plot 5 Phase 1", "411057", "IT Park gate", 18.5912, 73.7389},
		{"Aundh", "28 ITI Road", "411007", "Near Bremen Chowk", 18.5580, 73.8070},
	}

	insert := `INSERT INTO properties (
		id, owner_id, type, title, description, status,
		address, locality, city, state, pincode, landmark, latitude, longitude,
		rent, deposit, available_from, available_until,
		parking_two_wheeler, parking_four_wheeler,
		preferred_tenants, amenities, contact_phone,
		bedrooms, bathrooms, area_sq_ft, furnishing, sharing_type, gender_preference,
		floor, total_floors, bhk, meals_included, food_type,
		shop_category, frontage_ft, power_backup, washroom,
		created_at, updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

	from, _ := time.Parse("2006-01-02", "2026-08-01")
	until, _ := time.Parse("2006-01-02", "2027-07-31")
	n := 0

	roomShares := []string{"private", "1_sharing", "2_sharing", "private", "2_sharing", "1_sharing", "3_sharing", "private", "2_sharing", "private"}
	roomRents := []float64{11000, 7000, 8000, 13000, 7500, 9000, 6500, 15000, 8500, 12000}
	for i := 0; i < 10; i++ {
		a := areas[i]
		title := fmt.Sprintf("%s room in %s", shareLabel(roomShares[i]), a.Locality)
		if err := exec(db, insert,
			fmt.Sprintf("seed-room-%02d", i+1), ownerID, "room", title, "Comfortable room with WiFi and parking options.", "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat+offset(i), a.Lng+offset(i)*0.5,
			roomRents[i], roomRents[i]*2, from, until,
			1, boolInt(i%3 == 0),
			`["bachelor","student"]`, `["wifi","geyser","cctv"]`, "9876543210",
			1, 1, 120+float64(i*10), "semi_furnished", roomShares[i], "any",
			i%4, 5, 0, 0, "",
			"", 0.0, 0, 0,
			now, now,
		); err != nil {
			return n, err
		}
		n++
	}

	bhks := []int{1, 2, 2, 3, 1, 2, 3, 2, 1, 3}
	homeRents := []float64{16000, 25000, 28000, 40000, 18000, 26000, 45000, 30000, 17000, 42000}
	for i := 0; i < 10; i++ {
		a := areas[i]
		bhk := bhks[i]
		title := fmt.Sprintf("%dBHK home in %s", bhk, a.Locality)
		if err := exec(db, insert,
			fmt.Sprintf("seed-home-%02d", i+1), ownerID, "home", title, "Family-friendly home with society amenities.", "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat+offset(i+1), a.Lng-offset(i)*0.4,
			homeRents[i], homeRents[i]*2, from, until,
			1, 1,
			`["family"]`, `["lift","security","society","wifi"]`, "9876543210",
			bhk, bhk, 500+float64(bhk*200), pickFurnish(i), "", "any",
			(i%6)+1, 8, bhk, 0, "",
			"", 0.0, boolInt(i%2 == 0), 0,
			now, now,
		); err != nil {
			return n, err
		}
		n++
	}

	pgShares := []string{"1_sharing", "2_sharing", "3_sharing", "2_sharing", "1_sharing", "4_sharing", "2_sharing", "3_sharing", "1_sharing", "2_sharing"}
	pgGender := []string{"male", "male", "any", "female", "female", "male", "any", "female", "male", "any"}
	pgRents := []float64{12000, 9000, 6500, 9500, 14000, 5500, 8500, 7000, 13000, 8000}
	for i := 0; i < 10; i++ {
		a := areas[i]
		title := fmt.Sprintf("PG %s · %s (%s)", shareLabel(pgShares[i]), a.Locality, pgGender[i])
		if err := exec(db, insert,
			fmt.Sprintf("seed-pg-%02d", i+1), ownerID, "pg", title, "PG / co-living with meals option and WiFi.", "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat-offset(i)*0.3, a.Lng+offset(i)*0.3,
			pgRents[i], pgRents[i], from, until,
			1, 0,
			`["bachelor","student"]`, `["wifi","meals","cctv","housekeeping"]`, "9876543210",
			1, 1, 100+float64(i*8), "furnished", pgShares[i], pgGender[i],
			i%3, 4, 0, 1, pickFood(i),
			"", 0.0, 0, 0,
			now, now,
		); err != nil {
			return n, err
		}
		n++
	}

	cats := []string{"retail", "office", "showroom", "retail", "clinic", "warehouse", "office", "retail", "showroom", "other"}
	shopRents := []float64{45000, 70000, 90000, 50000, 35000, 60000, 80000, 55000, 95000, 40000}
	for i := 0; i < 10; i++ {
		a := areas[i]
		title := fmt.Sprintf("%s space in %s", cats[i], a.Locality)
		if err := exec(db, insert,
			fmt.Sprintf("seed-shop-%02d", i+1), ownerID, "shop", title, "Commercial property with frontage and power backup.", "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat+offset(i)*0.2, a.Lng-offset(i)*0.2,
			shopRents[i], shopRents[i]*3, from, until,
			1, boolInt(i%2 == 0),
			`["company","anyone"]`, `["power_backup","cctv","parking"]`, "9876543210",
			0, 1, 300+float64(i*50), "", "", "any",
			i%3, 5, 0, 0, "",
			cats[i], 15+float64(i), 1, 1,
			now, now,
		); err != nil {
			return n, err
		}
		n++
	}

	return n, nil
}

func exec(db *sql.DB, query string, args ...any) error {
	_, err := db.Exec(query, args...)
	return err
}

func offset(i int) float64 { return float64(i) * 0.004 }

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func shareLabel(s string) string {
	switch s {
	case "private":
		return "Private"
	case "1_sharing":
		return "1 sharing"
	case "2_sharing":
		return "2 sharing"
	case "3_sharing":
		return "3 sharing"
	case "4_sharing":
		return "4 sharing"
	default:
		return s
	}
}

func pickFurnish(i int) string {
	return []string{"furnished", "semi_furnished", "unfurnished"}[i%3]
}

func pickFood(i int) string {
	return []string{"veg", "both", "non_veg", "veg", "both"}[i%5]
}
