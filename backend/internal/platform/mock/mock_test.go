package mock

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

func TestStoryGenerator_ReturnsDeterministicOutput(t *testing.T) {
	gen := &StoryGenerator{}
	poi := &domain.POI{
		ID:     1,
		CityID: 1,
		Name:   "Test Monument",
		Lat:    41.7,
		Lng:    44.8,
		Type:   domain.POITypeMonument,
	}

	result, err := gen.GenerateStory(context.Background(), poi, "en")
	if err != nil {
		t.Fatalf("GenerateStory() error = %v", err)
	}

	if !strings.Contains(result.Text, "Test Monument") {
		t.Errorf("story text should contain POI name, got: %s", result.Text)
	}
	if !strings.Contains(result.Text, "[MOCK]") {
		t.Errorf("story text should be prefixed with [MOCK], got: %s", result.Text)
	}
	if result.LayerType != domain.StoryLayerGeneral {
		t.Errorf("LayerType = %q, want %q", result.LayerType, domain.StoryLayerGeneral)
	}
	if result.Confidence != 80 {
		t.Errorf("Confidence = %d, want 80", result.Confidence)
	}

	// Verify determinism — second call returns identical result.
	result2, _ := gen.GenerateStory(context.Background(), poi, "en")
	if result.Text != result2.Text {
		t.Error("GenerateStory should return deterministic output")
	}
}

func TestAudioGenerator_ReturnsValidData(t *testing.T) {
	gen := &AudioGenerator{}

	result, err := gen.GenerateAudio(context.Background(), "hello", "en")
	if err != nil {
		t.Fatalf("GenerateAudio() error = %v", err)
	}

	data, err := io.ReadAll(result.Audio)
	if err != nil {
		t.Fatalf("reading audio: %v", err)
	}

	if len(data) == 0 {
		t.Error("audio data should not be empty")
	}
	// Verify MP3 sync word (0xFF 0xFB).
	if data[0] != 0xFF || data[1] != 0xFB {
		t.Errorf("audio should start with MP3 sync bytes, got %02X %02X", data[0], data[1])
	}
	if result.Duration <= 0 {
		t.Error("duration should be positive")
	}
}

func TestObjectStore_UploadAndRetrieve(t *testing.T) {
	store := &ObjectStore{
		objects:  make(map[string][]byte),
		endpoint: "http://mock-s3",
		bucket:   "mock-bucket",
	}
	ctx := context.Background()

	// Upload
	url, err := store.Upload(ctx, "audio/1/2/3.mp3", strings.NewReader("fake-audio"), "audio/mpeg")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if url != "http://mock-s3/mock-bucket/audio/1/2/3.mp3" {
		t.Errorf("URL = %q, want http://mock-s3/mock-bucket/audio/1/2/3.mp3", url)
	}

	// Exists
	exists, err := store.Exists(ctx, "audio/1/2/3.mp3")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false after Upload, want true")
	}

	// Delete
	if err := store.Delete(ctx, "audio/1/2/3.mp3"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	exists, err = store.Exists(ctx, "audio/1/2/3.mp3")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true after Delete, want false")
	}
}

func TestObjectStore_DeleteNonExistent(t *testing.T) {
	store := &ObjectStore{
		objects:  make(map[string][]byte),
		endpoint: "http://mock-s3",
		bucket:   "mock-bucket",
	}

	// Deleting a non-existent key should succeed silently.
	if err := store.Delete(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestStoryGenerator_UsesDisplayName(t *testing.T) {
	gen := &StoryGenerator{}
	ruName := "Тестовый Монумент"
	poi := &domain.POI{
		Name:   "Test Monument",
		NameRu: &ruName,
		Lat:    41.7,
		Lng:    44.8,
		Type:   domain.POITypeMonument,
		Tags:   json.RawMessage(`[]`),
	}

	result, _ := gen.GenerateStory(context.Background(), poi, "ru")
	if !strings.Contains(result.Text, ruName) {
		t.Errorf("story text should use Russian display name, got: %s", result.Text)
	}
}
