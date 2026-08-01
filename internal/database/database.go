package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

func Connect(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(8000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL CHECK(role IN ('owner', 'seeker', 'admin')),
	phone TEXT DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS properties (
	id TEXT PRIMARY KEY,
	owner_id TEXT NOT NULL REFERENCES users(id),
	type TEXT NOT NULL CHECK(type IN ('room', 'home', 'pg', 'shop')),
	title TEXT NOT NULL,
	description TEXT DEFAULT '',
	status TEXT NOT NULL DEFAULT 'available' CHECK(status IN ('available', 'unavailable', 'rented')),

	address TEXT NOT NULL,
	locality TEXT NOT NULL DEFAULT '',
	city TEXT NOT NULL,
	state TEXT NOT NULL DEFAULT '',
	pincode TEXT NOT NULL DEFAULT '',
	landmark TEXT DEFAULT '',
	latitude REAL DEFAULT 0,
	longitude REAL DEFAULT 0,

	rent REAL NOT NULL,
	deposit REAL NOT NULL DEFAULT 0,
	available_from DATETIME NOT NULL,
	available_until DATETIME NOT NULL,

	parking_two_wheeler INTEGER NOT NULL DEFAULT 0,
	parking_four_wheeler INTEGER NOT NULL DEFAULT 0,

	preferred_tenants TEXT NOT NULL DEFAULT '[]',
	amenities TEXT NOT NULL DEFAULT '[]',
	contact_phone TEXT DEFAULT '',

	bedrooms INTEGER DEFAULT 0,
	bathrooms INTEGER DEFAULT 0,
	area_sq_ft REAL DEFAULT 0,
	furnishing TEXT DEFAULT '',
	sharing_type TEXT DEFAULT '',
	gender_preference TEXT DEFAULT 'any',
	floor INTEGER DEFAULT 0,
	total_floors INTEGER DEFAULT 0,
	bhk INTEGER DEFAULT 0,

	meals_included INTEGER DEFAULT 0,
	food_type TEXT DEFAULT '',

	shop_category TEXT DEFAULT '',
	frontage_ft REAL DEFAULT 0,
	power_backup INTEGER DEFAULT 0,
	washroom INTEGER DEFAULT 0,

	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bookings (
	id TEXT PRIMARY KEY,
	property_id TEXT NOT NULL REFERENCES properties(id),
	seeker_id TEXT NOT NULL REFERENCES users(id),
	start_date DATETIME NOT NULL,
	end_date DATETIME NOT NULL,
	message TEXT DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected', 'cancelled')),
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS media (
	id TEXT PRIMARY KEY,
	property_id TEXT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
	type TEXT NOT NULL CHECK(type IN ('photo', 'video')),
	file_path TEXT NOT NULL,
	file_name TEXT NOT NULL,
	mime_type TEXT NOT NULL,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	is_cover INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_properties_type ON properties(type);
CREATE INDEX IF NOT EXISTS idx_properties_city ON properties(city);
CREATE INDEX IF NOT EXISTS idx_properties_locality ON properties(locality);
CREATE INDEX IF NOT EXISTS idx_properties_owner ON properties(owner_id);
CREATE INDEX IF NOT EXISTS idx_properties_status ON properties(status);
CREATE INDEX IF NOT EXISTS idx_properties_latlng ON properties(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_bookings_property ON bookings(property_id);
CREATE INDEX IF NOT EXISTS idx_bookings_seeker ON bookings(seeker_id);
CREATE INDEX IF NOT EXISTS idx_media_property ON media(property_id);
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureAdminRoleAllowed(db); err != nil {
		return err
	}
	return nil
}

// SQLite CHECK on users.role cannot be altered in place — rebuild if needed.
func ensureAdminRoleAllowed(db *sql.DB) error {
	var sqlText string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='users'`).Scan(&sqlText)
	if err != nil {
		return nil
	}
	if strings.Contains(sqlText, "'admin'") {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmts := []string{
		`CREATE TABLE users_new (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('owner', 'seeker', 'admin')),
			phone TEXT DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO users_new (id, name, email, password_hash, role, phone, created_at)
			SELECT id, name, email, password_hash, role, phone, created_at FROM users`,
		`DROP TABLE users`,
		`ALTER TABLE users_new RENAME TO users`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return fmt.Errorf("admin role migrate: %w", err)
		}
	}
	return tx.Commit()
}
