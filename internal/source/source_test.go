package source

import "testing"

func TestParseValid(t *testing.T) {
	js := `{
	  "hey_manifest": "hey.source.v1",
	  "id": "boss",
	  "version": "0.4.0",
	  "prebuilt": {
	    "windows/amd64": {"path": "bin/boss.exe", "sha256": "abc"},
	    "darwin/arm64":  {"path": "bin/boss-darwin-arm64"}
	  },
	  "build": {"toolchain": "go", "command": "go build -o {out} ./cli/boss", "out": "bin/boss{exe}"}
	}`
	m, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.ID != "boss" || m.Version != "0.4.0" {
		t.Fatalf("bad manifest: %+v", m)
	}
	if p, ok := m.Prebuilt["windows/amd64"]; !ok || p.Path != "bin/boss.exe" || p.SHA256 != "abc" {
		t.Fatalf("windows prebuilt wrong: %+v", m.Prebuilt)
	}
	if m.Build == nil || m.Build.Toolchain != "go" {
		t.Fatalf("build semantics not parsed: %+v", m.Build)
	}
}

func TestParseRejects(t *testing.T) {
	for name, js := range map[string]string{
		"wrong schema": `{"hey_manifest":"hey.deploy.v1","id":"x","version":"1","prebuilt":{"a":{"path":"p"}}}`,
		"no id":        `{"hey_manifest":"hey.source.v1","version":"1","prebuilt":{"a":{"path":"p"}}}`,
		"empty":        `{"hey_manifest":"hey.source.v1","id":"x","version":"1"}`,
		"pathless":     `{"hey_manifest":"hey.source.v1","id":"x","version":"1","prebuilt":{"a":{}}}`,
	} {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected parse error", name)
		}
	}
}
