// Package source parses the hey.source.v1 manifest — a small descriptor a tool
// checks into its own repo (as hey.json) so buddy can install it from source.
// The manifest names a prebuilt native executable per platform (checked into
// the repo) and, optionally, the build semantics to produce one when a
// toolchain is present. hey never learns the tool's internals; it only reads
// this manifest and moves (or, later, builds) a native binary.
package source

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

// SchemaID is the manifest kind this package understands.
const SchemaID = "hey.source.v1"

// Manifest is a repo's hey.json.
type Manifest struct {
	HeyManifest string              `json:"hey_manifest"`
	ID          string              `json:"id"`
	Version     string              `json:"version"`
	Prebuilt    map[string]Prebuilt `json:"prebuilt"`
	Build       *Build              `json:"build,omitempty"`
	Launch      Launch              `json:"launch"`
}

// Prebuilt points at a native executable checked into the repo, keyed in the
// manifest by "<os>/<arch>" (e.g. "windows/amd64").
type Prebuilt struct {
	Path   string `json:"path"`             // repo-relative, e.g. bin/boss.exe
	SHA256 string `json:"sha256,omitempty"` // optional integrity check
}

// Build carries the semantics to build the tool after a clone when the
// toolchain is present. Declared in v1; buddy's build execution ships later.
type Build struct {
	Toolchain  string `json:"toolchain"`             // go | javac | ...
	MinVersion string `json:"min_version,omitempty"` // e.g. 1.26
	Command    string `json:"command"`               // e.g. go build -o {out} ./cli/boss
	Out        string `json:"out"`                   // e.g. bin/boss{exe}
}

// Launch describes how to run the installed executable.
type Launch struct {
	Args []string `json:"args,omitempty"`
}

// Platform is the running platform's manifest key, "<os>/<arch>".
func Platform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// Parse validates a hey.source.v1 manifest.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse source manifest: %w", err)
	}
	if m.HeyManifest != SchemaID {
		return nil, fmt.Errorf("not a %s manifest (got hey_manifest=%q)", SchemaID, m.HeyManifest)
	}
	if m.ID == "" || m.Version == "" {
		return nil, fmt.Errorf("source manifest needs id and version")
	}
	if strings.ContainsAny(m.ID, `/\@ `) {
		return nil, fmt.Errorf("invalid id %q", m.ID)
	}
	if len(m.Prebuilt) == 0 && m.Build == nil {
		return nil, fmt.Errorf("source manifest %q offers neither a prebuilt binary nor build semantics", m.ID)
	}
	for plat, p := range m.Prebuilt {
		if p.Path == "" {
			return nil, fmt.Errorf("prebuilt %q has no path", plat)
		}
	}
	return &m, nil
}

// PrebuiltFor returns the prebuilt entry for the running platform, if any.
func (m *Manifest) PrebuiltFor() (Prebuilt, bool) {
	p, ok := m.Prebuilt[Platform()]
	return p, ok
}

// Platforms lists the platform keys the manifest offers a prebuilt for.
func (m *Manifest) Platforms() []string {
	out := make([]string, 0, len(m.Prebuilt))
	for k := range m.Prebuilt {
		out = append(out, k)
	}
	return out
}
