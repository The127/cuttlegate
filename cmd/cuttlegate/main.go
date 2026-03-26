// Command cuttlegate is the CLI client for the Cuttlegate feature-flag server.
// It authenticates via OIDC device flow and calls the REST API.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	// Global flags consumed before subcommand.
	var globals globalFlags
	args = globals.parse(args)

	if len(args) == 0 {
		printUsage()
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "login":
		return cmdLogin(rest, &globals)
	case "whoami":
		return cmdWhoami(&globals)
	case "config":
		return cmdConfig(rest)
	case "projects":
		return cmdProjects(rest, &globals)
	case "flags":
		return cmdFlags(rest, &globals)
	case "eval":
		return cmdEval(rest, &globals)
	case "help", "--help", "-h":
		printUsage()
		return nil
	case "version", "--version":
		fmt.Println("cuttlegate dev")
		return nil
	default:
		return fmt.Errorf("unknown command %q — run 'cuttlegate help' for usage", cmd)
	}
}

func printUsage() {
	fmt.Print(`Usage: cuttlegate <command> [flags]

Commands:
  login       Authenticate via OIDC device flow
  whoami      Show current user from stored token
  config      Get or set CLI configuration
  projects    List and inspect projects
  flags       List, get, enable, or disable flags
  eval        Evaluate a flag

Global flags (placed before the command):
  --server URL          Override server URL
  --project SLUG        Override project slug
  --environment SLUG    Override environment slug
  --json                Output machine-readable JSON

Run 'cuttlegate <command> --help' for command-specific usage.
`)
}
