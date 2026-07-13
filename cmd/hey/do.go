package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kitsyai/hey/internal/plan"
)

type planOpts struct {
	registryOverride string
	yes              bool
	allowUntrusted   bool
}

// cmdDo runs a plan for an intent: `hey do <intent> [--param k=v]... [--yes]`.
func cmdDo(args []string) error {
	o := planOpts{}
	params := map[string]string{}
	var pos []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--yes", "-y":
			o.yes = true
		case "--allow-untrusted":
			o.allowUntrusted = true
		case "--registry", "--param":
			if i+1 >= len(args) {
				return fmt.Errorf("%s needs a value", args[i])
			}
			if args[i] == "--registry" {
				o.registryOverride = args[i+1]
			} else {
				k, v, ok := strings.Cut(args[i+1], "=")
				if !ok {
					return fmt.Errorf("--param expects key=value, got %q", args[i+1])
				}
				params[k] = v
			}
			i++
		default:
			pos = append(pos, args[i])
		}
	}
	if len(pos) != 1 {
		return fmt.Errorf("usage: hey do <intent|plan.json> [--param k=v]... [--yes] [--allow-untrusted]")
	}

	p, trusted, err := loadPlan(pos[0])
	if err != nil {
		return err
	}
	if !trusted && !o.allowUntrusted {
		return fmt.Errorf("UNTRUSTED plan %q — hey could not verify its source. Re-run with --allow-untrusted to run it on your say-so", pos[0])
	}

	vars, err := resolveInputs(p, params, o.yes)
	if err != nil {
		return err
	}
	return runPlan(p, vars, o)
}

