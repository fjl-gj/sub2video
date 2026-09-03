package cos

import (
	"context"
	"fmt"
	"time"
)

// Config holds Tencent COS settings. Leave empty until credentials are provided.
type Config struct {
	SecretID   string
	SecretKey  string
	Bucket     string
	Region     string
	UseIntranet bool
}

// Client is a placeholder until COS credentials are wired.
// Upload/SignURL will return clear errors when not configured.
type Client struct {
	Cfg Config
}

func New(cfg Config) *Client {
	return &Client{Cfg: cfg}
}

func (c *Client) Configured() bool {
	return c != nil && c.Cfg.SecretID != "" && c.Cfg.SecretKey != "" && c.Cfg.Bucket != "" && c.Cfg.Region != ""
}

func (c *Client) PublicBaseURL() string {
	if !c.Configured() {
		return ""
	}
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com", c.Cfg.Bucket, c.Cfg.Region)
}

// SignGetURL returns a temporary public URL. Not implemented until SDK is added.
func (c *Client) SignGetURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	_ = ctx
	_ = ttl
	if !c.Configured() {
		return "", fmt.Errorf("cos not configured")
	}
	return "", fmt.Errorf("cos sign url not implemented yet for key=%s", objectKey)
}

// PutObject uploads bytes. Not implemented until SDK is added.
func (c *Client) PutObject(ctx context.Context, objectKey string, body []byte, contentType string) error {
	_ = ctx
	_ = body
	_ = contentType
	if !c.Configured() {
		return fmt.Errorf("cos not configured")
	}
	return fmt.Errorf("cos put not implemented yet for key=%s", objectKey)
}
