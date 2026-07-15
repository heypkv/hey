package deploy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFetchBytesAuth proves buddy's private-manifest path: a token becomes an
// Authorization: Bearer header, and no token means no header.
func TestFetchBytesAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if gotAuth != "Bearer s3cr3t" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	old := Client
	Client = srv.Client()
	defer func() { Client = old }()

	if _, err := FetchBytes(srv.URL, "s3cr3t"); err != nil {
		t.Fatalf("authenticated fetch failed: %v", err)
	}
	if gotAuth != "Bearer s3cr3t" {
		t.Fatalf("Authorization header = %q, want Bearer s3cr3t", gotAuth)
	}

	// No token → no header → the server 401s (proving we didn't leak one).
	if _, err := FetchBytes(srv.URL); err == nil {
		t.Fatal("unauthenticated fetch should have been rejected")
	}
}
