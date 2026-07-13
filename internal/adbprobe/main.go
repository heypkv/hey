// adbprobe is a synthetic stand-in for the `adb` tool, used only by the mobile
// push tests. Like internal/testapp it is never shipped. Built as "adb" onto a
// temp PATH entry, it lets a test assert hey invokes adb correctly without a
// real device:
//
//   - "adb devices"                → prints a canned device list
//   - "adb -s <id> install <apk>"  → records its full argv (one arg per line)
//     to the file named by ADB_MOCK_LOG, then prints "Success"
//
// Any invocation appends its argv to ADB_MOCK_LOG when that env var is set, so
// the test can inspect exactly what hey called.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]
	if logPath := os.Getenv("ADB_MOCK_LOG"); logPath != "" {
		line := strings.Join(args, " ") + "\n"
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			f.WriteString(line)
			f.Close()
		}
	}

	// Find the subcommand (skip any leading global flags like -s <serial>).
	sub := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "-s" {
			i++ // skip the serial value
			continue
		}
		if !strings.HasPrefix(args[i], "-") {
			sub = args[i]
			break
		}
	}

	switch sub {
	case "devices":
		fmt.Println("List of devices attached")
		fmt.Println("emulator-5554\tdevice")
		fmt.Println("192.168.1.42:5555\tdevice")
	case "install":
		fmt.Println("Performing Streamed Install")
		fmt.Println("Success")
	default:
		fmt.Fprintf(os.Stderr, "adbprobe: unhandled args %v\n", args)
		os.Exit(1)
	}
}