// loadPlan resolves an intent to a plan: an embedded seed (trusted), or a local
// .json file (untrusted). Signed, fetchable @scope plans are a later layer.
func loadPlan(ref string) (*plan.Plan, bool, error) {
	if strings.HasSuffix(ref, ".json") || strings.ContainsAny(ref, `/\`) || strings.HasPrefix(ref, ".") {
		data, err := os.ReadFile(ref)
		if err != nil {
			return nil, false, err
		}
		p, err := plan.Parse(data, ref)
		return p, false, err
	}
	lib, err := plan.Library()
	if err != nil {
		return nil, false, err
	}
	p, ok := lib[ref]
	if !ok {
		return nil, false, fmt.Errorf("no plan for %q — try `hey plan list` (have: %s)", ref, strings.Join(plan.Intents(), ", "))
	}
	return p, true, nil
}

func resolveInputs(p *plan.Plan, params map[string]string, yes bool) (map[string]string, error) {
	vars := map[string]string{}
	for _, in := range p.Inputs {
		v := params[in.Name]
		if v == "" {
			v = in.Default
		}
		if v == "" {
			if yes {
				return nil, fmt.Errorf("missing --param %s (required, no default)", in.Name)
			}
			prompt := in.Prompt
			if prompt == "" {
				prompt = in.Name
			}
			fmt.Printf("%s: ", prompt)
			sc := bufio.NewScanner(os.Stdin)
			sc.Scan()
			v = strings.TrimSpace(sc.Text())
		}
		vars["inputs."+in.Name] = v
	}
	return vars, nil
}

func runPlan(p *plan.Plan, vars map[string]string, o planOpts) error {
	for _, s := range p.Steps {
		args := make([]string, len(s.Run))
		for i, a := range s.Run {
			args[i] = plan.Render(a, vars)
		}
		toolPath, err := resolveTool(s.Tool, o)
		if err != nil {
			return err
		}
		if s.Sensitive && !o.yes {
			if !confirm(fmt.Sprintf("Run: %s", s.Summary(vars))) {
				return fmt.Errorf("aborted at step %q", s.ID)
			}
		}
		out, err := runStep(toolPath, args, s.Capture)
		if err != nil {
			if s.Continue {
				fmt.Fprintf(os.Stderr, "hey: step %q failed (continuing): %v\n", s.ID, err)
			} else {
				return fmt.Errorf("step %q: %w", s.ID, err)
			}
		}
		vars["steps."+s.ID+".output"] = out
	}
	if p.Output != "" {
		if v := vars["steps."+p.Output+".output"]; v != "" {
			fmt.Println(v)
		}
	}
	return nil
}

// resolveTool returns the executable path for a step's tool. Registry apps are
// installed and trust-verified; system tools are found on PATH, and offered for
// install via the OS package manager (with consent) when missing.
func resolveTool(t plan.Tool, o planOpts) (string, error) {
	if t.App != "" {
		if strings.HasPrefix(t.App, "@") {
			return "", fmt.Errorf("plan tool %q: @scope bundle tools aren't supported yet — use a registry app name", t.App)
		}
		reg, err := loadRegistry(o.registryOverride)
		if err != nil {
			return "", err
		}
		app, err := lookupApp(reg, t.App)
		if err != nil {
			return "", err
		}
		version, err := resolveVersion(t.App, app, "", false)
		if err != nil {
			return "", err
		}
		return ensureInstalled(t.App, app, version)
	}
	// system tool
	if path, err := exec.LookPath(t.System); err == nil {
		return path, nil
	}
	return offerInstall(t.System, o)
}

// offerInstall proposes installing a missing system tool via the OS package
// manager, shows the exact command, and runs it only with consent.
func offerInstall(tool string, o planOpts) (string, error) {
	cmdArgs, ok := installerCommand(tool)
	if !ok {
		return "", fmt.Errorf("system tool %q is not installed and no known package manager was found — install it and re-run", tool)
	}
	shown := strings.Join(cmdArgs, " ")
	if !o.yes && !confirm(fmt.Sprintf("%q is not installed. Install it now with:\n  %s\n", tool, shown)) {
		return "", fmt.Errorf("system tool %q is required but not installed", tool)
	}
	fmt.Fprintf(os.Stderr, "hey: %s\n", shown)
	c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("install %q: %w", tool, err)
	}
	return exec.LookPath(tool)
}

// installerCommand builds an install command for the current OS's package
// manager (the first one found), or reports that none is available.
func installerCommand(tool string) ([]string, bool) {
	type pm struct {
		bin  string
		args []string
	}
	var candidates []pm
	switch runtime.GOOS {
	case "darwin":
		candidates = []pm{{"brew", []string{"install", tool}}}
	case "windows":
		candidates = []pm{{"winget", []string{"install", "--accept-source-agreements", "--accept-package-agreements", tool}}}
	default: // linux and friends
		candidates = []pm{
			{"apt-get", []string{"install", "-y", tool}},
			{"dnf", []string{"install", "-y", tool}},
			{"pacman", []string{"-S", "--noconfirm", tool}},
			{"zypper", []string{"install", "-y", tool}},
			{"apk", []string{"add", tool}},
		}
	}
	for _, c := range candidates {
		if path, err := exec.LookPath(c.bin); err == nil {
			cmd := append([]string{path}, c.args...)
			// Elevate package installs on unix when not already root.
			if runtime.GOOS != "windows" && os.Geteuid() != 0 {
				if sudo, err := exec.LookPath("sudo"); err == nil {
					cmd = append([]string{sudo}, cmd...)
				}
			}
			return cmd, true
		}
	}
	return nil, false
}

// runStep runs the tool with args, capturing per the step's capture mode.
func runStep(toolPath string, args []string, capture string) (string, error) {
	c := exec.Command(toolPath, args...)
	c.Stderr = os.Stderr
	if capture == "text" || capture == "json" {
		out, err := c.Output()
		return strings.TrimRight(string(out), "\n"), err
	}
	c.Stdin, c.Stdout = os.Stdin, os.Stdout
	return "", c.Run()
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	sc := bufio.NewScanner(os.Stdin)
	sc.Scan()
	a := strings.ToLower(strings.TrimSpace(sc.Text()))
	return a == "y" || a == "yes"
}

// cmdPlan handles `hey plan list` and `hey plan show <intent>`.
func cmdPlan(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hey plan <list|show <intent>>")
	}
	switch args[0] {
	case "list":
		lib, err := plan.Library()
		if err != nil {
			return err
		}
		for _, intent := range plan.Intents() {
			fmt.Printf("%-22s %s\n", intent, lib[intent].Description)
		}
		return nil
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: hey plan show <intent>")
		}
		p, _, err := loadPlan(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("%s — %s\n", p.Intent, p.Description)
		if len(p.Inputs) > 0 {
			fmt.Println("inputs:")
			for _, in := range p.Inputs {
				def := ""
				if in.Default != "" {
					def = " (default: " + in.Default + ")"
				}
				fmt.Printf("  %s%s\n", in.Name, def)
			}
		}
		fmt.Println("steps:")
		for _, s := range p.Steps {
			flag := ""
			if s.Sensitive {
				flag = "  [asks consent]"
			}
			fmt.Printf("  %s: %s%s\n", s.ID, s.Summary(map[string]string{}), flag)
		}
		return nil
	default:
		return fmt.Errorf("unknown plan subcommand %q (use list|show)", args[0])
	}
}
