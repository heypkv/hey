// Package fetch downloads release artifacts over HTTPS, enforces SHA-256
// checksums from a goreleaser checksums file, and extracts the single
// expected binary from zip/tar.gz archives.
package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	maxChecksumsBytes = 1 << 20   // 1 MB
	maxArchiveBytes   = 512 << 20 // 512 MB
)

// SigSpec is the seam for future signature verification (e.g. a minisign
// public key from the registry). It is always nil today; Verify documents
// where signature checks will plug in.
type SigSpec struct{}

// Client is the HTTP client used for all downloads. It is a package var (not a
// const literal) so tests can point it at an httptest TLS server, whose
// self-signed certificate the default client would reject; production keeps the
// long timeout for large artifacts.
var Client = &http.Client{Timeout: 10 * time.Minute}

// authHeader picks the first non-empty token from a variadic auth argument, so
// existing callers pass nothing and buddy can pass a keeper-resolved token.
func authHeader(auth []string) string {
	for _, a := range auth {
		if a != "" {
			return a
		}
	}
	return ""
}

func get(url string, cap int64, token string) (io.ReadCloser, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("refusing non-https download: %s", url)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	if token != "" {
		// Bearer for private GitHub release assets. Go's client drops this
		// header on the cross-host redirect to the signed CDN URL.
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(resp.Body, cap), resp.Body}, nil
}

// Checksums fetches and returns the raw checksums file. An optional auth token
// authenticates private-artifact fetches (buddy).
func Checksums(url string, auth ...string) ([]byte, error) {
	body, err := get(url, maxChecksumsBytes, authHeader(auth))
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

var checksumLine = regexp.MustCompile(`^([0-9a-fA-F]{64})\s+\*?(\S+)$`)

// ParseChecksums parses goreleaser "sha256  filename" lines.
func ParseChecksums(data []byte) map[string]string {
	sums := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if m := checksumLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			sums[m[2]] = strings.ToLower(m[1])
		}
	}
	return sums
}

// Download streams url into a temp file inside dir (same volume as the final
// destination so renames stay atomic) and returns the temp path and SHA-256.
// The caller owns the temp file and must remove it on failure.
func Download(url, dir string, auth ...string) (path, sha string, err error) {
	body, err := get(url, maxArchiveBytes, authHeader(auth))
	if err != nil {
		return "", "", err
	}
	defer body.Close()

	tmp, err := os.CreateTemp(dir, "hey-download-*")
	if err != nil {
		return "", "", err
	}
	h := sha256.New()
	_, err = io.Copy(tmp, io.TeeReader(body, h))
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", "", fmt.Errorf("download %s: %w", url, err)
	}
	return tmp.Name(), hex.EncodeToString(h.Sum(nil)), nil
}

// Verify enforces that gotSHA matches the checksums entry for asset. sig is
// the future signature-verification seam and must be nil today.
func Verify(asset, gotSHA string, checksums map[string]string, sig *SigSpec) error {
	if sig != nil {
		return fmt.Errorf("signature verification not implemented yet")
	}
	want, ok := checksums[asset]
	if !ok {
		return fmt.Errorf("checksums file has no entry for %s", asset)
	}
	if !strings.EqualFold(want, gotSHA) {
		return fmt.Errorf("checksum mismatch for %s:\n  want %s\n  got  %s", asset, want, gotSHA)
	}
	return nil
}
