package archtest

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/adsouza/africa2ice"

func TestSourceDependencyBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", relative, err)
		}
		for _, imported := range file.Imports {
			pathValue, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if reason := importViolation(filepath.ToSlash(relative), pathValue); reason != "" {
				violations = append(violations, relative+": import "+pathValue+": "+reason)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(violations)
	if len(violations) != 0 {
		t.Fatalf("architecture violations:\n%s", strings.Join(violations, "\n"))
	}
}

func importViolation(file, imported string) string {
	category := packageCategory(file)
	if (imported == "log" || imported == "log/slog") && category != "logging" {
		return "operational logging is confined to internal/adapters/logging"
	}
	if strings.HasPrefix(imported, module+"/internal/domain") && category != "domain" && category != "application" {
		return "only application may import the domain"
	}
	if imported == module+"/pkg/gameapi" && category == "domain" && file != "internal/domain/policy.go" {
		return "only domain/policy.go may import the shared boundary policy"
	}
	if strings.HasPrefix(imported, "math/rand") && category == "domain" && !strings.HasSuffix(file, "/rng.go") {
		return "domain randomness is confined to rng.go"
	}
	if category == "domain" && !strings.HasSuffix(file, "_test.go") && isStandardLibrary(imported) {
		switch imported {
		case "errors", "fmt", "math", "math/rand/v2", "sort", "sync":
		default:
			return "production domain standard-library imports require an explicit purity review"
		}
	}
	allowed, strict := allowedImports(category)
	if !strict || isStandardLibrary(imported) {
		return ""
	}
	for _, prefix := range allowed {
		if imported == prefix || strings.HasPrefix(imported, prefix+"/") {
			return ""
		}
	}
	return "not in the package's strict dependency allowlist"
}

func packageCategory(file string) string {
	switch {
	case strings.HasPrefix(file, "pkg/gameapi/"):
		return "gameapi"
	case strings.HasPrefix(file, "internal/domain/"):
		return "domain"
	case strings.HasPrefix(file, "internal/application/"):
		return "application"
	case strings.HasPrefix(file, "internal/adapters/storage/"):
		return "storage"
	case strings.HasPrefix(file, "internal/adapters/logging/"):
		return "logging"
	case strings.HasPrefix(file, "internal/verification/"):
		return "verification"
	case strings.HasPrefix(file, "pkg/render/"):
		return "render"
	case strings.HasPrefix(file, "pkg/ui/"):
		return "ui"
	case strings.HasPrefix(file, "pkg/audio/"):
		return "audio"
	case strings.HasPrefix(file, "pkg/hud/"):
		return "hud"
	case strings.HasPrefix(file, "pkg/app/"):
		return "app"
	default:
		return "other"
	}
}

func allowedImports(category string) ([]string, bool) {
	switch category {
	case "gameapi":
		return nil, true
	case "domain":
		return []string{module + "/pkg/gameapi"}, true
	case "application":
		return []string{module + "/internal/domain", module + "/pkg/gameapi"}, true
	case "storage":
		return []string{module + "/internal/application", "golang.org/x/sys/windows"}, true
	case "logging":
		return []string{module + "/internal/application", module + "/pkg/gameapi", module + "/pkg/ui"}, true
	case "verification":
		return []string{module + "/internal/application", module + "/pkg/gameapi"}, true
	case "render":
		return []string{module + "/pkg/gameapi", "github.com/hajimehoshi/ebiten/v2", "golang.org/x/image"}, true
	case "ui":
		return []string{module + "/pkg/gameapi", module + "/pkg/render", module + "/pkg/audio", "github.com/hajimehoshi/ebiten/v2"}, true
	case "audio":
		return []string{"github.com/ebitengine/oto/v3"}, true
	case "hud":
		return []string{module + "/pkg/gameapi", module + "/pkg/ui", module + "/pkg/render", "github.com/hajimehoshi/ebiten/v2", "github.com/ebitenui/ebitenui", "golang.org/x/image"}, true
	case "app":
		return []string{module + "/internal/application", module + "/internal/adapters/logging", module + "/internal/adapters/storage", module + "/pkg/gameapi", module + "/pkg/render", module + "/pkg/ui", module + "/pkg/hud", module + "/pkg/audio", "github.com/hajimehoshi/ebiten/v2", "github.com/ebitenui/ebitenui"}, true
	default:
		return nil, false
	}
}

func TestVerificationDependencyAllowlist(t *testing.T) {
	file := "internal/verification/checkpoint.go"
	tests := []struct {
		name     string
		imported string
		allowed  bool
	}{
		{name: "standard library", imported: "encoding/json", allowed: true},
		{name: "application port", imported: module + "/internal/application", allowed: true},
		{name: "public game API", imported: module + "/pkg/gameapi", allowed: true},
		{name: "domain bypass", imported: module + "/internal/domain", allowed: false},
		{name: "storage adapter", imported: module + "/internal/adapters/storage", allowed: false},
		{name: "render adapter", imported: module + "/pkg/render", allowed: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			violation := importViolation(file, test.imported)
			if test.allowed && violation != "" {
				t.Fatalf("allowed import %q rejected: %s", test.imported, violation)
			}
			if !test.allowed && violation == "" {
				t.Fatalf("disallowed import %q accepted", test.imported)
			}
		})
	}
}

