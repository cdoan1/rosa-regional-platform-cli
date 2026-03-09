package main

import (
	"os"

	"github.com/openshift-online/rosa-regional-platform-cli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
