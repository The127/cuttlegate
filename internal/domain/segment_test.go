package domain

import "testing"

func TestSegment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		segment Segment
		wantErr bool
	}{
		{"valid", Segment{Slug: "beta-users", Name: "Beta Users"}, false},
		{"valid single char slug", Segment{Slug: "b", Name: "B"}, false},
		{"empty name", Segment{Slug: "beta", Name: ""}, true},
		{"empty slug", Segment{Slug: "", Name: "Beta"}, true},
		{"slug uppercase", Segment{Slug: "Beta", Name: "Beta"}, true},
		{"slug with space", Segment{Slug: "beta users", Name: "Beta"}, true},
		{"slug starts with dash", Segment{Slug: "-beta", Name: "Beta"}, true},
		{"slug with underscore", Segment{Slug: "beta_users", Name: "Beta"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.segment.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
