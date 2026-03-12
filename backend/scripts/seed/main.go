package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type seedPOI struct {
	Name          string
	NameRu        string
	Lat           float64
	Lng           float64
	Type          string
	Address       string
	InterestScore int16
	StoriesEN     []seedStory
	StoriesRU     []seedStory
}

type seedStory struct {
	Text      string
	LayerType string
	Duration  int16
}

// tbilisiPOIs is defined in tbilisi_pois.go

func ensureTbilisiCity(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var cityID int
	err := pool.QueryRow(ctx, `SELECT id FROM cities WHERE name = $1`, "Tbilisi").Scan(&cityID)
	if err == nil {
		log.Printf("Tbilisi already exists (id=%d), skipping city insert", cityID)
		return cityID, nil
	}

	nameRu := "Тбилиси"
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		"Tbilisi", &nameRu, "Georgia", 41.7151, 44.8271, 10.0, true, 0.0,
	).Scan(&cityID)
	if err != nil {
		return 0, fmt.Errorf("insert Tbilisi: %w", err)
	}

	log.Printf("Created Tbilisi (id=%d)", cityID)
	return cityID, nil
}

func poiExistsByName(ctx context.Context, pool *pgxpool.Pool, cityID int, name string) (int, bool) {
	var poiID int
	err := pool.QueryRow(ctx, `SELECT id FROM poi WHERE city_id = $1 AND name = $2`, cityID, name).Scan(&poiID)
	if err != nil {
		return 0, false
	}
	return poiID, true
}

func insertPOI(ctx context.Context, pool *pgxpool.Pool, cityID int, p *seedPOI) (int, error) {
	tags, _ := json.Marshal(map[string]string{"source": "seed"})

	var poiID int
	err := pool.QueryRow(ctx, `
		INSERT INTO poi (city_id, name, name_ru, location, type, tags, address, interest_score, status)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography, $6, $7, $8, $9, 'active')
		RETURNING id`,
		cityID, p.Name, &p.NameRu, p.Lng, p.Lat, p.Type, tags, &p.Address, p.InterestScore,
	).Scan(&poiID)
	if err != nil {
		return 0, fmt.Errorf("insert poi %q: %w", p.Name, err)
	}
	return poiID, nil
}

