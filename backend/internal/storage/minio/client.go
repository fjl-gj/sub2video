package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sbu2video/backend/internal/storage/blob"
)

type Config struct {
	Endpoint   string // host:port or full URL
	AccessKey  string
	SecretKey  string
	Bucket     string
	UseSSL     bool
	PublicBase string // optional override, e.g. http://192.168.0.103:9000/sub2video
}

type Client struct {
	Cfg Config
	cli *minio.Client
}

func New(cfg Config) *Client {
	c := &Client{Cfg: normalize(cfg)}
	if !c.Configured() {
		return c
	}
	cli, err := minio.New(c.Cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.Cfg.AccessKey, c.Cfg.SecretKey, ""),
		Secure: c.Cfg.UseSSL,
	})
	if err != nil {
		return c
	}
	c.cli = cli
	return c
}

func normalize(cfg Config) Config {
	ep := strings.TrimSpace(cfg.Endpoint)
	ep = strings.TrimPrefix(ep, "http://")
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimRight(ep, "/")
	cfg.Endpoint = ep
	cfg.AccessKey = strings.TrimSpace(cfg.AccessKey)
	cfg.SecretKey = strings.TrimSpace(cfg.SecretKey)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.PublicBase = strings.TrimRight(strings.TrimSpace(cfg.PublicBase), "/")
	return cfg
}

func (c *Client) Configured() bool {
	return c != nil && c.Cfg.Endpoint != "" && c.Cfg.AccessKey != "" && c.Cfg.SecretKey != "" && c.Cfg.Bucket != ""
}

func (c *Client) Provider() string { return "minio" }

func (c *Client) EnsureBucket(ctx context.Context) error {
	if !c.Configured() || c.cli == nil {
		return fmt.Errorf("minio not configured")
	}
	exists, err := c.cli.BucketExists(ctx, c.Cfg.Bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.cli.MakeBucket(ctx, c.Cfg.Bucket, minio.MakeBucketOptions{})
}

func (c *Client) ObjectURL(objectKey string) string {
	key := strings.TrimLeft(objectKey, "/")
	if c.Cfg.PublicBase != "" {
		return c.Cfg.PublicBase + "/" + key
	}
	scheme := "http"
	if c.Cfg.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, c.Cfg.Endpoint, c.Cfg.Bucket, key)
}

func (c *Client) PutObject(ctx context.Context, objectKey string, body []byte, contentType string) error {
	if !c.Configured() || c.cli == nil {
		return fmt.Errorf("minio not configured")
	}
	key := strings.TrimLeft(objectKey, "/")
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	_, err := c.cli.PutObject(ctx, c.Cfg.Bucket, key, bytes.NewReader(body), int64(len(body)), opts)
	return err
}

func (c *Client) PutFile(ctx context.Context, objectKey, filePath, contentType string) (string, error) {
	body, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := c.PutObject(ctx, objectKey, body, contentType); err != nil {
		return "", err
	}
	return c.SignGetURL(ctx, objectKey, 24*time.Hour)
}

func (c *Client) SignGetURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	return c.signGetURL(ctx, objectKey, ttl, url.Values{})
}

func (c *Client) signGetURL(ctx context.Context, objectKey string, ttl time.Duration, params url.Values) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !c.Configured() || c.cli == nil {
		return "", fmt.Errorf("minio not configured")
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	u, err := c.cli.PresignedGetObject(ctx, c.Cfg.Bucket, strings.TrimLeft(objectKey, "/"), ttl, params)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) RefreshGetURL(ctx context.Context, storedURL string, ttl time.Duration) (string, error) {
	return c.refreshGetURL(ctx, storedURL, ttl, url.Values{})
}

func (c *Client) RefreshDownloadURL(ctx context.Context, storedURL, filename string, ttl time.Duration) (string, error) {
	return c.refreshGetURL(ctx, storedURL, ttl, url.Values{"response-content-disposition": {blob.AttachmentDisposition(filename)}})
}

func (c *Client) refreshGetURL(ctx context.Context, storedURL string, ttl time.Duration, params url.Values) (string, error) {
	if !c.Configured() || c.cli == nil {
		return "", fmt.Errorf("minio not configured")
	}
	scheme := "http"
	if c.Cfg.UseSSL {
		scheme = "https"
	}
	endpointBase := (&url.URL{Scheme: scheme, Host: c.Cfg.Endpoint, Path: "/" + c.Cfg.Bucket}).String()
	key, err := blob.TrustedObjectKey(storedURL, endpointBase, c.Cfg.PublicBase)
	if err != nil {
		return "", err
	}
	return c.signGetURL(ctx, key, ttl, params)
}

func (c *Client) PullURL(ctx context.Context, remoteURL, objectKey, contentType string) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("minio not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("download status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 200<<20))
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = res.Header.Get("Content-Type")
	}
	if err := c.PutObject(ctx, objectKey, body, contentType); err != nil {
		return "", err
	}
	// Result and source buckets are private by default. Return a presigned URL
	// after persisting the object so callers never expose an unsigned URL that
	// MinIO will reject with 403.
	return c.SignGetURL(ctx, objectKey, 24*time.Hour)
}

var _ blob.Store = (*Client)(nil)
