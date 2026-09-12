# Field Notes citation audit

This is the reviewed bibliography for compact author/year references emitted by
`pkg/ui/field_notes.go`. The audit checks whether each cited source supports the historical or
scientific context beside which it appears; it does **not** treat a source as validation of the
game's numerical rules. Those rules are identified in Field Notes as game abstractions and are
specified in `DESIGN.md`.

The source file and this bibliography are kept in exact lockstep by
`go run ./tools/check_audits`: every author/year citation in Field Notes must have one marker below,
and stale bibliography markers fail the check. Titles, authorship, year, venue, and stable links
were reviewed on 2026-09-01 against publisher, DOI, or institutional-repository records.

Academic citations in the Field Notes Sources line are clickable author/year links. Their DOI
destinations are cataloged in `pkg/ui/publications.go` and checked against this bibliography by
the UI tests. Desktop builds open the system's default browser; the web build opens a new tab.
Game-model and internal DESIGN references remain plain text.

## Human dispersal, archaic ancestry, and adaptation

<!-- field-note-citation: Reich et al. (2010) -->
### Reich et al. (2010)

“Genetic history of an archaic hominin group from Denisova Cave in Siberia.” *Nature* 468.
[doi:10.1038/nature09710](https://doi.org/10.1038/nature09710).

Field Notes use: Denisovan identification and archaic ancestry/admixture context. The simulation's
bands, explicit interbreeding action, and reciprocal trait-vector exchange are game abstractions.

<!-- field-note-citation: Chen et al. (2019) -->
### Chen et al. (2019)

“A late Middle Pleistocene Denisovan mandible from the Tibetan Plateau.” *Nature* 569.
[doi:10.1038/s41586-019-1139-x](https://doi.org/10.1038/s41586-019-1139-x).

Field Notes use: Denisovan presence on the Tibetan Plateau. The exact starting band and Baishiya
Karst tile are a scenario abstraction rather than a demographic reconstruction.

<!-- field-note-citation: Dannemann et al. (2016) -->
### Dannemann et al. (2016)

“Introgression of Neandertal- and Denisovan-like Haplotypes Contributes to Adaptive Variation in
Human Toll-like Receptors.” *The American Journal of Human Genetics* 98.
[doi:10.1016/j.ajhg.2015.11.015](https://doi.org/10.1016/j.ajhg.2015.11.015).

Field Notes use: introgressed immune variation. The scalar immune trait, selection response, and
whole-vector exchange are deliberately compressed mechanics.

<!-- field-note-citation: Jablonski & Chaplin (2010) -->
### Jablonski & Chaplin (2010)

“Human skin pigmentation as an adaptation to UV radiation.” *Proceedings of the National Academy
of Sciences* 107 (Supplement 2).
[doi:10.1073/pnas.0914628107](https://doi.org/10.1073/pnas.0914628107).

Field Notes use: the UV-protection/vitamin-D trade-off and its geographic selection pressure. The
single pigmentation value and latitude-driven game function are abstractions.

## Aquatic subsistence and Sahul

<!-- field-note-citation: Yellen et al. (1995) -->
### Yellen et al. (1995)

“A Middle Stone Age Worked Bone Industry from Katanda, Upper Semliki Valley, Zaire.” *Science* 268.
[doi:10.1126/science.7725100](https://doi.org/10.1126/science.7725100).

Field Notes use: worked bone points and fish remains at Katanda. The generic Cordage and Nets
technology bundles several capabilities and does not assign the site a modern fishing system.

<!-- field-note-citation: O'Connor et al. (2011) -->
### O'Connor et al. (2011)

“Pelagic Fishing at 42,000 Years Before the Present and the Maritime Skills of Modern Humans.”
*Science* 334. [doi:10.1126/science.1207703](https://doi.org/10.1126/science.1207703).

Field Notes use: Jerimalai pelagic catch near 42 ka and later shell-hook evidence. Named passages,
navigation prerequisites, and fishing yields are game abstractions, not inferred boat designs.

<!-- field-note-citation: McNiven et al. (2012) -->
### McNiven et al. (2012)

“Dating Aboriginal stone-walled fishtraps at Lake Condah, southeast Australia.” *Journal of
Archaeological Science* 39.
[doi:10.1016/j.jas.2011.09.007](https://doi.org/10.1016/j.jas.2011.09.007).

Field Notes use: the much later date of the Lake Condah stone-walled traps, expressly to prevent the
game's generic trap/weir improvement from being read as campaign-period Australian evidence.

<!-- field-note-citation: Clarkson et al. (2017) -->
### Clarkson et al. (2017)

“Human occupation of northern Australia by 65,000 years ago.” *Nature* 547.
[doi:10.1038/nature22968](https://doi.org/10.1038/nature22968).

Field Notes use: early occupation evidence for Sahul. The game's two Wallacea routes and their
opening requirements are route abstractions rather than claims about one documented crossing.

## Long-term and abrupt climate context

<!-- field-note-citation: Lisiecki & Raymo (2005) -->
### Lisiecki & Raymo (2005)

“A Pliocene-Pleistocene stack of 57 globally distributed benthic δ18O records.” *Paleoceanography*
20. [doi:10.1029/2004PA001071](https://doi.org/10.1029/2004PA001071).

Field Notes use: marine-isotope-stage context for the campaign's long climate trend. The three
named moisture epochs and their thresholds are compressed game summaries.

<!-- field-note-citation: Clark et al. (2009) -->
### Clark et al. (2009)

“The Last Glacial Maximum.” *Science* 325.
[doi:10.1126/science.1172873](https://doi.org/10.1126/science.1172873).

Field Notes use: Last Glacial Maximum timing and glacial context. Tile biomes and the Beringia gate
are deterministic game functions, not direct reconstructions from this paper.

<!-- field-note-citation: Rasmussen et al. (2014) -->
### Rasmussen et al. (2014)

“A stratigraphic framework for abrupt climatic changes during the Last Glacial period based on
three synchronized Greenland ice-core records: refining and extending the INTIMATE event
stratigraphy.” *Quaternary Science Reviews* 106.
[doi:10.1016/j.quascirev.2014.09.007](https://doi.org/10.1016/j.quascirev.2014.09.007).

Field Notes use: Greenland-interstadial onset chronology. Regional pulse shapes, amplitudes, and
moisture effects are deterministic gameplay overlays.

<!-- field-note-citation: Capron et al. (2021) -->
### Capron et al. (2021)

“The anatomy of past abrupt warmings recorded in Greenland ice.” *Nature Communications* 12.
[doi:10.1038/s41467-021-22241-w](https://doi.org/10.1038/s41467-021-22241-w).

Field Notes use: abrupt warmings differed in timing and shape. The game does not claim that its one
regional overlay is a reconstruction of each recorded event.

## Campanian Ignimbrite

<!-- field-note-citation: Giaccio et al. (2017) -->
### Giaccio et al. (2017)

“High-precision 14C and 40Ar/39Ar dating of the Campanian Ignimbrite (Y-5) reconciles the time-scales
of climatic-cultural processes at 40 ka.” *Scientific Reports* 7.
[doi:10.1038/srep45940](https://doi.org/10.1038/srep45940).

Field Notes use: the episode's approximately 39.85 ka date. The single-turn activation rule and
regional effect envelope are game abstractions.

<!-- field-note-citation: Silleni et al. (2020) -->
### Silleni et al. (2020)

“The Magnitude of the 39.8 ka Campanian Ignimbrite Eruption, Italy: Method, Uncertainties and
Errors.” *Frontiers in Earth Science* 8.
[doi:10.3389/feart.2020.543399](https://doi.org/10.3389/feart.2020.543399).

Field Notes use: eruption magnitude and the mapped proximal Campanian deposits. This corrects an
earlier mistaken “Scarpati et al.” attribution of the same DOI.

<!-- field-note-citation: Smith et al. (2016) -->
### Smith et al. (2016)

“Tephra dispersal during the Campanian Ignimbrite (Italy) eruption: implications for ultra-distal
ash transport during the large caldera-forming eruption.” *Bulletin of Volcanology* 78.
[doi:10.1007/s00445-016-1037-0](https://doi.org/10.1007/s00445-016-1037-0).

Field Notes use: central/eastern Mediterranean and eastern European ash dispersal. The game's
checked-in impact polygons are conservative gameplay interpretations, not ash-thickness surfaces.

<!-- field-note-citation: Pyle et al. (2006) -->
### Pyle et al. (2006)

“Wide dispersal and deposition of distal tephra during the Pleistocene ‘Campanian Ignimbrite/Y5’
eruption, Italy.” *Quaternary Science Reviews* 25.
[doi:10.1016/j.quascirev.2006.06.008](https://doi.org/10.1016/j.quascirev.2006.06.008).

Field Notes use: distal ash eastward to southern Russia and the wider Y-5 record. It supports the
outer dispersal context, not uniform lethality inside the modeled envelope.

Two generic mechanism statements in `DESIGN.md` are also grounded in the USGS Volcano Hazards
Program's reviewed pages [“Pyroclastic flows move fast and destroy everything in their
path”](https://www.usgs.gov/programs/VHP/pyroclastic-flows-move-fast-and-destroy-everything-their-path)
and [“Volcanoes Can Affect
Climate”](https://www.usgs.gov/programs/VHP/volcanoes-can-affect-climate). They explain the general
scale and atmospheric mechanism; they do not determine the episode's game coefficients.

## Toba timeline context

<!-- field-note-citation: Storey et al. (2012) -->
### Storey et al. (2012)

“Astronomically calibrated 40Ar/39Ar age for the Toba supereruption and global synchronization of
late Quaternary records.” *Proceedings of the National Academy of Sciences* 109.
[doi:10.1073/pnas.1208178109](https://doi.org/10.1073/pnas.1208178109).

Field Notes use: dating the Toba timeline marker. Toba deliberately has no simulation effect.

<!-- field-note-citation: Lane et al. (2013) -->
### Lane et al. (2013)

“Ash from the Toba supereruption in Lake Malawi shows no volcanic winter in East Africa at 75 ka.”
*Proceedings of the National Academy of Sciences* 110.
[doi:10.1073/pnas.1301474110](https://doi.org/10.1073/pnas.1301474110).

Field Notes use: evidence against a catastrophic East African volcanic-winter signal. It does not
prove that the eruption had no local effects elsewhere.

<!-- field-note-citation: Kappelman et al. (2024) -->
### Kappelman et al. (2024)

“Adaptive foraging behaviours in the Horn of Africa during Toba supereruption.” *Nature* 628.
[doi:10.1038/s41586-024-07208-3](https://doi.org/10.1038/s41586-024-07208-3).

Field Notes use: continued occupation and flexible subsistence around the Toba ash horizon in the
Horn of Africa. The note presents demographic consequences as uncertain.

## Lake-level context

<!-- field-note-citation: Cohen et al. (2007) -->
### Cohen et al. (2007)

“Ecological consequences of early Late Pleistocene megadroughts in tropical Africa.”
*Proceedings of the National Academy of Sciences* 104.
[doi:10.1073/pnas.0703873104](https://doi.org/10.1073/pnas.0703873104).

Lake Malawi rose in stages toward near-modern levels by approximately 60 ka;
levels were approximately 30–200 m below modern during 35–15 ka. These broad
intervals support lake-change context, not precise shoreline areas or single-year events.

<!-- field-note-citation: Bartov et al. (2002) -->
### Bartov et al. (2002)

“Lake Levels and Sequence Stratigraphy of Lake Lisan, the Late Pleistocene
Precursor of the Dead Sea.” *Quaternary Research* 57, 9–21.
[doi:10.1006/qres.2001.2284](https://doi.org/10.1006/qres.2001.2284).

Lake Lisan existed approximately 70–15 ka and rose sharply around 27 ka,
reaching its maximum elevation during approximately 26–23 ka before declining.
The chronology supports broad lake-change context, not exact outline scaling factors.

## Non-bibliographic references

References such as “Game model,” “Game abstraction,” and “sources in DESIGN §6/§7” are explicitly
internal provenance labels. They were reviewed to ensure they do not present a numerical mechanic
as an externally established scientific result. Event-record references describe accepted game
state and therefore require no external source.
