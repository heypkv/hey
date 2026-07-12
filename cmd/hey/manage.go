package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/heypkv/hey/internal/home"
)

func cmdInstall(args []string) error {
	registryOverride, rest, err := takeRegistryFlag(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("usage: hey install <app>[@version]")
	}
	name, pinned := splitAppRef(rest[0])
	reg, err := loadRegistry(registryOverride)
	if err != nil {
		return err
	}
	app, err := lookupApp(reg, name)
	if err != nil {
		return err
	}
	version, err := resolveVersion(name, app, pinned, false)
	if err != nil {
		return err
	}
	binPath, err := ensureInstalled(name, app, version)
	if err != nil {
		return err
	}
	fmt.Printf("%s %s -> %s\n", name, version, binPath)
	return nil
}

func cmdUpdate(args []string) error {
	registryOverride, rest, err := takeRegistryFlag(args)
	if err != nil {
		return err
	}
	reg, err := loadRegistry(registryOverride)
	if err != nil {
		return err
	}

	var names []string
	if len(rest) == 1 {
		names = []string{rest[0]}
	} else if len(rest) == 0 {
		names = installedApps()
		if len(names) == 0 {
			fmt.Println("nothing installed yet — try `hey install <app>`")
			return nil
		}
	} else {
		return fmt.Errorf("usage: hey update [<app>]")
	}

	for _, name := range names {
		app, err := lookupApp(reg, name)
		if err != nil {
			return err
		}
		version, err := resolveVersion(name, app, "", true) // bypass resolve cache
		if err != nil {
			return err
		}
		if cur, ok := currentVersion(name); ok && cur == version {
			fmt.Printf("%s %s is already the latest\n", name, version)
			continue
		}
		if _, err := ensureInstalled(name, app, version); err != nil {
			return err
		}
		fmt.Printf("%s updated to %s\n", name, version)
	}
	return nil
}

func cmdLs(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: hey ls")
	}
	names := installedApps()
	if len(names) == 0 {
		fmt.Println("nothing installed yet — try `hey install <app>`")
		return nil
	}
	binDir, err := home.BinDir()
	if err != nil {
		return err
	}
	for _, name := range names {
		cur, _ := currentVersion(name)
		versions := installedVersions(name)
		fmt.Printf("%-12s current %-10s versions [%s]  %s\n",
			name, cur, joinStrings(versions, ", "), filepath.Join(binDir, name))
	}
	return nil
}

func cmdWhich(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: hey which <app>")
	}
	name, pinned := splitAppRef(args[0])
	version := pinned
	if version == "" {
		var ok bool
		if version, ok = currentVersion(name); !ok {
			return fmt.Errorf("%s is not installed", name)
		}
	}
	appDir, err := home.AppDir(name)
	if err != nil {
		return err
	}
	// Try both plain and .exe names so `which` works for any cached platform.
	for _, candidate := range []string{name + ".exe", name} {
		p := filepath.Join(appDir, version, candidate)
		if _, err := os.Stat(p); err == nil {
			fmt.Println(p)
			return nil
		}
	}
	return fmt.Errorf("%s %s is not installed", name, version)
}

func cmdCache(args []string) error {
	if len(args) == 0 || args[0] != "clean" {
		return fmt.Errorf("usage: hey cache clean [<app>]")
	}
	binDir, err := home.BinDir()
	if err != nil {
		return err
	}
	switch len(args) {
	case 1:
		names := installedApps()
		for _, name := range names {
			if err := os.RemoveAll(filepath.Join(binDir, name)); err != nil {
				return err
			}
		}
		fmt.Printf("removed %d cached app(s)\n", len(names))
	case 2:
		name, _ := splitAppRef(args[1])
		target := filepath.Join(binDir, name)
		if _, err := os.Stat(target); err != nil {
			return fmt.Errorf("%s is not installed", name)
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
		fmt.Printf("removed cached %s\n", name)
	default:
		return fmt.Errorf("usage: hey cache clean [<app>]")
	}
	return nil
}

// --- helpers ---

func takeRegistryFlag(args []string) (override string, rest []string, err error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--registry" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--registry needs a value")
			}
			override = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	return override, rest, nil
}

func installedApps() []string {
	binDir, err := home.BinDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

func installedVersions(name string) []string {
	appDir, err := home.AppDir(name)
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return nil
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() && e.Name()[0] != '.' {
			versions = append(versions, e.Name())
		}
	}
	sort.Strings(versions)
	return versions
}

func joinStrings(s []string, sep string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}
