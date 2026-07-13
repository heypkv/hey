// Package plan defines hey.plan.v1 — declarative orchestration recipes that map
// an intent to a deterministic sequence of tool invocations. The package holds
// the data model, validation, argument templating, and the embedded seed plan
// library; execution (tool resolution, consent, running) lives in the CLI.
// See docs/plan-v0.md.
package plan

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// SchemaVersion is the hey.plan.v1 version this build understands.
const SchemaVersion = 1

//go:embed plans/*.json
var seedFS embed.FS

// Plan is a parsed hey.plan.v1 recipe.
type Plan struct {
	HeyPlan     int     `json:"hey_plan"`
	Intent      string  `json:"intent"`
	Description string  `json:"description,omitempty"`
	Inputs      []Input `json:"inputs,omitempty"`
	Steps       []Step  `json:"steps"`
	Output      string  `json:"output,omitempty"`
}

// Input is a named parameter, resolved from --param, then default, then prompt.
type Input struct {
	Name    string `json:"name"`
	Prompt  string `json:"prompt,omitempty"`
	Default string `json:"default,omitempty"`
}

// Tool references either a registry app (installed + trust-verified) or a
// system tool (looked up on PATH; offered for install if missing). Exactly one.
type Tool struct {
	App    string `json:"app,omitempty"`
	System string `json:"system,omitempty"`
}

// Step is one tool invocation.
type Step struct {
	ID        string   `json:"id"`
	Tool      Tool     `json:"tool"`
	Sensitive bool     `json:"sensitive,omitempty"`
	Run       []string `json:"run,omitempty"`
	Capture   string   `json:"capture,omitempty"` // text | json | none (default none)
	Continue  bool     `json:"continue,omitempty"`
}

// Parse validates a plan document.
func Parse(data []byte, from string) (*Plan, error) {
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse plan (%s): %w", from, err)
	}
	if p.HeyPlan == 0 {
		return nil, fmt.Errorf("plan (%s) is missing hey_plan", from)
	}
	if p.HeyPlan > SchemaVersion {
		return nil, fmt.Errorf("plan (%s) uses hey.plan.v%d; this hey understands v%d — update hey", from, p.HeyPlan, SchemaVersion)
	}
	if p.Intent == "" {
		return nil, fmt.Errorf("plan (%s) is missing intent", from)
	}
	if len(p.Steps) == 0 {
		return nil, fmt.Errorf("plan (%s) has no steps", from)
	}
	ids := map[string]bool{}
	for i, s := range p.Steps {
		if s.ID == "" {
			return nil, fmt.Errorf("plan (%s) step %d has no id", from, i)
		}
		if ids[s.ID] {
			return nil, fmt.Errorf("plan (%s): duplicate step id %q", from, s.ID)
		}
		ids[s.ID] = true
		if (s.Tool.App == "") == (s.Tool.System == "") {
			return nil, fmt.Errorf("plan (%s) step %q: set exactly one of tool.app or tool.system", from, s.ID)
		}
		switch s.Capture {
		case "", "none", "text", "json":
		default:
			return nil, fmt.Errorf("plan (%s) step %q: capture must be text, json or none", from, s.ID)
		}
	}
	if p.Output != "" && !ids[p.Output] {
		return nil, fmt.Errorf("plan (%s): output names unknown step %q", from, p.Output)
	}
	return &p, nil
}

var varRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

// Render substitutes {{ inputs.x }} / {{ steps.id.output }} in an argument. A
// template hey doesn't recognize (e.g. a tool's own {{ invoice.number }}) is
// left intact and passed through untouched. No shell is involved — the result
// is one argv entry.
func Render(arg string, vars map[string]string) string {
	return varRe.ReplaceAllStringFunc(arg, func(m string) string {
		key := varRe.FindStringSubmatch(m)[1]
		if v, ok := vars[key]; ok {
			return v
		}
		return m
	})
}

// Vars returns the variable names referenced by an argument (for validation).
func Vars(arg string) []string {
	var out []string
	for _, m := range varRe.FindAllStringSubmatch(arg, -1) {
		out = append(out, m[1])
	}
	return out
}

// Library returns the embedded seed plans, keyed by intent.
func Library() (map[string]*Plan, error) {
	out := map[string]*Plan{}
	entries, err := fs.ReadDir(seedFS, "plans")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		data, err := seedFS.ReadFile("plans/" + e.Name())
		if err != nil {
			return nil, err
		}
		p, err := Parse(data, e.Name())
		if err != nil {
			return nil, err
		}
		out[p.Intent] = p
	}
	return out, nil
}

// Intents lists the embedded plan intents, sorted.
func Intents() []string {
	lib, err := Library()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(lib))
	for n := range lib {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Summary is a one-line human description of a step's action.
func (s Step) Summary(vars map[string]string) string {
	tool := s.Tool.App
	if tool == "" {
		tool = s.Tool.System + " (system)"
	}
	args := make([]string, len(s.Run))
	for i, a := range s.Run {
		args[i] = Render(a, vars)
	}
	return strings.TrimSpace(tool + " " + strings.Join(args, " "))
}
