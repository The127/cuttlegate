package domain

import (
	"context"
	"testing"
)

func TestContextRoundtrip(t *testing.T) {
	u := User{Sub: "sub123", Email: "alice@example.com", Name: "Alice"}
	ctx := NewContext(context.Background(), u)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext returned ok=false")
	}
	if got != u {
		t.Fatalf("got %+v, want %+v", got, u)
	}
}

func TestFromContextMissing(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Fatal("expected ok=false for empty context")
	}
}
