// Package keeper stores and retrieves named credentials, delegating to the
// cnos runtime (authoring via the cnos CLI). hey -> keeper -> cnos. Secrets are
// named (e.g. gh-heypkv) and are only ever used when a command names one
// explicitly, so a token is never sent somewhere it wasn't asked for.
//
// v0 relies on the cnos CLI being installed (npm i -g @kitsy/cnos-cli). A
// future Go-baked cnos authoring surface would drop that dependency.
package keeper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cnos "github.com/kitsyai/cnos/packages/go"
	"github.com/kitsyai/hey/internal/home"
)

const (
	vault = "hey"
	// cnosDir is the project marker cnos writes on `cnos init`.
	cnosDir = ".cnos"
)

// PassphraseEnv is the env var cnos reads for keeper's vault passphrase; it is
// also resolvable from the OS keychain (keychain:cnos/hey) or an interactive
// prompt — cnos's own hybrid, which is exactly the storage hey wants.
const PassphraseEnv = "CNOS_SECRET_PASSPHRASE_HEY"

// root is keeper's own cnos project, isolated from whatever repo the user is in.
func root() (string, error) {
	h, err := home.Dir()
	if err != nil {
		return "", err
	}
	r := filepath.Join(h, "keeper")
	if err := os.MkdirAll(r, 0o700); err != nil {
		return "", err
	}
	return r, nil
}

func cnosCmd(args ...string) (*exec.Cmd, error) {
	bin, err := exec.LookPath("cnos")
	if err != nil {
		return nil, fmt.Errorf("cnos is not installed — keeper stores secrets via cnos.\n  install it with: npm i -g @kitsy/cnos-cli")
	}
	r, err := root()
	if err != nil {
		return nil, err
	}
	c := exec.Command(bin, args...)
	c.Dir = r // cnos operates on the project in its working directory
	return c, nil
}

// ensureProject scaffolds keeper's cnos project and local vault. It runs
// `cnos init` only when the project is absent, then ensures the vault exists.
func ensureProject() error {
	r, err := root()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(r, cnosDir)); os.IsNotExist(err) {
		if err := runInherit("cnos init", func() (*exec.Cmd, error) { return cnosCmd("init") }); err != nil {
			return err
		}
	}
	return ensureVault()
}

// ensureVault creates keeper's local vault, idempotently — an already-existing
// vault is fine. cnos needs a passphrase to secure a NEW vault; because the
// interactive prompt is unreliable when cnos is spawned by hey, keeper relies on
// CNOS_SECRET_PASSPHRASE_HEY (or an OS-keychain entry) and, when neither is
// present, surfaces cnos's own hint instead of a bare "exit status 1".
func ensureVault() error {
	c, err := cnosCmd("vault", "create", vault, "--provider", "local")
	if err != nil {
		return err
	}
	out, err := c.CombinedOutput()
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(string(out)), "already exists") {
		return nil // vault is already set up — nothing to do
	}
	return fmt.Errorf("could not create keeper's secret vault:\n%s\n\n"+
		"hey needs a passphrase to secure the vault. Set one and retry:\n"+
		"  export CNOS_SECRET_PASSPHRASE_HEY='<a-strong-passphrase>'\n"+
		"(keep it exported so `hey buddy install` can read secrets back)",
		strings.TrimSpace(string(out)))
}

func runInherit(what string, build func() (*exec.Cmd, error)) error {
	c, err := build()
	if err != nil {
		return err
	}
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stderr, os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}

// Set stores a named secret.
func Set(name, value string) error {
	if err := ensureProject(); err != nil {
		return err
	}
	return runInherit("cnos secret set", func() (*exec.Cmd, error) {
		return cnosCmd("secret", "set", name, value, "--local", "--vault", vault)
	})
}

// Get returns the raw value of a named secret via the cnos Go runtime client
// (the "read module" surface — pure Go, no CLI shell). The local vault is
// hydrated using the passphrase from the process env (CNOS_SECRET_PASSPHRASE_HEY)
// or the OS keychain (keychain:cnos/hey). buddy uses this to authenticate a
// private fetch/clone.
func Get(name string) (string, error) {
	r, err := root()
	if err != nil {
		return "", err
	}
	rt, err := cnos.Load(cnos.Options{Root: r, Profile: "local", Environment: processEnv()})
	if err != nil {
		return "", fmt.Errorf("load cnos runtime: %w", err)
	}
	v, ok, err := rt.Secret(name)
	if err != nil {
		return "", fmt.Errorf("read credential %q: %w", name, err)
	}
	if !ok {
		return "", fmt.Errorf("no credential named %q — run `hey keeper auth --name %s`", name, name)
	}
	return fmt.Sprint(v), nil
}

func processEnv() map[string]string {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			env[kv[:i]] = kv[i+1:]
		}
	}
	return env
}

// List shows stored secrets (masked).
func List() error {
	return runInherit("cnos secret list", func() (*exec.Cmd, error) {
		return cnosCmd("secret", "list")
	})
}

// Remove deletes a named secret.
func Remove(name string) error {
	return runInherit("cnos secret delete", func() (*exec.Cmd, error) {
		return cnosCmd("secret", "delete", name)
	})
}
