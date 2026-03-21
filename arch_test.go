package arch_test

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const module = "github.com/karo/cuttlegate"

func pkg(suffix string) string { return module + "/" + suffix }

// TestImportRules enforces the ports-and-adapters import direction:
//
//	cmd → adapters → app → domain
//
// Specific rules:
//  1. internal/domain imports stdlib only — no external or internal packages
//  2. internal/adapters/* must not import sibling adapter packages
//  3. internal/adapters/* must not import cmd
func TestImportRules(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports,
	}
	pkgs, err := packages.Load(cfg, module+"/...")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.FailNow()
	}

	for _, p := range pkgs {
		for imp := range p.Imports {
			checkRule(t, p.PkgPath, imp)
		}
	}
}

func checkRule(t *testing.T, from, imp string) {
	t.Helper()

	// Rule 1: domain layer must not import outside the domain layer.
	// Intra-domain imports (e.g. ports importing domain entities) are fine.
	if strings.HasPrefix(from, pkg("internal/domain")) {
		if strings.HasPrefix(imp, pkg("internal/domain")) {
			return
		}
		if !isStdlib(imp) {
			t.Errorf("IMPORT VIOLATION: %s\n\timports %s\n\treason: internal/domain allows stdlib and intra-domain imports only", from, imp)
		}
		return
	}

	// Rule 2: adapters must not import sibling adapters.
	if strings.HasPrefix(from, pkg("internal/adapters/")) &&
		strings.HasPrefix(imp, pkg("internal/adapters/")) {
		if adapterDir(from) != adapterDir(imp) {
			t.Errorf("IMPORT VIOLATION: %s\n\timports %s\n\treason: adapter packages must not cross-import siblings", from, imp)
		}
	}

	// Rule 3: adapters must not import cmd.
	if strings.HasPrefix(from, pkg("internal/adapters")) &&
		strings.HasPrefix(imp, pkg("cmd")) {
		t.Errorf("IMPORT VIOLATION: %s\n\timports %s\n\treason: adapters must not import cmd", from, imp)
	}

	// Rule 4: app layer allows stdlib and internal/domain/* only — no adapters, no third-party.
	if strings.HasPrefix(from, pkg("internal/app")) {
		if !isStdlib(imp) && !strings.HasPrefix(imp, pkg("internal/domain")) {
			t.Errorf("IMPORT VIOLATION: %s\n\timports %s\n\treason: internal/app allows stdlib and internal/domain/* only", from, imp)
		}
	}
}

// adapterDir returns the immediate subdirectory name under internal/adapters/.
// e.g. ".../internal/adapters/http/foo" → "http"
func adapterDir(pkgPath string) string {
	suffix := strings.TrimPrefix(pkgPath, pkg("internal/adapters/"))
	parts := strings.SplitN(suffix, "/", 2)
	return parts[0]
}

// isStdlib reports whether imp is a standard library package.
// Stdlib packages have no dot in their first path element (e.g. "context", "net/http").
func isStdlib(imp string) bool {
	first := strings.SplitN(imp, "/", 2)[0]
	return !strings.Contains(first, ".")
}
