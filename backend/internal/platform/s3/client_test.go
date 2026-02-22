//go:build integration

package s3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func testConfig() Config {
	endpoint := os.Getenv("S3_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:9000"
	}
	accessKey := os.Getenv("S3_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey := os.Getenv("S3_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin_secret"
	}
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "city-stories-test"
	}
	return Config{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
	}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()
	ctx := context.Background()
	cfg := testConfig()
	client, err := NewClient(ctx, &cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestNewClient_CreatesBucket(t *testing.T) {
	_ = newTestClient(t)
	// If we got here without error, the bucket was created or already existed.
}

func TestUpload_And_Exists(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := "test/upload-test.txt"
	content := []byte("hello, city stories!")

	url, err := c.Upload(ctx, key, bytes.NewReader(content), "text/plain")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if url == "" {
		t.Fatal("Upload returned empty URL")
	}

	exists, err := c.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("Exists returned false after Upload")
	}

	// Cleanup
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete cleanup: %v", err)
	}
}

func TestUpload_MP3_ContentType(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := AudioKey(1, 10, 100)
	fakeMP3 := []byte("fake mp3 data for testing")

	url, err := c.Upload(ctx, key, bytes.NewReader(fakeMP3), "audio/mpeg")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	t.Logf("Uploaded to: %s", url)

	// Cleanup
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete cleanup: %v", err)
	}
}

func TestGetPresignedURL(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := "test/presign-test.txt"
	content := []byte("presigned content")

	_, err := c.Upload(ctx, key, bytes.NewReader(content), "text/plain")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	presignedURL, err := c.GetPresignedURL(ctx, key, 15*time.Minute)
	if err != nil {
		t.Fatalf("GetPresignedURL: %v", err)
	}
	if presignedURL == "" {
		t.Fatal("GetPresignedURL returned empty URL")
	}
	t.Logf("Presigned URL: %s", presignedURL)

	// Verify the presigned URL actually works by downloading
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(presignedURL) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("GET presigned URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET presigned URL returned %d, want 200", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(body, content) {
		t.Fatalf("Downloaded content = %q, want %q", body, content)
	}

	// Cleanup
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete cleanup: %v", err)
	}
}

func TestDelete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := "test/delete-test.txt"
	content := []byte("to be deleted")

	_, err := c.Upload(ctx, key, bytes.NewReader(content), "text/plain")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	// Verify it exists
	exists, err := c.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists before delete: %v", err)
	}
	if !exists {
		t.Fatal("Object should exist before delete")
	}

	// Delete
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone
	exists, err = c.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("Object should not exist after delete")
	}
}

func TestExists_NotFound(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	exists, err := c.Exists(ctx, "test/nonexistent-file.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("Exists should return false for nonexistent key")
	}
}

func TestFullCycle_Upload_Presign_Delete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := AudioKey(1, 42, 100)
	mp3Data := []byte("simulated mp3 audio content for full cycle test")

	// 1. Upload
	url, err := c.Upload(ctx, key, bytes.NewReader(mp3Data), "audio/mpeg")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	t.Logf("Upload URL: %s", url)

	// 2. Verify exists
	exists, err := c.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("Object should exist after upload")
	}

	// 3. Get presigned URL and download
	presignedURL, err := c.GetPresignedURL(ctx, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("GetPresignedURL: %v", err)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(presignedURL) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("GET presigned URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Download status = %d, want 200", resp.StatusCode)
	}

	downloaded, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(downloaded, mp3Data) {
		t.Fatalf("Downloaded data mismatch")
	}

	// 4. Delete
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// 5. Verify gone
	exists, err = c.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("Object should not exist after delete")
	}
}

func TestAudioKey(t *testing.T) {
	tests := []struct {
		cityID, poiID, storyID int
		want                   string
	}{
		{1, 10, 100, "audio/1/10/100.mp3"},
		{5, 42, 7, "audio/5/42/7.mp3"},
	}

	for _, tt := range tests {
		got := AudioKey(tt.cityID, tt.poiID, tt.storyID)
		if got != tt.want {
			t.Errorf("AudioKey(%d, %d, %d) = %q, want %q", tt.cityID, tt.poiID, tt.storyID, got, tt.want)
		}
	}
}
