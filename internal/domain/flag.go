package domain

import (
	"errors"
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

var keyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func (f *Flag) Validate() error {
	// type
	switch f.Type {
	case FlagTypeBool, FlagTypeString, FlagTypeNumber, FlagTypeJSON:
	default:
		return errors.New("invalid flag type: " + string(f.Type))
	}
	// key
	if !keyRe.MatchString(f.Key) {
		return errors.New("flag key must match [a-z0-9][a-z0-9-]*")
	}
	// variants non-empty
	if len(f.Variants) == 0 {
		return errors.New("flag must have at least one variant")
	}
	// duplicate variant keys
	seen := make(map[string]struct{}, len(f.Variants))
	for _, v := range f.Variants {
		if _, dup := seen[v.Key]; dup {
			return errors.New("duplicate variant key: " + v.Key)
		}
		seen[v.Key] = struct{}{}
	}
	// bool invariant
	if f.Type == FlagTypeBool {
		if len(f.Variants) != 2 {
			return errors.New("bool flag must have exactly two variants")
		}
		keys := map[string]bool{f.Variants[0].Key: true, f.Variants[1].Key: true}
		if !keys["true"] || !keys["false"] {
			return errors.New(`bool flag variant keys must be "true" and "false"`)
		}
	}
	// default variant key present
	if _, ok := seen[f.DefaultVariantKey]; !ok {
		return errors.New("default_variant_key does not match any variant key")
	}
	return nil
}
