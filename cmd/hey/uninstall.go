package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kitsyai/hey/internal/home"
	"github.com/kitsyai/hey/internal/proc"
	"github.com/kitsyai/hey/internal/state"
	"github.com/kitsyai/hey/internal/svc"
)

// cmdUninstall removes hey as if it was never installed: it stops everything
// hey is managing, deletes ~/.hey (apps, bundles, service data, logs, state),
// removes the hey binary, and (on Windows) the PATH entry the installer added.
func cmdUninstall(args []string) error {
	yes := false
	for _, a := range args {
		switch a {
		case "--yes", "-y":
			yes = true
		default:
			return fmt.Errorf("unknown flag %q (usage: hey uninstall [--yes])", a)
		}
	}

	heyHome, err := home.Dir()
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate the hey binary: %w", err)
	}
	if resolved, rerr := filepath.EvalSymlinks(self); rerr == nil {
		self = resolved
	}

	fmt.Println("hey uninstall will remove:")
	fmt.Printf("  - %s\n      (every installed app, deployed bundle, service data, log and state file)\n", heyHome)
	fmt.Printf("  - %s\n      (the hey binary itself)\n", self)
	if running := managedNames(heyHome); len(running) > 0 {
		fmt.Printf("  - and first stop %d running item(s): %s\n", len(running), strings.Join(running, ", "))
	}
	fmt.Println("This cannot be undone — provisioned databases and their data are deleted too.")

	if !yes {
		fmt.Print("Type 'yes' to continue: ")
		sc := bufio.NewScanner(os.Stdin)
		sc.Scan()
		if strings.TrimSpace(sc.Text()) != "yes" {
			fmt.Println("aborted; nothing was removed.")
			return nil
		}
	}

	stopManaged(heyHome)

	if err := os.RemoveAll(heyHome); err != nil {
		return fmt.Errorf("remove %s: %w", heyHome, err)
	}
	fmt.Printf("removed %s\n", heyHome)

	if err := removeSelf(self); err != nil {
		return fmt.Errorf("remove the hey binary at %s: %w", self, err)
	}
	fmt.Println("hey uninstalled — gone as if it was never here.")
	return nil
}

// managedNames lists live UI apps and services hey is tracking (read-only).
func managedNames(heyHome string) []string {
	var names []string
	if procs, err := state.Load(filepath.Join(heyHome, "state")); err == nil {
		for _, p := range procs {
			if proc.Alive(p.PID) {
				names = append(names, p.App)
			}
		}
	}
	entries, _ := os.ReadDir(filepath.Join(heyHome, "svc"))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if inst, err := svc.LoadInstance(filepath.Join(heyHome, "svc", e.Name())); err == nil && proc.Alive(inst.PID) {
			names = append(names, "svc:"+e.Name())
		}
	}
	return names
}

// stopManaged best-effort terminates every tracked UI app and service so the
// removal doesn't orphan processes.
func stopManaged(heyHome string) {
	if procs, err := state.Load(filepath.Join(heyHome, "state")); err == nil {
		for _, p := range procs {
			if proc.Alive(p.PID) {
				_ = proc.KillTree(p.PID)
			}
		}
	}
	entries, _ := os.ReadDir(filepath.Join(heyHome, "svc"))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if inst, err := svc.LoadInstance(filepath.Join(heyHome, "svc", e.Name())); err == nil && proc.Alive(inst.PID) {
			_ = proc.KillTree(inst.PID)
		}
	}
}
