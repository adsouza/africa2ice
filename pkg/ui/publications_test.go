package ui

import (
	"strings"
	"testing"
)

func TestReferenceLinksPreserveNonAcademicText(t *testing.T) {
	const prose = "Game model; see DESIGN §7."
	if got := FieldNoteReferenceMarkup(prose); got != prose {
		t.Fatalf("nonacademic reference changed: %q", got)
	}
	for _, publication := range publicationLinks {
		got := FieldNoteReferenceMarkup(publication.citation + "; " + prose)
		want := "[link=" + publication.url + "]" + publication.citation + "[/link]; " + prose
		if got != want || !IsPublicationURL(publication.url) {
			t.Fatalf("unlinked publication: %q", got)
		}
	}
	got := FieldNoteReferenceMarkup(CampaignOverviewFieldNote().References)
	if strings.Count(got, "[link=") != 2 {
		t.Fatalf("welcome references = %q", got)
	}
	for _, url := range []string{"", "https://example.com", "javascript:alert(1)", publicationLinks[0].url + "?extra=1"} {
		if IsPublicationURL(url) {
			t.Fatalf("accepted URL outside publication catalog: %q", url)
		}
	}
}
