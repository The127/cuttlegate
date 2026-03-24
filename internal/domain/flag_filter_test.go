package domain

import "testing"

func TestFlagListFilter_Normalize(t *testing.T) {
	tests := []struct {
		name   string
		input  FlagListFilter
		expect FlagListFilter
	}{
		{
			name:   "zero values get defaults",
			input:  FlagListFilter{},
			expect: FlagListFilter{Page: 1, PerPage: DefaultFlagPerPage, SortBy: "created_at", SortDir: "asc"},
		},
		{
			name:   "explicit valid values preserved",
			input:  FlagListFilter{Page: 3, PerPage: 25, SortBy: "key", SortDir: "desc", Search: "beta"},
			expect: FlagListFilter{Page: 3, PerPage: 25, SortBy: "key", SortDir: "desc", Search: "beta"},
		},
		{
			name:   "per_page capped at max",
			input:  FlagListFilter{Page: 1, PerPage: 500, SortBy: "name", SortDir: "asc"},
			expect: FlagListFilter{Page: 1, PerPage: MaxFlagPerPage, SortBy: "name", SortDir: "asc"},
		},
		{
			name:   "negative page defaults to 1",
			input:  FlagListFilter{Page: -5, PerPage: 10, SortBy: "created_at", SortDir: "asc"},
			expect: FlagListFilter{Page: 1, PerPage: 10, SortBy: "created_at", SortDir: "asc"},
		},
		{
			name:   "invalid sort_by defaults to created_at",
			input:  FlagListFilter{Page: 1, PerPage: 50, SortBy: "DROP TABLE", SortDir: "asc"},
			expect: FlagListFilter{Page: 1, PerPage: 50, SortBy: "created_at", SortDir: "asc"},
		},
		{
			name:   "invalid sort_dir defaults to asc",
			input:  FlagListFilter{Page: 1, PerPage: 50, SortBy: "key", SortDir: "sideways"},
			expect: FlagListFilter{Page: 1, PerPage: 50, SortBy: "key", SortDir: "asc"},
		},
		{
			name:   "type is a valid sort column",
			input:  FlagListFilter{Page: 1, PerPage: 50, SortBy: "type", SortDir: "desc"},
			expect: FlagListFilter{Page: 1, PerPage: 50, SortBy: "type", SortDir: "desc"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.input
			f.Normalize()
			if f.Page != tc.expect.Page {
				t.Errorf("Page: got %d, want %d", f.Page, tc.expect.Page)
			}
			if f.PerPage != tc.expect.PerPage {
				t.Errorf("PerPage: got %d, want %d", f.PerPage, tc.expect.PerPage)
			}
			if f.SortBy != tc.expect.SortBy {
				t.Errorf("SortBy: got %q, want %q", f.SortBy, tc.expect.SortBy)
			}
			if f.SortDir != tc.expect.SortDir {
				t.Errorf("SortDir: got %q, want %q", f.SortDir, tc.expect.SortDir)
			}
			if f.Search != tc.expect.Search {
				t.Errorf("Search: got %q, want %q", f.Search, tc.expect.Search)
			}
		})
	}
}
