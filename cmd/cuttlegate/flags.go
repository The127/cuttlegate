package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func cmdFlags(args []string, g *globalFlags) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cuttlegate flags <list|get|enable|disable> [args]")
	}

	// Parse per-command flags that may appear after the key argument.
	subcmd := args[0]
	rest := args[1:]

	switch subcmd {
	case "list":
		return cmdFlagsList(rest, g)
	case "get":
		return cmdFlagsGet(rest, g)
	case "enable":
		return cmdFlagsSetEnabled(rest, g, true)
	case "disable":
		return cmdFlagsSetEnabled(rest, g, false)
	case "--help", "-h":
		fmt.Print("Usage: cuttlegate flags <list|get|enable|disable> [args]\n\n  list       List flags in an environment\n  get KEY    Get a flag by key\n  enable KEY   Enable a flag\n  disable KEY  Disable a flag\n")
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q", subcmd)
	}
}

func parseFlagArgs(args []string, g *globalFlags, requireKey bool) (key string, err error) {
	for i := 0; i < len(args); i++ {
		switch {
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
		case !strings.HasPrefix(args[i], "-") && key == "":
			key = args[i]
		default:
			return "", fmt.Errorf("unexpected argument %q", args[i])
		}
	}
	if requireKey && key == "" {
		return "", fmt.Errorf("flag key is required")
	}
	return key, nil
}

func cmdFlagsList(args []string, g *globalFlags) error {
	if _, err := parseFlagArgs(args, g, false); err != nil {
		return err
	}
	project, env, err := resolveProjectEnv(g)
	if err != nil {
		return err
	}
	client, err := newClient(g)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/projects/%s/environments/%s/flags", project, env)
	data, err := client.do("GET", path, nil)
	if err != nil {
		return err
	}

	if g.JSON {
		fmt.Println(string(data))
		return nil
	}

	var resp struct {
		Flags []struct {
			Key     string `json:"key"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Enabled bool   `json:"enabled"`
		} `json:"flags"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(resp.Flags) == 0 {
		fmt.Println("No flags found.")
		return nil
	}

	fmt.Printf("%-30s %-20s %-10s %s\n", "KEY", "NAME", "TYPE", "ENABLED")
	fmt.Printf("%-30s %-20s %-10s %s\n", strings.Repeat("-", 30), strings.Repeat("-", 20), strings.Repeat("-", 10), strings.Repeat("-", 7))
	for _, f := range resp.Flags {
		enabled := "no"
		if f.Enabled {
			enabled = "yes"
		}
		fmt.Printf("%-30s %-20s %-10s %s\n", f.Key, f.Name, f.Type, enabled)
	}
	return nil
}

func cmdFlagsGet(args []string, g *globalFlags) error {
	key, err := parseFlagArgs(args, g, true)
	if err != nil {
		return err
	}
	project, env, err := resolveProjectEnv(g)
	if err != nil {
		return err
	}
	client, err := newClient(g)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/projects/%s/environments/%s/flags/%s", project, env, key)
	data, err := client.do("GET", path, nil)
	if err != nil {
		return err
	}

	if g.JSON {
		fmt.Println(string(data))
		return nil
	}

	var f struct {
		ID                string `json:"id"`
		Key               string `json:"key"`
		Name              string `json:"name"`
		Type              string `json:"type"`
		Enabled           bool   `json:"enabled"`
		DefaultVariantKey string `json:"default_variant_key"`
		Variants          []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"variants"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	enabled := "no"
	if f.Enabled {
		enabled = "yes"
	}
	fmt.Printf("Key:       %s\n", f.Key)
	fmt.Printf("Name:      %s\n", f.Name)
	fmt.Printf("Type:      %s\n", f.Type)
	fmt.Printf("Enabled:   %s\n", enabled)
	fmt.Printf("Default:   %s\n", f.DefaultVariantKey)
	if len(f.Variants) > 0 {
		fmt.Printf("Variants:\n")
		for _, v := range f.Variants {
			fmt.Printf("  - %s (%s)\n", v.Key, v.Name)
		}
	}
	return nil
}

func cmdFlagsSetEnabled(args []string, g *globalFlags, enabled bool) error {
	key, err := parseFlagArgs(args, g, true)
	if err != nil {
		return err
	}
	project, env, err := resolveProjectEnv(g)
	if err != nil {
		return err
	}
	client, err := newClient(g)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(map[string]bool{"enabled": enabled})
	path := fmt.Sprintf("/api/v1/projects/%s/environments/%s/flags/%s", project, env, key)
	_, err = client.do("PATCH", path, bytes.NewReader(body))
	if err != nil {
		return err
	}

	action := "disabled"
	if enabled {
		action = "enabled"
	}
	fmt.Printf("Flag %q %s in %s/%s\n", key, action, project, env)
	return nil
}
