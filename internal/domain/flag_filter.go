package domain

// FlagListFilter holds pagination, search, and sort parameters for listing flags.
type FlagListFilter struct {
	Page    int
	PerPage int
	Search  string
	SortBy  string
	SortDir string
}

// DefaultFlagPerPage is the default number of flags per page.
const DefaultFlagPerPage = 50

// MaxFlagPerPage is the maximum allowed per_page value.
const MaxFlagPerPage = 100

// AllowedFlagSortColumns is the allowlist of columns that can be used for sorting.
var AllowedFlagSortColumns = map[string]bool{
	"key":        true,
	"name":       true,
	"type":       true,
	"created_at": true,
}

// Normalize applies defaults and clamps values to valid ranges.
// Page defaults to 1, PerPage defaults to DefaultFlagPerPage and is capped at MaxFlagPerPage.
// SortBy defaults to "created_at", SortDir defaults to "asc".
// Invalid SortBy values are reset to "created_at".
func (f *FlagListFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = DefaultFlagPerPage
	}
	if f.PerPage > MaxFlagPerPage {
		f.PerPage = MaxFlagPerPage
	}
	if !AllowedFlagSortColumns[f.SortBy] {
		f.SortBy = "created_at"
	}
	if f.SortDir != "asc" && f.SortDir != "desc" {
		f.SortDir = "asc"
	}
}
