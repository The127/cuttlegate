package domain

import "time"

// FlagStateChangedEvent is published when a flag's enabled state changes
// in a specific project environment.
type FlagStateChangedEvent struct {
	project     string
	environment string
	flagKey     string
	enabled     bool
	timestamp   time.Time
}

// NewFlagStateChangedEvent constructs a FlagStateChangedEvent.
func NewFlagStateChangedEvent(projectSlug, envSlug, flagKey string, enabled bool) FlagStateChangedEvent {
	return FlagStateChangedEvent{
		project:     projectSlug,
		environment: envSlug,
		flagKey:     flagKey,
		enabled:     enabled,
		timestamp:   time.Now().UTC(),
	}
}

// EventType returns the event type identifier.
func (e FlagStateChangedEvent) EventType() string { return "flag.state_changed" }

// OccurredAt returns the time the event was created.
func (e FlagStateChangedEvent) OccurredAt() time.Time { return e.timestamp }

// ProjectSlug returns the project slug.
func (e FlagStateChangedEvent) ProjectSlug() string { return e.project }

// EnvironmentSlug returns the environment slug.
func (e FlagStateChangedEvent) EnvironmentSlug() string { return e.environment }

// FlagKey returns the flag key.
func (e FlagStateChangedEvent) FlagKey() string { return e.flagKey }

// Enabled returns whether the flag was enabled or disabled.
func (e FlagStateChangedEvent) Enabled() bool { return e.enabled }
