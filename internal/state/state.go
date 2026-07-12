// Package state tracks long-running UI apps hey has started, in
// <heyHome>/state/procs.json. Writes go through temp-file + rename.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const procsFname = "procs.json"

// Proc is one running (or believed-running) UI app.
type Proc struct {
	App     string    `json:"app"`
	Version string    `json:"version,omitempty"`
	PID     int       `json:"pid"`
	Port    int       `json:"port,omitempty"`
	URL     string    `json:"url"`
	Started time.Time `json:"started"`
	Log     string    `json:"log,omitempty"`
}

// Load reads all tracked procs (empty slice if the file doesn't exist).
func Load(stateDir string) ([]Proc, error) {
	data, err := os.ReadFile(filepath.Join(stateDir, procsFname))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var procs []Proc
	if err := json.Unmarshal(data, &procs); err != nil {
		return nil, fmt.Errorf("parse %s: %w", procsFname, err)
	}
	return procs, nil
}

// Save writes the full proc list atomically.
func Save(stateDir string, procs []Proc) error {
	if procs == nil {
		procs = []Proc{}
	}
	data, err := json.MarshalIndent(procs, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(stateDir, procsFname)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get returns the tracked proc for app, if any.
func Get(stateDir, app string) (Proc, bool, error) {
	procs, err := Load(stateDir)
	if err != nil {
		return Proc{}, false, err
	}
	for _, p := range procs {
		if p.App == app {
			return p, true, nil
		}
	}
	return Proc{}, false, nil
}

// Put upserts a proc entry keyed by app name.
func Put(stateDir string, proc Proc) error {
	procs, err := Load(stateDir)
	if err != nil {
		return err
	}
	out := procs[:0]
	for _, p := range procs {
		if p.App != proc.App {
			out = append(out, p)
		}
	}
	out = append(out, proc)
	return Save(stateDir, out)
}

// Remove deletes the entry for app (no-op if absent).
func Remove(stateDir, app string) error {
	procs, err := Load(stateDir)
	if err != nil {
		return err
	}
	out := procs[:0]
	for _, p := range procs {
		if p.App != app {
			out = append(out, p)
		}
	}
	return Save(stateDir, out)
}

// Prune keeps only procs for which alive returns true and saves the result.
// It returns the surviving procs.
func Prune(stateDir string, alive func(Proc) bool) ([]Proc, error) {
	procs, err := Load(stateDir)
	if err != nil {
		return nil, err
	}
	out := procs[:0]
	for _, p := range procs {
		if alive(p) {
			out = append(out, p)
		}
	}
	if err := Save(stateDir, out); err != nil {
		return nil, err
	}
	return out, nil
}
