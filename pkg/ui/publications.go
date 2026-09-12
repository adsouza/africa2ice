package ui

import "strings"

// publicationLinks mirrors the reviewed DOI destinations in docs/CITATIONS.md.
// Keep citation prose plain in FieldNote; link markup belongs only to the HUD.
var publicationLinks = [...]struct{ citation, url string }{
	{"Scholz et al. (2007)", "https://doi.org/10.1073/pnas.0703874104"},
	{"Eccles (1974)", "https://doi.org/10.4319/lo.1974.19.5.0730"},
	{"Cohen et al. (2007)", "https://doi.org/10.1073/pnas.0703873104"},
	{"Bartov et al. (2002)", "https://doi.org/10.1006/qres.2001.2284"},
	{"Reich et al. (2010)", "https://doi.org/10.1038/nature09710"},
	{"Chen et al. (2019)", "https://doi.org/10.1038/s41586-019-1139-x"},
	{"Dannemann et al. (2016)", "https://doi.org/10.1016/j.ajhg.2015.11.015"},
	{"Jablonski & Chaplin (2010)", "https://doi.org/10.1073/pnas.0914628107"},
	{"Yellen et al. (1995)", "https://doi.org/10.1126/science.7725100"},
	{"O'Connor et al. (2011)", "https://doi.org/10.1126/science.1207703"},
	{"McNiven et al. (2012)", "https://doi.org/10.1016/j.jas.2011.09.007"},
	{"Clarkson et al. (2017)", "https://doi.org/10.1038/nature22968"},
	{"Lisiecki & Raymo (2005)", "https://doi.org/10.1029/2004PA001071"},
	{"Clark et al. (2009)", "https://doi.org/10.1126/science.1172873"},
	{"Rasmussen et al. (2014)", "https://doi.org/10.1016/j.quascirev.2014.09.007"},
	{"Capron et al. (2021)", "https://doi.org/10.1038/s41467-021-22241-w"},
	{"Giaccio et al. (2017)", "https://doi.org/10.1038/srep45940"},
	{"Silleni et al. (2020)", "https://doi.org/10.3389/feart.2020.543399"},
	{"Smith et al. (2016)", "https://doi.org/10.1007/s00445-016-1037-0"},
	{"Pyle et al. (2006)", "https://doi.org/10.1016/j.quascirev.2006.06.008"},
	{"Storey et al. (2012)", "https://doi.org/10.1073/pnas.1208178109"},
	{"Lane et al. (2013)", "https://doi.org/10.1073/pnas.1301474110"},
	{"Kappelman et al. (2024)", "https://doi.org/10.1038/s41586-024-07208-3"},
}

// FieldNoteReferenceMarkup links academic citations while leaving game-model
// and DESIGN references as plain text.
func FieldNoteReferenceMarkup(references string) string {
	for _, publication := range publicationLinks {
		references = strings.ReplaceAll(references, publication.citation,
			"[link="+publication.url+"]"+publication.citation+"[/link]")
	}
	return references
}

// IsPublicationURL restricts external navigation to the publication catalog.
func IsPublicationURL(url string) bool {
	for _, publication := range publicationLinks {
		if publication.url == url {
			return true
		}
	}
	return false
}
