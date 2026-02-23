package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	overpassURL = "https://overpass-api.de/api/interpreter" //nolint:gosec // Not a credential, this is a public API URL.

	// Deduplication radius in meters.
	deduplicationRadiusM = 50.0

	// HTTP timeout for Overpass API requests.
	httpTimeout = 120 * time.Second
)

// overpassResponse is the top-level JSON response from the Overpass API.
type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

// overpassElement represents a single OSM element (node, way, or relation).
type overpassElement struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Tags   map[string]string `json:"tags"`
	Center *overpassCenter   `json:"center,omitempty"`
}

// overpassCenter holds the center coordinates for ways and relations.
type overpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// importStats tracks the progress of the import process.
type importStats struct {
	imported int
	skipped  int
	errors   int
	noName   int
}

// mapOSMToPOIType converts OSM tags to internal POI types.
func mapOSMToPOIType(tags map[string]string) string {
	if v, ok := tags["tourism"]; ok {
		switch v {
		case "museum":
			return "museum"
		case "attraction", "viewpoint":
			return "monument"
		}
	}
	if v, ok := tags["amenity"]; ok && v == "place_of_worship" {
		return "church"
	}
	if _, ok := tags["bridge"]; ok {
		return "bridge"
	}
	if v, ok := tags["man_made"]; ok && v == "bridge" {
		return "bridge"
	}
	if v, ok := tags["leisure"]; ok && v == "park" {
		return "park"
	}
	if v, ok := tags["place"]; ok && v == "square" {
		return "square"
	}
	if v, ok := tags["historic"]; ok {
		switch v {
		case "monument", "memorial":
			return "monument"
		case "castle", "fort", "ruins", "archaeological_site": //nolint:misspell // OSM tag value.
			return "building"
		case "church":
			return "church"
		}
		return "monument"
	}
	return "building"
}

// defaultInterestScore returns a default interest score based on POI type.
func defaultInterestScore(poiType string) int16 {
	switch poiType {
	case "museum":
		return 70
	case "monument":
		return 65
	case "church":
		return 60
	case "bridge":
		return 55
	case "park":
		return 50
	case "square":
		return 50
	default:
		return 40
	}
}

// getName extracts the English name from OSM tags.
// Falls back to the default name tag if name:en is not available.
func getName(tags map[string]string) string {
	if name, ok := tags["name:en"]; ok && name != "" {
		return name
	}
	return tags["name"]
}

// getNameRu extracts the Russian name from OSM tags.
func getNameRu(tags map[string]string) *string {
	if name, ok := tags["name:ru"]; ok && name != "" {
		return &name
	}
	return nil
}

// getAddress extracts the address from OSM tags.
func getAddress(tags map[string]string) *string {
	parts := []string{}
	if street, ok := tags["addr:street"]; ok {
		parts = append(parts, street)
	}
	if number, ok := tags["addr:housenumber"]; ok {
		parts = append(parts, number)
	}
	if len(parts) > 0 {
		addr := strings.Join(parts, " ")
		return &addr
	}
	return nil
}

// buildOverpassQuery constructs the Overpass QL query for Tbilisi POIs.
// Uses a bounding box (south,west,north,east) for reliable results.
func buildOverpassQuery() string {
	// Tbilisi bounding box: south=41.63, west=44.70, north=41.82, east=44.90
	const bbox = "41.63,44.70,41.82,44.90"

	return `[out:json][timeout:90];
(
  node["tourism"="museum"]["name"](` + bbox + `);
  way["tourism"="museum"]["name"](` + bbox + `);
  node["tourism"="attraction"]["name"](` + bbox + `);
  way["tourism"="attraction"]["name"](` + bbox + `);
  node["tourism"="viewpoint"]["name"](` + bbox + `);
  way["tourism"="viewpoint"]["name"](` + bbox + `);
  node["amenity"="place_of_worship"]["name"](` + bbox + `);
  way["amenity"="place_of_worship"]["name"](` + bbox + `);
  node["historic"]["name"](` + bbox + `);
  way["historic"]["name"](` + bbox + `);
  node["leisure"="park"]["name"](` + bbox + `);
  way["leisure"="park"]["name"](` + bbox + `);
  node["man_made"="bridge"]["name"](` + bbox + `);
  way["man_made"="bridge"]["name"](` + bbox + `);
  way["bridge"="yes"]["name"]["highway"](` + bbox + `);
  node["place"="square"]["name"](` + bbox + `);
  way["place"="square"]["name"](` + bbox + `);
);
out center;`
}

