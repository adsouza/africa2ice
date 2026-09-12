# Lake geography

The 80,000–20,000 BP campaign includes Baikal, Tanganyika, Malawi/Nyasa, Victoria,
Turkana, Issyk-Kul, Qinghai, Van and Lisan. Malawi and Lisan have changing
shorelines; the other overlays remain fixed. Caspian water tiles are part of the
existing terrain mask.

## Map states and evidence

| Lake | Active interval (BP) | Authored shoreline |
| --- | --- | --- |
| Malawi | 80,000–60,001 | Early lower lake, retaining the deeper basin and excluding much of the shallow southern shelf |
| Malawi | 60,000–35,001 | Recovered lake with the long southern arms and broader western shelf inundated |
| Malawi | 35,000–20,000 | Later lower lake, with southern retreat but more area than the early lower state |
| Lisan | Before 70,000 | No Lisan overlay; this does not assert that the earlier Dead Sea basin was dry |
| Lisan | 70,000–27,001 | Lower lake occupying the Dead Sea basin and lower Jordan Valley |
| Lisan | 27,000–23,001 | Highstand extending toward the Sea of Galilee and farther south |
| Lisan | 23,000–20,000 | Partial retreat; not the much smaller Holocene Dead Sea |

Dates are representative switches within broad geological intervals. They do
not claim abrupt changes in those exact years, resolve short reversals, or
reconstruct continuous water-level curves. The highstand state summarizes the
rise beginning around 27 ka and the maximum around 26–23 ka. The last state
summarizes the beginning of retreat without importing the 15 ka shoreline into
a campaign that ends at 20 ka.

### Shape provenance and limitations

The rings are independently authored schematic interpretations of basin shape,
not uniform scaling, traced artwork, or redistributed bathymetry grids. Research
supports the direction and rough location of shoreline migration; the precise
vertices and area ratios are cartographic choices. In particular, modern
bathymetry does not account for all subsequent sedimentation or tectonic change.

- Malawi's basin geometry follows the steep north/east shores and shallow
  south/west shelves shown in [ILEC's bathymetric map AFR-13-01](https://wldb.ilec.or.jp/Display/html/3594)
  and described by [Eccles (1974)](https://doi.org/10.4319/lo.1974.19.5.0730).
  The early lower ring is informed by deeper contours; the later lower ring by
  the roughly 200–250 m shelf contour, retreating about 100 km at the southern
  end. These are schematic contour interpretations, not elevations assigned
  to every polygon vertex. The lower states preserve the deep basin.
- [Scholz et al. (2007)](https://doi.org/10.1073/pnas.0703874104) and
  [Cohen et al. (2007)](https://doi.org/10.1073/pnas.0703873104) support early
  recovery and near-modern levels around 60 ka, followed by the later
  35–15 ka low-level interval (30–200 m below modern). The deepest earlier
  megadrought minimum is outside the campaign. The 200 m end of the later
  range informs our representative lower outline, not a claim that the level
  stayed at that extreme throughout the interval.
- Lisan follows the basin extent in [Bartov et al. (2002)](https://doi.org/10.1006/qres.2001.2284)
  and the mapped highstand reproduced in [Ron et al. (2006), figure 1](https://www.tau.ac.il/~shmulikm/Publications/Ron_etal_Lisan-paleomag-DSbook.pdf).
  The northern extension follows the Jordan Valley toward Galilee; southern
  changes follow the shallow basin margin. The partial retreat is an authored
  intermediate shape, not a dated 23 ka contour survey.

The other lake locators are also schematic. Showing Victoria throughout the
campaign does not claim modern extent persisted through every drought.
The fixed Eurasian locators are supported by [Late Pleistocene Issyk-Kul deposits](https://www.sciencedirect.com/science/article/abs/pii/S0037073816301567),
[Qinghai lake-level evidence](https://doi.org/10.1016/j.quaint.2009.03.004) and
[Van's long sediment record](https://doi.org/10.1007/s10933-017-9973-z).
Makgadikgadi, Mega-Chad and the Aral Sea await separate supported chronologies.

## Projection and gameplay

The 96×64 grid has centres about 2.32° longitude and 1.90° latitude apart.
Outlines are clipped per tile and painted at camera/display resolution, so
changes can be visible within a tile without changing the entire tile to water.
Only explored fragments are published. Tile info names each intersecting lake,
including on currently uninhabitable land. The Turkana and Victoria starting
camps retain their adjacent-shore labels.

The overlays do not alter movement, resources, habitat, terrain checksums or
save compatibility. Regional tiles continue to represent surrounding land and
travel around shores. Loading a save reconstructs the correct stage from its
campaign date. Published point slices are independent of cached geometry.

`pkg/gameapi/lakes.go` is the shared stage clock; application projection uses
`lake_shorelines.go` and `lakes.go` to select and clip the rings. Field Notes
use the same stage boundaries and require an actual visible geometry change
on tiles explored before and after the turn. Exploration alone never triggers
an expansion note. Proximity checks include both old and new shores, so a
contracting lake can still notify a living sapiens band beside a retreating arm.

Each transition produces at most one note regardless of nearby band count.
Notes include publication links and distinguish the map change from unchanged
tile rules. Higher-priority discoveries and volcanic context defer notes to
later completed turns. Loading a save does not replay past changes and clears
the UI-local pending queue, as does starting a new campaign.
