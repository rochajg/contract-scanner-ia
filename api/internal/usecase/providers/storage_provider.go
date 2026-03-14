package providers

import (
	"context"
	"io"
)

type StorageProvider interface {
	// PutObject uploads content from body to S3 at key.
	// Pass size >= 0 when known; pass -1 to let the SDK buffer.
	PutObject(ctx context.Context, key string, body io.Reader, contentType string, size int64) error

	// GetObjectBytes downloads an S3 object and returns its full content.
	GetObjectBytes(ctx context.Context, key string) ([]byte, error)

	// GetObject downloads an S3 object to a local file path.
	GetObject(ctx context.Context, key string, destPath string) error

	// HeadObject checks whether an object exists in S3.
	HeadObject(ctx context.Context, key string) error

	// GeneratePresignedPutURL generates a short-lived pre-signed PUT URL (kept for future use).
	GeneratePresignedPutURL(ctx context.Context, key string, contentType string) (string, error)

	// DeleteObject removes an object from S3. Returns nil if the object does not exist.
	DeleteObject(ctx context.Context, key string) error
}
