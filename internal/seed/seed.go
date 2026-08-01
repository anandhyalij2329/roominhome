package seed

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"room-rental/internal/auth"
)

// SeedIfEmpty inserts demo listings when the database has no properties.
// If listings exist but seed media is missing, it attaches photos/videos.
func SeedIfEmpty(db *sql.DB) (int, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM properties`).Scan(&count); err != nil {
		return 0, err
	}
	if count > 0 {
		if err := ClearSeedContacts(db); err != nil {
			return 0, err
		}
		if err := EnsureSeedMedia(db); err != nil {
			return 0, err
		}
		if err := EnsureAdmin(db); err != nil {
			return 0, err
		}
		return 0, nil
	}
	n, err := Run(db)
	if err != nil {
		return n, err
	}
	if err := EnsureAdmin(db); err != nil {
		return n, err
	}
	return n, nil
}

// EnsureAdmin creates the default admin account if missing.
// Login: admin@roominhome.test / admin123
func EnsureAdmin(db *sql.DB) error {
	const email = "admin@roominhome.test"
	var id string
	err := db.QueryRow(`SELECT id FROM users WHERE email = ?`, email).Scan(&id)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	hash, err := auth.HashPassword("admin123")
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO users (id, name, email, password_hash, role, phone, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"seed-admin-1", "Site Admin", email, hash, "admin", "", time.Now().UTC(),
	)
	return err
}

// ClearSeedContacts removes demo contact numbers from seed users/properties.
func ClearSeedContacts(db *sql.DB) error {
	if _, err := db.Exec(`UPDATE properties SET contact_phone = '' WHERE id LIKE 'seed-%'`); err != nil {
		return err
	}
	_, err := db.Exec(`UPDATE users SET phone = '' WHERE id LIKE 'seed-%' OR email IN (?, ?)`,
		"owner@roominhome.test", "seeker@roominhome.test")
	return err
}

// Run clears seed data and inserts demo users + 40 properties with media.
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
		ownerID, "Suresh Patil", "owner@roominhome.test", ownerHash, "owner", "", now); err != nil {
		return 0, err
	}
	if err := exec(db, `INSERT INTO users (id, name, email, password_hash, role, phone, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		seekerID, "Ananya Deshmukh", "seeker@roominhome.test", seekerHash, "seeker", "", now); err != nil {
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
		{"Hadapsar", "19 Magarpatta Road", "411028", "Near Magarpatta City", 18.5089, 73.9260},
		{"Viman Nagar", "31 Nagar Road", "411014", "Near Phoenix Mall", 18.5679, 73.9143},
		{"Kalyani Nagar", "7 North Main Road", "411006", "Lane 5 cafe street", 18.5480, 73.9010},
		{"Deccan", "14 JM Road", "411004", "Near Goodluck Chowk", 18.5160, 73.8410},
		{"Hinjewadi", "Plot 5 Phase 1", "411057", "IT Park main gate", 18.5912, 73.7389},
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
	var propertyIDs []seedProp

	roomShares := []string{"private", "1_sharing", "2_sharing", "private", "2_sharing", "1_sharing", "3_sharing", "private", "2_sharing", "private"}
	roomRents := []float64{11000, 7000, 8000, 13000, 7500, 9000, 6500, 15000, 8500, 12000}
	roomTitles := []string{
		"Bright private room with balcony",
		"Budget 1-sharing near metro bus stop",
		"Spacious 2-sharing for working professionals",
		"Furnished private room with attached bath",
		"Compact 2-sharing close to IT park",
		"Quiet 1-sharing with study table",
		"Affordable 3-sharing for students",
		"Premium private room with AC",
		"Sunny 2-sharing with wardrobe space",
		"Independent private room, kitchen access",
	}
	for i := 0; i < 10; i++ {
		a := areas[i]
		id := fmt.Sprintf("seed-room-%02d", i+1)
		desc := fmt.Sprintf("%s in %s, Pune. Ideal for students and working professionals. WiFi, geyser and CCTV available. Nearby: %s.", roomTitles[i], a.Locality, a.Landmark)
		if err := exec(db, insert,
			id, ownerID, "room", roomTitles[i]+" · "+a.Locality, desc, "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat+offset(i), a.Lng+offset(i)*0.5,
			roomRents[i], roomRents[i]*2, from, until,
			1, boolInt(i%3 == 0),
			`["bachelor","student"]`, `["wifi","geyser","cctv"]`, "",
			1, 1, 120+float64(i*10), "semi_furnished", roomShares[i], "any",
			i%4, 5, 0, 0, "",
			"", 0.0, 0, 0,
			now, now,
		); err != nil {
			return n, err
		}
		propertyIDs = append(propertyIDs, seedProp{id, "room", i})
		n++
	}

	bhks := []int{1, 2, 2, 3, 1, 2, 3, 2, 1, 3}
	homeRents := []float64{16000, 25000, 28000, 40000, 18000, 26000, 45000, 30000, 17000, 42000}
	homeTitles := []string{
		"Cozy 1BHK for small family",
		"2BHK society flat with lift",
		"Well-lit 2BHK near schools",
		"Spacious 3BHK with parking",
		"Ready-to-move 1BHK apartment",
		"Modern 2BHK with modular kitchen",
		"Premium 3BHK in gated society",
		"Family 2BHK close to market",
		"Compact 1BHK with balcony",
		"Vastu-friendly 3BHK home",
	}
	for i := 0; i < 10; i++ {
		a := areas[i]
		bhk := bhks[i]
		id := fmt.Sprintf("seed-home-%02d", i+1)
		desc := fmt.Sprintf("%s in %s. %dBHK with society amenities, security and WiFi. Landmark: %s.", homeTitles[i], a.Locality, bhk, a.Landmark)
		if err := exec(db, insert,
			id, ownerID, "home", homeTitles[i]+" · "+a.Locality, desc, "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat+offset(i+1), a.Lng-offset(i)*0.4,
			homeRents[i], homeRents[i]*2, from, until,
			1, 1,
			`["family"]`, `["lift","security","society","wifi"]`, "",
			bhk, bhk, 500+float64(bhk*200), pickFurnish(i), "", "any",
			(i%6)+1, 8, bhk, 0, "",
			"", 0.0, boolInt(i%2 == 0), 0,
			now, now,
		); err != nil {
			return n, err
		}
		propertyIDs = append(propertyIDs, seedProp{id, "home", i})
		n++
	}

	pgShares := []string{"1_sharing", "2_sharing", "3_sharing", "2_sharing", "1_sharing", "4_sharing", "2_sharing", "3_sharing", "1_sharing", "2_sharing"}
	pgGender := []string{"male", "male", "any", "female", "female", "male", "any", "female", "male", "any"}
	pgRents := []float64{12000, 9000, 6500, 9500, 14000, 5500, 8500, 7000, 13000, 8000}
	pgTitles := []string{
		"Boys PG — single occupancy",
		"Working men PG — 2 sharing",
		"Budget co-living — 3 sharing",
		"Girls PG near college — 2 sharing",
		"Ladies PG — private room feel",
		"Affordable hostel-style 4 sharing",
		"Co-ed PG with meals — 2 sharing",
		"Girls PG with housekeeping",
		"Executive boys PG — 1 sharing",
		"Student PG — 2 sharing + WiFi",
	}
	for i := 0; i < 10; i++ {
		a := areas[i]
		id := fmt.Sprintf("seed-pg-%02d", i+1)
		desc := fmt.Sprintf("%s in %s (%s). Meals, WiFi, CCTV and housekeeping. Near %s.", pgTitles[i], a.Locality, pgGender[i], a.Landmark)
		if err := exec(db, insert,
			id, ownerID, "pg", pgTitles[i]+" · "+a.Locality, desc, "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat-offset(i)*0.3, a.Lng+offset(i)*0.3,
			pgRents[i], pgRents[i], from, until,
			1, 0,
			`["bachelor","student"]`, `["wifi","meals","cctv","housekeeping"]`, "",
			1, 1, 100+float64(i*8), "furnished", pgShares[i], pgGender[i],
			i%3, 4, 0, 1, pickFood(i),
			"", 0.0, 0, 0,
			now, now,
		); err != nil {
			return n, err
		}
		propertyIDs = append(propertyIDs, seedProp{id, "pg", i})
		n++
	}

	cats := []string{"retail", "office", "showroom", "retail", "clinic", "warehouse", "office", "retail", "showroom", "other"}
	shopRents := []float64{45000, 70000, 90000, 50000, 35000, 60000, 80000, 55000, 95000, 40000}
	shopTitles := []string{
		"Ground-floor retail shop",
		"Ready office cabin space",
		"Corner showroom with frontage",
		"Main-road retail outlet",
		"Clinic / consultation space",
		"Small warehouse / storage unit",
		"Furnished office for startups",
		"High-street shop for brand",
		"Premium showroom near mall",
		"Flexible commercial unit",
	}
	for i := 0; i < 10; i++ {
		a := areas[i]
		id := fmt.Sprintf("seed-shop-%02d", i+1)
		desc := fmt.Sprintf("%s (%s) in %s. Power backup, CCTV and parking. Landmark: %s.", shopTitles[i], cats[i], a.Locality, a.Landmark)
		if err := exec(db, insert,
			id, ownerID, "shop", shopTitles[i]+" · "+a.Locality, desc, "available",
			a.Address, a.Locality, "Pune", "Maharashtra", a.Pincode, a.Landmark, a.Lat+offset(i)*0.2, a.Lng-offset(i)*0.2,
			shopRents[i], shopRents[i]*3, from, until,
			1, boolInt(i%2 == 0),
			`["company","anyone"]`, `["power_backup","cctv","parking"]`, "",
			0, 1, 300+float64(i*50), "", "", "any",
			i%3, 5, 0, 0, "",
			cats[i], 15+float64(i), 1, 1,
			now, now,
		); err != nil {
			return n, err
		}
		propertyIDs = append(propertyIDs, seedProp{id, "shop", i})
		n++
	}

	for _, p := range propertyIDs {
		if err := attachMedia(db, p, now); err != nil {
			return n, err
		}
	}

	return n, nil
}

type seedProp struct {
	ID   string
	Type string
	Idx  int
}

// EnsureSeedMedia adds photos/videos to seed properties that have none.
func EnsureSeedMedia(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, type FROM properties WHERE id LIKE 'seed-%' ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now().UTC()
	iByType := map[string]int{}
	for rows.Next() {
		var id, ptype string
		if err := rows.Scan(&id, &ptype); err != nil {
			return err
		}
		var mc int
		if err := db.QueryRow(`SELECT COUNT(*) FROM media WHERE property_id = ?`, id).Scan(&mc); err != nil {
			return err
		}
		if mc > 0 {
			continue
		}
		idx := iByType[ptype]
		iByType[ptype] = idx + 1
		if err := attachMedia(db, seedProp{ID: id, Type: ptype, Idx: idx}, now); err != nil {
			return err
		}
	}
	return rows.Err()
}

func attachMedia(db *sql.DB, p seedProp, now time.Time) error {
	photos := photoURLs(p.Type, p.Idx)
	video := videoURL(p.Idx)

	for i, url := range photos {
		isCover := i == 0
		if err := exec(db, `INSERT INTO media (id, property_id, type, file_path, file_name, mime_type, size_bytes, is_cover, sort_order, created_at)
			VALUES (?, ?, 'photo', ?, ?, 'image/jpeg', 0, ?, ?, ?)`,
			uuid.NewString(), p.ID, url, fmt.Sprintf("%s-%d.jpg", p.Type, i+1), boolInt(isCover), i, now,
		); err != nil {
			return err
		}
	}
	if video != "" {
		if err := exec(db, `INSERT INTO media (id, property_id, type, file_path, file_name, mime_type, size_bytes, is_cover, sort_order, created_at)
			VALUES (?, ?, 'video', ?, ?, 'video/mp4', 0, 0, ?, ?)`,
			uuid.NewString(), p.ID, video, fmt.Sprintf("%s-tour.mp4", p.Type), len(photos), now,
		); err != nil {
			return err
		}
	}
	return nil
}

// Real-looking Unsplash property photos (stable image IDs).
func photoURLs(ptype string, idx int) []string {
	roomSets := [][]string{
		{
			"https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1595526114035-0d45ed16cfbf?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1631049307264-da0ec9d70304?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1505693416388-ac5ce068fe85?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1615874959474-d609969a20ed?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1540518614846-7eded433c457?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1560448204-e02f11c3d0e2?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1555854877-bab0e564b8d5?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1586023492125-27b2c045efd7?auto=format&fit=crop&w=1200&q=80",
		},
	}
	homeSets := [][]string{
		{
			"https://images.unsplash.com/photo-1600596542815-ffad4c1539a9?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1564013799919-ab600027ffc6?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1600566753190-17f0baa2a6c3?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1600210492486-724fe5c67fb0?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1600047509807-ba8f99d2cdde?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1600573472592-401b489a3cdc?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1493809842364-78817add7ffb?auto=format&fit=crop&w=1200&q=80",
		},
	}
	pgSets := [][]string{
		{
			"https://images.unsplash.com/photo-1555854877-bab0e564b8d5?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1522771739844-6a9f6d5f14af?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1631889993959-41b4e9c6e3c5?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1595526114035-0d45ed16cfbf?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1586023492125-27b2c045efd7?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1616594039964-ae9021a400a0?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1560185127-6ed189bf02f4?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1505693416388-ac5ce068fe85?auto=format&fit=crop&w=1200&q=80",
		},
	}
	shopSets := [][]string{
		{
			"https://images.unsplash.com/photo-1441986300917-64674bd600d8?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1604719312566-8912e9227c6a?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1497366811353-6870744d04b2?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1524758631624-e2822e304c36?auto=format&fit=crop&w=1200&q=80",
		},
		{
			"https://images.unsplash.com/photo-1560472355-536de3962603?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1556745757-8d76bdb6984b?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1600880292203-757bb62b4baf?auto=format&fit=crop&w=1200&q=80",
		},
	}

	var sets [][]string
	switch ptype {
	case "home":
		sets = homeSets
	case "pg":
		sets = pgSets
	case "shop":
		sets = shopSets
	default:
		sets = roomSets
	}
	return sets[idx%len(sets)]
}

func videoURL(idx int) string {
	videos := []string{
		"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerEscapes.mp4",
		"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4",
		"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerJoyrides.mp4",
		"https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerMeltdowns.mp4",
	}
	return videos[idx%len(videos)]
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

func pickFurnish(i int) string {
	return []string{"furnished", "semi_furnished", "unfurnished"}[i%3]
}

func pickFood(i int) string {
	return []string{"veg", "both", "non_veg", "veg", "both"}[i%5]
}
