package domain

import (
	"errors"
	"testing"
)

func TestEnvironment_Validate(t *testing.T) {
	tests := []struct {
		name      string
		e         Environment
		wantField string
	}{
		{
			name: "valid environment",
			e:    Environment{Name: "Production", Slug: "prod"},
		},
		{
			name:      "empty name",
			e:         Environment{Name: "", Slug: "prod"},
			wantField: "name",
		},
		{
			name:      "empty slug",
			e:         Environment{Name: "Production", Slug: ""},
			wantField: "slug",
		},
		{
			name:      "slug with uppercase",
			e:         Environment{Name: "Production", Slug: "My-Env"},
			wantField: "slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.e.Validate()
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var valErr *ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			if valErr.Field != tt.wantField {
				t.Errorf("field: got %q, want %q", valErr.Field, tt.wantField)
			}
		})
	}
}
