package arch_test

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const module = "github.com/The127/cuttlegate"

func pkg(suffix string) string { return module + "/" + suffix }

// TestImportRules enforces the ports-and-adapters import direction:
//
//	cmd → adapters → app → domain
//
// Specific rules:
//  1. internal/domain imports stdlib only — no external or internal packages
//  2. internal/adapters/* must not import sibling adapter packages
//  3. internal/adapters/* must not import cmd
//  4. internal/app imports stdlib and internal/domain/* only
//
// Tests: true causes packages.Load to produce additional test-related package
// variants beyond the normal production packages. These include:
//   - External test packages (PkgPath ends with "_test", e.g. "…/app_test")
//   - Test binary packages (PkgPath ends with ".test", e.g. "…/app.test")
//   - Synthesised test variants (PkgPath contains "[", e.g. "… [….test]")
//
// Production purity rules (1, 3, 4) must not fire for any test-related package.
// checkRule skips test packages; checkTestImportRule applies only Rule 2 to them.
func TestImportRules(t *testing.T) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports,
		Tests: true,
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
			checkTestImportRule(t, p.PkgPath, imp)
		}
	}
}

// isTestPackage reports whether pkgPath identifies a test-related package variant
// produced by packages.Load when Tests: true. This includes:
//   - External test packages ending in "_test" (e.g. "…/app_test")
//   - Test binary packages ending in ".test" (e.g. "…/app.test")
//   - Synthesised test variants containing "[" (e.g. "… [….test]")
func isTestPackage(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, "_test") ||
		strings.HasSuffix(pkgPath, ".test") ||
		strings.Contains(pkgPath, "[")
}

// checkRule enforces production import rules. It is a no-op for test-related
// package variants — those are handled by checkTestImportRule.
func checkRule(t *testing.T, from, imp string) {
	t.Helper()

	// Test package variants are not subject to production purity rules.
	if isTestPackage(from) {
		return
	}

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
	if isCrossAdapterImport(from, imp) {
		t.Errorf("IMPORT VIOLATION: %s\n\timports %s\n\treason: adapter packages must not cross-import siblings", from, imp)
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

// checkTestImportRule enforces Rule 2 (no cross-adapter imports) for test-related
// package variants. It is a no-op for production packages. The base package path
// is recovered via testPackageBasePkg so adapterDir can identify the adapter
// directory correctly even for "_test" and ".test" suffixed paths.
func checkTestImportRule(t *testing.T, from, imp string) {
	t.Helper()

	if !isTestPackage(from) {
		return
	}
	if isCrossAdapterImport(testPackageBasePkg(from), imp) {
		t.Errorf("IMPORT VIOLATION: %s\n\timports %s\n\treason: adapter packages must not cross-import siblings", from, imp)
	}
}

// testPackageBasePkg strips test variant suffixes from a test package path to
// recover the base production package path. Examples:
//
//	"…/adapters/http_test"        → "…/adapters/http"
//	"…/adapters/http.test"        → "…/adapters/http"
//	"…/app_test [….test]"         → "…/app"
func testPackageBasePkg(pkgPath string) string {
	// Strip synthesised variant suffix (e.g. " [github.com/…]").
	if i := strings.Index(pkgPath, " ["); i >= 0 {
		pkgPath = pkgPath[:i]
	}
	// Strip ".test" suffix (test binary package).
	pkgPath = strings.TrimSuffix(pkgPath, ".test")
	// Strip "_test" suffix (external test package).
	pkgPath = strings.TrimSuffix(pkgPath, "_test")
	return pkgPath
}

// isCrossAdapterImport reports whether from is an adapter package that imports a
// different sibling adapter package. Used by both checkRule and checkTestImportRule.
// Imports that are themselves test packages (e.g. "…_test") are skipped — a test
// binary package (e.g. "…/http.test") legitimately imports its own external test
// package (e.g. "…/http_test") and that is not a sibling adapter violation.
// crossAdapterAllowList permits specific cross-adapter imports that are
// architecturally intentional (e.g. tenant RLS middleware in http must call
// the db adapter's SetTenantContext / WithTenantTx).
var crossAdapterAllowList = map[[2]string]bool{
	{"http", "db"}: true, // tenant_middleware.go uses dbadapter.SetTenantContext / WithTenantTx
}

func isCrossAdapterImport(from, imp string) bool {
	if !strings.HasPrefix(from, pkg("internal/adapters/")) {
		return false
	}
	if !strings.HasPrefix(imp, pkg("internal/adapters/")) {
		return false
	}
	// Skip if the import is itself a test package variant.
	if isTestPackage(imp) {
		return false
	}
	fromDir := adapterDir(from)
	impDir := adapterDir(imp)
	if fromDir == impDir {
		return false
	}
	return !crossAdapterAllowList[[2]string{fromDir, impDir}]
}

// adapterDir returns the immediate subdirectory name under internal/adapters/.
// e.g. ".../internal/adapters/http/foo" → "http"
func adapterDir(pkgPath string) string {
	suffix := strings.TrimPrefix(pkgPath, pkg("internal/adapters/"))
	parts := strings.SplitN(suffix, "/", 2)
	return parts[0]
}

// TestCompositionRootExclusivity enforces that only cmd/server (the composition root)
// may import both internal/adapters and internal/app. Any other package doing so
// is wiring adapters and services outside the composition root.
//
// Test package variants are exempt: integration test files may legitimately import
// both layers as part of test setup without violating the composition root rule.
func TestCompositionRootExclusivity(t *testing.T) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, module+"/...")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.FailNow()
	}

	for _, p := range pkgs {
		// cmd/server is the legitimate composition root.
		if strings.HasPrefix(p.PkgPath, pkg("cmd/server")) {
			continue
		}
		// Test package variants are exempt from the composition root check.
		// They may import both adapters and app as part of test setup.
		// checkTestImportRule handles cross-adapter enforcement for test packages.
		if isTestPackage(p.PkgPath) {
			for imp := range p.Imports {
				checkTestImportRule(t, p.PkgPath, imp)
			}
			continue
		}

		// Adapter packages legitimately import app types (service interfaces).
		// The cross-adapter rule (Rule 2) handles sibling adapter violations.
		if strings.HasPrefix(p.PkgPath, pkg("internal/adapters/")) {
			continue
		}

		importsAdapters := false
		importsApp := false
		for imp := range p.Imports {
			if strings.HasPrefix(imp, pkg("internal/adapters")) {
				importsAdapters = true
			}
			if strings.HasPrefix(imp, pkg("internal/app")) {
				importsApp = true
			}
		}

		if importsAdapters && importsApp {
			t.Errorf("COMPOSITION ROOT VIOLATION: %s\n\timports both internal/adapters and internal/app\n\treason: only cmd/server may wire adapters and services together", p.PkgPath)
		}
	}
}

// isStdlib reports whether imp is a standard library package.
// Stdlib packages have no dot in their first path element (e.g. "context", "net/http").
func isStdlib(imp string) bool {
	first := strings.SplitN(imp, "/", 2)[0]
	return !strings.Contains(first, ".")
}
