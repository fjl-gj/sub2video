package blob

import (
	"errors"
	"mime"
	"net/url"
	"strings"
	"testing"
)

func TestAttachmentDispositionSupportsUnicodeWithoutHeaderOrPathInjection(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"海边镜头.mp4", "海边镜头.mp4"},
		{"scene.mp4", "scene.mp4"},
		{"../private\\movie\r\n.mp4", "-private-movie--.mp4"},
		{" . ", "download"},
	} {
		header := AttachmentDisposition(tc.name)
		kind, params, err := mime.ParseMediaType(header)
		if err != nil || kind != "attachment" || params["filename"] != tc.want || strings.ContainsAny(header, "\r\n") {
			t.Errorf("invalid attachment filename for %q: %q err=%v", tc.name, header, err)
		}
	}
}

func TestTrustedObjectKey(t *testing.T) {
	base := "https://storage.example.test:9000/media"
	key := "results/海 岸+100%#?.mp4"
	special := (&url.URL{Scheme: "https", Host: "storage.example.test:9000", Path: "/media/" + key, RawQuery: "old-signature=discard-me"}).String()
	tests := []struct{ name, stored, base, want string }{
		{"expired signature", base + "/results/video.mp4?X-Amz-Date=20000101T000000Z&X-Amz-Signature=expired", base, "results/video.mp4"},
		{"special characters", special, base, key},
		{"host case", "https://STORAGE.example.test:9000/media/video.mp4", base, "video.mp4"},
		{"public prefix", "https://cdn.example.test/videos/media/video.mp4?old=1", "https://cdn.example.test/videos/media/", "video.mp4"},
		{"virtual bucket host", "https://media.example.test/video.mp4?old=1", "https://media.example.test", "video.mp4"},
		{"different host", "https://other.example.test:9000/media/video.mp4", base, ""},
		{"host suffix", "https://storage.example.test.evil.test:9000/media/video.mp4", base, ""},
		{"different port", "https://storage.example.test:9001/media/video.mp4", base, ""},
		{"different scheme", "http://storage.example.test:9000/media/video.mp4", base, ""},
		{"userinfo", "https://someone@storage.example.test:9000/media/video.mp4", base, ""},
		{"other bucket", "https://storage.example.test:9000/private/video.mp4", base, ""},
		{"bucket prefix", "https://storage.example.test:9000/media-private/video.mp4", base, ""},
		{"missing object", base + "/", base, ""},
		{"dot segment", base + "/results/../private.mp4", base, ""},
		{"encoded dot segment", base + "/%2e%2e/private.mp4", base, ""},
		{"encoded separator", base + "%2fvideo.mp4", base, ""},
		{"encoded backslash", base + "/results%5cprivate.mp4", base, ""},
		{"double encoding", base + "/%252e%252e/private.mp4", base, ""},
		{"repeated slash", base + "//video.mp4", base, ""},
		{"trailing slash", base + "/video.mp4/", base, ""},
		{"fragment", base + "/video.mp4#extra", base, ""},
		{"empty fragment", base + "/video.mp4#", base, ""},
		{"control", base + "/video%00.mp4", base, ""},
		{"invalid escape", base + "/video%xx.mp4", base, ""},
		{"relative URL", "/media/video.mp4", base, ""},
		{"leading whitespace", " " + base + "/video.mp4", base, ""},
		{"invalid base userinfo", base + "/video.mp4", "https://user@storage.example.test:9000/media", ""},
		{"invalid base query", base + "/video.mp4", base + "?bucket=private", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TrustedObjectKey(tt.stored, tt.base)
			if tt.want == "" {
				if !errors.Is(err, ErrUntrustedObjectURL) || got != "" {
					t.Fatal("untrusted URL must not yield an object key")
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("object key mismatch: got %q, error %v", got, err)
			}
		})
	}
}
