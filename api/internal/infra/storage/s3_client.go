package storage

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Compile-time check that S3Client satisfies StorageProvider (via main wiring).
// Uncomment if you import the providers package here: var _ providers.StorageProvider = (*S3Client)(nil)

type S3Client struct {
	client       *s3.Client
	presigClient *s3.PresignClient
	bucket       string
}

func NewS3Client(bucket, region, accessKey, secretKey string) *S3Client {
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	client := s3.NewFromConfig(cfg)

	return &S3Client{
		client:       client,
		presigClient: s3.NewPresignClient(client),
		bucket:       bucket,
	}
}

func (s *S3Client) GeneratePresignedPutURL(ctx context.Context, key string, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	resp, err := s.presigClient.PresignPutObject(ctx, input, s3.WithPresignExpires(10*time.Minute))
	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

func (s *S3Client) HeadObject(ctx context.Context, key string) error {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3Client) GetObject(ctx context.Context, key string, destPath string) error {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// PutObject uploads body to S3 at key. When size >= 0 it is sent as ContentLength.
func (s *S3Client) PutObject(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}
	_, err := s.client.PutObject(ctx, input)
	return err
}

// DeleteObject removes an object from S3. A missing object is not an error.
func (s *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// GetObjectBytes downloads key from S3 and returns the raw bytes.
func (s *S3Client) GetObjectBytes(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
