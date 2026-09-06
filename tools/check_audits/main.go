// Command check_audits rejects drift between the dependency and Field Notes
// citation inventories and the repository inputs from which they are derived.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	goNoticePattern   = `(?m)^\| Go \| \x60([^\x60]+)\x60 \| \x60([^\x60]+)\x60 \|`
	npmNoticePattern  = `(?m)^\| npm \| \x60([^\x60]+)\x60 \| \x60([^\x60]+)\x60 \|`
	citationPattern   = `([A-Z][A-Za-z'’.-]+( et al\.| & [A-Z][A-Za-z'’.-]+)?) \([12][0-9]{3}\)`
	citationMarker    = `(?m)^<!-- field-note-citation: (.+) -->$`
	binaryenNoticeRow = "| Tool | `Binaryen` | `132` |"
)

type module struct {
	Path    string
	Version string
	Main    bool
}

type packageLock struct {
	Packages map[string]struct {
		Version string `json:"version"`
		License string `json:"license"`
	} `json:"packages"`
}

func main() {
	root, err := repositoryRoot()
	if err != nil {
		fail(err)
	}

	notices, err := os.ReadFile(filepath.Join(root, "THIRD_PARTY_NOTICES.md"))
	if err != nil {
		fail(err)
	}
	goModules, err := checkGoModules(root, notices)
	if err != nil {
		fail(err)
	}
	if err := checkNPMModules(root, notices); err != nil {
		fail(err)
	}
	if !strings.Contains(string(notices), binaryenNoticeRow) {
		fail(errors.New("THIRD_PARTY_NOTICES.md does not enumerate pinned Binaryen 132"))
	}
	if !strings.Contains(string(notices), "## Bundled Go Regular font") {
		fail(errors.New("THIRD_PARTY_NOTICES.md does not include the bundled Go Regular font license"))
	}
	fontSource := fmt.Sprintf("gofont/ttfs/README` at `%s`", goModules["golang.org/x/image"])
	if !strings.Contains(string(notices), fontSource) {
		fail(fmt.Errorf("bundled Go Regular license does not identify %s", fontSource))
	}
	if !strings.Contains(string(notices), "## Bundled Noto Emoji subset") {
		fail(errors.New("THIRD_PARTY_NOTICES.md does not include the bundled Noto Emoji subset license"))
	}
	for _, required := range [...]string{
		"### Apache License 2.0",
		"### `github.com/rivo/uniseg` — MIT License",
		"### `github.com/jezek/xgb` — BSD-3-Clause with patent grant",
		"### `golang.org/x/image`, `x/sync`, `x/sys`, and `x/text` — BSD-3-Clause",
		"### `github.com/go-text/typesetting` — Unlicense OR BSD-3-Clause",
		"### Noto Emoji — SIL Open Font License 1.1",
	} {
		if !strings.Contains(string(notices), required) {
			fail(fmt.Errorf("THIRD_PARTY_NOTICES.md is missing required license section %q", required))
		}
	}
	if err := checkFieldNoteCitations(root); err != nil {
		fail(err)
	}

	fmt.Println("citation and dependency-license audits match their repository inputs")
}

func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not find repository root containing go.mod")
		}
		dir = parent
	}
}

func checkGoModules(root string, notices []byte) (map[string]string, error) {
	command := exec.Command("go", "list", "-m", "-json", "all")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list Go module graph: %w", err)
	}

	actual := make(map[string]string)
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for decoder.More() {
		var dependency module
		if err := decoder.Decode(&dependency); err != nil {
			return nil, fmt.Errorf("decode Go module graph: %w", err)
		}
		if !dependency.Main {
			actual[dependency.Path] = dependency.Version
		}
	}

	documented, err := parseInventory(notices, goNoticePattern, "Go module")
	if err != nil {
		return nil, err
	}
	if err := compareInventory("Go module", actual, documented); err != nil {
		return nil, err
	}
	return actual, nil
}

func checkNPMModules(root string, notices []byte) error {
	contents, err := os.ReadFile(filepath.Join(root, "tools", "web-e2e", "package-lock.json"))
	if err != nil {
		return err
	}
	var lock packageLock
	if err := json.Unmarshal(contents, &lock); err != nil {
		return fmt.Errorf("decode package-lock.json: %w", err)
	}

	actual := make(map[string]string)
	for path, dependency := range lock.Packages {
		if path == "" {
			continue
		}
		name, found := strings.CutPrefix(path, "node_modules/")
		if !found || name == "" {
			return fmt.Errorf("unsupported package-lock path %q", path)
		}
		actual[name] = dependency.Version
	}

	documented, err := parseInventory(notices, npmNoticePattern, "npm package")
	if err != nil {
		return err
	}
	return compareInventory("npm package", actual, documented)
}

func checkFieldNoteCitations(root string) error {
	source, err := os.ReadFile(filepath.Join(root, "pkg", "ui", "field_notes.go"))
	if err != nil {
		return err
	}
	audit, err := os.ReadFile(filepath.Join(root, "docs", "CITATIONS.md"))
	if err != nil {
		return err
	}

	actual := uniqueMatches(source, citationPattern, 0)
	documented := uniqueMatches(audit, citationMarker, 1)
	return compareSet("Field Notes citation", actual, documented)
}

func parseInventory(contents []byte, pattern, kind string) (map[string]string, error) {
	matches := regexp.MustCompile(pattern).FindAllSubmatch(contents, -1)
	result := make(map[string]string, len(matches))
	for _, match := range matches {
		name := string(match[1])
		version := string(match[2])
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("duplicate %s inventory row for %s", kind, name)
		}
		result[name] = version
	}
	return result, nil
}

func uniqueMatches(contents []byte, pattern string, group int) map[string]struct{} {
	matches := regexp.MustCompile(pattern).FindAllSubmatch(contents, -1)
	result := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		result[string(match[group])] = struct{}{}
	}
	return result
}

func compareInventory(kind string, actual, documented map[string]string) error {
	var problems []string
	for name, version := range actual {
		documentedVersion, found := documented[name]
		switch {
		case !found:
			problems = append(problems, fmt.Sprintf("missing %s %s %s", kind, name, version))
		case documentedVersion != version:
			problems = append(problems, fmt.Sprintf("%s %s is %s, documented as %s", kind, name, version, documentedVersion))
		}
	}
	for name, version := range documented {
		if _, found := actual[name]; !found {
			problems = append(problems, fmt.Sprintf("stale %s %s %s", kind, name, version))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return errors.New(strings.Join(problems, "\n"))
}

func compareSet(kind string, actual, documented map[string]struct{}) error {
	var problems []string
	for name := range actual {
		if _, found := documented[name]; !found {
			problems = append(problems, fmt.Sprintf("missing %s %s", kind, name))
		}
	}
	for name := range documented {
		if _, found := actual[name]; !found {
			problems = append(problems, fmt.Sprintf("stale %s %s", kind, name))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return errors.New(strings.Join(problems, "\n"))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "audit check:", err)
	os.Exit(1)
}
