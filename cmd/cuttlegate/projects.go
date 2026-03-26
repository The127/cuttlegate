package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func cmdProjects(args []string, g *globalFlags) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cuttlegate projects <list|get> [args]")
	}
	switch args[0] {
	case "list":
		return cmdProjectsList(g)
	case "get":
		return cmdProjectsGet(args[1:], g)
	case "--help", "-h":
		fmt.Print("Usage: cuttlegate projects <list|get> [args]\n\n  list    List all projects\n  get     Get a project by slug\n")
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q — use 'list' or 'get'", args[0])
	}
}

func cmdProjectsList(g *globalFlags) error {
	client, err := newClient(g)
	if err != nil {
		return err
	}
	data, err := client.do("GET", "/api/v1/projects", nil)
	if err != nil {
		return err
	}

	if g.JSON {
		fmt.Println(string(data))
		return nil
	}

	var resp struct {
		Projects []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Slug      string `json:"slug"`
			CreatedAt string `json:"created_at"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	// Simple table output.
	fmt.Printf("%-20s %-30s %s\n", "SLUG", "NAME", "CREATED")
	fmt.Printf("%-20s %-30s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 30), strings.Repeat("-", 20))
	for _, p := range resp.Projects {
		created := p.CreatedAt
		if len(created) > 10 {
			created = created[:10]
		}
		fmt.Printf("%-20s %-30s %s\n", p.Slug, p.Name, created)
	}
	return nil
}

func cmdProjectsGet(args []string, g *globalFlags) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: cuttlegate projects get <SLUG>")
	}
	slug := args[0]

	client, err := newClient(g)
	if err != nil {
		return err
	}
	data, err := client.do("GET", "/api/v1/projects/"+slug, nil)
	if err != nil {
		return err
	}

	if g.JSON {
		fmt.Println(string(data))
		return nil
	}

	var p struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Printf("ID:       %s\n", p.ID)
	fmt.Printf("Name:     %s\n", p.Name)
	fmt.Printf("Slug:     %s\n", p.Slug)
	fmt.Printf("Created:  %s\n", p.CreatedAt)
	return nil
}
