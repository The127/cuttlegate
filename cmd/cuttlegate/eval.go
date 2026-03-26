package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func cmdEval(args []string, g *globalFlags) error {
	var key, userID string
	attrs := map[string]string{}

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--help" || args[i] == "-h":
			fmt.Print("Usage: cuttlegate eval KEY [--user-id U] [--attr key=value ...] [--project P] [--environment E]\n")
			return nil
		case args[i] == "--user-id" && i+1 < len(args):
			i++
			userID = args[i]
		case strings.HasPrefix(args[i], "--user-id="):
			userID = strings.TrimPrefix(args[i], "--user-id=")
		case args[i] == "--attr" && i+1 < len(args):
			i++
			k, v, ok := strings.Cut(args[i], "=")
			if !ok {
				return fmt.Errorf("--attr value must be key=value, got %q", args[i])
			}
			attrs[k] = v
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
			return fmt.Errorf("unexpected argument %q", args[i])
		}
	}

	if key == "" {
		return fmt.Errorf("usage: cuttlegate eval KEY [--user-id U] [--attr key=value ...]")
	}

	project, env, err := resolveProjectEnv(g)
	if err != nil {
		return err
	}
	client, err := newClient(g)
	if err != nil {
		return err
	}

	reqBody := map[string]any{
		"context": map[string]any{
			"user_id":    userID,
			"attributes": attrs,
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	path := fmt.Sprintf("/api/v1/projects/%s/environments/%s/flags/%s/evaluate", project, env, key)
	data, err := client.do("POST", path, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	if g.JSON {
		fmt.Println(string(data))
		return nil
	}

	var result struct {
		Key      string  `json:"key"`
		Enabled  bool    `json:"enabled"`
		Value    *string `json:"value"`
		ValueKey string  `json:"value_key"`
		Reason   string  `json:"reason"`
		Type     string  `json:"type"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	enabled := "no"
	if result.Enabled {
		enabled = "yes"
	}
	value := "(none)"
	if result.Value != nil {
		value = *result.Value
	}
	fmt.Printf("Key:       %s\n", result.Key)
	fmt.Printf("Enabled:   %s\n", enabled)
	fmt.Printf("Value:     %s\n", value)
	fmt.Printf("ValueKey:  %s\n", result.ValueKey)
	fmt.Printf("Reason:    %s\n", result.Reason)
	fmt.Printf("Type:      %s\n", result.Type)
	return nil
}
