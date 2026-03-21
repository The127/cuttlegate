package domain

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestBucket_Determinism(t *testing.T) {
	for i := range 20 {
		first := Bucket("my-flag", "user-123")
		second := Bucket("my-flag", "user-123")
		if first != second {
			t.Fatalf("call %d: Bucket returned different values for same inputs: %d vs %d", i, first, second)
		}
		if first < 0 || first > 99 {
			t.Fatalf("Bucket returned value out of [0, 99]: %d", first)
		}
	}
}

func TestBucket_NullByteSeparator(t *testing.T) {
	// ("ab", "c") and ("a", "bc") must not collide due to null-byte separator.
	a := Bucket("ab", "c")
	b := Bucket("a", "bc")
	if a == b {
		t.Fatalf("Bucket collision: (\"ab\", \"c\") and (\"a\", \"bc\") both returned %d", a)
	}
}

func TestBucket_Distribution(t *testing.T) {
	counts := make([]int, 100)
	rng := rand.New(rand.NewPCG(42, 0))

	for range 10_000 {
		flagKey := fmt.Sprintf("flag-%d", rng.IntN(500))
		userKey := fmt.Sprintf("user-%d", rng.IntN(10_000))
		b := Bucket(flagKey, userKey)
		counts[b]++
	}

	for i, c := range counts {
		if c < 30 || c > 170 {
			t.Errorf("bucket %d has count %d — outside expected range [30, 170]", i, c)
		}
	}
}

func TestBucket_RolloutSemantics(t *testing.T) {
	tests := []struct {
		flagKey    string
		userKey    string
		rollout    int
		wantServed bool
	}{
		// Bucket("flag", "user-in") must be < rollout for served=true.
		// We find a deterministic pair by inspecting the hash.
		{"flag-a", "user-1", 100, true}, // 100% rollout always serves
		{"flag-a", "user-1", 0, false},  // 0% rollout never serves
	}

	for _, tt := range tests {
		b := Bucket(tt.flagKey, tt.userKey)
		served := b < tt.rollout
		if served != tt.wantServed {
			t.Errorf("Bucket(%q, %q)=%d < rollout=%d: got served=%v, want %v",
				tt.flagKey, tt.userKey, b, tt.rollout, served, tt.wantServed)
		}
	}
}
