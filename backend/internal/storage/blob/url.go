package blob

import (
	"errors"
	"mime"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrUntrustedObjectURL deliberately omits the supplied URL: its query may
// contain an expired signature or credentials that must not reach logs.
var ErrUntrustedObjectURL = errors.New("stored object URL does not match configured storage")

// AttachmentDisposition supplies the filename through a signed response
// override. Strip path separators and controls before encoding Unicode names.
func AttachmentDisposition(filename string) string {
	filename = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, filename)
	filename = strings.Trim(filename, " .")
	if filename == "" {
		filename = "download"
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": filename})
}

// TrustedObjectKey accepts only URLs beneath an explicitly configured bucket
// base. It never fetches a URL, normalizes a path, or retains query parameters.
// Callers must perform their user/task/asset ownership checks before signing.
func TrustedObjectKey(storedURL string, bucketBases ...string) (string, error) {
	u, ok := parseObjectURL(storedURL)
	if !ok || !safeObjectPath(u, false) {
		return "", ErrUntrustedObjectURL
	}
	for _, rawBase := range bucketBases {
		base, valid := parseObjectURL(rawBase)
		if !valid || base.RawQuery != "" || base.ForceQuery || !safeObjectPath(base, true) {
			continue
		}
		if u.Scheme != base.Scheme || !strings.EqualFold(u.Host, base.Host) {
			continue
		}
		prefix := strings.TrimSuffix(base.Path, "/") + "/"
		if strings.HasPrefix(u.Path, prefix) {
			key := strings.TrimPrefix(u.Path, prefix)
			if key != "" {
				return key, nil
			}
		}
	}
	return "", ErrUntrustedObjectURL
}

func parseObjectURL(raw string) (*url.URL, bool) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.Contains(raw, "#") || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Opaque != "" {
		return nil, false
	}
	return u, true
}

func safeObjectPath(u *url.URL, base bool) bool {
	if !utf8.ValidString(u.Path) || strings.Contains(u.Path, "\\") || strings.IndexFunc(u.Path, unicode.IsControl) >= 0 {
		return false
	}
	escaped := strings.ToLower(u.EscapedPath())
	if strings.Contains(escaped, "%2f") || strings.Contains(escaped, "%5c") {
		return false // Encoded separators must not change the bucket boundary.
	}
	for i := 0; i+2 < len(u.Path); i++ {
		if u.Path[i] == '%' && isHex(u.Path[i+1]) && isHex(u.Path[i+2]) {
			return false // Reject paths whose meaning changes on a second decode.
		}
	}
	path := u.Path
	if base {
		path = strings.TrimSuffix(path, "/")
		if path == "" {
			return true
		}
	}
	if !strings.HasPrefix(path, "/") {
		return false
	}
	for _, segment := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func isHex(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F'
}
