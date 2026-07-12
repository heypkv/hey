// Package home resolves and prepares the hey home directory (~/.hey by
// default, HEY_HOME to override) and its well-known subdirectories.
package home

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns the hey home directory, creating it if needed.
func Dir() (string, error) {
	root := os.Getenv("HEY_HOME")
	if root == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		root = filepath.Join(userHome, ".hey")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create hey home %s: %w", root, err)
	}
	return root, nil
}

func subdir(name string) (string, error) {
	root, err := Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// BinDir returns ~/.hey/bin, creating it if needed.
func BinDir() (string, error) { return subdir("bin") }

// AppDir returns ~/.hey/bin/<app>, creating it if needed.
func AppDir(app string) (string, error) {
	bin, err := BinDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(bin, app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// LogsDir returns ~/.hey/logs, creating it if needed.
func LogsDir() (string, error) { return subdir("logs") }

// StateDir returns ~/.hey/state, creating it if needed.
func StateDir() (string, error) { return subdir("state") }

// SvcDir returns ~/.hey/svc, creating it if needed. Service instances live
// under it; this state is separate from app state (~/.hey/state).
func SvcDir() (string, error) { return subdir("svc") }

// InstanceDir returns ~/.hey/svc/<name>, creating it if needed.
func InstanceDir(name string) (string, error) {
	svc, err := SvcDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(svc, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}
