// Package contract implements hey's side of the app contract v0
// (docs/app-contract-v0.md): reading the startup handshake an app prints to
// stdout (captured in a log file), health checking, and graceful shutdown.
package contract

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Version is the contract version this hey speaks (HEY_CONTRACT env).
const Version = 0

const (
	maxLineBytes = 4096
	maxScanBytes = 64 << 10 // give up after 64 KB of pre-handshake noise
)

// Handshake is the single JSON line an app prints once its listener is bound.
type Handshake struct {
	Hey     int    `json:"hey"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	URL     string `json:"url"`
	PID     int    `json:"pid,omitempty"`
	Port    int    `json:"port,omitempty"`
}

// ErrNoHandshake is returned when the scan budget is exhausted.
var ErrNoHandshake = fmt.Errorf("no handshake line found")

// ScanHandshake scans data (the app's captured stdout so far) for the
// handshake line. Junk lines before it are tolerated up to maxScanBytes.
func ScanHandshake(data []byte) (Handshake, error) {
	if len(data) > maxScanBytes {
		data = data[:maxScanBytes]
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, maxLineBytes+1), maxLineBytes+1)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var h Handshake
		if err := json.Unmarshal(line, &h); err != nil {
			continue
		}
		if h.Hey != 1 {
			continue
		}
		if err := validate(h); err != nil {
			return Handshake{}, err
		}
		return h, nil
	}
	return Handshake{}, ErrNoHandshake
}

func validate(h Handshake) error {
	if h.Name == "" || h.URL == "" {
		return fmt.Errorf("handshake missing name or url: %+v", h)
	}
	u, err := url.Parse(h.URL)
	if err != nil {
		return fmt.Errorf("handshake url invalid: %w", err)
	}
	if u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") {
		return fmt.Errorf("handshake url must be loopback http, got %s", h.URL)
	}
	return nil
}

// WaitHandshakeFromLog tails logPath until a valid handshake appears, the
// process dies (checked via alive), or timeout elapses. Returns the log tail
// alongside the error for diagnostics.
func WaitHandshakeFromLog(logPath string, timeout time.Duration, alive func() bool) (Handshake, string, error) {
	deadline := time.Now().Add(timeout)
	for {
		data, _ := os.ReadFile(logPath)
		h, err := ScanHandshake(data)
		if err == nil {
			return h, "", nil
		}
		if err != ErrNoHandshake {
			return Handshake{}, tail(data), err
		}
		if !alive() {
			return Handshake{}, tail(data), fmt.Errorf("app exited before completing the handshake")
		}
		if time.Now().After(deadline) {
			return Handshake{}, tail(data), fmt.Errorf("timed out after %s waiting for handshake", timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func tail(data []byte) string {
	const n = 2048
	if len(data) > n {
		data = data[len(data)-n:]
	}
	return strings.TrimSpace(string(data))
}

// WaitHealthy polls GET url/healthz until 200 or timeout.
func WaitHealthy(baseURL string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for {
		resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("app did not become healthy within %s", timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Healthy is a single non-blocking-ish health probe.
func Healthy(baseURL string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/healthz")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Shutdown asks the app to exit via POST /hey/shutdown. Returns nil if the
// app acknowledged (200); callers still wait for process exit and escalate.
func Shutdown(baseURL string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(strings.TrimRight(baseURL, "/")+"/hey/shutdown", "", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("shutdown endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}
