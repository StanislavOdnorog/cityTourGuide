// Package mock provides deterministic mock implementations of external
// integration clients (Claude, ElevenLabs, S3) for local development.
//
// These mocks are activated by setting PROVIDER_MODE=mock and allow the full
// application flow to run without live API credentials. They return stable,
// predictable outputs suitable for manual QA and onboarding.
package mock

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/platform/claude"
	"github.com/saas/city-stories-guide/backend/internal/platform/elevenlabs"
)

// StoryGenerator returns deterministic story text for any POI.
type StoryGenerator struct{}

func NewStoryGenerator() *StoryGenerator {
	slog.Warn("using MOCK story generator — stories will contain placeholder text")
	return &StoryGenerator{}
}

func (m *StoryGenerator) GenerateStory(_ context.Context, poi *domain.POI, language string) (*claude.StoryResult, error) {
	text := fmt.Sprintf(
		"[MOCK] This is a generated story about %s. "+
			"Located at coordinates (%.4f, %.4f), this %s has a rich history "+
			"waiting to be explored. In a real deployment the Claude API would "+
			"generate a unique narrative for this point of interest.",
		poi.DisplayName(language), poi.Lat, poi.Lng, poi.Type,
	)

	return &claude.StoryResult{
		Text:       text,
		LayerType:  domain.StoryLayerGeneral,
		Confidence: 80,
		TokensIn:   150,
		TokensOut:  len(text) / 4, // rough approximation
		Duration:   50 * time.Millisecond,
	}, nil
}

// AudioGenerator returns a minimal valid MP3 frame as placeholder audio.
type AudioGenerator struct{}

func NewAudioGenerator() *AudioGenerator {
	slog.Warn("using MOCK audio generator — audio files will be silent placeholders")
	return &AudioGenerator{}
}

// silentMP3Frame is a minimal valid MPEG audio frame (Layer III, 128 kbps, 44100 Hz,
// stereo) containing silence. 417 bytes — enough for players to recognize the format.
var silentMP3Frame = func() []byte {
	// MPEG1 Layer 3 frame header: 0xFF 0xFB 0x90 0x00
	// This encodes: sync(12 bits), MPEG1, Layer III, no CRC, 128kbps, 44100Hz, stereo.
	header := []byte{0xFF, 0xFB, 0x90, 0x00}
	frame := make([]byte, 417) // standard frame size for 128kbps/44100Hz
	copy(frame, header)
	return frame
}()

func (m *AudioGenerator) GenerateAudio(_ context.Context, _, _ string) (*elevenlabs.AudioResult, error) {
	return &elevenlabs.AudioResult{
		Audio:    bytes.NewReader(silentMP3Frame),
		Duration: 26 * time.Millisecond, // duration of one 128kbps frame
	}, nil
}

// ObjectStore is an in-memory object store for development.
// It satisfies both the ObjectStorage and ObjectDeleter interfaces.
type ObjectStore struct {
	mu       sync.RWMutex
	objects  map[string][]byte
	endpoint string
	bucket   string
}

func NewObjectStore() *ObjectStore {
	slog.Warn("using MOCK object storage — uploads are stored in memory only")
	return &ObjectStore{
		objects:  make(map[string][]byte),
		endpoint: "http://mock-s3",
		bucket:   "mock-bucket",
	}
}

func (m *ObjectStore) Upload(_ context.Context, key string, reader io.Reader, _ string) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("mock s3: read upload data: %w", err)
	}

	m.mu.Lock()
	m.objects[key] = data
	m.mu.Unlock()

	url := fmt.Sprintf("%s/%s/%s", m.endpoint, m.bucket, key)
	return url, nil
}

func (m *ObjectStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.objects, key)
	m.mu.Unlock()
	return nil
}

func (m *ObjectStore) Exists(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	_, ok := m.objects[key]
	m.mu.RUnlock()
	return ok, nil
}
