package fetch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDownloadAuth proves buddy's private-artifact path: Download sends the
// token as a Bearer header, and omits the header entirely without one.
func TestDownloadAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if gotAuth != "Bearer tok123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, "payload")
	}))
	defer srv.Close()

	old := Client
	Client = srv.Client()
	defer func() { Client = old }()

	if _, _, err := Download(srv.URL, t.TempDir(), "tok123"); err != nil {
		t.Fatalf("authenticated download failed: %v", err)
	}
	if gotAuth != "Bearer tok123" {
		t.Fatalf("Authorization header = %q, want Bearer tok123", gotAuth)
	}
	if _, _, err := Download(srv.URL, t.TempDir()); err == nil {
		t.Fatal("unauthenticated download should have been rejected")
	}
}
