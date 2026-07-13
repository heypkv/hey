package plan

import (
	"strings"
	"testing"
)

func TestParseValidAndInvalid(t *testing.T) {
	good := `{"hey_plan":1,"intent":"x","steps":[{"id":"a","tool":{"system":"nmap"}}]}`
	if _, err := Parse([]byte(good), "t"); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
	bad := []string{
		`{"intent":"x","steps":[{"id":"a","tool":{"system":"n"}}]}`,                                // no hey_plan
		`{"hey_plan":1,"steps":[{"id":"a","tool":{"system":"n"}}]}`,                                 // no intent
		`{"hey_plan":1,"intent":"x","steps":[]}`,                                                    // no steps
		`{"hey_plan":1,"intent":"x","steps":[{"id":"a","tool":{"app":"g","system":"n"}}]}`,          // both tools
		`{"hey_plan":1,"intent":"x","steps":[{"id":"a","tool":{}}]}`,                                 // no tool
		`{"hey_plan":1,"intent":"x","steps":[{"id":"a","tool":{"system":"n"},"capture":"xml"}]}`,     // bad capture
		`{"hey_plan":1,"intent":"x","steps":[{"id":"a","tool":{"system":"n"}}],"output":"z"}`,        // unknown output
		`{"hey_plan":2,"intent":"x","steps":[{"id":"a","tool":{"system":"n"}}]}`,                     // newer schema
	}
	for i, b := range bad {
		if _, err := Parse([]byte(b), "t"); err == nil {
			t.Errorf("case %d should have failed: %s", i, b)
		}
	}
}

func TestRenderKeepsUnknownTemplates(t *testing.T) {
	vars := map[string]string{"inputs.subnet": "10.0.0.0/24", "steps.scan.output": "hosts"}
	if got := Render("{{ inputs.subnet }}", vars); got != "10.0.0.0/24" {
		t.Errorf("input render = %q", got)
	}
	if got := Render("{{ steps.scan.output }}", vars); got != "hosts" {
		t.Errorf("step render = %q", got)
	}
	// A tool's own template (unknown to hey) must pass through untouched.
	if got := Render("{{ invoice.number }}.pdf", vars); got != "{{ invoice.number }}.pdf" {
		t.Errorf("unknown template should be left intact, got %q", got)
	}
}

func TestEmbeddedLibrary(t *testing.T) {
	lib, err := Library()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"make-invoice", "scan-local-devices"} {
		if _, ok := lib[want]; !ok {
			t.Errorf("seed library missing %q", want)
		}
	}
	if !strings.Contains(strings.Join(Intents(), ","), "scan-local-devices") {
		t.Error("Intents() should list the seeds")
	}
}