func storyExists(ctx context.Context, pool *pgxpool.Pool, poiID int, lang string) bool {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM story WHERE poi_id = $1 AND language = $2)`, poiID, lang).Scan(&exists)
	return err == nil && exists
}

func insertStory(ctx context.Context, pool *pgxpool.Pool, poiID int, lang string, s seedStory) (int, error) {
	fakeAudioURL := fmt.Sprintf("https://example.com/audio/seed/%d_%s.mp3", poiID, lang)
	sources, _ := json.Marshal([]string{"seed_data"})

	var storyID int
	err := pool.QueryRow(ctx, `
		INSERT INTO story (poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status)
		VALUES ($1, $2, $3, $4, $5, $6, 0, false, 90, $7, 'active')
		RETURNING id`,
		poiID, lang, s.Text, fakeAudioURL, s.Duration, s.LayerType, sources,
	).Scan(&storyID)
	if err != nil {
		return 0, fmt.Errorf("insert story for poi %d (%s): %w", poiID, lang, err)
	}
	return storyID, nil
}

// ensureDemoUsers creates an admin user and a regular user for testing.
// Admin: admin@demo.local / demodemo
// User:  user@demo.local  / demodemo
func ensureDemoUsers(ctx context.Context, pool *pgxpool.Pool) (adminID, userID string, err error) {
	hash, err := bcrypt.GenerateFromPassword([]byte("demodemo"), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash password: %w", err)
	}

	users := []struct {
		email   string
		name    string
		isAdmin bool
	}{
		{"admin@demo.local", "Demo Admin", true},
		{"user@demo.local", "Demo User", false},
	}

	ids := make([]string, 2)
	for i, u := range users {
		var id string
		// Try to find existing user first.
		err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, u.email).Scan(&id)
		if err == nil {
			log.Printf("User %s already exists (id=%s), skipping", u.email, id)
			ids[i] = id
			continue
		}

		err = pool.QueryRow(ctx, `
			INSERT INTO users (email, name, auth_provider, password_hash, is_admin, is_anonymous, language_pref)
			VALUES ($1, $2, 'email', $3, $4, false, 'en')
			RETURNING id`,
			u.email, u.name, string(hash), u.isAdmin,
		).Scan(&id)
		if err != nil {
			return "", "", fmt.Errorf("insert user %s: %w", u.email, err)
		}
		log.Printf("Created user %s (id=%s, admin=%v)", u.email, id, u.isAdmin)
		ids[i] = id
	}

	return ids[0], ids[1], nil
}

// seedReports creates sample reports against the first few seeded stories.
func seedReports(ctx context.Context, pool *pgxpool.Pool, userID string) (int, error) {
	reports := []struct {
		reportType string
		comment    string
		status     string
		lat, lng   float64
	}{
		{"wrong_location", "This story plays at the wrong spot, should be 50m north", "new", 41.6880, 44.8090},
		{"wrong_fact", "The date mentioned is incorrect — fortress was rebuilt in the 5th century, not 4th", "new", 41.6875, 44.8089},
		{"inappropriate_content", "Story contains outdated terminology", "reviewed", 41.6950, 44.8015},
		{"wrong_location", "Story plays inside the park but the POI is on the street", "resolved", 41.7090, 44.7935},
		{"wrong_fact", "Population figure cited is from 2010, should be updated", "new", 41.7005, 44.7930},
	}

	// Get story IDs to attach reports to.
	rows, err := pool.Query(ctx, `SELECT id FROM story WHERE sources @> '["seed_data"]' ORDER BY id LIMIT $1`, len(reports))
	if err != nil {
		return 0, fmt.Errorf("query seed stories: %w", err)
	}
	defer rows.Close()

	var storyIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		storyIDs = append(storyIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(storyIDs) == 0 {
		log.Println("No seed stories found, skipping reports")
		return 0, nil
	}

	var created int
	for i, r := range reports {
		storyID := storyIDs[i%len(storyIDs)]

		// Check if report already exists (idempotency by comment).
		var exists bool
		_ = pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM report WHERE story_id = $1 AND user_id = $2 AND comment = $3)`,
			storyID, userID, r.comment,
		).Scan(&exists)
		if exists {
			continue
		}

		_, err := pool.Exec(ctx, `
			INSERT INTO report (story_id, user_id, type, comment, user_lat, user_lng, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			storyID, userID, r.reportType, r.comment, r.lat, r.lng, r.status,
		)
		if err != nil {
			log.Printf("  ERROR inserting report: %v", err)
			continue
		}
		created++
	}

	return created, nil
}

// seedListenings creates sample listening records for the demo user.
func seedListenings(ctx context.Context, pool *pgxpool.Pool, userID string) (int, error) {
	rows, err := pool.Query(ctx,
		`SELECT s.id, p.location FROM story s JOIN poi p ON s.poi_id = p.id WHERE s.sources @> '["seed_data"]' AND s.language = 'en' ORDER BY s.id LIMIT 8`)
	if err != nil {
		return 0, fmt.Errorf("query seed stories for listenings: %w", err)
	}
	defer rows.Close()

	type storyLoc struct {
		storyID  int
		location any // geography bytes, passed back as-is
	}
	var items []storyLoc
	for rows.Next() {
		var sl storyLoc
		if err := rows.Scan(&sl.storyID, &sl.location); err != nil {
			return 0, err
		}
		items = append(items, sl)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var created int
	for i, sl := range items {
		var exists bool
		_ = pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM user_listening WHERE user_id = $1 AND story_id = $2)`,
			userID, sl.storyID,
		).Scan(&exists)
		if exists {
			continue
		}

		completed := i < len(items)/2 // first half completed
		_, err := pool.Exec(ctx, `
			INSERT INTO user_listening (user_id, story_id, completed, location)
			VALUES ($1, $2, $3, $4)`,
			userID, sl.storyID, completed, sl.location,
		)
		if err != nil {
			log.Printf("  ERROR inserting listening: %v", err)
			continue
		}
		created++
	}

	return created, nil
}

