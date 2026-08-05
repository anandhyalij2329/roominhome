# Room Rental (Go)

Property rental API — **room / home / PG / shop** with **photo/video upload** and **map**.

## Features

- Property types: `room`, `home`, `pg`, `shop`
- Preferred tenants, sharing, parking, amenities, full location
- **Photos & videos** upload (multipart)
- **Map**: lat/lng required, nearby search, OSM + Google Maps links
- Bookings: approve / reject / cancel

## Run

```bash
go mod tidy
go run ./cmd/server
```

## Demo login (after seed)

```bash
go run ./cmd/seed
```

| Role | Email | Password |
|------|-------|----------|
| Owner | `owner@adnivara.test` | `owner123` |
| Seeker | `seeker@adnivara.test` | `seeker123` |

Seed adds **40 Pune properties**: 10 room · 10 home · 10 PG · 10 shop.

## Domain

Production domain: **https://www.adnivara.com**

When live, set:

```bash
PUBLIC_BASE_URL=https://www.adnivara.com
JWT_SECRET=your-long-random-secret
```

Local run still uses `http://localhost:8080`.

| Env | Default |
|-----|---------|
| `PORT` | `8080` |
| `UPLOAD_DIR` | `uploads` |
| `PUBLIC_BASE_URL` | `http://localhost:8080` (prod: `https://www.adnivara.com`) |
| `MAX_PHOTO_MB` | `5` |
| `MAX_VIDEO_MB` | `50` |

## Roles

| Role | Meaning |
|------|---------|
| `owner` | Property + media + location |
| `seeker` | Browse + booking |

## Media (photo / video)

| Method | Path | Who |
|--------|------|-----|
| POST | `/api/properties/{id}/media` | Owner — multipart upload |
| GET | `/api/properties/{id}/media` | Public |
| DELETE | `/api/properties/{id}/media/{mediaId}` | Owner |
| POST | `/api/properties/{id}/media/{mediaId}/cover` | Owner — set cover photo |

**Limits:** photos jpg/png/webp/gif (max 20, 5MB each) · videos mp4/webm/mov (max 5, 50MB each)

```bash
curl -X POST http://localhost:8080/api/properties/<id>/media \
  -H "Authorization: Bearer <token>" \
  -F "files=@room1.jpg" \
  -F "files=@tour.mp4"
```

Files: `http://localhost:8080/uploads/<property-id>/photos/...`

Property response includes `media[]`, `map_url`, `google_map_url`.

## Map

Create/update needs **latitude** + **longitude**.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/map/nearby?lat=18.52&lng=73.85&radius_km=5&type=pg` | Nearby listings |
| GET | `/api/properties/{id}/map` | Map links + embed URL |
| PUT | `/api/properties/{id}/location` | Update pin + address |

```json
PUT /api/properties/{id}/location
{
  "latitude": 18.5204,
  "longitude": 73.8567,
  "address": "12 FC Road",
  "locality": "Shivajinagar",
  "city": "Pune",
  "state": "Maharashtra",
  "pincode": "411005",
  "landmark": "Near college"
}
```

Frontend tip: pick a pin on Leaflet / Google Maps, then save `latitude` / `longitude`.

## Create property (with map)

```json
POST /api/properties
{
  "type": "pg",
  "title": "Boys PG near College",
  "address": "12 FC Road",
  "locality": "Shivajinagar",
  "city": "Pune",
  "state": "Maharashtra",
  "pincode": "411005",
  "latitude": 18.5204,
  "longitude": 73.8567,
  "rent": 8000,
  "deposit": 8000,
  "available_from": "2026-08-01",
  "available_until": "2027-03-31",
  "parking_two_wheeler": true,
  "parking_four_wheeler": false,
  "preferred_tenants": ["bachelor", "student"],
  "amenities": ["wifi", "meals"],
  "sharing_type": "2_sharing",
  "gender_preference": "male",
  "meals_included": true,
  "food_type": "veg"
}
```

## Project layout

```
cmd/server/
internal/
  auth/ config/ database/ handlers/ middleware/ models/
uploads/   # photos & videos (gitignored)
```
