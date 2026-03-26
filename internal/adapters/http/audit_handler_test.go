package httpadapter

import (
	"testing"
	"time"

	"github.com/The127/cuttlegate/internal/domain"
)

func TestToAuditEntryResponse(t *testing.T) {
	occurred := time.Date(2026, 3, 21, 14, 32, 0, 0, time.UTC)

	event := &domain.AuditEvent{
		ID:              "evt-1",
		ProjectID:       "proj-1",
		ActorID:         "user-1",
		ActorEmail:      "alice@example.com",
		Action:          "flag.enabled",
		EntityType:      "flag",
		EntityID:        "flag-uuid-123",
		EntityKey:       "checkout-v2",
		EnvironmentSlug: "production",
		Source:          "mcp",
		BeforeState:     `{"enabled":false}`,
		AfterState:      `{"enabled":true}`,
		OccurredAt:      occurred,
	}

	resp := toAuditEntryResponse(event, "acme")

	if resp.ID != "evt-1" {
		t.Errorf("ID = %q, want %q", resp.ID, "evt-1")
	}
	if resp.ActorID != "user-1" {
		t.Errorf("ActorID = %q, want %q", resp.ActorID, "user-1")
	}
	if resp.ActorEmail != "alice@example.com" {
		t.Errorf("ActorEmail = %q, want %q", resp.ActorEmail, "alice@example.com")
	}
	if resp.Action != "flag.enabled" {
		t.Errorf("Action = %q, want %q", resp.Action, "flag.enabled")
	}
	if resp.EntityType != "flag" {
		t.Errorf("EntityType = %q, want %q", resp.EntityType, "flag")
	}
	if resp.EntityID != "flag-uuid-123" {
		t.Errorf("EntityID = %q, want %q", resp.EntityID, "flag-uuid-123")
	}
	if resp.FlagKey != "checkout-v2" {
		t.Errorf("FlagKey = %q, want %q", resp.FlagKey, "checkout-v2")
	}
	if resp.EnvironmentSlug != "production" {
		t.Errorf("EnvironmentSlug = %q, want %q", resp.EnvironmentSlug, "production")
	}
	if resp.Source != "mcp" {
		t.Errorf("Source = %q, want %q", resp.Source, "mcp")
	}
	if resp.BeforeState != `{"enabled":false}` {
		t.Errorf("BeforeState = %q, want %q", resp.BeforeState, `{"enabled":false}`)
	}
	if resp.AfterState != `{"enabled":true}` {
		t.Errorf("AfterState = %q, want %q", resp.AfterState, `{"enabled":true}`)
	}
	if resp.ProjectSlug != "acme" {
		t.Errorf("ProjectSlug = %q, want %q", resp.ProjectSlug, "acme")
	}
	if !resp.OccurredAt.Equal(occurred) {
		t.Errorf("OccurredAt = %v, want %v", resp.OccurredAt, occurred)
	}
}

func TestToAuditEntryResponse_EmptyOptionalFields(t *testing.T) {
	event := &domain.AuditEvent{
		ID:         "evt-2",
		ActorID:    "user-2",
		ActorEmail: "bob@example.com",
		Action:     "project.updated",
		OccurredAt: time.Now(),
	}

	resp := toAuditEntryResponse(event, "acme")

	if resp.EntityType != "" {
		t.Errorf("EntityType = %q, want empty", resp.EntityType)
	}
	if resp.EntityID != "" {
		t.Errorf("EntityID = %q, want empty", resp.EntityID)
	}
	if resp.BeforeState != "" {
		t.Errorf("BeforeState = %q, want empty", resp.BeforeState)
	}
	if resp.AfterState != "" {
		t.Errorf("AfterState = %q, want empty", resp.AfterState)
	}
	if resp.Source != "" {
		t.Errorf("Source = %q, want empty", resp.Source)
	}
}
