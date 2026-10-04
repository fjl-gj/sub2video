package cos

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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/storage/blob"
	cossdk "github.com/tencentyun/cos-go-sdk-v5"
)

func testClient() *Client {
	return New(Config{SecretID: "test-access", SecretKey: "test-secret", Bucket: "media-123456", Region: "ap-test"})
}

func TestRefreshGetURL(t *testing.T) {
	c := testClient()
	key := "results/海 岸+100%#?.mp4"
	u, _ := url.Parse(c.PublicBaseURL())
	u.Path = "/" + key
	u.RawQuery = "q-sign-time=1;2&q-signature=expired&response-content-type=text/plain"
	fresh, err := c.RefreshGetURL(context.Background(), u.String(), 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got, err := url.Parse(fresh)
	if err != nil || got.Host != u.Host || got.Path != u.Path {
		t.Fatal("refreshed URL changed the bucket or object key")
	}
	query := got.Query()
	times := strings.Split(query.Get("q-sign-time"), ";")
	if len(times) != 2 || query.Get("q-signature") == "" || query.Get("q-signature") == "expired" || query.Has("response-content-type") {
		t.Fatal("old signature/query was retained")
	}
	start, _ := strconv.ParseInt(times[0], 10, 64)
	end, _ := strconv.ParseInt(times[1], 10, 64)
	if end-start != 900 || end <= time.Now().Unix() {
		t.Fatal("signature expiry was not refreshed")
	}
	if _, err := c.RefreshGetURL(context.Background(), "https://other-123456.cos.ap-test.myqcloud.com/results/video.mp4", time.Hour); !errors.Is(err, blob.ErrUntrustedObjectURL) {
		t.Fatal("another bucket must be rejected")
	}
}

func TestRefreshGetURLSigningFailure(t *testing.T) {
	c := testClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := c.RefreshGetURL(ctx, c.ObjectURL("result.mp4"), time.Hour); !errors.Is(err, context.Canceled) || got != "" {
		t.Fatal("canceled signing must not return a URL")
	}
}

func TestRefreshDownloadURLSignsAttachmentWithoutChangingPreview(t *testing.T) {
	c := testClient()
	stored := c.ObjectURL("results/video.mp4") + "?response-content-disposition=inline&q-signature=expired"
	fresh, err := c.RefreshDownloadURL(context.Background(), stored, "海边镜头.mp4", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(fresh)
	kind, params, err := mime.ParseMediaType(u.Query().Get("response-content-disposition"))
	if err != nil || kind != "attachment" || params["filename"] != "海边镜头.mp4" || !strings.Contains(u.Query().Get("q-url-param-list"), "response-content-disposition") || u.Query().Get("q-signature") == "" {
		t.Fatal("attachment override must be included in COS signed query parameters")
	}
	preview, err := c.RefreshGetURL(context.Background(), fresh, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := url.Parse(preview)
	if p.Query().Has("response-content-disposition") || p.Query().Get("q-signature") == u.Query().Get("q-signature") {
		t.Fatal("download override must be signed and must not survive preview renewal")
	}
	if _, err := c.RefreshDownloadURL(context.Background(), "https://other.test/video.mp4", "video.mp4", time.Hour); !errors.Is(err, blob.ErrUntrustedObjectURL) {
		t.Fatal("download signer must retain the configured storage boundary")
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUploadSigningFailureDoesNotReturnUnsignedURL(t *testing.T) {
	for _, operation := range []string{"PutFile", "PullURL"} {
		t.Run(operation, func(t *testing.T) {
			c := testClient()
			base, _ := url.Parse(c.PublicBaseURL())
			writes := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c.cli = cossdk.NewClient(&cossdk.BaseURL{BucketURL: base}, &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
				writes++
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{}, Body: &cancelOnClose{Reader: strings.NewReader(""), cancel: cancel}, Request: r}, nil
			})})
			var got string
			var err error
			if operation == "PutFile" {
				path := filepath.Join(t.TempDir(), "result.mp4")
				if err := os.WriteFile(path, []byte("test-media"), 0600); err != nil {
					t.Fatal(err)
				}
				got, err = c.PutFile(ctx, "results/test.mp4", path, "video/mp4")
			} else {
				source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("test-media")) }))
				defer source.Close()
				got, err = c.PullURL(ctx, source.URL, "results/test.mp4", "video/mp4")
			}
			if writes != 1 || !errors.Is(err, context.Canceled) || got != "" {
				t.Fatalf("successful upload followed by canceled signing must return no URL; writes=%d error=%v", writes, err)
			}
		})
	}
}

type cancelOnClose struct {
	io.Reader
	cancel context.CancelFunc
}

func (r *cancelOnClose) Close() error { r.cancel(); return nil }
