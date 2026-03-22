package domain

// Set is an unordered collection of unique strings.
// A nil Set is valid and treated as empty: Contains always returns false.
type Set map[string]struct{}

// NewSet returns a Set containing the given values.
func NewSet(vals ...string) Set {
	s := make(Set, len(vals))
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

// Contains reports whether v is in the set.
func (s Set) Contains(v string) bool {
	_, ok := s[v]
	return ok
}

// Add adds v to the set.
func (s Set) Add(v string) {
	s[v] = struct{}{}
}
