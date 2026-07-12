package contract

import (
	"strings"
	"testing"
)

func TestScanHandshakeHappyPath(t *testing.T) {
	data := []byte(`{"hey":1,"name":"djin","version":"0.1.0","url":"http://127.0.0.1:52341","pid":4242,"port":52341}` + "\n")
	h, err := ScanHandshake(data)
	if err != nil {
		t.Fatal(err)
	}
	if h.Name != "djin" || h.URL != "http://127.0.0.1:52341" || h.Port != 52341 {
		t.Errorf("bad handshake: %+v", h)
	}
}

func TestScanHandshakeToleratesJunkBefore(t *testing.T) {
	data := []byte("starting up...\nsome log line\n{\"other\":true}\n" +
		`{"hey":1,"name":"x","url":"http://localhost:8080"}` + "\n")
	h, err := ScanHandshake(data)
	if err != nil {
		t.Fatal(err)
	}
	if h.Name != "x" {
		t.Errorf("got %+v", h)
	}
}

func TestScanHandshakeIgnoresWrongVersion(t *testing.T) {
	data := []byte(`{"hey":2,"name":"x","url":"http://127.0.0.1:1"}` + "\n")
	if _, err := ScanHandshake(data); err != ErrNoHandshake {
		t.Errorf("hey:2 line should be skipped (forward compat), got %v", err)
	}
}

func TestScanHandshakeRejectsNonLoopback(t *testing.T) {
	for _, url := range []string{
		"http://0.0.0.0:8080",
		"http://192.168.1.5:8080",
		"https://evil.example.com",
		"http://example.com:8080",
	} {
		data := []byte(`{"hey":1,"name":"x","url":"` + url + `"}` + "\n")
		if _, err := ScanHandshake(data); err == nil || !strings.Contains(err.Error(), "loopback") {
			t.Errorf("url %s should be rejected, got %v", url, err)
		}
	}
}

func TestScanHandshakeRejectsMissingFields(t *testing.T) {
	data := []byte(`{"hey":1,"url":"http://127.0.0.1:1"}` + "\n")
	if _, err := ScanHandshake(data); err == nil {
		t.Error("missing name should be rejected")
	}
}

func TestScanHandshakeNoise(t *testing.T) {
	if _, err := ScanHandshake([]byte("just logs\nmore logs\n")); err != ErrNoHandshake {
		t.Errorf("want ErrNoHandshake, got %v", err)
	}
	if _, err := ScanHandshake(nil); err != ErrNoHandshake {
		t.Errorf("want ErrNoHandshake on empty, got %v", err)
	}
}
