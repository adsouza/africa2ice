# Lake geography

The campaign spans 80,000–20,000 BP. The map now includes schematic lake
footprints for Baikal, Tanganyika, Malawi/Nyasa, Victoria, Turkana, Issyk-Kul,
Qinghai, Van and (from 70,000 BP) Lisan. Caspian water tiles remain part of the
existing terrain mask.

## Map scale and gameplay

The 96×64 grid has centres about 2.32° longitude and 1.90° latitude apart.
Lake outlines are drawn within tiles, clipped independently to each explored
cell, and rendered at the current zoom/display resolution. Tile info names the
lake on each intersecting land tile, including currently uninhabitable tiles.
The original nearby-lake labels at the Turkana and Victoria starting camps
remain available even where the camp is on an adjacent shore tile.

These overlays describe lakes within a region-sized land tile. They do not
turn the entire tile into water or introduce new movement barriers, resource
bonuses or climate rules. Bands occupying these tiles represent camps on the
surrounding land; tile-to-tile movement abstracts travel around the shore.
Existing terrain classification, route checks, saved games and simulation
checksums are unchanged. Outlines are derived again on loading a save.

## Historical limits and sources

The authored rings are simplified geographic locators, not reconstructed
shorelines for particular dates. Lake levels and areas varied considerably;
we deliberately do not animate an unsupported lake-level curve. Showing
Victoria throughout the campaign is a schematic geographic convention, not a
claim that its modern extent persisted through every drought.

- Tanganyika and Malawi: [Scholz et al. (2007)](https://doi.org/10.1073/pnas.0703874104)
  and [Cohen et al. (2007)](https://doi.org/10.1073/pnas.0703873104) document deep
  lake records and severe changes in East African lake levels.
- Issyk-Kul: [Late Pleistocene lacustrine deposits](https://www.sciencedirect.com/science/article/abs/pii/S0037073816301567).
- Qinghai: [Paleoenvironmental and archaeological investigations](https://doi.org/10.1016/j.quaint.2009.03.004)
  document Late Pleistocene lake-level evidence.
- Van: [Long sediment record](https://doi.org/10.1007/s10933-017-9973-z)
  indicates persistence through the campaign period.
- Lisan: [Bartov et al., lake levels and sequence stratigraphy](https://cris.huji.ac.il/en/publications/lake-levels-and-sequence-stratigraphy-of-lake-lisan-the-late-plei/)
  dates the lake to approximately 70–15 ka. It is absent before 70 ka in the
  display catalog. Its schematic outline does not depict its maximum highstand
  or claim the earlier Dead Sea basin was dry.

Makgadikgadi, Mega-Chad and the Aral Sea are not added in this pass. Their
changing footprints require a separately supported chronology; a permanent
modern or maximum-highstand outline would imply more certainty than we have.

Implementation: `internal/application/lakes.go` contains the presentation
catalog and tile clipping; `pkg/render/lakes.go` paints the published fragments.
The catalog is outside the domain's authoritative water polygons intentionally.
