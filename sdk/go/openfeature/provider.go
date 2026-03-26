// Package openfeature provides an OpenFeature Provider backed by the Cuttlegate Go SDK.
//
// Usage:
//
//	import (
//	    cuttlegate "github.com/The127/cuttlegate/sdk/go"
//	    cuttlegateof "github.com/The127/cuttlegate/sdk/go/openfeature"
//	    "github.com/open-feature/go-sdk/openfeature"
//	)
//
//	client, _ := cuttlegate.NewClient(cuttlegate.Config{...})
//	openfeature.SetProvider(cuttlegateof.NewProvider(client))
//
//	ofClient := openfeature.NewClient("my-app")
//	value, _ := ofClient.BooleanValue(ctx, "dark-mode", false, openfeature.EvaluationContext{})
package openfeature

import (
	"context"

	cuttlegate "github.com/The127/cuttlegate/sdk/go"
	of "github.com/open-feature/go-sdk/openfeature"
)

// Compile-time check that Provider implements of.FeatureProvider.
var _ of.FeatureProvider = (*Provider)(nil)

// Provider implements the OpenFeature FeatureProvider interface using
// the Cuttlegate Go SDK.
type Provider struct {
	client cuttlegate.Client
}

// NewProvider creates a new OpenFeature-compatible provider backed by client.
func NewProvider(client cuttlegate.Client) *Provider {
	return &Provider{client: client}
}

// Metadata returns the provider metadata.
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{Name: "cuttlegate"}
}

// Hooks returns an empty hook slice.
func (p *Provider) Hooks() []of.Hook {
	return nil
}

func toEvalContext(fc of.FlattenedContext) cuttlegate.EvalContext {
	attrs := make(map[string]any, len(fc))
	var targetingKey string
	for k, v := range fc {
		if k == "targetingKey" {
			if s, ok := v.(string); ok {
				targetingKey = s
			}
			continue
		}
		attrs[k] = v
	}
	return cuttlegate.EvalContext{
		UserID:     targetingKey,
		Attributes: attrs,
	}
}

// BooleanEvaluation resolves a boolean flag.
func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, fc of.FlattenedContext) of.BoolResolutionDetail {
	val, err := p.client.Bool(ctx, flag, toEvalContext(fc))
	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:          of.ErrorReason,
				ResolutionError: of.NewGeneralResolutionError(err.Error()),
			},
		}
	}
	variant := "false"
	if val {
		variant = "true"
	}
	return of.BoolResolutionDetail{
		Value: val,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Variant: variant,
			Reason:  of.TargetingMatchReason,
		},
	}
}

// StringEvaluation resolves a string flag.
func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, fc of.FlattenedContext) of.StringResolutionDetail {
	val, err := p.client.String(ctx, flag, toEvalContext(fc))
	if err != nil {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:          of.ErrorReason,
				ResolutionError: of.NewGeneralResolutionError(err.Error()),
			},
		}
	}
	return of.StringResolutionDetail{
		Value: val,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Variant: val,
			Reason:  of.TargetingMatchReason,
		},
	}
}

// FloatEvaluation resolves a float flag by parsing the string variant.
func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, fc of.FlattenedContext) of.FloatResolutionDetail {
	return of.FloatResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:          of.ErrorReason,
			ResolutionError: of.NewGeneralResolutionError("float evaluation not supported — use string and parse"),
		},
	}
}

// IntEvaluation resolves an integer flag by parsing the string variant.
func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, fc of.FlattenedContext) of.IntResolutionDetail {
	return of.IntResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:          of.ErrorReason,
			ResolutionError: of.NewGeneralResolutionError("int evaluation not supported — use string and parse"),
		},
	}
}

// ObjectEvaluation resolves an object flag by parsing the JSON string variant.
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, fc of.FlattenedContext) of.InterfaceResolutionDetail {
	return of.InterfaceResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:          of.ErrorReason,
			ResolutionError: of.NewGeneralResolutionError("object evaluation not supported — use string and parse JSON"),
		},
	}
}
