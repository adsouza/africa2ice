package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func repositoryRootForTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func realNotices(t *testing.T) []byte {
	t.Helper()
	notices, err := os.ReadFile(filepath.Join(repositoryRootForTest(t), "THIRD_PARTY_NOTICES.md"))
	if err != nil {
		t.Fatal(err)
	}
	return notices
}

func TestRepositoryAuditPasses(t *testing.T) {
	if err := audit(repositoryRootForTest(t)); err != nil {
		t.Fatalf("audit of the checked-in repository failed: %v", err)
	}
}

// Every presence check in the notices audit must be able to fail: removing
// any one required line from the real notices is reported, by name.
func TestNoticeChecksRejectEachMissingRequirement(t *testing.T) {
	root := repositoryRootForTest(t)
	notices := realNotices(t)
	goModules, err := checkGoModules(root, notices)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkNotices(root, notices, goModules); err != nil {
		t.Fatalf("real notices rejected: %v", err)
	}
	for _, requirement := range noticeRequirements(goModules) {
		required := requirement.line
		t.Run(required, func(t *testing.T) {
			if !strings.Contains(string(notices), required) {
				t.Fatalf("real notices lack %q; the requirement list is stale", required)
			}
			stripped := []byte(strings.ReplaceAll(string(notices), required, ""))
			if err := checkNotices(root, stripped, goModules); err == nil {
				t.Fatalf("notices without %q were accepted", required)
			}
		})
	}
}

// A dependency bump the notices miss must fail the audit and name the module.
func TestGoModuleAuditRejectsAnUndocumentedVersion(t *testing.T) {
	root := repositoryRootForTest(t)
	notices := realNotices(t)
	row := regexp.MustCompile(goNoticePattern).FindSubmatch(notices)
	if row == nil {
		t.Fatal("no Go module rows in the notices")
	}
	module, version := string(row[1]), string(row[2])
	altered := []byte(strings.Replace(string(notices), "`"+module+"` | `"+version+"`", "`"+module+"` | `v0.0.0-altered`", 1))
	_, err := checkGoModules(root, altered)
	if err == nil || !strings.Contains(err.Error(), module) {
		t.Fatalf("err = %v, want a mismatch naming %s", err, module)
	}
}

func TestCompareInventoryReportsMissingChangedAndStale(t *testing.T) {
	err := compareInventory("Go module",
		map[string]string{"a": "v1", "b": "v2", "c": "v3"},
		map[string]string{"a": "v1", "b": "v1", "d": "v4"})
	if err == nil {
		t.Fatal("drifted inventory accepted")
	}
	for _, want := range []string{"missing Go module c v3", "Go module b is v2, documented as v1", "stale Go module d v4"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %v, want it to report %q", err, want)
		}
	}
	if err := compareInventory("Go module", map[string]string{"a": "v1"}, map[string]string{"a": "v1"}); err != nil {
		t.Fatalf("matching inventory rejected: %v", err)
	}
}

func TestParseInventoryRejectsDuplicateRows(t *testing.T) {
	rows := []byte("| Go | `a` | `v1` |\n| Go | `a` | `v2` |\n")
	if _, err := parseInventory(rows, goNoticePattern, "Go module"); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err = %v, want a duplicate-row error", err)
	}
}

func TestCompareSetReportsMissingAndStale(t *testing.T) {
	err := compareSet("Field Notes citation", map[string]struct{}{"A (2001)": {}}, map[string]struct{}{"B (2002)": {}})
	if err == nil || !strings.Contains(err.Error(), "missing Field Notes citation A (2001)") || !strings.Contains(err.Error(), "stale Field Notes citation B (2002)") {
		t.Fatalf("err = %v", err)
	}
}

func TestCitationPatternMatchesFieldNoteForms(t *testing.T) {
	source := []byte(`Reich et al. (2010); Chen & Smith (2019); O’Connell (2006); version 2 (not a citation)`)
	got := uniqueMatches(source, citationPattern, 0)
	for _, want := range []string{"Reich et al. (2010)", "Chen & Smith (2019)", "O’Connell (2006)"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("citations %v missing %q", got, want)
		}
	}
	if len(got) != 3 {
		t.Fatalf("citations = %v, want exactly three", got)
	}
}

// Windows checkouts convert non-Go text files to CRLF, and Go's multiline $
// matches only before \n. A marker pattern anchored at $ then found no
// citations at all on Windows, reporting every one as missing.
func TestCitationMarkersParseWithEitherLineEnding(t *testing.T) {
	for name, audit := range map[string]string{
		"LF":   "<!-- field-note-citation: Reich et al. (2010) -->\n<!-- field-note-citation: Chen & Smith (2019) -->\n",
		"CRLF": "<!-- field-note-citation: Reich et al. (2010) -->\r\n<!-- field-note-citation: Chen & Smith (2019) -->\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			got := uniqueMatches([]byte(audit), citationMarker, 1)
			_, reich := got["Reich et al. (2010)"]
			_, chen := got["Chen & Smith (2019)"]
			if len(got) != 2 || !reich || !chen {
				t.Fatalf("markers = %q, want exactly the two citations", got)
			}
		})
	}
}
