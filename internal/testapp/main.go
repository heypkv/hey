// testapp is the executable reference implementation of the hey app
// contract v0 (docs/app-contract-v0.md). Integration tests build it and run
// it through hey; it is never shipped.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	name    = "testapp"
	version = "0.0.1"
)

func main() {
	port := 0
	jsonOut := false
	args := os.Args[1:]
	// The first arg is the UI command name (e.g. "ui"); flags follow.
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--port":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &port)
				i++
			}
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen:", err)
		os.Exit(1)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hey %s %s\n", name, version)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/hey/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
	})

	if jsonOut {
		// Contract: exactly one flushed stdout line after the listener is
		// bound; os.Stdout is unbuffered in Go, so a plain write suffices.
		hs := map[string]any{
			"hey": 1, "name": name, "version": version,
			"url": url, "pid": os.Getpid(),
		}
		line, _ := json.Marshal(hs)
		fmt.Println(string(line))
	} else {
		fmt.Fprintln(os.Stderr, "listening at", url)
	}

	if err := http.Serve(ln, mux); err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		os.Exit(1)
	}
}
