package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
)

// Client wraps an S3-compatible storage client for uploading, downloading,
// and managing audio files.
type Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	endpoint      string
}

// Config holds the settings for initializing an S3 client.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
}

// NewClient creates a new S3-compatible storage client and ensures the bucket exists.
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: failed to load config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	c := &Client{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.Bucket,
		endpoint:      cfg.Endpoint,
	}

	if err := c.ensureBucket(ctx); err != nil {
		return nil, fmt.Errorf("s3: failed to ensure bucket: %w", err)
	}

	return c, nil
}

// ensureBucket creates the bucket if it does not already exist.
func (c *Client) ensureBucket(ctx context.Context) error {
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err == nil {
		return nil
	}

	log.Printf("S3 bucket %q not found, creating...", c.bucket)

	_, err = c.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("create bucket %q: %w", c.bucket, err)
	}

	log.Printf("S3 bucket %q created", c.bucket)
	return nil
}

// Upload uploads data to the given key in the bucket.
// It returns the public URL of the uploaded object.
// Key format example: audio/{city_id}/{poi_id}/{story_id}.mp3
func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("s3: upload %q: %w", key, err)
	}

	url := fmt.Sprintf("%s/%s/%s", c.endpoint, c.bucket, key)
	return url, nil
}

// GetPresignedURL generates a presigned URL for downloading the object at the given key.
// The URL is valid for the specified expiry duration.
func (c *Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	result, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("s3: presign %q: %w", key, err)
	}

	return result.URL, nil
}

// Delete removes the object at the given key from the bucket.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3: delete %q: %w", key, err)
	}

	return nil
}

// Exists checks whether an object exists at the given key.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("s3: head %q: %w", key, err)
	}

	return true, nil
}

// isNotFound returns true if the error indicates the object was not found.
func isNotFound(err error) bool {
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "NotFound" || code == "NoSuchKey" || code == "404"
	}
	return false
}

// AudioKey returns the S3 key for an audio file following the convention:
// audio/{cityID}/{poiID}/{storyID}.mp3
func AudioKey(cityID, poiID, storyID int) string {
	return fmt.Sprintf("audio/%d/%d/%d.mp3", cityID, poiID, storyID)
}