func TestHUDCategoryAllowsEbitenUIAndNothingElseDoes(t *testing.T) {
	ebitenui := "github.com/ebitenui/ebitenui/widget"
	if violation := importViolation("pkg/hud/panel.go", ebitenui); violation != "" {
		t.Fatalf("hud may import ebitenui: %s", violation)
	}
	if violation := importViolation("pkg/hud/panel.go", module+"/pkg/ui"); violation != "" {
		t.Fatalf("hud may import ui: %s", violation)
	}
	if violation := importViolation("pkg/app/game.go", "github.com/ebitenui/ebitenui/input"); violation != "" {
		t.Fatalf("app may read ebitenui input state: %s", violation)
	}
	for _, file := range []string{"pkg/render/map.go", "pkg/ui/bands.go", "internal/application/service.go"} {
		if importViolation(file, ebitenui) == "" {
			t.Fatalf("%s accepted an ebitenui import", file)
		}
	}
	if importViolation("pkg/hud/panel.go", module+"/internal/application") == "" {
		t.Fatal("hud accepted an application import")
	}
}

func TestDomainSharedPolicyImportConfinement(t *testing.T) {
	imported := module + "/pkg/gameapi"
	if violation := importViolation("internal/domain/policy.go", imported); violation != "" {
		t.Fatalf("domain policy import rejected: %s", violation)
	}
	if violation := importViolation("internal/domain/turn.go", imported); violation == "" {
		t.Fatal("gameapi import outside domain policy file was accepted")
	}
}

func isStandardLibrary(imported string) bool {
	first, _, _ := strings.Cut(imported, "/")
	return !strings.Contains(first, ".")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate architecture test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestDomainHasNoBuildTags(t *testing.T) {
	root := filepath.Join(repositoryRoot(t), "internal", "domain")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		file, err := os.Open(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "//go:build") || strings.HasPrefix(line, "// +build") {
				violations := entry.Name()
				_ = file.Close()
				t.Fatalf("domain file has a build tag: %s", violations)
			}
			if line != "" && !strings.HasPrefix(line, "//") {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// Standard-library status alone does not make an import safe for the domain:
// clocks, filesystem access, and network clients would break deterministic isolation.
func TestDomainStandardLibraryPurityBoundary(t *testing.T) {
	for _, imported := range []string{"os", "os/exec", "io/fs", "net/http", "time", "syscall", "unsafe", "encoding/json"} {
		if importViolation("internal/domain/world.go", imported) == "" {
			t.Errorf("production domain accepted %q", imported)
		}
	}
	for _, imported := range []string{"errors", "fmt", "math", "sort", "sync"} {
		if reason := importViolation("internal/domain/world.go", imported); reason != "" {
			t.Errorf("pure dependency %q rejected: %s", imported, reason)
		}
	}
	if reason := importViolation("internal/domain/world_test.go", "time"); reason != "" {
		t.Fatalf("test clocks must remain available: %s", reason)
	}
}
