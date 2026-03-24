package domain

import (
	"errors"
	"testing"
)

func TestProject_Validate(t *testing.T) {
	tests := []struct {
		name      string
		p         Project
		wantField string
	}{
		{
			name: "valid project",
			p:    Project{Name: "Acme Corp", Slug: "acme"},
		},
		{
			name: "single-char slug — valid",
			p:    Project{Name: "A", Slug: "a"},
		},
		{
			name:      "empty name",
			p:         Project{Name: "", Slug: "acme"},
			wantField: "name",
		},
		{
			name:      "empty slug",
			p:         Project{Name: "Acme", Slug: ""},
			wantField: "slug",
		},
		{
			name:      "slug with uppercase",
			p:         Project{Name: "Acme", Slug: "Acme"},
			wantField: "slug",
		},
		{
			name:      "slug with spaces",
			p:         Project{Name: "Acme", Slug: "Hello World!!"},
			wantField: "slug",
		},
		{
			name:      "slug starting with hyphen",
			p:         Project{Name: "Acme", Slug: "-acme"},
			wantField: "slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
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