func run() error {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Safety: refuse to run against anything that looks like production.
	lower := strings.ToLower(dbURL)
	for _, keyword := range []string{"prod", "production", "rds.amazonaws.com", "cloud.google.com"} {
		if strings.Contains(lower, keyword) {
			return fmt.Errorf("DATABASE_URL contains %q — refusing to seed what looks like a production database", keyword)
		}
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

	// 1. City + POIs + Stories (existing flow).
	cityID, err := ensureTbilisiCity(ctx, pool)
	if err != nil {
		return err
	}

	pois := tbilisiPOIs()
	var poisCreated, poisSkipped, storiesCreated, storiesSkipped int

	for i := range pois {
		p := &pois[i]
		poiID, exists := poiExistsByName(ctx, pool, cityID, p.Name)
		if exists {
			poisSkipped++
			log.Printf("  POI %q already exists (id=%d), skipping", p.Name, poiID)
		} else {
			poiID, err = insertPOI(ctx, pool, cityID, p)
			if err != nil {
				log.Printf("  ERROR: %v", err)
				continue
			}
			poisCreated++
			log.Printf("  Created POI %q (id=%d)", p.Name, poiID)
		}

		for _, s := range p.StoriesEN {
			if storyExists(ctx, pool, poiID, "en") {
				storiesSkipped++
				continue
			}
			if _, sErr := insertStory(ctx, pool, poiID, "en", s); sErr != nil {
				log.Printf("  ERROR: %v", sErr)
				continue
			}
			storiesCreated++
		}

		for _, s := range p.StoriesRU {
			if storyExists(ctx, pool, poiID, "ru") {
				storiesSkipped++
				continue
			}
			if _, sErr := insertStory(ctx, pool, poiID, "ru", s); sErr != nil {
				log.Printf("  ERROR: %v", sErr)
				continue
			}
			storiesCreated++
		}
	}

	// 2. Demo users (admin + regular).
	adminID, userID, err := ensureDemoUsers(ctx, pool)
	if err != nil {
		return fmt.Errorf("seed users: %w", err)
	}
	_ = adminID

	// 3. Reports (attached to seed stories, filed by demo user).
	reportsCreated, err := seedReports(ctx, pool, userID)
	if err != nil {
		return fmt.Errorf("seed reports: %w", err)
	}

	// 4. Listening history (demo user listened to some stories).
	listeningsCreated, err := seedListenings(ctx, pool, userID)
	if err != nil {
		return fmt.Errorf("seed listenings: %w", err)
	}

	fmt.Println()
	fmt.Println("=== Seed Summary ===")
	fmt.Printf("  POIs created:       %d\n", poisCreated)
	fmt.Printf("  POIs skipped:       %d\n", poisSkipped)
	fmt.Printf("  Stories created:    %d\n", storiesCreated)
	fmt.Printf("  Stories skipped:    %d\n", storiesSkipped)
	fmt.Printf("  Reports created:   %d\n", reportsCreated)
	fmt.Printf("  Listenings created: %d\n", listeningsCreated)
	fmt.Printf("  Total POIs:         %d\n", len(pois))
	fmt.Printf("  Total stories:      %d (%d EN + %d RU)\n", len(pois)*2, len(pois), len(pois))
	fmt.Println()
	fmt.Println("Demo credentials:")
	fmt.Println("  Admin: admin@demo.local / demodemo")
	fmt.Println("  User:  user@demo.local  / demodemo")

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
