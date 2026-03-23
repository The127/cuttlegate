package mcp_test

import (
	"testing"

	"github.com/karo/cuttlegate/internal/adapters/mcp"
	"github.com/karo/cuttlegate/internal/domain"
)

func TestBuildToolList_ReadTier_OnlyReadTools(t *testing.T) {
	tools := mcp.BuildToolListExported(domain.TierRead)
	names := toolNames(tools)
	if !names["list_flags"] {
		t.Error("read tier must include list_flags")
	}
	if !names["evaluate_flag"] {
		t.Error("read tier must include evaluate_flag")
	}
	if names["enable_flag"] || names["disable_flag"] {
		t.Error("read tier must NOT include write tools")
	}
}

func TestBuildToolList_WriteTier_IncludesWriteTools(t *testing.T) {
	tools := mcp.BuildToolListExported(domain.TierWrite)
	names := toolNames(tools)
	for _, n := range []string{"list_flags", "evaluate_flag", "enable_flag", "disable_flag"} {
		if !names[n] {
			t.Errorf("write tier must include %s", n)
		}
	}
}

func TestToolDescriptions_HaveCorrectTierPrefix(t *testing.T) {
	readTools := mcp.BuildToolListExported(domain.TierRead)
	writeTools := mcp.BuildToolListExported(domain.TierWrite)

	for _, tool := range readTools {
		m, _ := tool.(map[string]any)
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		if name == "list_flags" || name == "evaluate_flag" {
			if len(desc) < 6 || desc[:6] != "[read]" {
				t.Errorf("tool %s description should start with [read], got: %s", name, desc[:min(20, len(desc))])
			}
		}
	}

	for _, tool := range writeTools {
		m, _ := tool.(map[string]any)
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		if name == "enable_flag" || name == "disable_flag" {
			if len(desc) < 7 || desc[:7] != "[write]" {
				t.Errorf("tool %s description should start with [write], got: %s", name, desc[:min(20, len(desc))])
			}
		}
	}
}

func toolNames(tools []any) map[string]bool {
	names := make(map[string]bool)
	for _, tool := range tools {
		m, _ := tool.(map[string]any)
		if name, ok := m["name"].(string); ok {
			names[name] = true
		}
	}
	return names
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
