package gameapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

// declaredErrorCodeNames reads errors.go and returns the identifier of every
// constant declared with type ErrorCode, in source order. Enumerating the
// contract from the source is what makes ErrorCodes derived rather than a
// second hand-maintained list sitting next to the first.
func declaredErrorCodeNames(t *testing.T) []string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the error contract test")
	}
	path := filepath.Join(filepath.Dir(thisFile), "errors.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse errors.go: %v", err)
	}
	var names []string
	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || general.Tok != token.CONST {
			continue
		}
		for _, spec := range general.Specs {
			value, isValue := spec.(*ast.ValueSpec)
			if !isValue {
				continue
			}
			identifier, isIdentifier := value.Type.(*ast.Ident)
			if !isIdentifier || identifier.Name != "ErrorCode" {
				continue
			}
			for _, name := range value.Names {
				names = append(names, name.Name)
			}
		}
	}
	if len(names) == 0 {
		t.Fatal("no ErrorCode constants found; the parser and the source have diverged")
	}
	return names
}

func TestErrorCodesNamesEveryDeclaredCode(t *testing.T) {
	declared := declaredErrorCodeNames(t)
	listed := ErrorCodes()
	if len(listed) != len(declared) {
		t.Fatalf("ErrorCodes returns %d codes, errors.go declares %d (%v)", len(listed), len(declared), declared)
	}
	// Constant identifiers are compile-time-checked in the slice literal, so
	// comparing values against the source order catches an omission, a
	// duplicate, and a reordering alike.
	seen := make(map[ErrorCode]int, len(listed))
	for index, code := range listed {
		if previous, duplicate := seen[code]; duplicate {
			t.Fatalf("ErrorCodes repeats %q at positions %d and %d", code, previous, index)
		}
		seen[code] = index
		if code == "" {
			t.Fatalf("ErrorCodes position %d is the zero value", index)
		}
	}
}

func TestErrorCodeValuesAreDistinctAndStable(t *testing.T) {
	// The wire form of these codes crosses the port and is matched by adapters
	// and player copy, so a collision would silently merge two failures.
	byValue := make(map[ErrorCode]bool, len(errorCodes))
	for _, code := range errorCodes {
		if byValue[code] {
			t.Fatalf("duplicate error code value %q", code)
		}
		byValue[code] = true
	}
}

func TestErrorCodesReturnsACopy(t *testing.T) {
	first := ErrorCodes()
	if len(first) == 0 {
		t.Fatal("no error codes")
	}
	original := first[0]
	first[0] = "mutated"
	if second := ErrorCodes(); second[0] != original {
		t.Fatalf("caller mutated the shared contract: %q", second[0])
	}
}
