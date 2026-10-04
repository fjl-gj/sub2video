package blob

import (
	"context"
	"time"
)

// Store is the object storage used for task source images and result videos.
type Store interface {
	Configured() bool
	Provider() string
	PutObject(ctx context.Context, objectKey string, body []byte, contentType string) error
	PutFile(ctx context.Context, objectKey, filePath, contentType string) (publicURL string, err error)
	ObjectURL(objectKey string) string
	SignGetURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	RefreshGetURL(ctx context.Context, storedURL string, ttl time.Duration) (string, error)
	RefreshDownloadURL(ctx context.Context, storedURL, filename string, ttl time.Duration) (string, error)
	PullURL(ctx context.Context, remoteURL, objectKey, contentType string) (string, error)
	EnsureBucket(ctx context.Context) error
}
