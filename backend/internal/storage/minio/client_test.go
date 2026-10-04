package minio

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	minioclient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sbu2video/backend/internal/storage/blob"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	c := New(Config{Endpoint: "storage.example.test:9000", AccessKey: "test-access", SecretKey: "test-secret", Bucket: "media", UseSSL: true, PublicBase: "https://cdn.example.test/content/media"})
	var err error
	c.cli, err = minioclient.New(c.Cfg.Endpoint, &minioclient.Options{
		Secure: true, Region: "us-east-1", Creds: credentials.NewStaticV4("test-access", "test-secret", ""),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRefreshGetURL(t *testing.T) {
	c := testClient(t)
	key := "results/海 岸+100%#?.mp4"
	for _, base := range []string{"https://storage.example.test:9000/media", c.Cfg.PublicBase} {
		u, _ := url.Parse(base)
		u.Path += "/" + key
		u.RawQuery = "X-Amz-Date=20000101T000000Z&X-Amz-Signature=expired&response-content-type=text/plain"
		fresh, err := c.RefreshGetURL(context.Background(), u.String(), 15*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		got, err := url.Parse(fresh)
		if err != nil || got.Host != c.Cfg.Endpoint || got.Path != "/media/"+key {
			t.Fatal("refreshed URL changed the configured origin, bucket, or object key")
		}
		query := got.Query()
		if query.Get("X-Amz-Expires") != "900" || query.Get("X-Amz-Signature") == "" || query.Get("X-Amz-Signature") == "expired" || query.Get("X-Amz-Date") == "20000101T000000Z" || query.Has("response-content-type") {
			t.Fatal("old signature/query was retained or expiry was not refreshed")
		}
	}
	if _, err := c.RefreshGetURL(context.Background(), "https://storage.example.test:9000/other/video.mp4", time.Hour); !errors.Is(err, blob.ErrUntrustedObjectURL) {
		t.Fatal("another bucket must be rejected")
	}
}

func TestRefreshGetURLSigningFailure(t *testing.T) {
	c := testClient(t)
	stored := "https://storage.example.test:9000/media/result.mp4?X-Amz-Signature=expired"
	if got, err := c.RefreshGetURL(context.Background(), stored, 8*24*time.Hour); err == nil || got != "" {
		t.Fatal("invalid signing TTL must return no URL")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := c.RefreshGetURL(ctx, stored, time.Hour); !errors.Is(err, context.Canceled) || got != "" {
		t.Fatal("canceled signing must not return a URL")
	}
}

func TestRefreshDownloadURLSignsAttachmentWithoutChangingPreview(t *testing.T) {
	c := testClient(t)
	stored := c.ObjectURL("results/video.mp4") + "?response-content-disposition=inline&X-Amz-Signature=expired"
	fresh, err := c.RefreshDownloadURL(context.Background(), stored, "海边镜头.mp4", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(fresh)
	kind, params, err := mime.ParseMediaType(u.Query().Get("response-content-disposition"))
	if err != nil || kind != "attachment" || params["filename"] != "海边镜头.mp4" || u.Query().Get("X-Amz-Signature") == "" {
		t.Fatal("attachment filename must be part of a fresh signed URL")
	}
	preview, err := c.RefreshGetURL(context.Background(), fresh, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := url.Parse(preview)
	if p.Query().Has("response-content-disposition") || p.Query().Get("X-Amz-Signature") == u.Query().Get("X-Amz-Signature") {
		t.Fatal("download override must be signed and must not survive preview renewal")
	}
	if _, err := c.RefreshDownloadURL(context.Background(), "https://other.test/media/video.mp4", "video.mp4", time.Hour); !errors.Is(err, blob.ErrUntrustedObjectURL) {
		t.Fatal("download signer must retain the configured storage boundary")
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type expiringTestCredentials struct{ failed atomic.Bool }

var errTestSigning = errors.New("test credentials unavailable after upload")

func (p *expiringTestCredentials) Retrieve() (credentials.Value, error) {
	if p.failed.Load() {
		return credentials.Value{}, errTestSigning
	}
	return credentials.Value{AccessKeyID: "test-access", SecretAccessKey: "test-secret", SignerType: credentials.SignatureV4}, nil
}
func (p *expiringTestCredentials) RetrieveWithCredContext(*credentials.CredContext) (credentials.Value, error) {
	return p.Retrieve()
}
func (p *expiringTestCredentials) IsExpired() bool { return true }

func TestUploadSigningFailureDoesNotReturnUnsignedURL(t *testing.T) {
	for _, operation := range []string{"PutFile", "PullURL"} {
		t.Run(operation, func(t *testing.T) {
			c := testClient(t)
			provider := &expiringTestCredentials{}
			var writes atomic.Int32
			var err error
			c.cli, err = minioclient.New(c.Cfg.Endpoint, &minioclient.Options{
				Secure: true, Region: "us-east-1", Creds: credentials.New(provider),
				Transport: testTransport(func(r *http.Request) (*http.Response, error) {
					if r.Method != http.MethodPut {
						t.Fatal("unexpected storage request")
					}
					writes.Add(1)
					provider.failed.Store(true)
					return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Etag": []string{"test-etag"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
				}),
			})
			if err != nil {
				t.Fatal(err)
			}
			var got string
			if operation == "PutFile" {
				path := filepath.Join(t.TempDir(), "result.mp4")
				if err := os.WriteFile(path, []byte("test-media"), 0600); err != nil {
					t.Fatal(err)
				}
				got, err = c.PutFile(context.Background(), "results/test.mp4", path, "video/mp4")
			} else {
				source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("test-media")) }))
				defer source.Close()
				got, err = c.PullURL(context.Background(), source.URL, "results/test.mp4", "video/mp4")
			}
			if writes.Load() != 1 || !errors.Is(err, errTestSigning) || got != "" {
				t.Fatalf("successful upload followed by failed signing must return an error and no URL; writes=%d error=%v", writes.Load(), err)
			}
		})
	}
}
