package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/claude"
	"github.com/saas/city-stories-guide/backend/internal/platform/elevenlabs"
	s3client "github.com/saas/city-stories-guide/backend/internal/platform/s3"
)

const (
	// Delays between API calls to respect rate limits.
	claudeDelay     = 2 * time.Second
	elevenLabsDelay = 1 * time.Second

	// Estimated words-per-minute for audio duration calculation.
	wordsPerMinute = 150.0
)

// genStats tracks pipeline progress.
type genStats struct {
	generated int
	skipped   int
	errors    int
	total     int
}

// fetchPOIsWithoutStories returns active POIs that have no stories in the given language.
func fetchPOIsWithoutStories(ctx context.Context, pool *pgxpool.Pool, cityID int, language string, limit int) ([]domain.POI, error) {
	query := `
		SELECT p.id, p.city_id, p.name, p.name_ru,
		       ST_Y(p.location::geometry) AS lat,
		       ST_X(p.location::geometry) AS lng,
		       p.type, p.tags, p.address, p.interest_score, p.status, p.created_at, p.updated_at
		FROM poi p
		WHERE p.city_id = $1
		  AND p.status = 'active'
		  AND NOT EXISTS (
		    SELECT 1 FROM story s
		    WHERE s.poi_id = p.id AND s.language = $2
		  )
		ORDER BY p.interest_score DESC, p.id
		LIMIT $3`

	rows, err := pool.Query(ctx, query, cityID, language, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch POIs without stories: %w", err)
	}
	defer rows.Close()

	var pois []domain.POI
	for rows.Next() {
		var p domain.POI
		if err := rows.Scan(
			&p.ID, &p.CityID, &p.Name, &p.NameRu,
			&p.Lat, &p.Lng,
			&p.Type, &p.Tags, &p.Address, &p.InterestScore, &p.Status, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan POI: %w", err)
		}
		pois = append(pois, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate POIs: %w", err)
	}

	return pois, nil
}

// insertStory inserts a story record into the database and returns the generated ID.
func insertStory(ctx context.Context, pool *pgxpool.Pool, story *domain.Story) (int, error) {
	query := `
		INSERT INTO story (poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	var id int
	err := pool.QueryRow(ctx, query,
		story.POIID,
		story.Language,
		story.Text,
		story.AudioURL,
		story.DurationSec,
		story.LayerType,
		story.OrderIndex,
		story.IsInflation,
		story.Confidence,
		story.Sources,
		story.Status,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert story: %w", err)
	}

	return id, nil
}

// updateStoryAudioURL updates the audio_url and duration_sec for a story.
func updateStoryAudioURL(ctx context.Context, pool *pgxpool.Pool, storyID int, audioURL string, durationSec int16) error {
	_, err := pool.Exec(ctx, `
		UPDATE story SET audio_url = $2, duration_sec = $3, updated_at = NOW()
		WHERE id = $1`, storyID, audioURL, durationSec)
	if err != nil {
		return fmt.Errorf("update story audio: %w", err)
	}
	return nil
}

// clampConfidence safely converts a confidence value to int16 within [0, 100].
func clampConfidence(v int) int16 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return int16(v) //nolint:gosec // value is clamped to [0, 100]
}

// estimateDurationSec estimates audio duration from word count.
func estimateDurationSec(text string) int16 {
	words := len(strings.Fields(text))
	seconds := float64(words) / wordsPerMinute * 60.0
	if seconds < 10 {
		seconds = 10
	}
	if seconds > 120 {
		seconds = 120
	}
	return int16(seconds)
}

// getCityID looks up a city by ID or returns the first active city.
func getCityID(ctx context.Context, pool *pgxpool.Pool, requestedID int) (cityID int, cityName string, err error) {
	if requestedID > 0 {
		err = pool.QueryRow(ctx, `SELECT name FROM cities WHERE id = $1`, requestedID).Scan(&cityName)
		if err != nil {
			if err == pgx.ErrNoRows {
				return 0, "", fmt.Errorf("city with id=%d not found", requestedID)
			}
			return 0, "", fmt.Errorf("lookup city: %w", err)
		}
		return requestedID, cityName, nil
	}

	err = pool.QueryRow(ctx, `SELECT id, name FROM cities WHERE is_active = true ORDER BY id LIMIT 1`).Scan(&cityID, &cityName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, "", fmt.Errorf("no active cities found")
		}
		return 0, "", fmt.Errorf("lookup first city: %w", err)
	}
	return cityID, cityName, nil
}

// processPOI generates a story (text + audio) for a single POI in one language.
func processPOI(
	ctx context.Context,
	pool *pgxpool.Pool,
	claudeClient *claude.Client,
	elevenLabsClient *elevenlabs.Client,
	s3Client *s3client.Client,
	poi *domain.POI,
	language string,
) error {
	// Step 1: Generate story text via Claude
	log.Printf("  [Claude] Generating %s story for POI %d (%s)...", strings.ToUpper(language), poi.ID, poi.Name)
	storyResult, err := claudeClient.GenerateStory(ctx, poi, language)
	if err != nil {
		return fmt.Errorf("claude generate: %w", err)
	}
	log.Printf("  [Claude] Done: %d tokens in, %d tokens out, %s, layer=%s, confidence=%d",
		storyResult.TokensIn, storyResult.TokensOut, storyResult.Duration.Round(time.Millisecond), storyResult.LayerType, storyResult.Confidence)

	// Step 2: Insert story into DB (without audio_url initially)
	durationEst := estimateDurationSec(storyResult.Text)
	sourcesJSON, _ := json.Marshal(map[string]string{"generator": "claude", "model": "claude-sonnet-4-20250514"})

	story := &domain.Story{
		POIID:       poi.ID,
		Language:    language,
		Text:        storyResult.Text,
		DurationSec: &durationEst,
		LayerType:   storyResult.LayerType,
		OrderIndex:  0,
		IsInflation: false,
		Confidence:  clampConfidence(storyResult.Confidence),
		Sources:     sourcesJSON,
		Status:      domain.StoryStatusActive,
	}

	storyID, err := insertStory(ctx, pool, story)
	if err != nil {
		return fmt.Errorf("insert story: %w", err)
	}
	log.Printf("  [DB] Story created with id=%d", storyID)

	// Rate limit before TTS call
	time.Sleep(elevenLabsDelay)

	// Step 3: Generate audio via ElevenLabs
	log.Printf("  [ElevenLabs] Generating audio for story %d...", storyID)
	audioResult, err := elevenLabsClient.GenerateAudio(ctx, storyResult.Text, language)
	if err != nil {
		log.Printf("  [ElevenLabs] WARNING: audio generation failed for story %d: %v (story text saved without audio)", storyID, err)
		return nil // Story is saved without audio — not a fatal error
	}
	log.Printf("  [ElevenLabs] Audio generated in %s", audioResult.Duration.Round(time.Millisecond))

	// Read audio into buffer for upload
	audioData, err := io.ReadAll(audioResult.Audio)
	if err != nil {
		log.Printf("  [ElevenLabs] WARNING: failed to read audio data for story %d: %v", storyID, err)
		return nil
	}

	// Step 4: Upload audio to S3
	s3Key := s3client.AudioKey(poi.CityID, poi.ID, storyID)
	log.Printf("  [S3] Uploading to %s...", s3Key)
	audioURL, err := s3Client.Upload(ctx, s3Key, bytes.NewReader(audioData), "audio/mpeg")
	if err != nil {
		log.Printf("  [S3] WARNING: upload failed for story %d: %v (story text saved without audio)", storyID, err)
		return nil
	}
	log.Printf("  [S3] Uploaded: %s", audioURL)

	// Step 5: Update story with audio_url and duration
	if err := updateStoryAudioURL(ctx, pool, storyID, audioURL, durationEst); err != nil {
		return fmt.Errorf("update story audio url: %w", err)
	}
	log.Printf("  [DB] Story %d updated with audio_url", storyID)

	return nil
}

func run() error {
	_ = godotenv.Load()

	// Parse command-line flags
	cityID := flag.Int("city", 0, "City ID to process (default: first active city)")
	limit := flag.Int("limit", 0, "Maximum number of POIs to process (0 = all)")
	languages := flag.String("languages", "en,ru", "Comma-separated languages to generate (e.g. en,ru)")
	dryRun := flag.Bool("dry-run", false, "Show what would be processed without generating")
	flag.Parse()

	// Parse languages
	langs := strings.Split(*languages, ",")
	for i := range langs {
		langs[i] = strings.TrimSpace(langs[i])
	}

	// Validate environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}
	claudeKey := os.Getenv("CLAUDE_API_KEY")
	if claudeKey == "" {
		return fmt.Errorf("CLAUDE_API_KEY environment variable is required")
	}
	elevenLabsKey := os.Getenv("ELEVENLABS_API_KEY")
	if elevenLabsKey == "" {
		return fmt.Errorf("ELEVENLABS_API_KEY environment variable is required")
	}
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "city-stories"
	}

	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	if pingErr := pool.Ping(ctx); pingErr != nil {
		return fmt.Errorf("ping database: %w", pingErr)
	}
	log.Println("Database connected")

	// Resolve city
	resolvedCityID, cityName, err := getCityID(ctx, pool, *cityID)
	if err != nil {
		return fmt.Errorf("resolve city: %w", err)
	}
	log.Printf("Processing city: %s (id=%d)", cityName, resolvedCityID)
	log.Printf("Languages: %s", strings.Join(langs, ", "))

	// Initialize API clients
	claudeClient := claude.NewClient(&claude.Config{
		APIKey: claudeKey,
	})

	elevenLabsClient := elevenlabs.NewClient(&elevenlabs.Config{
		APIKey: elevenLabsKey,
	})

	s3Client, err := s3client.NewClient(ctx, &s3client.Config{
		Endpoint:  s3Endpoint,
		AccessKey: s3AccessKey,
		SecretKey: s3SecretKey,
		Bucket:    s3Bucket,
	})
	if err != nil {
		return fmt.Errorf("initialize S3 client: %w", err)
	}
	log.Println("S3 client initialized")

	// Process each language
	totalStats := genStats{}
	pipelineStart := time.Now()

	for _, lang := range langs {
		log.Printf("\n=== Processing language: %s ===", strings.ToUpper(lang))

		// Determine POI limit per language
		poiLimit := 1000
		if *limit > 0 {
			poiLimit = *limit
		}

		// Fetch POIs that need stories
		pois, err := fetchPOIsWithoutStories(ctx, pool, resolvedCityID, lang, poiLimit)
		if err != nil {
			return fmt.Errorf("fetch POIs for %s: %w", lang, err)
		}

		log.Printf("Found %d POIs without %s stories", len(pois), strings.ToUpper(lang))
		totalStats.total += len(pois)

		if *dryRun {
			for i := range pois {
				log.Printf("  [%d/%d] Would process POI %d: %s (type=%s, score=%d)",
					i+1, len(pois), pois[i].ID, pois[i].Name, pois[i].Type, pois[i].InterestScore)
			}
			continue
		}

		// Process each POI
		for i := range pois {
			log.Printf("\n[%d/%d] POI %d: %s (type=%s, score=%d)",
				i+1, len(pois), pois[i].ID, pois[i].Name, pois[i].Type, pois[i].InterestScore)

			if err := processPOI(ctx, pool, claudeClient, elevenLabsClient, s3Client, &pois[i], lang); err != nil {
				log.Printf("  ERROR: %v", err)
				totalStats.errors++
				// Continue with next POI
				continue
			}

			totalStats.generated++

			// Rate limit between POIs to avoid hitting API limits
			if i < len(pois)-1 {
				time.Sleep(claudeDelay)
			}
		}
	}

	// Print summary
	elapsed := time.Since(pipelineStart).Round(time.Second)
	fmt.Println()
	fmt.Println("=== Generation Summary ===")
	fmt.Printf("  City: %s (id=%d)\n", cityName, resolvedCityID)
	fmt.Printf("  Languages: %s\n", strings.Join(langs, ", "))
	fmt.Printf("  Generated: %d stories\n", totalStats.generated)
	fmt.Printf("  Skipped (already exist): %d\n", totalStats.skipped)
	fmt.Printf("  Errors: %d\n", totalStats.errors)
	fmt.Printf("  Total POIs to process: %d\n", totalStats.total)
	fmt.Printf("  Duration: %s\n", elapsed)

	if *dryRun {
		fmt.Println("  (DRY RUN — no stories were actually generated)")
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
