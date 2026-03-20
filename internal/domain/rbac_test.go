package domain

import (
	"context"
	"errors"
	"testing"
)

func TestRole_Valid(t *testing.T) {
	tests := []struct {
		role  Role
		valid bool
	}{
		{RoleAdmin, true},
		{RoleEditor, true},
		{RoleViewer, true},
		{Role("superuser"), false},
		{Role(""), false},
		{Role("ADMIN"), false},
	}
	for _, tc := range tests {
		if got := tc.role.Valid(); got != tc.valid {
			t.Errorf("Role(%q).Valid() = %v, want %v", tc.role, got, tc.valid)
		}
	}
}

func TestAuthContext_Roundtrip(t *testing.T) {
	ac := AuthContext{UserID: "u42", Role: RoleEditor}
	ctx := NewAuthContext(context.Background(), ac)
	got, ok := AuthContextFrom(ctx)
	if !ok {
		t.Fatal("AuthContextFrom: ok = false")
	}
	if got != ac {
		t.Fatalf("got %+v, want %+v", got, ac)
	}
}

func TestAuthContext_AbsentReturnsNotOk(t *testing.T) {
	_, ok := AuthContextFrom(context.Background())
	if ok {
		t.Fatal("expected ok = false for empty context")
	}
}

func TestErrForbidden_Sentinel(t *testing.T) {
	if !errors.Is(ErrForbidden, ErrForbidden) {
		t.Fatal("errors.Is(ErrForbidden, ErrForbidden) should be true")
	}
	if errors.Is(ErrForbidden, errors.New("forbidden")) {
		t.Fatal("ErrForbidden should not match a different 'forbidden' error")
	}
}
