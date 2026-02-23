package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
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

func insertStory(ctx context.Context, pool *pgxpool.Pool, poiID int, lang string, s seedStory) error {
	fakeAudioURL := fmt.Sprintf("https://example.com/audio/seed/%d_%s.mp3", poiID, lang)
	sources, _ := json.Marshal([]string{"seed_data"})

	_, err := pool.Exec(ctx, `
		INSERT INTO story (poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status)
		VALUES ($1, $2, $3, $4, $5, $6, 0, false, 90, $7, 'active')`,
		poiID, lang, s.Text, fakeAudioURL, s.Duration, s.LayerType, sources,
	)
	if err != nil {
		return fmt.Errorf("insert story for poi %d (%s): %w", poiID, lang, err)
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
			if sErr := insertStory(ctx, pool, poiID, "en", s); sErr != nil {
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
			if sErr := insertStory(ctx, pool, poiID, "ru", s); sErr != nil {
				log.Printf("  ERROR: %v", sErr)
				continue
			}
			storiesCreated++
		}
	}

	fmt.Println()
	fmt.Println("=== Seed Summary ===")
	fmt.Printf("  POIs created:    %d\n", poisCreated)
	fmt.Printf("  POIs skipped:    %d\n", poisSkipped)
	fmt.Printf("  Stories created: %d\n", storiesCreated)
	fmt.Printf("  Stories skipped: %d\n", storiesSkipped)
	fmt.Printf("  Total POIs:      %d\n", len(pois))
	fmt.Printf("  Total stories:   %d (%d EN + %d RU)\n", len(pois)*2, len(pois), len(pois))

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
