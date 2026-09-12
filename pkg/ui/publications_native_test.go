//go:build !js

package ui

import (
	"os"
	"regexp"
	"testing"
)

func TestPublicationDestinationsMatchReviewedBibliography(t *testing.T) {
	doc, err := os.ReadFile("../../docs/CITATIONS.md")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`(?s)<!-- field-note-citation: (.*?) -->.*?\]\((https://doi.org/[^)]+)\)`)
	matches := pattern.FindAllStringSubmatch(string(doc), -1)
	if len(matches) != len(publicationLinks) {
		t.Fatalf("bibliography has %d publications, link catalog has %d", len(matches), len(publicationLinks))
	}
	for _, match := range matches {
		want := "[link=" + match[2] + "]" + match[1] + "[/link]"
		if got := FieldNoteReferenceMarkup(match[1]); got != want {
			t.Errorf("bibliography mismatch: got %q, want %q", got, want)
		}
	}
}
