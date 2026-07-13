// planprobe is a mock "system tool" used by the plan executor tests: it prints
// its arguments so a step's capture/templating can be asserted. Never shipped.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("planprobe " + strings.Join(os.Args[1:], " "))
}
