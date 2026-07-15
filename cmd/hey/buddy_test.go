package main

import "testing"

func TestGitURL(t *testing.T) {
	cases := map[string]string{
		"heypkv/heyboss":                  "https://github.com/heypkv/heyboss.git",
		"heypkv/heyboss.git":              "https://github.com/heypkv/heyboss.git",
		"https://example.com/x/y.git":     "https://example.com/x/y.git",
		"git@github.com:heypkv/heyboss.git": "git@github.com:heypkv/heyboss.git",
	}
	for in, want := range cases {
		if got := gitURL(in); got != want {
			t.Errorf("gitURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRepoDir(t *testing.T) {
	cases := map[string]string{
		"heypkv/heyboss":                    "heyboss",
		"heypkv/heyboss.git":                "heyboss",
		"https://github.com/heypkv/heyboss": "heyboss",
		"git@github.com:heypkv/heyboss.git": "heyboss",
	}
	for in, want := range cases {
		if got := repoDir(in); got != want {
			t.Errorf("repoDir(%q) = %q, want %q", in, got, want)
		}
	}
}
