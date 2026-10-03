package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// panicGuardExempt lists production files whose goroutines run outside any
// session, so there is no log for a guard to write to. Each entry needs a reason.
var panicGuardExempt = map[string]string{
	"internal/application/balance.go": "calibration workers run only inside tools/report_moisture_balance",
}

// DESIGN.md §3: the entrypoint's deferred guard cannot observe a panic on
// another goroutine, so every project-owned goroutine and JavaScript callback
// must itself defer the session's panic hook. This holds future goroutines to
// the rule as well as today's.
func TestOwnedGoroutinesAndCallbacksDeferAPanicGuard(t *testing.T) {
	violations := ownedConcurrencyViolations(t, repositoryRoot(t))
	if len(violations) != 0 {
		t.Fatalf("goroutines or JavaScript callbacks without `defer panicGuard()`:\n%s", strings.Join(violations, "\n"))
	}
}

// The rule must be able to fail, or a clean result proves nothing.
func TestPanicGuardRuleRejectsUnguardedConcurrency(t *testing.T) {
	directory := t.TempDir()
	source := `package example

import "syscall/js"

type worker struct{ panicGuard func() }

func (w *worker) run()     {}
func (w *worker) guarded() { defer w.panicGuard() }

func start(w *worker) {
	go func() {}()
	go w.run()
	go w.guarded()
	go func() { defer w.panicGuard() }()
	js.FuncOf(func(js.Value, []js.Value) any { return nil })
	js.FuncOf(func(js.Value, []js.Value) any { defer w.panicGuard(); return nil })
}
`
	if err := os.MkdirAll(filepath.Join(directory, "pkg", "example"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "pkg", "example", "example.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := len(ownedConcurrencyViolations(t, directory)); got != 3 {
		t.Fatalf("rule reported %d violations, want 3: two unguarded goroutines and one unguarded callback", got)
	}
}

func ownedConcurrencyViolations(t *testing.T, root string) []string {
	t.Helper()
	packages := map[string][]*ast.File{}
	positions := token.NewFileSet()
	for _, tree := range []string{"internal", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return walkErr
			}
			relative, _ := filepath.Rel(root, path)
			if _, exempt := panicGuardExempt[filepath.ToSlash(relative)]; exempt {
				return nil
			}
			file, err := parser.ParseFile(positions, path, nil, 0)
			if err != nil {
				return err
			}
			packages[filepath.Dir(path)] = append(packages[filepath.Dir(path)], file)
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	var violations []string
	report := func(node ast.Node, what string) {
		position := positions.Position(node.Pos())
		relative, _ := filepath.Rel(root, position.Filename)
		violations = append(violations, filepath.ToSlash(relative)+":"+itoa(position.Line)+": "+what)
	}
	for _, files := range packages {
		declared := map[string]*ast.FuncDecl{}
		for _, file := range files {
			for _, decl := range file.Decls {
				if function, ok := decl.(*ast.FuncDecl); ok && function.Body != nil {
					declared[function.Name.Name] = function
				}
			}
		}
		for _, file := range files {
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.GoStmt:
					if !goroutineDefersGuard(node.Call.Fun, declared) {
						report(node, "goroutine")
					}
				case *ast.CallExpr:
					if isJSFuncOf(node) && (len(node.Args) != 1 || !literalDefersGuard(node.Args[0])) {
						report(node, "js.FuncOf callback")
					}
				}
				return true
			})
		}
	}
	sort.Strings(violations)
	return violations
}

func goroutineDefersGuard(target ast.Expr, declared map[string]*ast.FuncDecl) bool {
	switch target := target.(type) {
	case *ast.FuncLit:
		return bodyDefersGuard(target.Body)
	case *ast.Ident:
		function, ok := declared[target.Name]
		return ok && bodyDefersGuard(function.Body)
	case *ast.SelectorExpr:
		function, ok := declared[target.Sel.Name]
		return ok && bodyDefersGuard(function.Body)
	}
	return false
}

func literalDefersGuard(expression ast.Expr) bool {
	literal, ok := expression.(*ast.FuncLit)
	return ok && bodyDefersGuard(literal.Body)
}

// bodyDefersGuard reports a top-level `defer panicGuard()` or
// `defer x.panicGuard()` (any receiver) in the function body.
func bodyDefersGuard(body *ast.BlockStmt) bool {
	for _, statement := range body.List {
		deferred, ok := statement.(*ast.DeferStmt)
		if !ok || len(deferred.Call.Args) != 0 {
			continue
		}
		switch fun := deferred.Call.Fun.(type) {
		case *ast.Ident:
			if fun.Name == "panicGuard" {
				return true
			}
		case *ast.SelectorExpr:
			if fun.Sel.Name == "panicGuard" {
				return true
			}
		}
	}
	return false
}

func isJSFuncOf(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "FuncOf" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "js"
}
