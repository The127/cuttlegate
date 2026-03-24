package domain

import (
	"regexp"
	"time"
)

type FlagType string

const (
	FlagTypeBool   FlagType = "bool"
	FlagTypeString FlagType = "string"
	FlagTypeNumber FlagType = "number"
	FlagTypeJSON   FlagType = "json"
)

type Variant struct {
	Key  string
	Name string
}

type Flag struct {
	ID                string
	ProjectID         string
	Key               string
	Name              string
	Type              FlagType
	Variants          []Variant
	DefaultVariantKey string
	CreatedAt         time.Time
}

// slugRe validates slugs and flag keys: lowercase alphanumeric with internal hyphens.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// keyRe is an alias for slugRe — flag keys use the same format.
var keyRe = slugRe

const MaxKeyLength = 128

func (f *Flag) Validate() error {
	// type
	switch f.Type {
	case FlagTypeBool, FlagTypeString, FlagTypeNumber, FlagTypeJSON:
	default:
		return &ValidationError{Field: "type", Message: "invalid flag type: " + string(f.Type)}
	}
	// name
	if f.Name == "" {
		return &ValidationError{Field: "name", Message: "must not be empty"}
	}
	// key
	if len(f.Key) > MaxKeyLength {
		return &ValidationError{Field: "key", Message: "flag key must be 128 characters or fewer"}
	}
	if !keyRe.MatchString(f.Key) {
		return &ValidationError{Field: "key", Message: "flag key must start with a letter or digit and contain only lowercase letters, digits, and hyphens"}
	}
	// variants non-empty
	if len(f.Variants) == 0 {
		return &ValidationError{Field: "variants", Message: "flag must have at least one variant"}
	}
	// duplicate variant keys
	seen := NewSet()
	for _, v := range f.Variants {
		if seen.Contains(v.Key) {
			return &ValidationError{Field: "variants", Message: "duplicate variant key: " + v.Key}
		}
		seen.Add(v.Key)
	}
	// bool invariant
	if f.Type == FlagTypeBool {
		if len(f.Variants) != 2 {
			return &ValidationError{Field: "variants", Message: "bool flag must have exactly two variants"}
		}
		keys := map[string]bool{f.Variants[0].Key: true, f.Variants[1].Key: true}
		if !keys["true"] || !keys["false"] {
			return &ValidationError{Field: "variants", Message: `bool flag variant keys must be "true" and "false"`}
		}
	}
	// default variant key present
	if !seen.Contains(f.DefaultVariantKey) {
		return &ValidationError{Field: "default_variant_key", Message: "default_variant_key does not match any variant key"}
	}
	return nil
}
