package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kitsyai/hey/internal/keeper"
)

// cmdKeeper is the credentials module: hey keeper auth|ls|rm. Secrets are
// named and used only when a command names one (buddy passes --cred <name>).
func cmdKeeper(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hey keeper <auth|ls|rm> ...")
	}
	switch args[0] {
	case "auth":
		return keeperAuth(args[1:])
	case "ls", "list":
		return keeper.List()
	case "rm", "delete":
		if len(args) != 2 {
			return fmt.Errorf("usage: hey keeper rm <name>")
		}
		return keeper.Remove(args[1])
	default:
		return fmt.Errorf("unknown keeper subcommand %q (use auth|ls|rm)", args[0])
	}
}

// keeperAuth stores a named credential. Prefer --token-file (a token typed on
// the command line lands in shell history).
func keeperAuth(args []string) error {
	var name, token, tokenFile string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "--token", "--token-file":
			if i+1 >= len(args) {
				return fmt.Errorf("%s needs a value", args[i])
			}
			switch args[i] {
			case "--name":
				name = args[i+1]
			case "--token":
				token = args[i+1]
			case "--token-file":
				tokenFile = args[i+1]
			}
			i++
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if name == "" {
		return fmt.Errorf("hey keeper auth needs --name <name> (e.g. gh-heypkv)")
	}
	if token == "" && tokenFile != "" {
		b, err := os.ReadFile(tokenFile)
		if err != nil {
			return err
		}
		token = strings.TrimSpace(string(b))
	}
	if token == "" {
		fmt.Fprintf(os.Stderr, "paste token for %q (visible — prefer --token-file): ", name)
		sc := bufio.NewScanner(os.Stdin)
		sc.Scan()
		token = strings.TrimSpace(sc.Text())
	}
	if token == "" {
		return fmt.Errorf("no token provided (use --token-file <path> or --token <value>)")
	}
	if err := keeper.Set(name, token); err != nil {
		return err
	}
	fmt.Printf("stored credential %q\n", name)
	return nil
}
