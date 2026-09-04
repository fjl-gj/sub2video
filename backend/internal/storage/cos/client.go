package cos

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

	"github.com/tencentyun/cos-go-sdk-v5"
)

type Config struct {
	SecretID    string
	SecretKey   string
	Bucket      string
	Region      string
	UseIntranet bool
}

type Client struct {
	Cfg Config
	cli *cos.Client
}

func New(cfg Config) *Client {
	c := &Client{Cfg: cfg}
	if !c.Configured() {
		return c
	}
	base := c.PublicBaseURL()
	if cfg.UseIntranet {
		base = fmt.Sprintf("https://%s.cos-internal.%s.myqcloud.com", cfg.Bucket, cfg.Region)
	}
	u, err := url.Parse(base)
	if err != nil {
		return c
	}
	b := &cos.BaseURL{BucketURL: u}
	c.cli = cos.NewClient(b, &http.Client{
		Timeout: 120 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})
	return c
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

func (c *Client) ObjectURL(objectKey string) string {
	return strings.TrimRight(c.PublicBaseURL(), "/") + "/" + strings.TrimLeft(objectKey, "/")
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
	// Prefer long-lived signed URL so private buckets still work for AutoDL fetch.
	signed, err := c.SignGetURL(ctx, objectKey, 24*time.Hour)
	if err == nil && signed != "" {
		return signed, nil
	}
	return c.ObjectURL(objectKey), nil
}

func (c *Client) PutObject(ctx context.Context, objectKey string, body []byte, contentType string) error {
	if !c.Configured() || c.cli == nil {
		return fmt.Errorf("cos not configured")
	}
	opt := &cos.ObjectPutOptions{}
	if contentType != "" {
		opt.ObjectPutHeaderOptions = &cos.ObjectPutHeaderOptions{ContentType: contentType}
	}
	_, err := c.cli.Object.Put(ctx, strings.TrimLeft(objectKey, "/"), bytes.NewReader(body), opt)
	return err
}

func (c *Client) SignGetURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	_ = ctx
	if !c.Configured() || c.cli == nil {
		return "", fmt.Errorf("cos not configured")
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	u, err := c.cli.Object.GetPresignedURL(ctx, http.MethodGet, strings.TrimLeft(objectKey, "/"), c.Cfg.SecretID, c.Cfg.SecretKey, ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PullURL downloads remoteURL and stores under objectKey; returns public object URL.
func (c *Client) PullURL(ctx context.Context, remoteURL, objectKey, contentType string) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("cos not configured")
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
	return c.ObjectURL(objectKey), nil
}
