package main

import "fmt"

// cmdRunner is the app-running module surface: `hey runner run @heydemo/hello`.
// hey is a lean facade — running apps is one module among others (keeper,
// buddy). The top-level `hey run`/`install`/… stay as aliases into these same
// handlers, so nothing an existing user types changes.
func cmdRunner(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hey runner <run|install|update|ls|ps|stop|which> ...")
	}
	switch args[0] {
	case "run":
		return cmdRun(args[1:])
	case "install":
		return cmdInstall(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "ls":
		return cmdLs(args[1:])
	case "ps":
		return cmdPs(args[1:])
	case "stop":
		return cmdStop(args[1:])
	case "which":
		return cmdWhich(args[1:])
	default:
		return fmt.Errorf("unknown runner subcommand %q (use run|install|update|ls|ps|stop|which)", args[0])
	}
}
