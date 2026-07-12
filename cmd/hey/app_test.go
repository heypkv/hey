package main

import "testing"

func TestSplitAppRef(t *testing.T) {
	cases := []struct{ in, name, ver string }{
		{"guten", "guten", ""},
		{"guten@0.2.7", "guten", "0.2.7"},
		{"guten@v0.2.7", "guten", "0.2.7"},
		{"djin@1.0.0", "djin", "1.0.0"},
	}
	for _, c := range cases {
		name, ver := splitAppRef(c.in)
		if name != c.name || ver != c.ver {
			t.Errorf("splitAppRef(%q) = (%q, %q), want (%q, %q)", c.in, name, ver, c.name, c.ver)
		}
	}
}

func TestHasFlag(t *testing.T) {
	if !hasFlag([]string{"ui", "--port", "8080"}, "--port") {
		t.Error("should find --port")
	}
	if hasFlag([]string{"ui"}, "--port") {
		t.Error("should not find --port")
	}
}