// fetchOverpassData sends a query to the Overpass API and returns parsed elements.
func fetchOverpassData(query string) ([]overpassElement, error) {
	client := &http.Client{Timeout: httpTimeout}

	body := "data=" + query
	req, err := http.NewRequest(http.MethodPost, overpassURL, bytes.NewBufferString(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log.Println("Querying Overpass API for Tbilisi POIs...")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("overpass request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("overpass returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result overpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode overpass response: %w", err)
	}

	log.Printf("Overpass returned %d elements", len(result.Elements))
	return result.Elements, nil
}

// ensureTbilisiCity ensures the Tbilisi city record exists and returns its ID.
func ensureTbilisiCity(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var cityID int
	err := pool.QueryRow(ctx, `SELECT id FROM cities WHERE name = $1`, "Tbilisi").Scan(&cityID)
	if err == nil {
		log.Printf("Tbilisi city found with id=%d", cityID)
		return cityID, nil
	}

	nameRu := "Тбилиси"
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		"Tbilisi", &nameRu, "Georgia", 41.7151, 44.8271, 15.0, true, 0.0,
	).Scan(&cityID)
	if err != nil {
		return 0, fmt.Errorf("insert Tbilisi city: %w", err)
	}

	log.Printf("Created Tbilisi city with id=%d", cityID)
	return cityID, nil
}

// poiExistsNearby checks if a POI already exists within deduplicationRadiusM of the given coordinates.
func poiExistsNearby(ctx context.Context, pool *pgxpool.Pool, cityID int, lat, lng float64) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM poi
			WHERE city_id = $1
			  AND ST_DWithin(
				location,
				ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
				$4
			  )
		)`, cityID, lng, lat, deduplicationRadiusM).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check poi exists nearby: %w", err)
	}
	return exists, nil
}

// insertPOI inserts a single POI into the database.
func insertPOI(ctx context.Context, pool *pgxpool.Pool, cityID int, name string, nameRu *string, lat, lng float64, poiType string, address *string, interestScore int16, osmTags map[string]string) error {
	tagsJSON, err := json.Marshal(osmTags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO poi (city_id, name, name_ru, location, type, tags, address, interest_score, status)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography, $6, $7, $8, $9, 'active')`,
		cityID, name, nameRu, lng, lat, poiType, tagsJSON, address, interestScore,
	)
	if err != nil {
		return fmt.Errorf("insert poi: %w", err)
	}
	return nil
}

func run() error {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	if pingErr := pool.Ping(ctx); pingErr != nil {
		return fmt.Errorf("ping database: %w", pingErr)
	}
	log.Println("Database connected")

	// Ensure Tbilisi city exists.
	cityID, err := ensureTbilisiCity(ctx, pool)
	if err != nil {
		return fmt.Errorf("ensure Tbilisi city: %w", err)
	}

	// Fetch POIs from Overpass API.
	query := buildOverpassQuery()
	elements, err := fetchOverpassData(query)
	if err != nil {
		return fmt.Errorf("fetch Overpass data: %w", err)
	}

	// Deduplicate elements by OSM ID to avoid duplicate ways/nodes.
	seen := make(map[string]bool)
	var unique []overpassElement
	for _, el := range elements {
		key := fmt.Sprintf("%s/%d", el.Type, el.ID)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, el)
		}
	}
	log.Printf("After OSM deduplication: %d unique elements", len(unique))

	// Import POIs into the database.
	stats := importStats{}

	for _, el := range unique {
		lat, lng := el.Lat, el.Lon
		if el.Type != "node" && el.Center != nil {
			lat, lng = el.Center.Lat, el.Center.Lon
		}

		// Skip elements without valid coordinates.
		if lat == 0 && lng == 0 {
			stats.errors++
			continue
		}

		// Extract name.
		name := getName(el.Tags)
		if name == "" {
			stats.noName++
			continue
		}

		// Check for existing POI nearby (deduplication).
		exists, checkErr := poiExistsNearby(ctx, pool, cityID, lat, lng)
		if checkErr != nil {
			log.Printf("  ERROR checking dedup for %q: %v", name, checkErr)
			stats.errors++
			continue
		}
		if exists {
			stats.skipped++
			continue
		}

		// Map OSM tags to POI type and score.
		poiType := mapOSMToPOIType(el.Tags)
		interestScore := defaultInterestScore(poiType)
		nameRu := getNameRu(el.Tags)
		address := getAddress(el.Tags)

		// Store relevant OSM tags.
		osmTags := make(map[string]string)
		for _, k := range []string{"wikidata", "wikipedia", "website", "opening_hours", "phone", "tourism", "historic", "amenity", "leisure"} {
			if v, ok := el.Tags[k]; ok {
				osmTags[k] = v
			}
		}

		if insertErr := insertPOI(ctx, pool, cityID, name, nameRu, lat, lng, poiType, address, interestScore, osmTags); insertErr != nil {
			log.Printf("  ERROR inserting %q: %v", name, insertErr)
			stats.errors++
			continue
		}

		stats.imported++
	}

	// Print summary.
	fmt.Println()
	fmt.Println("=== Import Summary ===")
	fmt.Printf("  Imported: %d\n", stats.imported)
	fmt.Printf("  Skipped (duplicate): %d\n", stats.skipped)
	fmt.Printf("  Skipped (no name): %d\n", stats.noName)
	fmt.Printf("  Errors: %d\n", stats.errors)
	fmt.Printf("  Total processed: %d\n", stats.imported+stats.skipped+stats.noName+stats.errors)

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
