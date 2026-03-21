package domain

import "time"

// FlagStateChangedEvent is published when a flag's enabled state changes
// in a specific project environment.
type FlagStateChangedEvent struct {
	ProjectSlug     string
	EnvironmentSlug string
	FlagKey         string
	Enabled         bool
	Timestamp       time.Time
}

// EventType returns the event type identifier.
func (e FlagStateChangedEvent) EventType() string { return "flag.state_changed" }

// OccurredAt returns the time the event was created.
func (e FlagStateChangedEvent) OccurredAt() time.Time { return e.Timestamp }
