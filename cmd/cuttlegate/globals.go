package main

import "strings"

// globalFlags are flags parsed before the subcommand.
type globalFlags struct {
	Server      string
	Project     string
	Environment string
	JSON        bool
}

// parse consumes global flags from the front of args and returns the remaining args.
func (g *globalFlags) parse(args []string) []string {
	var rest []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--json":
			g.JSON = true
		case args[i] == "--server" && i+1 < len(args):
			i++
			g.Server = args[i]
		case strings.HasPrefix(args[i], "--server="):
			g.Server = strings.TrimPrefix(args[i], "--server=")
		case args[i] == "--project" && i+1 < len(args):
			i++
			g.Project = args[i]
		case strings.HasPrefix(args[i], "--project="):
			g.Project = strings.TrimPrefix(args[i], "--project=")
		case args[i] == "--environment" && i+1 < len(args):
			i++
			g.Environment = args[i]
		case strings.HasPrefix(args[i], "--environment="):
			g.Environment = strings.TrimPrefix(args[i], "--environment=")
		default:
			// Once we hit a non-global flag, the rest is subcommand + its args.
			rest = append(rest, args[i:]...)
			return rest
		}
	}
	return rest
}
