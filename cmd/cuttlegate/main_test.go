package main

import (
	"testing"
)

func TestDispatchUnknownCommand(t *testing.T) {
	err := run([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	want := `unknown command "nonexistent"`
	if got := err.Error(); got[:len(want)] != want {
		t.Errorf("error = %q, want prefix %q", got, want)
	}
}

func TestDispatchHelp(t *testing.T) {
	// Should not error.
	if err := run([]string{"help"}); err != nil {
		t.Errorf("help: %v", err)
	}
	if err := run([]string{"--help"}); err != nil {
		t.Errorf("--help: %v", err)
	}
}

func TestDispatchVersion(t *testing.T) {
	if err := run([]string{"version"}); err != nil {
		t.Errorf("version: %v", err)
	}
}

func TestDispatchNoArgs(t *testing.T) {
	if err := run(nil); err != nil {
		t.Errorf("no args: %v", err)
	}
}

func TestGlobalFlagsParsing(t *testing.T) {
	var g globalFlags
	rest := g.parse([]string{"--server", "https://s.com", "--project", "p1", "--json", "flags", "list"})

	if g.Server != "https://s.com" {
		t.Errorf("server = %q", g.Server)
	}
	if g.Project != "p1" {
		t.Errorf("project = %q", g.Project)
	}
	if !g.JSON {
		t.Error("expected JSON=true")
	}
	if len(rest) != 2 || rest[0] != "flags" || rest[1] != "list" {
		t.Errorf("rest = %v", rest)
	}
}
