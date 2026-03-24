package dbadapter_test

import (
	"context"
	"testing"

	dbadapter "github.com/karo/cuttlegate/internal/adapters/db"
)

func TestTenantTxFromContext_ReturnsFalseWhenAbsent(t *testing.T) {
	_, ok := dbadapter.TenantTxFromContext(context.Background())
	if ok {
		t.Error("expected ok=false for empty context")
	}
}

func TestTenantDBTX_ReturnsFallbackWhenNoTx(t *testing.T) {
	// Use a nil DBTX as the fallback sentinel
	result := dbadapter.TenantDBTX(context.Background(), nil)
	if result != nil {
		t.Error("expected nil fallback")
	}
}
