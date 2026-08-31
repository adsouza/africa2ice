# Africa 2 Ice: Paleolithic Dispersal — Design & Architecture

**Status:** Proposed — revised after design review, not yet implemented
**Date:** 2026-08-29
**Scope:** Playable vertical slice implementing the complete core loop at working depth

---

## 1. Overview

**Africa 2 Ice: Paleolithic Dispersal** is a turn-based eco-strategy game tracing the spread of
humanity out of East Africa across Eurasia and toward destinations from Frangistan and South Asia to
the Yellow River Basin, Sahul, and Beringia until the last ice age. The player manages Homo sapiens
bands, allocating people to foraging, hunting, toolcraft, megafauna tracking, and shelter. Archaic
hominin bands are computer-controlled ecological competitors. Both populations grow against local
carrying capacity, compete for shared resources, degrade their tiles, split under crowding, and
migrate into new biomes as the climate turns against them. Band-level heritable variation responds
to environment and local contact; the player may actively interbreed a sapiens band with a
co-located archaic band, while a persistent Field Notes panel explains the evidence, abstraction,
and strategic consequences. Long-cycle climate, irregular multi-century warm/cold pulses, and rare
major eruptions create shared environmental pressure without granting a random event permission to
erase an entire region. The world beyond East Africa begins beneath a persistent exploration veil
and is revealed outward as surviving sapiens bands reach new frontiers.

This document is the implementation design and the game's single source of truth. It resolves the
open questions and contradictions in the game concept, and records _why_ each choice was made, so
the reasoning survives past the conversation that produced it.

**Naming.** The authoritative title is **Africa 2 Ice: Paleolithic Dispersal**. The full title is
used in the Ebitengine window, browser `<title>`, and title scene; compact HUD copy may use
**Africa 2 Ice**. `africa2ice` remains the Go module path and save-namespace name.

### Scope of this build

The complete core loop at **working depth** rather than one system at full depth — breadth over
depth. The player allocates each sapiens band's five worker roles and directs eligible migration or
splitting; a deterministic computer policy plans for archaic bands. A turn then harvests flora,
fauna, and water and resolves food, health, hazards, population changes, and migration against a
shifting climate. §7's five-phase turn pipeline defines that internal order. Concretely: nine technologies rather than thirty, shared-resource
competition between player-controlled Homo sapiens and computer-controlled archaic hominins rather
than combat or diplomacy, and one band-splitting rule rather than a lineage system. The result is a
complete, playable game end to end. The four date-based campaign eras in §7 are a separate clock
contract and are always labeled with their BP ranges.

---

## 2. Decision register

Each row records a choice among viable ways to satisfy the product goals and constraints stated in
this document. The **Resolution** column is current and normative; the **Point** and **Why** columns
state the problem and rationale in self-contained form. **Appendix C is the authoritative
configuration manifest** for each selected or open release value, structural prerequisite, status,
and owning contract. Owning sections remain
normative for behavior and may repeat a current manifest value where a rule, example, bound, fixture,
or wire expectation needs it; any disagreement with the manifest is a document defect. Algorithm and
schema version identifiers remain in their owning contracts and are dependencies to update, not
duplicate manifest entries.
Capitalized lifecycle terms in this register — **Locked**, **Initial**, **Policy**, **Step N**,
**Derived**, and **Open** — use the vocabulary defined in Appendix C.1.

| Point                                                                                                                                                   | Resolution                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | Why                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| ------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persistence must use one cross-platform wire format without cgo                                                                                         | **No SQLite**; versioned JSON behind a pure-Go application port, using file and IndexedDB adapters on desktop and web                                                                                                                                                                                                                                                                                                                                                                                                | This avoids cgo/driver dependencies, preserves one wire format, and keeps the browser-storage path without putting persistence inside the domain.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| The biome set is specified, but nothing says where the world map comes from                                                                             | **Stylized real geography** — Africa across Eurasia to Sahul and the Bering Strait                                                                                                                                                                                                                                                                                                                                                                                                                                   | A broader dispersal game needs the northern, southern, and far-northeastern routes on one map; chokepoints only mean something if they are the real ones.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| The full world map is initially visible                                                                                                                 | **Persistent sapiens exploration veil:** East Africa starts explored; each surviving sapiens band reveals its local frontier and currently usable passage endpoints, and discoveries never regress                                                                                                                                                                                                                                                                                                                   | Geographic expansion gains the classic map-uncovering rhythm without adding tactical line-of-sight. Archaic movement cannot reveal the map for the player, and the reveal mask changes presentation and player knowledge rather than simulation outcomes.                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Audio feedback should not require binary assets                                                                                                         | **Synthesize PCM tones in Go**; `SoundManager` interface unchanged                                                                                                                                                                                                                                                                                                                                                                                                                                                   | No binary assets need to enter the repo, and real `.wav` files can replace the synth later without an API change.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| Moving climate-derived biome boundaries need legible spatial presentation                                                                               | **3D terrain via Tetra3D with a rotatable orbit camera**                                                                                                                                                                                                                                                                                                                                                                                                                                                             | §6 derives biomes every turn, so the tundra line and desert margin move; a rotatable 3D view is what makes that legible.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| High-DPI behavior is undefined                                                                                                                          | **Automatic DPI-aware rendering:** `LayoutF` converts the window's device-independent dimensions to a physical render surface using the current monitor scale, capped at `MaxRenderScale = 2.0`; HUD layout and hit targets remain in device-independent pixels                                                                                                                                                                                                                                                      | Text, vector strokes, and map details stay sharp on Retina/high-DPI desktop and browser displays without making controls physically smaller. One explicit viewport transform keeps UI input, 3D picking, resize behavior, and browser DPR handling consistent; the cap bounds fill rate and render-target memory on WASM.                                                                                                                                                                                                                                                                                                                                                                                  |
| Target platforms and title                                                                                                                              | **Desktop and web; Africa 2 Ice: Paleolithic Dispersal**                                                                                                                                                                                                                                                                                                                                                                                                                                                             | These are the current baseline, not later extensions or deviations.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| The web release has no public delivery contract                                                                                                         | **After the step-13 release-readiness gate exists, deploy the optimized repository project site to GitHub Pages after every fully successful push to `main`; pull requests validate the same release build but never publish**                                                                                                                                                                                                                                                                                        | This supplies the web target's stable public release path without credentials or a separate hosting stack, while preventing an uncalibrated implementation from becoming the public build merely because its platform checks pass.                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| Region naming                                                                                                                                           | **Frangistan**, not "Europe"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | This is the selected in-world naming convention.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| The two hominin species could both be read as player-managed                                                                                            | **The player controls only Homo sapiens; a deterministic computer policy controls archaic bands**                                                                                                                                                                                                                                                                                                                                                                                                                    | Explicit product clarification. Archaics remain dynamic ecological competitors without becoming a second player faction or introducing combat and diplomacy.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| There is no heritable-adaptation model                                                                                                                  | **Band-level heritable state:** six bounded continuous values change through deterministic selection and local gene flow, plus rare seeded mutation; technology remains a separate acquired-knowledge system                                                                                                                                                                                                                                                                                                         | This permits environmental adaptation, inherited tradeoffs, and introgression without simulating individuals or confusing biological inheritance with research. Splits copy exact values and v1 omits background random drift.                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Contact with archaic hominins has no player-directed genetic action                                                                                     | **`Interbreed` is an active sapiens spatial action:** target one co-located archaic band; reciprocal whole-catalog gene flow resolves simultaneously for the next turn                                                                                                                                                                                                                                                                                                                                               | Ordinary co-location still means resource competition only. The action does not transfer technology or change species identity, and the player cannot cherry-pick a trait.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| Historical/scientific explanation has no persistent UI home                                                                                             | **A visible-by-default, toggleable Field Notes panel** with historical context, explicit game-abstraction copy, hints, and compact sources                                                                                                                                                                                                                                                                                                                                                                           | Players can understand the evidence and the model without leaving the game. Its visibility is a locally persisted UI preference, never campaign state or a simulation input.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Campaign progress has no persistent spatial indicator                                                                                                   | **A thin, non-interactive timeline rail directly below the top bar:** 80,000 BP to 20,000 BP, with elapsed fill, a current-date marker, and 10,000-year major ticks                                                                                                                                                                                                                                                                                                                                                  | Campaign position stays visible without competing with the lower-edge Field Notes panel. The rail is derived UI only; its dates are chronological tick labels, not development-roadmap or campaign phases. Starting 80,000 BP provides lead-in to the key 70,000–60,000 BP dispersal interval.                                                                                                                                                                                                                                                                                                                                                                                                             |
| Five workforce categories are named, with neither their effects nor their allocation lifecycle                                                          | **Five distinct deterministic roles with persisted proportions:** save five basis-point shares totaling 100%; derive workers from start-of-turn population; population changes preserve the shares and splits copy them                                                                                                                                                                                                                                                                                              | This creates real allocation tradeoffs without an idle-worker state or repetitive reassignment after demographic change, while keeping work inside the existing bounded band, resource, research, and hazard systems. §7's foraging, hunting, and megafauna tables select the collection coefficients; shelter, work-risk, and research coefficients remain separate decisions.                                                                                                                                                                                                                                                                                                                               |
| The resource label "Flora" can be read as all vegetation                                                                                                | **Flora means plant food people can forage, not all vegetation**                                                                                                                                                                                                                                                                                                                                                                                                                                                     | The stock measures forageable plant food, not total plant biomass or herbivore fodder. Low flora can coexist with abundant fauna; initial capacities and collection rates are defined in §7 and Appendix B.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Food quantities have no defined common unit                                                                                                             | **One food unit (FU) sustains one person for one game turn; one normalized flora or fauna harvest unit converts to 1 FU before source-specific modifiers**                                                                                                                                                                                                                                                                                                                                                           | Flora and fauna remain separately conserved environmental stocks, but their stock units share one edible-yield baseline. Plant, Animal, and Aquatic sources therefore start comparable; ordinary terrestrial hunting and megafauna tracking both feed Animal before its one source modifier. Collection rate, access, and explicit source-specific modifiers create differences rather than hidden unit scales. The compressed campaign clock still does not imply literal calorie or monthly-ration accounting.                                                                                                                                                                                                                                                                      |
| Water quantities and regional demand have no defined scale                                                                                              | **One WU meets one person's baseline need per turn; demand rises linearly by `0.02` per °C from `1.0` at 20°C to `1.3` at 35°C, clamped outside that interval; initial `WaterHealthLossRate = 0.40`**                                                                                                                                                                                                                                                                                                                | Both species respond to current local temperature, including seasonal and long-term changes. Shortfall uses adjusted demand without a second heat multiplier on damage. The curve is selected for initial playtesting; absolute local temperature comes from `TemperatureAlgorithm: "lat-elev-offset-v1"` with the locked `28.0°C`, `-12.0°C`, and `6.5°C/km` coefficients plus the checked-in latitude table.                                                                                                                                                                                                                                                                                             |
| Fresh food and reserves have no consumption/carry-over rule                                                                                             | **Automatic per-band consumption:** combine actual harvest in FU with the post-spoilage reserve, meet the band's requirement, and carry forward any remainder subject to the end-of-turn cap                                                                                                                                                                                                                                                                                                                         | Food needs cannot remain unmet while that band still holds usable food. Fresh-first versus reserve-first makes no difference without food-age/type inventories; there is no extra rationing command or inter-band food pool.                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Food deficits have no size-independent severity measure                                                                                                 | **Use the fraction of food needs unmet:** divide the post-consumption deficit by the start-of-turn requirement                                                                                                                                                                                                                                                                                                                                                                                                       | Equal percentage shortfalls have equal hunger severity across band sizes. Nutrition, squared starvation, and the selected positive-growth scaling use this input; it is not itself a death rate.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Band health is named, but its scale, initialization, recovery, and carry-over are undefined                                                             | **Persistent normalized `float64` health in `[0, 1]`:** new-game bands start at `1.0`; nutrition contributes `-0.20 * FoodDeficitFraction` on shortage or `+0.05` when fully fed; combine this with water, endemic-disease, and applicable heritable-burden damage before one clamp; chronic vulnerability is `1 + (1 - Health)`                                                                                                                                                                                     | The nutritional rates are approved initial playtest defaults, not final balance. Water damage is linear in unmet-water fraction; outbreaks damage survivors in phase 5 without replaying phase 3. Direct water/disease deaths remain separate. The locked temperature contract and genetics effect tables remain implementation inputs. Health is a condition score; starvation stays independently quadratic, and splits/reloads preserve health.                                                                                                                                                                                                                                                         |
| Background disease has no health-damage function or mitigation scope                                                                                    | **Sum separate camp-related and non-camp disease health losses:** hygiene reduces only the camp-related portion; applicable technology and hygiene multiply remaining risk per component                                                                                                                                                                                                                                                                                                                             | Both rates come from the origin's disease-health profile inputs, independently of direct mortality. Caves provide no disease-health bonus.                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| Outbreak health damage has no magnitude or severity rule                                                                                                | **Initially 5–15 health percentage points before technology or heritable mitigation, interpolated linearly using the same severity draw as direct deaths**                                                                                                                                                                                                                                                                                                                                                           | Apply once to survivors in phase 5. Hygiene changes outbreak probability, not the conditional hit; only explicitly applicable acquired technology or heritable effects mitigate health damage.                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Starvation was defined from extracted food, with no reserve accounting                                                                                  | **Apply its squared-shortfall curve directly:** raw loss is start-of-turn population times `StarvationCoefficient = 0.10` times the post-reserve deficit fraction squared                                                                                                                                                                                                                                                                                                                                            | The coefficient is an approved initial playtest default: no food gives 10% raw loss; a 50% shortfall gives 2.5%, before mortality caps. There is no health gate, grace period, or health multiplier.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| Hunger's effect on positive population growth is unspecified                                                                                            | **Scale only positive logistic growth by the fraction of food needs met:** multiply it by `1 - FoodDeficitFraction`                                                                                                                                                                                                                                                                                                                                                                                                  | Half-fed means half the positive growth; no food means no positive growth. Zero or negative logistic growth is unchanged, so starvation cannot reduce crowding-driven decline. Growth remains separate from the existing mortality causes.                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| Turn metrics are named without a specified UI location                                                                                                  | **Stats grouped by scope:** population/clock/season in the top bar, population/health in the selected-band header, food/mortality/workforce/research below, and environment in the tile inspector                                                                                                                                                                                                                                                                                                                    | Essential stats stay visible alongside their controls. Both species remain inspectable; food actuals use the bounded last-turn report below.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Food-panel timing and freshness are unspecified                                                                                                         | **Last completed turn's actuals:** show required, consumed, and missing FU plus the unmet percentage, labeled “Last turn” with the turn number                                                                                                                                                                                                                                                                                                                                                                       | One display-only report per living band survives save/load, stays unchanged during ordinary planning, and clears on both sides of a split. Unavailable is not zero; the panel is not a forecast.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Carried-food capacity has no scaling rule                                                                                                               | **Population-scaled storage:** capacity is current population times `FoodStorageTurns = 3`, an approved initial playtest default                                                                                                                                                                                                                                                                                                                                                                                     | Larger bands carry proportionally more FU; splits create no aggregate capacity. Spoilage means a full reserve does not guarantee three fed turns without harvest. Overflow uses the end-of-turn rule below.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| Food can exceed carrying capacity after harvesting or population loss                                                                                   | **Discard overflow at turn end:** after consumption and all demographic/acute population changes, retain at most final population times `FoodStorageTurns`                                                                                                                                                                                                                                                                                                                                                           | The cap follows survivors without removing food before it can meet this turn's requirement. Excess is lost, not cached, transferred, or returned to tile stocks.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Stored food has no spoilage model or checkpoint                                                                                                         | **Start-of-turn fixed-percentage loss:** after planning succeeds, remove `FoodSpoilageRate = 0.10` of carried-over food before gathering or eating                                                                                                                                                                                                                                                                                                                                                                   | Ten percent is the approved initial playtest default. Fresh harvest first faces spoilage on the following turn if still stored; overflow remains a separate end-of-turn discard.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Foraging output per worker is undefined                                                                                                                 | **Linear potential collection, limited by shared flora:** each forager has a biome- and acquired-technology-adjusted collection rate; co-located bands share a shortage in proportion to their full potential demands                                                                                                                                                                                                                                                                                                | More workers increase potential harvest linearly, while scarce plants limit actual collection. There is no extra workforce-saturation curve, random harvest roll, or separate foraging inventory. The `2.5` reference rate, biome indices, and `PlantKnowledge ×1.30` modifier are selected initial values.                                                                                                                                                                                                                                                                                                                                                                                                |
| Fauna is associated with biomes, without distinguishing regional animal communities                                                                     | **Region × biome fauna profiles, with one shared fauna stock per tile:** local prey mixes include terrestrial, inshore-aquatic, pelagic-aquatic, and megafauna opportunities                                                                                                                                                                                                                                                                                                                                         | The same biome can offer different animal-food opportunities in different parts of the map. Profiles are derived, versioned configuration, not separately depleted species populations; the closed prey-group catalog distinguishes access and catch composition without pretending to simulate separate animal stocks. The eight profile vectors, the baseline archetype row and its five regional exceptions, region indices, access rules, catch coefficients, and work-risk coefficients are selected initial values.                                                                                                                                                                                                                    |
| Fishing was named only indirectly by two technology labels                                                                                              | **Fishing is an explicit part of the Hunting role:** local aquatic prey contributes its own rate and food-source component, while all catch still competes for and depletes the tile's one shared fauna stock                                                                                                                                                                                                                                                                                                        | This makes freshwater, shore, and pelagic opportunities strategically meaningful without a sixth workforce assignment, fishing inventory, or independently regenerating fish stock. Baseline inshore procurement is possible where the profile supports it; hafted tools, cordage/nets, trapping, and coastal navigation improve or unlock authored aquatic channels.                                                                                                                                                                                                                                                                                                                                      |
| Ordinary hunting output is unquantified                                                                                                                 | **Linear potential catch, limited by shared fauna:** hunters times the sum of accessible terrestrial and aquatic group rates; hunting and megafauna tracking share shortages proportionally                                                                                                                                                                                                                                                                                                                          | This makes workforce changes predictable without a separate saturation curve or random harvest roll. The allocated Hunting catch is split conservatively into ordinary-animal and aquatic food sources from its rate composition, while the one fauna stock is allocated and depleted once across both roles and species. The `2.5` reference rate, group weights, regional indices, and technology modifiers are selected initial values.                                                                                                                                                                                                                                                                 |
| Megafauna-tracking output is unquantified                                                                                                               | **Linear potential catch at a higher per-worker rate than ordinary terrestrial hunting where accessible megafauna exists, using the same shared fauna allocation and added field risk**                                                                                                                                                                                                                                                                                                                              | This creates a higher-yield, higher-risk terrestrial workforce choice without a separate success roll, minimum hunting party, or animal stock; rich aquatic opportunity is excluded from that rate comparison. The `1.75×` aggregate rate, profile support, `HaftedTools ×1.30` modifier, and higher `0.14` full-share work-risk total are selected initial values.                                                                                                                                                                                                                                                                                                                                        |
| The workforce contribution to hunting danger is unquantified                                                                                            | **Linear added risk from workforce shares, with a higher megafauna coefficient:** equal percentages give equal unmitigated work-risk weights under equivalent conditions, regardless of population                                                                                                                                                                                                                                                                                                                   | Risk reflects the fraction of the band doing dangerous work, not total hunters or catch. It joins the existing uncovered acute components before technology mitigation and the shared cap; the activation rule and complete event-kind coefficient vectors are selected in §7.                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Resource abundance is qualitative, with no starting balance tables                                                                                      | **An initial resource-balance appendix:** exact reference capacities, biome × season indices, regeneration rates, initial-stock fractions, and representative fauna-profile rows                                                                                                                                                                                                                                                                                                                                     | Appendix B makes the accepted initial configuration inspectable without treating balance values as historical measurements. §7 contains the complete authoritative fauna profiles and mapping; Appendix C tracks their lifecycle status.                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Shelter creation is named without a lifetime or workforce scaling                                                                                       | **Share-based, turn-local camp work with diminishing returns and a cap below immunity:** shelter includes exposure protection, camp security, and camp hygiene; use workforce share, not headcount, with no accumulated camp state                                                                                                                                                                                                                                                                                   | Equivalent allocations give equivalent proportional protection regardless of band size when other inputs match. Security reduces camp-related predation, and hygiene provides modest, targeted disease protection. Acute camp protection affects probability, never severity. Technology and camp work multiply the remaining risk of each component; two 50% reductions give 75% combined protection.                                                                                                                                                                             |
| Natural terrain shelter has no labor rule                                                                                                               | **Sparse authored shelter masks give land tiles ratings `0`, `0.25`, `0.50`, or `1.00`; the greatest overlapping rating improves exposure-protection efficiency only**                                                                                                                                                                                                                                                                                                                                               | Stylized belts of caves and rock overhangs create route-level shelter tradeoffs without inferring caves from biome/elevation, tracking cave inventories, or claiming an archaeological site census. The bonus belongs to the tile where exposure resolves, not the migrating band; its selected workforce-efficiency coefficient and acute base-weight/class profiles remain Initial balance values.                                                                                                                                                                                                                                                                                                       |
| Nothing says how five individually edited shares remain a complete allocation                                                                           | **Draft and apply, with an inline exit guard:** only explicit Apply of a complete 10,000-point vector mutates the simulation; leaving a dirty draft or ending the turn is blocked until Apply or Discard, then the player retries                                                                                                                                                                                                                                                                                    | This preserves exact player-entered shares and prevents accidental loss without hidden rebalancing, a confirmation modal, or deferred action replay.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| Technological development and band-level `T_tech` are named, but not who owns knowledge                                                                 | **Band-local technology with saturating toolcraft research, stacking gradual diffusion, and close-contact inter-species exchange; no species-wide unlock pool**                                                                                                                                                                                                                                                                                                                                                      | A band's active target gains bounded, diminishing-return progress from its toolcraft workers. Same-species bands exchange through co-location or ordinary adjacency, while sapiens and archaics must share a tile; each distinct knowledgeable contact contributes 20% of the recipient's technology cost per turn.                                                                                                                                                                                                                                                                                                                                                                                        |
| The compact technology slice does not enumerate its nodes, prerequisite edges, or starting state                                                        | **A closed nine-node DAG with an empty start:** `Firecraft`, `HaftedTools`, and `PlantKnowledge` are roots; the other six nodes use the exact prerequisites in §7; every new-game band starts with no acquired technology, zero progress, and no target                                                                                                                                                                                                                                                              | The DAG represents advances beyond universal baseline abilities, including basic fire use and ordinary stone tools. Acquired sets are prerequisite-closed; all three roots are initially targetable, while locked technologies have zero progress and cannot be targeted; diffusion uses the recipient's frozen prerequisite state.                                                                                                                                                                                                                                          |
| The mathematical model needs boundaries stronger than technical package grouping                                                                        | **Domain-driven Clean Architecture:** one Paleolithic Dispersal bounded context, `World` as aggregate root, framework-free application use cases and ports, and presentation/storage adapters whose source dependencies point inward                                                                                                                                                                                                                                                                                 | The simulation's language, invariants, units, and deterministic equations remain the center of the codebase. Ebitengine, JSON, files, IndexedDB, and view DTOs can change without becoming domain concepts. This uses interfaces at architectural seams rather than turning every domain type into an interface or every subsystem into a repository.                                                                                                                                                                                                                                                                                                                                                      |
| Separation of concerns was asserted in prose, with nothing enforcing it                                                                                 | **Enforced** by golangci-lint + depguard, plus a linter-independent Go test                                                                                                                                                                                                                                                                                                                                                                                                                                          | Prose does not survive contact with a deadline.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| Nothing stops draw code mutating simulation state                                                                                                       | **Structural** — render/UI receive isolated `gameapi.Frame` value snapshots and emit commands; only the application layer may import `internal/domain`                                                                                                                                                                                                                                                                                                                                                               | See §4.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| Ebitengine updates at 60 TPS, but the game is turn-based                                                                                                | **An explicit `ui.EndTurn` action is the only operation that advances simulation time**                                                                                                                                                                                                                                                                                                                                                                                                                              | Camera movement, menus, assignment edits, saving, and rendering may run every Ebitengine tick without silently advancing the campaign.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| A seed alone cannot resume a consumed pseudorandom sequence                                                                                             | **Persist a serializable PCG source from `math/rand/v2` in every save**                                                                                                                                                                                                                                                                                                                                                                                                                                              | Reloading must continue the next random draw, not restart the original sequence.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| A per-turn climate equation versus a directed LGM campaign                                                                                              | **Hybrid:** fixed long-term LGM trend, 12-turn seasonal oscillation, and small bounded noise derived from world seed + turn                                                                                                                                                                                                                                                                                                                                                                                          | This preserves the intended historical arc and the seasonal/stochastic texture without coupling climate to unrelated random-event draw order.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| There is no abrupt-climate or major-eruption model                                                                                                      | **Versioned macro-environmental episodes:** source-backed warm/cold pulses alter the climate curve; rare eruptions cause bounded local mortality and wider one-turn habitat/resource shocks, followed by ordinary ecological recovery                                                                                                                                                                                                                                                                                | Glacial climate was not a perfectly smooth cycle, while an eruption's most extreme direct effects are local and its aerosol cooling is much shorter than a 50–300-year game turn. No episode can be configured to guarantee a whole-band or whole-region kill.                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Persistent overuse could permanently reduce base `K`                                                                                                    | **Reversible:** bounded `Degradation` temporarily lowers ecological capacity and resource caps, then recovers below baseline `K`                                                                                                                                                                                                                                                                                                                                                                                     | This keeps overexploitation consequential without letting an early mistake permanently poison a 400-turn campaign; permanent ecological scarring is outside this slice.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| Regeneration was called logistic but written as multiplicative growth                                                                                   | **Toward-cap:** each stock recovers a configured fraction of its gap to its current cap before extraction                                                                                                                                                                                                                                                                                                                                                                                                            | This is bounded, recovers from zero, and is easier to explain and balance than either the mislabeled printed equation or a zero-absorbing logistic recurrence.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Migration could be automatic or a player prompt                                                                                                         | **Assisted manual for sapiens, deterministic automation for archaics:** rank reachable land/passages for every band; no sapiens band moves or splits without a player command                                                                                                                                                                                                                                                                                                                                        | This retains player agency over sapiens while giving computer-controlled archaics the same spatial-pressure model and authoritative movement rules.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Repeated splitting has no world-level bound                                                                                                             | **Global `MaxBands = 256` across both species; reject `SplitBand` atomically at the cap**                                                                                                                                                                                                                                                                                                                                                                                                                            | This bounds turn work, RNG draws, frames, markers, and saves for WASM while leaving enough bands for a broad dispersal campaign; deaths free capacity naturally.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| “Adjacent” was used without defining grid topology                                                                                                      | **Eight-way ordinary movement with no water-corner cutting:** cardinal step length `1`, diagonal step length `√2`; a diagonal exists only when both corner-adjacent tiles are land                                                                                                                                                                                                                                                                                                                                   | Eight-way movement avoids blocky routes, while the corner rule prevents a diagonal shortcut from silently crossing a strait that should require a named passage.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| Each biome has a qualitative movement cost, but no numeric curve or edge formula                                                                        | **Destination-environment cost:** ordinary edge cost is step length multiplied by the destination tile's current `MovementCurve(V)` and geographic-biome factor                                                                                                                                                                                                                                                                                                                                                      | A band pays to enter the terrain ahead. This is simpler to explain than blending endpoints, while intentionally allowing a difficult destination to be cheaper to leave than to enter. The numeric curve and factors are Initial DESIGN values.                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `M_hazard` was subtracted during demographics, yet hazards were also named as phase 5                                                                   | **Split chronic/acute:** deterministic chronic attrition resolves at the origin in phase 3; bounded random acute events resolve on the final tile after migration in phase 5                                                                                                                                                                                                                                                                                                                                         | Persistent risk remains part of demographics, while discrete incidents follow the pipeline phase order and make the chosen destination or crossing consequential without charging either risk twice.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| The climate model had only a temperature axis, so aridity could vary in space but never in time                                                         | **A second derived climate scalar:** a long-term moisture offset built like the temperature curve — directed aridification plus a damped ~21,000-year precession oscillation — times a 13-region aridity weight vector, feeding `ClassifyBiome`; three epochs derived from it drive only §8's render grade and Field Notes                                                                                                                                                                                           | Desertification is the dispersal pressure this campaign is about, and a temperature-only model could advance ice but never desert. Building it as an offset over the rasterizer's fixed base moisture adds no save field, because climate is already derived rather than serialized. The oscillation is not decoration: a monotonic ramp would assert that MIS 3 dried steadily when it was in fact the mildest stretch of the window. Epochs stay presentation-only so no domain rule branches on them, preserving the decoupling that keeps abrupt, seasonal, and noise components out of Beringian eligibility.                                                                                         |
| Ordinal biome capacity and movement grades supply neither `V` nor a shared scale, leaving `ClassifyBiome`, `BaselineK`, and movement disconnected in v1 | **Introduce a DESIGN-selected vegetation index `V`:** `V = EffectiveMoisture * ThermalSuitability(°C)` drives biome, a piecewise-linear `BaselineKCurve(V)`, and a piecewise-linear `MovementCurve(V)`; the six biome labels remain, with two geographic ones classified before `V` and the lowest vegetative band split by temperature                                                                                                                                                                              | `V` turns the qualitative per-biome intent into one coherent numeric scale. Its formula, thresholds, capacity ranges, and movement knots are Initial gameplay choices rather than measurements or structural necessities. Within-band curves avoid most carrying-capacity and movement whiplash while allowing Semi-Arid Desert and Glacial Tundra to cover both hyperarid/cold and milder low-vegetation states.                                                                                                                                                                                                                                                                                          |
| Highland polygons supplied neither numeric elevation nor a biome threshold                                                                              | **A sparse authored elevation layer:** non-highland land and water are `0 km`; each of the ten highland polygons has a representative `1.25–3.0 km` height; overlaps take the maximum; `HighlandElevationKm = 1.0 km`, with `ElevationKm > HighlandElevationKm` classifying Mountainous Highlands                                                                                                                                                                                                                         | This makes absolute temperature, highland classification, altitude UV, hypoxia pressure, orographic moisture, and terrain height computable from one deterministic geography input. The coarse plateau values are Initial gameplay abstractions, not range-wide mean elevations or a paleotopographic reconstruction.                                                                                                                                                                                                                                                                                                                                                                                        |
| Named water gaps need migration semantics                                                                                                               | **Historically asymmetric passages:** coastal navigation unlocks high-cost Wallacea crossings; a climate-derived Beringian land bridge opens independently of abrupt/seasonal/noise variation                                                                                                                                                                                                                                                                                                                        | The deep Wallacea channels required water travel, while the far-northeastern route should respond to long-term glacial conditions without implementing global dynamic coastlines.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| There is no geographic victory rule                                                                                                                     | **Route-neutral regional achievements:** track sapiens establishment in every named region; Frangistan, South Asia, Yellow River Basin, Sahul, and Beringia are equal destinations                                                                                                                                                                                                                                                                                                                                   | No single dispersal route is the canonical win path. Reaching any destination proves dispersal; the complete regional record distinguishes broader outcomes.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Save slots and non-modal UX                                                                                                                             | **Slots 1–3 manual, 99 quick-save, 101–103 rolling autosaves; no overwrite modal; two-second result toasts**                                                                                                                                                                                                                                                                                                                                                                                                         | Ctrl+S on Windows/Linux and Cmd+S on macOS quick-save, while manual saves open from Pause. Autosave triggers are coalesced so asynchronous storage remains bounded.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| Campaign length and terminal conditions are undefined                                                                                                   | **A 400-turn campaign from 80,000 BP to 20,000 BP in four 100-turn campaign eras:** turns advance 300, 150, 100, then 50 years, with explicit victory and loss states                                                                                                                                                                                                                                                                                                                                                | A finite campaign needs a clock and terminal invariant before it can be implemented or balanced. Progressively shorter turn spans provide finer late-game decisions while preserving the 400-turn play length.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| Web release requirements                                                                                                                                | **Stripped, trimmed, `wasm-opt -O3` release build; Brotli compressed-size gate that ratchets down to the measured build; native CI matrix; automated Chromium boot/action/save smoke; measured performance gate; step 2a measures size and frame rate on the dependency skeleton**                                                                                                                                                                                                                                                                                                                                                      | Compilation and static artifact checks cannot detect a loader, console-panic, IndexedDB, or unusably slow runtime failure. The browser test toolchain is development-only: the game and deployed site remain Go/WASM plus the required static loader. Transfer size and frame rate are properties of the pinned dependency set and the release geometry rather than of game code, so both are measured at the walking skeleton where the answer is free, and `wasm-opt -O3` is not the size lever it looks like — it optimizes decompressed size and startup.                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| Desktop is a target but has no distribution contract                                                                                                    | **SemVer-tagged GitHub Releases contain unsigned portable archives for Linux amd64, Windows amd64, and macOS amd64/arm64 plus `SHA256SUMS`; the web build remains the recommended friction-free release**                                                                                                                                                                                                                                                                                                              | A native target should produce something users can run, not merely prove that `go build` succeeds. Signing, notarization, installers, and automatic updates remain outside v1, so the release notes must state the resulting OS trust prompts rather than implying a signed desktop package.                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Determinism was asserted without naming a portability scope                                                                                             | **Release cross-target determinism:** §5's three archtest arithmetic rules require every floating-point product chain in `internal/domain` to end at an explicit rounding conversion, reject `math.FMA` plus every unapproved production `math` call, and permit only the audited `RoundPopulation` float-to-integer conversion after finite/range validation; `ClimateAlgorithm` and `TemperatureAlgorithm` use four checked-in exact trigonometric tables; `RNGAlgorithm` owns the seed expansion and the unit-draw mapping so no step from seed to draw lives in `math/rand/v2`; `internal/verification.ReferenceRun` produces all three canonical checkpoint records, which CI compares across amd64, arm64, and js/wasm, and supported saves are portable | Go may fuse `a*b + c` on one architecture and not another, and Go leaves out-of-range float-to-integer conversion implementation-dependent. Requiring each product-chain end itself to be written inside an explicit conversion defeats fusion, while the single named population conversion first proves that its input is finite and within `uint32`; the source gates make both disciplines mechanical. `AllocationBP` remains fixed-point because its five entries have an exact sum invariant. `RNGAlgorithm` also owns its unit-draw mapping because Go guarantees `math/rand/v2`'s API, not a particular `Rand.Float64` output sequence. |
| Logistic growth read only the acting band's population against a shared tile's capacity                                                                 | **The crowding factor uses the origin's total population across both species:** `BaseGrowth = r · P · (1 − P_total_origin / K_eff)`, and `Stress` uses the same whole-tile denominator                                                                                                                                                                                                                                                                                                                               | Two bands of 50 on a `K_eff` of 100 previously each read the tile as half-empty and jointly overshot toward 200. Total population already drives degradation and migration scoring; growth was the one demographic term that did not see co-location. `K_eff` stays band-specific because `T_tech` is the acting band's own multiplier.                                                                                                                                                                                                                                                                                                                                                                    |
| `MaxBands = 256` is global, but only the archaic side splits automatically                                                                              | **`MaxArchaicBands = 96` caps the computer policy's split branch**, reserving at least 160 places that only sapiens can occupy                                                                                                                                                                                                                                                                                                                                                                                       | Player-first ordering decides one turn's last slot and says nothing about the balance accumulated over 400. Without a floor, the archaic policy can consume the budget by mid-campaign and structurally remove the player's dispersal verb. The sub-cap constrains the policy only: it never kills, merges, or invalidates an archaic band.                                                                                                                                                                                                                                                                                                                                                                |
| Absolute local temperature was a recurring "calibration remains open" note with no owner                                                                | **`TemperatureAlgorithm: "lat-elev-offset-v1"`:** a versioned latitude/elevation/offset function with a checked-in 64-row latitude table and locked `28.0°C`, `-12.0°C`, and `6.5°C/km` coefficients                                                                                                                                                                                                                                                                                                                 | `ClassifyBiome`, water demand, the endemic-disease table, and cold-exposure genetics all need degrees Celsius rather than a signed offset. Build step 4 implements and verifies this already-selected contract before those consumers; the balance pass validates it without retuning it. The table converts the map's degree-valued latitude to the sine-squared factor without runtime trigonometry. Changing the table, function, or coefficients reclassifies every tile in an existing save, so any such change requires a new algorithm version.                                                                                                                                                     |
| §7's tables say "closed before implementation" while §12 tunes them later                                                                               | **Structure before values:** shape and validation close before the consumer; Locked and Initial values ship with that consumer, the step-5e viability gate may tune Initial domain values, and step 12 performs final calibration; only an explicitly Open row may wait for step 12 to select its first release value                                                                                                                                                                                                     | Consumers remain implementable and testable without placeholder release data. Appendix C currently has no Open rows, so neither pass is allowed to invent a missing contract or retune a Locked value.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| A layer-complete roadmap could defer its riskiest integrations and whole-campaign feedback until the end                                                | **Required green substeps, an early desktop/browser walking skeleton after the public contract, a domain-only viability gate immediately after the full turn loop, and final release closure after calibration**                                                                                                                                                                                                                                                                                                      | Clean Architecture still governs dependencies, but implementation proceeds in reviewable increments and proves platform boot and campaign viability before persistence, presentation, and publication multiply the cost of correcting a model or integration failure.                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Victory, an entire destination, and the chronic cap rested on unassigned constants                                                                      | **`MinEstablishedBand`, `BeringiaOpenFraction`, and `MaxChronicRate` have approved initial playtest values; `BeringiaOpenTemp` is derived from the fraction and `LGM_cooling`**                                                                                                                                                                                                                                                                                                                                      | The three independent inputs were referenced but never given values or listed in §12's otherwise exhaustive tuning pass, and the establishment threshold plus derived Beringian threshold independently determine whether the campaign is winnable. Deriving the latter from the campaign's own cooling depth means tuning `LGM_cooling` cannot silently strand or permanently open the bridge, and the derived value is never tuned separately.                                                                                                                                                                                                                                                           |
| The `gameapi.Frame` band contract omitted fields §7/§8 behavior requires                                                                                | **Add `BandID`, `Species`, `TileID`, `Stress`, and `SpatialActionUsed` to the projected band, plus a `TerrainRevision` cache key to the frame**                                                                                                                                                                                                                                                                                                                                                        | Build step 2 constructs `pkg/gameapi` from §4, and the enumeration there could not drive the specified UI: species markers, the sapiens-only top-bar sum, the `Stress > SplitStressThreshold` split gate, and disabling movement after an accepted `Interbreed` all need projected state. `WorldRevision` cannot key terrain caches because it increments on every `SetAssignment`.                                                                                                                                                                                                                                                                                                                        |
| The heritable catalog carried `HbS`, whose value was gated to the campaign's final tenth                                                                | **Cut `HbS` from v1:** `HeritableTraitCount` drops to 6, and the allele, its Hardy-Weinberg genotype shares, the malaria/anemia tradeoff, and the region × biome `FalciparumPressure` table move to §14                                                                                                                                                                                                                                                                                                              | Published age estimates run from roughly 22,000 years ago to a Holocene origin, so even the most permissive reading made it eligible only from turn 360 of 400, it was tuned to be rare enough to miss entirely, and Field Notes had to call its appearance partly counterfactual. It nonetheless cost a trait slot, genotype-share derivation, a fully authored pressure table with its own validation, and separate anemia health and mortality components — all on the critical path through the balance pass. It was the last enum entry and nothing else read it, so removal shifts no ordinal. It also removes genotype decomposition and the only date gate from the genetics subsystem.            |
| Constant values were restated across the rule, its fixtures, §9, §12, and §13                                                                           | **Appendix C is the authoritative configuration manifest:** selected value or source, lifecycle status, and owning contract in one place; owning rules may repeat values for clarity but must agree                                                                                                                                                                                                                                                                                                                  | A survey found 126 value restatements across dozens of named constants, so retuning one number took several consistent edits and a missed one drifted silently — the mechanism that left 19 stale "calibration remains open" notes after a single decision moved. The manifest gives §12 a finishable checklist while retaining the rule sections as the behavioral contracts. Naming things it had to list surfaced `SplitStressThreshold`, which gated the player's split verb as a bare `0.9` in nine semantic places.                                                                                                                                                                                  |
| Test specifications were restated as build work in §12 and again as verification in §13          | **One question per section:** §7 owns the rule and the fixtures that pin its numbers, §12 owns placement and integration-scale assertions, §13 owns the CI gate and the display- or human-required checks; each cites the others rather than repeating them | The same list written three times drifts three ways, and roughly a quarter of the document was that list. The rule also makes placement decidable for a new test instead of leaving it to whichever section the author was editing. A step that re-enumerates a contract's fixtures is now a defect, the same way a value disagreeing with Appendix C is.                                                                                                                                                                                                                                            |
| Cross-cutting obligations were restated at nearly every rule                                      | **A Standing conventions block after this register** states the versioning-and-migration, no-`WorldRNG`, and derived-not-serialized defaults once; rules state only their exceptions                                                                     | Thirty-seven migration clauses and thirty-nine no-RNG clauses said the same thing in slightly different words, which buried the handful of genuinely surprising cases — the two-identifier couplings, and the counter-based hashes that look like random sources but are pure functions. Stating the default once makes a repetition informative again.                                                                                                                                                                                                                                              |
| Post-incident diagnosis had no durable, cross-cutting operational trace                                                                                 | **Aspect-oriented structured session logging:** explicit port decorators at the composition root; a new bounded JSONL temp file for every desktop session and JSONL in the JavaScript console on web                                                                                                                                                                                                                                                                                                                 | The same use-case and persistence seams are observable without scattering logging through the mathematical model or coupling simulation outcomes to I/O. Desktop users can retrieve a known per-session file after a failure, while web uses the platform's existing console and adds no persistence or network service.                                                                                                                                                                                                                                                                                                                                                                                   |

### Standing conventions

Three obligations hold everywhere below unless a specific rule says otherwise. Stating them once
here is what lets the rule sections state only their exceptions; a rule that repeats one of them is
either flagging a case that is easy to get wrong or is redundant.

- **Versioning and migration.** Every simulation rule, table, or constant that can change the future
  behavior or interpretation of a saved campaign belongs to one of the algorithm contracts §9
  enumerates. Changing any such input after release requires that contract's version identifier to
  change and an explicit save migration, and a change to the serialized shape additionally requires
  a `SaveState.SchemaVersion` bump. Presentation, platform, observability, and verification policies
  in Appendix C.13 do not inherit this obligation merely because they appear in this document;
  serialized UI preferences use `UISettings.SchemaVersion`, while policies that consume no saved
  state require no campaign migration. V1 is unreleased throughout, so no migration from a
  provisional identifier exists anywhere. A rule repeats this only where one edit must move two
  identifiers at once, or where the owning contract is not the one a reader would guess —
  `TemperatureAlgorithm` versioning absolute °C separately from `ClimateAlgorithm`, or
  `GeneticSelectionAlgorithm` reading a table `TemperatureAlgorithm` owns.
- **Randomness.** No rule consumes `WorldRNG` unless it says so. The complete set of consumers is
  acute event selection, acute severity, and rare mutation emergence. Everything else — climate,
  moisture, resource initialization, allocation, conversion, consumption, health, growth, chronic
  attrition, shelter, research, diffusion, gene flow, selection, exploration, achievements, archaic
  planning, frame projection, saving, and loading — is deterministic given saved state and the
  campaign turn. A rule restates this only where a reader might reasonably expect a draw, such as a
  hazard rule that resolves next to one, or a counter-based hash that looks like a random source
  but is a pure function.
- **Derived, not serialized.** Anything computable from saved state plus versioned configuration is
  derived each turn and never written to a save, a state hash, or slot metadata. §9 lists the
  deliberate exceptions, of which the exploration bitset is the only one that records history rather
  than future-affecting state.

---

## 3. Architecture

```
                         source-code dependencies point inward

main / main_js ─┬─► internal/adapters/logging (session lifecycle + platform sink)
                └─► pkg/app (Ebitengine host + composition)
                             ├─► internal/application ─► internal/domain
                             │          └──────────────► pkg/gameapi
                             ├─► internal/adapters/storage ─► internal/application
                             ├─► internal/adapters/logging ─┬─► internal/application
                             │                              ├─► pkg/gameapi
                             │                              └─► pkg/ui
                             └─► pkg/ui ──┬─► pkg/render ─► pkg/gameapi
                                         ├─► pkg/audio
                                         └─► pkg/gameapi
```

This is a domain-driven Clean Architecture, not merely a folder convention. Source dependencies
point toward the mathematical model. `internal/domain` imports only the standard library;
`internal/application` imports the domain and the adapter-facing `pkg/gameapi` contract; adapters
implement inward-owned ports and never become dependencies of either inner layer. `pkg/app` is a
driving adapter and composition root: it wires concrete implementations together and is the only
long-lived object that holds both the game use-case port and the UI scene stack.

`pkg/gameapi` is the only simulation vocabulary visible to presentation code. It contains commands,
isolated frame DTOs, and the inbound `Game` port, but no equations or mutable domain objects. Neither
`pkg/gameapi` nor its DTOs are the domain model. The application layer explicitly translates
`gameapi.Command` values into domain operations and projects domain state into `gameapi.Frame`.
That deliberate boundary duplication prevents display, wire-format, and framework concerns from
leaking into `World`, while depguard makes `*domain.World` unnameable in drawing and input code.
Closed enums exist in both boundary and domain form; application mappings use exhaustive switches,
never ordinal casts. Count-sentinel tests fail whenever one side gains a value without a mapping.
The same rule applies to storage: repository operation IDs, slot values, metadata, and completions
are application-owned port types, which `GameService` maps to the corresponding `gameapi` DTOs. A
storage adapter therefore imports application types but never the driving API.

### Domain model and ubiquitous language

The current slice is one **Paleolithic Dispersal bounded context**. Climate, ecology, bands,
technology, genetics, hazards, and migration interact within one turn and therefore are not split
into artificial micro-contexts or repositories. Terms defined by this design are the code's
ubiquitous language: `Band`, `Tile`, `Region`, `CampaignTurn`, `CampaignEra`, `AssignmentBP`, `FU`,
`WU`, `Health`, `TechnologyDAG`, `HeritableTrait`, `Passage`, and `MacroEpisode` keep these names and
meanings across domain code, tests, Field Notes, and boundary mappings.

- `World` is the aggregate root and consistency boundary. A use case may not load or save an
  independently mutable band, tile, stock, or research record. All accepted planning changes and
  the complete five-phase turn transition pass through `World` methods and either succeed
  atomically or leave the aggregate unchanged.
- `Band` and `Tile` are entities with stable typed identifiers. Population, food and water units,
  health, probabilities, basis points, temperatures, trait values, and other easy-to-confuse
  quantities are validated value types. Conversion to raw JSON numbers or display strings happens
  outside the domain.
- Rules local to an entity or aggregate are methods. Cross-entity mathematics such as resource
  allocation, climate evaluation, contact diffusion, migration ranking, and hazards are concrete,
  pure domain services. They remain close to their equations and invariants and return typed domain
  events; they do not perform I/O, read wall-clock time, or emit UI text.
- The owned, serializable RNG is part of the `World` aggregate state. Stable iteration order,
  algorithm identifiers, and exact draw ownership are domain invariants, so a saved aggregate
  continues the same deterministic history after reload.
- Interfaces exist at use-case boundaries: the inbound `gameapi.Game` port and outbound application
  ports such as `CampaignRepository`. Domain algorithms use concrete types and pure functions; the
  design explicitly rejects an interface-per-type, repository-per-entity, anemic-domain style.
- DDD here does not imply event sourcing or separate CQRS databases. Domain events describe typed
  outcomes and feed the bounded player history, but `SaveState` is the authoritative aggregate
  snapshot and load never rebuilds the world by replaying the event feed. `gameapi.Frame` is an
  ephemeral projection, not a second durable model.

The domain exports a persistence-neutral `State` memento with domain types and no JSON tags. The
application layer maps between that memento and the versioned `SaveState` persistence DTO, owns
schema/algorithm compatibility and `WorldRevision`, and swaps a fully validated replacement
aggregate only on the game goroutine. `CampaignRepository` is an application-owned outbound port;
the file and IndexedDB adapters persist immutable data and report completions but never validate or
mutate `World` directly. `ExportState` deep-copies every slice and owned byte sequence, so changing a
memento cannot mutate the live aggregate; `RestoreWorld` accepts no partially validated state.
The canonical campaign-state hash is likewise an application concern: it is computed from the
normalized `SaveState` payload and excludes repository metadata, timestamps, and operation IDs.
Domain tests may compare `State` values directly but do not gain a JSON or storage dependency.

### Package layout

```
main.go                  desktop flags/window/reference-checkpoint setup, session-log close/recovery, app.New, RunGame   (//go:build !js)
main_js.go               wasm entry: session-log recovery, canvas sizing, RunGame                   (//go:build js)
.golangci.yml            depguard boundary rules + forbidigo
.github/workflows/ci.yml native Linux/macOS/Windows matrix + WASM release + Pages deploy jobs
.github/workflows/release.yml tagged native archives + checksums
web/index.html           canvas host, loader, full game title
web/wasm_exec.js         copied from GOROOT by the build script
build_web.sh             dev build or stripped + wasm-opt release build
tools/web-e2e/package.json pinned Playwright/Chromium test tooling only; never shipped
tools/web-e2e/package-lock.json exact browser-test dependency graph
tools/web-e2e/smoke.mjs  local-build boot/action/save/reload, frame-summary, and console-failure gate
tools/web-e2e/profile.mjs reference-machine 30-second render/turn/memory measurement
tools/run_wasm_go_tests.mjs compile/serve/run js-tagged Go tests in pinned Chromium
tools/run_wasm_checkpoint.mjs run the js-tagged checkpoint binary and extract reference-checkpoints.json
testdata/performance_baseline.json reviewed step-8 normalized-time + bytes/op baselines
testdata/performance_profile_save.json fixed turn-300/256-band/all-explored render workload
tools/check_benchmarks.sh calibrated same-runner 25% regression comparison for release readiness
docs/PERFORMANCE.md      step-2a skeleton size/FPS baseline, release-candidate machine identity, and measured performance record
THIRD_PARTY_NOTICES.md   reviewed runtime/tooling license notices shipped with native archives

pkg/gameapi/             DRIVING PORT + DTO CONTRACT — stdlib only, no dependencies
  doc.go                 package contract and architectural role
  enums.go               Biome, Season, Tech, HeritableTrait, Species, Region, FaunaGroup, MacroEpisode + String()
  frame.go               Frame, climate/macro summaries, passages/achievements, Tile incl. elevation/temperature/movement/Band/FoodTurnReport/OutcomeReport/Event/MigrationCandidate values — no pointers
  command.go             Command iface + SetAssignment, QueueMigration, SplitBand, ResearchTech, Interbreed
  errors.go              stable ErrorCode/GameError boundary values; no domain types
  game.go                Game inbound port: Snapshot, NewCampaign, Apply, EndTurn, and storage use cases
  storage.go             operations/results, slot constants/kinds, SlotMetadata

internal/domain/         CORE DOMAIN — one package, stdlib only; no gameapi, application, JSON, or frameworks
  doc.go                 package contract and architectural role
  identifiers.go         BandID, TileID, RegionID, PassageID and stable ordering
  concepts.go            domain Species, Biome, Season, Technology, traits, events, and count sentinels
  quantities.go          uint32 Population + audited rounding boundary; validated Health, FU, WU, AssignmentBP, Probability, trait values
  errors.go              typed invariant and domain-rule failures; no presentation strings
  geo.go                 lat/lon <-> tile projection
  geodata.go             real-coordinate landmasses, height-valued highlands, rivers, natural-shelter regions
  region.go              Region table, destination set, and display names (South Asia, Yellow River Basin, ...)
  worldgen.go            concrete WorldGenerator domain service -> elevation/moisture/natural-shelter grid
  biome.go               DESIGN vegetation index V; ClassifyBiome + BaselineKCurve(V) + MovementCurve(V)
  fauna.go               closed region × biome profiles, prey mixes, shared exploitation inputs
  tile.go                stocks, bounded Degradation, CarryingCapacity(), toward-cap Regenerate()
  grid.go                Grid, indexing, bounded eight-way OrdinaryEdges(), no-water-corner rule
  tech.go                band-local research state, T_tech/modifier tables, local diffusion
  genetics.go            bounded heritable state, selection, mutation, and local gene flow
  band.go                Band, assignments, MaxBands=256, atomic Split()
  competition.go         workforce-derived per-tile demand + proportional allocation across species
  archaic.go             deterministic computer policy invoked only within aggregate turn advancement
  passage.go             named Wallacea crossings + climate-derived Beringian land bridge
  migration.go           directed destination-environment costs, ranking + queued movement resolution
  hazard.go              deterministic chronic attrition + RNG-driven acute events
  scenario.go            deterministic sapiens + archaic starting bands
  exploration.go         persistent sapiens-only explored-tile mask + bounded frontier reveal
  campaign.go            four-era CampaignDate + calendar progress conversion
  climate.go             LGM/orbital + moisture/precession trends, abrupt pulses, season, noise, epochs
  macroevent.go          authored episode catalog, warnings, volcanic impact masks + refugia
  turn.go                atomic five-phase turn pipeline / population equation
  rng.go                 serializable PCG WorldRNG + versioned climate/resource counter hashes
  events.go              typed domain outcomes consumed by application projectors
  state.go               persistence-neutral aggregate memento; no JSON tags
  world.go               World aggregate: NewWorld, RestoreWorld, PlanPlayer, AdvanceTurn, ExportState

internal/application/    FRAMEWORK-FREE USE CASES — imports domain + gameapi + stdlib only
  service.go             GameService implements gameapi.Game and owns the live World
  mapping.go             exhaustive boundary/domain enum and value mappings; no ordinal casts
  errors.go              exhaustive typed domain failure -> gameapi ErrorCode mapping
  commands.go            boundary validation and gameapi command -> domain operation mapping
  end_turn.go            one computer-planning/five-phase turn use case
  snapshot.go            domain state/events -> isolated gameapi.Frame projection
  ports.go               outbound CampaignRepository port and immutable completion values
  persistence.go         domain State <-> versioned SaveState mapping, validation, migrations
  autosave.go            WorldRevision, monotonic-time input, bounded trigger coalescing

internal/adapters/storage/ OUTER STORAGE ADAPTERS — implement application.CampaignRepository
  file.go                file-backed JSON records                       (//go:build !js)
  lock_unix.go           process-lifetime advisory save-directory lock  (//go:build !js && !windows)
  lock_windows.go        process-lifetime LockFileEx wrapper             (//go:build windows)
  indexeddb.go           generation-addressed IndexedDB records         (//go:build js)
  contract_test.go       backend-neutral CampaignRepository contract suite

internal/adapters/logging/ OUTER OBSERVABILITY ADAPTER — stdlib + application/gameapi/ui only
  session.go             session identity, stable event/attribute names, lifecycle, panic reporting
  game.go                gameapi.Game decorator for campaign and storage use cases
  repository.go          application.CampaignRepository decorator
  settings.go            ui.UISettingsStore decorator
  sink_desktop.go        unique size-bounded JSONL temp file             (//go:build !js)
  sink_js.go             JSONL stdout -> browser JavaScript console      (//go:build js)

pkg/audio/               stdlib + Ebitengine audio only
  synth.go               PCM tone generation (enveloped sine/square)
  manager.go             SoundManager, lazily constructed on first user gesture

pkg/render/              DRAWING ADAPTER — gameapi + Ebitengine + Tetra3D; no domain/application/ui
  fonts.go               goregular -> text/v2 face, cached at three sizes
  palette.go             biome colors, UI chrome colors, three-anchor epoch grade
  viewport.go            DPI-aware logical-DIP/render-pixel transforms + viewport revision
  terrain.go             bounded 32×32 terrain chunks + per-snapshot vertex recolor
  veil.go                bounded opaque cover geometry for unexplored tiles
  scene3d.go             Tetra3D scene, lighting, model registry
  orbit.go               orbit camera: azimuth / elevation / distance / focus
  picking.go             BoundingTriangles ray hit -> tile resolution
  passages.go            open/locked Wallacea and Beringian route overlays
  markers.go             species-distinct band markers, hover highlight, selection ring
  hud.go                 2D overlay: top bar/timeline, band/tile inspectors, passages, migration ranking, event feed, Field Notes

pkg/ui/                  DRIVING PRESENTATION ADAPTER — gameapi + render + audio; no domain/application
  scene.go               Scene interface + stack router
  action.go              UI actions: sim command, EndTurn, manual/quick save, load/delete, scene navigation
  widgets.go             Button, Label, Slider, ListRow + hit-testing
  toast.go               two-second queued success/error notifications
  title_scene.go
  game_scene.go          input routing, one workforce draft + inline exit guard, Ctrl/Cmd+S or click quick-save; emits typed ui.Action values
  pause_overlay.go       translucent modal over live gameplay
  save_load_scene.go     grouped manual/quick/auto slots, metadata preview, immediate actions
  settings_scene.go      terrain-detail toggle, master-volume slider, mute checkbox
  field_notes.go         bundled sourced entries keyed by gameapi context values; no domain imports
  ui_settings.go         versioned local-preference record and store interface
  ui_settings_file.go    desktop JSON preference store                    (//go:build !js)
  ui_settings_idb.go     separate web IndexedDB preference store          (//go:build js)
  end_scene.go           victory / extinction / dispersal-failed result

pkg/app/                 Ebitengine HOST + COMPOSITION ROOT — outermost driving adapter
  game.go                implements ebiten.Game + LayoutFer; owns gameapi.Game, viewport, current Frame, scenes, action queue
  e2e_summary.go         bounded published-frame summary for the opt-in browser test observer
  wire.go                accepts the observability session; constructs decorated ports, GameService, renderer, audio, and UI graph
  game_test.go           turn-counting + async storage polling/load-freeze tests

internal/verification/   REFERENCE-CAMPAIGN DRIVER — imports application + gameapi + stdlib only
  policy.go              the five deterministic frame-driven sapiens route policies, incl. reference
  checkpoint.go          CheckpointRecord + canonical sorted-key JSON encoding
  run.go                 ReferenceRun(seed, turns, policy) -> []CheckpointRecord; no I/O, no wall clock
  run_js_test.go         js-tagged: emits the sentinel-delimited record block on stdout  (//go:build js)

internal/archtest/
  arch_test.go           parses imports in every .go file, including inactive build tags
```

**Package-organization decision.** Placing the gameplay scene in `render` would force `render` to
import `ui` in order to push the pause overlay, creating an import cycle. All scenes therefore live
in `pkg/ui`, and `pkg/render` stays a drawing adapter. UI-settings persistence remains a presentation
port implemented within `pkg/ui` because it stores local panel preferences, not campaign or domain
state.

### Operational session logging

V1 has pervasive structured operational logging for post-incident diagnosis. “Aspect-oriented” has a
specific, idiomatic-Go meaning here: explicit decorators implement the same ports as the objects they
wrap and are installed once in the composition root. There is no runtime weaving, reflection,
code generation, package-global logger, or logger dependency in `internal/domain` or
`internal/application`. The production object graph is:

```text
platform repository -> repository log decorator -> GameService -> game log decorator -> pkg/app
UI-settings store   -> settings log decorator   -------------------------------> pkg/app / pkg/ui
```

The session owns one `*slog.Logger`. Its decorators and typed lifecycle/host recorder methods are the
only logging surface passed into composition; no other package imports `log` or `log/slog`. A bare
`GameService`, repository, or settings store remains usable in focused tests. The decorators may
observe arguments and returned values but may not call a wrapped operation more than once, request an
extra snapshot, reorder a completion, retry an error, or change a return value. Logging failure is
therefore observational: it cannot reject an action, alter a save, consume a `WorldRNG` draw, or
change simulation state.

Every JSON record uses `slog`'s `time`, `level`, and `msg` fields. `msg` is a stable dotted event name,
not prose; `time` is normalized to UTC and encoded as RFC 3339 with nanoseconds. Every record also has
`session_id` and `target` (`desktop` or `web`); operation records add a process-local monotonic
`operation_id`, `layer`, `operation`, and `outcome`. End records add non-negative floating-point
elapsed monotonic `duration_ms` and, when returned by the executing operation, the campaign turn and
`WorldRevision`. Relevant optional identifiers include command kind, band ID, slot kind/ID, public
storage operation ID, repository operation ID, and stable `gameapi.ErrorCode`. Wall time, elapsed
time, session/operation IDs, and log-wrap count are diagnostic values only: none appears in a
`Frame`, `SaveState`, slot metadata, campaign hash, or deterministic fixture, and none is derived from
`WorldRNG`. The session factory accepts an `io.Reader` entropy dependency; production passes
`crypto/rand.Reader`, and tests pass a fake without replacing a package global. `session_id` is the
lowercase hexadecimal encoding of 16 bytes read with `io.ReadFull(entropy, b)`; if the platform source
fails, the logger emits one warning and uses a process-local fallback composed from the target, UTC
Unix nanoseconds, and an atomic counter.
It deliberately does not call Go 1.26's `crypto/rand.Read`, which irrecoverably terminates the process
on an entropy-source error and would violate the logging-failure rule above. This identifier is for
correlation, not authentication. `operation_id` is a separate atomic counter. The repository decorator keeps
diagnostic start times keyed by the already supplied `RepositoryOpID` so asynchronous completion
duration requires no extra repository call or application state.

`session.start` records the application/module version, VCS revision and modified flag when available
from `runtime/debug.ReadBuildInfo`, Go version, `GOOS`, `GOARCH`, target, selected verification/game
mode, and `log_destination` (the absolute desktop path, `"stderr"` after desktop creation failure, or
`"console"` on web). Missing build metadata is the literal `"unknown"`, never an omitted key. This
makes a retrieved log attributable to a binary without adding release-only linker flags or letting
build metadata enter campaign compatibility.

| Join point                           | Required records                                                                                                                                                  |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Process/session lifecycle            | build-attributed `session.start`, `session.end`, `init.error`, `run.error`, and `session.panic` with a stack before the panic is re-raised                        |
| `gameapi.CampaignUseCases`           | paired `use_case.start` / `use_case.end` records for `Snapshot`, `Apply`, and `EndTurn`; commands log their bounded typed scalar fields, never serialized objects |
| `gameapi.StorageUseCases`            | paired records for accepted/rejected begin operations; `PollStorage` emits only when it returns one or more completions, never once per empty frame poll          |
| `application.CampaignRepository`     | `repository.start`, `repository.enqueued`/`repository.rejected`, and `repository.completion`, with op/slot plus save schema/algorithm IDs but no save payload     |
| `ui.UISettingsStore`                 | `settings.start`, `settings.enqueued`/`settings.rejected`, and `settings.completion` records containing schema version and outcome but not the preference record  |
| Composition and host action dispatch | target/mode and initialization; `action.dispatch` per typed `ui.Action`, `action.rejected`, `scene.transition`, and `host.error`; never raw key/pointer events    |

The host validates a complete action batch before logging dispatch: a rejected batch emits one
`action.rejected` summary and no `action.dispatch` records, while an accepted batch emits one
`action.dispatch` immediately before each typed action is invoked. Logging therefore cannot make a
batch look partially executed when §4's atomic preflight rejected all of it.

`Debug` covers starts, successful snapshots, and successful low-level I/O; `Info` covers session
lifecycle, accepted state-changing use cases, and asynchronous completions; `Warn` covers expected
validation/repository failures, clipping, wrapping, and recoverable fallback; `Error` covers
initialization/run failures and panics. An end record's level follows its outcome, so a failed call is
searchable without finding and joining its start record first.

The minimum level is `Debug` on both targets. “Pervasive” does not mean noisy frame tracing:
`Update`, `Draw`, `LayoutF`, empty storage polls, pointer movement, hover, camera interpolation,
workforce-draft slider movement, tile iteration, and individual RNG draws emit nothing. An explicit
player action, use case, asynchronous completion, lifecycle transition, or error is a join point. A
single JSON record is at most `MaxOperationalLogRecordBytes = 64 * 1024`; longer string attributes or
panic stacks are clipped on UTF-8 boundaries as needed to keep the encoded line within the limit, and
the record gains `truncated=true`; the limit includes the one terminating newline, and the sink must
still emit one valid JSON object plus that newline. Logs never contain a serialized `SaveState`, RNG
state, complete `Frame`, raw IndexedDB/file contents, or free-form Field Notes text. A command records
its kind and bounded scalar payload: actor/target/tile/technology IDs and, for `SetAssignment`, the
five named basis-point values. Other objects record type, identity, size/count, stable error code, and
a bounded error description instead.

On desktop, `main.go` creates the session before constructing any other adapter by calling
`os.CreateTemp(os.TempDir(), "africa2ice-*.jsonl")`. `CreateTemp` supplies an exclusive, unpredictable
name with mode `0600`, so every process launch creates a new file and never appends to or overwrites a
previous session. The entrypoint normalizes the returned name with `filepath.Abs`. Immediately after
creation—and before the first structured record—it prints this single plain-text line to standard
output:

```text
Africa 2 Ice session log: <absolute path returned by os.CreateTemp>
```

The entrypoint writes and synchronizes `session.start` before constructing the remaining object graph,
so even an initialization crash leaves the announced file attributable. The file contains
newline-delimited JSON and is limited to
`MaxDesktopSessionLogBytes = 32 * 1024 * 1024`. Before a record would cross that limit, the sink
holds its write mutex, truncates and seeks the same file back to byte zero, writes `log.wrapped` with
an incremented wrap count, original session-start time, and the same build-attribution fields as
`session.start`, writes the pending record, and synchronizes before releasing the mutex. Thus one
session still owns exactly one bounded file while retaining its most recent diagnostic segment and
enough identity to interpret it. Normal shutdown writes
`session.end`, synchronizes, and closes the file. A deferred panic guard writes `session.panic` with
the bounded stack, synchronizes, and re-raises the original panic; it does not convert a crash into a
successful exit. Abrupt process termination may omit the final record but direct, unbuffered record
writes normally leave prior complete lines inspectable; a kernel or power failure may still lose the
unsynchronized tail. The operating system owns eventual cleanup of its temp directory; the game does
not delete a session log on exit. The guard is also installed at every
project-owned worker-goroutine and JavaScript-callback boundary, because a defer in the entrypoint
cannot observe a panic on another goroutine. The sink and `Close` are concurrency-safe and idempotent;
`Close` emits `session.end` with `outcome=success|error` once only when no panic was recorded. A panic
path writes `session.panic`, synchronizes, and closes without a contradictory normal end marker. A
late adapter record after closure is dropped rather than writing through a closed file or appearing
after the terminal marker.

For asynchronous adapters, composition passes the method value `session.PanicGuard` as an opaque
zero-argument function and each owned goroutine/callback begins with `defer panicGuard()`. Storage and
UI-settings packages therefore gain the required join point without importing the logging adapter or
knowing how a panic is recorded. `PanicGuard` calls `recover` only from that deferred invocation,
records the current goroutine's stack, performs the terminal synchronization, and panics again with
the original value.

Desktop uses `main() { os.Exit(run()) }`; `run` has a named exit-code result and defers
`func() { session.Close(exitCode) }()`, so the final value maps zero/nonzero to the terminal
success/error outcome. It returns only after logging any `RunGame` error. Code below session
construction does not call `os.Exit`,
`log.Fatal`, or another defer-bypassing terminator. Thus ordinary success and reported failure both
synchronize and close the announced file before the outer `os.Exit` selects the process status.
`run` registers the close closure first and `defer session.PanicGuard()` second; Go's LIFO defer order
therefore lets the guard mark/log a panic before `Close` decides whether an end record is appropriate.

Failure to create the desktop temp file is the only exception to the one-file rule: startup prints one
warning to standard error, installs the same JSON handler on standard error, and continues. A later
file write/sync failure similarly reports at most one non-recursive warning to standard error and
falls back there; logging availability never becomes a gameplay dependency. Tests inject the temp
directory and output writers rather than writing to the developer's real temp directory.

On web, the build-tagged sink writes the same JSONL records to `os.Stdout`. The pinned Go
`wasm_exec.js` forwards newline-terminated stdout to `console.log`, so logs appear in the browser's
JavaScript console and create no IndexedDB records, downloads, server requests, or campaign data.
Browser console retention is browser-owned; v1 adds no remote collection or web log export.

This operational stream is distinct from §13's balance telemetry. The former explains what one
interactive process did and may contain nondeterministic timestamps and durations; the latter is
deterministic reference-run output with explicit balance assertions. Tests compare decorated and bare
results and state hashes, never log timestamps or durations.

---

## 4. Read-only domain access

Draw and input code never receives a pointer into live simulation state.

**Downward (state).** `application.GameService` projects a fresh `*gameapi.Frame` after each
accepted planning command, completed turn, or successful load. The frame is a plain value struct
holding campaign turn/year/era/calendar progress, `Season`, the global long-term/seasonal/noise
offsets, the long-term moisture offset with its derived `AridityIndex` and `ClimateEpoch`,
bounded regional abrupt-offset vector, per-tile total offset, and warned/current/elapsed
`MacroEpisodes`, `Passages []gameapi.Passage`,
`SapiensEstablishedRegions []Region`, `Tiles []gameapi.Tile`,
`Bands []gameapi.Band`, closed `CampaignResult` (`Ongoing`, `Victory`, `Extinction`, or
`DispersalFailed`), the bounded `Events []gameapi.Event` feed, and the derived cache key
below, all values with no pointers back
into the `World`.

Each band value carries three groups of fields. **Identity and position:** `BandID`, `Species`, and
current `TileID`. These are not optional presentation details — §7 restricts every player command to
a `HomoSapiens` actor, and §8 requires species-distinct markers, the “Computer controlled” label, and
a top-bar population sum over sapiens bands only. **Persisted state:** current whole-person
`Population uint32`, normalized
`Health float64` in `[0, 1]`, `StoredFood`, fixed five-role basis-point allocation, acquired-tech
bitset, research target/progress, the fixed six-value heritable state, the value-typed
`LastFoodReport FoodTurnReport`, last completed turn's starvation, seasonal, chronic, macro-event, and
acute mortality breakdown, and the bounded `LastOutcomeReport` containing that turn's population
and health endpoints plus causal components, the `SpatialActionUsed` marker, and the queued-migration destination plus
presence flag. The projection exposes that already-persisted intent so presentation can show what
the player selected; it adds no second queue or simulation authority. **Derived previews:**
`OriginalResearchGainPreview`, the current tile's seasonal/chronic mortality-rate preview, the fixed nine-entry projected `ResearchOptions`
availability/acquired/current-target view, the ranked `MigrationCandidates` with destination-specific
seasonal/chronic mortality-rate previews and an arrival crowding-decline preview, the freshly allocated co-located
archaic `InterbreedCandidateIDs`, a fixed three-entry `PassageStatuses` array, and `Stress`. Together these let the HUD explain the band's
capabilities, food outcome, and population change.

Three of those fields exist solely because presentation rules in §7 and §8 depend on them, and a
`Frame` that omits them cannot drive the specified UI. `Stress` is what gates the non-modal
spatial-pressure alert and the enabled/disabled state of the split control at the
`SplitStressThreshold` boundary;
the UI reads the projected value rather than recomputing `EcologicalK · T_tech`. `SpatialActionUsed`
is what lets the inspector disable split and migration after an accepted `Interbreed` while leaving
workforce and research controls live. Both are authoritative in the domain: the projected copies
explain the aggregate's answer, never substitute for its validation. The preview is the `saturating-toolcraft-v1`
contribution under the current planning assignment, capped at the target's remaining cost, if the
band survives the turn; it excludes contact diffusion. Each tile value also exposes its fixed finite
`ElevationKm`, current `Biome`, current finite `LocalTemperatureC`, destination `MovementCost`, fixed
`NaturalShelter` rating, and a fixed-size `FaunaSummary`
containing the current profile's prey-group weights and ordinary-hunting/megafauna support flags,
plus the tile's authoritative `Explored` flag and any currently visible macro-impact summary. The
renderer reads `ElevationKm` directly rather than importing or rerasterizing geography. These are environment values,
not band-specific yield predictions or animal
counts. The inspector explains shelter efficiency and local prey opportunities from the frame,
without importing domain tables or deriving its own geography/ecology rules. The stats layout in
§8 places these values in the top bar and band/tile inspectors. `LastFoodReport` and
`LastOutcomeReport` are bounded, persisted display records specified in §7, not future-turn previews
or simulation inputs. Frame projection copies them without recomputing consumption, demographics,
health, or history from current state; UI and rendering only format their values. Projection requires
`0 <= Turn <= 400` and derives the campaign year, era, and calendar progress through
`CampaignDate` before publication; an invariant
failure returns an error rather than exposing a clamped or internally inconsistent frame.
Render and UI hold that frame until `pkg/app` replaces it wholesale.

**Derived cache key.** The frame carries one monotonic `uint64` counter so drawing code can skip
the one rebuild expensive enough to be worth skipping. `TerrainRevision` changes only when a tile's
rendered geometry inputs change — its biome, `Explored` bit, or visible macro-impact factors. It is
not `WorldRevision`: `WorldRevision` increments on every accepted planning mutation, including
`SetAssignment`, so keying terrain rebuilds off it would recolor all six chunks on every workforce
Apply — exactly the cost §8's recolor rule exists to avoid. The application computes the key while
projecting, from the same state it is already walking; it is a derived frame value, never persisted,
never hashed, and never a simulation input. `pkg/render` compares it against the value it last drew
and rebuilds only the affected caches. It starts at `1` on the first published frame and increments
only when its named projected inputs differ. It belongs to the long-lived application projector and
is not reset when a load or new campaign replaces the world; such a replacement therefore increments
it whenever it replaces those inputs. `TerrainRevision` is a coarse dirty signal,
not proof that every terrain-related cache changed: after it advances, each cache compares its own
inputs, so a biome-only recolor does not rebuild the exploration veil and an exploration-only change
does not rewrite unchanged biome colors. §8's epoch grade is deliberately not among these inputs:
it is a per-frame uniform on lighting and chrome, so a changing `AridityIndex` never dirties a
terrain cache.

**Band markers need no such key.** A frame is published only after an accepted command, a completed
turn, or a successful load — never on an idle update or draw — so the renderer already rebuilds
band markers at most once per accepted action, and there are at most `MaxBands = 256` of them. A
`BandRevision` counter would guard that bounded work by making the projector diff every band value
in the frame against the previous projection on every publish, and its only discriminating case
would be "a new frame in which terrain changed but no band did", which a completed turn essentially
never produces. The diff would cost more than the rebuild it saves, so v1 carries one cache key,
not two.

**Upward (intent).** UI emits `ui.Action` values. Editing a workforce slider changes only the
selected sapiens band's UI-local draft; it is not an action. Pressing its enabled Apply control
emits one simulation-command action wrapping a complete `SetAssignment` vector. Other
simulation-command actions wrap one of `QueueMigration`, `SplitBand`, `ResearchTech`, or
`Interbreed`;
application actions cover `EndTurn`, save/load/delete, and scene navigation. The shared UI guard
in §7 blocks any selection/inspector exit, actual load/title transition, or `EndTurn` attempted
while the workforce draft is dirty, before changing UI context or emitting the corresponding
action. It displays inline Apply/Discard controls and retains no deferred action; only fresh input
after resolution may proceed. Player simulation commands may target only
`HomoSapiens` bands. `app.Game.Update()` drains the batch in order. It sends simulation commands to
`game.Apply(cmd)`, where `game` implements `gameapi.Game`, but calls `game.EndTurn()` **only** when
it receives `EndTurn`.
`EndTurn` must be the final action in a batch; more than one in the same Ebitengine update
is rejected. Pause, camera input, drawing, saving, and loading do not advance time.
Save/load/delete/list actions begin asynchronous storage operations; `Game.Update()` polls completed
operations without waiting. A successful load replaces the frame from loaded state without taking a
turn.

The host adapter reconciles Ebitengine's fixed-rate `Update()` callback with turn-based application
use cases. Calls into the use-case port happen at one point in `app.Game.Update()`, outside every
draw call. The UI never receives `gameapi.Game`, `application.GameService`, or `*domain.World`;
`pkg/app` depends on the `gameapi.Game` inbound port so the loop can be tested with a counting fake.

The contract needed by that host adapter is deliberately explicit:

```go
type CampaignUseCases interface {
 Snapshot() (*Frame, error)
 Apply(Command) (*Frame, error)
 EndTurn() (*Frame, error)
}

type StorageUseCases interface {
 BeginSave(slot int) (StorageOpID, error)
 BeginLoad(slot int) (StorageOpID, error)
 BeginDelete(slot int) (StorageOpID, error)
 BeginListSlots() (StorageOpID, error)
 PollStorage() []StorageResult
}

type Game interface {
 CampaignUseCases
 StorageUseCases
}
```

The application-owned outbound port is separate from that driving contract:

```go
type CampaignRepository interface {
 BeginWrite(RepositoryOpID, SlotID, SaveState) error
 BeginRead(RepositoryOpID, SlotID) error
 BeginDelete(RepositoryOpID, SlotID) error
 BeginList(RepositoryOpID) error
 Poll() []RepositoryCompletion
}
```

These parameters and results are immutable application values. `GameService` owns queueing,
autosave coalescing, public operation IDs, load freezes, schema/domain validation, and mapping to
`gameapi.StorageResult`; the repository owns only asynchronous JSON and platform commit mechanics.
No repository method returns or accepts `*domain.World`.

`StorageResult` contains the operation ID, operation kind, slot, captured world revision, optional
slot metadata, current `StorageWritable` capability, an optional replacement frame, and an error.
The loading/list result publishes the capability before mutation controls enable; `BeginSave` and
`BeginDelete` also reject with the stable read-only error if the lease/lock is absent. Repository adapters place completion
values into a private FIFO; only `application.GameService.PollStorage`, called from `Game.Update`,
may validate a decoded loaded `SaveState`, construct a candidate with `domain.RestoreWorld`, and
replace the live aggregate. JavaScript callbacks and desktop worker goroutines never touch `World`,
`Frame`, scenes, or UI state directly. `StorageOpID` comes from a process-local monotonic counter;
it is excluded from saves and state hashes and never consumes `WorldRNG`.

Beginning a save materializes an immutable `SaveState` snapshot on the game goroutine before the
backend starts, so later turns cannot change what that operation commits. Saves may finish while
gameplay continues. Accepting a load request, including one queued behind another storage operation,
freezes simulation-changing actions and workforce draft editing until that load completes. The
dirty-draft guard runs before a load can start or enter the queue. Failure preserves the old world
and clean editor; success swaps the validated world, refreshes its frame, and clears the old world's
selection/draft. Input unfreezes only when no accepted load remains active or queued; no new dirty
draft can be created against a world that is awaiting replacement.
The application service permits one active storage operation and at most one explicit user
operation queued FIFO, making save-then-load and delete-then-list ordering identical on desktop and
web. Autosave pressure
is represented separately by the single `autoNeeded` flag in §9. Additional or duplicate clicks are
rejected with a two-second “storage operation already pending” toast, so neither user input nor
autosave triggers can create an unbounded queue.

`ui.Action` is a tagged value containing an action kind plus only the relevant payload
(`gameapi.Command` or slot number). A batch belongs to exactly one closed class: a **campaign batch**
contains zero or more simulation commands followed by at most one final `EndTurn`; a **storage
batch** contains exactly one save, load, delete, or list action; and a **navigation batch** contains
exactly one scene-navigation action. Classes may not mix, storage actions may not repeat, and no
action follows `EndTurn`. `pkg/app` preflights the entire batch's tags, payload shapes, and class
before invoking a use case, so a structural or ordering error invokes none of its members.

After a valid campaign preflight, actions execute in order. If a command returns a semantic domain
error, execution stops, later commands and any trailing `EndTurn` are skipped, already accepted
earlier commands remain applied, and the host publishes their latest returned frame; v1 does not
promise transactional rollback across independent player commands. A storage or navigation batch
has only one semantic result. Tests cover every legal class, every forbidden mixture, early/middle/
late semantic failure, and the exact resulting call sequence. The UI omits mutation controls for archaic bands, but that
is only presentation. `GameService.Apply` validates the boundary command and calls
`World.PlanPlayer`; the aggregate authoritatively rejects every external command whose acting
`BandID` is not `HomoSapiens` with the typed `ComputerControlledBand` domain error. `GameService`
maps it to `gameapi.ErrComputerControlledBand`, and the UI owns the “computer-controlled band” copy.
Computer authority is never carried in a forgeable `gameapi.Command`. `World.AdvanceTurn` invokes
its internal archaic policy exactly once before phase 1, as specified in §7.

### Why snapshots rather than read-only getter interfaces

- A `WorldView` interface **can be type-asserted back** to `*domain.World`; on its own it is
  advisory, not a guarantee. The real guarantee comes from depguard allowing only
  `internal/application` to import `internal/domain` — and once that rule exists, the simpler value type is strictly
  better than a getter interface.
- Values are immune to torn reads if the simulation ever moves off the main goroutine.
- No getter-call overhead in per-frame loops.

**Bounded snapshot cost.** Projection work is:

```text
O(
    len(Tiles) * (1 + FaunaGroupCount)
  + len(Bands) * (10 + TechCount + HeritableTraitCount + AssignmentCount + PassageCount)
  + CrossSpeciesCoLocatedPairs
)
```

Projection runs when an accepted simulation command changes planning state, when a turn completes,
or when a load succeeds — never on an idle Ebitengine update, draw, or draft slider edit. An accepted
assignment Apply creates one snapshot; its preceding draft edits create none. `MaxBands = 256` bounds
candidate plus research-progress and heritable values to
`256 * (10 + 9 + 6 + 3) = 7_168`, and assignment entries to `256 * 5 = 1_280`. The sapiens-only
co-located interbreeding-candidate lists contain at most
`floor(MaxBands^2 / 4) = 16_384` cross-species pairs. Renderer caches key off snapshot revisions; tile
fauna summaries contain at most `6_144 * FaunaGroupCount` copied weights. Food and outcome reporting
each add at most 256 fixed-size value copies, not a growing turn history. Terrain colors update only when tile/biome
state changed; assignment/research-only snapshots update band/HUD data without rebuilding terrain
colors.

### The honest limit of this guarantee

`Frame` contains slices, including `Passages`, `SapiensEstablishedRegions`, `Events`, and nested
`Band.MigrationCandidates` and `Band.InterbreedCandidateIDs`, so draw code _can_ write to them. What it cannot do is reach the
simulation by doing so: writing to a frame corrupts only that frame, which is discarded at the next
snapshot replacement. Snapshot projection never exposes or aliases domain-owned storage: every slice
in a published frame has backing memory the application owns outright. This is
**isolation**, not deep immutability.

For v1, ownership includes the lifetime of published backing memory: every projection allocates
fresh backing for every slice in the returned frame, and application code never recycles a published
buffer. A caller may retain and mutate any older frame after arbitrarily many later projections
without reaching the domain or another frame. This is intentionally stricter than the host's normal
one-current-frame usage because `gameapi.Game` returns an ordinary `*Frame` and exposes no lease,
release, or invalid-after-swap contract. A finite pool cannot make arbitrary retention safe.

The allocation volume is bounded and occurs only after accepted commands, completed turns, or loads,
not on idle updates or draws. Measure it on `js/wasm` before weakening the ownership rule. A future
optimization may add an explicit lease/release API and version its lifetime contract, but cannot
silently reuse backing behind the v1 port. True field-level immutability would still require getter
methods on every value; fresh backing supplies the required isolation without that per-frame-loop
cost.

---

## 5. Enforced package boundaries

`.golangci.yml` (golangci-lint v2 config format):

```yaml
version: "2"
linters:
  enable: [depguard, forbidigo]
  settings:
    depguard:
      rules:
        gameapi-must-stay-dependency-free:
          list-mode: strict
          files: ["**/pkg/gameapi/**/*.go"]
          allow: ["$gostd"]

        domain-is-the-dependency-center:
          list-mode: strict
          files: ["**/internal/domain/**/*.go"]
          allow: ["$gostd"]

        application-points-inward:
          list-mode: strict
          files: ["**/internal/application/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/internal/domain"
            - "github.com/adsouza/africa2ice/pkg/gameapi"

        only-application-may-import-domain:
          list-mode: lax
          files:
            - "**/*.go"
            - "!**/internal/domain/**/*.go"
            - "!**/internal/application/**/*.go"
          deny:
            - pkg: "github.com/adsouza/africa2ice/internal/domain"
              desc: "only application use cases may touch the domain; adapters use ports and DTOs"

        only-observability-adapter-may-import-log:
          list-mode: lax
          files:
            - "**/*.go"
            - "!**/internal/adapters/logging/**/*.go"
          deny:
            - pkg: "log$"
              desc: "route operational events through the injected observability session"
            - pkg: "log/slog$"
              desc: "route operational events through the injected observability session"

        storage-adapters-implement-inward-ports:
          list-mode: strict
          files: ["**/internal/adapters/storage/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/internal/application"

        logging-adapter-observes-ports-not-domain:
          list-mode: strict
          files: ["**/internal/adapters/logging/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/internal/application"
            - "github.com/adsouza/africa2ice/pkg/gameapi"
            - "github.com/adsouza/africa2ice/pkg/ui"

        domain-randomness-goes-through-world-rng:
          list-mode: lax
          files:
            - "**/internal/domain/**/*.go"
            - "!**/internal/domain/rng.go"
          deny:
            - pkg: "math/rand"
              desc: "domain randomness must go through the aggregate-owned serializable WorldRNG"

        render-adapter-dependencies:
          list-mode: strict
          files: ["**/pkg/render/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/pkg/gameapi"
            - "github.com/hajimehoshi/ebiten/v2"
            - "github.com/solarlune/tetra3d"
            - "golang.org/x/image"

        ui-adapter-dependencies:
          list-mode: strict
          files: ["**/pkg/ui/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/pkg/gameapi"
            - "github.com/adsouza/africa2ice/pkg/render"
            - "github.com/adsouza/africa2ice/pkg/audio"
            - "github.com/hajimehoshi/ebiten/v2"

        audio-is-a-leaf:
          list-mode: strict
          files: ["**/pkg/audio/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/hajimehoshi/ebiten/v2"

        verification-drives-the-inbound-port:
          list-mode: strict
          files: ["**/internal/verification/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/internal/application"
            - "github.com/adsouza/africa2ice/pkg/gameapi"

        ebitengine-host-is-the-composition-root:
          list-mode: strict
          files: ["**/pkg/app/**/*.go"]
          allow:
            - "$gostd"
            - "github.com/adsouza/africa2ice/internal/application"
            - "github.com/adsouza/africa2ice/internal/adapters/logging"
            - "github.com/adsouza/africa2ice/internal/adapters/storage"
            - "github.com/adsouza/africa2ice/pkg/gameapi"
            - "github.com/adsouza/africa2ice/pkg/render"
            - "github.com/adsouza/africa2ice/pkg/ui"
            - "github.com/adsouza/africa2ice/pkg/audio"
            - "github.com/hajimehoshi/ebiten/v2"

    forbidigo:
      analyze-types: true
      forbid:
        - pattern: '^rand\.(Int|Int31|Int31n|Int63|Int63n|Intn|Int32|Int32N|Int64|Int64N|IntN|Uint|Uint32|Uint32N|Uint64|Uint64N|UintN|Float32|Float64|N|Perm|Shuffle|Read|ExpFloat64|NormFloat64)$'
          msg: "use domain.WorldRNG; package-level random sources break save determinism"
        - pattern: '^rand\.New$'
          msg: "WorldRNG owns a *rand.PCG directly; *rand.Rand's draw mapping is not a Go compatibility guarantee"
```

Four details make these rules architectural rather than cosmetic:

- **depguard `deny` entries are prefix matches** unless suffixed with `$`. A rule denying
  `.../internal/domain` therefore also denies any future domain subpackage. The global deny rule
  exempts only `internal/domain` itself and `internal/application`; even the composition root cannot
  bypass the application use cases.
- The `gameapi`, domain, application, storage, logging, render, UI, audio, verification, and host
  rules are **strict allowlists**.
  Every import not matched by an explicit architectural dependency or `$gostd` is denied, so a
  framework, storage adapter, or UI dependency cannot silently enter the wrong layer.
  `pkg/gameapi` is a top-level
  boundary package rather than a domain subpackage because adapter-facing DTOs are not domain types.
- The global lax logging rule applies in addition to each package's strict allowlist and exempts only
  `internal/adapters/logging`. Exact `log$` and `log/slog$` denials keep both standard packages out of
  every other package without accidentally matching an unrelated import whose path merely begins
  with `log`. The source-level backstop reasserts the same two-path rule.
- **`files` globs must be prefixed `**/`**, because they are matched against absolute paths.

The `forbidigo` rules use type analysis, so aliasing `math/rand/v2` does not bypass them. The first
enumerates every package-level draw function in both `math/rand` and `math/rand/v2`, including the
unsigned variants, while leaving constructors such as `rand.NewPCG` legal. depguard additionally
makes `rng.go` the only domain file allowed to import either random package. Calls on the owned
`WorldRNG` stay legal.

The second rule is anchored `^rand\.New$`, so it forbids constructing a `*rand.Rand` while leaving
`rand.NewPCG` untouched. §7 gives the reason: `WorldRNG` derives its unit draw from raw PCG output
in project-owned code, because `Rand.Float64`'s mapping from generator output to `[0, 1)` is not
covered by Go's compatibility promise and a toolchain bump could change every draw in the campaign
without a compile error. The rule is what keeps a later convenience call from quietly reintroducing
that dependency.

**Backstop.** `internal/archtest/arch_test.go` walks every repository `.go` source file, parses its
imports with `go/parser`, classifies it by directory, and re-asserts the rules above. It deliberately
does **not** use `go list`, because a host-side `go list` omits `//go:build js` files. The source walk
therefore checks desktop, web, test, and currently inactive build-tagged files under plain
`go test ./...`, even on a machine with no linter installed. depguard gives fast editor feedback;
the source-level test is the cross-target hard gate.

**Every floating-point product chain in the domain is explicitly rounded where it ends.** `arch_test.go` carries three rules
about arithmetic rather than imports — the product-rounding rule, the closed `math` allowlist, and
the float-to-integer conversion gate — of which the first exists for the reason §7 gives: Go may fuse `a*b + c` into a single
operation, and may do so even across statements such as `p := a*b; r := p+c`. A rule that inspects
only multiplication directly nested under addition would miss a form the language explicitly permits
the compiler to fuse.

A floating-point product appears either as a multiplication expression or as compound assignment, and
the rule visits both. The test type-checks `internal/domain` and uses `types.Info` to inspect the result
type of every `*ast.BinaryExpr` whose operator is `*` and the left-hand expression of every
`*ast.AssignStmt` whose token is `*=`. A type is floating when its underlying basic type is `float32`
or `float64`, so named floating types cannot bypass the gate. A type-parameter result whose type set
admits either floating type is rejected outright; generic floating arithmetic is outside this domain
contract. The compound form is not a `BinaryExpr` and contains no multiplication expression, so a rule
written against expressions alone never visits it — yet `x *= y; r := x + c` is precisely the
cross-statement shape this rule exists to catch.

The gate does not rewrite source. For a simple identifier `x` of exact static type `T`, the required
form is `x = T(x * y)`, such as `x = float64(x * y)` or `x = float32(x * y)`. An indexed, selected,
map-indexed, or dereferenced left side must be expanded in a way that evaluates every address, receiver,
index, and map expression exactly once, preserving Go's compound-assignment semantics; copying the
left-side source text into both sides of `=` is forbidden when any part can have side effects.

A product is rejected unless its immediate parent is the sole argument of an explicit conversion to
the same floating-point type. Thus
`float64(a*b) + c` and `p := float64(a*b); r := p+c` are accepted, while `a*b + c`, `p := a*b`,
`p := a*b; r := p+c`, `*p = a*b; r = *p+c`, `x *= y`, and `float64(a*b + c)` all fail. This deliberately
rounds products that never happen to reach a sum: the stronger syntactic rule is what makes later
refactoring safe without data-flow analysis. Integer multiplication remains legal without a
conversion. Division is not implicated because no fused multiply-divide operation exists.

**First exemption: a product the type-checker has already folded to a constant.** The rule skips any
`*ast.BinaryExpr` for which `types.Info.Types[expr].Value` is non-nil — that is, any product whose
value Go computes at compile time. Constant arithmetic in Go is exact and arbitrary-precision, is
performed by the compiler rather than the target's floating-point unit, and is rounded once when the
constant is converted to its type; there is no runtime multiplication for an `FMADDD` to absorb, so
there is nothing for this rule to prevent.

Without the exemption the rule fires inconsistently on expressions that differ only in whether a
constant carries a type. An untyped `const Q = 0.5 * 2.0` escapes, because its `types.Info` type is
`UntypedFloat` and the rule tests for underlying `float32`/`float64`; but `const Q float64 = 0.5 * 2.0`
and the typed constant derivations in this document's configuration tables are rejected, and must be
rewritten as `float64(0.5 * 2.0)` — a wrapper that buys nothing and that makes the manifest's
literals harder to read against the values they instantiate. A gate whose output depends on whether
a constant was written typed is teaching the wrong lesson about where the hazard is. The
`Value != nil` test is exactly the compile-time/run-time boundary the hazard follows, and it costs
one field lookup on information the pass already has.

**Second exemption: a product whose immediate parent is another multiplication.** In a chain such as
`a*b*c`, only the outermost product can reach an addition; no fused multiply-multiply operation
exists on the supported targets, so the inner `a*b` has no fusion opportunity and needs no separate
conversion. In `float64(a*b*c) + d`, the outer conversion is the language-level rounding barrier that
prevents the chain's final multiplication from fusing with the addition. For the typed, non-constant
runtime operands this rule checks, wrapping the inner product as well —
`float64(float64(a*b)*c)` — is redundant and noisier rather than a determinism improvement; the first
multiplication already rounds on each supported target. The rule therefore accepts a `*` whose parent
is a `*`, and requires the conversion only where a chain ends. `P_start * BaseWaterPerPerson *
WaterDemandMultiplier` is written `float64(P_start * BaseWaterPerPerson * WaterDemandMultiplier)`.

The same typed source walk applies a closed allowlist to calls resolved into package `math` from
non-test `internal/domain` files, even when `math` is imported under an alias.

**The admission criterion, so the list can be extended without relying on current implementations:** a
`math` function may be allowed only when the Go package contract uniquely determines its exact result
bits for every input reachable from valid production state, including signed zero and any permitted
non-finite value, or when the project supplies its own deterministic implementation or checked-in
table with exact-bit cross-target oracle tests. An IEEE 754 property, current compiler lowering, or
matching results on today's CI matrix is not by itself a Go-level portability contract: package
[`math`](https://pkg.go.dev/math#pkg-overview) explicitly does not guarantee bit-identical results
across architectures.

The current production allowlist is `Abs`, `Copysign`, `Float64bits`, `Float64frombits`, `IsInf`,
`IsNaN`, `Max`, `Min`, and `Signbit` — bit manipulation or exact comparisons whose outputs are fixed
for the values their validated call sites admit. `Sqrt`, `Floor`, `Ceil`, `Trunc`, `Round`, and `Mod`
are **not** pre-approved; adding any one requires a function-specific proof against the criterion
above, not an appeal to IEEE 754 or an instruction available on `arm64`, `amd64`, or wasm. Package
constants such as `Pi` and `Sqrt2` are not calls and remain legal.

Everything else fails the gate, and the criterion says why rather than leaving it to precedent.
`FMA` is the sharpest case: it is exactly, deliberately the unrounded multiply-add the product rule
forbids, so it must stay banned however exact it is. `Sin`, `Cos`, and `Sincos` are approximations
whose Go implementations carry no cross-architecture bit-identity guarantee — the reason the
checked-in tables exist. `Hypot`, `Exp`, `Log`, and `Pow` fail for the same reason and cannot be
admitted by arguing that they are accurate; accuracy is not the test, specified identity is.
Domain `_test.go` files may call `Sin`/`Cos` only to compare the checked-in tables within tolerance;
those calls cannot enter a production simulation path.

**A third rule: no floating-point value is converted to an integer type in `internal/domain` except
by `RoundPopulation`.** The same typed source walk rejects any conversion whose operand type
underlies to `float32` or `float64` and whose target type underlies to any integer kind, in production
and test files alike, unless it is the return conversion inside the named helper in `quantities.go`.

This closes the last cross-target divergence class the other two rules leave open. Go specifies that
in a non-constant conversion, "if the result type cannot represent the value the conversion succeeds
but the result value is **implementation-dependent**" — so a single `int(x)` on a value that is NaN,
infinite, or merely out of range is free to differ between `amd64`, `arm64`, and wasm, silently and
without any of the rounding discipline above applying. It is a narrower hazard than FMA fusion only
because it needs an out-of-range operand to fire, and a sharper one because when it does fire the
two targets do not differ by an ULP; they differ by an arbitrary amount.

Mortality estimates, worker counts, food, and water remain continuous quantities. Actual population
is `type Population uint32`, bounded by `MaxPopulation = 2^32 - 1`; `gameapi.Band.Population` and
`BandSave.Population` are also `uint32`. Per-tile, save-metadata, and top-bar population totals use
`uint64`, whose capacity exceeds `MaxBands * MaxPopulation`. A migration candidate's destination
population is likewise a `uint64` count; scoring converts that already-bounded total to `float64`
only at the formula call.

`RoundPopulation(value, rng)` first rejects NaN, infinity, negative values, and values greater than
`float64(MaxPopulation)`. Only then does its single `Population(value + rng.Float64())` conversion
execute. Because `value` is non-negative and every possible truncated integer part is representable
by `uint32`, Go's otherwise implementation-dependent out-of-range case is unreachable; adding a draw
from `[0, 1)` before truncating implements unbiased stochastic rounding. The draw is the reason the
helper takes the aggregate-owned `WorldRNG` rather than reading a package-level source: the sequence
has to belong to the campaign so that a restored save resumes it exactly. The helper returns
`(Population, error)`, and phase-3 demographics, macro-event loss, and acute loss must propagate the
error before publication. Raw demographic terms retain fractional precision, so the model rounds
once at each population-changing checkpoint rather than inside each formula.

Discrete simulation state — population, tile IDs, band IDs, the turn, the exploration bitset, and
`AllocationBP` — remains integer end to end outside the named demographic checkpoint. Odd-population
splits use integer `P/2` for the new descendant and give the one-person remainder to the source,
conserving the total without any conversion. The rule stops a later display-shaped edit such as
`int(workers)` from introducing another unchecked conversion where no fixture would catch it.

No second helper or inline cast may turn a continuous quantity into discrete state. A future domain
rule needing another such conversion must add a separately audited helper that validates finiteness
and range before converting, take an archtest exemption by name, and carry its own cross-target
fixture — the same admission discipline the `math` allowlist uses, for the same reason.

Two properties make this cheap here rather than a research project. First, the rule needs types, and
`internal/domain` imports only the standard library — enforced by `domain-is-the-dependency-center`
above — so `go/types` can check it with a stdlib importer alone. Maintaining the AST-parent stack,
visiting the two product forms, and resolving conversions and package-qualified `math` calls require
no data-flow framework, `golang.org/x/tools`, or
`packages.Load`, and therefore do not reintroduce the `go list` build-tag blindness that the import
walk exists to avoid. All three rules share that one type-check pass and that one parent stack. The dependency-center rule turns out to pay for itself twice. Second, no file
in `internal/domain` carries a build tag, so one type-check pass covers every target; the multi-tag
handling stays confined to the import walk, which needs no type information.

All three rules are scoped to `internal/domain` because that is where simulation arithmetic lives and where
the campaign-state hash is ultimately determined. Presentation and adapter code may compute
`a*b + c`, call any `math` function, and convert a float to an int freely: a fused camera transform,
a `math.Round` in a layout calculation, and an `int(x)` in a formatter change no simulation result.
The test reports the file, line, and the unrounded product, forbidden call, or float-to-integer
conversion, so a violation reads as a specific edit rather than a category of sin.

---

## 6. World and geography

### Geography as coordinates, not pixel art

`geodata.go` holds real lat/lon vertices, rasterized deterministically at world creation:

- **Landmass polygons:** Africa, Arabia, the Levant/Anatolia, **Frangistan**, the rest of Eurasia
  through Siberia, India, SE Asia, Sahul, the Beringian coasts, and western Alaska.
- **Water/negative polygons:** the Mediterranean, Red Sea, Persian Gulf, Caspian, Black Sea, the
  deep-water Wallacea gaps between Sunda and Sahul, and the Bering Strait. The last two are named
  route gaps rather than coastlines accidentally erased by coarse rasterization.
- **Height-valued highland polygons:** Atlas, Ethiopian Highlands, Zagros, Caucasus, Himalaya / Tibetan Plateau, Alps,
  Urals, Altai, Central Range of New Guinea, and Alaska Range, using the authored kilometre values
  below.
- **River polylines:** Nile, Niger, Congo, Zambezi, Tigris–Euphrates, Indus, Ganges, Danube, Yellow
  (Huang He), Amur, Lena, and Yukon — rasterized as high-moisture corridors.
- **Natural-shelter regions:** a closed, authored geographic layer representing the relative
  availability of caves and rock overhangs at this map's scale, not individual archaeological sites.

**Projection** (`geo.go`): the map is **96×64**, widened from the earlier 64×64 crop so adding the
far-northeastern route does not erase Africa's internal geography. `x=0 → 20°W` and `x=95 → 200°E`
(`160°W`) using monotonically increasing unwrapped longitudes across the date line;
`y=0 → 72°N` and `y=63 → 48°S`. This puts the East African Rift near `(24, 35)`, Bab-el-Mandeb near
`(27, 31)`, Sinai near `(23, 22)`, the central Yellow River Basin near `(57, 19)`, northern Sahul
near `(70, 44)`, and the Bering Strait near `(91, 3)`. The important route landmarks therefore
remain separated by multiple tiles and land where a player expects them.

**Exact v1 coordinate catalog.** Coordinates below are `(longitude, latitude)` in degrees; west and
south are negative, and Alaska longitudes use the unwrapped `180–200°E` range. A polygon ring closes
from its last vertex to its first. Rings use the even-odd rule, count a point on an edge as inside,
and are evaluated with signed integers at `PolygonCoordinateScale = 10` units per degree, so every
listed decimal is exact. Land is the
union of the land rings minus the union of the water rings; water wins on a shared boundary.
Rasterization tests evaluate each tile center and freeze the resulting 6,144-value land mask.

| Stable ID | Land ring vertices, in order |
| --------- | ---------------------------- |
| `Africa` | `(-17,37), (10,37), (25,33), (35,30), (52,12), (44,-12), (34,-35), (18,-35), (10,-25), (0,-5), (-17,15)` |
| `Arabia` | `(34,32), (58,30), (57,16), (44,12), (34,26)` |
| `LevantAnatolia` | `(25,42), (45,42), (45,30), (34,27), (28,32)` |
| `Frangistan` | `(-10,36), (25,36), (45,42), (40,60), (20,70), (-10,60)` |
| `Eurasia` | `(25,36), (45,42), (60,31), (92,20), (125,18), (160,48), (180,58), (176,70), (35,70)` |
| `SouthAsiaPeninsula` | `(64,26), (91,25), (86,7), (76,6), (67,18)` |
| `SoutheastAsia` | `(90,25), (122,23), (132,7), (126,-9), (104,-10), (96,7)` |
| `Sahul` | `(112,-10), (154,-8), (160,-44), (113,-45)` |
| `NortheastSiberia` | `(155,50), (180,52), (188,66), (176,72), (158,70)` |
| `WesternAlaska` | `(190,52), (200,54), (200,72), (184,70), (184,61)` |

| Stable ID | Water/negative ring vertices, in order |
| --------- | -------------------------------------- |
| `Mediterranean` | `(-6,36), (0,42), (10,45), (20,44), (30,41), (37,36), (32,31), (20,31), (10,35), (0,35)` |
| `RedSea` | `(32,29), (37,30), (44,13), (39,12), (34,22)` |
| `PersianGulf` | `(47,31), (57,30), (57,24), (49,24)` |
| `Caspian` | `(46,47), (55,47), (55,36), (47,36)` |
| `BlackSea` | `(27,47), (42,47), (42,40), (28,40)` |
| `NorthWallaceaGap` | `(119,4), (137,4), (137,-7), (119,-7)` |
| `SouthWallaceaGap` | `(121,-7), (139,-7), (139,-16), (121,-16)` |
| `BeringStrait` | `(178,68), (193,68), (193,61), (178,61)` |

Highland rings use the same polygon rule and the heights in the next table:

| Highland feature | Exact ring vertices |
| ---------------- | ------------------- |
| Atlas | `(-10,36), (11,36), (11,28), (-10,28)` |
| Ethiopian Highlands | `(33,15), (43,15), (43,4), (33,4)` |
| Zagros | `(43,38), (57,38), (57,27), (43,27)` |
| Caucasus | `(37,46), (51,46), (51,39), (37,39)` |
| Himalaya / Tibetan Plateau | `(69,37), (105,37), (105,34), (101,34), (101,26), (69,26)` |
| Alps | `(4,49), (17,49), (17,43), (4,43)` |
| Urals | `(54,68), (69,68), (69,50), (54,50)` |
| Altai | `(79,54), (99,54), (99,43), (79,43)` |
| Central Range of New Guinea | `(130,0), (151,0), (151,-11), (130,-11)` |
| Alaska Range | `(184,69), (200,69), (200,56), (184,56)` |

River entries are exact ordered polylines. Project each vertex with the grid transform, then apply
the integer supercover-line algorithm: visit every tile square intersected by a segment; at an exact
corner crossing include both side-adjacent cells in increasing tile-ID order. Union duplicate tiles.

| River | Exact polyline vertices |
| ----- | ----------------------- |
| Nile | `(31,-3), (32,5), (31,15), (32,24), (31,31)` |
| Niger | `(-11,11), (-5,14), (2,13), (7,10), (6,5)` |
| Congo | `(27,-11), (25,-4), (20,1), (15,-4), (12,-6)` |
| Zambezi | `(24,-12), (28,-17), (33,-18), (36,-19)` |
| TigrisEuphrates | `(39,38), (42,34), (45,31), (48,29)` |
| Indus | `(74,34), (72,29), (69,24)` |
| Ganges | `(78,30), (84,27), (90,24)` |
| Danube | `(9,48), (19,47), (27,45), (29,44)` |
| Yellow | `(96,35), (103,37), (111,35), (119,37)` |
| Amur | `(121,50), (132,49), (141,52)` |
| Lena | `(106,53), (118,61), (126,69)` |
| Yukon | `(191,61), (197,64), (200,63)` |

Region resolution is an ordered total function over land tile centers. The first matching rule wins:

1. `Beringia`: longitude `>= 165` and latitude `>= 50`;
2. `YellowRiverBasin`: longitude in `[96, 122]` and latitude in `[30, 42]`;
3. `EastAfrica`: inside `Africa`, longitude `>= 28`, and latitude in `[-15, 18]`;
4. `RestOfAfrica`: any other tile inside `Africa`;
5. `Arabia`: inside `Arabia`;
6. `Levant`: longitude in `[25, 45]` and latitude in `[28, 42]`;
7. `Frangistan`: longitude `< 45` and latitude `>= 35`;
8. `Sahul`: inside `Sahul`;
9. `SoutheastAsia`: longitude in `[90, 140]` and latitude `< 30`;
10. `SouthAsia`: longitude in `[60, 96)` and latitude `< 32`;
11. `Siberia`: latitude `>= 50`;
12. `CentralAsia`: longitude in `[45, 96)` and latitude `>= 30`;
13. `EastAsia`: every remaining land tile.

Validation requires every land tile to match at least one rule and the first-match materialization
to assign exactly one region, every region to be non-empty, and every water tile to have none. Raw
rule predicates may overlap only because the published order resolves them. The exact land, water, region, highland, and river
catalogs above are the source of truth; step 3 freezes separate checksums for their rasterized
outputs rather than selecting geometry. A **coastal tile** is a land tile with at least one of its
eight coordinate neighbors either out of bounds or rasterized as water. This closed definition is
used by `ClassifyBiome`, rendering, exploration coastline reveal, and every coastline fixture.

The three passage endpoint pairs resolve from exact geographic anchors after rasterization. For each
anchor, select the nearest land tile on its named side of the corresponding water ring, breaking a
distance tie by tile ID: north Wallacea `(118,-3) → (138,-3)`, south Wallacea
`(120,-11) → (140,-14)`, and Bering Strait `(176,65) → (194,65)`. Step 4 freezes the six resulting
tile IDs; runtime never searches for a different crossing.

### Authored elevation

V1 uses a sparse, deterministic elevation layer rather than a sampled global digital-elevation
model. Each highland polygon carries one representative height in kilometres; this deliberately
creates broad gameplay plateaus at the map's coarse resolution. `HighlandElevationKm = 1.0 km` is the
strict Mountainous Highlands threshold.

| Stable order | Highland feature            | `ElevationKm` |
| -----------: | --------------------------- | ------------: |
|            0 | Atlas                       |        `1.50` |
|            1 | Ethiopian Highlands         |        `2.00` |
|            2 | Zagros                      |        `1.50` |
|            3 | Caucasus                    |        `2.00` |
|            4 | Himalaya / Tibetan Plateau |        `3.00` |
|            5 | Alps                        |        `2.00` |
|            6 | Urals                       |        `1.25` |
|            7 | Altai                       |        `2.00` |
|            8 | Central Range of New Guinea |        `2.50` |
|            9 | Alaska Range                |        `2.50` |

After land/water rasterization, elevation is evaluated at each tile center:

```text
ElevationKm(tile) = 0                                                     if tile is water
                    max(height of every covering highland polygon)       if any covers land tile
                    0                                                     otherwise
```

Highland polygons are clipped to land, and maximum rather than sum resolves overlap, so feature
iteration order cannot change a tile. A land tile is Mountainous Highlands exactly when
`ElevationKm(tile) > HighlandElevationKm`; a value at the threshold is non-highland. Non-highland
land and all water therefore sit at the model's `0 km` reference. There is no random perturbation,
interpolation, slope, depression, sea-level change, or separate render-only height field in v1.

The catalog, strict threshold, land clipping, and maximum-overlap rule belong to
`GeographyAlgorithm: "dispersal-map-v2"`. The resulting fixed tile elevations drive temperature,
biome classification, orographic moisture, altitude UV, hypoxia pressure, and the 3D mesh, but are
derived geography rather than mutable or serialized campaign state. The Initial values may be tuned
before release; afterward, changing any height, mask, threshold, or overlap rule changes downstream
simulation and requires a geography-version change and explicit migration.

Validation requires exactly ten unique entries in stable order; finite heights strictly above
`HighlandElevationKm` and no greater than the catalog maximum `3.0 km`; a finite non-negative strict
threshold; at least one contributing land tile from every feature; elevation exactly zero on water
and uncovered land; finite tile values in `[0, 3]`; maximum-overlap and feature-order invariance; seed
independence; and a checked-in exact `Float64bits` checksum of all 6,144 tile elevations. Boundary
fixtures classify synthetic elevations `1 ulp` below, exactly at, and `1 ulp` above the threshold.

**Starting-band anchors.** New-world placement uses eight authored geographic anchors rather than
choosing arbitrary cells after the raster exists. They are broad gameplay starting areas, not claims
that each band represents a population at one exact archaeological site:

| Stable order | Band profile                            | Geographic anchor                 | Population | Latitude | Longitude |
| -----------: | --------------------------------------- | --------------------------------- | ---------: | -------: | --------: |
|            0 | `HomoSapiens`, East Africa              | Afar                              |      `120` | `11.5°N` |  `41.0°E` |
|            1 | `HomoSapiens`, East Africa              | Lake Turkana                      |      `120` |  `3.5°N` |  `36.0°E` |
|            2 | `HomoSapiens`, East Africa              | Lake Victoria Rift                |      `120` |  `1.0°S` |  `33.0°E` |
|            3 | `HomoSapiens`, East Africa              | Southern East African Rift        |      `120` |  `7.0°S` |  `35.0°E` |
|            4 | `ArchaicHominin`, Levant                | Northern Levant                   |      `120` | `34.5°N` |  `36.0°E` |
|            5 | `ArchaicHominin`, Frangistan            | Balkans                           |       `60` | `43.0°N` |  `22.0°E` |
|            6 | `ArchaicHominin`, Frangistan            | Iberia                            |       `60` | `40.0°N` |   `4.0°W` |
|            7 | `ArchaicHominin`, Siberia               | Denisova Cave (Altai)             |       `12` | `51.40°N` |  `84.68°E` |
|            8 | `ArchaicHominin`, Southeast Asia        | Tam Pa Ling                       |       `12` | `20.20°N` | `103.40°E` |
|            9 | `ArchaicHominin`, East Asia             | Harbin (Denisovan)                |       `90` | `45.75°N` | `126.63°E` |

The three eastern anchors represent a Denisovan population within the shared `ArchaicHominin`
simulation type. They establish an eastern archaic presence without introducing separate Neanderthal
and Denisovan mechanics, technology graphs, or computer policies, and their shared
`HighAltitudeAdaptation` is what distinguishes the lineage from the western archaic populations.
Denisova Cave in the Altai is the type site and the eponym; Harbin carries the largest of the three,
matching the size of the ground available there; Tam Pa Ling stands for the southern range implied by
the Denisovan ancestry carried by present-day Southeast Asian and Papuan populations.

The Baishiya Karst Cave site on the Tibetan Plateau is deliberately **not** an anchor, despite being
the best-attested Denisovan occupation outside the Altai
([Xia et al. 2024](https://www.nature.com/articles/s41586-024-07612-9)). It resolves to a 3,000-metre
tile whose mean temperature of −3.3 °C gives a thermal suitability of 0.12, and because the
vegetation index is the product of moisture and thermal suitability, that tile yields
`BaselineK = 4.1` despite excellent moisture. A band of any playable size there begins in maximal
crowding collapse. The model has no capacity channel for high-altitude adaptation — the trait acts on
hypoxia risk — so the site cannot be represented as a viable starting population without inventing
one. This is the model being accurate about the Tibetan Plateau rather than a defect, and the sites
that make Denisovans archaeologically famous are marginal for exactly the reason they preserved.

After step 4 can classify turn-0 habitat, an authoring generator projects each anchor to fractional
grid coordinates and processes this stable order. Its candidates are unused tiles that are land,
belong to the anchor's required region, and have finite positive turn-0 `BaselineK` under the
seed-independent habitat-temperature curve. Positive `BaselineK` is a necessary condition and not a
sufficient one: a tile with `K = 4.1` satisfies it while supporting nobody, so the catalog
additionally requires that no anchor begin in maximal crowding collapse. The crowding term is pinned
at `MaxCrowdingDeclineFraction` once `P / K_eff` reaches `1 + MaxCrowdingDeclineFraction / r`, so an
anchor at or beyond that ratio would take the maximum loss every turn from turn one. Anchors may
begin stressed, and several deliberately do; they may not begin in free fall. Rank them by squared Euclidean distance from the fractional
projected anchor, then by ascending tile ID. Select the first and exclude it from later anchors. The
generator writes the resulting ten exact tile IDs into the checked-in scenario fixture; runtime
initialization reads those IDs and does not repeat the search. A generation test reruns the resolver
and requires an exact match, so geography or climate changes cannot silently move a starting band.
The habitat predicate makes resolution independent of `WorldSeed`; the resolver and placement
consume no `WorldRNG`.

**Complete new-world initializer.** `domain.NewWorld(seed)` is a total constructor, not a collection
of defaults spread across callers. It resolves the checked-in scenario tile IDs in the stable order
above and creates band IDs `1` through `8`; `NextBandID` is `9`. Every band starts with its exact
catalog population above, `Health = 1`, `StoredFood = 0`, unavailable all-zero `LastFoodReport` and
`LastOutcomeReport` values, an all-zero last-mortality breakdown, no acquired technologies, nine zero progress values, no research target, no
queued spatial or interbreeding intent, and `SpatialActionUsed = false`. The four sapiens bands use
the exact initial assignment vector `3500 / 3000 / 1500 / 500 / 1500` in assignment-enum order;
each archaic band uses `ArchaicAssignmentPreset[resolved region, turn-0 biome]`. Every band receives
the standing-variation vector selected below for its species and start region.

The constructor sets `Turn = 0`, terminal result `Ongoing`, empty established-region and event
collections, and the exploration mask exactly as specified below. It derives turn-0 geography,
habitat temperature, biome, caps, and natural shelter; initializes every tile with
`Degradation = 0` and the resource-stock rule in §7; and constructs `WorldRNG` by expanding `seed`
through §7's two `SplitMix64` applications into PCG's two 64-bit words. Construction sets application
state only through the New Game use case, which sets `WorldRevision = 1`; the application-derived
`TerrainRevision` is also `1` for the first accepted
frame. Initialization performs no turn, health, selection, mutation, event, achievement, autosave,
or policy pass and consumes no `WorldRNG` draw. Validation and one golden new-world fixture assert
every field above for multiple seeds, including exact band order/IDs, exact PCG bytes, stocks within
their turn-0 caps, no hidden defaults, and `ExportState`/`RestoreWorld` equality.

Chosen over a hand-typed 96×64 ASCII block because it is verifiable by test, self-documenting
(`{Lat: 12.5, Lon: 43.3} // Bab-el-Mandeb`), and survives a change of grid size.

**Region naming.** `region.go` holds one closed `Region` enum and a single `Region -> display name`
table for East Africa, the rest of Africa, Arabia, the Levant, Frangistan, Central Asia, South Asia,
Southeast Asia, East Asia, the Yellow River Basin, Sahul, Siberia, and Beringia. The basin is an
explicit subregion replacing `EastAsia` on its rasterized tiles; region masks never overlap. The
landmass north of the Mediterranean is **Frangistan**. Every user-visible region string reads from
this table, so naming changes in one place, never at call sites. Region masks are deterministic
geography, not biome labels, and therefore do not move with climate.

Fauna uses these same non-overlapping regions together with the current biome (§7). Regional
fauna profiles may differ even where biomes match; neighboring rows may also deliberately share
the same profile. No second overlapping set of fauna-region boundaries is introduced.

The rasterized map deliberately preserves water in Wallacea and at the Bering Strait. Neither gap is
ordinary tile adjacency; the named, bounded passage edges in §7 provide their only crossings. The
Sahul destination lies beyond Wallacea, and the Beringia destination mask lies on the Alaska side of
the strait, so each achievement proves that its passage was actually crossed.

### Persistent exploration veil

The full geography exists and simulates from turn 0, but the player map begins with only East
Africa explored. `World` stores a fixed-size `ExploredTiles [ExplorationWordCount]uint64` bitset,
where `ExplorationWordCount = (96 * 64 + 63) / 64 = 96`; bit `i` corresponds exactly to tile ID `i`.
This is bounded campaign knowledge, not temporary sight range. Bits only change from zero to one and
are never cleared by migration, death, climate change, load, or the absence of a nearby band.

New-world setup marks every East Africa land tile and its immediately adjacent water tiles explored,
so the starting region and its coastline are legible. It does not reveal land in another region
merely because that land touches the East Africa region boundary, and the archaic starting bands in
the Levant and Frangistan reveal nothing. After placing the starting sapiens bands, setup also runs
the ordinary frontier rule below; scenario validation requires that this add no non-East-African
land at turn 0.

`RevealFromSapiens` unions the following tiles into the bitset without consuming RNG:

1. for each surviving sapiens band, its current tile and every in-bounds tile in the surrounding
   3×3 coordinate footprint, including water and diagonals;
2. for each named passage incident on that band's tile, the far endpoint when that passage is
   currently traversable by the band under its technology, long-term-climate, and destination-
   habitability requirements.

The 3×3 rule is observation, not movement topology: it may show water or a diagonal that the
no-water-corner rule does not permit the band to enter. It nevertheless guarantees that every
ordinary one-edge destination exposed in a sapiens band's migration candidates is already explored.
The eligible-passage rule gives the same guarantee for named-passage candidates without revealing
Sahul or western Alaska while the relevant crossing remains unavailable. A locked passage may show
only its local endpoint glyph under the rendering rule below; that glyph does not set the far tile's
bit.

Reveal runs after new-world placement, immediately after an accepted sapiens split and before its
replacement frame, and in phase 5 after movement, macro/acute hazards, and zero-population cleanup but
before terminal/achievement publication. A band that does not survive the completed turn therefore
does not reveal a new frontier from its terminal position. Whole-band migration reveals nothing when
queued and reveals from the destination only after the move actually resolves. Split destinations
are already in the source band's explored 3×3 footprint; the post-split pass adds the new band's
next frontier. Union order is immaterial, but implementation iterates bands by ascending ID and
passages by stable passage ID for reproducible diagnostics.

Only `HomoSapiens` contributes to exploration. Archaic bands continue to move, compete, research,
and experience hazards everywhere under the same simulation, but do not uncover terrain for the
player. Once an explored tile contains an archaic band, that marker and the tile's current public
stats remain visible even after sapiens leave: v1 has persistent geographic knowledge, not a second
temporary line-of-sight or espionage system. Exploration never changes resource regeneration,
climate, biome classification, migration scoring or legality, archaic policy, hazards, genetics,
technology, achievements, or victory. `SapiensEstablishedRegions` remains a separate thresholded
historical achievement; seeing part of a region is not establishing it.

### Fixed natural shelter

Each tile has one finite `NaturalShelter` rating in `[0, 1]`, generated from the closed geographic
layer above. Zero is the ordinary shelter-work baseline; a higher rating means more favorable
terrain for shelter work, not a percentage of deaths prevented. Unmarked land and all water tiles
have rating zero. Where authored shelter regions overlap, the greatest rating wins, so feature
iteration order cannot change the map. The release uses sparse authored masks rather than a
region-wide default or a continuous geology raster. They are stylized gameplay geography rather
than a cave census.

`NaturalShelterMaskAlgorithm: "authored-ellipse-v1"` evaluates the following closed catalog in grid
coordinates after land rasterization. A tile center `(x, y)` is inside an entry when
`((x-cx)/rx)^2 + ((y-cy)/ry)^2 <= 1`; radii are strictly positive. The table is deliberately
coarse and may be retuned as Initial configuration, but it is complete for v1—implementation must
not infer missing masks from elevation, region, or biome.

| Shelter feature                          | Center `(cx, cy)` | Radii `(rx, ry)` | Rating |
| ---------------------------------------- | ----------------: | ---------------: | -----: |
| East African Rift                        |        `(24, 36)` |        `(3, 10)` | `0.50` |
| Southern African escarpment              |        `(21, 50)` |         `(6, 5)` | `0.50` |
| Atlas–Levant–Anatolia karst belt         |        `(27, 18)` |        `(10, 4)` | `0.50` |
| Zagros and Central Asian uplands         |        `(39, 20)` |        `(10, 5)` | `0.50` |
| South and Southeast Asian limestone belt |        `(54, 30)` |        `(12, 6)` | `1.00` |
| Yellow River and East Asian uplands      |        `(59, 19)` |         `(7, 5)` | `0.25` |
| Altai and southern Siberian cave belt    |        `(62, 10)` |        `(10, 4)` | `0.50` |
| Sahul escarpments                        |        `(73, 44)` |         `(6, 5)` | `0.50` |
| Beringian and western Alaskan uplands    |         `(91, 5)` |         `(4, 4)` | `0.25` |

Rasterization clips every mask to in-bounds land and then quantizes the selected maximum to the
only permitted values `0`, `0.25`, `0.50`, or `1.00`. Validation requires the exact nine unique
feature IDs and table rows above, finite inputs, permitted ratings, at least one land tile in each
mask, and at least one land tile at every nonzero tier. A checked fixture hashes the resulting
`6_144` ratings and asserts that water remains zero, making an accidental polygon or projection
change visible without storing per-tile cave state in the save.

The rating is immutable geography, independent of world seed, current biome, climate, degradation,
and resident population. It is not inferred merely from elevation or a biome label: a biome change
must not create or remove caves. World generation and load reconstruction use projection/land data
under `GeographyAlgorithm: "dispersal-map-v2"` and shelter catalog/raster rules under
`NaturalShelterMaskAlgorithm`, consume no `WorldRNG`, and validate finite in-range ratings. After
release, changing the table, mask equation, or rating changes the natural-shelter-mask identifier;
changing projection or land changes both identifiers because it changes the rasterized mask. Either
requires an explicit save migration. A world and its
frame each hold at most `96 * 64 = 6_144` scalar ratings; no cave IDs, ownership, occupancy,
depletion, or construction state exists.

`NaturalShelter` has exactly two gameplay consumers: it modifies the exposure-protection part of
shelter-work efficiency under §7, not camp security or hygiene, and `MacroEventAlgorithm` reads it
for the Campanian-refugium rule. It is not another resource stock, carrying-capacity term,
migration-score input, or generic base-hazard subtraction. The tile inspector exposes it; existing movement ranking and the
archaic planning policy otherwise remain unchanged.

### Base moisture is rasterized, not authored

`BaseMoisture` is the **fixed geographic** component of a tile's water availability — the pattern set
by atmospheric circulation, rivers, continentality, and relief. It is not a paleoclimate
reconstruction. Everything time-varying lives in §7's `TileMoistureOffset`, which carries the
aridification trend, the precession oscillation, and the regional aridity weights. The large-scale
controls behind `BaseMoisture` are effectively unchanged across the campaign's 60,000 years at this
resolution, so one fixed field plus a time-varying offset is the correct decomposition.

```text
BaseMoisture(tile) = clamp01( ZonalMoisture[Row(tile)]
                            + RiverProximityBonus(tile)
                            - ContinentalityPenalty(tile)
                            + OrographicBonus(tile) )

RiverProximityBonus(tile) = RiverCorridorBonus  if a river polyline crosses the tile
                            RiverAdjacentBonus  if the tile is orthogonally adjacent to a corridor
                            0                   otherwise
```

`geodata.go` supplies every geographic input to this formula; no additional sourced layer is
required.

**Zonal component.** `ZonalMoisture` is a 64-entry table, one value per map row, checked in as exact
`float64` bit patterns under a SHA-256 checksum and used verbatim at run time, exactly as
`TemperatureAlgorithm`'s `LatitudeSinSquared` table is. The two share a row key — map row `y` — but
not an owning contract, so they stay separate tables: `LatitudeSinSquared` belongs to
`TemperatureAlgorithm` and `ZonalMoisture` to `GeographyAlgorithm`, and §9's versioning split exists
precisely so that changing temperature coefficients need not re-checksum unrelated geographic data.
The table is generated from a profile symmetric in absolute latitude, piecewise-linear between these
anchors:

| Absolute latitude | `ZonalMoisture` | Circulation feature     |
| ----------------: | --------------: | ----------------------- |
|              `0°` |          `0.95` | ITCZ                    |
|              `8°` |          `0.80` |                         |
|             `15°` |          `0.45` | Hadley descending limb  |
|             `25°` |          `0.12` | subtropical high        |
|             `35°` |          `0.35` |                         |
|             `50°` |          `0.60` | mid-latitude westerlies |
|             `65°` |          `0.35` |                         |
|             `72°` |          `0.25` | polar drying            |

This is what makes the deserts **emergent rather than painted**, the same argument the biome
subsection below makes for biome labels. Nothing authors a Sahara: the profile's two subtropical minima land on map rows 24–25 and 51–52,
which under this section's projection carry the Saharan/Arabian and Kalahari latitudes, and the same
profile puts its maximum on row 38 at the Congo. Map rows do not fall exactly on the `25°` anchor,
so the sampled values there are near the minimum rather than equal to it; the assertion in §12 is
therefore stated over latitude bands, not over a single row.

**River corridors.** `geodata.go`'s river polylines rasterize as high-moisture corridors under the
piecewise mapping above. A tile the polyline crosses receives `RiverCorridorBonus =
0.35` and an orthogonally adjacent tile `RiverAdjacentBonus = 0.20`; corridor membership takes
precedence, so no tile receives both. River
corridor tiles also count as water for `WaterDistance` below, so a river suppresses continentality
along its length rather than fighting it. The Nile is the case that matters: against a central
Saharan zonal value of `0.12`, an on-river tile reaches `0.47` and classifies as Savanna while the
surrounding desert stays hyperarid, with the immediate fringe near the scrub threshold. A habitable
green ribbon through otherwise lethal terrain falls out of two constants rather than an authored
mask.

**Continentality.** `ContinentalityPenalty(tile) = ContinentalityMax * min(1, WaterDistance(tile) /
ContinentalityRange)` with `ContinentalityMax = 0.20` and `ContinentalityRange = 8` tiles, where
`WaterDistance` is Chebyshev distance to the nearest water or river-corridor tile. This dries deep
interiors that latitude alone would leave temperate — Central Asia near `45°N` starts from a zonal
`0.5167` and loses the full `0.20`, leaving `BaseMoisture ≈ 0.3167` before any river or relief bonus.
For the synthetic unrivered, non-highland `45°N` turn-0 fixture, Central Asia's `0.75` regional
moisture weight and the selected precession offset produce `V ≈ 0.38` under turn-0
`HabitatTemperatureC`; seed-dependent climate noise does not enter `V`, so
the classifier returns the modeled **Semi-Arid Desert** biome rather than an
unmodeled “steppe” or “woodland” label. Continentality also deepens the Saharan and Arabian
interiors below their already-low zonal floor.

**Relief.** Tiles with `ElevationKm > HighlandElevationKm` receive `OrographicBonus = 0.10` for
orographic lift. The bonus is one fixed highland predicate, not proportional to elevation; height
already affects temperature, UV, hypoxia, classification, and rendering. Rain shadow is deliberately
excluded: expressing it needs a prevailing-wind field, a second geographic layer added for one
effect.

`BaseMoisture` is fixed geography. The new-game seed does not perturb it, any more than it perturbs
landmasses. Validation requires exactly 64 finite `ZonalMoisture` entries in `[0, 1]`, a profile
symmetric in absolute latitude within the table's stated tolerance, finite non-negative bonus
constants, `ContinentalityRange` a positive tile count, and a finite `BaseMoisture` in `[0, 1]` for
all 6,144 tiles. Because `ClassifyBiome` reads it, changing the table, its anchors, or any bonus
constant reclassifies every tile in an existing save and therefore requires a `GeographyAlgorithm`
version bump with explicit migration — the same obligation §9 places on the temperature
coefficients. Derived `BaseMoisture` is never serialized; the grid regenerates from `geodata.go` and
these tables at world creation.

### Biome is derived, never stored

The rasterizer produces **elevation**, **base moisture**, and **natural shelter** per tile.
`ClassifyBiome` computes the biome each turn from `(latitude, elevation, EffectiveMoisture,
HabitatTemperatureC)`; natural shelter remains a separate fixed attribute. `HabitatTemperatureC`
uses every temperature component except the seed-dependent `ClimateNoise`; `LocalTemperatureC`
retains that small variation for water demand and hazards. Base moisture is fixed
geography, but §7's long-term moisture offset shifts it every turn, so aridity varies in time as
well as in space.

Biomes are therefore dynamic rather than authored. Both offsets feed the DESIGN-selected vegetation
index `V`, which §7 defines and which drives biome, `BaselineK`, and movement cost together. As `TileTempOffset`
falls the tundra threshold moves south, and as `TileMoistureOffset` falls savanna gives way to
semi-arid desert — the Last
Glacial Maximum becomes an emergent consequence of two derived scalars rather than a scripted event.
**A map with painted biomes could not do this**, because the tundra line and the desert margin would
both be frozen in the data. It is also what makes the 3D view worth having: the ice sheet advances
from the north while desert spreads from the south, and the precession term lets habitat visibly
reopen during a wetter interval before closing again.

The new-game `seed` perturbs initial **resource abundance** (flora/fauna/water stocks) and the small
bounded local hazard/demand temperature-noise component. Landmasses, `BaseMoisture` and its zonal table, biome history, the directed LGM
trend, the moisture trend with its precession term and regional aridity weights, the authored abrupt-climate and
eruption catalogs, and the seasonal phase remain fixed, so every seed follows the same historical
arc without producing an identical turn-by-turn climate trace.

---

## 7. Simulation

Every release configuration value this section names is registered in **Appendix C** with its current
value or source, status, and owning contract. The rules below remain authoritative for semantics and
may repeat values where the rule, a worked example, a bound derivation, or a fixture needs to be
followable. A value change starts in Appendix C and must then update every dependent rule, example,
derived bound, version identifier, fixture, and migration named by the checklist there.

### Species and inter-species competition

`Species` has two simulated values in this slice: `HomoSapiens` and `ArchaicHominin`, but only
`HomoSapiens` is player-controlled. New-world scenario data deterministically places sapiens bands
as four bands of `120` people on four distinct East African land tiles, for `480` total sapiens.
It places one archaic band of `120` in the Levant, two of `60` on distinct Frangistan tiles, and three
Denisovan-representative bands — `12` at Denisova Cave in Siberia, `12` at Tam Pa Ling in Southeast
Asia, and `90` at Harbin in East Asia —
for `354` total archaics and `834` people overall. Step 4 resolves §6's ten geographic anchors and freezes
the exact generated tile IDs after validating their region, land, and turn-0 habitability; placement
and population consume no RNG.
Every new-game band of either
species starts at turn 0 with `Health = 1.0` (100% condition), an empty acquired-technology bitset,
exactly nine zero research-progress values, and no research target. This technology initialization
is identical across species and consumes no RNG. Each band also receives its complete deterministic
species-and-start-region heritable-state vector under the genetics contract below. The player may
select and inspect either species, but may assign, research for, split, migrate, or initiate
interbreeding only through a sapiens actor. A versioned,
deterministic domain policy makes those decisions for archaic bands when the player ends the turn.
HUD labels and map markers distinguish the species and label archaic selections “Computer
controlled.” Direct combat, diplomacy, player control of archaics, and species-specific technology
graphs (DAGs) remain out of scope.

Competition is ecological rather than military. All bands on a tile draw from the same
flora/fauna/water stocks and contribute to its total population pressure. To prevent map or band-ID
iteration order from deciding who eats, the demographic phase first computes every band's
workforce-derived flora and fauna demands plus its population-wide water demand, then allocates each
resource proportionally across the tile. For each resource, allocations are non-negative, their sum
never exceeds the available stock, and permuting band IDs cannot change the per-species totals.
Total population across both species is used for tile degradation and for the
destination-population term in migration scoring.

### Biome intent

Six land biomes are normative. Their qualitative grades are design intent rather than measurements.
The four vegetation-classified biomes are ordered by the maximum capacity within their respective
`V` bands. The two geography-classified biomes remain responsive to `V`, so their capacity and
movement grades are reference values evaluated at savanna-grade `V = 0.60`, not global bounds.

| Biome                 | Base capacity `K` | Primary subsistence resources | Risk factor           | Movement cost |
| --------------------- | ----------------- | ----------------------------- | --------------------- | ------------- |
| Riverine Woodland     | Very high         | Fish, fruits, fresh water     | Waterborne disease    | Low           |
| Savanna               | High              | Large ungulates and tubers    | Seasonal drought      | Low           |
| Coastal Shrubland     | High              | Shellfish and marine life     | Flash floods, storms  | Moderate      |
| Mountainous Highlands | Moderate          | Goats and hardy roots         | Cold and fall hazards | Very high     |
| Semi-Arid Desert      | Low               | Seeds, small game, roots      | Extreme dehydration   | High          |
| Glacial Tundra        | Very low          | Megafauna (mammoths)          | Severe exposure       | High          |

The subsistence-resources column informs §7's flora/fauna profiles and Appendix B.2's flora, fauna,
and water targets; it does not make water food or permit water to enter `StoredFood`. The risk column
informs the seasonal and chronic hazard rates. Nothing reads these words at runtime — the validation
rules below define exactly where their qualitative grades constrain numeric configuration.

### Vegetation index and biome classification

The intent table fixes the six biome labels and their qualitative grades, but not the numeric
thresholds between them. This build introduces a per-tile
**vegetation/aridity index** `V` in `[0, 1]`, derived from temperature and moisture availability, as a
gameplay model. `V` supplies the threshold scale `ClassifyBiome` needs, and it is why
biome, `BaselineK`, and movement cost all derive from a single scalar rather than three unrelated
tables. Its formula, thresholds, capacity knots, and movement knots are Initial values owned by this
design and remain subject to §12's balance pass; none is a measurement.

```text
ThermalSuitability(T) = clamp01((T - VegetationColdCutoffC)
                                / (VegetationWarmthC - VegetationColdCutoffC))
V(tile, t) = clamp01(EffectiveMoisture(tile, t) * ThermalSuitability(HabitatTemperatureC(tile, t)))
```

The product is deliberate: vegetation needs both water and warmth, so either at zero drives `V` to
zero. A cold wet tile and a hot dry tile both score low; the separate temperature split below then
distinguishes Glacial Tundra from Semi-Arid Desert. Selected coefficients are
`VegetationColdCutoffC = -5.0` and `VegetationWarmthC = 10.0`.

**Six biomes from a mixed classifier.** Two of the six labels are geographic rather than
purely vegetative and are classified before `V` is consulted: **Mountainous Highlands** by elevation
and **Coastal Shrubland** by coastline. Remaining tiles classify by DESIGN-selected `V` bands, with
the lowest band split by temperature so a glacier and a hyperarid desert are not the same biome — a
distinction a campaign spanning Beringia and Arabia needs:

| Condition                                             | Biome                 |
| ----------------------------------------------------- | --------------------- |
| `ElevationKm > HighlandElevationKm`                   | Mountainous Highlands |
| coastal tile below that elevation                     | Coastal Shrubland     |
| `V < 0.30` and `HabitatTemperatureC < TundraSplitTempC` | Glacial Tundra        |
| `V < 0.30` otherwise                                  | Semi-Arid Desert      |
| `0.30 <= V < 0.45`                                    | Semi-Arid Desert      |
| `0.45 <= V < 0.75`                                    | Savanna               |
| `0.75 <= V`                                           | Riverine Woodland     |

`TundraSplitTempC = 0.0`. Semi-Arid Desert and Glacial Tundra each span a range of low `V`, which is
why baseline capacity and movement cost vary within a biome rather than using one constant per
label: a single value can express neither the hyperarid nor the scrub end of Semi-Arid Desert. The
discrete biome label remains the lookup key for regional fauna profiles, the endemic-disease table,
and Appendix B.2's resource targets.

`ClassifyBiome` produces an **instantaneous candidate**. The published biome table is derived once
for all 401 turns in order so seasonal threshold crossings cannot make resource caps and movement
grades flicker. Turn 0 publishes its candidate. On every later turn, a candidate different from the
published biome must remain identical for `MinBiomeDwellTurns` consecutive turns before it replaces
the published value; returning to the published biome resets the candidate run. The table is
seed-independent, derived rather than saved, and `BuildHabitat` indexes it for the requested turn.
This persistence rule does not smooth `V`, either temperature value, moisture, or the continuous
capacity and movement curves; it stabilizes only the discrete lookup class.

Validation requires `VegetationColdCutoffC < VegetationWarmthC`, finite coefficients, a finite `V`
within `[0, 1]` for all 6,144 tiles at all 401 supported turns, and the three `V` thresholds strictly
ordered within `(0, 1)`. The existing calibration assertions are restated against `V`: turn 0 must
classify East Africa as habitable savanna and woodland, and turn 400 must place the Glacial Tundra
boundary strictly south of its turn-0 latitude.

Validation checks that the numeric curves keep the four vegetation-classified biomes' capacity
ordering. Across those bands, maximum `BaselineK` must be non-decreasing from Glacial Tundra through
Semi-Arid Desert and Savanna to Riverine Woodland — `34`, `77`, `137`, `171` at the Initial
knots. At the `V = 0.60` reference condition, the two geographic biomes must land on their grades: Coastal
Shrubland with Savanna at High (`138` against `120`) and Mountainous Highlands at Moderate, strictly
between Semi-Arid Desert's band maximum and Savanna (`77 < 78 < 120`). That last margin is one
person wide; §12's balance pass must re-check it after any change to the `0.45` step or the
Mountainous Highlands factor. The Riverine Woodland maximum falls at `CanopyClosureV`, not at
`V = 1.00`.

Movement validation uses the same reference scope. At equal `V`, the factors order the `1.00`
vegetative baseline below Coastal Shrubland below Mountainous Highlands — `1.00 < 1.25 < 2.50` — and
the maximum composed vegetative cost falls strictly below Mountainous Highlands' minimum,
`2.20 < 2.50`, so the Very high mountain reference outranks the High desert/tundra maximum.

These geographic grades are deliberately not global bounds. A coastal tile with `V` far below
`0.60` can reach the movement ceiling despite its Moderate reference grade, and a coastal tile near
`CanopyClosureV` can exceed the Riverine Woodland capacity maximum after its `1.15` factor. Retaining
that behavior lets geographic biomes respond to local climate; the document therefore uses their
grades only at the stated reference condition and never asserts their ordering across all `V`.

### Reversible ecological degradation

`BaselineK` is derived each turn from the tile's current `V` and climate-derived biome before human
pressure. A tile persists only a dimensionless `Degradation` value; it never mutates or saves an
absolute base capacity. For total resident population `P_total`:

```
pressure = P_total / BaselineK

if pressure > 1:
    Degradation_next = min(MaxDegradation,
                           Degradation + damage_rate · (pressure − 1)²)
else:
    Degradation_next = max(0,
                           Degradation − recovery_rate · (1 − pressure))

EcologicalK = BaselineK · (1 − Degradation_next) · MacroHabitatFactor(tile, turn)
K_eff       = EcologicalK · T_tech
```

The approved initial rates are **`damage_rate = 0.06`** and **`recovery_rate = 0.03`** per game
turn. At pressure `1.5`, damage rises by `0.06 * 0.5² = 0.015`; at pressure `2.0`, it rises by
`0.06`. An empty tile removes `0.03` degradation per turn, so recovery from the locked maximum
`0.75` takes 25 uninterrupted turns. These are dimensionless per-turn gameplay rates and do not
scale with the campaign era's calendar years.

`BaselineK` has exactly two inputs: the DESIGN-selected vegetative curve and the current biome's
deviation factor. Fixed geography and climate determine those inputs, but region does not apply an
additional multiplier:

```text
BaselineK(tile, t) = BaselineKCurve(V(tile, t))
                     * BiomeCapacityFactor[Biome(tile, t)]
```

`BaselineKCurve` is piecewise-linear within its bands and right-continuous at its one selected step.
Its Initial knots translate the intent table's Very low through Very high capacity grades
onto this build's tile scale, with `120` people as the selected savanna-grade reference:

|                          `V` | `BaselineKCurve(V)` people |
| ---------------------------: | -------------------------: |
|                       `0.00` |                        `0` |
|                       `0.15` |                        `9` |
|                       `0.30` |                       `34` |
| `0.45` approached from below |                       `77` |
|                       `0.45` |                      `103` |
|                       `0.75` |                      `137` |
|                       `0.90` |                      `171` |
|                       `1.00` |                      `145` |

The step at `0.45` is a DESIGN-selected Initial value expressing grassland as qualitatively more
supportive than scrub. It is a balance choice and may be retuned or smoothed during the
balance pass. `BaselineKCurve` is right-continuous there.

The curve peaks at `CanopyClosureV = 0.90` and falls to `145` at `V = 1.00`. Riverine Woodland is the
highest vegetation-classified capacity grade, so the wettest, densest non-geographic tiles must not
become the most habitable terrain in that sequence: a closed canopy shades out the understorey the
woodland mosaic feeds foragers from, and what yield remains is harder to reach. The decline keeps the
vegetative peak attached to the woodland optimum instead of the wet extreme. `CanopyClosureV` is the
same threshold that turns `MovementCurve` upward, so one constant carries both consequences of a
canopy closing.

Each biome then applies fixed factors from the **per-biome deviation table**. Both factors answer the
same question — how far this biome departs from its purely vegetative baseline — so they share one
table rather than sitting in two:

| Biome                 | `BiomeCapacityFactor` | `BiomeMovementFactor` |
| --------------------- | --------------------: | --------------------: |
| Savanna               |                `1.00` |                `1.00` |
| Riverine Woodland     |                `1.00` |                `1.00` |
| Semi-Arid Desert      |                `1.00` |                `1.00` |
| Glacial Tundra        |                `1.00` |                `1.00` |
| Coastal Shrubland     |                `1.15` |                `1.25` |
| Mountainous Highlands |                `0.65` |                `2.50` |

Only the two geographic biomes deviate: the four vegetative biomes _are_ the `V` bands, so their
factors are `1.00` by construction. At the `V = 0.60` reference, the capacity formula yields `120`
for the vegetative baseline, `138` for Coastal Shrubland, and `78` for Mountainous Highlands. Those
values instantiate the reference grades and the ordering validation above. The Mountainous
Highlands movement factor is derived: at `2.50` its cheapest composed cost sits strictly above the
`2.20` vegetative maximum, which makes its Very high reference outrank the High grades of Semi-Arid
Desert and Glacial Tundra.

Configuration validation enforces that construction rather than trusting it. Both columns must have
exactly `BiomeCount` finite positive entries, **and the four vegetation-classified biomes must be
exactly `1.00` in both columns**; only the Coastal Shrubland and Mountainous Highlands entries are
tunable. Without that check a balance pass could set Savanna's capacity factor to `1.10`, pass every
other gate, and silently break the `V = 0.60 → 120` reference, the one-person `77 < 78 < 120` grade
margin, and the identity between a vegetative biome and its own `V` band.

The function applies identically to both species. Region, season, current stock, and resident species
do not select another baseline; technology and degradation enter later through the displayed formula.
A single constant per biome could express neither the hyperarid end of the scale nor within-biome
variation, and would make a tile's capacity jump at every climate reclassification. Under
`BaselineKCurve(V)`, capacity changes smoothly within each band; only the selected `0.45` step and a
change to a geographic biome factor remain discontinuous. The dwell and churn gates in §7 cover
those remaining transitions.

World/config validation requires every habitable tile to have finite positive `BaselineK`, finite
rates in `[0, 1]`, and `MaxDegradation = 0.75`, so ecological capacity never falls below 25% of the
climate-derived baseline from degradation alone. An active macro episode may temporarily reduce it
further through a strictly positive factor. An uninhabitable tile has `BaselineK == 0`, cannot contain a living band,
keeps `Degradation == 0`, and bypasses the pressure calculation. New damage is measured against
`BaselineK`, not the already-reduced `EcologicalK`; this prevents degradation from amplifying its
own pressure ratio into a runaway loop. Each flora/fauna/water cap is multiplied by the same
`(1 − Degradation)` factor and its current `MacroResourceFactor(s, tile, turn)`. Below-baseline pressure therefore restores both carrying capacity and
resource potential gradually; no separate permanent land-health or overuse-streak state exists in
this build.

The frame exposes `BaselineK`, `EcologicalK`, `Degradation`, current macro factors, and each resource's current stock and
cap so the tile inspector can explain damage, episode pressure, and recovery. Tests assert monotonic damage above
baseline, monotonic recovery below it, the `[0, 0.75]` bound under extreme populations and long
runs, recovery after evacuation, and identical behavior when multiple species contribute the same
total population.

### Resource regeneration and extraction

**Flora means plant food people can forage, not all vegetation.** Its stock and capacity describe
that forageable food resource, not total plant biomass or the fodder available to herbivores.
A tile can therefore have little flora for people to collect while still supporting abundant fauna.
Both stocks use the selected normalized harvest-unit scale and `1.0 FU` baseline conversion. The
initial capacities, collection rates, and source-specific modifiers below complete that balance
contract; they remain tunable **Initial** values unless Appendix C marks them **Locked**.

Appendix B contains the selected initial biome/season capacity indices and complete regional fauna
configuration. It is part of the versioned balance contract rather than a set of non-binding
suggestions. The normalized reference capacities are:

| Resource stock |           Reference capacity |
| -------------- | ---------------------------: |
| Flora          | `300` normalized flora units |
| Fauna          | `300` normalized fauna units |
| Water          |                     `250` WU |

For each resource `s`, Appendix B supplies a complete six-biome by four-season
`ResourceCapIndex_s` table. Calculate the undegraded and current caps exactly once:

```text
UndegradedResourceCap_s = ReferenceResourceCap_s
                          * ResourceCapIndex_s[current biome, current season]
ResourceCap_s = 0                                           if BaselineK == 0
                UndegradedResourceCap_s
                * (1 - Degradation)
                * MacroResourceFactor(s, tile, turn)        otherwise
```

There is no region, species, technology, or workforce multiplier on capacity. A reference index of
`1.0` therefore means 300 flora/fauna units or 250 WU before degradation and a macro episode.

**Habitat-collapse transition.** A climate update can make a previously habitable land tile reach
`BaselineK == 0`. Phase 2 resolves that state before degradation, regeneration, extraction, health,
or logistic growth: set all three resource caps and stocks on the tile to zero, reset its
`Degradation` to zero, and reduce every resident band's population to zero. Append one typed
`BandExtinct{Cause: HabitatCollapse}` event per affected band in ascending band-ID order, discard its
queued spatial/interbreeding intent and turn-local work, and exclude it from every later phase.
There is no automatic evacuation or destination search; current habitat/candidate previews are the
player's means of avoiding a marginal route. Terminal-state evaluation still occurs at
the ordinary phase-5 checkpoint, so several simultaneous collapses cannot publish competing results.
This explicit branch is the only time a band may transiently occupy a zero-capacity land tile. It
prevents division by zero in degradation, stress, and logistic growth and ensures that every living
band entering phase 3 has finite positive `BaselineK` and `K_eff`.

Flora, fauna, and water are separate resource stocks with stock-specific regeneration rates. For
each stock `s`, `ResourceCap_s` is derived from the current biome and season, then multiplied by the
reversible `(1 − Degradation)` factor and current `MacroResourceFactor(s, tile, turn)` above. With
`0 < regen_rate_s <= 1`:

```
capped_s  = min(stock_s, ResourceCap_s)
regrown_s = clamp(capped_s + regen_rate_s · (ResourceCap_s − capped_s),
                  0, ResourceCap_s)
next_s    = clamp(regrown_s − allocated_extraction_s, 0, ResourceCap_s)
```

The approved initial gap-recovery fractions are **flora `0.30`, fauna `0.15`, and water `0.50`**.
They apply in every biome and season after the current cap has been derived. Thus a zero-stock
resource recovers 30%, 15%, or 50% of that turn's current capacity before extraction. These are
per-game-turn balance values, not annual biological replenishment rates.

A falling biome/degradation/macro cap is a hard environmental maximum, so `capped_s` discards any amount
above the new cap before growth; a rising cap creates new recovery headroom. A depleted stock with a
positive cap recovers to `regen_rate_s · ResourceCap_s` on the next turn. An uninhabitable tile or a
resource unavailable in the current biome has a zero cap and remains at zero.

Regeneration happens once, before current-turn extraction. The proportional allocator in the next
phase guarantees `0 <= allocated_extraction_s <= regrown_s`, so `next_s` stays in
`[0, ResourceCap_s]`; the explicit clamps normalize floating-point edge error, not excess demand.
`Degradation` is not subtracted again from `next_s`; its entire resource effect is already
represented by the reduced cap. Config validation rejects non-finite caps/rates and rates outside
`(0, 1]`.

Tests use one-step fixtures to lock this ordering and assert zero-stock recovery, exact fractional
gap closure, monotonic convergence with no consumers, immediate clamping after a cap decrease,
bounded stock under extreme extraction, and conservation between regrown stock, all per-band
allocations, and `next_s`.

### Initial resource abundance

New-game resource stocks begin at an authored biome/resource fraction of their fully derived
turn-0 cap, then receive bounded two-scale seed variation. The approved initial base fractions are:

| Turn-0 current biome  |  Flora |  Fauna |  Water |
| --------------------- | -----: | -----: | -----: |
| Savanna               | `0.80` | `0.80` | `0.75` |
| Riverine Woodland     | `0.90` | `0.85` | `0.95` |
| Coastal Shrubland     | `0.80` | `0.85` | `0.90` |
| Semi-Arid Desert      | `0.60` | `0.70` | `0.55` |
| Mountainous Highlands | `0.70` | `0.75` | `0.80` |
| Glacial Tundra        | `0.50` | `0.85` | `0.75` |

For each habitable tile and stock, derive a broad region offset and a smaller local offset from the
world seed without consuming `WorldRNG`:

```text
region_offset = 0.08 * UnitResourceAbundanceV1(seed, Region, Region(tile), stock)
tile_offset   = 0.04 * UnitResourceAbundanceV1(seed, Tile, tileID, stock)
initial_fraction = clamp(InitialStockFraction[current biome, stock]
                         + region_offset + tile_offset, 0, 1)
InitialStock = Turn0ResourceCap(tile, stock) * initial_fraction
```

`UnitResourceAbundanceV1` is a pure, versioned, domain-separated counter hash in `rng.go` mapping
its tuple to a finite value in `[-1, 1]`. `Region` and `Tile` above are distinct domain tags, not
strings stored at runtime; the stock enum is part of the key, so flora, fauna, and water do not
receive the same perturbation. Region offsets are shared only by tiles with the same region and
stock, while tile offsets vary independently by tile and stock. The sum changes the base fraction
by at most `0.12` before the clamp. It does not perturb capacity, regeneration, biome, or the
authored profile itself.

Derive the turn-0 biome, season, degradation-zero cap, and any active turn-0 macro factor before
initializing stock. A zero-cap resource initializes to zero regardless of offsets. Iterate tile IDs
and stock enums in stable order even though the function is order-independent. Validate the complete
six-by-three base table, finite entries in `[0, 1]`, exact `0.08`/`0.04` amplitudes, finite hash
output, and final stock in `[0, Turn0ResourceCap]`. Identical seed and configuration produce
identical stocks on desktop and Web; changing the function, domains, amplitudes, or base table
requires a `ResourceAlgorithm` version change. Saves persist the resulting absolute stocks and do
not rerun initialization on load.

### Absolute local temperature

Several rules in this section need a tile's temperature in degrees Celsius, not the signed
`TileTempOffset` the climate model produces. Those two quantities were previously conflated under a
recurring “local-temperature calibration remains open” note. `TemperatureAlgorithm:
"lat-elev-offset-v1"` now closes that high-fan-in input as its own versioned contract:

```text
LatitudeRadians(i)       = abs(LatitudeDegrees(i)) * π / 180
LatitudeFactor(i)        = LatitudeSinSquared[TileRow(i)]
                         = sin(LatitudeRadians(i))^2       // generation definition only
SeaLevelTempC(i)         = EquatorTempC
                         - (EquatorTempC - PolarTempC) * LatitudeFactor(i)
HabitatTemperatureC(i, t) = SeaLevelTempC(i)
                             - LapseRateCPerKm * ElevationKm(i)
                             + HabitatTempOffset(i, t)
LocalTemperatureC(i, t)   = HabitatTemperatureC(i, t) + ClimateNoise(seed, t)
```

The projection in §6 stores latitude in degrees. `LatitudeSinSquared` is therefore a checked-in,
64-entry table keyed by tile row and generated from the degree-to-radian definition above. Its
runtime representation uses exact `float64` bit patterns; the simulation never calls `math.Sin` for
latitude. Generation tests recompute all 64 actual rows with `math.Sin` within a stated tolerance,
assert that the generator returns zero at a synthetic `0°` input and equal values for synthetic
paired `+latitude`/`-latitude` inputs, and lock the table's exact bit checksum. The map rows themselves
need not contain the equator or equal-magnitude north/south pairs. Changing any entry requires a
`TemperatureAlgorithm` version change.

`LatitudeDegrees(i)` and the authored `ElevationKm(i)` in §6 come from the same versioned geography as
every other tile attribute. `HabitatTempOffset(i, t)` is the sum of long-term, seasonal, and regional
abrupt-pulse components; the separate seed/turn `ClimateNoise` is added exactly once to produce
`LocalTemperatureC`. The locked coefficients are **`EquatorTempC = 28.0`**,
**`PolarTempC = -12.0`**, and **`LapseRateCPerKm = 6.5`**, all in degrees Celsius. The ordering
`EquatorTempC > PolarTempC` and positive lapse rate therefore hold by construction. These values
make the fixed 72°N map edge approximately `-8.18°C` at sea level before climate offsets, while the
48°S edge is approximately `5.91°C`; elevation then subtracts `6.5°C` per kilometre. They are
gameplay calibration anchors, not a reconstructed paleoclimate dataset. The function is pure,
consumes no `WorldRNG`, and is evaluated after phase 1 so every consumer in one turn reads the same
value.

At equal latitude and climate offset, a `1.0 km` elevation difference lowers local temperature by
exactly `6.5°C`; every authored height therefore has a direct fixture against its sea-level value.
Changing the elevation catalog is a `GeographyAlgorithm` change; changing this lapse formula or its
coefficient is a `TemperatureAlgorithm` change. The saved pair of identifiers fences both inputs
without assigning the same table to two owners.

The following 64 `uint64` hexadecimal values are the authoritative `float64` bit patterns for
`LatitudeSinSquared[row]`. The latitude column is an audit aid derived from
`72 - row * (120 / 63)` degrees; runtime simulation reads the bit pattern, not the printed decimal
or a trigonometric function.

| Row |            Latitude ° | `LatitudeSinSquared` bits |
| --: | --------------------: | ------------------------: |
|   0 |                  `72` |      `0x3fecf1bbcdcbfa53` |
|   1 |  `70.095238095238102` |      `0x3fec4a74158e4a2f` |
|   2 |   `68.19047619047619` |      `0x3feb9544cbb00e73` |
|   3 |  `66.285714285714292` |      `0x3fead2fae966c0f9` |
|   4 |   `64.38095238095238` |      `0x3fea04723af1d0d9` |
|   5 |  `62.476190476190474` |      `0x3fe92a9466f29d8c` |
|   6 |  `60.571428571428569` |      `0x3fe84657e6187012` |
|   7 |  `58.666666666666664` |      `0x3fe758beec4b7af9` |
|   8 |  `56.761904761904759` |      `0x3fe662d644925232` |
|   9 |  `54.857142857142861` |      `0x3fe565b420fc4f3e` |
|  10 |  `52.952380952380949` |      `0x3fe46276dfe8e430` |
|  11 |  `51.047619047619051` |      `0x3fe35a43c80fe878` |
|  12 |  `49.142857142857139` |      `0x3fe24e45bcb9602e` |
|  13 |  `47.238095238095241` |      `0x3fe13fabeb9c1614` |
|  14 |  `45.333333333333329` |      `0x3fe02fa875e18e67` |
|  15 |  `43.428571428571431` |      `0x3fde3ede2ba6aed1` |
|  16 |  `41.523809523809526` |      `0x3fdc20678573045b` |
|  17 |   `39.61904761904762` |      `0x3fda0652a8e83614` |
|  18 |  `37.714285714285715` |      `0x3fd7f30050b38c23` |
|  19 |   `35.80952380952381` |      `0x3fd5e8c991c869db` |
|  20 |  `33.904761904761905` |      `0x3fd3e9fd335ffd89` |
|  21 |                  `32` |      `0x3fd1f8dd12a0fd1b` |
|  22 |  `30.095238095238095` |      `0x3fd0179b94e387ab` |
|  23 |   `28.19047619047619` |      `0x3fcc90b256e97ad5` |
|  24 |  `26.285714285714285` |      `0x3fc91a43d753a662` |
|  25 |   `24.38095238095238` |      `0x3fc5cfd67bf6d9d2` |
|  26 |  `22.476190476190474` |      `0x3fc2b5234d8ff741` |
|  27 |  `20.571428571428569` |      `0x3fbf9b5aacfb5b04` |
|  28 |  `18.666666666666664` |      `0x3fba397b5357b339` |
|  29 |  `16.761904761904759` |      `0x3fb54abf56256b74` |
|  30 |  `14.857142857142854` |      `0x3fb0d4bb3bfc87a2` |
|  31 |  `12.952380952380949` |      `0x3fa9b8f5f40dc3c0` |
|  32 |  `11.047619047619051` |      `0x3fa2ccf87cd44e1d` |
|  33 |  `9.1428571428571459` |      `0x3f99daa56e9c1737` |
|  34 |  `7.2380952380952408` |      `0x3f9041536ab3d483` |
|  35 |  `5.3333333333333286` |      `0x3f81b1adc52afe06` |
|  36 |  `3.4285714285714306` |      `0x3f6d4c8a96d7cf3f` |
|  37 |  `1.5238095238095184` |      `0x3f472c089c8c8ec9` |
|  38 | `-0.3809523809523796` |      `0x3f072d585328c0f8` |
|  39 | `-2.2857142857142918` |      `0x3f5a0f9238cbbd73` |
|  40 | `-4.1904761904761898` |      `0x3f75def43bbd3a1e` |
|  41 |  `-6.095238095238102` |      `0x3f871715440988a4` |
|  42 |                  `-8` |      `0x3f93d581ca184804` |
|  43 |  `-9.904761904761898` |      `0x3f9e4c41f536698c` |
|  44 |  `-11.80952380952381` |      `0x3fa571fa56f71d4c` |
|  45 | `-13.714285714285708` |      `0x3facc72d7fd4c1e0` |
|  46 |  `-15.61904761904762` |      `0x3fb28eb7651d66bd` |
|  47 | `-17.523809523809518` |      `0x3fb735a7de80573d` |
|  48 | `-19.428571428571431` |      `0x3fbc5324de251bf8` |
|  49 | `-21.333333333333329` |      `0x3fc0f0b27c7b7208` |
|  50 | `-23.238095238095241` |      `0x3fc3ed0f99531496` |
|  51 | `-25.142857142857139` |      `0x3fc71b490cf2d72d` |
|  52 | `-27.047619047619051` |      `0x3fca77c5b676a005` |
|  53 | `-28.952380952380949` |      `0x3fcdfeb81ece67aa` |
|  54 | `-30.857142857142861` |      `0x3fd0d611630288e1` |
|  55 | `-32.761904761904759` |      `0x3fd2bdee536f58d9` |
|  56 | `-34.666666666666671` |      `0x3fd4b4caf5d971a5` |
|  57 | `-36.571428571428569` |      `0x3fd6b86e679815a8` |
|  58 | `-38.476190476190482` |      `0x3fd8c69151c1d424` |
|  59 |  `-40.38095238095238` |      `0x3fdadce07d1a75be` |
|  60 | `-42.285714285714292` |      `0x3fdcf8ff73706a54` |
|  61 |  `-44.19047619047619` |      `0x3fdf188b2b6ff16a` |
|  62 | `-46.095238095238102` |      `0x3fe09c8e5df330f3` |
|  63 |                 `-48` |      `0x3fe1ac2609b3c577` |

For the exact-table test, canonicalize the 64 bit payloads as lowercase 16-digit hexadecimal
strings without `0x`, joined in row order by `\n` with a final newline. Their SHA-256 checksum is
`a034a27d3d70715a766d64d14102952dd75c854316f7d6f2867187e724cbaf43`.

Its consumers, and why the contract cannot wait for the balance pass:

- **`ClassifyBiome`** takes absolute `HabitatTemperatureC`, so the tundra and savanna thresholds are stated in
  °C. This is what makes the advancing ice sheet an emergent consequence of one derived scalar
  rather than a scripted event, and it is exercised in build step 4 — well before §12. The tundra
  boundary remains temperature-driven alone; §7's moisture offset moves the savanna/desert margin
  independently, so the two thresholds respond to different scalars.
- **The water demand multiplier** reads `LocalTemperatureC[b]` directly against its 20 °C and 35 °C
  anchors.
- **The endemic-disease table** is selected by the derived biome, so it inherits this function
  transitively.
- **`ColdAdaptation` and the seasonal exposure components** measure cold pressure against absolute
  temperature, not against an offset whose zero point is arbitrary.

Because biome classification depends on its seed-independent habitat form, build step 4 implements
and verifies this already-locked table and coefficient set. §12 reports the exact habitat/biome
envelope and validates the noise-added local-temperature envelope against `BalanceSeedCorpus`; it
does not retune the locked temperature coefficients.
Validation requires finite coefficients in the stated ordering, finite `HabitatTemperatureC` for
every tile/turn, finite `LocalTemperatureC` at both `UnitNoiseV1` bounds and every corpus seed, and —
as a calibration check rather than a runtime clamp — that the resulting
turn-0 map classifies East Africa as habitable savanna and woodland while the turn-400 map pushes
tundra south of its turn-0 boundary. `TemperatureAlgorithm` is saved as an identifier and validated
on load; the temperatures themselves are derived and never serialized. Changing the function,
latitude table, or coefficients after release requires a version update and explicit migration,
because biome history,
water demand, and disease exposure all shift with it.

### Water units and heat-adjusted demand

**One water unit (`WU`) meets one person's baseline need for one game turn.** This is a normalized
gameplay scale, not liters or water for the historical years represented by a turn. The existing
tile water stock, capacity, regeneration, demand, and allocation all use WU; water never becomes
FU or enters `StoredFood`. Both species use the same baseline `BaseWaterPerPerson = 1.0`.

Regional heat modestly changes per-person demand: hotter conditions require more water than
cooler reference conditions. Compute each band's requirement before shared allocation in phase 3:

```text
HeatIncrement[b] = 0.02 * clamp(LocalTemperatureC[b] - 20.0, 0.0, 15.0)
AridHeatRemaining[b] = 1.0 - 0.40 * HeritableState[b, AridClimateAdaptation]
WaterDemandMultiplier[b] = 1.0 + HeatIncrement[b] * AridHeatRemaining[b]
WaterRequired[b] = P_start[b] * BaseWaterPerPerson * WaterDemandMultiplier[b]
```

`LocalTemperatureC[b]` is the origin's absolute local temperature in degrees Celsius, not
`TileTempOffset` alone. Derive it consistently from latitude/elevation and the current offset
used by the geography/climate model. This includes seasonal warming/cooling, the long-term LGM
and authored abrupt-climate terms, and existing bounded climate noise; reuse that value rather than
drawing new noise.
Equivalent environmental inputs give the same multiplier for either species.

Before heritable adaptation, the approved initial curve has range **`[1.0, 1.3]`**, inclusive: baseline demand at or below
**20°C**, an additive **`0.02` per °C** above 20°C, and maximum demand at or above **35°C**.
The increase is linear, not compounded: each degree within the ramp adds 2% of baseline demand,
not 2% of the previous degree's demand. Use the formula above as the canonical calculation.
Its clamp limits the demand response, not the shared temperature used by other simulation systems.

The anchors, slope, endpoints, and seasonal/long-term sensitivity are selected initial playtest
settings, not physiological estimates. Changing them must preserve one coherent versioned curve.
Build-step 4 calibration validates absolute local temperature and this response together; neither
has a remaining curve or threshold decision.

Capture the multiplier after the advancing turn's phase-1 climate update and before allocation,
using the phase-3 origin and start-of-turn population. Do not use the earlier planning frame's
temperature. Keep the multiplier and requirement fixed for that turn's allocation, normalized shortfall,
and water-mortality inputs. Growth, deaths, or phase-4 migration must not change the denominator,
trigger another extraction, or substitute destination demand. Both species share the existing
proportional tile-stock allocator; workforce assignment does not change population-wide need.

Use the demand multiplier exactly once. It does not also scale water capacity, regeneration,
`WaterHealthLossRate`, or the resulting health loss. Water availability and heat-related need are
separate inputs; do not infer thirst from the amount of stock left after allocation. Direct water
mortality is the named seasonal `Uncovered` drought component; there is no fourth free-standing
dehydration mortality coefficient. Health damage remains the separate normalized-shortage channel.

Validate finite non-negative population, finite `LocalTemperatureC` before the curve's clamp,
the selected baseline and heritable state, a finite multiplier in `[1.0, 1.3]`, and a finite requirement before allocation. Reject invalid inputs/results
rather than repairing them with the response curve's bounds; even a bounded multiplier can overflow
an extreme product.
Zero population requires zero WU; a living band with a positive baseline and multiplier has a positive
requirement. `LocalTemperatureC`, the requirement, and the multiplier are derived, not saved band fields.
`ResourceAlgorithm: "toward-cap-v1"` owns these water units and demand rules, including the initial
20°C/35°C anchors, `0.02` slope, multiplier bounds, and current-climate input, alongside
allocation/extraction; `HazardAlgorithm` owns shortage-related health damage. Use the existing
unreleased contracts, with affected version updates and explicit save migration after release.

Fixtures lock the one-WU baseline, population scaling, inclusive pre-adaptation `1.0`/`1.3` bounds,
the exact `0.40` heritable reduction of only the heat increment, and the selected linear ramp.
At `AridClimateAdaptation = 0`, temperatures `20`, `25`, `27.5`, `30`, `32.5`, and `35` °C produce
multipliers `1.0`, `1.1`, `1.15`, `1.2`, `1.25`, and `1.3` respectively. Test values below
20°C and above 35°C, both endpoints and their immediate neighbors, fractional temperatures, and
rejection of non-finite temperature before clamping. The ramp adds `0.02` per degree rather than
compounding. At fixed population, geography, and other climate components, warming cannot lower
demand and cooling cannot raise it; allow plateaus at the bounds. Across actual turns, compare
total local temperature rather than the long-term trend alone. Distinguish current phase-1 heat
from the earlier planning frame and absolute temperature from a temperature offset. Equivalent
species/iteration orders give equivalent results; migration keeps current-turn origin accounting.

A 100-person band requires 100 WU at/below 20°C, 110 WU at 25°C, 120 WU at 30°C, and 130 WU
at/above 35°C. At 32.5°C it requires 125 WU, so 100 WU allocated leaves a 20% shortfall.
At 35°C, the same allocation leaves a `30/130` shortfall (about 23.08%), not 20%.
Cover zero population, invalid/overflowing inputs, shared-stock conservation, save/load future
equivalence, no extra RNG draw or water inventory, and no second heat multiplier on health damage.
The tile inspector labels existing water amounts in WU without implying that one WU always meets
one person's heat-adjusted need.

### Food units and consumption scale

**One food unit (`FU`) sustains one person for one game turn.** It is a normalized gameplay unit,
not a literal kilocalorie, monthly ration, or food supply for the 50–300 historical years represented
by a turn across the four campaign eras. Both hominin species use the same unit.

Phase 3's baseline food requirement is `FoodRequired[b] = P_start[b]` FU, using the band's
start-of-turn population before growth or mortality. The whole population needs food regardless
of workforce assignment; demographic changes affect the next turn's requirement, not food already
required this turn. For example, a band of 50 requires 50 FU; if its total usable food for that
turn is 40 FU, its food deficit is 10 FU. The nutrition rules below determine its health effect;
the missing FU are not themselves a death count.

Usable food from actual harvest, food consumed, food surplus/deficit, and the existing `StoredFood`
field are measured in FU. Flora and fauna remain separate environmental stocks, but both use
**normalized harvest units**: one actually allocated flora or fauna stock unit has a baseline edible
yield of `1.0 FU`. The common number is a gameplay normalization, not a claim that equal physical
biomass or labor produced the two stocks. Convert each actual allocated harvest once under its
existing role/profile/technology rules; potential or unallocated harvest is not food.
`StoredFood` is already in FU and is not converted again when consumed. Water retains its separate
stock and survival requirement; it cannot satisfy a food deficit or enter `StoredFood`.

Food values shown in the UI use FU and explain the one-person/one-turn meaning, automatic use of
reserves to cover a harvest shortfall, and carry-over of uneaten food up to the final cap. Unit fixtures
must cover populations 20 and 200 requiring 20 and 200 FU, the 50/40/10 example, and unchanged
current-turn requirements after demographic changes. Conversion fixtures must assert the selected
equal baseline edible yield while keeping the two stocks separately conserved, and must never
credit potential harvest or water as edible food. This unit convention adds no new
resource pool or save field. Storage capacity follows the population-scaled rule below. The food,
water, health, and starvation constants this section uses carry the status **Initial** in
Appendix C; the water, health, starvation, and storage subsections below define their use and
versioned configuration. They may be retuned by §12's balance pass, but are not per-band,
per-save, or player-adjustable settings. Starvation's direct, health-independent raw response,
automatic consumption, gradual nutritional health changes, capacity, end-of-turn overflow, and
fixed-percentage spoilage are specified below.

### Normalized harvest conversion and food sources

`FoodConversionAlgorithm: "normalized-source-v1"` classifies actual harvest into the closed transient
`FoodSource` order `Plant`, `Animal`, and `Aquatic`, with `FoodSourceCount = 3`. Its three base
conversion factors are all exactly `1.0 FU` per allocated stock unit. `Plant` receives flora
allocation; `Animal` receives the megafauna role's fauna allocation plus the terrestrial part of the
Hunting role's, which is split from `Aquatic` by the terrestrial-versus-aquatic rate composition
defined below. Freshwater, inshore, and pelagic catch all use `Aquatic`; this
replaces the narrower and otherwise undefined `marine` label.

**Megafauna is a prey group, not a food source.** The `Megafauna` `FaunaGroup` remains: it carries a
profile weight, gates megafauna tracking, and drives that role's higher collection rate. But once
allocated fauna reaches conversion, megafauna and ordinary terrestrial catch are indistinguishable —
the same `1.0 FU` base factor, the same heritable conversion factor, and the same destination in
`FreshFood`. Nothing downstream reads which role produced a terrestrial unit. A fourth enum value
would therefore be a distinction the model never makes, and one more axis every conversion fixture,
wire check, and effect table has to cover.

The equal base factors do not make the sources strategically identical. Biome and profile support,
technology access, per-worker collection rates, shared-stock scarcity, and work risk determine how
much of each source is actually obtained. In particular, megafauna's intended high yield comes from
its higher collection rate, not from silently multiplying the same allocated fauna a second time.
Applicable heritable effects may modify a source's conversion or health-suitability channel only as
specified in their effect table. Acquired technology changes access or collection rates and does
not also multiply FU conversion in v1.

For each band, apply any selected finite non-negative source-specific conversion modifier to each
actual allocation once, then sum the three FU results in stable `FoodSource` order. The modifier is
`1.0` when no effect applies. Conversion must produce finite non-negative FU, conserve the Hunting
allocation across its two outputs before modifiers, and never read or deplete an environmental
stock. It creates at most three transient values per band and no source-specific inventory: after
conversion, all usable food combines into the existing fresh-harvest total and then `StoredFood`.

### Automatic per-band food consumption

In phase 3, each band automatically meets its food requirement from its actual fresh harvest plus
its reserve remaining after phase-1 spoilage. Both species use the same rule, regardless of
workforce allocation; even a band with no food-gathering workers can eat its reserves. This is a
band-local balance, not a shared food pool: another band's surplus cannot cover this band's deficit.

`FreshFood[b]` is the sum of that band's actually allocated `Plant`, `Animal`, and `Aquatic`
harvests converted to FU once under `normalized-source-v1`. Accumulate contributions
in stable `FoodSource` order, regardless of discovery order. Potential or unallocated harvest,
water, and another band's food do not enter this sum. Using the already selected
`FoodRequired[b] = P_start[b]` before any demographic change:

```text
FoodAvailable[b] = StoredFood_after_spoilage[b] + FreshFood[b]
FoodConsumed[b]  = min(FoodAvailable[b], FoodRequired[b])
FoodDeficit[b]   = FoodRequired[b] - FoodConsumed[b]
FoodRemaining[b] = FoodAvailable[b] - FoodConsumed[b]
```

Consumption happens once before growth or mortality; later population changes do not recalculate
this turn's requirement or refund food eaten. A deficit and a positive remainder cannot coexist.
When food is sufficient, consume exactly the requirement, not all available food. All uneaten FU,
including unused old reserves and fresh surplus, flow into the existing phase-5 cap as
`StoredFood_before_cap = FoodRemaining`. Do not add the old reserve again, discard surplus early,
or spoil fresh harvest this turn. Phase 5 publishes only the capped remainder as `StoredFood`.
The existing shared-stock allocator already deducted extraction; eating does not debit tile
flora/fauna again, and discarded food cannot be refunded to those stocks.

There is no source-priority decision, reserve-withholding/rationing action, food-age queue, or
separate fresh-food inventory. Once converted, FU are interchangeable for this accounting. The
remaining food is uneaten, not extra consumption. The gradual-health rule below governs nutritional
health changes, with linear nutritional decline on shortage turns. The squared starvation response
applies directly with initial coefficient `0.10`; the fed fraction scales positive growth only.
The combined health update and chronic vulnerability are specified below. A numerical food deficit
is not directly a death count.

Validate finite non-negative input amounts and source-specific conversion modifiers before arithmetic. These are
mathematical sums: use overflow-safe scaled/wider transient arithmetic if a valid sum or conversion
product exceeds `float64`, carrying the remainder to phase 5 rather than writing infinity into
`StoredFood` or clipping to capacity before eating. Consumption and deficit remain bounded by the
finite requirement; the final reserve remains bounded by the validated finite capacity. Keep
fractional FU and conservative roundoff, with `FoodAvailable = FoodConsumed + FoodRemaining` and
`FoodRequired = FoodConsumed + FoodDeficit`. All food values remain finite in their working
representation. Intermediate totals, source breakdowns, and remaining food are turn-local; at most
one fixed-size food-accounting record per band survives to the final cap, bounded by `MaxBands`.
Only the completed-turn requirement and deficit are retained for display in
`LastFoodReport` below; that copy is not another food balance or a simulation input.
The computation consumes no RNG and runs only inside an advancing turn. Planning, migration,
saving, and loading cannot eat again. The existing `FoodStorageAlgorithm` versions this rule.

Formula fixtures supply post-spoilage reserves and actual harvest already in FU. For a requirement
of 50 FU: reserve 20 plus harvest 30 consumes 50 with zero deficit/remainder; reserve 5 plus harvest
30 consumes 35 with deficit 15 and zero remainder; reserve 90 plus harvest 80 consumes 50 and
leaves 120 before the final cap. Cover reserve-only meals, fresh-only meals, no food, zero
requirement, fractional and extreme finite inputs, source-order invariance, and no cross-band
transfer. Across a full turn, starting reserve plus fresh food equals spoiled, consumed, discarded,
and finally stored food within roundoff. Integration fixtures preserve the selected phase-1
spoilage and final-population cap, including growth, deaths, splits, and terminal turns.

### Proportional food-shortage severity

Measure hunger severity by the fraction of this turn's food requirement left unmet after automatic
consumption, not the absolute number of missing FU:

```text
FoodDeficitFraction[b] = FoodDeficit[b] / FoodRequired[b]   if FoodRequired[b] > 0
                        0                                otherwise
```

This dimensionless value is in `[0, 1]`: zero means needs were met, and one means no food was
available for a positive requirement. The zero-requirement boundary has zero deficit and zero
severity; never divide by zero. Validate finite non-negative inputs with
`FoodDeficit <= FoodRequired` before evaluating the direct ratio; clamping must not hide invalid
inputs. Fractional FU remain valid.

Both species use the same band-local measure. Compute it once in phase 3 from the actual deficit
after spoilage, harvest, and consumption, using the already-fixed start-of-turn requirement.
Later growth, deaths, migration, or overflow discard do not change that turn's fraction. It is
not a raw-harvest shortfall: reserves have already covered whatever they can. For example, a
10-FU deficit against a 100-FU requirement and a 1-FU deficit against a 10-FU requirement both
give `0.10` (10%). Equal fractions mean equal food-shortage severity, not equal absolute losses
or guaranteed identical outcomes when health and other conditions differ.

Use this fraction as the shortage input to the gradual-health model and squared starvation base
below; retain `FoodDeficit` in FU for accounting. A 10% shortfall does not itself select a 10% health
loss or mortality rate. The selected initial loss rate makes a 10% shortfall request `0.02`
health loss, or two percentage points, before the zero floor. Recovery and chronic vulnerability
are specified below. The initial starvation coefficient is `0.10`; the fraction of needs met
scales positive logistic growth, leaving non-positive growth unchanged. Starvation has no health
gate or health multiplier. UI explanations must distinguish missing FU, the percentage of needs unmet, and
actual health/population losses; the last-turn report below supplies the historical food values
for the stats layout in §8.

The simulation uses this transient scalar in its existing per-band food-accounting record, not a
saved hunger meter or food stock. A display-only copy is retained in `LastFoodReport`; it never
feeds a later health, starvation, or food calculation. The computation consumes no RNG, is bounded
by `MaxBands`, and is versioned with the existing unreleased `FoodStorageAlgorithm` contract.
Save/load and planning cannot re-evaluate consumption or accumulate hunger.

Fixtures cover full feeding, no food, the zero-requirement boundary, fractional requirements, and
the two equivalent 10% examples. The 50-required/40-consumed example gives 10 FU missing and a
fraction of `0.20`. Scaling both requirement and deficit by the same positive factor preserves
severity within floating-point tolerance; cover tiny/large finite inputs and reject non-finite,
negative, or above-requirement deficits. Integration fixtures prove reserves reduce the fraction
before demographics and that later population changes cannot change its denominator.

### Last completed-turn food report

The band inspector shows actual food results from the last completed turn, not a forecast under
the current workforce draft or assignment. Each domain `Band` and its projected `gameapi.Band` value carries one
`LastFoodReport FoodTurnReport`, with this fixed-size shape in `gameapi/frame.go`:

```go
type FoodTurnReport struct {
 Turn       int     // 0 means unavailable
 RequiredFU float64
 DeficitFU  float64
}
```

The record stores only the two independent quantities plus the turn they belong to. Consumption and
the unmet fraction the panel displays are **derived at display time**, not stored:

```text
ConsumedFU      = RequiredFU - DeficitFU
DeficitFraction = DeficitFU / RequiredFU   if RequiredFU > 0, otherwise 0
```

Storing either one alongside its inputs would make the same fact representable twice, and every
persisted duplicate becomes a consistency invariant that save/load must police and a way for a
migration to produce a self-contradictory record. Availability is encoded the same way: valid turns
are `1` through `400`, so `Turn == 0` *is* the unavailable state and no separate `Valid` flag exists
to disagree with it.

In phase 3, capture `nextTurn`, `FoodRequired`, and `FoodDeficit` from the existing
automatic-consumption calculation before health, growth, or mortality. Do not calculate another meal
or use final population as a denominator. Keep this candidate in the bounded turn-local accounting
record; publish it for each surviving band only at completed-turn finalization in phase 5, before
building the frame or starting an autosave. Growth, deaths, migration, and overflow discard do not
alter these captured actuals. Moving bands report the food required at the origin, not a destination
harvest.

The lifecycle is the same for both species:

- New-game bands at turn 0 have no report: `Turn = 0`, with both FU fields zero.
- Every completed turn replaces each survivor's prior report, including terminal turns.
  Cleanup discards extinct bands' candidates and reports; this feature adds no extinct-band
  archive or event-feed entries.
- Accepted assignment/research changes and queued migration preserve the report. Draft edits,
  selection, snapshots, pausing, and storage operations cannot refresh it or advance its turn.
  Rejected commands/turn requests and a failed computer-planning batch leave it unchanged.
- A successful player or computer split clears the report to the all-zero unavailable value on
  **both** resulting bands, matching the existing last-mortality rule. Do not divide, copy, or relabel the
  parent's actuals as either descendant's own turn. A rejected split preserves the original.
- Saving preserves the report as part of that captured world revision. Successful load restores
  it exactly, without replaying food consumption; failed load leaves the current report intact.
  Reports from a previously loaded world never leak into the replacement world.

An available report has `1 <= Turn == World.Turn <= 400` at every externally visible checkpoint.
`RequiredFU` and `DeficitFU` are finite and non-negative with `DeficitFU <= RequiredFU`; a zero
requirement requires a zero deficit. Those are the only invariants, because the shape admits no
derived field that could disagree with them. A report's requirement need not equal the surviving
population or its current storage capacity. `Turn == 0` requires both FU fields zero and means
unavailable, never a measured zero shortage. Validate this on save/load without silently repairing
an inconsistent report.

The record is display-only, persists under §9, and consumes no RNG. It adds at most `MaxBands = 256`
fixed-size reports to world/save/frame state plus bounded turn-local candidates, not a log,
resource pool, forecast, command, hunger streak, or new storage subsystem. No demographic,
workforce, migration-ranking, or computer-policy calculation may read it. The existing
`FoodStorageAlgorithm` owns the source accounting; the save schema owns this report's wire shape.
The display report does not own or override food-conversion, storage/spoilage, health, or starvation
configuration; those contracts are specified separately in this section.

Fixtures pin the stored values and the two derivations: a stored `{Turn, 50, 10}` must display
40 consumed and a `0.20` unmet fraction, and a zero requirement must display zero consumed and a
zero fraction without dividing by zero. Use a survivor whose population differs from its
start-of-turn requirement, so a renderer that reconstructs the report from current state fails.
Reject non-finite, negative, over-required, and wrong-turn reports, and an unavailable report
carrying a nonzero FU field.

### Last completed-turn outcome report

Population and health can move independently, and current values alone cannot explain either
change. Each domain and projected band therefore carries one fixed-size display-only record:

```go
type OutcomeReport struct {
    Turn                    int // 0 means unavailable
    StartingPopulation      uint32
    EndingPopulation        uint32
    Growth                  float64
    StartingHealth          float64
    EndingHealth            float64
    NutritionDelta          float64 // signed: recovery is positive
    WaterHealthLoss         float64
    DiseaseHealthLoss       float64
    GeneticBurdenHealthLoss float64
    MacroHealthLoss         float64
    AcuteDiseaseHealthLoss  float64
}
```

The domain representation uses its corresponding `Population` and `Health` value types. Population
endpoints are the exact whole-person values before phase 3 and after phase 5; their subtraction is
the displayed net change. `Growth` is the signed food-adjusted logistic term before combined
mortality and whole-person rounding. The separate existing `LastMortality` record remains the source
for applied starvation, seasonal, chronic, macro, and acute population-loss causes, avoiding a
duplicate mortality vector here. Health endpoints bracket the full turn. Phase 3 records the signed
nutrition contribution and raw non-negative water, endemic-disease, and genetic-burden losses;
phase 5 records actual macro and outbreak losses after the health floor. Clamping can make the net
health change differ from a naive sum, so presentation always uses endpoints for the delta and
components only to explain causes.

`Turn == 0` requires every field to be zero and means unavailable. Otherwise the turn equals
`World.Turn`, both starting endpoints are valid and the ending endpoints equal the band's current
population and health. Growth and nutrition are finite signed values; health-loss components are
finite and non-negative. New-game bands start unavailable, every completed turn replaces the
report, extinct bands leave no report archive, and a split clears both descendants. Planning,
snapshotting, and save/load preserve but never recalculate it. No simulation, computer-policy,
migration-ranking, or balance rule may read the report.

### Persistent health and combined response

`Health` is the band's persistent condition score, stored as a validated value over Go `float64` in domain band state
and `gameapi.Band`, with the inclusive valid range `[0.0, 1.0]`. It must be finite: `0.0` is the
lowest condition and `1.0` is full health. Both species use these same fixed bounds. It is neither
an integer percentage nor a count or fraction of healthy/surviving people, and the endpoints add
no separate population-loss or terminal rule. `HealthVulnerability` reads the updated normalized
value using the selected `1 + (1 - Health)` mapping below.

Carry health from one completed turn to the next, rather than recalculating it from this turn's
food ratio. Preserve fractional precision through simulation, snapshots, and saves; display
`100 * Health` as a percentage in §8 and keep any presentation rounding out of the authoritative value.
For example, `0.375` represents 37.5% health, independent of population.

**New-game health is `1.0`.** During `domain.NewWorld(seed)`, initialize every scenario band of both
species to full health at turn 0, independent of seed, region, population, or starting food.
This initialization itself consumes no RNG, advances no turn, and performs no health update.
It does not fill reserves, create a completed-turn food report, or grant a grace period or immunity:
the first advancing turn uses the same consumption, health, and starvation rules as later turns.
This is a new-campaign initializer, not a default applied whenever a band value is constructed.

The nutritional contribution has two mutually exclusive branches:

- A positive `FoodDeficitFraction` contributes the linear health loss below, with no well-fed recovery.
- Fully met food needs contribute the fixed recovery amount below, not an unconditional reset or a
  fraction of missing health. Water shortages and disease can offset or exceed that recovery.

Neither branch writes or clamps health independently; the combined phase-3 update below does so once.

**Linear nutritional decline.** For a shortage turn, use the existing post-reserve
`FoodDeficitFraction` to calculate a signed contribution to the one phase-3 health update:

```text
NutritionHealthDelta[b] = -HealthLossRate * FoodDeficitFraction[b]
```

`HealthLossRate` is one fixed, finite positive configuration coefficient shared by
both species, measured in normalized health-score units per completely unfed game turn before the
zero floor. Require `HealthLossRate > 0`; the approved initial playtest value is **`0.20`**.
This means a completely unfed turn requests a loss of 20 health percentage points. It is not the
starvation coefficient, a death rate, or a percentage of current health. Do not multiply this
decrement by population, workforce, current health, or a hunger streak. At the same rate, equal
unmet-food fractions request equal health-score decrements regardless of band size or prior health;
the floor can reduce the actual drop for a band already near zero.

With other health losses absent and before the combined floor, a 20% shortfall causes twice the
nutritional health loss of a 10% shortfall, not four times.
With the selected initial `HealthLossRate = 0.20`, a band at `0.80` health with a 10% shortfall loses
`0.02` health (two percentage points), ending at `0.78`; a 20% shortfall loses `0.04`, ending
at `0.76`. Starting at `0.01` with the same 10% shortfall ends at zero, not negative health.
A 50% shortfall loses `0.10`; a completely unfed turn loses `0.20`, before the zero floor.
The coefficient is an approved playtest default, not a historical estimate or a completed balance pass.

Validate finite `Health_before` and deficit fraction in `[0, 1]`, plus the positive finite rate,
before arithmetic. Keep `float64` precision, with no deliberate rounding to integer percentages or
minimum-loss threshold. Multiplication by a fraction at most one keeps the requested decrement
bounded by the finite rate; the floor must not conceal invalid input. A zero fraction gives zero
nutritional loss, but the shortage branch does not run on a fully fed turn: that turn uses the
fixed-recovery rule below. Zero loss does not mean zero recovery.

**Fixed well-fed recovery.** For a living band with `FoodDeficitFraction = 0`, use the other
branch of the nutritional contribution:

```text
NutritionHealthDelta[b] = HealthRecoveryRate
```

The approved initial playtest value is **`HealthRecoveryRate = 0.05`**, five health percentage
points per fully fed game turn before the cap. It is a separate configuration constant, not
computed from `HealthLossRate`, population, current/missing health, or surplus food. Require a
finite `0 < HealthRecoveryRate < 1` so recovery remains gradual; validate it before arithmetic.
Both species use the same rate. Even a small positive shortfall selects decline, not recovery;
a living band at zero health can recover a positive net amount when other losses allow it,
while an extinct band receives no update.

With no other health effects, `0.50` becomes `0.55`; `0.98` reaches `1.0`; `1.0` stays `1.0`.
A band at `0.80` with a 50% shortfall falls to `0.70`, then two fully fed turns restore it through
`0.75` to `0.80`. These are health-score changes, not population changes. Recovery does not depend
on whether fresh food or surviving reserves met the requirement.

Food recovery depends on meeting needs after fresh food and reserves, not on gross harvest or
uneaten surplus. Full feeding contributes `+0.05` even when water shortage or endemic disease
contributes damage; there is no second recovery-eligibility gate for those conditions. Their
combined effect may still lower health. Do not consume extra food or grant a surplus-healing bonus.

**Linear water-shortage damage.** Use the band's existing population-wide water demand and its
actual phase-3 allocation, in the same water units. They are not FU or another food calculation:

```text
WaterDeficitFraction[b] = (WaterRequired[b] - WaterAllocated[b]) / WaterRequired[b]
                         if WaterRequired[b] > 0, otherwise 0
WaterHealthLoss[b]      = WaterHealthLossRate * WaterDeficitFraction[b]
```

Validate finite `0 <= WaterAllocated <= WaterRequired`; zero requirement must have zero allocation.
The fraction is in `[0, 1]`. Use demand fixed before growth/mortality and the origin's actual shared
allocation, not final population, potential availability, or destination water. Do not extract or
consume water again. `WaterHealthLossRate` is a separate finite positive configuration coefficient
shared by both species, with approved initial playtest value **`0.40`**. The one-WU baseline
and selected heat-demand curve are specified above; absolute local temperature comes from the shared,
Locked `lat-elev-offset-v1` table and coefficient contract that build step 4 implements.
A 50% water shortfall requests `0.20` health loss (20 percentage points); no water requests
`0.40`, before the combined clamp and other health contributions. The loss is in normalized
health-score units, not people or a fraction of current health. Do not multiply it by regional
heat again: the unmet fraction already uses the adjusted requirement. The rate is a tunable
playtest default, not a physiological estimate or the direct water-mortality coefficient.

**Background disease.** Each origin's current biome selects two distinct, non-overlapping
disease-health rates: `CampDiseaseHealthRate(origin)` and `NonCampDiseaseHealthRate(origin)`.
The approved initial playtest table is:

| Current biome         | Camp-related rate | Non-camp rate | Unmitigated total |
| --------------------- | ----------------: | ------------: | ----------------: |
| Riverine Woodland     |           `0.020` |       `0.010` |           `0.030` |
| Savanna               |           `0.010` |       `0.010` |           `0.020` |
| Coastal Shrubland     |           `0.010` |       `0.010` |           `0.020` |
| Semi-Arid Desert      |           `0.005` |       `0.005` |           `0.010` |
| Mountainous Highlands |           `0.005` |       `0.005` |           `0.010` |
| Glacial Tundra        |           `0.005` |       `0.005` |           `0.010` |

Each rate is normalized health-score loss per game turn before mitigation: `0.020` means two
health percentage points, not a death rate or fraction of current health. The table is complete
for the six land biomes, with no implicit default. It applies identically to both species and all
regions and seasons; current biome is derived after phase-1 climate, so climate reclassification
selects the new row without an additional seasonal or regional multiplier. These are tunable
game-balance defaults, not epidemiological estimates. The table is independent of the chronic
mortality coefficients. Non-camp disease means only disease unrelated to camp conditions, not all
`Uncovered` risk; predation, exposure, falls, and direct water shortages must not become
disease-health damage.

In phase 3, both species use the current origin profile and captured shelter share, before the
combined health update, vulnerability, and demographics. Evaluate the camp term, then the
non-camp term, and add them once:

```text
CampDiseaseHealthLoss[b] = CampDiseaseHealthRate(origin) *
  RemainingRisk(CampDiseaseHealthTech(b), CampMitigation(b, CampDisease, origin)) *
  DiseaseHealthGeneticRemainingRisk(b, CampDisease)
NonCampDiseaseHealthLoss[b] = NonCampDiseaseHealthRate(origin) *
  RemainingRisk(NonCampDiseaseHealthTech(b), 0) *
  DiseaseHealthGeneticRemainingRisk(b, NonCampDisease)
EndemicDiseaseHealthLoss[b] = CampDiseaseHealthLoss[b] + NonCampDiseaseHealthLoss[b]
```

Use the shelter subsection's `RemainingRisk(tech, camp) = (1 - tech) * (1 - camp)` directly.
`CampMitigation(b, CampDisease, origin)` is `CampHygieneScale * ShelterCurve(captured_share, 0)`:
hygiene reduces only the camp-related health loss. Neither disease term reads `NaturalShelter`,
so caves provide no disease-health bonus. Non-camp disease has zero camp mitigation, not an
exemption from applicable technology.
`DiseaseHealthGeneticRemainingRisk` is a finite `[0, 1]` factor from the frozen start-of-turn
`InnateImmuneReactivity` state and is `1` for any disease component not covered by its authored
effect table. It neither receives hygiene nor repeats in the biome base rate.

`CampDiseaseHealthTech(b)` and `NonCampDiseaseHealthTech(b)` are technology-only fractions
from the existing acquired-tech modifier tables, explicitly applicable to the corresponding
background-disease **health** component. No applicable acquired effect means zero.
Validate each technology fraction and genetic remaining-risk factor in `[0, 1]`. A mortality-, probability-, or severity-only effect does not
automatically affect health; partial research and technologies
acquired at turn end do not apply here. Do not include hygiene in these fractions or apply any
effect again to the base rates, aggregate disease loss, or combined health sum. Exact applicable
technology effects and their numerical values are the technology-mitigation rows in the shelter subsection.

Validate finite rates, mitigation inputs, products, and their sum before the health update.
Do not clamp each component against remaining health: only the existing combined phase-3 clamp
limits the result. Do not scale these losses by population, current health, actual deaths, or
`HealthVulnerability`; equal profiles and protection give equal health-score losses across band
sizes. Migration cannot replay background damage at the destination. Direct chronic deaths
retain their separate mortality inputs, mitigation, and caps.

This adds two fixed health contributions per living band, at most `512` across `MaxBands = 256`,
not new protection classes, population-loss terms, persistent state, or RNG draws. The rates and
applicable health-tech effects are versioned `HazardAlgorithm` configuration; the contributions
are transient. Outbreaks use the separate phase-5 health-loss rule below; acute camp protection
still reduces event probability, never the hit's severity.

**Other heritable health burdens.** The genetics effect table derives fixed bounded phase-3
health-score losses for vitamin-D mismatch and low-pathogen inflammatory/allergic burden.
UV damage uses its selected chronic-mortality channel, while `FattyAcidMetabolism` affects source
conversion and selection rather than adding a second health loss. Each named pressure contributes at
most once, in stable trait order; an effect already represented by an endemic-disease health factor
or direct mortality component cannot reappear here. Their sum is
`GeneticBurdenHealthLoss[b]`, finite and non-negative. These are transient band-level values derived
from the frozen start-of-turn heritable vector, not persistent diagnoses or new RNG draws.

**One combined phase-3 health update.** After food consumption and water allocation, determine all
contributions from their original inputs, then evaluate in this fixed order:

```text
HealthSum[b] = Health_before[b] + NutritionHealthDelta[b]
               - WaterHealthLoss[b] - EndemicDiseaseHealthLoss[b]
               - GeneticBurdenHealthLoss[b]
Health_after_phase3[b] = clamp(HealthSum[b], 0, 1)
```

`Health_before` is the carried-forward condition, including any previous outbreak. Validate finite
inputs, non-negative loss terms, and a finite signed sum before the one clamp; supported
configuration must keep the arithmetic finite. Never clamp nutrition first or floor each loss
against the current health separately. With illustrative loss inputs, `0.98 + 0.05 - 0.02 - 0.02`
ends at `0.99`, not `0.96`. Those water/disease loss amounts are fixtures, not chosen rate defaults.
Assign the result once to `Health`; the subsequent phase-3 chronic vulnerability reads this combined
value before growth/mortality. Each health decrement is applied once and is not a population loss.

**Outbreak health damage in phase 5.** After migration, an already-selected `DiseaseOutbreak`
retains its existing direct mortality. If the band survives that incident, apply one additional
health loss using the **same severity draw `v`** already used for its direct deaths:

```text
BaseOutbreakHealthLoss[b] = lerp(MinOutbreakHealthLoss, MaxOutbreakHealthLoss, v)
OutbreakHealthLoss[b] = BaseOutbreakHealthLoss[b]
                        * (1 - OutbreakHealthTech(b))
                        * OutbreakHealthGeneticRemainingRisk(b)
```

The approved initial playtest bounds are `MinOutbreakHealthLoss = 0.05` and
`MaxOutbreakHealthLoss = 0.15`, shared by both species and all regions. Before technology, an
outbreak removes 5–15 percentage points of normalized health, not 5–15% of current health or
population. These are tunable game-balance defaults, not physiological estimates. The existing
`WorldRNG.Float64()` severity draw is finite in `[0, 1)`; do not draw again, force an endpoint,
use the event-selection draw, or substitute the mitigated mortality `loss_fraction`. The upper
health bound is the interpolation limit, not a requirement to produce a draw of `1`.

`OutbreakHealthTech(b)` is a finite technology-only mitigation fraction in `[0, 1]`, from acquired
technology explicitly applicable to the `DiseaseOutbreak` **health-damage** channel. No applicable
acquired effect means zero. Probability-only, direct-mortality-only, and background-disease-only
effects do not automatically protect outbreak health. Partial research and technologies acquired
at turn end do not apply to this incident. Apply the selected technology-mitigation table and the selected
`InnateImmuneReactivity` microbial remaining-risk factor once, without a second mitigation pass.
`OutbreakHealthGeneticRemainingRisk` is a finite `[0, 1]` factor from the frozen start-of-turn
heritable state and is `1` when no trait is authored for this channel. A genetic probability or
severity factor does not automatically enter the health hit; coverage is explicit per channel.

Hygiene affects only the camp-related outbreak probability components before event selection;
no camp work, `NaturalShelter`, or probability-mitigation factor reduces this conditional health
hit. Do not scale the hit by population, current health, `HealthVulnerability`, or the number or
fraction killed. The same draw correlates outbreak severity across the two channels, but health
bounds and health-technology/genetic effects remain independent of direct-death bounds and mitigation.

Validate finite configuration with `0 <= MinOutbreakHealthLoss <= MaxOutbreakHealthLoss <= 1`,
a valid severity draw, technology fraction, genetic remaining-risk factor, and finite derived losses before applying the floor.
`0 <= OutbreakHealthLoss <= BaseOutbreakHealthLoss <= MaxOutbreakHealthLoss`; technology `1`
eliminates this health loss only, not the event or its direct deaths; a zero genetic remaining-risk
factor has the same channel-local effect. Apply once before
completed-turn publication:

```text
Health_after_outbreak[b] = max(0, Health_after_phase3[b] - OutbreakHealthLoss[b])
```

Other incident kinds and no-event outcomes do not apply this outbreak loss. An extinct band
receives no further health update. A selected outbreak with zero deaths can still damage survivors'
health; do not equate damage with `M_acute`. With the initial bounds, `v = 0.50` and no applicable
health technology and genetic factor `1` give loss `0.10`: `0.70` health becomes `0.60`. A 50% applicable health-tech
fraction reduces that hit to `0.05`, leaving `0.65`. A hit exceeding remaining health floors it
at zero, and zero health alone does not kill a surviving band.

The outbreak result is the condition stored in the completed frame/save and carried to the next
turn. Do not replay nutrition, dehydration, endemic damage, growth, or earlier mortality, and do
not retroactively recalculate phase-3 vulnerability. `HealthVulnerability` still scales only the
existing chronic mortality components, not these health-loss amounts or acute severity. The base
and mitigated losses are bounded transient work, at most one pair per surviving outbreak band
and no more than `MaxBands = 256` pairs. No extra RNG draw, infection tracker, population-loss
entry, persistent field, or command is introduced.

**Direct deaths remain separate.** Retain water mortality in the seasonal `Uncovered` drought term, endemic
disease mortality in the existing chronic terms, and outbreak deaths in `M_acute`, with their
existing caps and mitigation rules. Tune health damage and direct-death severity independently.
Health damage represents condition loss among survivors, not another deduction from population or
another entry in the mortality breakdown. Do not relabel population-loss event fields as health
points. No injury/infection state, health-history log, water inventory, command, or extra RNG is added.

Population changes alone do not rescale or reset condition. Splits copy current `Health` exactly
to both descendants. Movement, assignments, research, snapshots, saving, and loading do not heal
or apply either health checkpoint. Both species use the same lifecycle; no new-game initializer
is reapplied after a split or load. `HealthSum`, the contributions, water fraction, and named
checkpoint values are bounded transient work per band, not additional saved or frame fields.
Only the existing current `Health float64` persists, at most `MaxBands` values.

**Health-dependent chronic vulnerability.** After the phase-3 health update, evaluate:

```text
HealthVulnerability(b) = 1 + (1 - Health[b])
```

This is a finite multiplier in `[1, 2]`, not a probability or remaining-risk fraction. Health
`1.0`, `0.50`, and `0.0` produce `1.0`, `1.5`, and `2.0` respectively. Full health retains the
configured chronic baseline; it does not grant immunity. Do not clamp this multiplier to `[0, 1]`.
Apply it once to each existing origin-tile chronic-risk component before the chronic-rate cap and
shared mortality cap. Keep component-specific technology/camp mitigation unchanged. Caps may
limit the final difference in deaths, so twice the component rate need not mean twice the applied loss.

Read the updated health, not the turn-start value: a fully fed band starting at `0.50` reaches
`0.55` and uses `1.45`; a completely unfed band starting at `1.0` reaches `0.80` and uses `1.20`,
with other health effects held absent. This mapping supplies no modifier to raw starvation,
seasonal losses, acute-event probability/severity, or growth, and creates no separate mortality
cause. The food-growth coupling is specified separately in the turn pipeline.

This section's five health coefficients are **Initial** in Appendix C and may be retuned
independently before release; none is a per-band or per-save setting.
Validate input state before any update's boundary clamp.
Reject NaN, either infinity, and values below zero or above one at new-game/save/load validation;
do not silently repair invalid state, interpret `75` as 75% health, or clamp it into a valid save.
The existing `HazardAlgorithm` contract owns the health representation, bounds, new-game value,
linear nutritional-loss and fixed-recovery rules and rates, normalized water-deficit damage,
endemic/outbreak health channels, the two endemic health-rate inputs and component-specific
health-tech effects, the outbreak bounds and shared-draw linear health-loss rule with explicit
technology/genetic mitigation, the combined phase-3 clamp and phase-5 outbreak checkpoint, coexistence with direct
mortality, the vulnerability mapping, and associated health configuration;
changes after release need the affected version update and explicit migration, with a schema-version
change too if the wire shape changes.
`GeneticSelectionAlgorithm` separately owns each heritable factor/burden function and its coverage
table; `HazardAlgorithm` owns where those named factors compose with health and mortality. Changing
either side after release updates both identifiers when their shared integration contract changes.
The health representation remains part of the unreleased hazard v1; heritable behavior uses the
separate genetics identifiers specified below.

Persist and snapshot the existing single `Health float64` value per band, at most `MaxBands = 256`
scalars. Do not add a saved percentage or convert through `float32`. Apart from the display-only
last-turn food report, there is no hunger streak, multi-turn nutrition history, disease/infection
state, worker role, or player command. `HealthVulnerability` is derived, not a saved or frame field.
The combined phase-3 update and vulnerability calculation consume no RNG. Outbreak health damage
adds no draw beyond the existing event selection and severity draws.

Representation fixtures cover `0.0`, `1.0`, and fractional values such as `0.375`, plus a value
that would change if narrowed to `float32`. Require exact `float64` preservation in frame copies,
split inheritance, and JSON save/load, with no display-rounding writeback. Reject NaN, both
infinities, and values just outside `[0, 1]`; valid endpoints must remain valid. Percentage
conversion maps `0.0`, `0.375`, and `1.0` to 0%, 37.5%, and 100%, without scaling population or
changing the selected health-response coefficients. New-game fixtures assert exact `1.0` health for every
sapiens and archaic scenario band across seeds, including the initial frame and turn-0 save, with
no extra RNG draw or health update from health initialization. The food report stays unavailable
until a turn completes. Separately, split a band at `0.375` through either authority and load a
save at `0.375`: both descendants and the loaded band must retain `0.375`, not reset to `1.0`.
A first-turn shortage must use the selected `0.20` loss rate without a startup grace period.
Isolated nutritional fixtures hold water and endemic-disease losses at zero and omit outbreaks;
combined fixtures below exercise the selected composition.

Linear-loss fixtures cover fractions `0`, `0.10`, `0.20`, and `1`, the examples above, repeated
shortages without recovery, equal fractions across band sizes/species, and different prior health
with and without a binding zero floor. Check rate scaling, a small positive fractional decrement,
zero starting health, finite extreme rates, and rejection of zero/negative/non-finite rates or
invalid health/fraction inputs. At zero deficit assert zero nutritional loss, not zero total health
change; recovery remains a separate nutritional branch. Prove one combined origin-phase update,
no population deduction from the health loss itself, no extra RNG, and matching subsequent decline
after split/save/load for equal health and health-driver inputs. The selected-default fixtures lock `0.20`; alternative rates
exercise arithmetic/validation only and do not replace that default.

Recovery fixtures lock `0.05` and cover the `0.50`, `0.98`, and full-health examples, recovery
from zero, two fully fed turns repairing the 50%-shortfall example, and repeated shortages without
recovery. Test small representable positive deficits without adding a shortage tolerance; only
the existing computed zero deficit selects recovery. Cover invalid/non-finite recovery rates and
health, equal nutritional changes across band sizes/species, and reserves preventing a harvest-only
shortfall. Vulnerability fixtures lock the three endpoint/midpoint multipliers and the updated-health
examples, reject invalid health, and prove no `[0, 1]` clamp, second multiplier, immunity at full
health, extra RNG, or direct effect on starvation/seasonal/acute terms. Check component-rate scaling
both without and with the chronic/shared mortality caps, not an unconditional doubling of deaths.
Also test no extra recovery from uneaten surplus, exactly one combined phase-3 update before
vulnerability and demographics, split inheritance, and identical future health after save/load. A near-maximum
band may reach the cap; a depleted band must not be reset to maximum merely because it ate.
Reject non-finite/out-of-range state without replacing the running world. Planning and rejected
turns preserve health.

Water fixtures cover no, partial, and complete shortfall, zero demand with zero allocation, finite
extreme values, and rejection of negative/non-finite amounts, over-allocation, and invalid rates.
Equal unmet-water fractions use equal decrements across band sizes/species/regions, even when
their absolute requirements differ. Lock Appendix C's `WaterHealthLossRate`; at its initial value,
shortfalls `0`, `0.20`,
`0.50`, and `1` request losses `0`, `0.08`, `0.20`, and `0.40`. The 32.5°C
100/125-WU case above requests `0.08` loss; the 35°C 100/130-WU case requests
`0.40 * (30/130)` (about `0.0923077`). Do not apply the heat multiplier again or repeat
extraction/consumption. Other rates exercise arithmetic only, not replacement defaults.
Background-disease fixtures lock all six rows of the selected table and assert shared values across
species, regions, and seasons at a fixed current biome. Cover every climate-driven transition
between valid biome rows, with the phase-1 biome selecting phase-3 rates exactly once. With all
technology/hygiene mitigation absent and genetic remaining risk `1`, assert unmitigated totals
`0.030`, `0.020`, and `0.010` for their listed rows.
A fully fed, fully watered band with no other damage has net health contribution `+0.020` in
Riverine Woodland, `+0.030` in Savanna/Coastal Shrubland, and `+0.040` in the other three biomes
before the combined clamp. Separately, with non-default arithmetic rates `0.04` camp-related and
`0.02` non-camp, camp technology
`0.50` and hygiene `0.50` leave camp loss `0.01`; with no non-camp technology, total endemic loss
is `0.03`. These are arithmetic fixtures, not approved rate defaults or a chosen hygiene balance.
Cover explicit zero rates, zero technology/hygiene, component-specific technology, and technology
`1` eliminating only its applicable component. Vary `NaturalShelter` alone without changing either
loss; hygiene must not reduce the non-camp portion. Unrelated or mortality-only tech and unfinished
or end-of-turn research cannot supply a health effect. Apply remaining-risk products directly,
including small positive residuals, without double mitigation or per-component health clamping.
Reject missing/negative/non-finite rates, invalid mitigation fractions, and non-finite totals.
Equal inputs give equal health loss across populations/species. Prove origin-only evaluation,
camp-then-non-camp order, no health/vulnerability feedback, no derivation from deaths, and no extra
RNG or mortality charge; the sum contributes exactly once before vulnerability.
Add separate and combined `InnateImmuneReactivity` remaining-risk fixtures; unrelated traits must
leave both disease terms unchanged, and no genetic factor may be applied to the biome rates twice.

Combined fixtures verify the signed sum and single clamp, including
`0.98 + 0.05 - 0.02 - 0.02 = 0.99`, followed by vulnerability `1.01`; also cover recovery
outweighed by other losses, recovery from zero with a positive net sum, binding bounds, and
non-finite arithmetic rejected before clamping. Nutritional recovery must not be capped before
subtracting other losses. Each source contributes once; direct mortality retains its existing
causes and caps without another population deduction from health points.
Add one-at-a-time and combined fixtures for each genetic health burden, prove stable trait order,
and reject a coverage table that routes the same pressure through both an existing disease factor
and `GeneticBurdenHealthLoss`.

Outbreak fixtures lock Appendix C's outbreak bounds: at their initial values, with no technology and genetic remaining
risk `1`, severity draws `0`, `0.25`, `0.50`, and
`0.75` give unmitigated health losses `0.05`, `0.075`, `0.10`, and `0.125`. Cover the largest
supported draw below one without requiring a draw of one, and nondecreasing loss at fixed technology.
Test the `0.70 - 0.10 = 0.60` example and a 50% applicable health-tech fraction with genetic
remaining risk `1` giving `0.65`;
zero tech preserves the base, and full tech removes health damage without canceling the event or
its deaths. Add independent genetic-factor fixtures and the technology/genetic product. Vary
direct-death bounds/mitigation, population, health, and vulnerability while holding
`v`, health-tech, and health-genetic inputs fixed: requested health loss must not change, though the floor can bind.
Vary hygiene and natural shelter with the selected kind and severity draw held fixed: neither may
reduce the hit. Unrelated, partial, or end-of-turn technology cannot supply a health modifier.
Reject invalid/reversed/non-finite bounds, a draw outside `[0, 1)`, invalid technology fractions or
genetic remaining-risk factors,
and invalid derived loss before clamping. Cover the zero floor, no event, other event kinds,
a zero-death outbreak with positive health damage, and an outbreak that kills the band.
Apply damage only to survivors, reuse the same `v` rather than mortality's mitigated loss fraction,
retain the original one-draw/no-event and two-draw/event rules, and never replay phase-3 work or
retroactively change its vulnerability. The completed frame/save must expose post-outbreak health;
split/save/load must preserve it exactly and reproduce future health with identical inputs and RNG state.

These fixtures lock the rules and initial defaults above. Non-default arithmetic inputs exercise
arithmetic and validation only; they are not replacement rates.

### Quadratic starvation response

Starvation follows a squared-shortfall curve, using the reserve-aware `FoodDeficitFraction` rather
than a ratio based only on freshly extracted food. The response is:

```text
raw_starvation[b] = P_start[b] * StarvationCoefficient * FoodDeficitFraction[b]^2
```

`StarvationCoefficient` is the curve's β: a fixed, finite non-negative configuration coefficient,
not a worker share, food conversion, or random probability. The initial playtest value is `0.10`,
shared by both species and subject to later balance tuning. The result
is an absolute number of people before the shared mortality cap. For the same starting population
and coefficient, a 20% shortfall gives four times the raw starvation loss of a 10% shortfall;
a 50% shortfall gives 25% of the no-food raw loss. Zero deficit gives zero raw loss, including
when reserves fully cover a harvest shortfall. With no food and a positive requirement, the raw
loss is `P_start * StarvationCoefficient`; zero population gives zero without dividing by zero.
With the selected initial coefficient, a 100-person band has raw starvation losses of `0`, `0.1`,
`0.4`, `2.5`, and `10` people for deficit fractions `0`, `0.10`, `0.20`, `0.50`, and `1`.
Thus a 50% shortfall means 2.5% raw loss and no food means 10%, not necessarily the final applied
loss after all causes share the population available after growth.

Apply this curve directly whenever food needs remain unmet. There is no health threshold,
minimum delay, minimum shortfall, hunger-streak requirement, or
extra health multiplier. With a positive coefficient, even a first shortage at full health
contributes the curve's fractional raw loss; do not round the curve itself. The shared population
checkpoint rounds survivors only after all phase-3 growth and mortality terms have been combined.
Zero deficit gives zero raw starvation even at poor health.

For identical starting population, coefficient, and deficit fraction, changing health cannot change
`raw_starvation`. Health still carries nutritional consequences and feeds the separately specified
`HealthVulnerability` calculation for chronic losses. Applied `M_starvation` can differ if those
other causes change the shared mortality scale; health independence is a property of the raw
starvation term, not a promise of identical final cause attribution under a binding cap.

Compute `raw_starvation` once in phase 3 after automatic consumption, with the fixed
start-of-turn population and shortage fraction. Both species use the same formula. Growth,
mortality, movement, and overflow discard do not change its inputs or cause another evaluation.
It is one computed quantity, not a saved field.
`raw_starvation` participates in the existing shared proportional phase-3 mortality cap; only
the resulting `M_starvation` is subtracted and shown as applied starvation loss. Neither it
nor the food-deficit fraction is a second mortality term or a promise of the final loss.

Validate the coefficient and inputs before arithmetic and require finite non-negative output
under the supported demographic configuration. Use multiplication/scaling that avoids avoidable
intermediate overflow; the shared cap must not conceal invalid or non-finite raw values.
The calculation consumes no RNG and adds only bounded transient work per band, not a saved hunger
meter or additional history. `HazardAlgorithm: "split-v1"` owns the direct response, its absence
of health gating/scaling, and `StarvationCoefficient` as registered in Appendix C; `FoodStorageAlgorithm`
still owns the deficit fraction. No starvation clock, eligibility flag, or extra save field is introduced.
After release, changes require the affected version update and explicit migration; no new
algorithm identifier is introduced in this unreleased design.

Formula fixtures cover fractions `0`, `0.10`, `0.20`, `0.50`, and `1`, the ratios above,
linear scaling with population/coefficient, both species, zero population, and invalid/non-finite
inputs or outputs. Lock Appendix C's `StarvationCoefficient` and the 100-person examples above.
With that default, assert raw loss on a first shortage at full health and on a small positive
shortfall, equal raw loss across valid health values at fixed
population/shortfall, and zero raw loss at poor health when reserves meet needs. There is no
health-dependent grace period or minimum rounding threshold.
Full-turn fixtures use the selected food/health inputs plus explicit synthetic boundary coefficients
and prove reserve protection, unchanged start-of-turn inputs after growth/deaths, exactly
one applied starvation term under the shared cap, non-negative survivors, and identical future
losses after save/load with no extra RNG use. Cover both a non-binding cap and a binding cap where
health changes chronic mortality but not raw starvation. Alternative coefficients used for
arithmetic/validation fixtures do not replace the approved initial default or close full-system
balancing. The food-limited growth fixtures below cover its separate effect on `P_grown`.

### Population-scaled food-storage capacity

`FoodStorageAlgorithm: "population-food-turns-v1"` sets the carrying-capacity model for the existing
band-local `StoredFood` reserve. `FoodStorageTurns` is one fixed, finite positive configuration
constant shared by both species and all bands. Its approved initial playtest value is **`3`**,
subject to tuning before release. With one FU required per person per turn:

```text
FoodStorageCapacity(b) = Population(b) * FoodStorageTurns   // FU
```

Use the band's current living population, not its assigned foragers, its previous maximum
population, or a population cached when the reserve was collected. With the selected initial
`FoodStorageTurns = 3`, populations 50 and 100 have capacities 150 and 300 FU respectively.
Zero population has zero capacity;
normal extinction cleanup still removes the band rather than creating an abandoned food cache.
The `uint32` population is intrinsically non-negative and exactly convertible to `float64`;
configuration validation must still reject an invalid `FoodStorageTurns` or unrepresentable derived
capacity.

Capacity is a ceiling, not food production. Growth increases carrying room without filling it;
population loss lowers carrying room without changing this turn's already-established food
requirement. Migration and workforce edits alone cannot change capacity. This rule adds no terrain,
species, or technology multiplier. The exact limit is fixed for a supported configuration, not
recalculated from the band's available harvest or current `StoredFood`.

A full reserve is not a guarantee of three fully fed no-harvest turns, because spoilage happens
before each meal. In an isolated fixture with population held at 100, no harvest or demographic
changes, and the selected 10% spoilage rate, 300 FU leaves 170 after the first meal and 53 after
the second. The third turn has only 47.7 FU after spoilage, leaving a 52.3-FU shortfall and zero
reserve. This is food accounting at fixed population, not a prediction of demographic outcomes.

A split divides `StoredFood` exactly `50/50`, conserving the total reserve rather than copying it.
Population divides as evenly as whole people allow; for an odd parent the source's one-person
remainder gives it three additional FU of capacity at the selected `FoodStorageTurns = 3`. Therefore
splitting a within-cap reserve leaves both descendants within their own caps. The source must satisfy the separate
`MinSplitSourcePopulation = 40` eligibility rule. Stored food remains an absolute FU amount, not a saved
percentage or a number of food-turns. Capacity is derived again from each resulting population.

**Enforcement: discard overflow once at turn end.** Phase 3 first resolves food consumption and
its demographic consequences. Do not discard food merely because the reserve temporarily exceeds
capacity during harvesting or population change. After all phase-5 acute losses, but before
zero-population cleanup and the completed-turn frame, apply this rule in stable band-ID order to
both species, including bands reduced to zero:

```text
capacity_final[b]       = P_final[b] * FoodStorageTurns
StoredFood_after_cap[b] = min(StoredFood_before_cap[b], capacity_final[b])
DiscardedFood[b]        = StoredFood_before_cap[b] - StoredFood_after_cap[b]
```

`StoredFood_before_cap` is exactly the phase-3 `FoodRemaining` balance, including unconsumed
actual harvest and reserves. It is not gross harvest, potential yield,
or food already eaten. Inputs and results must be finite and non-negative; the minimum operation
must not conceal invalid arithmetic. An above-cap intermediate is internal turn state, never a
planning snapshot or save. `StoredFood_after_cap` is the final reserve for publication. Its input
already reflects start-of-turn spoilage and current-turn consumption; neither loss is applied
again here or after the cap. The initial 10% spoilage rate and normalized food conversion are
selected. Automatic consumption and the spoilage/consumption/overflow ordering are fixed.

Discarded food is lost. It cannot refill flora/fauna, become a cache, transfer to another band,
or count as consumed food. The cap never reruns health, starvation, growth, or mortality, and
population growth increases room without creating food. A zero-population band retains zero FU
before normal removal. This finalization also runs on terminal turns before the result frame and
autosave can observe the state. It is bounded by `MaxBands`, consumes no RNG, and adds no saved
loss ledger, reserve pool, or event kind; `DiscardedFood` is transient accounting only.

At new game, after every accepted planning command, at completed-turn publication, and in saves,
each live band must satisfy finite `0 <= StoredFood <= FoodStorageCapacity(b)`. New-game reserve
configuration must satisfy the bound. Splits preserve it through the existing proportional,
conservative population/food arithmetic, without a second overflow-discard pass. Inspecting
capacity, building a frame, saving, or loading must not clamp or rescale the reserve. Load validation
rejects an above-cap reserve after any explicit supported-version migration; it must not repair
it by dropping food. The UI distinguishes stored FU from capacity FU and the configured food-turn
limit.

Fixtures cover the 50/100-population capacity example, growth creating room but not food, zero
population, invalid/non-finite inputs and products, and split conservation. With the selected initial
three-turn limit, a 100-person band holding 240 FU splits into two 50-person bands with
120 FU each and capacities of 150 FU each. Migration and assignment changes leave
capacity unchanged at fixed population.

For an ordering fixture, start with 100 people, 300 FU, no harvest, the selected initial three-turn
limit, and the selected 10% spoilage rate. Start-of-turn spoilage removes 30 FU, leaving 270;
consuming the required 100 FU leaves 170. If phase-3 and acute losses leave 50 people, the final
cap is 150 FU, so discard 20 and retain 150. The requirement remains 100 FU, not 50, and no earlier
capacity clamp may remove food before consumption. Accounting is `300 = 30 + 100 + 20 + 150` FU
for spoiled, consumed, discarded, and retained food. The rate and limit are approved initial
defaults; the population losses are fixture inputs, not a selected mortality formula. Test empty,
below-cap, exactly-at-cap,
and above-cap reserves; growth, phase-3 deaths, acute deaths, zero-population removal, and
terminal-turn finalization.
Assert `StoredFood_before_cap = StoredFood_after_cap + DiscardedFood` within conservative roundoff,
no food creation or tile-stock refund, and zero RNG use. Valid save/reload preserves absolute FU
and reconstructs identical capacity without a serialized cap or another discard; above-cap saves
are rejected without mutating the running world.

### Fixed-percentage stored-food spoilage

Each turn starts by removing a fixed fraction of carried-over food. `FoodSpoilageRate` is one
constant shared by both species and all bands, with finite `0 < FoodSpoilageRate < 1`. Its approved
initial playtest value is **`0.10`**, or 10% per game turn, subject to tuning before release.
It is a proportion of the FU reserve, not a fixed number of FU per person, a
workforce cost, or a probability of losing the whole reserve. This decision adds no food-age
buckets, food-type inventories, or terrain/technology modifiers.

At the single start-of-turn checkpoint, `StoredFood_before_spoilage` is the finite non-negative
FU reserve after all accepted planning, including the atomic archaic batch, and before any
current-turn harvest or consumption:

```text
SpoiledFood[b]              = StoredFood_before_spoilage[b] * FoodSpoilageRate
StoredFood_after_spoilage[b] = StoredFood_before_spoilage[b] - SpoiledFood[b]
```

Validate the input and rate before arithmetic. Both outputs stay finite and in
`[0, StoredFood_before_spoilage]`, with retained plus spoiled food conserving the input within
conservative roundoff. Fractional FU remain fractional; do not round losses to whole food items.
With the selected initial 10% rate, 100 FU becomes 90 FU. A second isolated application with no
other food changes gives 81 FU, not 80: the fraction applies to the current reserve each time.
These are isolated spoilage applications, not complete advancing turns with consumption.

Spoilage and overflow are distinct losses. Each operation reads the reserve left at its own
checkpoint; neither subtracts the other's loss again or uses gross harvest or already eaten food.
Spoiled food is lost, not consumed, transferred, cached, or refunded to tile stocks. Spoilage adds
no direct health or mortality term beyond the consequences of the food remaining available.

Only simulation turn advancement applies spoilage, once per band reserve at this start-of-turn
checkpoint, including on terminal turns. Planning commands, splits, movement intent, rendering, pausing,
saving, and loading do not age food or apply another loss. There is no wall-clock or offline decay.
The operation is deterministic, consumes no RNG, and is bounded by `MaxBands`; its temporary
amounts add no persistent ledger, timestamps, inventory, or per-food age state. The existing
`FoodStorageAlgorithm` versions the formula, selected checkpoint, and approved initial rate.

**Timing: the beginning of phase 1, before `climate.Advance`.** After `AdvanceTurn` has passed its
terminal/turn guards and the atomic archaic planning batch has succeeded, apply spoilage in stable
band-ID order to both species. Rejected turn requests and failed planning batches leave food
unchanged. This is the first operation of the existing five-phase turn, not a sixth phase or a
planning command; no intermediate frame or save can observe its result.

The reduced reserve is available for phase-3 consumption. Spoilage can therefore contribute to
this turn's food deficit, but adds no separate immediate damage. Food gathered in phase 3 is not
part of the spoilage input. Any food left after consumption and phase-5 overflow becomes the next
turn's carried-over reserve, which faces spoilage only when that next turn advances. A zero
starting reserve means zero spoilage even when this turn later gathers food. Initial scenario
food follows the same rule on the first advancing turn, not during new-game creation. Planning
splits divide food normally, and each resulting reserve is processed once on the next `AdvanceTurn`.

Formula fixtures cover zero reserve, the 100 → 90 → 81 example, proportional loss for different
reserve sizes, finite tiny/large inputs, and rejection of negative/non-finite reserves or rates
outside `(0, 1)`. Equivalent-checkpoint splits conserve aggregate retained and spoiled FU within
roundoff; splitting itself causes no spoilage. Valid reloads preserve the reserve without decay and
use the same configured rate at the next turn's start. The ordering fixture above covers spoilage,
consumption, population loss, and overflow together. A fresh-harvest fixture starts with 50 people
and 100 stored FU, using the selected initial 10% rate and three-turn limit with population held at 50:
spoil 10 FU first, then gather 50 actual FU and consume 50 FU, leaving 90 FU, not 85. With no
intervening food changes, the next turn starts by spoiling 9 FU from that 90, before any new
harvest or consumption. A shortage
fixture with 50 people, 50 stored FU, and no harvest leaves 45 FU available and a 5-FU deficit;
it does not select the resulting mortality. These examples supply actual harvest in FU and use
the automatic-consumption rule above after the selected stock conversion.

Ordering fixtures also cover zero starting food with positive harvest, new-game and split
reserves, canceled migration without another loss, terminal-turn finalization, and rejected
post-terminal `AdvanceTurn` calls. Repeated saves/reloads between turns must reproduce the same next
start-of-turn loss, never skip it or apply it twice. Lock the initial 10% default and the three-turn
capacity in configuration fixtures, including the fixed-population no-harvest sequence above.

### Regional fauna profiles

`FaunaProfileAlgorithm: "region-biome-v1"` defines a total, deterministic lookup
`FaunaProfile(Region(i), Biome(i))` for every valid land region/biome pair. `Region(i)` is fixed
geography; `Biome(i)` is the current climate-derived biome. Water and uninhabitable tiles use an
explicit empty profile. The mapping is authored configuration, not sampled from the seed, inferred
from resident hominin species, or changed by a band's technology. Different regions with the same
biome may have different profiles; sharing a biome does not imply identical prey.

Each profile contains a fixed-size relative-prey-weight vector over the closed `FaunaGroup` order
`SmallGame`, `MediumGame`, `LargeGame`, `Megafauna`, `InshoreAquatic`, and `PelagicAquatic`, plus
environmental support flags for Hunting and megafauna tracking and the inputs to the corresponding
yield, fauna-related work-risk, and acquired-technology modifier tables. `InshoreAquatic` covers
freshwater, shoreline, shellfish, and nearshore opportunities; `PelagicAquatic` covers offshore
fish. The groups describe broad prey opportunities rather than separately simulated animal species. Weights
must be finite and non-negative with a finite positive total for a non-empty profile; the empty
profile has all-zero weights and supports neither role. Any supported role must have a positive
weight in a compatible prey group. `FaunaGroupCount = 6`; the public/domain enums and their
exhaustive boundary mapping close in step 2.

The selected regional fauna-profile configuration uses eight reusable archetypes. Weight tuples are in exact
`FaunaGroup` order and sum to `1.0`. Ordinary Hunting is supported by all eight; megafauna tracking
is supported exactly where the final column is true.

| ID   | Profile name      |  Small | Medium |  Large |   Mega | Inshore | Pelagic | Mega supported |
| ---- | ----------------- | -----: | -----: | -----: | -----: | ------: | ------: | -------------- |
| `OM` | Open Mixed        | `0.15` | `0.25` | `0.30` | `0.20` |  `0.10` |  `0.00` | Yes            |
| `WR` | Woodland Riverine | `0.20` | `0.20` | `0.15` | `0.10` |  `0.30` |  `0.05` | Yes            |
| `CM` | Coastal Mixed     | `0.15` | `0.15` | `0.10` | `0.05` |  `0.35` |  `0.20` | Yes            |
| `AS` | Arid Small Game   | `0.40` | `0.30` | `0.15` | `0.00` |  `0.15` |  `0.00` | No             |
| `HM` | Highland Mixed    | `0.25` | `0.30` | `0.20` | `0.15` |  `0.10` |  `0.00` | Yes            |
| `CS` | Cold Steppe       | `0.10` | `0.20` | `0.30` | `0.35` |  `0.05` |  `0.00` | Yes            |
| `TI` | Tropical Island   | `0.20` | `0.15` | `0.10` | `0.10` |  `0.25` |  `0.20` | Yes            |
| `BC` | Beringian Coast   | `0.10` | `0.20` | `0.25` | `0.30` |  `0.10` |  `0.05` | Yes            |

All `13 * 6 = 78` region/biome pairs resolve to an archetype. Authoring them as 78 explicit cells
would repeat the six-entry baseline row unchanged for nine regions and restate most of it for the
four regions whose five exception cells carry the actual regional information, so the mapping is
authored the way `ArchaicAssignmentPreset` is:
one baseline row plus its named exceptions, materialized and validated as all 78 results.

The **baseline archetype row**, in the stable biome order used elsewhere:

| Savanna | Riverine | Coastal | Desert | Highlands | Tundra |
| ------- | -------- | ------- | ------ | --------- | ------ |
| `OM`    | `WR`     | `CM`    | `AS`   | `HM`      | `CS`   |

The complete set of **regional exceptions** — five cells across four regions:

| Region         | Biome             | Archetype | Why it differs                                    |
| -------------- | ----------------- | --------- | ------------------------------------------------- |
| Arabia         | Savanna           | `AS`      | its savanna is arid scrub, not East African grass |
| Southeast Asia | Coastal Shrubland | `TI`      | island and reef coasts, not continental shore     |
| Sahul          | Coastal Shrubland | `TI`      | same island/reef character                        |
| Beringia       | Coastal Shrubland | `BC`      | cold coast with megafauna, not temperate shore    |
| Beringia       | Glacial Tundra    | `BC`      | mammoth-steppe coast rather than interior steppe  |

Every other region takes the baseline row unchanged. This is **not** a fallback for a missing key:
every one of the 78 pairs is defined, an unknown region or biome remains an error, and validation
still materializes and checks all 78 results plus the reachability of all eight archetypes. What
changes is that a balance pass reads and edits five decisions instead of scanning 78 cells for the
nine rows that are byte-identical, and a regional distinction can no longer be lost in the noise of
a repeated default.

Region also varies the rate directly, through one reference hunting index per region. It applies to
every current biome in that region; biome changes alter the six prey weights and accessible sum
through the archetype mapping, so the evaluated rate is still region-and-biome-specific. This index,
not the archetype map, is where most regional variation lives — which is why the map needs so few
exceptions.

| Region             | Hunting index |
| ------------------ | ------------: |
| East Africa        |        `1.00` |
| Rest of Africa     |        `1.00` |
| Arabia             |        `0.65` |
| Levant             |        `0.95` |
| Frangistan         |        `1.15` |
| Central Asia       |        `0.80` |
| South Asia         |        `1.05` |
| Southeast Asia     |        `0.85` |
| East Asia          |        `0.95` |
| Yellow River Basin |        `1.00` |
| Sahul              |        `0.90` |
| Siberia            |        `1.25` |
| Beringia           |        `1.15` |

Configuration requires exactly eight unique archetypes, a complete six-entry baseline row, an
exception list whose every entry names a valid region/biome pair and changes that pair's resolved
archetype, 78 materialized valid mappings that reach all eight archetypes, 13 finite
positive regional indices, exact six-weight vectors, and agreement between megafauna support and a
positive `Megafauna` weight. A duplicate or no-op exception is a configuration error, so the
exception list cannot quietly accumulate rows that restate the baseline. Unknown or missing keys are errors; water/uninhabitable tiles alone use
the explicit empty profile. Ordinary hunting and megafauna tracking use the selected linear
production and technology tables below. Work risk retains the share-based rule below, with its
activation/event-kind mapping closed in step 5 and numerical coefficients selected separately.
These aggregate archetypes are gameplay opportunity mixes, not historical animal censuses.

The profile is shared environmental information. Both hominin species use it under the same
rules; their acquired technologies and worker allocations determine what each band can exploit.
A technology can improve access or efficiency for applicable prey without creating animals or
changing the environmental mix. Apply each collection/conversion/risk modifier only in its
specified channel, not again through a generic capacity bonus. These profile effects must affect
the eventual hunting/megafauna calculations, not merely the names shown in the inspector.

The Hunting workforce role includes fishing; its player-facing label is **Hunting & fishing** while
the stable enum remains `Hunting`. Where a profile has positive `InshoreAquatic` weight, baseline
inshore procurement is available even with an empty acquired-technology set. `HaftedTools` improves
authored large-game and inshore spear/harpoon rates; `CordageAndNets` improves inshore collection
and unlocks pelagic collection; `Trapping` improves small/medium game and authored inshore trap/weir
collection; and `CoastalNavigation` further improves pelagic collection while retaining its
Wallacea-passage role. The exact hunting collection modifiers appear in the linear hunting contract below;
configuration may not move an effect into FU conversion or grant pelagic access without
`CordageAndNets`. These are access/rate channels, not
separate actions, gear inventories, success rolls, or durable facilities.

There is still exactly one fauna stock per tile. Prey weights are opportunities, not fractions
with independently conserved biomass. All ordinary-hunting and megafauna demands from all bands
compete in the same proportional allocation and deplete that one stock exactly once. A missing
role opportunity produces zero demand for that role; positive total fauna alone cannot enable
megafauna tracking. Overhunting ordinary prey also reduces the shared stock available to megafauna
tracking, and vice versa. This slice deliberately cannot model selective species depletion,
extinction, prey replacement, or independently regenerating animal populations.

Changing a profile never refills, duplicates, or independently clears the stock. The existing
biome/season/degradation cap and toward-cap regeneration still govern total fauna; the profile is
not another multiplier on resource capacity, regeneration, `BaselineK`, or `T_tech`. Flora,
water, and the selected foraging rule are unchanged. Climate can select a different fauna profile
by changing the biome, while the geographic region remains fixed. Seasons affect total availability
through the existing stock rules; they do not independently resample the prey mix.

Phase 3 reads the origin's profile after phase 1's biome update to derive animal-food demands and
capture any fauna-dependent, unmitigated work-risk inputs before demographics and migration. Do not
reduce these inputs by technology and then apply the same reduction again in phase 5; technology
follows the existing probability/severity effect-channel contract. Such work risk
joins the existing uncovered hazard components, not camp protection, new event kinds, or extra
RNG draws. Phase 5 uses those captured origin-work inputs alongside the existing final-tile acute
profile; it must not reinterpret work already performed using the destination's fauna. Newly
acquired technology cannot retroactively change that work. Migration candidates instead use each
candidate destination's current fauna profile for their fauna-value preview, counting the shared
stock once rather than once per prey group or workforce role.

The lookup table has exactly `RegionCount * BiomeCount = 78` valid land entries, each with exactly
`FaunaGroupCount = 6` weights, plus one explicit empty profile for water/uninhabitable tiles. Frames
copy one fixed-size `FaunaSummary` per tile, at most 6,144 summaries and
`6_144 * FaunaGroupCount` weights, without aliases to domain configuration. Fauna-dependent per-band
work inputs remain bounded by `MaxBands = 256` and are discarded after acute resolution. Profile lookup,
snapshot creation, and previewing consume no RNG. Save/load derives the profile and summary from
the supported profile/geography/climate versions, not a persisted prey mix or animal ledger.

### Distinct workforce roles

Every band has the same closed `Assignment` enum: `Foraging`, `Hunting`, `Toolcraft`,
`MegafaunaTracking`, and `Shelter`, with `AssignmentCount = 5`. A band carries one fixed five-entry
`AllocationBP[AssignmentCount]` vector under
`AssignmentAlgorithm: "proportional-basis-points-v1"`. Each unsigned entry is in `[0, 10_000]`, and
all five sum exactly to `AllocationBasisPoints = 10_000`; the vector therefore represents 100% of
the band with 0.01-percentage-point resolution and has no unassigned-worker state. Fixed-point
shares avoid floating-point normalization drift in commands, JSON, state hashes, and reloads.
`SetAssignment` carries a complete candidate vector and replaces the old vector atomically only
after the band, authority, entry bounds, and exact-sum invariant pass. Validation accumulates the
five `uint16` entries in a `uint32`; it never allows a narrow sum to wrap into an apparently valid
total.

The sapiens inspector uses a **draft-and-apply** editor. `game_scene` owns at most one fixed
`AssignmentDraftBP[AssignmentCount]` vector, associated with the selected sapiens band. A draft is
dirty exactly when it differs from that band's last accepted allocation. When there is no dirty
draft, selecting a sapiens band initializes the vector and its accepted baseline from that band's
current frame. Each slider edits one draft entry independently, so the draft sum may temporarily
be below or above 10,000. The inspector shows the allocated total, the
remaining or excess amount, and draft percentages and derived people. Domain-derived outcome fields,
including `OriginalResearchGainPreview`, continue to describe the accepted frame until Apply.

Apply is enabled only when every entry is in bounds, a `uint32` sum is exactly 10,000, the draft
differs from its accepted baseline, and no Apply result or load is pending. Slider edits emit no
`ui.Action`, consume no RNG, increment no `WorldRevision`, build no frame, and cannot enter a save.
Apply emits exactly one `SetAssignment`
containing the whole vector. The editor is locked until `pkg/app` returns that command's result;
submission alone does not clear dirty state or permit duplicate Apply actions. `GameService.Apply`
and `World.PlanPlayer` remain authoritative: success replaces the frame and resets the draft and baseline from that frame;
rejection leaves the world and accepted frame unchanged, retains the draft, and shows the error
inline. The draft is UI-local, bounded to five
entries, excluded from `SaveState`, autosaves, and canonical campaign-state hashes, and never exists for an
archaic selection. Saving with a dirty draft captures the last accepted allocation.

**Dirty-draft exit policy: block and resolve inline.** Changing selection, closing the inspector,
actually loading/replacing the world, leaving gameplay for the title, or ending the turn is blocked
while the draft is dirty. Every mouse, keyboard, and menu entry point uses the same UI guard before
changing selection/scenes or emitting a protected action. The selected band, draft, world, and
storage queue remain unchanged by the blocked attempt. A persistent inline message says “Apply or
discard workforce changes”; it is not a modal or a two-second toast.

Apply uses the exact-vector validation above and stays disabled for an invalid total. Discard is
available for any dirty draft when the editor is not awaiting an Apply result or load: it restores
the selected band's current accepted vector, clears the inline message/error, and emits no action,
snapshot, revision, or RNG draw. Editing the vector back to the accepted baseline also makes it
clean and clears the message/error. A successful Apply clears the message; a failed Apply keeps the
draft dirty and shows the error. Neither resolution automatically performs the previously blocked action: the player must
retry it. Protected input received while dirty, even in the same input batch as Apply, is dropped
as intent rather than queued for later replay. Repeated attempts reuse one inline message, so the
guard adds no action backlog or durable state.

Camera movement, hover, and quick-save remain available. Pause/settings/save-browser overlays may
retain the gameplay scene and its draft without discarding it; actual load/title actions are still
guarded, with the same inline Apply/Discard controls reachable in the active overlay and a way back
to the editor for invalid drafts. An unrelated planning-frame refresh or storage completion must
not reset a dirty draft: it updates accepted-state displays and derives draft people from the
current selected population without changing the entered shares. Only a resolved context change
may initialize a different band's draft. Native window/tab closure, browser refresh, and crashes
can still lose this non-persisted UI state; this policy adds no shutdown interception or recovery.

For role `a` and the finite non-negative start-of-turn population `P_start`:

```text
assignment_share(b, a) = AllocationBP[b, a] / AllocationBasisPoints
workers(b, a)          = P_start[b] * assignment_share(b, a)
```

The implementation converts the basis-point entry to a ratio before multiplying, so the ratio is
always in `[0, 1]` and the derived worker value is finite and in `[0, P_start]`. The mathematical
sum across roles is exactly `P_start`; floating-point conservation assertions allow at most eight
ULPs of `P_start` and never write a rounding remainder back into the saved vector. The same formula
supplies the inspector's percentage and derived-people display. The saved shares, not a stale worker
count, are authoritative.

Population growth, phase-3 mortality, and acute events change population but do not rewrite the
proportion vector. The next planning frame therefore derives new worker counts automatically from
the survivor population. Whole-band migration also preserves the vector. `SplitBand` copies the
parent's complete vector to both results; because it conserves population, the descendants'
mathematical per-role worker totals equal the parent's pre-split totals. Both research previews are
recomputed from their smaller derived toolcraft workforces. New-game scenario allocations and every
`ArchaicAssignmentPreset` are closed basis-point vectors that satisfy the same exact-sum rule.

At turn resolution, the accepted start-of-turn allocation routes work to five distinct consumers:

- **Foraging** creates flora demand equal to its worker count times the origin biome's
  acquired-technology-adjusted collection rate. Actual collection is limited by the shared-stock
  allocator, with proportional sharing when flora is insufficient.
- **Hunting** creates ordinary-fauna demand equal to its worker count times the origin's
  region × biome-profile-and-acquired-technology-adjusted hunting rate. Actual catch shares the
  one fauna stock proportionally with every band's hunting and megafauna demand.
- **Megafauna tracking** creates fauna demand equal to its worker count times a higher
  profile-and-acquired-technology-adjusted rate than ordinary terrestrial hunting, only where the
  origin's profile and the band's technology allow it. It shares hunting's one fauna stock and carries
  added field-risk pressure. Greater potential extraction does not guarantee more food under
  scarcity or apply extra stock damage. Hunting and megafauna work risk scales with their workforce
  shares, not worker counts. The selected hunting and megafauna rate/access tables are implemented with the roles;
  work-risk activation, event-kind mapping, and numerical coefficients are selected in the
  share-based risk contract.
- **Shelter** uses its share of the band's workforce to create turn-local protective effort that
  covers environmental exposure, camp security against predation, and modest disease protection
  through camp hygiene. The relevant tile's natural shelter improves only exposure protection;
  security and hygiene use the ordinary workforce curve. All effects have diminishing returns and
  stay below full immunity. Acute protection reduces covered incident probability, not severity;
  seasonal/chronic protection reduces the covered phase-3 losses. This remains one assignment and
  creates no persistent camp or infection state. Applicable technology and camp protection each
  reduce the remaining risk; the shelter constants, acute class partitions, seasonal/chronic risk
  profiles, and technology-mitigation effects are selected initial values.
- **Toolcraft** produces only the bounded original research gain defined by
  `saturating-toolcraft-v1` below. It does not create a generic tool inventory.

Water remains a population-wide survival requirement and proportional shared-stock demand; it is
not owned by one role and does not create a sixth assignment. Assignment evaluation itself is
deterministic and consumes no RNG. It introduces no new resource stock, generic inventory, durable
building collection, or work-history log. Role evaluation is bounded by
`MaxBands * AssignmentCount = 1_280` entries per turn. Allocation entries require no floating-point
save validation, demographic rescaling writes, rounding/remainder history, or idle-worker state.

### Linear foraging and shared flora

`ForagingAlgorithm: "linear-shared-flora-v1"` converts the accepted start-of-turn forager count
into potential plant collection. `ForagingRate(b, i)` is a finite non-negative number of flora-stock
units per worker per turn, derived from the closed biome-rate and acquired-technology modifier
tables. It reads tile `i`'s current biome after the phase-1 climate update and only technology the
band already knows. It is independent of band size, assigned worker count, and other bands'
demands; unavailable collection can have rate zero. Collection modifiers are applied once in this
rate, not again to the allocated harvest, and the generic capacity multiplier `T_tech` is not an
additional collection bonus. The approved initial rate is:

```text
ForagingRate(b, i) = 2.5 * BiomeForagingIndex[current biome]
                     * PlantKnowledgeForagingMultiplier(b)
```

| Current biome         | `BiomeForagingIndex` | Base flora units per worker |
| --------------------- | -------------------: | --------------------------: |
| Savanna               |               `1.00` |                      `2.50` |
| Riverine Woodland     |               `1.20` |                      `3.00` |
| Coastal Shrubland     |               `0.75` |                     `1.875` |
| Semi-Arid Desert      |               `0.40` |                      `1.00` |
| Mountainous Highlands |               `0.60` |                      `1.50` |
| Glacial Tundra        |               `0.30` |                      `0.75` |

`PlantKnowledgeForagingMultiplier` is `1.30` when the band has acquired `PlantKnowledge` and `1.0`
otherwise. No other v1 technology modifies foraging. Partial research and technology acquired at
turn end do not apply to that turn's collection. This table applies identically to both species;
seasonal scarcity remains entirely in the available flora stock and does not multiply the rate a
second time.

For each origin tile `i`, let `F_i` be its finite non-negative flora stock after phase-2 clamping
and regeneration. All resident sapiens and archaic bands participate in one allocation:

```text
flora_demand[b] = workers(b, Foraging) * ForagingRate(b, i)
D_i            = sum_b(flora_demand[b])

flora_allocated[b] = 0                               if D_i == 0 or F_i == 0
                     flora_demand[b]                 if D_i <= F_i
                     F_i * flora_demand[b] / D_i     otherwise
```

These are mathematical demands, not promises that every intermediate fits in a `float64`.
The allocator must use overflow-safe product, sum, and multiply/divide arithmetic, retaining scaled
demand magnitudes through allocation when necessary. Finite workers and rates must not produce
an infinite/NaN allocation merely because their naive product or the demand total overflows.
Do not cap each band's demand at `F_i` before computing proportions: that would give equal shares
to unequal large demands. For example, demands of 200 and 100 against 60 flora allocate 40 and 20,
not 30 each. Apply the existing allocator's conservative roundoff handling so every allocation is
non-negative, no greater than its potential demand, and the summed extraction never exceeds `F_i`.
Unused flora remains on the tile; it is not forced into bands with no collection demand.

At fixed biome and technology, doubling foragers doubles potential collection, not necessarily
the actual harvest. There is no additional saturating workforce curve or random foraging roll.
Zero foragers or a zero rate creates zero flora demand even when the band needs food. Seasonal
availability and degradation already affect `F_i` through the resource-cap/regeneration rules;
they are not additional collection-rate penalties. Other species, additional bands, and their
allocation changes affect the shared stock and allocation, not this band's per-worker rate.

Resolve collection in phase 3 on the origin, before growth/mortality and migration. Subtract the
sum of allocated flora exactly once through the shared extraction path. Only the allocated
harvest enters the existing food-consumption pipeline; unmet potential demand is never food or
`StoredFood`. `normalized-source-v1` converts each allocated flora unit to `1.0 FU` before an
applicable source-specific heritable modifier; the automatic-consumption rule
combines converted harvest with post-spoilage reserves and carries the remainder to the final cap.
Starvation applies the squared response above directly, with initial `StarvationCoefficient = 0.10`.
A move does not collect again at the destination, and knowledge acquired after acute events cannot
retroactively improve this turn's collection.

The rule is identical for both species. A split preserves total potential demand when its copied
shares, inherited knowledge, and both destinations' rate inputs are equivalent; different biomes
can legitimately change that total. Population changes affect next turn's worker counts without
rewriting the accepted shares. Demand and allocation are bounded transient per-band values, not
new save fields or inventories: at most 256 of each, using the existing tile buckets and stable
allocation ordering. Their computation consumes no RNG. Save/reload reconstructs them from shares,
population, acquired technology, geography/climate, and resource stocks under the same versioned
foraging tables; changing the formula or tables after release requires an explicit version migration.

### Linear hunting and shared fauna

`HuntingAlgorithm: "linear-shared-fauna-v1"` converts accepted start-of-turn hunters into potential
terrestrial and aquatic catch. For each non-megafauna group `g`, `GroupHuntingRate(b, i, g)` is a
finite non-negative rate in fauna-stock units per hunter per turn, derived from the origin's current
`region-biome-v1` profile weight, the group's structural access rule, and the band's acquired
technologies. `HuntingRate(b, i)` is their stable-enum-order sum.
It reads the biome after phase 1's climate update, not the planning period's earlier biome. The
rate is zero when the profile does not support ordinary hunting or the band cannot exploit any
of its terrestrial or aquatic prey. The `Megafauna` group contributes only through the separate
megafauna role below. Rate-table inputs and evaluated rates must be finite and non-negative;
the selected hunting reference rate is **`HuntingReferenceRate = 2.5` fauna units per hunter per game
turn**. For each non-megafauna group:

```text
GroupHuntingRate(b, i, g) = 0                              if GroupAccessible(b, i, g) is false
  HuntingReferenceRate * RegionHuntingIndex[Region(i)]
  * FaunaWeight[FaunaProfile(Region(i), Biome(i)), g]
  * GroupTechnologyMultiplier(b, g)                        otherwise
```

`SmallGame`, `MediumGame`, `LargeGame`, and `InshoreAquatic` are accessible at baseline when their
profile weight is positive. `PelagicAquatic` additionally requires acquired `CordageAndNets`;
because `CoastalNavigation` depends on that technology, it cannot bypass the gate. The selected
collection multipliers are:

| Technology          | Prey group       | Yield multiplier or access effect |
| ------------------- | ---------------- | --------------------------------: |
| `HaftedTools`       | `LargeGame`      |                            `1.30` |
| `HaftedTools`       | `InshoreAquatic` |                            `1.15` |
| `HaftedTools`       | `Megafauna`      |                            `1.30` |
| `CordageAndNets`    | `InshoreAquatic` |                            `1.25` |
| `CordageAndNets`    | `PelagicAquatic` |          unlock; multiplier `1.0` |
| `Trapping`          | `SmallGame`      |                            `1.35` |
| `Trapping`          | `MediumGame`     |                            `1.20` |
| `Trapping`          | `InshoreAquatic` |                            `1.20` |
| `CoastalNavigation` | `PelagicAquatic` |                            `1.40` |

For a group with multiple applicable acquired technologies, multiply the listed yield factors once
in stable technology-enum order. An empty product is `1.0`; unlisted technology/group pairs have no
collection effect. These are potential-yield multipliers, not additional stock, FU conversion,
capacity, or hazard mitigation. Partial research and technologies acquired at turn end do not
apply to the current collection pass.

At fixed profile and acquired technology, the rate is independent of band size, hunter count,
other bands, and current stock. Doubling hunters doubles potential catch; zero hunters or a zero
rate produces zero ordinary-hunting demand. There is no extra workforce-saturation curve or random
hunting-success/yield roll. Profile and technology collection modifiers enter the rate once,
not again after allocation; `T_tech` is not an extra catch multiplier. Seasonal availability and
degradation already affect the shared stock through the existing cap/regeneration rules. The
separately specified hunting work risk and existing acute-hazard draws remain in force.

Define the two Hunting components and catch share before shared-stock allocation:

```text
OrdinaryTerrestrialRate(b, i) = GroupHuntingRate(b, i, SmallGame)
                                + GroupHuntingRate(b, i, MediumGame)
                                + GroupHuntingRate(b, i, LargeGame)
AquaticRate(b, i) = GroupHuntingRate(b, i, InshoreAquatic)
                   + GroupHuntingRate(b, i, PelagicAquatic)
HuntingRate(b, i) = OrdinaryTerrestrialRate(b, i) + AquaticRate(b, i)
AquaticShare(b, i) = 0                                      if HuntingRate(b, i) == 0
                      AquaticRate(b, i) / HuntingRate(b, i) otherwise
```

The share is finite in `[0, 1]`. It describes expected catch composition at the current profile and
technology, not another extraction demand or an independently conserved animal population.

For origin tile `i`, let `A_i` be its finite non-negative fauna stock after phase-2 clamping and
regeneration. The two fauna-consuming roles, in stable assignment-enum order, are `Hunting` and
`MegafaunaTracking`. The latter supplies the linear potential demand defined below under
`linear-megafauna-v1`, or zero when unavailable, to the same allocator:

```text
fauna_demand[b, Hunting]          = workers(b, Hunting) * HuntingRate(b, i)
fauna_demand[b, MegafaunaTracking] = megafauna_demand[b]
D_i = sum over resident bands b and both fauna roles a of fauna_demand[b, a]

fauna_allocated[b, a] = 0                              if D_i == 0 or A_i == 0
                        fauna_demand[b, a]            if D_i <= A_i
                        A_i * fauna_demand[b, a] / D_i otherwise
```

This is one proportional allocation across both species and roles, not a hunting pass followed
by a megafauna pass and not two independent allocations of the same stock. For example, one band
with hunting demand 100 and megafauna demand 100, and another with hunting demand 100, share
60 fauna as 20 per nonzero role demand: 40 for the first band and 20 for the second. These are
allocator fixture demands, not a choice of per-worker rates or balance values.

As with foraging, these are mathematical demands: finite workers/rates can produce products or
totals larger than `float64` can represent. Retain scaled magnitudes and use overflow-safe
product/sum/multiply-divide arithmetic; never cap a role's or band's demand at `A_i` before
computing proportions. Conservative roundoff handling keeps every allocation non-negative and no
greater than its demand, with the summed extraction at most `A_i`. Unused stock remains on the
tile; zero-demand roles receive none. This does not create separately depleted prey stocks.

Resolve the allocation in phase 3 at the origin, before demographics and migration. Subtract
the total allocated fauna once; credit each role's actual allocation to the food pipeline once,
not again through its band's aggregate fauna total. Split each Hunting allocation without another
stock read or rounding remainder:

```text
aquatic_allocated[b] = fauna_allocated[b, Hunting] * AquaticShare(b, i)
ordinary_allocated[b] = fauna_allocated[b, Hunting] - aquatic_allocated[b]
```

Evaluate the product with the domain's explicit deterministic-rounding rule; derive the ordinary
remainder by subtraction so the two values sum exactly to the Hunting allocation. Credit them to
`Aquatic` and `Animal` respectively; credit the tracking allocation to `Animal` as well. For
example, an aquatic rate of `3` within a total Hunting rate of `5` gives `AquaticShare = 0.6`, so
an allocated Hunting catch of `20` credits `12` to `Aquatic` and `8` to `Animal` before modifiers,
their exact sum still `20`.
Potential catch and unfilled demand are not usable food units or `StoredFood`.
`normalized-source-v1` converts each allocated unit at the common `1.0 FU` baseline before applicable
source-specific heritable modifiers; actual converted harvest feeds
the automatic per-band consumption rule, with the remainder subject to the final storage cap.
Neither migration nor newly completed research grants a second or retroactively improved catch.
Fauna-dependent work-risk inputs retain the origin timing and effect-channel contract above.

Both species use the same rule. Population changes preserve saved workforce shares and change
next turn's hunter count. Splits preserve total potential ordinary catch when copied shares,
inherited technology, and profile/rate inputs are equivalent; actual catch may differ when the
descendants encounter different stocks or competitors. At most `2 * MaxBands = 512` fauna-role
demands and 512 allocations are transient, with at most 256 derived band totals. They add no RNG
draw, inventory, or persisted field. Save/load reproduces them from accepted shares, population,
position, acquired technology, resource stocks, and supported hunting/megafauna/profile configuration.

### Linear megafauna tracking

`MegafaunaAlgorithm: "linear-megafauna-v1"` uses the same workforce-to-potential-catch shape as
ordinary terrestrial hunting, with a higher rate and the already-required added field risk. `MegafaunaRate(b, i)`
is measured in fauna-stock units per tracking worker per turn. It reads the origin's current
region × biome profile after phase 1 and only technology already acquired by the band:

```text
megafauna_demand[b] = workers(b, MegafaunaTracking) * MegafaunaRate(b, i)
```

The rate is zero if the profile does not support megafauna tracking or the band cannot exploit any
of its megafauna prey. Otherwise it is finite and positive and must exceed
`OrdinaryTerrestrialRate(b, i)` for that same profile and acquired technology. Rich aquatic
opportunity is intentionally excluded from this higher-risk/higher-terrestrial-yield comparison.
The comparison is per worker, not between the bands' total catches, and includes any applicable
ordinary-terrestrial technology benefit. Require finite non-negative
rate inputs/results and validate the higher-rate ordering across the closed profile/technology combinations; reject invalid
configuration rather than silently adjusting a rate. The selected megafauna rate is:

```text
MegafaunaRate(b, i) = 0
    if FaunaProfile(Region(i), Biome(i)).MegafaunaSupported is false
MegafaunaRate(b, i) = 1.75 * OrdinaryTerrestrialRate(b, i)
                      * MegafaunaTechnologyMultiplier(b)
    otherwise
```

Every selected megafauna-supporting archetype has positive ordinary-terrestrial and `Megafauna`
weights, so the non-zero branch is valid. `MegafaunaTechnologyMultiplier` is `1.30` with acquired
`HaftedTools` and `1.0` otherwise; no other v1 technology changes this rate. The archetype's
`Megafauna` weight controls environmental support and the player-facing prey emphasis, while the
fixed `1.75` higher-yield ratio supplies the tracking role's aggregate potential catch without
pretending to count individual animals. Baseline megafauna tracking needs no acquired technology
where the profile supports it. Partial research and end-of-turn acquisition do not apply to this
turn's rate.

At fixed profile and acquired technology, `MegafaunaRate` is independent of population, assigned
worker count, competitors, and remaining stock. Doubling tracking workers doubles potential
demand; zero workers or a zero rate gives zero demand. There is no additional saturation curve,
minimum-party threshold, encounter/success draw, or discrete animal kill to round to. Profile and
technology collection modifiers enter the rate once; neither `T_tech` nor the allocation stage
adds another yield bonus. Seasonal availability and degradation continue to act through stock
caps and regeneration, not an extra rate multiplier.

Feed this full potential demand into the single `linear-shared-fauna-v1` allocation alongside all
ordinary-hunting demands, across both species. The same scaled-magnitude arithmetic, conservative
roundoff, and 512-role-entry bounds apply; no second allocator, stock, or demand ledger is added.
With illustrative rates of 1 for hunting and 2 for megafauna tracking, ten workers in each role
create demands of 10 and 20. With 30 or more fauna they collect 10 and 20; with only 15 they collect
5 and 10. These are test values, not selected balance constants. Higher potential can exhaust the
stock faster but cannot extract beyond it or justify a second depletion penalty. Only actual
allocated amounts enter the selected normalized food conversion once; no extra megafauna
food-conversion multiplier is implied.

Collection still resolves at the origin in phase 3 before demographics and migration. Migration
does not grant a destination catch, and research completed after acute events cannot improve work
already performed. Added megafauna work risk remains captured from the origin and enters the
existing uncovered acute components; camp work cannot mitigate it. Its workforce scaling follows
`linear-share-work-risk-v1` below with the selected activation rule, event-kind mapping, and
coefficient vectors.
This production rule adds no hazard event kind
or RNG draw and does not promise that every attempt causes an incident.

Both species use the same production and access rules. Population changes preserve the saved
shares; splits preserve total potential at equivalent profile/technology rates, not necessarily
actual catch when stocks or competition differ. Rates and demands are derived from existing
planning state, not saved as an expedition, carcass inventory, or accumulated tracking progress.
Save/load restores the same future production from supported algorithm/profile versions and the
ordinary population, allocation, position, technology, and resource fields.

### Share-based hunting work risk

`HuntingRiskAlgorithm: "linear-share-work-risk-v1"` defines the additional acute danger from
ordinary hunting and megafauna tracking. For each role `a`, capture its share directly from the
accepted start-of-turn basis-point vector, never by dividing worker counts by population. For
each existing acute event kind `k`, the unmitigated work-risk contribution is:

```text
work_share[b, a] = float64(AllocationBP[b, a]) / float64(AllocationBasisPoints)
active_hunting_share[b] = work_share[b, Hunting]  if HuntingRate(b, origin) > 0
                           0                       otherwise
active_megafauna_share[b] = work_share[b, MegafaunaTracking]
                             if MegafaunaRate(b, origin) > 0
                             0 otherwise
work_risk[b, k]  = active_hunting_share[b] * WorkRiskCoeff(Hunting, k)
                   + active_megafauna_share[b]
                     * WorkRiskCoeff(MegafaunaTracking, k)
```

`WorkRiskCoeff` supplies a finite non-negative full-share risk weight by role and acute-event kind
from the table below; it has no fauna-profile dimension. The origin's current region × biome profile
and acquired technology gate only whether that role's coefficient is active, through the positive-rate
rule below. Coefficients are independent of population, assigned shares, and potential or actual
catch amounts. Technology's
risk-mitigation fractions are excluded from these coefficients and applied once through the
existing phase-5 probability channel; a technology-access gate is not a mitigation fraction.
Work risk activates when the role's accepted share is positive and the origin profile plus acquired
technology produces a positive role rate. It remains active when stock contention reduces the
band's allocation or actual catch to zero: attempted field work created the exposure. It is inactive
when the share is zero, the profile does not support the role, or the band's current technology
leaves its rate at zero. The selected full-share coefficients, in stable acute-kind order, are:

| Acute event kind  | Hunting | Megafauna tracking |
| ----------------- | ------: | -----------------: |
| `Predation`       |  `0.04` |             `0.10` |
| `DiseaseOutbreak` |     `0` |                `0` |
| `FloodStorm`      |     `0` |                `0` |
| `ExposureFall`    |  `0.01` |             `0.04` |
| `CrossingMishap`  |     `0` |                `0` |

Thus the full-share totals are `0.05` and `0.14`, respectively. The values are added only to the
matching `Uncovered` acute components, without introducing a population or catch-size multiplier.
`CrossingMishap` coefficients are always zero: only a successfully traversed named passage may add
that risk.

For otherwise equivalent conditions where both roles' work-risk contributions are active, the
sum of megafauna's full-share coefficients across event kinds must exceed ordinary hunting's.
This is the unmitigated risk tradeoff; technology and the shared probability cap may change the
observed comparison. Both coefficient totals must be finite. Configuration validation rejects
negative/non-finite entries, invalid totals, reversed/equal active-role ordering, or a nonzero
work-related crossing coefficient; no silent rate correction or extra normalization is applied.

With coefficients and other conditions fixed, equal shares give equal work-risk vectors for
bands of 20 or 200 people; absolute casualties can differ. Zero share contributes zero from that
role, and doubling both hunting shares doubles the unmitigated work-risk vector when the complete
allocation remains valid. For the selected coefficients, 20% Hunting plus 10%
MegafaunaTracking adds `0.018` Predation weight and `0.006` ExposureFall weight; doubling both
shares doubles those additions to `0.036` and `0.012`. These are base risk weights, not final
incident probabilities. More workers caused solely by population growth cannot increase this
workforce contribution. There is no saturation curve, per-worker roll, catch-size risk bonus, or
direct population-loss calculation in this rule.

Phase 3 captures the shares and resulting unmitigated work-risk vector at the origin after climate
has selected its current profile, before demographics or migration. Phase 5 adds each kind's
captured work weight exactly once to that kind's final-tile `Uncovered` base component, before
`ProbabilityRemainingRisk` and the existing shared cap. Final-tile profiles must not already
include that same assigned-work contribution. Camp protection and natural shelter cannot reduce
it; applicable technology still can. Do not also subtract it as chronic mortality or apply it to
severity. Migration, later deaths, and newly acquired technology cannot recompute work already
performed. A band extinct before acute resolution has no acute evaluation.

Both species follow the same rule. A split copies shares; each descendant has the same per-band
work-risk contribution in equivalent conditions, not half the fraction or inherited cached risk.
This does not promise equal world-wide event counts for split and unsplit runs: the existing
one-or-two draws per surviving band still apply. At most five work weights per band, or 1,280
across 256 bands, feed the existing bounded acute components; they do not add event kinds or
protection classes. Use stable role/kind order and safe weighted sums. Since the two shares sum
to at most one, the mathematical total work weight is at most the larger full-share coefficient
total. All inputs and combined base totals must pass the existing finite-value validation.
Discard the transient vector after acute resolution. No effort history, new RNG draw, or mutable
save field is added; reload reconstructs the next turn's weights from normal saved inputs and
supported configuration.

### Share-based, turn-local shelter protection

The `Shelter` assignment represents camp setup, security, and hygiene within the modeled turn, not
just roof construction and not a saved building. These are bundled effects of one workforce role,
not three extra assignments or independent worker pools. Phase 3 captures one bounded transient
shelter-effort record per living band from
its accepted start-of-turn shelter share, before demographics change population. This record keeps
the unmodified share, not a terrain-adjusted mitigation result:

```text
shelter_share(b) = float64(AllocationBP[b, Shelter]) / float64(AllocationBasisPoints)
```

The share is in `[0, 1]` and is derived directly from the basis-point vector, never by dividing
derived worker counts by population. This avoids rounding/underflow in a worker-count intermediate
and division by zero for an extinct band; extinct bands receive no protection evaluation. Unlike
toolcraft research, shelter's workforce input is a proportion, not an absolute worker count.

The mitigation curve must use that share without a band-size multiplier. With environment,
including the tile's `NaturalShelter`, technology, and other non-workforce inputs held constant,
the same share produces the same workforce-derived mitigation fraction for covered causes in bands
of any size. For example, 2,000 basis points is the same 20% workforce input for a band of 20 or
200; their absolute losses and lives saved may differ. A 20% workforce input does not mean 20%
damage reduction, and a 100% workforce input cannot grant immunity through shelter alone.

`ShelterAlgorithm: "saturating-share-terrain-v1"` uses a smooth diminishing-returns base curve with
the selected initial values `MaxShelterMitigation = 0.60`, `ShelterHalfSaturation = 0.25`, and
`NaturalShelterEfficiencyBonus = 1.00`. For workforce share `s` and an applicable natural-shelter
rating `n`:

```text
efficiency       = 1 + NaturalShelterEfficiencyBonus * n
effective_effort = s * efficiency
ShelterCurve(s, n) = MaxShelterMitigation
                     * saturation_ratio(effective_effort, ShelterHalfSaturation)
```

`saturation_ratio(w, h)` is the overflow-safe evaluation of `w / (w + h)` specified in the
toolcraft section below; use that evaluation, not a direct sum of potentially large operands.
`s` and `n` are both in `[0, 1]`, so efficiency and effective effort stay finite under the validated
constants. Effective effort is a dimensionless efficiency-weighted input, not extra workers or a
rewritten allocation; it may exceed one without allocating more than 100% of the band.

For fixed terrain, the mathematical curve increases with shelter share and has diminishing marginal
returns: equal increases in share provide progressively smaller gains. At effective effort equal
to `ShelterHalfSaturation`, mitigation is exactly half of `MaxShelterMitigation`; that point is
reachable only if the corresponding share is in `[0, 1]`. The asymptotic cap need not be attainable
within a legal allocation. Floating-point results must stay in `[0, MaxShelterMitigation]`;
extreme ratios may round to an endpoint, but never to full immunity because the cap itself is
strictly below one. Full allocation on the highest-rated terrain remains subject to the same cap.

This bound applies to each cause component's entire labor-derived shelter contribution, including
any natural-terrain efficiency bonus; it is not a cap on the combined effect of technology and camp
work. Their remaining-risk composition is specified below. It does not imply a positive loss on
every turn: base risk can be zero or an acute event may not occur. Acute camp protection applies
only to probability before event selection, never to severity, and the terrain bonus must not be
added a second time outside this curve. All curve and cause-scale constants below are tuned before
release; changing them or their mapping afterward requires a new shelter-algorithm version and
explicit save migration, not silent reinterpretation of a campaign.

For covered environmental exposure, `NaturalShelter` is a labor-efficiency modifier: favorable
terrain allows a smaller shelter share to reach the same attainable mitigation. At a fixed share,
raising the rating must never reduce protection; positive-share, unsaturated fixtures must show an
actual benefit. Rating zero uses the ordinary workforce curve, not zero protection, and rating one
does not mean immunity. The modifier changes how efficiently work reaches the shelter cap, not the
cap itself. It applies identically to sapiens and archaics and does not depend on band size or the
number of other bands on the tile. This is a regional abstraction, not competition for a finite
number of cave places.

**Cause-specific camp protection.** The closed domain enum `ShelterProtectionClass` has four
values, in stable order: `Exposure`, `CampPredation`, `CampDisease`, and `Uncovered`, with
`ShelterProtectionClassCount = 4`. For captured share `s = shelter_share(b)` and the relevant
phase's tile rating `n`, each component receives exactly one labor-mitigation fraction:

| Protection class | Covered risk                                                           | Labor mitigation                         |
| ---------------- | ---------------------------------------------------------------------- | ---------------------------------------- |
| `Exposure`       | Temperature/weather exposure, not injury or direct food/water shortage | `ShelterCurve(s, n)`                     |
| `CampPredation`  | Attacks attributable to camp/resting-site danger                       | `CampSecurityScale * ShelterCurve(s, 0)` |
| `CampDisease`    | Disease risk attributable to camp hygiene and waste handling           | `CampHygieneScale * ShelterCurve(s, 0)`  |
| `Uncovered`      | All remaining risk components                                          | `0`                                      |

The two additional selected inputs are `CampSecurityScale = 0.75` and
`CampHygieneScale = 0.40`; the latter is a modest benefit, not general disease resistance.
Configuration requires finite `0 < CampSecurityScale <= 1` and finite
`0 < CampHygieneScale < 1`.
Both scale the ordinary terrain-zero curve, so caves cannot silently improve hygiene or camp
security. The existing natural-terrain bonus remains exposure-only. Every class uses the same
captured share; the class outputs are applied to disjoint risk components, never added together as
three reductions of the same loss. Scaling preserves diminishing returns and the below-immunity
bound, while zero share gives zero protection in every class.

Disease/predation class membership comes from closed risk profiles, not simply the band's tile or
the event-kind label. Hunting, megafauna-tracking, and travel-related predation are `Uncovered`, as
are disease components unrelated to camp conditions, direct food/water shortages, falls, flood
injury, and crossing mishaps. A `DiseaseOutbreak` is not wholly preventable by hygiene merely because
it resolves while a band occupies a land tile. This is an aggregate camp-care abstraction, with no
daily schedule, guard roster, sanitation stock, infection/recovery state, or epidemic/contact model.

The chronic profile and each of the five acute event kinds expose at most one aggregate component
per protection class; the seasonal profile covers three of the four classes, omitting
`CampPredation` for the reason its table gives. Components partition the original profile rather
than duplicating it, are finite and non-negative with a finite total, and are evaluated in the
stable class order.
For example, `Predation` separates camp attacks from hunting/travel attacks, `DiseaseOutbreak`
separates camp-related from other disease, and `ExposureFall` separates exposure from falls.
Camp security must not cancel megafauna-tracking's added field risk. Profile dimensions,
partition/completeness rules, and validation close before implementation against the selected acute
profiles below. Adding this attribution requires no new event kinds or RNG draws. Apply the selected
class effect before aggregating a mixed cause, never to its whole total.
Keep the existing phase-3 mortality scaling, acute probability cap, and one-event-per-band rule.
Uncovered component inputs receive no shelter modifier; shared caps and changed populations can
still affect final aggregate losses or normalized probabilities.

This adds at most one mortality/probability contribution per class per profile: the three-class
seasonal profile, the four-class chronic profile, and five four-class acute kinds give
`3 + 4 + (4 * 5) = 27` bounded component slots per band, at most `6_912` across
`MaxBands = 256`. The separate background-disease health channel uses the two fixed contributions
specified above, not extra protection classes or copies of these mortality/probability losses.
They are configured/derived turn data, not a persisted cause history. No dynamic risk-component
list is introduced. Acute shelter effects reduce the probability components for covered harm
before event selection; they never reduce the severity of an incident that occurs. For exposure,
this models avoiding a harmful incident, not preventing weather itself. The phase-5 contract below
specifies aggregation and the shared cap. No integration may apply the same labor protection twice.

**Combining technology and camp protection.** For one risk component and effect channel, let
`tech` be its technology-only mitigation fraction and `camp = CampMitigation(b, c, tile)` be the
class's labor fraction from the shelter table, using the captured share and the applicable tile.
Use multiplicative remaining risk:

```text
RemainingRisk(tech, camp)      = (1 - tech) * (1 - camp)
CombinedMitigation(tech, camp) = 1 - RemainingRisk(tech, camp)
```

For example, `tech = 0.50` and `camp = 0.50` leave `0.25` of the component's unmitigated risk,
or 75% combined protection. Neither fraction is added to the other or applied again after this
composition. Inputs require finite `tech` in `[0, 1]` and finite `camp` in
`[0, MaxShelterMitigation]`. Zero technology preserves the camp-only result; zero camp work preserves
the technology-only result. `Uncovered` has `camp = 0`, not an automatic technology exemption.
Each source applies only to the risks and effect channels its own contract covers.

The technology-only fraction comes from the band's acquired-tech modifier tables, excluding camp
work and its natural-terrain bonus. When more than one acquired technology applies to the same
component and effect channel, combine them in stable technology-enum order by multiplying their
remaining-risk fractions:

```text
TechnologyRemainingRisk(b, scope, channel) = product_t (1 - TechEffect[t, scope, channel])
TechnologyMitigation(b, scope, channel)     = 1 - TechnologyRemainingRisk(b, scope, channel)
```

An empty product is one, so no applicable technology gives zero technology mitigation. Do not add
technology effects: two effects of `0.50` and `0.30` leave `0.50 * 0.70 = 0.35` risk, or `0.65`
technology mitigation. Feed that one derived fraction into `RemainingRisk(tech, camp)` above.
Partial research and technologies acquired at the end of this turn supply no current-turn
mitigation.

The approved initial **strong technology-mitigation table** is complete for hazard and health channels:

| Technology           | Applicable hazard/health channel | Component or event scope                                      | Mitigation |
| -------------------- | -------------------------------- | ------------------------------------------------------------- | ---------: |
| `Firecraft`          | Seasonal/chronic remaining risk  | `Exposure`                                                    |     `0.20` |
| `Firecraft`          | Acute probability                | `Predation/CampPredation`                                     |     `0.15` |
| `Firecraft`          | Acute probability                | `ExposureFall/Exposure`                                       |     `0.20` |
| `HaftedTools`        | Acute probability                | `Predation/Uncovered`                                         |     `0.20` |
| `HaftedTools`        | Acute severity                   | `Predation`                                                   |     `0.15` |
| `TailoredClothing`   | Seasonal/chronic remaining risk  | `Exposure`                                                    |     `0.50` |
| `TailoredClothing`   | Acute probability                | `ExposureFall/Exposure`                                       |     `0.40` |
| `Campcraft`          | Seasonal/chronic remaining risk  | `Exposure`                                                    |     `0.30` |
| `Campcraft`          | Chronic remaining risk           | `CampPredation`                                               |     `0.40` |
| `Campcraft`          | Seasonal/chronic remaining risk  | `CampDisease`                                                 |     `0.30` |
| `Campcraft`          | Endemic-disease health           | Camp disease                                                  |     `0.30` |
| `Campcraft`          | Acute probability                | `Predation/CampPredation`                                     |     `0.40` |
| `Campcraft`          | Acute probability                | `DiseaseOutbreak/CampDisease`                                 |     `0.30` |
| `Campcraft`          | Acute probability                | `ExposureFall/Exposure`                                       |     `0.30` |
| `MedicinalKnowledge` | Chronic remaining risk           | `CampDisease`                                                 |     `0.45` |
| `MedicinalKnowledge` | Endemic-disease health           | Camp and non-camp disease                                     |     `0.45` |
| `MedicinalKnowledge` | Acute probability                | `DiseaseOutbreak/CampDisease` and `DiseaseOutbreak/Uncovered` |     `0.40` |
| `MedicinalKnowledge` | Acute severity                   | `DiseaseOutbreak`                                             |     `0.35` |
| `MedicinalKnowledge` | Outbreak health damage           | `DiseaseOutbreak`                                             |     `0.50` |
| `Trapping`           | Acute probability                | `Predation/Uncovered`                                         |     `0.20` |
| `CoastalNavigation`  | Acute probability                | `CrossingMishap/Uncovered`                                    |     `0.50` |
| `CoastalNavigation`  | Acute severity                   | `CrossingMishap`                                              |     `0.35` |

`PlantKnowledge` and `CordageAndNets` have no direct hazard or health modifier; they remain
prerequisites and affect their separately authored collection/access channels. A table row applies
only to the named consumer, event kind, class, and effect channel. In particular, an exposure
modifier does not reduce a fall component, `MedicinalKnowledge` does not reduce predation, and a
probability effect does not imply severity or health protection. The `Campcraft` rows do not
create a fifth protection class or a blanket camp modifier. `Uncovered` remains eligible only for
the explicitly named event rows above.

Validate every listed value as finite and in `[0, 1]`, reject duplicate technology/channel/scope
rows, and require every lookup dimension to be a closed enum value. The table is versioned
`HazardAlgorithm` configuration. It supplies the selected strong initial values without changing
the locked technology DAG or the collection/access effects owned by other algorithms.
An authored severity-only technology effect stays in `SeverityMitigation`, not in acute probability;
camp work never enters severity. An effect already included in `tech` must not also reduce the base
risk or be reapplied after aggregation.

Compute and apply `RemainingRisk` directly to the unmitigated component, before the existing shared
caps. Do not recover it by subtracting a rounded `CombinedMitigation` from one: the derived display
fraction can round to one when a small positive residual remains. The residual product is finite
and in `[0, 1]`; it decreases monotonically as either input increases. Combined protection may
exceed the labor-only cap. Mathematically it remains below one when `tech < 1`; an explicit
`tech = 1` eliminates that component through technology alone, not through additive stacking.
This adds no new combined cap, mutable field, RNG draw, or risk component, and does not change
the existing mortality/probability caps or the one-event-per-band rule.

Zero shelter share produces zero workforce-derived shelter protection at every natural-shelter
rating; this efficiency modifier adds no separate passive protection. Any independently acquired
technology mitigation is unchanged. Each new turn starts without shelter effort from earlier turns.
The persisted allocation still assigns shelter workers automatically until the player changes it,
so turn-local protection does not require repeating the same assignment command every turn.

That effort is available for the covered phase-3 losses at the origin and the covered phase-5
hazards on the final tile, following the existing hazard timing. Phase 3 evaluates class mitigation
against the origin's risk components; phase 5 uses the final tile's components with the same
captured share. Only `Exposure` reads that phase's tile `NaturalShelter`; security and hygiene
always use rating zero. Migration preserves labor input, not the origin's cave bonus, a
physical camp, or a second allotment of work. A canceled move uses the origin for both phases;
moving between sheltered and open terrain changes the applicable terrain modifier, not the share.
Growth or mortality changes the people exposed to loss, not the captured shelter share; there is
no division by the surviving population that could inflate protection after deaths. All
shelter-effort records are discarded after acute resolution; none reaches the next turn.
Planning-time splits retain the existing population/food division and allocation/knowledge
inheritance rules. Each descendant derives its own effort on the next `AdvanceTurn` from the copied share
and uses its own tile's rating. In otherwise equivalent conditions, both receive the parent's same
workforce-derived mitigation fraction, not a bonus from splitting or a penalty for their smaller
populations; shelter adds no cached protection or building to copy.

Neither bands nor tiles store a constructed shelter level, stock, maintenance counter, or camp
history; the tile's immutable geographic rating is not accumulated work. Shelter evaluation
consumes no RNG, uses constant-time terrain lookups per band and phase, and keeps at most
`MaxBands = 256` transient records per turn. Saves and state hashes include the geography version,
accepted allocation, and resulting population/mortality, not spent shelter effort. Reload
reconstructs the fixed ratings and the next turn's effort from their normal inputs.

This settles the share-based workforce input, selected saturating curve, below-immunity cap,
exposure-only natural-shelter efficiency, camp-security/hygiene coverage, probability-only acute
protection, remaining-risk composition with the selected technology-effect table, lifetime, and
per-phase timing. The selected values remain initial balance data to validate against the seed
corpus. No persistent camp or disease system is implied by those choices.

### Band-local technology, toolcraft research, and local knowledge diffusion

The prerequisite structure is a **technology graph (DAG)**, meaning a directed acyclic graph.
A technology may have multiple prerequisites, but dependencies cannot form cycles. The closed
`TechCount = 9` enum uses this stable topological order and exact prerequisite table:

| Enum order | Technology           | Direct prerequisites          | Primary gameplay role                              |
| ---------: | -------------------- | ----------------------------- | -------------------------------------------------- |
|          0 | `Firecraft`          | None                          | Advanced and systematic fire use                   |
|          1 | `HaftedTools`        | None                          | Improved hunting                                   |
|          2 | `PlantKnowledge`     | None                          | Improved foraging and medicinal foundation         |
|          3 | `TailoredClothing`   | `HaftedTools`                 | Cold-exposure protection                           |
|          4 | `CordageAndNets`     | `HaftedTools`                 | Inshore fishing and pelagic-prey access            |
|          5 | `Campcraft`          | `Firecraft`, `HaftedTools`    | Exposure and camp-risk protection                  |
|          6 | `MedicinalKnowledge` | `Firecraft`, `PlantKnowledge` | Disease mitigation                                 |
|          7 | `Trapping`           | `CordageAndNets`              | Small/medium game and inshore trap/weir efficiency |
|          8 | `CoastalNavigation`  | `CordageAndNets`              | Wallacea passage access and pelagic efficiency     |

The three roots and six dependent nodes make one shared DAG; both species use the same nodes and
edges. Stable enum order is also the progress-vector, cost-table, bitset, UI, and deterministic
iteration order. Configuration validation requires exactly nine unique entries and nine prerequisite
sets, no unknown/self/duplicate edge, and an acyclic graph whose declared order places every direct
prerequisite before its dependent. The table fixes membership, order, prerequisite edges, and
high-level roles. Define `PrerequisitesMet(acquired, k)` as true exactly when every direct
prerequisite of `k` is in the
acquired bitset; it is true for each root. Every accepted acquired set—including initialized worlds,
splits, frames, and saves—must be transitively prerequisite-closed. An
unacquired technology whose prerequisites are not met is **locked**: its progress is exactly zero
and it cannot be the active target. Because technologies are never lost, an unlocked technology
cannot relock during valid play.

These nodes are gameplay capability bundles, not claims that one invention appeared on the turn a
band acquires them. Baseline `InshoreAquatic` access reflects the Katanda evidence for barbed and
unbarbed bone points associated with abundant fish at approximately 90,000 years ago or older
(Yellen et al. 1995, [doi:10.1126/science.7725100](https://doi.org/10.1126/science.7725100)).
Jerimalai preserves pelagic fish from about 42,000 years ago, while its definite shell hook belongs
to a later roughly 23,000–16,000-year context (O'Connor, Ono, and Clarkson 2011,
[doi:10.1126/science.1207703](https://doi.org/10.1126/science.1207703)). `CordageAndNets` and
`CoastalNavigation` therefore represent expanding capability rather than a claim that the hook
itself dates to 42,000 BP. `Trapping` represents fish traps and weirs as a broad mechanic, but Field
Notes must not date Australian stone-walled traps to 40,000 BP: directly dated Lake Condah channel
construction is about 6,600 calibrated years BP, outside this campaign (McNiven et al. 2012,
[doi:10.1016/j.jas.2011.09.007](https://doi.org/10.1016/j.jas.2011.09.007)).

The DAG is an improvement layer, not a gate on baseline survival. Every band can forage, hunt
ordinary terrestrial and inshore aquatic prey, allocate shelter work, use basic fire, and use ordinary stone tools with no acquired
technology. `Firecraft` therefore means advanced, systematic fire use rather than the first discovery
of fire; the other nodes likewise modify or extend the baseline actions only where their authored
effect tables say so.

Every new-game sapiens and archaic band begins with an empty acquired bitset, all nine
`ResearchProgress` entries exactly zero, and no active target. Consequently all three roots are
immediately unlocked and targetable, while all six dependent nodes are locked. Neither species
receives a starting knowledge advantage. This initializer is fixed scenario state, not an RNG result.

`ResearchTech` may select only an unacquired technology whose direct prerequisites are currently
acquired. A locked request returns typed `MissingTechnologyPrerequisite` without state, revision,
frame, or RNG change; `GameService` maps it to a stable boundary code and `pkg/ui` renders
“missing technology prerequisite,” following the same typed-error-to-copy split as `SplitBand`'s
typed failures. During phase-5 diffusion, eligibility uses the recipient's acquired bits in the
already-frozen pre-gain contact snapshot. A prerequisite completed by original research or diffusion
this turn cannot unlock progress for a dependent until the next turn. These rules apply equally to
both species and internal computer commands; the archaic priority check does not bypass command
validation.

This settles prerequisite enforcement for original research and diffusion and the empty new-game
technology state.

Technology is owned by bands, not species. The closed nine-value `Tech` enum is shared by both
species, but each band independently persists an acquired-tech bitset, one optional research target,
and a fixed `ResearchProgress[TechCount]` vector. `ResearchTech` selects that band's target; it never
unlocks a species-wide technology. Capacity multipliers, resource conversions, hazard mitigation,
and passage requirements always read the acting band's acquired-tech bitset. A sapiens discovery
therefore does not silently affect a remote sapiens band, and the archaic computer policy chooses
research separately for each archaic band.

The closed `ResearchCost[TechCount]` table contains finite positive thresholds. Progress for
technology `k` is always clamped to `[0, ResearchCost[k]]`. When research or diffusion reaches its
threshold, the band acquires the technology, its progress remains normalized at the threshold, and
an active target for that technology clears. Acquired technology is never lost. `ResearchTech`
rejects an already acquired target and rejects a locked target with that same typed error,
without mutation. A valid target remains unlocked because its acquired prerequisites cannot be lost.
The cost table is versioned with the technology state contract; changing a threshold requires an
explicit save migration rather than silently
reinterpreting stored progress.

The selected initial costs, in stable technology order, are:

| Technology           | `ResearchCost` |
| -------------------- | -------------: |
| `Firecraft`          |           `80` |
| `HaftedTools`        |           `90` |
| `PlantKnowledge`     |           `70` |
| `TailoredClothing`   |          `120` |
| `CordageAndNets`     |          `100` |
| `Campcraft`          |          `140` |
| `MedicinalKnowledge` |          `130` |
| `Trapping`           |          `120` |
| `CoastalNavigation`  |          `150` |

`T_tech` is the product of one selected capacity factor for each acquired technology, evaluated in
stable enum order; the empty product is exactly `1`:

| Technology           | Capacity factor |
| -------------------- | --------------: |
| `Firecraft`          |          `1.05` |
| `HaftedTools`        |          `1.03` |
| `PlantKnowledge`     |          `1.04` |
| `TailoredClothing`   |          `1.05` |
| `CordageAndNets`     |          `1.04` |
| `Campcraft`          |          `1.08` |
| `MedicinalKnowledge` |          `1.06` |
| `Trapping`           |          `1.03` |
| `CoastalNavigation`  |          `1.02` |

All factors are finite and at least one. A band holding all nine has
`T_tech = 1.4772347614192902` under the domain's explicit product-rounding rule. Apply these factors
only to `K_eff` and the corresponding stress denominator; a technology's separately authored
collection, hazard, health, or passage effect is not folded into this product or applied twice.

`ResearchProductionAlgorithm: "saturating-toolcraft-v1"` uses the selected positive inputs
`MaxResearchPerTurn = 20` and `ResearchHalfSaturation = 25`. For a band `b`, `toolcraft_workers(b)` is the
finite non-negative worker count derived by `proportional-basis-points-v1` from its toolcraft share
after all player planning and the atomic archaic planning batch, but before phase 1 changes the
world. Its basis-point bound makes the result no greater than the band's start-of-turn population.
For technology `k`:

```text
research_gain(b, k) =
    MaxResearchPerTurn * toolcraft_workers(b)
        / (toolcraft_workers(b) + ResearchHalfSaturation)
        if k is b's valid, unacquired, prerequisite-met active target
    0   otherwise
```

The implementation evaluates the saturation ratio without directly adding two potentially large
finite values. With `w = toolcraft_workers(b)`, `h = ResearchHalfSaturation`, and `q` in `[0, 1]`:

```text
saturation_ratio(w, h) =
    q / (1 + q), where q = w / h   if w <= h
    1 / (1 + q), where q = h / w   otherwise
research_gain = MaxResearchPerTurn * saturation_ratio(w, h)
```

This is algebraically identical to `w / (w + h)`, but its only addition has operands in `[0, 1]`,
so every valid finite assignment produces a finite floating-point gain in
`[0, MaxResearchPerTurn]`; representative gameplay-scale inputs remain strictly below the
asymptotic maximum.

Zero toolcraft workers, no active target, or an already acquired technology produces zero original
progress. At `toolcraft_workers == ResearchHalfSaturation`, gain is exactly half the maximum; the
mathematical function remains below `MaxResearchPerTurn`, increases monotonically with workers, and
has diminishing marginal returns. There is no generic research pool: only the active target's
fixed progress-vector entry changes, excess over its remaining cost is discarded, and no overflow
selects or advances another technology. Phase 3 computes this gain as bounded transient turn data
before demographics change population, but does not apply it yet. An acute event later in the turn
therefore does not retroactively reduce work already performed; if the band does not survive phase-5
cleanup, its transient gain disappears with it. The formula consumes no RNG.

Configuration validation requires finite `MaxResearchPerTurn > 0` and
`ResearchHalfSaturation > 0`. Twenty-five toolcraft workers therefore produce exactly `10`
progress, 50 produce `13.333...`, and 100 produce `16`. Step 5 exercises the consumer with these
selected fixtures; step 5e first validates them against the campaign seed corpus and step 12 performs
the final calibration. After the save contract ships,
changing either requires a new research-production algorithm version
and an explicit save migration rather than silently changing the future of an existing run.
`gameapi.Band.OriginalResearchGainPreview` exposes
`min(research_gain, ResearchCost[target] - ResearchProgress[target])` so the HUD can show “Own
research next turn: +X if this band survives; contact diffusion is additional.” It is zero when
there is no unacquired active target, recomputed after `SetAssignment`, `ResearchTech`, split, turn
completion, and load, never reserves progress, and is not serialized.

Splitting copies the parent's acquired-tech bitset, research target, progress vector, and exact
heritable-state vector to both
results. Knowledge is non-rival: dividing population and stored food does not make either descendant
forget what the original band knew. Both descendants copy the same proportional allocation, so
their derived toolcraft worker counts sum mathematically to the parent's pre-split derived count;
the split cannot invent workers even though it copies prior knowledge. Every technology field is
fixed-size, so total live technology state is bounded by
`MaxBands * TechCount = 2_304` progress values plus 256 bitsets and targets.

Diffusion is local. Under `KnowledgeContactAlgorithm: "co-located-cross-species-v1"`, a surviving
post-event source and recipient of the same species are eligible when they occupy the same tile or
the endpoints of one current ordinary land edge. A sapiens/archaic pair is eligible only when both
bands occupy the same tile. The predicate is symmetric: either species can teach the other at the
same rate when their acquired technologies differ, and computer control creates no knowledge bonus
or penalty. Ordinary adjacency alone creates no exchange across the species boundary.

Named passages do not make their distant endpoints neighbors; bands can exchange knowledge after a
crossing only when their final positions satisfy the same eligibility rule on the far side. During
phase 5, after acute events and zero-population cleanup, diffusion reads one frozen contact snapshot
and accumulates all recipient updates before applying any of them. A technology learned through
diffusion therefore cannot relay onward until the next turn, and band iteration order cannot change
the result. Tile buckets avoid scanning the grid for every pair. Contact construction emits each
eligible unordered band pair once. There are at most `256 * 255 / 2 = 32_640` such pairs and, with
nine technologies, at most `32_640 * 9 = 293_760` pair-technology evaluations in one turn; all 256
bands sharing one tile reaches both bounds, and tests enforce them.

`KnowledgeDiffusionAlgorithm: "stacked-acquired-v1"` uses the closed constant
`DiffusionRate = 0.20`. For a recipient band `b` and an unacquired technology `k`, let
`source_count(b, k)` be the number of distinct eligible contact-band IDs that had `k` acquired in
the frozen snapshot. Let `recipient_prereqs_met(b, k)` be `PrerequisitesMet` evaluated from that
same snapshot's acquired bits for recipient `b`. Partial progress in a source does not spread, a
band is never its own source, and the same source ID counts once even if contact construction
encounters it by more than one path.
Because a valid split creates two distinct live band IDs, its surviving descendants count as two
sources when both independently satisfy the contact rule; no lineage or interaction history is
retained to collapse them back into their parent.

The simultaneous update is:

```text
diffusion_gain(b, k) =
    source_count(b, k) * DiffusionRate * ResearchCost[k]
        if k is unacquired and recipient_prereqs_met(b, k)
    0   otherwise
ResearchProgress_next[b, k] =
    min(ResearchCost[k],
        ResearchProgress_snapshot[b, k]
        + research_gain(b, k)
        + diffusion_gain(b, k))
```

The frozen snapshot contains the survivors' acquired bits, progress vectors, and active targets
before either kind of gain is applied. The domain service first counts unique sources and then computes one
aggregate diffusion gain; it does not add one floating-point contribution at a time in discovery
order. Diffusion applies to an unlocked technology whether or not `k` is the recipient's active
research target. It gives zero progress to a locked technology even when eligible sources know it.
A newly completed prerequisite—whether completed by original research, diffusion, or their sum—does
not change any `recipient_prereqs_met` result from the frozen snapshot, so it cannot enable a dependent
until the next turn. A newly completed technology also cannot teach another band or affect the
just-completed turn's extraction, demographics, movement, or acute events. It is available in the
next planning frame and participates in that frame's `T_tech` stress. Zero total gain leaves
progress unchanged. Reaching or exceeding the threshold normalizes progress to exactly
`ResearchCost[k]`, sets the acquired bit, and clears a matching research target. A band that already
has `k` receives neither gain. At the selected diffusion rate, one knowledgeable source moves a
zero-progress recipient 20% of the way to that technology's cost, and five sources complete it in a
single contact turn — in both cases only when the recipient's prerequisites were already met in the
frozen snapshot. Configuration validation requires a finite `DiffusionRate` in
`(0, 1]`; changing the rate requires a new algorithm version and an explicit save migration.

This settles technology ownership, original production, state bounds, same- and cross-species
contact geometry, split inheritance, transfer strength, simultaneous resolution, and availability
timing. The nine-node membership, order, edges, high-level roles, and strict prerequisite rules are
also settled above.

### Heritable variation, selection, and introgression

Heritable adaptation is deliberately separate from the technology DAG. Technology is acquired
knowledge with prerequisites, research progress, and local teaching; heritable state has no
prerequisites or player-selected research target. Every band persists a fixed
`HeritableState[HeritableTraitCount]` vector of six finite `float64` values in `[0, 1]`, in this
stable enum order:

| Enum order | Trait                    | Stored interpretation                                | Environmental effect and tradeoff                                                                                     |
| ---------: | ------------------------ | ---------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
|          0 | `ColdAdaptation`         | variant frequency                                    | Reduces temperature-driven exposure; no generic bonus outside cold pressure.                                          |
|          1 | `HighAltitudeAdaptation` | variant frequency                                    | Reduces elevation-derived hypoxia pressure, especially in Mountainous Highlands.                                      |
|          2 | `InnateImmuneReactivity` | variant frequency                                    | Reduces configured microbial disease risk but can add inflammatory or allergic burden where pathogen pressure is low. |
|          3 | `AridClimateAdaptation`  | variant frequency                                    | Modestly reduces the temperature-driven portion of water demand in hot terrain; it never creates water.               |
|          4 | `PigmentationLevel`      | quantitative band mean from low to high pigmentation | Higher values protect against high-UV damage; under low UV they increase the vitamin-D constraint.                    |
|          5 | `FattyAcidMetabolism`    | variant frequency                                    | Trades off usable-food conversion for plant-heavy versus animal- or aquatic-fat-heavy intake.                         |

Every trait is a **frequency-valued** effective frequency for a modeled variant or variant bundle,
except `PigmentationLevel`, which is explicitly a quantitative band mean. The vector is a band-level
population abstraction, not six individual genotypes and not a claim that every named adaptation is
controlled by one locus. Species identity remains a separate immutable field:
selection, mutation, and interbreeding never turn a sapiens band into an archaic band or vice versa.

V1 excludes a seventh candidate trait, `HbS`, which would require derived Hardy-Weinberg genotype
shares, a heterozygote malaria benefit, a homozygote anemia burden, and a
`FalciparumPressure(region, biome)` lookup; §14 records that scope boundary. The reason is cost
against reachable value rather than any doubt about the biology: published age estimates run
from roughly 22,000 years ago to a Holocene origin, so even the most permissive reading made the
allele eligible only from turn 360 of 400, it was designed to be rare enough that a campaign could
end without ever seeing it, and Field Notes would have had to label its appearance partly
counterfactual. For that, v1 would have paid a trait slot, genotype-share derivation, a complete
authored region × biome pressure table with its own validation, and separate anemia health and
mortality components — all of it on the critical path through the balance pass. Removing it also
removes the only date-gated behavior from the mutation system. Nothing else in the design depended
on it: no other trait reads it, and its enum position was last, so no ordinal shifts.

New-game scenario data supplies a complete species-and-start-region-specific vector for every band.
This is standing variation, not an RNG draw. Configuration validation requires exactly six finite
inclusive `[0, 1]` values for every starting profile. Values below follow stable trait order
`ColdAdaptation`, `HighAltitudeAdaptation`, `InnateImmuneReactivity`, `AridClimateAdaptation`,
`PigmentationLevel`, `FattyAcidMetabolism`:

| Species and start region     | Selected standing-variation vector        |
| ---------------------------- | ----------------------------------------- |
| `HomoSapiens`, East Africa   | `0.10 / 0.00 / 0.45 / 0.55 / 0.85 / 0.40` |
| `ArchaicHominin`, Levant     | `0.45 / 0.00 / 0.60 / 0.25 / 0.65 / 0.55` |
| `ArchaicHominin`, Frangistan | `0.75 / 0.10 / 0.55 / 0.00 / 0.45 / 0.65` |
| `ArchaicHominin`, Yellow River Basin | `0.80 / 0.35 / 0.60 / 0.05 / 0.50 / 0.65` |

These are initial game-balance abstractions rather than population-genetic estimates. A split copies
the parent's exact vector
to both descendants, with no founder-effect roll or frequency renormalization. Migration preserves
the vector. Population change alone does not rescale it.

Trait effects are individual, deterministic functions rather than one universal linear modifier.
Each function reads the band's **start-of-turn** heritable value and the same versioned environmental
input used by the affected mechanic. Let `x` be the frozen trait value. The selected initial effect
and selection functions are:

Pigmentation uses a pure derived UV proxy rather than mutable tile state or a new authored table. For
tile `i` and turn `t`, evaluate the following under §5's explicit product-rounding rule:

```text
LatitudeUV(i)       = 1 - LatitudeSinSquared[TileRow(i)]

SeasonUVFactor[SeasonWarm]    = 1.00
SeasonUVFactor[SeasonCooling] = 0.85
SeasonUVFactor[SeasonCold]    = 0.70
SeasonUVFactor[SeasonWarming] = 0.85

AltitudeUVFactor(i) = 1 + UVAltitudeGainPerKm * ElevationKm(i)
UVAltitudeGainPerKm = 0.10

UVExposure(i, t) = clamp01(
    LatitudeUV(i) * SeasonUVFactor[Season(t)] * AltitudeUVFactor(i)
)
```

This deliberately reuses `TemperatureAlgorithm`'s exact 64-row `LatitudeSinSquared` table, the
versioned geography's elevation, and the global four-season gameplay cycle. It introduces no runtime
trigonometry, RNG draw, mutable field, or saved value. `Season(t)` is the synchronized abstract season
defined by the campaign clock, not hemisphere-aware solar declination. The shape reflects latitude,
season, and altitude as major UV influences; `0.10` per kilometre follows the WHO's approximate
[10% increase per 1,000 metres](https://www.who.int/news-room/questions-and-answers/item/radiation-ultraviolet-%28uv%29).
It remains a gameplay proxy, not a paleoradiative reconstruction: v1 does not model cloud cover,
ozone, surface reflection, time of day, or behavior-specific exposure.

- `ColdAdaptation` uses
  `cold_pressure = clamp((10 - LocalTemperatureC) / 30, 0, 1)`. Each explicitly named
  temperature-exposure component receives remaining-risk factor
  `1 - 0.50 * x * cold_pressure`; its selection delta is
  `0.012 * (cold_pressure - 0.25) * x * (1 - x)`.
- `HighAltitudeAdaptation` uses
  `hypoxia_pressure = clamp((ElevationKm - 1.5) / 2.5, 0, 1)`. The named high-altitude chronic
  `Exposure` component receives remaining-risk factor `1 - 0.60 * x * hypoxia_pressure`; its selection delta
  is `0.012 * (hypoxia_pressure - 0.15) * x * (1 - x)`. Hypoxia is not a random acute event.
- `InnateImmuneReactivity` uses
  `pathogen_pressure = clamp((CampDiseaseHealthRate + NonCampDiseaseHealthRate) / 0.04, 0, 1)` from
  the current biome. Named microbial-disease components receive remaining-risk factor
  `1 - 0.45 * x`. Low pathogen pressure separately adds health burden
  `0.010 * x * (1 - pathogen_pressure)`. Its selection delta is
  `0.010 * (pathogen_pressure - 0.45) * x * (1 - x)`. The abstraction is informed by evidence for
  archaic immune introgression at loci such as TLR and OAS, but does not label `x` as one haplotype.
- `AridClimateAdaptation` uses
  `heat_pressure = clamp((LocalTemperatureC - 20) / 15, 0, 1)`. It multiplies only the heat-added
  portion of per-person water demand by `1 - 0.40 * x`; baseline `BaseWaterPerPerson` remains
  payable. Its selection delta is `0.010 * (heat_pressure - 0.25) * x * (1 - x)`.
- `PigmentationLevel` reads the derived `UVExposure` in `[0, 1]`. Low
  pigmentation under high UV adds chronic-mortality burden
  `0.012 * UVExposure * max(0, UVExposure - x)`; high pigmentation under low UV adds vitamin-D
  health burden `0.010 * (1 - UVExposure) * max(0, x - UVExposure)`. Its quantitative mean has
  selection delta `0.005 * (UVExposure - x)`, so latitude influences selection rather than
  directing mutation. Neither burden may be copied into the other's channel.
- `FattyAcidMetabolism` uses source conversion factors `1.10 - 0.20 * x` for Plant,
  `0.95 + 0.10 * x` for Animal, and `0.90 + 0.20 * x` for Aquatic. Apply the
  factor once before source totals combine. Food consumption retains no persisted provenance, so
  phase 3 carries four turn-local components—post-spoilage `Reserve`, `Plant`, `Animal`, and
  `Aquatic`—and attributes a partial meal pro rata with no source priority:

  ```text
  ConsumedComponent[q] = FoodConsumed * AvailableComponent[q] / FoodAvailable
  ```

  The zero-available case produces four zeros. Iterate in the component order above and assign the
  final component the conservative residual so the components sum exactly to `FoodConsumed` under
  the product-rounding rule. Let `animal_food_share` be
  `(ConsumedComponent[Animal] + ConsumedComponent[Aquatic]) / FoodConsumed`; carried reserve is in
  the denominator but never guessed into a source numerator. Its selection delta is
  `0.012 * (animal_food_share - 0.50) * x * (1 - x)`, or exactly zero when no food was consumed.

The following table is exhaustive; no genetic modifier reaches a channel not listed here:

| Trait | Exact effect channels |
| ----- | --------------------- |
| `ColdAdaptation` | Phase-3 seasonal `Exposure` and chronic `Exposure` remaining risk; cold selection pressure. |
| `HighAltitudeAdaptation` | Phase-3 chronic `Exposure` remaining risk; hypoxia selection pressure. |
| `InnateImmuneReactivity` | Endemic camp/non-camp disease health loss; seasonal `CampDisease`; chronic `CampDisease` and `Uncovered`; all acute `DiseaseOutbreak` class probabilities; outbreak health loss; low-pathogen inflammatory health burden; pathogen selection pressure. It does not reduce acute severity. |
| `AridClimateAdaptation` | Only the heat-added part of canonical water demand; heat selection pressure. |
| `PigmentationLevel` | Additive chronic UV mortality burden; phase-3 vitamin-D health burden; UV selection pressure. |
| `FattyAcidMetabolism` | The three `FoodSource` conversion factors and the pro-rata meal-attribution selection pressure above. |

When multiple heritable remaining-risk factors affect one named component, multiply them in
`HeritableTrait` enum order after technology/camp factors. Sum additive health or mortality burdens
in that same order before the owning channel's single cap. `GeneticSeverityMitigation = 0` in v1:
no trait reduces acute severity, and `InnateImmuneReactivity` affects outbreak probability and
health loss only. Each table entry is applied exactly once; generic genetic fitness, hazard, health,
or survival multipliers are forbidden.

Every pressure, modifier, burden, and delta must be finite and within its stated bound before use.
The formulas do not create a generic fitness multiplier, and an effect assigned to one health,
mortality, demand, or conversion channel cannot be repeated in another.

Function fixtures exercise every trait's distinct effect shape and exclusive integration channel,
including pure derived UV and hypoxia inputs, reduction of only the heat-added portion of water
demand, per-source fatty-acid conversion, and the separate microbial-benefit and inflammatory-burden
immune components. UV fixtures cover all four season factors, a synthetic equatorial row and paired
positive/negative latitudes, sea level and a one-kilometre elevation difference, and the high-end
clamp. Before a shared clamp collapses the values, one fixed-tile fixture requires
`SeasonWarm > SeasonCooling = SeasonWarming > SeasonCold`; at an unclamped synthetic input, moving
from sea level to one kilometre multiplies exposure by exactly `1.10`. Reject non-finite inputs and
intermediates rather than using the final clamp to repair them. None of these function fixtures
consumes `WorldRNG`.

No trait derives genotype shares. Every effect function reads the band's single stored frequency or
mean directly, so v1 contains no Hardy-Weinberg step, no genotype decomposition, and no trait whose
benefit and burden fall on different genotypes of the same value. That was the one modelling shape
`HbS` required, and it left with it.

**Local gene flow.** Surviving same-species bands exchange all six values automatically when they
finish the turn co-located or at the endpoints of one ordinary land edge, using the same local
geometry and named-passage exclusion as knowledge contact. Cross-species co-location by itself
continues only the existing resource competition and technology contact; it causes no genetic
exchange. A sapiens band must actively queue `Interbreed{BandID, TargetBandID}` during planning.
The actor and target must be distinct, living, co-located bands; the actor must be `HomoSapiens` and
the target `ArchaicHominin`. The frame exposes the eligible target IDs for each sapiens band. The
action transfers no technology, lets the player select no individual trait, and is limited to one
target per sapiens band per turn. There is no separate `Compete` command: refusing interbreeding
simply leaves both bands under the ordinary shared-resource competition rules.

`Interbreed` is one of a band's spatial actions. A band with a queued migration or an already-used
spatial action rejects it; after an accepted interbreeding intent, that sapiens band cannot migrate
or split in the same planning period. A successful `SplitBand` marks both descendants' spatial
actions used until the turn advances, while `QueueMigration` marks its band used. Assignment and
research commands remain available. The persisted `SpatialActionUsed` marker is necessary so a save
made after an immediate split cannot evade the rule after reload. It clears only after the completed
turn. Interbreeding does not consume the target archaic band's spatial action, so the computer may
still move or split it. The accepted intent records the two band IDs and their co-location at command
time; it resolves if both named bands survive through phase-5 cleanup even if the archaic later
moves. If either is gone, it has no genetic effect and is discarded.
That failed resolution appends a typed non-acute cancellation alert so the spent choice is visible;
it does not consume an extra RNG draw or play the acute-event sound.

Gene flow reads one frozen survivor snapshot and applies simultaneously. Every eligible contact edge
contributes reciprocally across the entire vector. Same-species edges use
`SameSpeciesGeneFlowRate = 0.025`; an accepted interbreeding edge uses
`InterbreedGeneFlowRate = 0.10`. The shared cap is `MaxGeneFlowPerTurn = 0.20`. For recipient `b`, trait `k`, and its
eligible partners `c` sorted by band ID, define `w_bc = rate_bc * Population_snapshot[c]`, then:

```text
influence    = sum_c(w_bc)
partner_mean = sum_c(w_bc * state_snapshot[c,k]) / influence
mix          = min(MaxGeneFlowPerTurn, influence / (Population_snapshot[b] + influence))
gene_flow_delta[b,k] = mix * (partner_mean - state_snapshot[b,k])
```

Zero influence gives zero delta. This population-weighted aggregate is order-independent, remains
bounded when several sapiens bands target one archaic band, and prevents an arbitrary last partner
from winning. Implement the weighted mean and `influence / (population + influence)` with scaled
weights and the same overflow-safe branch form used by research saturation; a final clamp must not
hide a non-finite intermediate. Configuration requires finite
`0 < SameSpeciesGeneFlowRate <= MaxGeneFlowPerTurn <= 1` and
`0 < InterbreedGeneFlowRate <= MaxGeneFlowPerTurn`. For equal-sized isolated pairs, the selected
rates move a recipient by `0.025 / 1.025` or `0.10 / 1.10` of the difference—about `2.44%` or
`9.09%`—before the cap. Changing the formula or those values after release requires an
algorithm-version migration.

**Selection and rare mutation.** At phase 5, each survivor also gets one per-trait deterministic
`SelectionDelta(trait, start state, captured environment and outcome)` from its versioned trait
function. All selection and gene-flow deltas read frozen inputs and are accumulated before one clamp,
so a value gained from one partner cannot relay or affect survival until the next turn. Selection
consumes no RNG. There is no background random genetic drift in v1.
Selection pressure uses the same checkpoints as the trait's authored gameplay channels: phase-3
origin captures for food, water, seasonal, and chronic inputs; phase-5 final-tile captures for acute
inputs; and both in a fixed tuple when a trait explicitly spans both. It never re-queries a moved
band's destination as though it had gathered there or moves an acute pressure back to the origin.

Rare emergence is the only genetic use of `WorldRNG`. In ascending band-ID and trait-enum order,
the domain draws once through `WorldRNG` for each mutation-eligible frequency-valued trait that is exactly absent from
that survivor. A successful draw contributes that trait's small configured
`MutationEntryFrequency[trait]`;
otherwise it contributes zero. `PigmentationLevel` has no point-emergence draw in v1, being a
quantitative mean rather than a frequency, so five of the six traits are mutation-eligible. There is
no date or species gate on emergence: every eligible trait is drawn for on every turn from turn 1,
which is simpler than the rule an `HbS` chronology would have forced and removes the mutation
system's only dependency on `CampaignDate`. The selected fixed arrays are
`MutationEntryFrequency[HeritableTraitCount]` and
`MutationProbability[HeritableTraitCount]`, both keyed by the closed trait enum:

| Trait                    | Probability constant ID                 | `MutationEntryFrequency` | `MutationProbability` when exactly absent |
| ------------------------ | --------------------------------------- | -----------------------: | ------------------------------------------: |
| `ColdAdaptation`         | `MutationProbabilityCold`               |                  `0.020` |                                `0.00020` |
| `HighAltitudeAdaptation` | `MutationProbabilityHighAltitude`       |                  `0.020` |                                `0.00015` |
| `InnateImmuneReactivity` | `MutationProbabilityImmune`             |                  `0.015` |                                `0.00025` |
| `AridClimateAdaptation`  | `MutationProbabilityArid`               |                  `0.020` |                                `0.00020` |
| `PigmentationLevel`      | `MutationProbabilityPigmentation`       |       `0` (not eligible) |                                      `0` |
| `FattyAcidMetabolism`    | `MutationProbabilityFattyAcid`          |                  `0.020` |                                `0.00020` |

Every `MutationProbability[trait]` and `MutationEntryFrequency[trait]` must be finite and in
`[0, 1]`; a mutation-eligible
row requires a positive entry frequency, while the pigmentation row requires both values to be zero.
Draws and results are bounded by `MaxBands * HeritableTraitCount = 1_536` checks per turn.
The simultaneous update is:

```text
state_next[b,k] = clamp(state_snapshot[b,k]
                         + selection_delta[b,k]
                         + gene_flow_delta[b,k]
                         + mutation_entry[b,k], 0, 1)
```

Validate every delta and the unclamped sum as finite before applying the boundary clamp; clamping
cannot repair invalid configuration or arithmetic.

Mutation fixtures fix the exact ascending band-ID/trait-enum draw order, successful and unsuccessful
entry, and the `MaxBands * HeritableTraitCount = 1_536` check bound. They prove that
`PigmentationLevel` never consumes a draw and that, for identical state and a controlled draw
sequence, turns 1 and 399 produce identical emergence results for both species; no hidden date or
species gate is permitted.

The selected numeric trait effects, standing-variation table, selection functions, gene-flow rates,
mutation probabilities, and entry frequencies remain initial values to validate in the balance
pass. The settled contract is the six-value representation, inheritance, active cross-species choice, automatic
same-species contact, reciprocal all-trait exchange, deterministic simultaneous resolution,
seeded-emergence-only randomness, no drift, and next-turn availability.

### Assisted migration

Every `gameapi.Band` carries a ranked list of its currently reachable destinations. For sapiens,
movement remains entirely player-controlled: `Stress > SplitStressThreshold` adds a non-modal spatial-pressure alert
and permits splitting, but it neither reveals the list nor changes whole-band migration eligibility,
and the domain policy never queues a sapiens move or split in response. A reachable edge is either an
ordinary grid-neighbor land edge or an eligible named passage edge. The player may queue a sapiens
whole-band migration over either edge type at any stress level, split a stressed sapiens band into
any valid adjacent land tile, or ignore the recommendation and accept continued capacity/resource
pressure. Computer-controlled archaic bands use the same candidates and movement validators through
the deterministic policy below.

`grid.go` defines the ordinary topology once for candidate generation, command validation, and
phase-4 revalidation. From `(x, y)`, the eight possible direction deltas are the stable order
`N=(0,-1)`, `NE=(1,-1)`, `E=(1,0)`, `SE=(1,1)`, `S=(0,1)`, `SW=(-1,1)`, `W=(-1,0)`,
`NW=(-1,-1)`; out-of-bounds and non-land destinations are omitted. Cardinal edges have step length
`1`. A diagonal edge has step length `math.Sqrt2` and exists only when both
orthogonal corner tiles `(x+dx, y)` and `(x, y+dy)` are also land. The corner test uses the fixed
land mask, not current habitability or biome, so climate changes cannot make the graph asymmetric or
open a water shortcut. Destination habitability remains a separate current-turn eligibility check.
For an ordinary edge from `i` to `j`:

```
step_length(i, j) = 1            for a cardinal edge
                    math.Sqrt2   for a diagonal edge
C_j               = clamp(MovementCurve(V(j)) * BiomeMovementFactor[Biome(j)],
                            1.0, MaxMovementCost)
MovementCost(i, j) = step_length(i, j) · C_j
```

`Biome(j)` is the destination's current derived biome in the planning snapshot and `V(j)` its
current vegetation index. `MovementCurve` is a piecewise-linear translation of the intent table's
Low through Very high movement grades. Its Initial numeric knots use two low-vegetation reference
points, the Semi-Arid Desert and Savanna band midpoints, `CanopyClosureV`, and the wet endpoint; only
values below the first knot are held flat:

|               `V` | `MovementCurve(V)` |
| ----------------: | -----------------: |
| `0.075` and below |             `2.20` |
|           `0.225` |             `2.00` |
|           `0.375` |             `1.50` |
|           `0.600` |             `1.00` |
|           `0.900` |             `1.00` |
|           `1.000` |             `1.20` |

The curve is deliberately non-monotonic at its top, and the turn is placed at `CanopyClosureV`: the
woodland mosaic from `0.600` to `CanopyClosureV` travels as cheaply as open grassland — both are the
intent table's Low grade — while closed canopy above it costs more. That keeps the penalty on the
terrain that earns it rather than spreading it across all of Riverine Woodland. These knots are
Initial values, subject to §12's balance pass.
`BiomeMovementFactor` is the second column of the per-biome deviation table in the carrying-capacity
rule above. The `V = 0.60` baseline cost is `1.00`, so the geographic factors directly produce the
Coastal Shrubland `1.25` and Mountainous Highlands `2.50` reference costs.

`MaxMovementCost = 2.75` is a selected Initial ceiling constrained by a derived bound. Named Wallacea routes must cost
strictly more than `math.Sqrt2 * max(ordinary cost)`, and the cheaper northern route is `4.00`, so
the ordinary maximum must stay strictly below `4.00 / math.Sqrt2 ~= 2.828`. At `2.75` the maximum
ordinary adjacent-land cost is `math.Sqrt2 * 2.75 ~= 3.889`, preserving the invariant with headroom.
The ceiling binds only on composed cost: the curve's own maximum is `2.20`, so Mountainous
Highlands and sufficiently low-`V` Coastal Shrubland can reach `2.75`. That headroom beneath the
Wallacea bound lets the geographic Very high grade sit strictly above the vegetative High grade.
Configuration validation asserts `MaxMovementCost > 0` and
`math.Sqrt2 * MaxMovementCost < min(Wallacea route costs)` so the selected value may be tuned only
inside the derived admissible interval.

`BiomeMovementFactor` has exactly `BiomeCount` finite positive entries — with the four vegetative
biomes pinned to `1.00` by the deviation-table validation above — and every composed cost is
finite and at least `1.0`. The topology is symmetric, but cost
intentionally need not be: entering difficult terrain can cost more than leaving it. Named passages
use their own configured route cost instead of this ordinary-edge formula. An ordinary move covers
exactly one edge in one turn; there is no hidden multi-edge pathfinding inside `SplitBand` or
`QueueMigration`.

`MovementCostAlgorithm: "destination-vegetation-v1"` versions the six curve knots including shared
`CanopyClosureV`, biome factors, destination-only composition, clamp, and step-length multiplication.
`ResourceAlgorithm` also owns that shared knot for `BaselineKCurve`; changing it moves both
identifiers. The identifier is selected now,
while v1 is unreleased; it is the only supported movement-cost identifier, and no migration from a
provisional identifier exists.

This makes `MaxGridNeighbors = 8` and therefore caps a band's candidate list at
`8 + MaxPassageEdgesPerTile = 10`. The edge relation is symmetric: if `i` has an ordinary edge to
`j`, `j` has one to `i`, even though the two directed costs may differ. Tests construct coast
corners explicitly so changing direction order, weakening the two-corner requirement, or using
habitability instead of land cannot silently alter the graph.

For band `b` and reachable destination `j`, the attraction score is:

```
WaterSurvivalEquivalent(b, j) =
    min(EcologicalK_j,
        WaterStock_j / EffectiveWaterDemandPerPerson(b, j))
R_j = UsableFoodEquivalent(b, j) + WaterSurvivalEquivalent(b, j)
W_j = 1 − WarnedMacroImpact(j, nextTurn)
C_j = min(1, K_eff_j / (P_total_j + P_b))          // 0 where K_eff_j is zero
S_j = EcologicalK_j · R_j · W_j · C_j / (MovementCost(origin, j) · (1 + P_total_j))
```

`C_j` is the crowding safety factor: the share of the arriving population — the
destination's residents plus the band considering the move — that the destination can
actually support, where `K_eff_j` applies the band's own capacity multiplier to
`EcologicalK_j`. Without it the band's own population `P_b` appeared nowhere in the score, so a
tile ranked identically for a band of twenty and one of four thousand, and the HUD paints the
top-ranked candidate gold as a recommendation. Measured over 19,316 migration decisions before it
existed, the top-ranked candidate would have cost the band people on 36 of every 1,000, rising to 79
per 1,000 for bands of 100 or more, with a safe alternative available in 218 of those cases.

`C_j` deliberately measures the sustained cost rather than one turn's. Ranking on the previewed
`CrowdingDecline` instead would make the penalty proportional to `r`: at the selected coefficient a
tile twice over capacity would be marked down by two percent, while attraction varies between tiles
by orders of magnitude, so the factor could not reorder anything. A band arriving at twice capacity
does not lose two percent, it loses half of itself over however many turns it stays. Note the
asymmetry with the preview beside it, which stays bounded by `MaxCrowdingDeclineFraction`: that bound
is a mercy applied to the outcome so arriving somewhere hostile is survivable, and feeding it into
the ranking would hide severity from the one decision that could avoid it.

Like `W_j`, `C_j` lowers the ranking without making the destination ineligible: a band may still be
sent somewhere that will hurt it and is simply no longer advised to go. Evaluated against the
unfactored score on identical world states across three corpus campaigns, it changed the top-ranked
candidate in 57 of 22,570 decisions and **every change was an improvement, with none worsened**.

`UsableFoodEquivalent` is the following complete, non-extractive preview. Let `w_g` be the six
normalized fauna-profile weights at the destination. A group weight is accessible only when the
band owns the group's required technology and the producing role is available: `SmallGame`,
`MediumGame`, and `LargeGame` use Hunting; `MegaGame` uses MegafaunaTracking; and `InshoreAquatic`
and `PelagicAquatic` use Hunting. Inaccessible weights are zero.

```text
PlantFU = FloraStock_j * FoodConversionFactor(Plant, b)
AnimalFU = FaunaStock_j
           * sum(accessible w_g for SmallGame, MediumGame, LargeGame, MegaGame)
           * FoodConversionFactor(Animal, b)
AquaticFU = FaunaStock_j
            * sum(accessible w_g for InshoreAquatic, PelagicAquatic)
            * FoodConversionFactor(Aquatic, b)
UsableFoodEquivalent(b, j) = PlantFU + AnimalFU + AquaticFU
```

Evaluate groups and then sources in stable enum order under §5's product-rounding rule. Profile
weights sum to one, so the one fauna stock is partitioned, never counted once per role; an
inaccessible group contributes exactly zero. This preview excludes worker counts, collection rates,
catch competition, reserves, and water: it measures potentially usable stock for ranking, not this
turn's harvest. `FoodConversionFactor` uses the band's start-of-turn `FattyAcidMetabolism`.
`WaterSurvivalEquivalent` is the number of people-turns the current water
stock could support at the candidate's effective per-person demand, capped by that tile's current
`EcologicalK`. This puts it on the same survival-equivalent scale as FU while preventing a large
water stock from overwhelming every other input. It is a ranking weight, not edible food, and never
satisfies hunger or enters `StoredFood`. Technology gates access and never also multiplies FU
conversion. Validation proves the two fauna sums are each in `[0, 1]`, their sum is at most one, the
all-inaccessible result is zero, full access uses the fauna stock exactly once, and the function
consumes no RNG or stock.
The water preview uses `EffectiveWaterDemandPerPerson > 0` from the same baseline, destination
temperature, and `AridClimateAdaptation` reduction of only the heat increment that actual demand
would use there; it does not alter the stored stock. Validate finite non-negative stock and K plus a
finite positive demand before division. It is a preview only: calculating it neither reserves stock
nor consumes randomness. `P_total_j` includes both
species. Candidates must have a currently eligible land or passage edge, a habitable destination,
and finite positive movement cost; all inputs are finite and non-negative, and a zero-resource
candidate has score zero. An ordinary candidate uses the directed destination-environment formula above;
a named-passage candidate uses that passage's fixed configured cost. Both are preview values for the
current snapshot, not movement points that a band spends.
`WarnedMacroImpact` is the greatest next-transition eruption intensity affecting the destination,
or zero without a warning. It is finite in `[0, 1]`, makes the safety factor `W_j` finite in
`[0, 1]`, and is exposed on each candidate so a player can distinguish low attraction from warned
danger. It changes only the preview/ranking: the destination remains eligible, and phase 5 derives
the authoritative impact from the final tile. Because every candidate shown to sapiens is already
explored, this field leaks no hidden geography.

`internal/domain/migration.go` computes candidates while `internal/application/snapshot.go` builds
the isolated frame projection.
Each `gameapi.Band` carries at most one `MigrationCandidate` per reachable edge, sorted by
descending score and then ascending tile ID. A candidate exposes the score, its four inputs,
the warning-suitability factor, `EdgeKind`, and optional passage ID so the HUD can explain the recommendation rather than present an
opaque number.

It also exposes `CrowdingDecline`: the people this band would lose to the logistic crowding term on
its first turn at that destination, or zero where the tile has room. It is a magnitude rather than a
signed growth value, and it is computed against the destination's whole-tile population plus the
arriving band, matching the phase-3 rule that crowding uses the whole tile while growth uses the
band; the deficit fraction is zero because a non-positive logistic result is never scaled by the fed
fraction. Capacity alone cannot tell a player this — the same tile is ample for one band and lethal
for another four times its size — and crowding is the largest population loss in the model wherever
it applies, exceeding the previewed seasonal and chronic rates by orders of magnitude. It is a
derived preview like the mortality rates beside it: not serialized, recomputed after load, consuming
no RNG, and it adds no configuration input, being a projection of `LogisticGrowth` under the
already-selected `r` and `MaxCrowdingDeclineFraction`. Candidate lists are derived and never serialized; snapshot/load recomputes them from
persisted world state, including the destination biome and directed cost. Their count is bounded by
`MaxGridNeighbors + MaxPassageEdgesPerTile`, with
`MaxGridNeighbors = 8` and the closed passage table enforcing `MaxPassageEdgesPerTile = 2`. Tests
prove the formula, stable tie-break, invariance to tile/band iteration order, bounded candidate
count, zero RNG consumption, and identical rankings before save and after load. A frame therefore
contains at most `10 * len(Bands) <= 2_560` candidate values.

`SplitBand` is the only operation that increases `len(Bands)`. V1 uses a fixed `50/50` split; the
command selects a destination but carries no ratio. Before changing population, stored food, IDs,
revision, spatial-action state, or any queue, it requires
`Population >= MinSplitSourcePopulation`, where
`MinSplitSourcePopulation = 2 * MinEstablishedBand = 40`, plus `len(Bands) < MaxBands`, an unused
spatial action for the source band, and `NextBandID` below the
maximum `uint64` value so allocating it and incrementing the counter cannot wrap. At the cap it
returns typed `BandLimitReached`; at ID exhaustion it returns typed `BandIDExhausted`.
A too-small source returns typed `SplitPopulationTooSmall`; stress at or below the strict threshold
returns typed `SplitStressTooLow`; a target without a currently eligible ordinary edge returns typed
`SplitDestinationNotAdjacent`; and an adjacent land target whose current biome is uninhabitable
returns typed `SplitDestinationUninhabitable`. `GameService` maps all six split-specific failures to
stable boundary codes, and `pkg/ui` owns their player-facing copy, including “band limit reached,”
“band ID space exhausted,” and the three new eligibility explanations. All six failures cause no
mutation or RNG consumption and do not consume an ID. An accepted split allocates
exactly one new monotonic band ID. The original/source band keeps its ID and remains at the origin;
the newly allocated ID is the descendant placed at the selected destination. It gives the new
descendant `source population / 2` people using `uint32` integer division and leaves the source with the remainder, while
giving each descendant exactly half its stored FU. Thus an even count splits evenly and an odd count
gives the source one extra person. It increases the count by one, copies the exact allocation, health, technology, research,
and heritable-state values, and marks both descendants' spatial actions used for this planning
period. The source has no queued intent by precondition and neither descendant receives a new queue.
Population conservation is exact for every valid whole-person count; stored FU remains continuous
and is conserved by exact halving. Each
descendant begins with at least `MinEstablishedBand = 20` people.
Before publishing the accepted planning frame, it runs `RevealFromSapiens` for the two surviving
descendants and unions any newly exposed frontier into the world bitset; this is part of the split's
single planning mutation and revision, not another command or RNG step.
Phase-5
zero-population cleanup can reduce the count, so a later planning period may split again. Whole-band
migration never changes the count, and this build performs no automatic merge or deletion of living
bands. Multiple player split commands from different eligible source bands in one planning period
check the live count in application order: the split that reaches 256 succeeds, and each later
attempt fails independently without disturbing earlier accepted commands. A second spatial command
against either descendant of an already accepted split instead fails the spent-action check.
Player planning is complete before the archaic policy runs,
so accepted sapiens splits have first use of remaining capacity. The policy observes the resulting
live count; when it cannot split at the cap, it may still choose count-neutral whole-band migration.

The score is advisory, not command validation: selecting a lower-ranked valid tile succeeds.
`SplitBand` still requires `Stress > SplitStressThreshold`; the equality case is
`SplitStressTooLow`. `QueueMigration` may move a whole band regardless of stress. `SplitBand`
requires a currently eligible ordinary edge to a habitable land target; failures use
`SplitDestinationNotAdjacent` and `SplitDestinationUninhabitable`, respectively,
including the two-land-corner rule for a diagonal; it never performs an instantaneous water or
land-bridge crossing during the planning period. `QueueMigration` requires and consumes the band's
unused spatial action, accepts an
ordinary land edge or a currently eligible named passage and records the edge kind/passage ID with
the intent. Queued whole-band migrations revalidate the exact edge, passage eligibility, and target
after the next climate/resource update. If climate changes the destination biome while it remains
habitable, the new cost changes later rankings but does not cancel the already chosen move; cost and
rank are advisory rather than a command budget. If the edge, passage eligibility, or target is no
longer valid, that move is canceled, the band remains at its origin, and the HUD receives an event
rather than failing or partially rolling back the turn.

### Computer-controlled archaic planning

`archaic.go` implements `ArchaicPolicyAlgorithm: "ranked-pressure-v1"`. It has no hidden durable
state and consumes no `WorldRNG`; identical planning state always produces the same computer action
batch. `World.AdvanceTurn` invokes it exactly once after all accepted player planning commands and before
phase 1. It captures the living archaic band IDs present at the start of that pass in ascending
order. A band created by a computer split copies the parent's proportional allocation, acquired
technology, research target, progress vector, and heritable state, but it does not receive its own computer action
until the next turn, bounding the pass to the starting band count.

For each captured band that still exists, in ascending ID order, the policy:

1. Applies the closed basis-point `ArchaicAssignmentPreset[current region, current biome]`, authored
   against that pair's fauna profile; every preset allocates
   exactly 10,000 points across the same foraging, hunting, toolcraft, megafauna-tracking, and shelter
   categories available to sapiens and is validated with the normal assignment rules. A preset
   assigns zero to hunting/megafauna roles that its environmental profile does not support.
   This lookup uses the pre-phase-1 planning biome; climate may change the actual opportunities
   before extraction, just as for a player's accepted allocation. There is no mid-turn reallocation
   or weaker technology requirement for archaics.
2. If the band has no current research target, chooses the first missing technology whose
   prerequisites pass in the closed `ArchaicTechPriority` list and submits `ResearchTech`; if none
   is currently valid, it does nothing. On its first planning pass, an initial band's empty acquired
   set makes only the three roots valid, so the rule selects the first root in that priority list;
   it cannot select a dependent node.
3. If a one-turn macro warning affects the band's current tile, it recomputes candidates from the
   current scratch state. When a candidate in stable ranking order has warned impact strictly lower
   than the origin's, the policy queues whole-band migration to the first such candidate, never
   splits in response to that warning, and performs no further spatial action for the band. If no
   safer candidate exists, or if no warning affects the origin, it falls through to the ordinary
   pressure rule. If `Stress <= SplitStressThreshold` or there is no eligible migration candidate,
   take no spatial action. Otherwise recompute candidates after all earlier computer actions,
   choose the highest-ranked eligible ordinary-land candidate, and use `SplitBand` when the normal
   split validator, global band cap, the archaic sub-cap below, and ID allocator permit it. If no
   split can be made, it queues
   whole-band migration to the first candidate in the full ranking, which may be a named passage.
   Stable candidate order supplies every tie-break.

The assignment lookup expands the resolved region × biome fauna-profile mapping
through this complete archetype table, in assignment order Foraging, Hunting, Toolcraft,
MegafaunaTracking, Shelter:

| Fauna archetype | Foraging | Hunting | Toolcraft | MegafaunaTracking | Shelter |
| --------------- | -------: | ------: | --------: | ----------------: | ------: |
| `OM`            |   `3000` |  `2500` |    `1200` |            `1500` |  `1800` |
| `WR`            |   `3000` |  `3000` |    `1200` |             `800` |  `2000` |
| `CM`            |   `2500` |  `3500` |    `1200` |             `500` |  `2300` |
| `AS`            |   `3000` |  `4000` |    `1200` |               `0` |  `1800` |
| `HM`            |   `2800` |  `2800` |    `1200` |            `1000` |  `2200` |
| `CS`            |    `800` |  `3200` |    `1500` |            `2000` |  `2500` |
| `TI`            |   `2200` |  `3800` |    `1200` |             `800` |  `2000` |
| `BC`            |    `800` |  `3400` |    `1500` |            `1800` |  `2500` |

Every row totals exactly 10,000 basis points. Expanding through the fauna-profile table, rather than
maintaining a second independent 78-row list, is the normative `ArchaicAssignmentPreset` definition;
validation still materializes and checks all 78 region/biome results. An environmentally unsupported
MegafaunaTracking role receives zero because its mapped archetype does; a climate change after the
planning lookup does not trigger a second assignment pass.

The selected `ArchaicTechPriority`, from first to last, is
`PlantKnowledge`, `HaftedTools`, `Firecraft`, `TailoredClothing`, `Campcraft`,
`MedicinalKnowledge`, `CordageAndNets`, `Trapping`, `CoastalNavigation`. The policy scans the whole
list and chooses the first missing entry whose prerequisites are met; an unmet earlier entry does not
block a later valid one. Configuration requires every technology exactly once.

**The archaic sub-cap.** `MaxBands = 256` is a global bound across both species, and only one side of
it is automated. Archaic bands split deterministically whenever they are stressed and a candidate
exists, every turn, for 400 turns, starting from the Levant and Frangistan with the whole of Eurasia
to fill. Sapiens receive first use of remaining capacity, but only _within_ a single turn: player
planning completes before the policy runs, which decides who gets the last slot on one turn and says
nothing about the balance accumulated over hundreds. Without a floor, the archaic policy can occupy
most of the budget by mid-campaign and quietly remove the player's central dispersal verb — not as a
tuning problem but as a structural one, since a sapiens split would then fail on every remaining turn.

`BandAlgorithm: "fixed-half-global-cap-v1"` therefore adds `MaxArchaicBands`, a fixed configuration constant
with the approved initial playtest value **`96`**. The policy's split branch requires
`archaic_band_count < MaxArchaicBands` in addition to the global cap. Validation requires
`1 <= MaxArchaicBands < MaxBands`, which leaves at least `MaxBands - MaxArchaicBands = 160` places
that only sapiens can occupy. The sub-cap constrains the computer policy alone: it is not a
population cap, it never kills or merges an archaic band, and archaic bands already above it through
earlier growth remain valid. Reaching it is an expected branch to count-neutral whole-band migration,
exactly like reaching the global cap, and produces no error. Phase-5 cleanup of extinct archaic bands
frees archaic places normally.

The §12 seed corpus must report the species split of the band budget at every checkpoint, not merely
assert `len(Bands) <= MaxBands`. A run in which sapiens are unable to split for a sustained stretch
because of archaic pressure is a balance failure even when no invariant is violated.

These are policy choices, not weaker rules for archaics: assignments, research, splits, and queues
pass through the same semantic validators and have the same effects as sapiens commands. The only
extra check is unexported command authority. A planning pass emits at most three commands for each
of at most 256 starting bands. It applies them sequentially to a scratch copy of band planning state,
including assignments, research targets/progress, migration/interbreeding queues, spatial-action
markers, `NextBandID`, newly split bands, and the `LastFoodReport` values cleared by those splits.
Only a fully valid batch replaces live planning state; an unexpected validator failure returns from
`AdvanceTurn()` before climate, campaign turn, or RNG changes. `GameService.EndTurn()` then leaves
`WorldRevision` unchanged. Reaching the band cap is an expected policy branch to whole-band
migration, not an error. The computer batch and ensuing
five-phase turn are one unobservable `AdvanceTurn`: no frame is published and no save can capture the
intermediate archaic queues.

### Named passages: Wallacea and Beringia

`passage.go` defines a closed table of three bidirectional edges whose endpoints are generated land
tile IDs: two Wallacea routes (a northern island-hopping route toward New Guinea and a southern route
toward northern Australia) and one Bering Strait route between Siberia and western Alaska. Passage
validation at new game and load requires distinct in-range land endpoints, no duplicate endpoint
pair, endpoints that are not connected by `OrdinaryEdges` under the same diagonal-corner rule,
finite positive movement cost, and no more than two passage edges incident on one tile. Bands move
directly between land endpoints in one turn and never occupy an ocean tile.

The selected route costs are `4.00` for northern Wallacea, `4.50` for southern Wallacea, and
`3.00` for Beringia. The maximum ordinary adjacent-land cost is
`math.Sqrt2 * MaxMovementCost ≈ 3.889`, so both Wallacea values satisfy the required strict high-cost invariant;
the southern route is also the costlier of the two island crossings.

The two Wallacea edges require the band's `CoastalNavigation` technology. Their configured movement
cost is strictly greater than `math.Sqrt2 * MaxMovementCost`, the maximum possible ordinary
adjacent-land cost, so the attraction score communicates the difficulty even when the sapiens player
or archaic policy deliberately chooses the crossing. Before the technology is unlocked, queue
commands over those edges are rejected without mutation and the HUD's passage overlay shows the
projected `NeedsCoastalNavigation` reason for the selected band; locked edges do not appear as
migration candidates.
Technology is never lost, so an accepted Wallacea queue can later fail only if its endpoint becomes
uninhabitable or the band no longer survives/remains at the recorded origin; a save with an invalid
passage identity is rejected during load validation.

For every living band, the domain computes one fixed-order status per passage through the same
predicate used by queue validation and migration-candidate construction. `PassageStatus` is a closed
enum with this first-match precedence: `NotAtEndpoint`; `SpatialActionSpent`;
`DestinationUninhabitable`; `BeringiaClosed`
for the Bering route; `NeedsCoastalNavigation` for either Wallacea route; otherwise `Available`.
The frame projects all three `(PassageID, Status)` values even when unavailable. The HUD renders the
projected reason and never infers availability from technology, global overlay state, occupancy, or
biome. Tests cover every status and prove that `Available` is equivalent to accepting the same
passage edge for that band in the same snapshot.

The Bering Strait edge requires no technology and is eligible exactly when:

```
BeringiaOpen = LongTermTempOffset(turn) <= BeringiaOpenTemp
```

`BeringiaOpenTemp` is defined as a fraction of the campaign's own cooling depth rather than as a
free-standing temperature, because `LongTermTempOffset` reaches exactly `−LGM_cooling` at turn 400
and `LGM_cooling` is itself tuned in §12. An absolute threshold would silently strand or permanently
open the bridge every time that amplitude changed. The contract is therefore:

```
BeringiaOpenTemp = −BeringiaOpenFraction · LGM_cooling
```

with finite `0 < BeringiaOpenFraction < 1`. The approved initial playtest value is **`0.85`**. The
eligibility test uses the complete `LongTermTempOffset`, not its smoothstep component alone. Across
the 401 supported turns at the selected coefficients, the bridge first opens at turn 178
(`38,300 BP`, Middle era), remains open through turn 194, closes for turns 195–230 as the orbital
term lifts the offset above the threshold, and reopens at turn 231 (`31,900 BP`, Late era) through
turn 400. At the first opening, `LongTermTempOffset ≈ -5.193°C` is just below
`BeringiaOpenTemp = -5.185°C`; at turn 204 it is approximately `-5.131°C`, so the bridge is closed.
Because the smoothstep trend is monotonic and the decaying orbital term vanishes at
`u = 1`, `−LGM_cooling <= −BeringiaOpenFraction · LGM_cooling` holds at turn 400 for any valid
fraction, so a campaign that reaches its end always finds the bridge open.

Configuration validation evaluates `BeringiaOpen` across all 401 supported turns and requires that it
becomes true at some turn no later than 300 and remains true at turn 400; a catalog that never opens
the bridge, or opens it only in the final few turns, is rejected rather than shipped as an
unreachable destination. The orbital term can carry `LongTermTempOffset` back above the threshold
near the crossing, so the bridge may close and reopen; that is intended long-cycle behavior, and
§7's phase-4 revalidation already cancels a queued crossing that a closure invalidates.
Only `LongTermTempOffset` participates: seasonal temperature and
`ClimateNoise` cannot flicker the bridge or make its state depend on `WorldRNG`. This is a named
long-term sea-level proxy, not a global dynamic-coastline system; all rasterized land and water tiles
remain fixed. If the bridge closes between planning and phase-4 resolution, revalidation cancels the
queued crossing and emits a non-fatal event. A band already on either land endpoint remains there.

`Frame.Passages` contains exactly three freshly allocated `gameapi.Passage` values with passage ID,
kind, endpoints, movement cost, optional `RequiredTech`, and global `ClimateOpen`. It does not encode
band-specific availability. For the selected band, UI derives exactly one typed status in priority
order: `BeringiaClosed` when `ClimateOpen == false`, `NeedsCoastalNavigation` when the required tech
is absent, otherwise `Available`. The authoritative candidate/command checks perform the same test in
the domain; a renderer/UI calculation can explain availability but cannot grant it. The map draws the
two Wallacea routes and Beringian route as overlays only to the extent permitted by the explored-
endpoint masking rule in §8, coloring visible portions relative to the selected band when one exists
and otherwise showing their neutral requirement/state. Tests cover both Wallacea routes
locked/unlocked for bands with different technologies, UI/domain availability agreement, the
high-cost invariant, Beringian threshold equality and both sides, immunity to abrupt/seasonal/noise changes,
the selected turn-178 opening/turn-195 closure/turn-231 reopening trajectory, command-time and
phase-4 revalidation, cancellation without partial movement, no ocean occupancy,
passage candidate ranking, zero RNG consumption, and identical passage state/rankings across save
and reload.

The campaign's primary objective remains sapiens dispersal out of Africa, but no route is privileged.
`SapiensEstablishedRegions` is a persisted, sorted, duplicate-free set of `Region` values. During
phase 5, after acute hazards and zero-population cleanup, every region containing a surviving
`HomoSapiens` band with at least `MinEstablishedBand` living people at the end of the completed turn
is inserted into the set. A band reduced below the threshold by that turn's acute event does not
establish a new region.

The first insertion of either `Arabia` or `Levant` also selects a sourced Field Notes context entry
about last-dispersal founder-population and effective-population estimates and their uncertainty.
This presentation trigger is derived from the newly latched achievement; it adds no founder-flow
counter, lineage flag, gameplay target, event RNG, or saved state beyond the ordinary achievement.
After a successful `EndTurn`, `game_scene` compares the prior and replacement frame's established
sets and gives that entry one completed-turn context focus; loading an already-latched achievement
does not replay it.

`MinEstablishedBand` is a fixed positive integer configuration constant; the approved initial
playtest value is **`20`**. It distinguishes a foothold that survived its arrival turn from a
remnant that merely passed through, without demanding a large settled population from a band that has
just crossed Wallacea or the Bering land bridge. Because this threshold is the only quantitative gate
on every regional achievement, and therefore on the turn-400 victory test, it is a balance constant
in §12 rather than an implementation detail: too high and a legitimate dispersal cannot be recorded,
too low and a single doomed scouting band claims a destination. Validation requires
`MinEstablishedBand >= 1` so a zero-population band can never establish anything. Entries latch permanently: later migration, decline, or local extinction
does not erase the historical achievement. The enum bounds the set to at most `RegionCount` entries;
insertion is idempotent and consumes no randomness.

`DestinationRegions = {Frangistan, SouthAsia, YellowRiverBasin, Sahul, Beringia}` is a closed set used
only for campaign outcome. Establishment in **any one** destination proves successful out-of-Africa
dispersal; additional destinations improve the end-scene achievement summary but are not required
for victory. These are destination achievements, not a claim that each belongs to a distinct route:
for example, a single eastward dispersal may establish both South Asia and the Yellow River Basin.
The HUD and end scene show every established region, not a single boolean. An archaic band already
living in a destination cannot add to `SapiensEstablishedRegions`; archaic survival and distribution
are reported separately. Sapiens extinction remains an immediate loss even if archaic bands survive
or an earlier sapiens destination achievement had latched.

### Campaign clock and terminal states

The campaign has **400 turn transitions** from 80,000 BP to 20,000 BP, divided into four fixed
100-turn **campaign eras** whose turns represent progressively shorter calendar intervals:

| Campaign era | Turn interval | Date interval    | Years advanced by each completed turn |
| ------------ | ------------- | ---------------- | ------------------------------------: |
| Early        | `[0, 100)`    | 80,000–50,000 BP |                                   300 |
| Middle       | `[100, 200)`  | 50,000–35,000 BP |                                   150 |
| Late         | `[200, 300)`  | 35,000–25,000 BP |                                   100 |
| Final        | `[300, 400]`  | 25,000–20,000 BP |                                    50 |

The player-facing era labels always include their date ranges; the short names are navigational,
not assertions of formal archaeological periodization. These campaign eras are wholly independent
of the turn-resolution pipeline's five phases.
Turn 400 is the inclusive terminal endpoint of the Final era; the 100 Final-era transitions begin
at turns 300 through 399.
No campaign-era boundary resets or grants resources, technology, genetics, movement, hazards,
season, assignments, or actions.

`CampaignDate(turn)` is the single authoritative piecewise-linear conversion:

```text
EraForTurn(t) = Early  for   0 <= t < 100
                Middle for 100 <= t < 200
                Late   for 200 <= t < 300
                Final  for 300 <= t <= 400

CurrentYearBP(t) = EraStartYearBP - YearsPerTurn * (t - EraStartTurn)
CalendarProgress(t) = (80_000 - CurrentYearBP(t)) / 60_000
```

It maps turns `0`, `100`, `200`, `300`, and `400` exactly to 80,000, 50,000, 35,000, 25,000,
and 20,000 BP with no discontinuity or rounding. Application frame projection requires
`0 <= Turn <= 400`
and derives both campaign year and era from this table; UI, metadata, climate,
and reports call the same function rather than reimplementing it. Rates described as “per turn”
remain one application per decision turn and are not multiplied by `YearsPerTurn`; the variable
clock provides finer player decision and technology resolution later in the campaign, not literal
annual numerical integration.

Climate's 12-turn seasonal cycle remains an orthogonal gameplay abstraction. It is not a claim that
one turn represents one literal month or that twelve turns represent a literal year, and changing
campaign era neither resets nor stretches the cycle. Turns modulo 12 map `0–2` to `SeasonWarm`,
`3–5` to `SeasonCooling`, `6–8` to `SeasonCold`, and `9–11` to `SeasonWarming`, aligning the four
three-turn seasonal phases with the temperature term without claiming Northern Hemisphere calendar
names for a map spanning both hemispheres.

The HUD timeline uses `CalendarProgress(Turn)`, not `Turn / 400`. Major ticks are fixed at
10,000-year intervals—80,000 BP, 70,000 BP, 60,000 BP, 50,000 BP, 40,000 BP, 30,000 BP, and
20,000 BP—at equal one-sixth positions along the rail. Tick dates need not coincide with integer
turns. Subtle era boundaries appear at 50,000, 35,000, and 25,000 BP, so the marker visibly advances
farther per early turn and more finely per late turn. A normal turn-400 resolution reaches the final
tick; an immediate extinction leaves the terminal marker at its exact `CampaignDate`. Destination
achievements do not become markers on this rail: it answers “when are we?” while the established-
region display answers “where have we reached?”

Climate is a pure function of calendar progress and the persisted world seed. For
`t = turn` and `u = CalendarProgress(t)`:

```
smoothstep(u) = 3u² − 2u³
LongTermTempOffset(t) = −LGM_cooling · smoothstep(CalendarProgress(t))
                        + orbital_amplitude · (1 − CalendarProgress(t)) · OrbitalSin[t]
SeasonalTempOffset(t) = seasonal_amplitude · SeasonalCos[t mod 12]
ClimateNoise(seed, t) = noise_amplitude · UnitNoiseV1(seed, t)
GlobalTempOffset(t)   = LongTermTempOffset(t)
                        + SeasonalTempOffset(t)
                        + ClimateNoise(seed, t)
AbruptClimateOffset(tile, t) =
    sum_e AuthoredClimatePulse(e, CurrentYearBP(t)) · RegionalAbruptWeight[Region(tile)]
HabitatTempOffset(tile, t) = LongTermTempOffset(t)
                             + SeasonalTempOffset(t)
                             + AbruptClimateOffset(tile, t)
TileTempOffset(tile, t)    = HabitatTempOffset(tile, t) + ClimateNoise(seed, t)
```

`OrbitalSin[t] = sin(8π * CalendarProgress(t))` and
`SeasonalCos[s] = cos(2πs / 12)` are generation definitions only; runtime uses their checked-in
exact-bit tables.

The selected baseline coefficients are `LGM_cooling = 6.1°C`, `orbital_amplitude = 1.5°C`,
`seasonal_amplitude = 2.0°C`, and `noise_amplitude = 0.35°C`. The last value satisfies the
subordinate-noise constraint `noise_amplitude <= seasonal_amplitude / 4`.

The orbital term supplies long-cycle warm/cool pressure while decaying to zero; the long-term trend
therefore reaches exactly `−LGM_cooling` on turn 400. `UnitNoiseV1` is a versioned, domain-separated,
counter-based hash in `rng.go` that maps `(world seed, turn)` deterministically into `[-1, 1]`; it
does not consume or depend on `WorldRNG`. Configuration validation enforces finite non-negative
amplitudes and `noise_amplitude <= seasonal_amplitude / 4`, keeping stochastic variation subordinate
to the readable seasonal signal. Each source-backed climate-pulse entry has a stable ID, start,
peak, and end date in BP, a signed finite amplitude, and a smooth attack/decay envelope evaluated in
calendar years. Entries are summed in stable ID order and configuration requires
`abs(AbruptClimateOffset) <= MaxAbruptClimateOffset = 2.5°C` independently for every region and
supported turn. The seven pulse windows do not overlap, so the worst-case regional sum is the
largest single sampled weighted pulse. GI-14's authored peak falls between supported turns: at turn
87 (`53,900 BP`) its envelope is `smoothstep(0.9) = 0.972`, and its `+2.5°C` amplitude times
Frangistan's `AbruptWeight` of `1.00` therefore reaches a sampled maximum of `2.43°C`. The selected
catalog has `0.07°C` of sampled-offset headroom. At weight `1.00`, GI-14's amplitude could rise only
to `2.5 / 0.972 ≈ 2.572°C` before reaching the cap; configuration may tune a pulse amplitude,
regional weight, or the cap independently provided the complete 401-turn × 13-region validation
still passes and the owning rule and Appendix C remain synchronized. Regional weights keep a strong
North Atlantic signal from becoming an equally strong East African or Sahul temperature change;
they are authored scientific/gameplay data, not
seeded noise. The v1 catalog uses irregular authored pulses rather than treating abrupt glacial
changes as another perfectly periodic sine wave. The instantaneous turn-400 `TileTempOffset` may
differ from the long-term target by at most
`MaxAbruptClimateOffset + seasonal_amplitude + noise_amplitude`. Beringian passage eligibility is
the only geographic rule derived from `LongTermTempOffset`; it changes a named graph edge, never the
rasterized land/water mask. Abrupt, seasonal, and noise components cannot flicker the bridge.

For `StartBP > PeakBP > EndBP`, a pulse is zero outside its date window, uses
`smoothstep((StartBP - y) / (StartBP - PeakBP))` from start through peak, and uses
`1 - smoothstep((PeakBP - y) / (PeakBP - EndBP))` after peak through end; multiply that envelope by
the entry's amplitude and regional weight. Validation rejects invalid dates or any catalog whose
stable-order sum exceeds the configured per-region bound at any of the 401 supported campaign turns rather than hiding bad data with a
runtime clamp.
Every catalog pulse must affect at least one supported turn. A scientifically relevant excursion
shorter than the overlapping campaign-era turn span is represented by a source-backed one-turn
average amplitude and labeled as temporal compression in Field Notes, not silently missed between
two sampled dates.

The selected v1 catalog uses seven representative Greenland Interstadial warming onsets from
Rasmussen et al. (2014). That chronology is in years before AD 2000 (`b2k`); subtracting 50 produces
the campaign's conventional years BP. For compound interstadials, the table uses the onset of the
oldest named subdivision—GI-14e, GI-12c, and GI-8c—as the onset of the displayed GI-14, GI-12, and
GI-8 pulse. Amplitudes and the smooth gameplay windows are balance abstractions, not reconstructed
local temperature series:

| Stable ID / display label | Source onset (b2k) | `StartBP` | `PeakBP` |  `EndBP` | Amplitude |
| ------------------------- | -----------------: | --------: | -------: | -------: | --------: |
| `GI19_2` / GI-19.2        |           `72,340` |  `72,290` | `71,990` | `70,790` |  `+1.5°C` |
| `GI18` / GI-18            |           `64,100` |  `64,050` | `63,750` | `62,550` |  `+1.5°C` |
| `GI14` / GI-14            |           `54,220` |  `54,170` | `53,870` | `52,670` |  `+2.5°C` |
| `GI12` / GI-12            |           `46,860` |  `46,810` | `46,660` | `46,060` |  `+2.0°C` |
| `GI8` / GI-8              |           `38,220` |  `38,170` | `38,020` | `37,420` |  `+2.0°C` |
| `GI5_2` / GI-5.2          |           `32,500` |  `32,450` | `32,350` | `31,950` |  `+1.5°C` |
| `GI2_2` / GI-2.2          |           `23,340` |  `23,290` | `23,240` | `23,040` |  `+1.0°C` |

The attack spans one turn at the campaign era containing the onset and the decay spans the next
four such turn lengths; the exact dates above are checked-in values so a later clock change cannot
silently retime them. Field Notes label this envelope as temporal compression and cite the onset,
not the gameplay peak or end, as the scientific anchor.

All seven entries use the `AbruptWeight` column of the **regional climate-response table**. Both of
`ClimateAlgorithm`'s per-region vectors share one row key and one owning contract, so they are one
table with two columns rather than two tables that could drift apart when a region is added or
renamed:

| Region             | `AbruptWeight` | `AridityWeight` |
| ------------------ | -------------: | --------------: |
| East Africa        |         `0.10` |          `0.55` |
| Rest of Africa     |         `0.15` |          `0.90` |
| Arabia             |         `0.20` |          `1.00` |
| Levant             |         `0.50` |          `0.60` |
| Frangistan         |         `1.00` |          `0.50` |
| Central Asia       |         `0.65` |          `0.75` |
| South Asia         |         `0.20` |          `0.65` |
| Southeast Asia     |         `0.10` |          `0.30` |
| East Asia          |         `0.35` |          `0.55` |
| Yellow River Basin |         `0.40` |          `0.70` |
| Sahul              |         `0.05` |          `0.80` |
| Siberia            |         `0.55` |          `0.45` |
| Beringia           |         `0.35` |          `0.40` |

`AbruptWeight` scales the GI pulses above; `AridityWeight` scales the long-term moisture offset
below. The two columns are deliberately uncorrelated: Frangistan takes the strongest North Atlantic
thermal signal at `1.00` but only middling desiccation at `0.50`, while Arabia inverts that. Reading
one region's full climate response is now one row.

Validation requires exactly seven uniquely ordered entries, exactly `RegionCount = 13` finite
non-negative weights no greater than one, exact b2k-to-BP differences of 50 years, valid windows,
and the `2.5°C` combined regional cap at every one of the 401 campaign turns.

### Long-term moisture and derived climate epochs

Temperature is not the only climate axis. `ClimateAlgorithm` also produces a **long-term moisture
offset**, so desertification can advance from the south while tundra advances from the north. Its
construction deliberately mirrors `LongTermTempOffset` — a directed trend plus a damped oscillation
— rather than introducing a second style of curve:

```text
LongTermMoistureOffset(t) = -aridification_amplitude * smoothstep(CalendarProgress(t))
                            + precession_amplitude * (1 - CalendarProgress(t)) * PrecessionSin[t]
TileMoistureOffset(tile, t) = LongTermMoistureOffset(t)
                              * RegionalAridityWeight[Region(tile)]
EffectiveMoisture(tile, t)  = clamp01(BaseMoisture(tile) + TileMoistureOffset(tile, t))
```

`PrecessionSin[t] = sin(2π * (60_000 * CalendarProgress(t) / precession_period) +
precession_phase)` is likewise a generation definition only; runtime uses the mandatory 401-entry
exact-bit table.

`ClassifyBiome(latitude, elevation, EffectiveMoisture, HabitatTemperatureC)` keeps its existing
signature; only the moisture argument stops being constant. `BaseMoisture` remains the rasterizer's
fixed per-tile output, rivers included as high-moisture corridors. Like every other climate
component, moisture is derived from `turn` and fixed geography, consumes no `WorldRNG` draw, and is never
serialized.

No independent gameplay subsystem reads raw moisture. Existing ecological consumers inherit its
effect through `V` or biome: `BaselineKCurve(V)`, `MovementCurve(V)`, regional fauna profiles, the
endemic-disease table, and Appendix B.2's biome x season resource targets — which include water, so
shrinking water availability is emergent rather than a new stock or hazard system.

**The oscillation is deliberate.** Aridity across this window was not a ramp. The precession term at
`precession_period = 21_000` years reproduces the insolation-driven African monsoon cycle, placing
wetter intervals at the campaign's opening and near 62,000, 42,650, and 24,150 BP. The `(1 - u)` damping serves the same two
purposes it serves in the temperature curve: it guarantees the trend reaches exactly
`-aridification_amplitude` at turn 400, and it reflects that monsoon response to insolation is
suppressed under full glacial conditions — the strong African Humid Period followed deglaciation,
after this campaign ends. Because the curve oscillates, tiles near a moisture threshold reclassify
more than once and the derived epoch below can return to an earlier value. That is intended:
`Season` already cycles, and an interstadial that visibly reopens habitat is the point.

All regions dry, at authored rates given by the `AridityWeight` column of the regional
climate-response table above. Arabia and the Saharan interior carry the strongest desiccation because they are the corridor the
campaign is about; everwet Southeast Asia is the most buffered, which makes it a refugium rather
than a route. Weights are constrained to `[0, 1]`, so no region gets wetter as others dry. Glacial
pluvials and the interhemispheric seesaw that wetted southeastern Africa during Heinrich stadials
would need signed weights; that is deliberately deferred, as is any coupling between the seven GI
pulses and moisture.

**Derived climate epochs.** `AridityIndex(t) = clamp01(-LongTermMoistureOffset(t) /
aridification_amplitude)` normalizes the composite curve to `[0, 1]`. For
`x = AridityIndex(t)`, two authored thresholds split it into three named epochs:

| Epoch            | Nominal `AridityIndex` band | First entered       | Reference trajectory               |
| ---------------- | --------------------------- | ------------------- | ---------------------------------- |
| `HumidOptimum`   | `x < 0.25`                  | turn 0, 80,000 BP   | returns once at turn 42, 67,400 BP |
| `AridTransition` | `0.25 <= x < 0.96`          | turn 26, 72,200 BP  | re-entered at turn 79, 56,300 BP   |
| `GlacialMaximum` | `0.96 <= x`                 | turn 258, 29,200 BP | holds through turn 400             |

Thresholds are anchored to chronology rather than to even pacing: the first `0.25` crossing falls
near the MIS 4 onset and the `0.96` crossing near the MIS 2 onset. The Last Glacial Maximum proper,
conventionally 26,500–19,000 BP, sits inside the third epoch rather than beginning it.
The bands above are nominal: after turn 0, the displayed label may remain in an adjacent band's
`EpochHysteresis` hold interval under the state machine below.

The trajectory column is the model's actual behavior at the selected values, not an aspiration. The
caption enters `AridTransition` at turn 26, returns to `HumidOptimum` for 37 turns as the first
precession maximum passes, then leaves it for good at turn 79. That return is the oscillation doing
its job: a wetter interval reopens habitat, and the epoch name says so. Because these dates derive
from Initial-status amplitudes and phase, the balance pass may move them; the validation rules below
constrain the shape, not the dates.

**The epoch is a label, never a cause.** No rule in `internal/domain` branches on it. Carrying
capacity, movement cost, water, fauna, and hazards respond to `EffectiveMoisture` through biome
classification, exactly as they already respond to temperature. The epoch exists only for §8's
render grade and for Field Notes, and therefore adds no save field, no revision increment, and no
simulation input. This preserves the decoupling that already keeps abrupt, seasonal, and noise
components out of Beringian eligibility.

Because `AridityIndex` oscillates, a crossing near a boundary could flicker the displayed name
between adjacent turns. `ClimateEpoch(t)` therefore folds the index trajectory from turn `0` through
turn `t` with `EpochHysteresis = 0.02`. Turn `0` is `HumidOptimum` for `x < lower`,
`AridTransition` for `lower <= x < upper`, and `GlacialMaximum` otherwise. For each later turn, with
`x = AridityIndex(t)`, `lower` and `upper` the two thresholds, and `h` the hysteresis:

```text
from HumidOptimum:
    x >= upper + h -> GlacialMaximum
    x >= lower + h -> AridTransition
    otherwise      -> HumidOptimum
from AridTransition:
    x >= upper + h -> GlacialMaximum
    x <= lower - h -> HumidOptimum
    otherwise      -> AridTransition
from GlacialMaximum:
    x <= lower - h -> HumidOptimum
    x <= upper - h -> AridTransition
    otherwise      -> GlacialMaximum
```

The direct outer-to-outer cases make the definition total even if a future curve crosses both bands
in one turn. At configuration load, `ClimateAlgorithm` precomputes this bounded 401-entry derived
table in turn order. Frame projection reads the indexed result, so starting directly at a turn,
loading a save, and advancing normally produce the same caption without serializing epoch state.
Hysteresis affects the caption only; §8's continuous grade reads `AridityIndex` directly and is
unaffected.

`HumidOptimum` is named for the moisture state this model produces, not for an interglacial. The
campaign opens at 80,000 BP, roughly 45,000 years after MIS 5e ended, so no part of it is
interglacial. Field Notes must say so, must present the compressed single drying trend as a gameplay
abstraction rather than a reconstruction, and must note that MIS 3 was an oscillating and
comparatively mild interval rather than a steady desiccation. Climate entries cite Lisiecki and
Raymo (2005) for the marine isotope stage framework and Clark et al. (2009) for the Last Glacial
Maximum definition.

Validation requires exactly `RegionCount = 13` finite `RegionalAridityWeight` entries in `[0, 1]`;
finite `aridification_amplitude > 0`, finite `precession_period > 0`, and
`precession_amplitude <= aridification_amplitude / 3`, keeping the oscillation subordinate to the
directed trend exactly as `noise_amplitude <= seasonal_amplitude / 4` keeps noise subordinate to
season; a finite in-domain `EffectiveMoisture` for all 6,144 tiles at all 401 supported turns; two
ordered thresholds in `(0, 1)` separated by more than `2 * EpochHysteresis` and satisfying
`upper + EpochHysteresis < 1` and `lower - EpochHysteresis > 0`, so neither outer epoch is made
unreachable by its own hysteresis band against the clamped index; a reference trajectory entering
all three epochs at least once across the 401 turns; `MinBiomeDwellTurns = 15`, so no tile may hold
a biome for fewer than fifteen consecutive turns before reclassifying again; and `BiomeChurnCap =
12` — no tile may change biome more than twelve times across the campaign. Because the candidate
classifier and the persistence fold read seed-independent `HabitatTemperatureC`, these gates are
checked exhaustively over exactly `6,144 * 401` tile-turn states; no seed corpus substitutes for
that proof. The dwell floor is enforced by the derived table rather than by mutating runtime state;
a configuration whose resulting table exceeds the churn cap is rejected. The dwell floor is derived
from Appendix B.2 rather than chosen. Closing 90% of a
toward-cap gap at gap-recovery fraction `g` takes `ceil(ln(0.10) / ln(1 - g))` turns, so

```text
MinBiomeDwellTurns = ceil(ln(0.10) / ln(1 - regen_rate_fauna))
```

which at the fauna fraction `0.15` gives `ceil(14.17) = 15`. A tile reclassifying faster than that
could never refill its stock at either cap and would be permanently poorer than either steady
state. `BaselineKCurve(V)` removes most of the capacity half of this problem because a
tile within a vegetation band no longer sits at one constant. The selected `0.45` capacity step and
geographic biome factors still change discretely, and flora, fauna, and water caps remain keyed to
the discrete biome through Appendix B.2, so those still step. `BiomeChurnCap` alone cannot prevent
this — twelve changes across 401 turns
bounds only the mean dwell at roughly 31 turns and permits arbitrarily fast individual flips. If
the fauna regeneration fraction changes, recompute and recheck this floor; it cannot move
independently.

### Macro-environmental episodes

`ClimateAlgorithm` owns the climate-pulse entries above. `MacroEventAlgorithm: "bounded-regional-v1"`
owns a closed, source-cited catalog of major volcanic episodes. Both catalogs
are deterministic historical scenario data, not a random event deck: episode activation is derived
from `CampaignDate(nextTurn)` and consumes no `WorldRNG` draw.
`MaxClimatePulses = 32` and `MaxMacroEpisodes = 16` bound catalog validation, frame summaries, and
per-turn scans; all warned/current/elapsed macro summaries together cannot exceed
`MaxMacroEpisodes`.
An episode whose event date lies within the calendar interval crossed by a completed turn activates
exactly once even when campaign-era turn lengths differ. For a transition `t -> t+1`, the half-open
test is `CurrentYearBP(t) > EventYearBP >= CurrentYearBP(t+1)`; the upper date's strict inequality
prevents a boundary event from replaying on the following turn. Climate-pulse windows may cover multiple
turns through `AbruptClimateOffset`; a major eruption's aerosol and ash disruption is aggregated
into one turn because its atmospheric lifetime is far shorter than this game's 50–300-year turns.
It never changes `LongTermTempOffset`, Beringian eligibility, the fixed land mask, or geographic
region membership.

Toba is retained at `73,880 BP` as a source-cited timeline and Field Notes marker, not as an active
`MacroEpisode`. It has no warning and changes no population, health, resources, habitat, climate,
biome, passage, or RNG state. Its note explains the debated catastrophe hypothesis, distant ash,
evidence of continued human occupation and flexible subsistence, and the uncertainty around any
demographic effect. This preserves its educational value without projecting an unsupported lethal
effect onto the player's distant East African bands.

The selected active v1 catalog contains exactly one episode:
`CampanianIgnimbrite` at `EventYearBP = 39,850`, with immutable Campi Flegrei epicenter
`(14.14°E, 40.83°N)` and three
non-overlapping geographic impact masks: a **direct zone**, a **proximal ash zone**, and a wider
**aerosol/ash zone**. The date is the selected central value from the source-backed `39.85 ka`
estimate. Masks use the exact evidence-shaped polygon catalog below, compiled into checked-in tile
bitsets rather than circular radii or gameplay-region boundaries. Because a map tile is much
larger than a pyroclastic-flow path, direct-zone intensity explicitly abstracts sub-tile exposure
and dispersed bands; it never means that every organism or person in the tile was inside the flow.
Changing this active catalog after release requires a new macro-event algorithm version. V1 does not
invent unrecorded eruptions from the world seed.

The catalog separates the locally mapped Campanian pyroclastic-current footprint from the much wider
eastward tephra record. Its outer extent follows the documented central/eastern Mediterranean,
Balkan–Lower Danube, and Pontic–Don evidence without drawing one convex envelope through every gap.
This is still a conservative gameplay interpretation of a sparse deposit record, not a reconstructed
ash-thickness surface. The source basis is
[Smith et al. (2016)](https://ora.ox.ac.uk/objects/uuid%3A66ec6abc-0d75-46ca-9cf8-edfe256d1cde),
[Pyle et al. (2006)](https://doi.org/10.1016/j.quascirev.2006.06.008), and
[Scarpati et al. (2020)](https://doi.org/10.3389/feart.2020.543399).

Vertices are `(longitude, latitude)` in degrees, ordered clockwise; the closing edge from the final
vertex to the first is implicit:

```text
DirectCampanianPlain = [
    (12.7°E, 41.5°N),
    (14.8°E, 42.3°N),
    (16.5°E, 41.2°N),
    (15.8°E, 39.8°N),
    (12.8°E, 40.0°N),
]

ProximalSouthernItaly = [
    ( 9.0°E, 44.5°N),
    (14.0°E, 45.0°N),
    (19.0°E, 43.0°N),
    (20.0°E, 39.0°N),
    (17.0°E, 36.0°N),
    (12.0°E, 36.5°N),
    ( 9.0°E, 40.0°N),
]

WideMediterranean = [
    (13.5°E, 40.0°N),
    (18.0°E, 42.0°N),
    (27.0°E, 40.0°N),
    (34.0°E, 36.0°N),
    (32.0°E, 31.0°N),
    (22.0°E, 30.0°N),
    (15.0°E, 34.0°N),
    (13.0°E, 38.0°N),
]

WideBalkanLowerDanube = [
    (15.0°E, 44.0°N),
    (20.0°E, 48.0°N),
    (28.0°E, 49.0°N),
    (31.0°E, 46.0°N),
    (30.0°E, 42.0°N),
    (25.0°E, 39.0°N),
    (19.0°E, 39.0°N),
    (15.0°E, 41.0°N),
]

WidePonticDon = [
    (28.0°E, 48.0°N),
    (33.0°E, 52.5°N),
    (41.5°E, 53.0°N),
    (44.0°E, 50.0°N),
    (42.0°E, 46.0°N),
    (35.0°E, 43.5°N),
    (30.0°E, 45.0°N),
]
```

Polygon vertices are stored as signed integer deci-degrees (`PolygonCoordinateScale = 10`); every
listed decimal is therefore exact. Step 4 represents projected tile-center coordinates as rationals
with the fixed `95`/`63` grid denominators and uses cross-multiplied integer orientation tests, with
boundary points inside. It assigns zones by strict precedence:
`Direct` when inside `DirectCampanianPlain`; otherwise `Proximal` when inside
`ProximalSouthernItaly`; otherwise `Wide` when inside the union of the three `Wide*` lobes; otherwise
`Unaffected`. Thus source polygons may overlap while the compiled masks cannot. Water centers may
carry a presentation bit, preserving the marine dispersal shape, but have no band, resource, habitat,
or mortality effect. Runtime uses only the generated masks. Step 4 freezes their exact bitsets
and checksums. Under the locked 96×64 projection and precedence, their raw cardinalities are exactly
`1` Direct, `16` Proximal, and `72` Wide tile centers, including water presentation bits. Validation
also proves that Direct contains the epicenter's tile, that tile is Frangistan land, and no zone covers
every land tile of any named region.

For each active eruption and tile, `macroevent.go` derives finite values in stable episode-ID order:

```text
impact_intensity(tile, episode) in [0, 1]
resource_factor[tile, resource] in (0, 1]
habitat_factor[tile] in (0, 1]
refugium_mitigation[tile] in [0, MaxRefugiumMitigation]
raw_macro_loss_fraction = impact_intensity * DirectLossScale[episode]
macro_loss_fraction = min(MaxMacroLoss,
                          raw_macro_loss_fraction * (1 - refugium_mitigation))
macro_health_loss = impact_intensity * HealthLossScale[episode]
                    * (1 - refugium_mitigation)
```

The selected zone intensity is `1.00` for Direct, `0.55` for Proximal, `0.20` for Wide, and zero
outside all masks. The selected one-turn environmental factors are:

| Zone       |  Flora |  Fauna |  Water | Habitat |
| ---------- | -----: | -----: | -----: | ------: |
| Direct     | `0.25` | `0.35` | `0.75` |  `0.50` |
| Proximal   | `0.50` | `0.60` | `0.85` |  `0.70` |
| Wide       | `0.80` | `0.85` | `0.95` |  `0.90` |
| Unaffected | `1.00` | `1.00` | `1.00` |  `1.00` |

For Campanian Ignimbrite, `DirectLossScale = 0.40` and `HealthLossScale = 0.25`. The shared caps are
`MaxMacroLoss = 0.45` and `MaxRefugiumMitigation = 0.20`, with
`refugium_mitigation = MaxRefugiumMitigation * NaturalShelter(tile)`. Thus the configured episode's
raw direct-zone population loss is `0.40` before refugium and never reaches the broader `0.45` cap.
These are gameplay balance values, not historical mortality estimates.

`MaxMacroLoss` is finite in `[0, 1)`; every resource/habitat factor is strictly
positive and no greater than `1`, direct/health scales are finite and non-negative, and
`MaxRefugiumMitigation` is finite in `[0, 1)`. Configuration therefore cannot sterilize a tile, guarantee a band's extinction, set
carrying capacity to zero, or describe an entire named region as lifeless. The direct zone may be
devastating, but impact falls across the proximal and wider masks. Fixed natural shelter supplies a
modest, capped refugium mitigation for direct mortality and health damage; shelter workforce,
technology, and heritable traits do not provide a generic eruption shield. Stored food remains
valuable because the episode suppresses collection rather than deleting reserves, while migration
offers the main way to leave a warned impact zone. Archaic and sapiens bands use identical impact
rules.
As elsewhere in the pipeline, a migrating band gathers at its origin: a warned departure still
experiences that origin's phase-2/3 resource shock, but avoids direct phase-5 impact when its final
tile lies outside every active impact mask.

During phase 2, `habitat_factor` multiplies `EcologicalK` once and the resource-specific factors
multiply flora, fauna, and water caps once before stock clamping and toward-cap regeneration. No
second factor is hidden in forage, hunt, consumption, degradation, or hazard rates. When the episode
ends, the factors return to one; stocks and capacity recover only through the existing toward-cap
and degradation rules, so aftermath persists without an ash counter, destroyed-tile flag, or new
resource ledger. Volcanic cooling is not added to `GlobalTempOffset` or `TileTempOffset` and cannot reclassify a biome
for an entire multi-century turn.

During phase 5, before the ordinary acute-event distribution, apply the eruption's
`macro_loss_fraction` once to every surviving band on its final tile, record the result in the
separate macro-event field of the mortality breakdown, and subtract `macro_health_loss` once from
survivors' health with a zero floor. For each active episode in stable order, compute
`M_macro_raw = P_before_macro * macro_loss_fraction` and
`P_after_macro = RoundPopulation(P_before_macro - M_macro_raw)`; the persisted applied macro loss is
`P_before_macro - P_after_macro`, and the ordinary acute formula then uses
`P_after_macro`. This deterministic impact consumes no hazard draw and does not
compete under `MaxAcuteProbability`; the following ordinary acute event acts on the remaining
population, so the two percentage losses compose rather than add into a hidden 100% kill. One typed
episode alert is emitted globally, plus one bounded per-band impact event for each affected band.
All use the existing `MaxEvents = 128` FIFO feed.

The planning frame immediately before the active Campanian eruption exposes a one-turn warning for explored affected
tiles and resident sapiens bands. This is a coarse-turn gameplay abstraction for environmental
precursors, not a claim of centuries-early prediction. It reveals neither an unexplored epicenter
nor affected hidden archaic bands. Warned, active, and elapsed episode summaries, current impact
factors on explored tiles, and any player-relevant warning are derived frame values; only resulting population, health, resource
stocks, mortality breakdowns, and bounded historical event-feed entries persist. Field Notes must
distinguish Campanian's destructive local flows from wider, shorter-lived ash/aerosol effects and
must keep Toba's no-effect marker separate from the active episode catalog.

Before implementing this slice, step 4 closes the catalog/mask schemas, stable IDs and ordering,
source and completeness requirements, and the generated Campanian bitset fixtures and checksums. The
polygon vertices and precedence are already Locked above. The release data must preserve the selected
non-zero habitat/resource floors and the strict below-one mortality cap above. A combined-loss fixture
starts from any finite positive post-demographic population and proves that the selected macro factor
followed by the maximum selected acute severity leaves a finite positive survivor; caps may not
conceal invalid arithmetic.

There are two terminal paths, resolved in strict precedence after phase-5 cleanup and achievement
latching:

1. **Immediate loss:** if total `HomoSapiens` population is zero after the completed turn, set the
   persisted result to extinction. This check wins even when the completed transition reaches turn
   400 and even when an earlier destination achievement had latched.
2. **Turn-400 resolution:** only when at least one sapiens band survives and the completed turn is
   400, set victory if `SapiensEstablishedRegions ∩ DestinationRegions != ∅`; otherwise set the
   persisted result to “dispersal failed.” Every achievement becomes permanently true under the
   species-specific establishment rule above.

Reaching a destination early records the achievement but does not end or skip the rest of the
campaign. The epilogue reports one established destination as successful dispersal, two through four
as broad dispersal, and all five as complete destination coverage. These labels measure geographic
breadth rather than the number of routes taken and do not change the terminal result enum.

Once a terminal result is set, `AdvanceTurn()` rejects further turns, the result is persisted in saves, and
`pkg/app` opens `end_scene.go` by reading `Frame.CampaignResult`; the UI never infers a result from
turn or population. The modal end scene names the outcome, exact turn/date, final sapiens and
archaic populations and band counts, all established campaign destinations, and the corresponding
one/two-to-four/five-destination breadth label. It blocks every planning command while leaving
Ctrl/Cmd+S available for preserving the final state. A prominent **New Campaign** button and the
terminal-only `N` shortcut invoke the application `NewCampaign` use case. That use case replaces the
aggregate with a turn-0 world using the next deterministic seed derived from the completed world's
seed, resets `WorldRevision` to `1`, advances the long-lived `TerrainRevision`, and leaves existing
save slots untouched. Tests assert exact clock endpoints, the exact long-term turn-400
climate target, the bounded instantaneous climate envelope including abrupt pulses, 12-turn
periodicity, deterministic seeded noise, deterministic episode activation and capped regional
impact, sorted/idempotent species-specific achievement latching for each destination, the
one/two-to-four/five-destination epilogue classification, sapiens extinction with surviving archaic
bands, turn-400 sapiens extinction taking precedence over dispersal failure, successful survival
through any one destination, and turn-400 failure without a destination.

### Turn pipeline (`internal/domain/turn.go`)

The population calculation extends the logistic growth term with the already-required starvation
mechanic and separates persistent exposure from discrete incidents. Phase 3 uses the band's origin
tile for every demographic input because it foraged there during this turn. Here `P` is the same
fixed `P_start` used for food requirements and the squared `raw_starvation` above, which applies
without a health gate or health multiplier. The food-deficit multiplier on positive growth is not
an additional mortality cause; `raw_starvation` remains one of the three phase-3 causes. Calculate:

```
K_eff          = EcologicalK(origin) · T_tech
BaseGrowth     = r · P · (1 − P_total_origin / K_eff)
Growth         = BaseGrowth · (1 − FoodDeficitFraction)    if BaseGrowth > 0
                 BaseGrowth                              otherwise
P_grown        = max(0, P + Growth)
seasonal_component_rate[c] = SeasonalRisk(origin, Season, c)
                             · SeasonalRemainingRisk(b, c)
                             · SeasonalGeneticRemainingRisk(b, c)
seasonal_rate  = SeasonalMortalityScale · sum_c(seasonal_component_rate[c])
raw_seasonal   = P · seasonal_rate
chronic_component_rate[c] = ChronicRisk(origin, c) · HealthVulnerability(b)
                            · ChronicRemainingRisk(b, c)
                            · GeneticRemainingRisk(b, c)
genetic_burden_rate = GeneticBurdenRate(b, origin)
chronic_rate   = clamp(ChronicMortalityScale
                       · (sum_c(chronic_component_rate[c]) + genetic_burden_rate),
                       0, MaxChronicRate)
raw_chronic    = P · chronic_rate
raw_mortality  = raw_starvation + raw_seasonal + raw_chronic
mortality_scale = 1                                      if raw_mortality = 0
                  min(1, P_grown / raw_mortality)        otherwise
M_x             = raw_x · mortality_scale                for each phase-3 cause x
P_demographic_raw = P_grown − M_starvation − M_seasonal − M_chronic
P_demographic   = RoundPopulation(P_demographic_raw)
```

`RoundPopulation` rounds **stochastically**: it adds a draw from `[0, 1)` before truncating, so a
value lands on the lower whole person with probability `1 - frac` and the upper one with probability
`frac`, and is therefore unbiased in expectation while the stored count stays integral. Nearest-
integer rounding, the approach first selected here, created a deadband of half a person either side
of zero net change; at the growth and mortality scales this model actually runs, a band's entire
per-turn demographic movement fell inside that deadband every turn and was discarded, so populations
could not move at all. Drawing is also the better model of the thing being represented: whether a
small band grows in a given period is genuinely uncertain, and demographic stochasticity is exactly
what makes small populations fragile. The draw comes from the aggregate-owned `WorldRNG`, so a
campaign remains a pure function of its seed and the sequence survives save and restore. It is
applied once after the complete phase-3 expression; it does not separately round growth or any
cause's raw or capped mortality estimate. The persisted mortality breakdown therefore preserves those analytic
fractional estimates, while current population is always an integer-valued count. Validation rejects
fractional JSON populations and values above `MaxPopulation = 2^32 - 1` during decoding.

The approved initial logistic coefficient is **`r = 0.020` per game turn** for both species. At
negligible crowding and full feeding it requests growth of approximately 2% of the band's
start-of-turn population before other effects. It is not an annual rate and is deliberately held
constant across campaign eras even though the calendar span represented by a turn changes. Era
duration therefore affects the displayed chronology, not this simulation coefficient. It remains an
Initial balance parameter tunable in §12.

Step 5e first selected `r = 0.002`, on the reading that higher per-turn growth drove whole-tile
crowding decline faster than outward dispersal could relieve it. That reading was wrong, and the
correction is recorded here because the number alone does not explain itself. The runaway decline
came from the crowding term being unbounded below rather than from growth being too high; once it is
bounded, mortality absorbs three times the value that reading had forced, and `0.002` is revealed to
produce no dispersal whatsoever — a campaign at that coefficient ends holding the four bands it was
founded with, having grown them in place. The corpus was re-swept across `0.002`, `0.005`, `0.010`,
`0.020`, `0.030`, and `0.060`: `0.005` subdivides on only two seeds of eight, and everything from
`0.010` upward establishes every named destination on every seed.

`r` and `SplitStressThreshold` are selected **together**, because they move band size in opposite
directions and neither is meaningful alone. The threshold sets the ceiling a band may reach, at
roughly `SplitStressThreshold · BaselineK`; `r` sets how fast the ceiling is reached and therefore
how often a split fires. Raising `r` without raising the threshold does not grow bands, it multiplies
them at a smaller average size: at the `0.5` threshold, six times the growth rate gave six times the
bands with the median falling from 74 people to 28. The pair `r = 0.020` with
`SplitStressThreshold = 0.67` was selected for a band-size distribution with a genuine upper tail —
median 74, upper quartile 95, largest 148 — rather than the tight 74-to-89 cluster the previous pair
produced.

**Crowding uses the whole tile, growth uses the band.** The logistic term has two distinct
population inputs and they are deliberately different. The leading `r · P` scales the band's own
increase by its own people. The crowding factor `(1 − P_total_origin / K_eff)` uses the origin's
**total** start-of-turn population across every resident band of both species, matching the same
`P_total` that drives tile degradation and the destination-population term in migration scoring. A
tile at capacity therefore stops growth for everyone standing on it, rather than letting two bands of
50 each read a `K_eff` of 100 as half-empty and jointly overshoot toward 200.

`K_eff` remains band-specific because `T_tech` is the acting band's own capacity multiplier: on a
crowded tile, a technologically better-equipped band still finds room where a less-equipped one does
not. The crowding factor may go negative when `P_total_origin > K_eff`, which is the crowding-driven
decline path and is intentionally not scaled by the fed fraction below.

**Crowding decline is bounded.** The decline the logistic term may request in one turn is capped at
`MaxCrowdingDeclineFraction = 0.25` of the band's start-of-turn population:

```
BaseGrowth = max(r · P · (1 − P_total_origin / K_eff), −MaxCrowdingDeclineFraction · P)
```

Without the bound the crowding factor is unbounded below, and the paragraph above describing a full
tile as stopping growth becomes false at any real overshoot: at thirty times capacity the same
expression removes most of a band in a single turn. What makes that unacceptable is not the
magnitude but the accounting. Every phase-3 mortality cause is scaled so it cannot exceed the
population and is recorded in the persisted breakdown §9 stores and the HUD displays; crowding
decline is reported as growth and belongs to no cause, so the people are simply gone with every
category reading zero. A band of 112 on a tile whose `K_eff` had fallen to 10.96 lost 99 people in
one turn with starvation, seasonal, chronic, macro, and acute all zero and no food deficit, and
ordinary route-neutral dispersal produced roughly ninety such events per campaign.

The bound makes an over-capacity tile a sustained squeeze a player can see coming and answer — by
splitting and moving through a chokepoint in smaller groups — rather than an unexplained cull on the
turn of arrival. A tile that cannot support anyone at all takes the same bound rather than
annihilating its occupants, which keeps the function continuous as `K_eff` approaches zero and
leaves a climate shift under a settled band survivable long enough to answer. The value is
deliberately chosen from a range in which it does not act as a balance parameter: at `0.10`, `0.15`,
`0.20`, and `0.25` the corpus peaks at 1012, 1033, 1035, and 1035 people, and that insensitivity is
what distinguishes a guard against a pathology from a tuning knob. This is the
only place total tile population enters demographics; food, water, health, starvation, and mortality
all remain band-local, and shared stock scarcity continues to act through the proportional
allocator rather than through this term.

**Food-limited positive growth.** Evaluate the logistic `BaseGrowth` once using the origin's
current `K_eff`, the origin's start-of-turn `P_total_origin`, and the band's own start-of-turn `P`.
If it is positive, multiply it by the fraction of food
needs actually met, `1 - FoodDeficitFraction`; otherwise leave it unchanged. Both species use the
same rule. An analytic logistic gain of `0.2` people becomes `0.2`, `0.1`, or zero when the band is fully fed,
half-fed, or completely unfed. An analytic logistic decline of `0.2` people remains `-0.2` in all three
cases. Full feeding restores the unscaled logistic result, not extra growth from surplus food.

Use the existing post-reserve fraction, not health, gross harvest, final population, or a second
food calculation. Do not charge food again for growth, change the frozen food requirement, turn
foregone growth into a mortality cause, or multiply negative crowding-driven decline by the fed
fraction. Starvation remains a separate direct loss based on `P_start`, not `P_grown`.
`BaseGrowth` and the scaled result are transient simulation values copied only into the completed
turn's display-only `LastOutcomeReport`; validate
inputs and arithmetic before `P_grown` or mortality caps. The growth coefficient `r` and carrying-
capacity balance still need tuning together; this rule selects `r` but not the biome carrying
capacities. `HazardAlgorithm: "split-v1"`
owns this demographic coupling, with no new simulation input, RNG draw, or algorithm identifier.

Fixtures lock `r = 0.020` and cover positive, zero, and negative `BaseGrowth` at deficit fractions `0`, `0.50`, and `1`,
including a decline held at `−MaxCrowdingDeclineFraction · P` when the unbounded term would exceed
it, a mild overshoot left unbounded so the cap cannot become a floor every decline snaps to, and a
`K_eff` of zero taking the same bound rather than removing the band,
including zero growth at carrying capacity, a zero logistic coefficient, and valid tiny fractional
growth whose combined phase-3 survivor result receives exactly one whole-person rounding. Co-location fixtures place two bands of 50 on a tile whose
`K_eff` is 100 and assert that both compute zero `BaseGrowth`, not the positive growth a band-local
crowding term would produce; a mixed-species pair with the same total must behave identically, and
the same total split across one, two, and four bands must give the same crowding factor. A third
fixture holds `P_total_origin` fixed while varying one band's own `P` and asserts that only the
leading `r · P` term changes. `Stress` fixtures assert the matching whole-tile denominator, so a
band sharing a tile crosses `SplitStressThreshold` at the population its neighbours help create. Assert `0 <= Growth <= BaseGrowth` for positive bases and
exactly unchanged non-positive bases. Keep the start-of-turn requirement fixed and verify that
reserves can restore full growth by meeting the food requirement. Cover both species, invalid
inputs, combined mortality caps, and identical future results after save/load.

The proportional scale matters only when combined mortality would exceed the population available
after growth; it preserves an exact, order-independent mortality breakdown while keeping
`P_demographic >= 0`. All raw and applied mortality values are finite, non-negative absolute people.
`raw_seasonal` covers predictable seasonal temperature and water stress; `raw_chronic` covers the
biome's persistent endemic-disease, predation, and exposure baseline. Here `c` iterates the
`ShelterProtectionClass` values a profile actually covers, in their stable order: all four for
chronic risk, and the three seasonal ones — `Exposure`, `CampDisease`, `Uncovered` — for
`raw_seasonal`. `ChronicRemainingRisk(b, c)` is
`RemainingRisk(tech, CampMitigation(b, c, origin))`, with `tech` taken only from the technology
effects applicable to that chronic component. Seasonal losses use the same product with their
own component-specific technology inputs and origin camp fractions as
`SeasonalRemainingRisk(b, c)` before summing into `raw_seasonal`; neither path applies a blanket
reduction after summing risks. `SeasonalGeneticRemainingRisk` is `1` unless a trait explicitly
covers that seasonal component. Seasonal rates do not receive `HealthVulnerability`; the multiplier
belongs only to chronic attrition. Direct water stress remains uncovered by shelter. The
last-mortality display has five aggregate fields after adding the
separately resolved macro-event cause;
component attribution introduces no extra persisted history.

The approved initial **moderate seasonal-risk table** gives the unmitigated fraction of the band's
start-of-turn population at risk per game turn. Its component columns are in exact
`ShelterProtectionClass` order and their sum is shown only as an audit aid.

**Seasonal risk has no `CampPredation` component**, so the profile is authored over three classes
rather than four. Predation is an encounter, not a season-long background rate: camp-attributable
attacks resolve as discrete `Predation` incidents in phase 5 and as the chronic baseline, both of
which do carry a `CampPredation` component. Authoring a fourth seasonal column would have meant 24
structurally zero cells and a `SeasonalRemainingRisk(b, CampPredation)` product computed against a
zero base for every band on every turn — generality that states nothing and that a balance pass
could accidentally make nonzero without deciding to. `CampSecurityScale` therefore reaches chronic
and acute components only.

| Current biome         | Season          | `Exposure` | `CampDisease` | `Uncovered` |   Total |
| --------------------- | --------------- | ---------: | ------------: | ----------: | ------: |
| Savanna               | `SeasonWarm`    |    `0.006` |       `0.000` |     `0.004` | `0.010` |
| Savanna               | `SeasonCooling` |    `0.004` |       `0.000` |     `0.002` | `0.006` |
| Savanna               | `SeasonCold`    |    `0.003` |       `0.000` |     `0.001` | `0.004` |
| Savanna               | `SeasonWarming` |    `0.004` |       `0.000` |     `0.002` | `0.006` |
| Riverine Woodland     | `SeasonWarm`    |    `0.001` |       `0.001` |     `0.002` | `0.004` |
| Riverine Woodland     | `SeasonCooling` |    `0.001` |       `0.001` |     `0.002` | `0.004` |
| Riverine Woodland     | `SeasonCold`    |    `0.002` |       `0.002` |     `0.002` | `0.006` |
| Riverine Woodland     | `SeasonWarming` |    `0.001` |       `0.001` |     `0.002` | `0.004` |
| Coastal Shrubland     | `SeasonWarm`    |    `0.003` |       `0.000` |     `0.005` | `0.008` |
| Coastal Shrubland     | `SeasonCooling` |    `0.002` |       `0.000` |     `0.004` | `0.006` |
| Coastal Shrubland     | `SeasonCold`    |    `0.003` |       `0.000` |     `0.005` | `0.008` |
| Coastal Shrubland     | `SeasonWarming` |    `0.002` |       `0.000` |     `0.004` | `0.006` |
| Semi-Arid Desert      | `SeasonWarm`    |    `0.012` |       `0.000` |     `0.008` | `0.020` |
| Semi-Arid Desert      | `SeasonCooling` |    `0.007` |       `0.000` |     `0.005` | `0.012` |
| Semi-Arid Desert      | `SeasonCold`    |    `0.005` |       `0.000` |     `0.003` | `0.008` |
| Semi-Arid Desert      | `SeasonWarming` |    `0.007` |       `0.000` |     `0.005` | `0.012` |
| Mountainous Highlands | `SeasonWarm`    |    `0.005` |       `0.000` |     `0.003` | `0.008` |
| Mountainous Highlands | `SeasonCooling` |    `0.008` |       `0.000` |     `0.004` | `0.012` |
| Mountainous Highlands | `SeasonCold`    |    `0.014` |       `0.000` |     `0.006` | `0.020` |
| Mountainous Highlands | `SeasonWarming` |    `0.008` |       `0.000` |     `0.004` | `0.012` |
| Glacial Tundra        | `SeasonWarm`    |    `0.008` |       `0.000` |     `0.002` | `0.010` |
| Glacial Tundra        | `SeasonCooling` |    `0.015` |       `0.000` |     `0.003` | `0.018` |
| Glacial Tundra        | `SeasonCold`    |    `0.030` |       `0.000` |     `0.005` | `0.035` |
| Glacial Tundra        | `SeasonWarming` |    `0.015` |       `0.000` |     `0.003` | `0.018` |

The corresponding **moderate chronic-risk table** is biome-specific but season-independent:

| Current biome         | `Exposure` | `CampPredation` | `CampDisease` | `Uncovered` |   Total |
| --------------------- | ---------: | --------------: | ------------: | ----------: | ------: |
| Savanna               |    `0.002` |         `0.004` |       `0.002` |     `0.004` | `0.012` |
| Riverine Woodland     |    `0.001` |         `0.002` |       `0.008` |     `0.007` | `0.018` |
| Coastal Shrubland     |    `0.002` |         `0.002` |       `0.003` |     `0.005` | `0.012` |
| Semi-Arid Desert      |    `0.004` |         `0.002` |       `0.001` |     `0.003` | `0.010` |
| Mountainous Highlands |    `0.005` |         `0.002` |       `0.001` |     `0.004` | `0.012` |
| Glacial Tundra        |    `0.008` |         `0.002` |       `0.001` |     `0.004` | `0.015` |

Seasonal totals range from `0.004` in benign phases to `0.035` in cold tundra; chronic totals range
from `0.010` to `0.018`, preserving the intent table's risk ordering without claiming historical
mortality estimates. These are separate direct-mortality inputs, not health-score loss rates. The
seasonal `Uncovered` column includes the profile's drought, flood/storm, and fall pressure; it is not
multiplied by camp protection. `WaterDeficitFraction` independently damages health under the water
rule and is not added again as a duplicate mortality fraction. Likewise, the chronic
`CampDisease`/`Uncovered` entries are direct attrition and do not duplicate the separate endemic
disease health-score table.

The tables are relative per-turn risk weights. After component-specific technology, shelter,
health, and genetic effects, multiply their stable sums by the selected Initial
`SeasonalMortalityScale = 0.10` and `ChronicMortalityScale = 0.10`. Apply the chronic scale before
`MaxChronicRate`; invalid or non-finite unscaled arithmetic is still rejected before scaling. The
scales let the profile retain readable biome and season ratios while step 5e calibrates how quickly
those weights remove people across a compressed 401-turn campaign.

For each profile, select exactly one current-biome row (and one current season for the seasonal
table), apply technology, camp, health-vulnerability where specified, and genetic factors to each
component once, then sum in stable class order. Reject missing/duplicate biome-season keys,
non-zero data outside the closed dimensions, non-finite or negative rates, and an authored total
that disagrees with the component sum. The total column is not read by the simulation.

The profiles distinguish seasonal pressure, persistent chronic attrition, and discrete acute
incidents; class partitioning must not duplicate a configured risk contribution within or across
those profiles. `HealthVulnerability(b) = 1 + (1 - Health[b])` reads the completed phase-3 health
update and lies in `[1, 2]`; it is not clamped to one or applied again after the component sum.
`GeneticRemainingRisk(b, c)` is the finite `[0, 1]` start-of-turn trait factor authored for that
specific cold-exposure, high-altitude chronic-exposure, or microbial-disease component; it is `1`
when no trait applies. `GeneticBurdenRate` is the finite non-negative UV-damage burden; the separate
low-pathogen immune and vitamin-D terms modify health in phase 3 instead. Component configuration
must choose a modifier, mortality burden, or health burden for each channel and reject duplicate
application. `raw_seasonal` uses the equivalent component-specific genetic factor for temperature
exposure. Genetic factors compose with technology/camp remaining risk in the documented component
order; they do not reduce unrelated predation, shortage, or injury.
`MaxChronicRate` is a fixed configuration constant bounding the scaled, vulnerability-adjusted chronic
rate before the shared mortality scale; the approved initial playtest value is **`0.25`**. It is the
ceiling on the fraction of a band that persistent endemic disease, predation, and exposure can remove
in one turn, after `HealthVulnerability` has already doubled the underlying components at zero
health. It exists so that a band in a hostile biome at poor health declines steeply without chronic
attrition alone becoming an extinction guarantee. Under the selected initial configuration, exact
extinction can occur through phase 3's shared mortality cap when demographic decline and multiple
causes exhaust the available population; the selected strictly fractional phase-5 losses cannot
create it from a positive survivor. The value is deliberately well above any
expected baseline sum so that it binds only in compounded worst cases; §12 tunes it against the seed
corpus alongside the component rates it caps.
Validate finite chronic inputs, `MaxChronicRate` in `[0, 1]`, vulnerability in `[1, 2]`, and
remaining-risk/genetic factors in `[0, 1]`, so the resulting capped chronic rate is finite and in `[0, 1]`.
The caps must not conceal invalid or non-finite component arithmetic. Full health preserves
baseline chronic risk, while direct starvation, seasonal losses, and acute events do not receive
this multiplier. A living band must occupy a habitable tile with finite
`K_eff > 0`; world generation, command validation, and save validation enforce that invariant before
the division.

Phase 5 resolves discrete acute events after migration and after any separately accounted
macro-event impact. Macro episodes are world-level deterministic forcing, not a sixth weight in the
per-band acute distribution; they neither consume a hazard draw nor share its cap. The stable acute
event-kind order is
`Predation`, `DiseaseOutbreak`, `FloodStorm`, `ExposureFall`, `CrossingMishap`. For each surviving
band in ascending band-ID order, `hazard.go` derives unmitigated `base_probability[k, c]` components
from the **final** tile's risk profile, biome, and current season, including fauna-dependent hunting/
megafauna work risk captured under `linear-share-work-risk-v1` from the origin's regional profile
and accepted workforce shares in phase 3. Work-risk inputs feed
the existing uncovered components without duplicating a final-tile risk contribution or expanding
the four-class bound. These are non-negative risk weights; their finite sum is capped only
after mitigation and aggregation.

The selected acute environment profile starts with one biome weight for each non-crossing event:

| Current biome         | `Predation` | `DiseaseOutbreak` | `FloodStorm` | `ExposureFall` |
| --------------------- | ----------: | ----------------: | -----------: | -------------: |
| Savanna               |     `0.025` |           `0.015` |      `0.010` |        `0.015` |
| Riverine Woodland     |     `0.015` |           `0.035` |      `0.030` |        `0.010` |
| Coastal Shrubland     |     `0.015` |           `0.020` |      `0.040` |        `0.015` |
| Semi-Arid Desert      |     `0.010` |           `0.010` |      `0.010` |        `0.040` |
| Mountainous Highlands |     `0.015` |           `0.010` |      `0.020` |        `0.045` |
| Glacial Tundra        |     `0.020` |           `0.010` |      `0.010` |        `0.055` |

Multiply that row component-wise by the current season's selected factor:

| Season          | `Predation` | `DiseaseOutbreak` | `FloodStorm` | `ExposureFall` |
| --------------- | ----------: | ----------------: | -----------: | -------------: |
| `SeasonWarm`    |       `1.0` |             `1.2` |        `1.2` |          `0.8` |
| `SeasonCooling` |       `1.0` |             `1.0` |        `1.0` |          `1.0` |
| `SeasonCold`    |       `0.9` |             `0.8` |        `0.8` |          `1.3` |
| `SeasonWarming` |       `1.0` |             `1.0` |        `1.1` |          `1.0` |

The resulting event weight is partitioned across protection classes without changing its total:

| Event kind        | `Exposure` | `CampPredation` | `CampDisease` | `Uncovered` |
| ----------------- | ---------: | --------------: | ------------: | ----------: |
| `Predation`       |        `0` |          `0.40` |           `0` |      `0.60` |
| `DiseaseOutbreak` |        `0` |             `0` |        `0.50` |      `0.50` |
| `FloodStorm`      |        `0` |             `0` |           `0` |      `1.00` |
| `ExposureFall`    |     `0.70` |             `0` |           `0` |      `0.30` |
| `CrossingMishap`  |        `0` |             `0` |           `0` |      `1.00` |

`CrossingMishap` does not read the biome or season tables. A successful northern Wallacea,
southern Wallacea, or Beringian crossing supplies an unmitigated `Uncovered` weight of `0.08`,
`0.10`, or `0.04`, respectively. A band that did not successfully traverse a named passage supplies
zero. Finally add the captured Hunting and MegafaunaTracking work-risk weights from their origin to
the applicable `Uncovered` components. In symbols, with `partition[k,c]` from the table:

```text
environment_weight[k] = AcuteBiomeBase[Biome(final_tile), k]
                        * AcuteSeasonFactor[Season, k]       for k != CrossingMishap
                        0                                   otherwise
base_probability[k, c] = environment_weight[k] * partition[k, c]
                         + WorkRisk[b, k]                    when c == Uncovered
                         + PassageRisk[b]                    when k == CrossingMishap
                                                            and c == Uncovered
```

The event and class tables are complete; missing or duplicate rows, negative/non-finite values,
partition rows whose finite entries do not sum exactly to one, and any nonzero crossing base outside
a successful named passage are invalid configuration. Camp work reduces each covered component
before event selection:

```text
probability_component[k, c] = base_probability[k, c]
                              * ProbabilityRemainingRisk(b, k, c)
                              * GeneticProbabilityRemainingRisk(b, k, c)
raw_probability[k] = sum_c(probability_component[k, c])
probability[k] = AcuteProbabilityScale · raw_probability[k]
```

`ProbabilityRemainingRisk(b, k, c)` is
`RemainingRisk(tech, CampMitigation(b, c, final tile))`, with `tech` taken only from technology
effects on that kind/class's probability channel. The finite `[0, 1]` product is used directly;
the unmitigated base and the aggregated event weight receive no second copy of either reduction.
Without applicable technology the factor is `1 - camp`; with no camp coverage it is `1 - tech`.
A configured `GeneticProbabilityRemainingRisk` reads the frozen start-of-turn trait state and is `1`
where no trait applies. The trait-effect coverage table must not repeat that factor in the base
profile or camp/technology product. A 50% mitigation halves its remaining component; it does not subtract 50 percentage points from
an event probability. Shelter's magnitude follows `saturating-share-terrain-v1` using the captured
share; only exposure protection reads the final tile's `NaturalShelter`. The origin's cave bonus is never
carried into a destination's hazard calculation. A mixed event kind receives no blanket camp-work
reduction: camp security cannot reduce its field attacks, and hygiene cannot reduce unrelated
disease components. Work-risk activation, mapping, and coefficients use the selected
linear-share rule above.

`CrossingMishap` has a non-zero raw probability only when the band successfully traversed a named
passage during phase 4; a canceled crossing supplies neither that weight nor a destination tile.
Sum components in stable class order and event weights in stable kind order, validating that all
components and totals are finite and non-negative. Multiply every event total by the selected
Initial `AcuteProbabilityScale = 0.10`, then, if the scaled aggregate exceeds
`MaxAcuteProbability`, scale every event probability proportionally so their sum equals the cap;
otherwise leave them unchanged. An aggregate mitigated weight of `0.40` therefore becomes `0.04`
under the selected scale and does not hit the `0.25` cap. Configuration validation requires
`MaxAcuteProbability` in `[0, 1]`. The approved initial playtest value is **`0.25`**, so even a
maximally capped band retains a `0.75` no-incident interval and, if held continuously at the cap,
averages one acute incident per four turns. This is a per-band probability rather than a global
one: several bands may experience different incidents in the same turn. There is no second
normalization after event selection.

This preserves the existing shared cap. With base risks and effect coverage fixed, increasing either
technology or camp probability mitigation cannot increase the total acute-event probability, but it
may remain at the cap. Individual normalized probabilities can change as competing weights fall, including
uncovered probabilities rising while their raw components stay unchanged. Tests and UI copy must
not promise that every event kind's final probability falls by its shelter mitigation percentage.

One `WorldRNG.Float64()` draw selects no event or one event by cumulative scaled probability in the
**Kin support lowers acute risk.** A band in ordinary contact with same-species bands — sharing or
neighbouring its tile — faces less acute risk than one standing alone:

```
kin_share      = contacts / (contacts + KinContactHalfSaturation)      contacts > 0
               = 0                                                     otherwise
kin_remaining  = 1 − MaxKinAcuteReduction · kin_share
probability[k] = AcuteProbabilityScale · kin_remaining · raw_probability[k]
```

with `KinContactHalfSaturation = 1.0` and `MaxKinAcuteReduction = 0.40`. This is an Allee effect:
mutual aid, shared watch, and shared knowledge of the ground make a shock survivable that would
otherwise take a chunk of the band. It applies before the `MaxAcuteProbability` cap and lowers every
event kind alike, because a neighbouring band helps with a predator, a flood, or a fall without
distinction.

`contacts` counts living same-species bands under `ordinaryContact`, the same predicate same-species
gene flow already uses, so the model holds one notion of a neighbouring band rather than two that can
drift apart. It is read from the start-of-turn snapshot, so the count cannot depend on the order the
turn pipeline happens to walk the bands in. The share saturates rather than accumulating, so a dense
cluster cannot outrun the reduction cap, and `kin_remaining` is exactly `1` for an isolated band, so
the rule is invisible where it does not apply and never falls below its stated floor of `0.60`. It
follows the remaining-risk shape `InnateImmuneRemainingRisk` and `AridHeatRemaining` already use.

The rule attaches to acute risk rather than to fertility for a reason that no longer holds: under
nearest-integer rounding the logistic term fell inside the rounding deadband every turn, so a
fertility bonus had nothing to act on. Stochastic rounding and the selected `r` have since made
growth register, so attaching an adjacency effect to fertility is now possible and is left as an open
balance question rather than a settled one. `HazardAlgorithm` owns the coupling; the persisted
`KinSupportAlgorithm: "saturating-kin-acute-v1"` names it, and it consumes no RNG draw of its own.

stable enum order, using half-open intervals: select the first kind for which the draw is strictly
less than its cumulative upper bound. Zero-width intervals cannot select an event; a draw at or
above the final cumulative bound means no event. If an event is selected, and only then, a second
draw selects its severity:

```
loss_fraction = clamp(lerp(MinAcuteLoss[k], MaxAcuteLoss[k], v)
                      · (1 − SeverityMitigation(b, k))
                      · (1 − GeneticSeverityMitigation(b, k)), 0, 1)
M_acute_raw   = min(P_after_macro, P_after_macro · loss_fraction)
P_final       = RoundPopulation(P_after_macro − M_acute_raw)
M_acute       = P_after_macro − P_final
```

When no macro episode affects the band, `P_after_macro = P_demographic`.

The approved initial **moderate acute-severity table** gives bounds as fractions of the population present
after any macro-event loss:

| Acute event kind  | `MinAcuteLoss[k]` | `MaxAcuteLoss[k]` |
| ----------------- | ----------------: | ----------------: |
| `Predation`       |            `0.02` |            `0.08` |
| `DiseaseOutbreak` |            `0.03` |            `0.12` |
| `FloodStorm`      |            `0.02` |            `0.10` |
| `ExposureFall`    |            `0.01` |            `0.06` |
| `CrossingMishap`  |            `0.05` |            `0.15` |

The table is complete in stable acute-event order and applies to both species and all regions.
Biome, season, movement, and workforce determine whether and how often an event occurs, not a
second hidden severity bound. Technology and genetics may reduce the selected event's severity only
through their explicit effect-table rows after interpolation.

`SeverityMitigation(b, k)` excludes all camp-work protection and the natural-shelter bonus; it
retains only separately specified technology effects, or zero when none apply.
`GeneticSeverityMitigation` is separately authored from the frozen heritable state and is zero when
no trait applies. Each kind/trait pair must choose its documented probability, severity, health, or
chronic channel without applying one configured benefit twice. For the same event
kind, severity draw, population, and other severity inputs, changing shelter share or natural
shelter cannot change the conditional loss. The class components need no subtype-selection draw
because this model retains one severity distribution per event kind. There is no additional
prevention roll, reroll, post-hit shelter reduction, or prevented-incident event/sound.

Thus every surviving band consumes one draw when no incident occurs and two when one occurs, with at
most one acute event per band per turn. `MinAcuteLoss`, `MaxAcuteLoss`, and mitigation are finite and
in `[0, 1]`, with each minimum no greater than its maximum. Chronic attrition never consumes RNG,
and no phase subtracts any phase-3 or acute mortality value a second time.

After the existing direct loss, a selected `DiseaseOutbreak` additionally reduces each surviving
band's health once, as specified in the health subsection. Reuse the same raw severity draw `v`
to interpolate between initial health bounds `0.05` and `0.15`, then multiply the explicitly
applicable health-technology and `OutbreakHealthGeneticRemainingRisk` factors. The genetic factor is
`1` when no trait applies and must not repeat a probability/severity benefit unless the coverage
table explicitly defines separate channels. Do not reuse mortality's `loss_fraction` or add a draw, camp reduction,
or health/vulnerability multiplier. No-event and other incident kinds do
not apply this health hit. It does not alter the event's population-loss field, replay phase-3
health or mortality, or retroactively change chronic vulnerability. Publish the resulting health
in the completed frame/save; background disease already contributed in phase 3.

Every triggered acute incident appends a typed event containing turn, ordinal within the turn, band
ID, final tile ID, kind, applied loss, and optional passage ID. Migration cancellations and other HUD
alerts use the same chronological `World.Events` feed. It holds at most `MaxEvents = 128` values;
append order is deterministic and an overflow discards the oldest value. The per-turn ordinal avoids
an unbounded persisted event counter. Routine chronic losses do not spam the feed: the fixed-size
last-mortality breakdown on each surviving band exposes starvation, seasonal, chronic, macro, and acute
losses in the frame instead. A band that reaches zero during phase 3 receives one `BandExtinct`
event with its origin and mortality breakdown before cleanup; an acute incident that reduces a band
to zero already supplies the corresponding event and is not duplicated.

Between turns, `GameService.Apply` maps boundary commands and `World.PlanPlayer` validates them
without consuming randomness or advancing the clock; the aggregate rejects any command whose acting
band is archaic. `Interbreed` is the only
public command allowed to name an archaic target, and its acting band must still be sapiens.
`SplitBand` is valid only
for a band whose previous snapshot has `Stress > SplitStressThreshold`, when
its population is at least `MinSplitSourcePopulation = 40`, `len(Bands) < MaxBands`, and
`NextBandID` can allocate another positive ID without wrapping; it
targets one currently eligible ordinary cardinal or diagonal edge to a habitable land tile and
divides population as evenly as whole people allow, divides stored food exactly `50/50` without
loss, and copies the exact
proportional allocation and heritable state to both results before the next turn begins. Both
descendants' spatial actions are spent. Because neither result is the exact band that experienced the
previous turn, the command clears the display-only last-mortality breakdown, `LastFoodReport`, and
`LastOutcomeReport` on both resulting bands. The reports remain unavailable until each completes a turn.
`QueueMigration` and `Interbreed` record intents for resolution during the next turn and consume
only the sapiens actor's spatial action. Attraction rank never constrains a command. Invalid split,
migration, interbreeding, assignment, or research commands, including a
split at the cap, return a UI-visible error and leave world state unchanged. The computer authority
uses the same rules through the atomic policy batch, not through this public entry point.

After `ui.EndTurn`, the computer planning batch runs against the final player-planning state. If it
succeeds, the turn proceeds through the five pipeline phases in this order. Spoilage is the first
operation of phase 1; consumption remains in phase 3 and overflow discard in phase 5. Rejected
turn requests or failed computer planning do not reach the spoilage checkpoint.

1. In stable band-ID order, apply `population-food-turns-v1` spoilage once to each band's
   carried-over `StoredFood`, after all accepted planning and before any harvest or consumption.
   Then `climate.Advance(nextTurn)` derives the named three-turn season and all four temperature
   components from the fixed campaign curve, authored abrupt-pulse catalog, and
   `(world seed, nextTurn)` as specified above; derive Beringian passage eligibility from the
   long-term component alone. Resolve which macro episodes activate in the calendar interval crossed
   by this transition and derive the active impact and next-turn warning summaries without RNG.
2. For each tile in sorted order, derive `BaselineK`. Resolve the habitat-collapse branch above
   immediately for any zero-capacity tile; affected bands and stocks take no later phase work.
   Otherwise update bounded `Degradation` from start-of-turn total population, derive the
   biome/season/degradation-adjusted ecological and resource caps,
   multiply active macro habitat/resource factors once, clamp stocks to any lowered caps, and apply
   the toward-cap recurrence exactly once.
3. Per tile, **in sorted tile order**: derive every resident band's five worker counts from its
   accepted basis-point shares and `P_start`; freeze its six start-of-turn heritable values and
   evaluate each applicable trait-effect function against this phase's origin environment; calculate its linear flora demand under
   `linear-shared-flora-v1` from the origin's current biome and acquired technology, linear
   combined terrestrial/aquatic Hunting demand under `linear-shared-fauna-v1`, and linear megafauna demand under
   `linear-megafauna-v1` from that same origin's current profile and acquired technology, plus
   population-wide water demand in WU, using `BaseWaterPerPerson = 1.0` and the origin's
   current-local-heat multiplier in `[1.0, 1.3]`, derived after phase 1 and captured before allocation;
   allocate each shared stock proportionally across both species; and deplete each stock by exactly
   the allocated total, with hunting and megafauna demands participating in one fauna allocation.
   Then, per band in sorted ID order:
   capture its turn-local shelter share and calculate class-specific mitigation under
   `saturating-share-terrain-v1`, using the origin's `NaturalShelter` only for exposure, without a
   population multiplier or carryover from earlier turns. Combine each seasonal/chronic component's
   camp fraction with its applicable technology-only fraction via `RemainingRisk`; derive its
   unmitigated hunting/megafauna work risk under `linear-share-work-risk-v1` from that same origin
   profile and accepted shares, plus its bounded transient
   `research_gain` from toolcraft without applying the research yet → convert three source totals:
   Plant from flora, Animal from ordinary terrestrial Hunting plus MegafaunaTracking, and Aquatic
   from Hunting's aquatic rate composition. Apply the applicable `FattyAcidMetabolism` modifier
   exactly once to each source total before combining them, then combine that harvest with the
   post-spoilage reserve, automatically consume
   `min(FoodAvailable, FoodRequired)` → retain `FoodRemaining` for phase 5 and derive
   `FoodDeficitFraction` from the fixed start-of-turn requirement as the shortage input to the
   gradual-health model and squared starvation response. Use that response directly as
   `raw_starvation`, with initial `StarvationCoefficient = 0.10` and no health gate or multiplier.
   Keep `FoodDeficit` in FU for accounting, and capture the requirement and deficit
   with `nextTurn` for the pending `LastFoodReport`; do not publish it yet. Resolve
   the water allocation separately; it is not food. Derive `WaterDeficitFraction` from actual
   allocation and the fixed, heat-adjusted pre-demographic water requirement, after
   `AridClimateAdaptation` reduces only the heat increment. Compute the signed nutritional
   contribution: `-HealthLossRate * FoodDeficitFraction` for a shortage, otherwise
   `+HealthRecoveryRate`, with initial rates `0.20` and `0.05`. Derive `EndemicDiseaseHealthLoss`
   from the origin's separate camp/non-camp health rates: apply component-specific technology and
   hygiene through remaining-risk products, with hygiene only on the camp term and no cave bonus,
   then sum the camp and non-camp terms in that order. Subtract
   `WaterHealthLossRate * WaterDeficitFraction` and then `EndemicDiseaseHealthLoss` from prior
   health plus the nutritional contribution and any configured vitamin-D or immune-inflammatory
   health components, then clamp once to `[0, 1]`. Apply cold,
   high-altitude, pigmentation, and immune modifiers only to their named risk components; do
   not create a generic fitness multiplier. Do not clamp nutrition/components
   independently or consume water again.
   Health changes add no population-loss term; retain direct drought mortality only through the
   seasonal `Uncovered` component and disease mortality through the named seasonal/chronic channels.
   Derive `1 + (1 - Health)` from the combined value for chronic components only, then calculate
   logistic `BaseGrowth`, scale only positive growth by `1 - FoodDeficitFraction`, and calculate
   the three phase-3 mortality causes above on the origin tile → assign
   `P_demographic` once. Capture the starting and rounded demographic population, signed logistic
   growth, starting and ending health, signed nutrition contribution, and non-negative water,
   endemic-disease, and genetic-burden health losses in the pending fixed-size
   `LastOutcomeReport`. These are completed-turn explanatory actuals and no later calculation reads
   them. Retain
   remaining food without capacity clipping here; phase 5 enforces the final-population cap. A band
   reduced to zero cannot migrate and its queued intent is discarded during revalidation. Population
   changes leave `AllocationBP` untouched for the next planning period. The
   foraging/hunting/megafauna rates and access tables, fauna profiles, work-risk vectors, food
   conversion, camp-risk profiles, technology effects, and genetics effects follow their stable
   enum orders and validation contracts above.
4. Queued migrations resolve in sorted band-ID order after revalidation that the band remains at the
   recorded origin, the destination remains habitable, and the selected route remains valid: either
   the exact cardinal/diagonal `OrdinaryEdge` (including both diagonal corner-land checks) or the
   named passage ID with its band/climate eligibility. Neither the planning-time score nor movement
   cost is persisted or compared during resolution. A band gathers from its origin for the completed
   turn, then moves; a now-invalid edge or target cancels only that move and emits an event.
5. In stable episode-ID then band-ID order, apply any active eruption's capped deterministic
   population and health impact on the final tile, emitting bounded macro-impact events and recording
   macro mortality separately. Then resolve acute events in sorted band-ID order using the final tile, any successfully crossed named
   passage, and the shelter share and origin-fauna work-risk vector captured in phase 3. Add each
   captured work weight once to the matching uncovered base component. Evaluate class-specific
   shelter mitigation against final-tile risk components, using `NaturalShelter` only for exposure,
   not a cached origin-adjusted result. Combine it with applicable probability-channel technology
   via `RemainingRisk` and apply the product to each base component before aggregating and
   capping the event distribution; select at most one incident, then resolve severity without any
   camp-work reduction. Apply the existing direct loss; only a surviving band with a selected
   `DiseaseOutbreak` then receives one `OutbreakHealthLoss`, floored at zero health: linearly
   interpolate the initial `0.05`–`0.15` bounds with the same severity draw `v`, then multiply by
   `1 - OutbreakHealthTech(b)` and `OutbreakHealthGeneticRemainingRisk(b)`. Hygiene and natural shelter do not reduce this hit. Do not add RNG,
   use mortality's mitigated loss fraction, or replay phase-3 health, nutrition, or mortality;
   completed-turn publication must expose the resulting current health.
   Discard all transient shelter-effort and fauna-work-risk records after acute resolution;
   neither carries into a later turn. After all macro and acute losses, finalize every band's
   remaining food under `population-food-turns-v1`: discard only the excess above final population
   times `FoodStorageTurns`, retaining zero for an extinct band. Do this once before cleanup,
   terminal-state publication, or autosave, without repeating spoilage, consumption, or demographics. Remove
   zero-population bands, freeing capacity below `MaxBands` for the next planning period, and discard
   extinct bands' transient research gains and food-report candidates. Publish each survivor's
   captured `LastFoodReport` and `LastOutcomeReport` for this completed turn, updating only the
   latter's final population/health endpoints and actual macro/outbreak health losses after phase 5,
   without changing either report's phase-3 inputs.
   From the remaining bands' final positions and pre-gain technology state, capture the frozen
   local-contact snapshot, apply
   `co-located-cross-species-v1`, count each distinct eligible knowledgeable source once per
   recipient and technology, and combine `saturating-toolcraft-v1` with `stacked-acquired-v1` in one
   simultaneous progress update. Knowledge acquired here cannot mitigate an acute event from the
   turn just completed or relay until the next turn. From the same survivor set, capture the frozen
   heritable/contact snapshot. Add automatic same-species local-contact edges and each still-valid
   accepted interbreeding edge, then calculate the population-weighted simultaneous gene-flow
   deltas. Add per-trait deterministic selection deltas from the captured turn environment and
   outcome, followed by bounded mutation-emergence draws in ascending band-ID/trait order. Clamp and
   publish the complete next heritable vector once. No updated value can affect the completed turn
   or relay through another band until the next turn. Discard every resolved or invalid
   interbreeding intent and clear every surviving band's `SpatialActionUsed` marker. Then calculate each survivor's
   `Stress = P_total_final(final tile) / (EcologicalK(final tile) · T_tech)` for the next planning
   period, using the same whole-tile population as the crowding term above so a band cannot read a
   shared tile as uncrowded. Stress above
   `SplitStressThreshold` permits `SplitBand`; the player decides for sapiens, while the computer policy considers
   archaics at the next `EndTurn`. Run `RevealFromSapiens` once from these surviving final positions
   and newly acquired technology, unioning the persistent mask before candidate calculation and
   application frame projection;
   archaic positions contribute nothing. Finally latch regional achievements from the surviving
   post-event populations, evaluate terminal conditions, append any resulting typed alerts, and build a new
   `gameapi.Frame`. A newly arrived band that the acute event reduces below `MinEstablishedBand` does
   not establish that region, while an achievement latched on an earlier turn remains historical.
   Because the selected phase-5 macro loss and acute-severity maxima are strictly below one, neither
   can turn a finite positive population into exactly zero under the initial fractional arithmetic.
   With the selected initial configuration, the immediate sapiens-extinction result can therefore
   arise only when phase 3 has already reduced the last sapiens population to zero; the combined-loss
   fixture locks that invariant.

### Determinism is the load-bearing invariant

**Scope: release behavior is bit-identical across `amd64`, `arm64`, and `js/wasm`.** Go may fuse
`a*b + c` on one architecture and not another, and runtime transcendental functions need not return
identical low bits. §5's mechanically enforced product-rounding rule removes fusion differences.
Runtime simulation uses four mandatory checked-in exact-bit tables and no transcendental call:
`LatitudeSinSquared[64]`, `OrbitalSin[401]`, `SeasonalCos[12]`, and
`PrecessionSin[401]`. Generation tools may call `math.Sin`/`math.Cos` and compare within tolerance;
release code reads only the tables. Build step 4 and its source guard must be green before any
determinism claim or save fixture is accepted.

The resulting contract is unconditional for a supported release: identical normalized `SaveState`,
seed, algorithms, and command stream produce an identical normalized state hash on all three target
architectures. Save files are therefore portable across supported desktop architectures and the web
target. Same-process save/reload self-comparison remains useful for lifecycle bugs, but it is not a
substitute for the cross-target gate. Release CI has each Linux amd64, macOS arm64, and Chromium
js/wasm target run the exact `ReferenceSeed`/reference-policy checkpoint fixture and upload canonical
checkpoint JSON; an aggregation job compares every state hash and named balance margin byte for byte.
Any mismatch fails release and Pages deployment.

Gameplay outcome margins remain required because a robust reference policy should not barely pass,
not because architectures are allowed to disagree. The balance seed corpus is the exact ordered set
`{0, 1, 2, 3, 0x9e3779b97f4a7c15, 0xd1b54a32d192ed03,
0x94d049bb133111eb, 0xffffffffffffffff}` and `ReferenceSeed` is
`0x9e3779b97f4a7c15`; CI, §12 reports, and local reproduction use those same values.

**The mitigation is explicit rounding, not fixed-point.** Writing the end of every floating product
chain inside a conversion such as `float64(a*b)` forces that result to round before it can flow into
this or any later statement and defeats fusion, because an explicit conversion to the target type must
round. Inner multiplication nodes in the same chain need no separate conversion. Applied throughout
`internal/domain`, paired with the ban on `math.FMA` and the closed `math` allowlist, this makes results
bit-identical on every target while keeping the mathematics readable as the real arithmetic this
document is written in, and keeping `float64`'s dynamic range — which §7 leans on heavily, given how
many passages here worry about products or totals exceeding what `float64` can represent. The cost is
a small loss of readability and a negligible loss of speed. It is a discipline no reviewer can hold
by hand, so §5 makes it a build failure rather than a convention.

**Integer state is not the general alternative.** `AllocationBP` is a basis-point vector because it has
an **exact invariant** — five entries summing to exactly 10,000 — that drift would destroy. That is
the test for when fixed-point earns its cost, and most simulation state fails it: `Health`,
`Degradation`, and the heritable vector are continuous scalars with clamps and stated tolerances, and
nothing conserves them. Population is the deliberate second exception: it is a count, so durable
state is `uint32`, splitting is integer-conservative, and the explicitly rounded output of each
continuous demographic checkpoint returns to that count type. Resource stocks, their allocations,
and `StoredFood` remain genuine candidates only if their conservative-roundoff tests begin failing;
converting those allocators to integer apportionment with a deterministic largest-remainder rule
would then be appropriate because exactness would be the point.

**Quantizing after floating-point arithmetic is not a determinism mechanism.** Computing an allocation in
`float64` and rounding the result into a fixed-point stock does not by itself remove divergence; it makes it
discrete and rare. Two targets differ by roughly one ULP, so the rounded values differ only when the
true result lands within one ULP of a quantization boundary — a probability on the order of the
ULP divided by the quantum, which across a 400-turn campaign's operation count yields something that
happens in a small fraction of campaigns rather than never. A determinism defect that reproduces once
in a thousand runs is harder to diagnose than one that always reproduces, and harder to trust than one
that cannot happen. Population rounding is safe only because the product-rounding and cross-target
checkpoint gates first require its entire floating demographic input to be bit-identical; conversion
then supplies the separate gameplay invariant that a person count is discrete. It is not evidence
that partially converting food or resource arithmetic would make those systems deterministic.

Everything below preserves that cross-target contract whether or not the run is interrupted by save
and reload.

- `population-food-turns-v1` automatically consumes each band's actual harvested FU plus its
  post-spoilage reserve up to the start-of-turn requirement, once in phase 3. It cannot leave a
  food deficit alongside usable reserves, consume another band's food, or debit extraction twice.
  Derive the bounded `FoodDeficitFraction` once from that deficit and the fixed requirement;
  demographics cannot change its denominator, and the fraction is not itself a mortality rate.
  The combined phase-3 update under `HazardAlgorithm` carries prior normalized `Health float64`
  forward: add the nutritional contribution, subtract water, endemic-disease, and genetic-burden health losses,
  then clamp once to `[0, 1]` before vulnerability and demographics. Never clamp nutrition alone.
  The nutritional contribution is `-HealthLossRate * FoodDeficitFraction` on shortage, otherwise
  `+HealthRecoveryRate`, with initial rates `0.20` and `0.05`. Water loss is
  `WaterHealthLossRate * WaterDeficitFraction`, with initial rate `0.40`, using the origin's
  heat-adjusted requirement fixed before allocation and the actual water allocated. The demand
  multiplier follows current phase-1 local heat, including seasonal and long-term changes, within
  `[1.0, 1.3]`. Do not apply the multiplier again to the fraction or damage.
  Endemic loss evaluates the origin's camp and non-camp health rates in that order, each with
  its applicable health-technology and innate-immune remaining-risk factors; only the camp term also receives hygiene.
  Neither term receives a cave bonus, population scaling, or a mortality-derived rate.
  The combined health supplies `1 + (1 - Health)` in `[1, 2]` to chronic components only.
  In phase 5, only a selected `DiseaseOutbreak` damages survivors' health once after direct
  mortality. Interpolate initial bounds `0.05` and `0.15` using the same raw severity draw `v`,
  then apply its explicit health-technology and genetic remaining-risk factors. No camp protection or mortality-
  derived loss fraction enters this hit. It cannot replay phase 3 or retroactively change its
  vulnerability.
  These health updates add no RNG draw, food/water consumption, or population-loss term.
  Snapshot, split, and save/load preserve current health; percentage formatting never writes it
  back. Planning, movement, food finalization, and save/load cannot repeat either checkpoint,
  and splits preserve the parent's condition. The squared starvation response uses the frozen
  requirement/population and post-reserve fraction once, directly as `raw_starvation`, with no health threshold, delay,
  health multiplier, or RNG, using initial `StarvationCoefficient = 0.10`. Positive logistic
  growth alone is scaled by `1 - FoodDeficitFraction`; non-positive growth is unchanged. Growth
  uses no extra food/RNG and does not revise the frozen requirement. Applied starvation uses
  the existing shared mortality cap, never a
  second subtraction. The remainder is carried through the turn without
  an early capacity clamp.
  Source order, saving, loading, and UI actions cannot change this accounting or cause another meal.
  `LastFoodReport` and `LastOutcomeReport` copy phase-3/final actuals only at completed-turn
  publication, remain fixed during ordinary planning, and survive save/load. Splits clear both descendants' reports;
  new-game and cleared reports are unavailable rather than zero. It adds no RNG draw or future
  simulation input, and extinct bands leave no report archive.
  `FoodStorageAlgorithm` discards overflow once after consumption and all population changes,
  using the final population and finite remaining FU reserve. The stable, bounded pass consumes no
  RNG and runs before every completed-turn frame, including terminal turns. Planning commands
  preserve the resulting storage bound; snapshot/save/load never repeat the discard or manufacture
  food. Valid reloads preserve the absolute reserve, while invalid above-cap saves are rejected.
  The same storage contract applies fixed-percentage spoilage once at the start of phase 1, after
  planning succeeds and before climate, harvest, or consumption. It uses only carried-over FU;
  fresh harvest is not exposed until the next advancing turn if still stored. Its two constants are
  **Initial** in Appendix C. There is no RNG or wall-clock
  aging, and saving/loading cannot apply spoilage or
  reset food age because no age state exists.
- **`WorldRNG` owns a `*rand.PCG` and nothing else from `math/rand/v2`.** It does not hold a
  `*rand.Rand`, and `rng.go` may name only `rand.NewPCG`, the `*rand.PCG` methods `Uint64`,
  `MarshalBinary`, and `UnmarshalBinary`, and the `rand.PCG` type itself. Every step between the
  seed and a simulation draw is therefore project-owned code in `rng.go`, versioned by
  `RNGAlgorithm`, and covered by the same fixtures as any other domain function.

  This is deliberate and it is the reason `*rand.Rand` is excluded. PCG's *state* is pinned by
  `MarshalBinary`, but the mapping from PCG's `uint64` output to a `float64` in `[0, 1)` lives in
  `math/rand/v2` if `Rand.Float64` supplies it. Go's compatibility promise covers API, not the
  numeric output of a v2 generator's convenience methods, so a toolchain bump could silently change
  every acute-severity draw, every mutation-emergence draw, and with them every golden fixture,
  every checked-in save, and the cross-target checkpoint hashes — with no compile error and no
  failing test until the fixtures themselves were regenerated against the new behavior. §13 states
  the determinism contract as unconditional for a supported release; that claim is only true if the
  whole chain from seed to draw belongs to this repository. Pinning the toolchain in CI bounds the
  blast radius but does not make the sentence true.

  **Seed expansion.** `SplitMix64` is a pure, versioned function in `rng.go` over a `uint64` state,
  using wrapping arithmetic throughout:

  ```text
  SplitMix64(state) -> (nextState, output)
      nextState = state + 0x9e3779b97f4a7c15
      z         = nextState
      z         = (z XOR (z >> 30)) * 0xbf58476d1ce4e5b9
      z         = (z XOR (z >> 27)) * 0x94d049bb133111eb
      output    = z XOR (z >> 31)
  ```

  A new-game `uint64` world seed is expanded by two successive calls — `state := seed`, then two
  applications, taking each `output` in order — to produce PCG's two 64-bit seed words. This
  replaces the earlier `rand.NewPCG(seed, seed ^ 0x9e3779b97f4a7c15)` rule, which had two defects
  the expansion removes. That rule left the first word entirely unmixed, so the four low-entropy
  corpus seeds `0`, `1`, `2`, and `3` differed only in the two lowest bits of a shared pattern; and
  it sent `ReferenceSeed = 0x9e3779b97f4a7c15` — the single most heavily exercised seed in the
  entire suite, and the one whose checkpoints gate every release — to a second word of exactly zero.
  Neither is incorrect, because PCG's 128-bit LCG decorrelates regardless, but a seeding rule that
  degenerates precisely on the value the release gate uses is a coincidence worth not having.
  `SplitMix64`'s output stage is a bijection on `uint64` applied to two distinct states, so the two
  words are always distinct and can never both be zero.

  Fixtures pin the expansion for the exact `BalanceSeedCorpus`, in corpus order:

  | World seed | PCG word 1 | PCG word 2 |
  | ---------- | ---------- | ---------- |
  | `0x0000000000000000` | `0xe220a8397b1dcdaf` | `0x6e789e6aa1b965f4` |
  | `0x0000000000000001` | `0x910a2dec89025cc1` | `0xbeeb8da1658eec67` |
  | `0x0000000000000002` | `0x975835de1c9756ce` | `0xbfc846100bfc1e42` |
  | `0x0000000000000003` | `0x1d0b14e4db018fed` | `0xb3466f8a7b81a989` |
  | `0x9e3779b97f4a7c15` | `0x6e789e6aa1b965f4` | `0x06c45d188009454f` |
  | `0xd1b54a32d192ed03` | `0x2d0f28c7e7e786b2` | `0x75856f745165f252` |
  | `0x94d049bb133111eb` | `0xbb1fd59c964c1554` | `0xee4b135308a7ae87` |
  | `0xffffffffffffffff` | `0xe4d971771b652c20` | `0xe99ff867dbf682c9` |

  **Unit draw.** `WorldRNG.Float64()` is the only floating-point draw, and `rng.go` defines it
  directly over PCG output:

  ```go
  func (r *WorldRNG) Float64() float64 {
   return float64(float64(r.pcg.Uint64()>>11) * 0x1p-53)
  }
  ```

  The shift takes the high 53 bits, and `0x1p-53` is an exact power of two, so the scaling is exact
  on every target and the result is uniform over the 2^53 representable multiples of 2^-53 in
  `[0, 1)`. The outer conversion is §5's product-rounding rule, which applies to `rng.go` like every
  other `internal/domain` file; it is redundant here because the product cannot round, and it is
  written anyway because the gate is syntactic by design and a reader should not have to re-derive
  the exemption. Validation asserts the closed `[0, 1)` range, that a maximal `Uint64` yields
  `1 - 2^-53` rather than `1`, and that a zero `Uint64` yields exactly `+0`.

  `WorldRNG.MarshalBinary` supplies the opaque PCG bytes
  that the application mapper copies into `SaveState.RNGState`; restore receives those bytes and
  calls `UnmarshalBinary` before any simulation work. The expansion runs only at `NewWorld`; a
  restored world resumes from marshalled state and never re-expands its seed. Save, snapshot,
  rendering, and metadata generation consume no simulation randomness.
- Every stateful simulation draw goes through `WorldRNG`; package-level random functions and direct
  random imports outside `rng.go` are lint- and architecture-test failures (§5). `UnitNoiseV1` and
  `UnitResourceAbundanceV1` are the two counter-based helpers: both are implemented in `rng.go`, are
  pure functions rather than consumed draws, and use distinct domain constants from PCG seeding and
  from each other. Resource initialization further separates region, tile, and stock keys.
- Genetics uses no random draw for inheritance, gene flow, selection, or background drift. Only
  rare mutation emergence consumes `WorldRNG`, after acute-event resolution, in ascending surviving
  band-ID and trait-enum order. All trait effects in the completed turn read its frozen
  start-of-turn vector; phase-5 gene-flow, selection, and mutation deltas are computed from frozen
  inputs and committed simultaneously for the next turn. Contact construction emits each automatic
  same-species edge and accepted interbreeding edge once. Saving, loading, snapshots, Field Notes,
  candidate display, and rejected interbreeding commands consume no draw and cannot change a trait.
- Tiles and bands are iterated in stable order at every sequential phase of the pipeline. Shared
  resource allocation is additionally tested for permutation invariance, so stable iteration cannot
  conceal first-band advantage. Go's randomized map iteration order would otherwise silently
  desynchronize a reloaded save from an uninterrupted run — a bug that would surface as "loading a
  save changes the future", intermittently.
- `region-biome-v1` derives prey opportunities from fixed region and current biome. Seed, species,
  band order, or acquired technology cannot mutate the environmental mix; only the supported lookup
  and climate-derived biome select it. Harvesting and work-risk capture use the origin after climate
  update; candidate previews use the destination, and phase-5 work risk retains its origin inputs.
  Every animal-food role competes for one stock, with no per-group refill, depletion history, or
  additional RNG consumption. Profile changes and reloads must preserve that accounting.
- `linear-shared-fauna-v1` derives combined terrestrial/aquatic Hunting demand from start-of-turn
  hunters and the origin's current profile/acquired-technology group rates. One allocator shares fauna across both roles
  and species using full potential demands; neither role has first access or a duplicate stock.
  Its fixed aquatic-rate share conservatively partitions the actual Hunting allocation before the
  selected food conversion; only actual allocations reach conversion, once. No random harvest roll, hidden
  saturation, destination catch, or retroactive technology bonus is added.
- `linear-megafauna-v1` contributes tracking workers times the origin's current accessible-megafauna
  rate to that same allocation. The rate exceeds ordinary terrestrial hunting's under equivalent profile/tech
  inputs when available; unsupported or inaccessible prey gives zero demand. The higher rate
  does not add stock, bypass scarcity, increase a food-conversion factor, or consume RNG. Added
  field risk still uses the separate origin-work hazard contract.
- `linear-share-work-risk-v1` uses the accepted hunting-role basis-point shares directly, not
  absolute workers or catch amounts. Fixed origin-profile coefficients give identical work-risk
  vectors at equal shares regardless of population. Capture once before demographics, add once
  to the existing uncovered acute components after movement, and apply technology and the shared
  cap only through their established channels. There is no camp mitigation, severity modifier,
  crossing-risk contribution, or extra RNG draw from this rule.
- The closed five-role basis-point vector is evaluated once from the accepted start-of-turn planning
  state. Its five unsigned entries sum exactly to 10,000; population changes never mutate it, and a
  split copies it. Each entry routes only to its named consumer, at most 1,280 entries are evaluated,
  and role evaluation consumes no RNG. `linear-shared-flora-v1` derives full potential demands from
  start-of-turn foragers and origin biome/acquired-tech rates, then shares the regrown flora once
  across both species. It neither clips individual demands before proportional allocation nor
  grants uncollected food, destination harvest, or a random yield bonus. Shelter's workforce input
  comes directly from the accepted basis-point share, not absolute workers or a ratio using the
  survivor population; equal shares imply equal labor-derived mitigation fractions in otherwise
  equivalent conditions, including
  `NaturalShelter`. The captured share remains fixed through demographics and migration; terrain
  efficiency is evaluated separately at the origin in phase 3 and final tile in phase 5. Fixed
  ratings derive from versioned geography without RNG, and migration cannot carry a cave bonus.
  `saturating-share-terrain-v1` applies the same finite configuration inputs and overflow-safe ratio in each
  phase; each class's labor mitigation stays at or below `MaxShelterMitigation < 1`. Class profiles
  partition risk rather than duplicating it; only exposure uses natural terrain, security and
  hygiene use their closed scales on the terrain-zero curve, and uncovered components receive
  zero camp-work mitigation. Technology and camp work combine once per component through the
  directly evaluated `(1 - tech) * (1 - camp)` factor, with current acquired technology and no
  double subtraction of the natural-terrain bonus. Acute camp protection acts only on probability
  components before the shared cap and event draw, never on severity. Stable component order, the four-class bound, and
  the unchanged one-or-two-draw contract preserve determinism without introducing a camp roster
  or infection state. Different allocations may change whether an event occurs and therefore
  whether its severity draw is consumed; reload equivalence compares identical planning choices,
  not campaigns with different shelter assignments.
  The share is discarded after acute resolution; prior shelter work cannot affect a later turn
  except through already-persisted outcomes such as survival. Any balance tuning of the selected
  role formulas must retain these allocation, terrain, and lifetime invariants.
- The archaic policy captures and processes IDs in ascending order, reads closed region/biome presets and
  technology priorities, uses the already-stable migration ranking, and consumes no RNG. Tests
  shuffle backing storage before planning and require the exact same computer command batch and
  post-turn state. Player commands always precede that batch, so neither map iteration nor UI frame
  timing can decide which species receives the last available band slot.
- Technology contact pairs are derived in stable band-ID/edge order from one frozen post-event
  snapshot. `co-located-cross-species-v1` filters a pair solely by its snapshot species and final
  tiles: same species may be co-located or ordinary-adjacent, while different species must be
  co-located. All diffusion writes apply simultaneously. For each recipient and technology,
  `stacked-acquired-v1` deduplicates fully knowledgeable source IDs, counts them, and computes one
  aggregate `source_count * 0.20 * ResearchCost[k]` gain before the threshold clamp only when the
  recipient's prerequisites were met in that same snapshot; otherwise the gain is zero. Neither
  algorithm consumes RNG; their result is invariant to contact-pair discovery order and cannot
  relay newly learned technology or use a newly learned prerequisite during the same turn.
- `saturating-toolcraft-v1` reads only the derived worker count from the accepted start-of-turn
  proportional allocation, active target, and validated finite configuration inputs. Phase 3 records one transient
  gain per start-of-turn band; phase 5 discards gains for extinct bands, combines each survivor's
  gain with the order-independent diffusion gain in one expression, and normalizes the threshold
  once. Original research consumes no RNG, does not depend on mortality or contact iteration order,
  and cannot create a same-turn knowledge source.
- Enforced by golden tests: seed 42 → 100 turns → canonical campaign-state hash, versus save at
  turn 37 → reload →
  63 more turns → identical hash; the fixture exercises bounded integers, floats, chronic attrition,
  both the no-acute-event and triggered-event RNG paths, regional achievement latching, passage
  eligibility/queues, eight-way ordinary-edge queues, and shuffles so cached or partially consumed
  RNG state cannot hide. It also compares every population, `Health` value, `LastFoodReport` and
  `LastOutcomeReport` (including availability, turn, endpoints, and causal components), and mortality breakdown,
  acute and macro events and eviction, RNG state, every climate component, active/announced macro
  episode summaries, reconstructed natural-shelter ratings and phase-specific
  mitigation, directed ordinary-edge costs/candidates, passage view,
  established-region set, exact archaic actions, every band's technology/progress and six-value
  heritable state, technology and genetic contact graphs, accepted interbreeding/spatial-action state,
  band-cap boundary behavior, exact exploration bitset, and `NextBandID`, so restoring the PCG while
  losing the world seed,
  campaign-clock/exploration/policy/assignment/shelter/ownership/research/contact/diffusion/genetics versions, movement or
  interbreeding intent, event history,
  foraging, hunting, and megafauna versions/rates/allocations, including shared per-role fauna allocation,
  hunting-risk version and captured share-based work weights,
  fauna-profile version/derived summaries/role eligibility,
  achievement history, or band allocation state cannot pass.
  A second test repeatedly saves and
  reloads the same run and proves that saving itself neither advances the turn nor consumes a draw.

---

## 8. Rendering

### 3D terrain: bounded chunks, one shared material, per-snapshot recolor

Tetra3D transforms vertices on the **CPU**, so a naive model-per-tile at 6,144 tiles would become
the performance ceiling of the whole game — and WASM is slower than native. A single whole-map
`MeshPart` is unsafe too: 6,144 top quads contribute 12,288 triangles, and a maximally stepped 96×64
grid can add more than 24,000 wall triangles, exceeding the 21,845-triangle range of Tetra3D v0.18's
`uint16` display indices.

The 96×64 terrain is therefore divided into six spatial **32×32 chunks**. Each chunk has its own
`Mesh` and one `MeshPart`, and all chunks share one `Material`. The worst-case triangle count of a
chunk, including walls assigned along its outer edge, is below 6,500. `terrain.go` nevertheless
defines `MaxTrianglesPerPart = 21_000` and fails construction if any chunk crosses it, so changing
grid or chunk dimensions cannot silently wrap an index. The count and six-chunk partition are tests,
not performance assumptions.

- Each tile contributes a flat-topped quad at `y = Tile.ElevationKm * heightScale`, plus side-wall quads
  only where it borders a lower neighbour. Flat tops keep tiles visually discrete, which a strategy
  game needs for unambiguous clicking.
- Biome color is written to a `VertexColorChannel` with `VertexActiveColorChannel` set. Geography
  fixes elevation, so only colors change when a new simulation snapshot arrives; vertex heights are
  not rewritten every turn.
- The normal mode material has `Shadeless = false` and uses one directional light. Tetra3D's default
  depth rendering remains enabled, so each visible chunk has depth and color work; the design claims
  six bounded render batches, not one literal GPU draw call.
- The terrain-detail toggle rebuilds six top-only meshes, their colliders, and lookup tables once;
  low-detail material has `Shadeless = true`, so it performs neither side-wall nor lighting work.
  Switching back rebuilds the normal meshes from the unchanged elevation grid. Normal detail is the
  **Policy** launch default; the toggle is session-local and does not add a fifth `UISettings` field.
  Both detail modes must pass their own release floors. Low detail is a player fallback and a useful
  diagnostic, not a remedy for a failing mandatory normal-detail floor. A reviewed launch-default
  change updates Appendix C explicitly and does not waive either gate.

`camera.ColorTexture()` is blitted to the Ebitengine screen, then the 2D HUD is drawn on top with
`text/v2` and `vector` — a crisp 2D interface over a 3D world.

### High-DPI viewport and coordinate spaces

Desktop and web use one automatic DPI-aware viewport contract. `pkg/app.Game` implements
`ebiten.LayoutFer`; Ebitengine v2.9 supplies `LayoutF`'s `outsideWidth` and `outsideHeight` in
device-independent pixels (DIPs). `LayoutF` reads `ebiten.Monitor().DeviceScaleFactor()` after game
startup, never from `init`, and returns a physical-resolution game screen:

The desktop host enables OS window resizing and maximization. Its initial window is the largest
16:9 rectangle that fits within 90% of the current monitor's DIP width and height, preserving room
for desktop chrome; if the monitor dimensions are unavailable, it falls back to `1280 × 720` DIPs.
The initial-size policy affects presentation only and does not constrain subsequent user resizing,
fullscreen behavior, simulation state, saves, hashes, or RNG.

```text
rawScale = Monitor().DeviceScaleFactor()
if rawScale is non-finite or <= 0:
    rawScale = 1

RenderScale   = clamp(rawScale, 1, MaxRenderScale)
RenderWidthPx = max(1, ceil(outsideWidthDIP  * RenderScale))
RenderHeightPx= max(1, ceil(outsideHeightDIP * RenderScale))

ScaleX = RenderWidthPx  / outsideWidthDIP
ScaleY = RenderHeightPx / outsideHeightDIP
```

`MaxRenderScale = 2.0` is a presentation **Policy**. It gives a Retina-class 2× backing surface
while bounding color/depth targets and fill work to four times the 1× pixel count. A transient
non-positive outside dimension during minimization retains the last valid viewport; before the first
valid layout it returns `1 × 1`. Scale and dimensions are never inferred from the world frame.

`pkg/app` receives a presentation-only `DeviceScaleSource func() float64`; production composition
wraps `ebiten.Monitor().DeviceScaleFactor`, while tests inject exact values. This seam is not a domain
or application port and may not appear below `pkg/app` or `pkg/render`.

`render.Viewport` is one immutable value containing logical DIP dimensions, render-pixel dimensions,
`RenderScale`, `ScaleX`, `ScaleY`, and a monotonic presentation-only `ViewportRevision`. `LayoutF`
atomically publishes a new value only when one of those inputs changes; repeated calls with the same
inputs preserve the revision. `Update` and `Draw` each load one complete value, so a monitor move or
browser resize cannot expose mixed old/new dimensions. `ViewportRevision`, like font and camera
caches, is absent from `gameapi.Frame`, `WorldRevision`, `TerrainRevision`, saves, hashes, and RNG.

All HUD layout constants, responsive breakpoints, scroll distances, and hit rectangles are authored
in DIPs. The supported minimum gameplay viewport is **`960 × 600 DIPs`**; smaller windows show a
resize overlay and suspend every gameplay/scene action except resize and application exit, without
advancing or resizing the simulation. The narrow-layout
breakpoint is **`1,100 DIPs`** wide. Drawing maps DIP positions through `ScaleX`/`ScaleY`; one-DIP vector rules snap their edges
to the nearest physical-pixel boundary after scaling. `fonts.go` caches exactly the three current
logical face sizes multiplied by `RenderScale` and replaces that three-face set when scale changes,
rather than accumulating one cache entry per resize or monitor. Text measurement and drawing use the
same scaled face, so labels do not clip merely because glyphs became sharper.

Ebitengine pointer and touch positions are in its returned game-screen coordinate space. The input
router therefore keeps both representations for the current event:

```text
PointerRenderPx = Ebitengine cursor/touch position
PointerDIP      = (PointerRenderPx.x / ScaleX, PointerRenderPx.y / ScaleY)
```

Widgets and HUD exclusion zones consume `PointerDIP`; Tetra3D ray picking consumes
`PointerRenderPx` against a camera whose color/depth targets equal `RenderWidthPx × RenderHeightPx`.
No handler may compare render pixels with a DIP rectangle. The same logical point must hit the same
control and map tile at 1×, fractional scale, and 2×.

A viewport change reallocates the Tetra3D camera color/depth targets and screen-sized presentation
buffers once, replaces the scaled font set if necessary, and recomputes responsive HUD geometry. It
does **not** rebuild terrain chunks, veil topology, colliders, band markers, or any simulation frame;
the camera projection alone adopts the new physical aspect ratio. Resize or monitor movement emits
no UI action, storage operation, sound, revision outside `ViewportRevision`, or RNG draw.

High-DPI scaling is automatic, not another `UISettings` field. The existing terrain-detail toggle
still controls geometry and lighting rather than pixel density. If either 2× detail mode misses its
release floor, optimize it or explicitly revise the measured policy; lowering `MaxRenderScale`
remains a separately reviewed presentation-policy change. The build must not silently change
simulation grid dimensions, launch detail, or introduce frame-time-dependent dynamic resolution.

### Release performance contract

Performance is a measured release property, not an impression from one developer machine. Step 8
adds deterministic maximum-workload benchmarks for `World.AdvanceTurn` and frame projection; their
workloads use 6,144 tiles and 256 bands and report `ns/op`, bytes, and allocations. The same process
runs a fixed pure-Go calibration benchmark with no game imports. `tools/check_benchmarks.sh` takes five
samples, compares the median target/calibration time ratio plus bytes/op to the checked-in step-8
baselines, and records the raw measurements for diagnosis. On the pinned Go toolchain and runner
image, a greater-than-25% normalized-time regression or bytes/op increase fails the release lane
unless the baseline change is reviewed together with the responsible code and this contract.
Unnormalized wall-clock time remains telemetry, avoiding a gate that mistakes GitHub-host hardware
variation for a game regression.

`NativeBenchmarkReference` is the GitHub-hosted `ubuntu-24.04` x86-64 runner
(`linux/amd64`). Baseline and release records also capture the runner's reported image version, so an
image refresh is visible even though the stable OS label and architecture remain fixed.

The automated Chromium smoke test serves the release bundle locally and requires the canvas-ready
signal within 10 seconds and each scripted new-game, end-turn, quick-save, reload, and restored-frame
checkpoint within 5 seconds. Its timeouts detect hangs and gross regressions; they are not an FPS
benchmark. The interactive reference machine is a 2024 Mac mini (`Mac16,10`) with an Apple M4
10-core CPU/integrated GPU, 16 GB RAM, and macOS 26.6.1; `docs/PERFORMANCE.md` additionally records
the exact browser version used by the release candidate. Step 13 loads the checked-in
`testdata/performance_profile_save.json` maximum-render fixture (`ReferenceSeed`, turn 300, 256 live
bands, every tile explored), uses a 1280×720-DIP viewport, warms for 5 seconds, then runs the exact
30-second `tools/web-e2e/profile.mjs` orbit/pan script with one `EndTurn` every 5 seconds. It records
each DPR/detail combination separately. At DPR 1, normal detail must sustain a median 20 FPS and low
detail a median 30 FPS; at DPR 2, normal detail must sustain a median 15 FPS and low detail a median
20 FPS. Every profile must have a
95th-percentile frame gap no greater than 150 ms and a maximum successful `EndTurn` latency no
greater than 2 seconds. **All four profiles are mandatory.** The release record names the CPU, GPU, OS, browser, browser version, detail
mode, DPR, median FPS, frame-gap percentile, turn latency, and peak process memory so later results
are comparable. Missing a floor requires optimization or an explicit revision of this policy; it
cannot silently reduce the simulation grid, omit bands, skip turn work, or publish anyway.

Pixel-golden and screenshot fixtures inject and record an explicit render scale; they never derive
their expected dimensions from the test runner's monitor. Manual screenshots may use the host scale,
but comparisons are valid only between captures with the same logical viewport and render scale.

### Epoch grade: atmosphere, not terrain

The three climate epochs shift the scene's palette, but never per-tile biome color. The grade
applies to the directional light's color and intensity, ambient level, the veil's blue-grey, water
tiles, and HUD chrome accents. Biome vertex colors remain the authoritative answer to what a tile
_is_.

That split is required, not stylistic. `TerrainRevision` advances only when a tile's biome,
`Explored` bit, or visible macro-impact factors change; a grade multiplied into vertex colors would
dirty all six chunks every turn and reintroduce exactly the per-turn recolor cost that counter
exists to avoid. It would also overload one channel with two meanings, leaving desertification and
epoch drift visually indistinguishable. Applied to lighting and chrome instead, the grade is a
per-frame uniform: no mesh rebuild, no cache invalidation, no revision increment.

`palette.go` gains a pure `EpochGrade(aridityIndex) -> GradeColors`. `GradeColors` has exactly six
outputs: directional-light RGBA, directional-light intensity, ambient level, veil RGBA, water RGBA,
and HUD-chrome-accent RGBA. Three anchor records sit at `AridityIndex` `0.0`, `0.5`, and `1.0`:

| Anchor             | Character                                                                                 |
| ------------------ | ----------------------------------------------------------------------------------------- |
| `0.0` humid        | warm sunlight, higher ambient level, saturated blue water, green-gold chrome              |
| `0.5` transitional | pale sunlight, middling ambient level, dusty blue water, ochre/olive chrome               |
| `1.0` glacial      | cool sunlight, lower ambient level, steel-blue water, slate veil, ice-blue chrome accents |

The selected **naturalistic, restrained** anchors are exact. Each hexadecimal color is eight-bit sRGB
`RRGGBBAA`; every alpha is opaque `FF`:

| `AridityIndex` | Directional light | Intensity | Ambient | Veil        | Water       | HUD accent  |
| -------------: | ----------------- | --------: | ------: | ----------- | ----------- | ----------- |
|          `0.0` | `#FFE8BCFF`       |    `0.94` |  `0.72` | `#485860FF` | `#206C9CFF` | `#9E9E48FF` |
|          `0.5` | `#F1DAB6FF`       |    `0.82` |  `0.60` | `#505A60FF` | `#386884FF` | `#A68042FF` |
|          `1.0` | `#CFE0F0FF`       |    `0.70` |  `0.48` | `#4C5868FF` | `#345474FF` | `#70AACCFF` |

Interpolation converts each channel to normalized `[0, 1]`, linearly interpolates in sRGB channel
space without gamma conversion, rounds to the nearest eight-bit channel with halves away from zero,
and linearly interpolates the two scalar levels as `float64`. The exact segment-midpoint fixtures are:

| `AridityIndex` | Directional light | Intensity | Ambient | Veil        | Water       | HUD accent  |
| -------------: | ----------------- | --------: | ------: | ----------- | ----------- | ----------- |
|         `0.25` | `#F8E1B9FF`       |    `0.88` |  `0.66` | `#4C5960FF` | `#2C6A90FF` | `#A28F45FF` |
|         `0.75` | `#E0DDD3FF`       |    `0.76` |  `0.54` | `#4E5964FF` | `#365E7CFF` | `#8B9587FF` |

Both scalar levels must be finite and in `[0, 1]`. Build step 8 implements and screenshot-reviews
these values but does not select replacements. This is a presentation contract rather than a
cross-platform simulation-hash input; the exact anchor and midpoint fixtures still prevent an
accidental palette change.

The grade reads the continuous index, not the discrete epoch, so it never steps at a boundary; the
epoch name changes only the caption. `scene3d.go` applies the grade to light and ambient, while
`veil.go` and `hud.go` read colors from the same function. Appendix C marks the anchor table Locked
and points here without duplicating its values; the two epoch thresholds and the hysteresis margin
are climate configuration and appear separately in C.11.

Color is never the sole cue. The current recurring climate-epoch name appears as text in the top bar
without a date range; the separate campaign era carries its fixed date range. The timeline marker's
shape, position, and textual date communicate progress without color. `AridityIndex` rides the frame alongside the existing climate
components so render never reimplements the curve.

### Exploration veil and hidden-map behavior

`veil.go` turns `Tile.Explored` into six bounded chunk-aligned cover meshes. Every unexplored tile
contributes one opaque top quad at a common `VeilHeight` above the maximum possible terrain height;
boundary edges add vertical skirts below the minimum terrain height so orbiting the camera cannot
expose hidden elevation, coastlines, biome color, or side walls from a shallow angle. The worst case
is two top triangles plus eight skirt triangles per tile, or 10,240 triangles in a 32×32 veil chunk,
below `MaxTrianglesPerPart = 21_000`. The veil therefore adds at most six bounded mesh parts rather
than one object per tile.

The veil material is opaque, shadeless, and a quiet blue-grey from `palette.go`, with a restrained
boundary treatment rather than animated clouds. It needs no transparency sorting, texture asset,
clock, or RNG. Meshes rebuild only when the explored bitset changes; ordinary biome recolors,
selection, camera movement, and idle frames do not rebuild them. Newly explored terrain appears in
the next accepted frame with a short presentation-only fade permitted at the cutout edge, but save,
load, screenshots, and deterministic state hashes depend only on the authoritative bit, never on a
fade timer.

Unexplored tiles have no marker, selection ring, biome/resource label, tile inspector, natural-
shelter cue, fauna summary, destination highlight, or full passage line. Picking may still resolve
the underlying tile ID for bounds checking, but input discards it when `Explored == false`; hidden
tile clicks cannot open an inspector or become a migration action. A passage with neither endpoint
explored is absent. With exactly one endpoint explored, the map may show a local locked/closed glyph
at that endpoint but no line or far endpoint; the full overlay appears only after both endpoint bits
are set. All ordinary and named-passage destinations exposed in a sapiens band's candidate list are
required by §6 to be explored, so masking cannot strand the player without a selectable next move.
An explored archaic band may retain hidden-target candidates in its isolated frame for the computer
policy, but its read-only inspector omits those rows; filtering cannot change policy rank or movement.
The same filter applies to macro episodes: a global atmospheric episode may be named, but its hidden
epicenter, hidden impact masks, tile factors, and per-band archaic outcomes are omitted until the
relevant tiles are explored. Filtering only the frame copy never changes the authoritative event
catalog, episode resolution, bounded event feed, or computer policy.
Regional abrupt-climate magnitudes follow the same exploration rule: the current pulse may be named
globally, but numeric values for wholly unexplored regions are not player-facing.

This is an exploration veil, not tactical fog-of-war. Once a tile is explored, its current terrain,
public stats, and resident band markers remain visible from anywhere. The camera may orbit or pan
across the whole rectangular world extent, but unexplored cover reveals no geographic shape beneath
it. Developer-only `-dumpmap`, `-headless`, and test fixtures may inspect the full authoritative map;
they are not player-facing discovery paths.

### Gameplay stats layout

The same 2D HUD layout applies on desktop and web, over the 3D map:

- **Persistent top bar:** total living `HomoSapiens` population across all player bands, campaign
  turn/year, campaign era with its fixed date range, season, and current climate epoch name without
  a date range, plus any current or player-relevant warned macro episode. Climate epochs can recur
  and therefore must never be displayed as though each owns one chronological interval. The total excludes
  archaic bands and does not change merely because a different band or tile is selected. A thin
  campaign timeline runs immediately beneath this bar.
- **Selected-band inspector header:** the selected band's population and current health.
  Health is per band, not a global average. Both species are inspectable; archaic bands retain
  the “Computer controlled” label and expose no player commands.
- **Band-selection keys:** `Tab` and `Shift+Tab` select the next and previous sapiens band,
  respectively, wrapping at either end. Both clear any UI-local keyboard migration preview before
  changing selection, and the persistent controls legend names both directions. The compact side
  panel reserves five band rows and renders the five-row page containing the selected sapiens band;
  selecting the first band on another page replaces the visible page immediately. Its header shows
  the one-based visible range and total (for example, `Sapiens bands 6–10/17`), so later bands are
  never silently hidden and the selected band is always visible. Archaic bands count in neither the
  range nor the total. This page is derived from the selected ID and accepted frame and adds no
  independent scroll position or saved UI state.
- **Persistent terrain legend:** the strip immediately above the map shows a swatch and a short
  liveability explanation for each of the six biome classes, plus open water and unexplored terrain.
  Biome swatches use the same current palette function as their map tiles, not a duplicated set of
  approximate colors. The accompanying text names the characteristic resource opportunity and
  principal environmental hazard; water says it cannot be occupied and unexplored terrain says its
  details are not yet known. The legend therefore remains meaningful without color perception.
- **Reachability overlay:** when the selected sapiens band still has its spatial action, every tile
  in its authoritative `MigrationCandidates` list receives a cyan outline and the first-ranked
  candidate receives a gold outline. A persistent legend explains both colors, and turn-0 Field
  Notes call out the outlines explicitly. The overlay disappears after the spatial action is spent;
  it never marks merely adjacent but currently ineligible tiles.
- **Queued-migration marker:** after a sapiens migration is accepted and before the next turn
  resolves it, draw a thin red arrow from the band's current tile center to the queued destination
  center. The arrow shape and persistent “red arrow: queued” legend make the meaning non-color-only.
  It is driven exclusively by the frame's queued-destination presence/value pair, remains visible
  when another band is selected, and disappears from the replacement frame when the move resolves
  or is canceled. Never infer an intent from `SpatialActionUsed` or the candidate ranking, and never
  expose a computer-controlled archaic intent through this player-planning overlay.
- **Keyboard migration:** arrow keys move a UI-local destination cursor inside the selected band's
  one-turn `3 × 3` neighborhood; successive cardinal presses can therefore select any corner without
  requiring a diagonal key. The cursor may pass through a currently ineligible cardinal tile so the
  player can reach a valid corner behind it, but `Enter` emits `QueueMigration` only when the final
  tile is in the authoritative projected candidate list. `Esc` clears the cursor, returning it to
  the band's origin also clears it, and ending the turn is blocked until the player confirms or
  clears an outstanding choice. The preview is UI-local and never enters `World`, `Frame`, or a save;
  the existing red arrow remains frame-driven after confirmation. Pointer clicks still queue an
  eligible destination immediately, and non-adjacent named passages remain clickable. The persistent
  controls legend names the arrow, `Enter`, and `Esc` bindings.
- **Keyboard splitting:** plain `N` sends `SplitBand` toward the selected band's first-ranked
  eligible ordinary-land candidate. Named passages are never split destinations. The domain still
  owns population, stress, spatial-action, band-cap, adjacency, and habitability validation; a
  rejection or the absence of an ordinary-land candidate produces player-facing feedback rather
  than changing Field Notes or failing silently.
- **Rejected-destination feedback:** clicking any tile absent from that authoritative candidate list
  produces a short, player-facing explanation instead of silently doing nothing or repeating a
  generic “choose an outlined tile” message. `pkg/ui` classifies the frame projection into spent
  spatial action, current tile, unexplored area, open water, currently uninhabitable land, named
  passage technology/climate lock, blocked diagonal, too-distant tile, or no traversable route.
  The unexplored check precedes terrain inspection so this feedback never reveals whether hidden
  geography is land or water. Bands cannot occupy water tiles: Coastal Navigation unlocks only the
  eligible named land-to-land Wallacea passages, so its water message directs the player to a named
  passage endpoint rather than implying arbitrary sea movement. Beringia's message instead explains
  its climate gate. These diagnostics explain the current accepted frame; the candidate list remains
  the sole authority for whether `QueueMigration` may be sent.
- **Band details below the header:** current food reserves and the “Last turn” food report,
  the last completed turn's starvation/seasonal/chronic/macro/acute mortality breakdown, workforce
  assignments, and research.
  For sapiens, keep the five-role editor and research controls alongside the stats they help explain.
- **Persistent research-key legend:** list all nine numbered technologies by name. For the selected
  sapiens band, color available targets normally, the current target gold, acquired technologies
  green, and prerequisite-locked technologies grey. These states come from the frame's projected
  `ResearchOptions`, not a prerequisite table duplicated in presentation code. The legend remains
  visible even when Field Notes are hidden, so pressing `1`–`9` is never an unexplained action.
- **Tile inspector and migration comparison:** one persistent column describes the selected band's
  current tile and a second describes the UI-local arrow-key destination cursor, or the accepted
  queued destination when no cursor is active. Each known, habitable land column shows biome,
  region, local temperature, combined and source-split food stocks, water stock/cap, current
  `EcologicalK`, degradation, natural shelter, movement cost, and the selected band's projected
  seasonal/chronic mortality rates for that tile. It also exposes the current regional abrupt-climate
  anomaly, current visible macro-impact factors, and the already-specified regional prey summary in
  the expanded inspector. Distinguish undegraded `BaselineK` from current `EcologicalK`; neither is
  an extra population resource. The compact comparison labels the cursor destination reachable only
  when it appears in the authoritative `MigrationCandidates`; invalid water and uninhabitable land
  explain their state. An unexplored destination says only that its details are hidden and must not
  reveal land, water, biome, resource, hazard, or passage information.

Display current population, `Health`, `StoredFood`, and environmental values from the accepted
frame. The top-bar population is a display-only sum over at most `MaxBands = 256` band values;
it adds no saved aggregate or second source of truth. Health displays `100 * Health` as a labeled
0–100% condition score, using the frame's normalized `float64` value. For example, `0.375` means
37.5% health; it does not mean 37.5% of the band is alive or healthy. Display rounding/decimal
formatting is presentation-only: compute the percentage from `float64`, never write a rounded
score back to state, and never reinterpret it as a mortality rate. Food reserves use FU; shortage
explanations must distinguish missing FU, the fraction of needs unmet, and
applied starvation deaths. Show the saved last-mortality breakdown, not the uncapped starvation
base; splitting clears that breakdown under the existing split rule.

Population labels use no fractional digits because the frame contract permits only whole-person
values; this is direct formatting, not presentation rounding that conceals fractional state.

Each visible band row appends the exact signed last-turn population change and the health change in
percentage points when `LastOutcomeReport.Turn != 0`; current health uses one fractional percentage
digit so a small real decline is not rounded back to an unexplained `100%`. For the selected band,
show a persistent explanation for each negative net change. Population causes rank negative
logistic growth as **crowding / habitat limits** alongside the saved starvation, seasonal, chronic,
macro-event, and acute mortality actuals. Health causes rank negative nutrition as **food shortage**
alongside water shortage, endemic disease, adaptation trade-offs, macro-event, and outbreak health
losses. Display the two largest positive cause magnitudes in stable order and append “+N more” when
needed. The signed net changes come only from the report's endpoints; never sum rounded cause labels
to reconstruct them. Omit a cause line when that metric did not decline. Population mortality and
health are independent channels, so `Health = 1` must not hide or contradict seasonal, chronic,
macro, or acute population loss.

For an available `LastFoodReport` in the frame — one whose `Turn` is nonzero — label the food
subsection **“Last turn · Turn N”** using its recorded turn. Show **Required**, **Consumed**, and
**Shortfall** in FU, with the shortfall's percentage of needs unmet beside it; consumption and the
percentage come from §7's two display derivations, computed here rather than stored. Keep **Current reserves** separately labeled: those are
post-turn/current stored food, not the food available or consumed during the reported turn.
For example, a report of 50 required, 40 consumed, and 10 missing FU shows a 20% shortfall even
if deaths have since reduced the band to 45 people. A fully fed band may finish with zero reserves;
that is zero shortfall, not evidence of starvation. Display rounding must not change stored values
or turn a small positive shortfall into a claim of no shortage.

When `Turn` is zero, show “No completed-turn food report” and omit the numeric actuals rather
than rendering zeroes. This covers new games and both descendants after a split; §7 defines
replacement after a turn and preservation across planning/save/load. A workforce edit may change
the current research preview, but cannot change these historical food actuals. Never reconstruct
last turn's shortage from survivors or reserves, rerun consumption in the renderer, or present
these actuals as a prediction. Detailed future-turn food-output previews remain outside this
historical report and are not part of the v1 UI contract.

Selection, hover, camera movement, and drawing only inspect the frame. Dirty workforce drafts
leave accepted stats unchanged; refreshing a frame follows the existing Apply/Discard and load
guards. Completed-turn outcomes appear only after the full turn, not during phase 3. Panel
placement/sizing and responsive details remain presentation work except for the Field Notes contract
below; the top bar, band header, band details, and tile-inspector grouping above are required.
The selected-band details include all six heritable values with plain-language current-effect and
local-pressure summaries. A co-located sapiens selection lists eligible archaic interbreeding
partners and makes clear that ordinary co-location exchanges no genes. Preserve passage status,
migration ranking, established-region display, event feed, and save feedback alongside these stats.

### Campaign timeline rail

The timeline is a thin, persistent, full-width rail inside the gameplay HUD's safe horizontal
insets, directly below the top bar and above the map/inspector content. It contains a quiet base
track, a subtle elapsed fill, the seven major ticks and four low-contrast era segments defined by
the campaign clock, and a distinct current-position marker with a compact `CurrentYearBP` label. Its
purpose is chronological orientation, not precision turn selection; the top bar remains the exact
turn/year/era readout. `CalendarProgress = 0` maps to the left 80,000 BP endpoint and
`CalendarProgress = 1` maps to the right 20,000 BP endpoint. The marker is linearly interpolated
between them, and the elapsed fill runs from the left endpoint to the marker.

Era boundaries at 50,000, 35,000, and 25,000 BP may be thinner than the 10,000-year tick marks, but
the active segment and its date range remain readable in the top bar without calling an era a
development phase. The rail is strictly display-only. It has no hit target, hover requirement,
focus state, drag or scrub behavior, time-skip command, forecast, selection side effect, RNG use,
revision increment, or saved field. It reads the frame's projected `CurrentYearBP`,
`CalendarProgress`, and campaign era rather than recomputing the clock from `Turn`; macro glyphs use
the projected warned/current/elapsed summaries, and the current
pulse accent uses the projected bounded abrupt-offset vector. The only static input is the Toba
presentation-catalog entry. It reads no live domain state and derives no future catalog disclosure.
Pointer input inside its reserved HUD strip is ignored: clicking or dragging it changes neither the
campaign clock nor map selection.

At narrow widths, the 80,000 BP and 20,000 BP endpoints and the current-date label remain visible;
the five intermediate 10,000-year text labels may disappear while their tick marks and the era-
boundary segmentation remain. Layout prevents the current label from clipping or colliding with an
endpoint. When the current date exactly equals a
major tick, one shared label serves both the current marker and that tick rather than drawing
duplicate text. The elapsed fill is not the sole cue:
marker shape/position and the textual current date communicate progress without color. The rail
never shifts to the lower edge or overlaps the lower-edge Field Notes panel, whether that panel is
open, hidden to its tab, or presented as a narrow-screen drawer.

When a macro episode becomes active, the rail draws one small non-interactive glyph at its calendar
date and retains it as elapsed historical context. It does not reveal future catalog entries; the
one-turn player-relevant warning appears as a top-bar alert and on already explored affected tiles
instead. Glyphs and warnings come from the accepted frame, have text/icon redundancy, and add no
timeline hit target, simulation action, or saved timeline state.
When the campaign crosses `73,880 BP`, the rail similarly adds the Toba context glyph from the
presentation catalog, but it emits no warning or simulation event and never appears in active macro
summaries. Before that date it remains hidden under the same no-future-entry rule.
An active abrupt-climate pulse may add a low-contrast warm/cold accent to the current marker, but no
future pulse is drawn and exact regional magnitude remains in the explored tile inspector.

### Field Notes: context, abstraction, and hints

A persistently available **Field Notes** text panel is visible by default and docked along the lower
edge of the gameplay HUD. It has a capped responsive height and its own scroll position, and it must
not cover the top bar, selected-band controls, tile inspector, or required alerts. The top bar has a
book-button toggle; plain `F` performs the same action when no text-editing control has keyboard
focus. Hiding the panel leaves a small labeled tab that can restore it. On narrow windows, the panel
may become a lower drawer, but it retains the same visible/hidden states and never replaces a
simulation inspector.

The displayed entry follows a stable context priority:

1. a focused heritable trait, technology, passage, or available interbreeding action;
2. a technology newly acquired by a sapiens band in the most recently completed turn and not yet
   acknowledged;
3. a regional achievement newly latched by the most recently completed turn and not yet
   acknowledged by subsequent user focus/selection/action;
4. a current abrupt-climate pulse or current/warned macro episode affecting the selected band/tile;
5. the selected band;
6. the selected tile's named region and current biome;
7. the newest completed-turn event when there is no selection; otherwise
8. the campaign overview.

`pkg/app` detects newly acquired sapiens technology bits only across an accepted `EndTurn` frame
transition, considering the selected band first and then stable band/technology order. A loaded
frame and technology inherited by a newly introduced split descendant do not replay a discovery.
The first discovery becomes the Field Notes focus, a toast reports it and any additional count, and
the panel or its hidden restore tab receives a gold breakthrough treatment for 600 UI update ticks.
This treatment does not force open a panel the player explicitly hid. After the accent expires, the
latest technology entry remains available as ordinary Field Notes context; all of this state is
UI-local and absent from saves.

`pkg/ui` sets one `AchievementFocusPending` value from the newest newly latched achievement event in
the accepted frame, choosing stable region order when several latch together. It persists for at
least one rendered update and then until the next explicit focus, selection, simulation action, or
panel hide; it is UI-local and never saved. This lets an achievement surface even when band/tile
selection remains persistent without overriding a focus the player just chose.

Each bundled entry contains three visibly separated blocks: **Historical context**, **Game
abstraction**, and **Hint**. Historical context summarizes the best-supported evidence and labels
material uncertainty; Game abstraction states exactly where the mechanic compresses, combines, or
departs from that evidence; Hint explains an actionable relationship without revealing hidden RNG
or guaranteeing an outcome. Compact author/year references accompany the historical block. The
initial catalog includes every region, biome, technology, heritable trait, passage, acute event,
species, interbreeding opportunity, regional achievement, abrupt-climate pulse, presentation-only
timeline marker, and macro episode used by v1. It is packaged with the binary, works completely
offline, and makes no network request. Source-sensitive examples include archaic admixture and
immune introgression, the UV/pigmentation tradeoff, abrupt
climate pulses, and every macro episode in the release catalog. A
catalog test rejects missing keys, empty blocks, or entries without at least one reference where a
scientific claim is made.
The initial source baseline includes Reich et al. (2010) for archaic admixture, Dannemann et al.
(2016) for introgressed TLR variation, and Jablonski and Chaplin (2010) for UV/pigmentation.
Aquatic-subsistence entries cite Yellen et al. (1995) for the Katanda bone points and fish remains,
O'Connor, Ono, and Clarkson (2011) for 42,000-year-old pelagic catch and the later 23,000–16,000-year
shell hook, and McNiven et al. (2012) when explaining why the game's generic trap/weir improvement
is not evidence for Australian stone-walled traps during the campaign. The historical and
abstraction blocks must keep those three claims and dates separate.
Climate entries additionally cite Lisiecki and Raymo (2005) for the marine isotope stage framework
that anchors the epoch thresholds, Clark et al. (2009) for the Last Glacial Maximum definition,
Rasmussen et al. (2014) for the selected GI onset chronology, and
Capron et al. (2021) for the varied anatomy of abrupt last-glacial warmings. Macro-event entries cite
the USGS for pyroclastic-flow scale and volcanic-climate duration, Giaccio et al. (2017) for the
Campanian Ignimbrite date, Scarpati et al. (2020) for the mapped local deposits, and Smith et al.
(2016) plus Pyle et al. (2006) for the wider eastward dispersal. The Toba context entry cites Storey et al. (2012) for its date, Lane et al.
(2013) for the absence of a catastrophic East African volcanic-winter signal in Lake Malawi, and
Kappelman et al. (2024) for adaptive behavior around the ash horizon in the Horn of Africa. It must
present Toba population effects as uncertain rather than a settled extinction bottleneck and state
plainly that the marker has no gameplay effect.

Field Notes consumes only selection/focus state and closed `gameapi` enums/IDs from the current
frame. The historical prose and hints live in `pkg/ui`, never `internal/domain` or the world memento, so
they cannot become simulation rules by accident. Changing the focused entry, toggling visibility,
or scrolling never changes band/tile selection, a workforce draft, `WorldRevision`, `WorldRNG`, the
save payload, or any action queue.

Presentation preferences persist locally across application sessions in a versioned `UISettings`
record; they are intentionally absent from `SaveState`, slot metadata, campaign hashes, and
cloud/export semantics. The unreleased v1 record has four required JSON fields:
`SchemaVersion = 1`, `FieldNotesVisible bool`, `MasterVolume float64`, and `Muted bool` (§11).
Desktop stores it in
`os.UserConfigDir()/africa2ice/ui_settings.json`; web stores one record in a separate
`africa2ice-ui` IndexedDB database so world-save locking and migrations remain independent.

Defaults are **per record, not per field**: an absent, malformed, or unsupported-future-version
record supplies all three preferences at once — Field Notes visible, `MasterVolume` `0.5`, `Muted`
false — and a v1 record is accepted only when all four required fields are present, non-null, have the exact JSON
types above, and `SchemaVersion == 1`. Unknown extra fields are ignored. A syntactically valid object
with a missing, null, or wrongly typed required field is malformed and defaults as a whole; ordinary
Go zero-value decoding is not presence validation. Because v1 is unreleased, an older development
record containing only `FieldNotesVisible` is deliberately defaulted rather than migrated.
`MasterVolume` must be finite and is clamped to `[0, 1]` on both read and write, so a hand-edited
preference file cannot produce negative or above-unity gain. Field Notes defaults toward _visible_
and audio defaults toward _quiet_: an unreadable preference must not surprise the player with
full-gain audio, which is the opposite of the visibility bias and is deliberate.

Preference writes are best-effort and non-blocking; failure keeps the session's current values and
reports a non-fatal toast. Desktop goroutines and JavaScript callbacks enqueue only
preference-store completions; `game_scene.Update` polls them before touching panel/toast state, so an
asynchronous completion never mutates UI from another callback or thread. Loading or deleting a
campaign never changes any of them.

Writes use one active operation plus one `pendingLatest` complete record. A control change updates
the live UI/audio value immediately. If no write is active it starts one; otherwise it replaces
`pendingLatest` rather than appending. Each request/completion carries a UI-local preference
revision. Completion may report success or failure but never installs its older record into the
current UI. After completion, start `pendingLatest` once when it differs from the completed record;
otherwise become idle. Thus rapid slider/toggle input is bounded and last-value-wins even when
completion order is delayed. Failure shows one toast but does not revert the current preference;
the next user change supplies the next retry.

Until the initial read settles, presentation renders the three defaults but every preference-
mutating control — the Field Notes visibility toggles, master-volume slider, and mute checkbox — is
disabled with a compact “Loading preferences…” label. The completion atomically installs either the
validated stored record or the complete default record before enabling those controls. A write can
therefore never race the initial read or overwrite stored preferences that the player had not yet
seen.

### Orbit camera and picking

`orbit.go` stores `focus`, `azimuth`, `elevation`, and `distance`, and derives the camera transform
each frame. Right-drag or `Q`/`E` rotates azimuth; middle-drag or `R`/`F` changes elevation (clamped
against gimbal flip and sub-horizon views); scroll changes distance; `WASD`/arrows pan the focus.

For each chunk, `terrain.go` calls `Mesh.UpdateBounds()` and constructs a
`tetra3d.BoundingTriangles` from the same mesh, with broadphase enabled and the same transform as
the visible model. `picking.go` calls `camera.MouseRayTest` with `TestAgainst` set to the collection
of those six colliders. A `RayHit` against `BoundingTriangles` contains the exact triangle; a
`map[*tetra3d.Triangle]int` built with the mesh maps both top and wall triangles to their tile (a wall
belongs to its higher tile).

This is more robust than inverting the orbit transform by hand and stays correct at any camera
angle. Tests shoot known screen/world rays at a top, a side wall, a chunk boundary, ocean, and empty
space in both detail modes. Rebuilding detail mode must replace the collider collection and lookup
atomically before the next input update.

---

## 9. Persistence

One asynchronous `application.CampaignRepository` outbound port, two build-tagged storage adapters,
and identical versioned JSON payloads. Starting an operation returns immediately with an operation
ID; adapter completion or failure enters the application service's private FIFO and is surfaced
through `game.PollStorage()` on the `gameapi.Game` port to `pkg/app`. The domain knows none of these
types. A failed save never advances the turn and never replaces the last committed slot.

### Slot policy, autosave, and feedback

The valid slot IDs are closed constants, not arbitrary integers:

- `Manual1`–`Manual3`: IDs `1`, `2`, `3`. Chosen from Pause → Save; selecting a row begins the
  overwrite immediately with no confirmation dialog.
- `QuickSave`: ID `99`. `Ctrl+S` on Windows/Linux or `Cmd+S` on macOS during gameplay begins a
  save immediately; the shortcut never opens the manual save scene.
- `Auto1`–`Auto3`: IDs `101`, `102`, `103`. The application chooses the target; the player may load
  or explicitly delete these rows but cannot save to them manually.

On desktop startup, `pkg/app` first lists slots asynchronously and loads the record with the highest
`CommitSequence` among Quick and Auto 1–3 before accepting gameplay input. Manual slots are
explicit checkpoints and never selected by auto-resume. When no Quick or Auto record exists, startup
keeps the new campaign; a list/load failure likewise leaves it playable and reports the storage
error. Browser startup does not auto-load because an origin can host multiple independent sessions
and retains the explicit load flow. The desktop host handles the OS close request: if a player has
begun a quick-save whose completion has not yet been polled, it keeps updating storage with gameplay
input disabled and returns `ebiten.Termination` only after that operation succeeds or fails. Thus
Ctrl/Cmd+S followed immediately by Cmd+Q/window close cannot truncate the requested write.

`SlotMetadata` includes `SlotKind`, `WorldRevision`, `CampaignClockAlgorithm`, turn/year/era,
`uint64` population totals by species,
`CommitSequence`, and the integrity fields below. The save/load browser displays Manual, Quick, and
Auto groups even when rows are empty. Overwriting is always non-modal. Deletion requires the
selected row plus an explicit DEL/BACKSPACE action, but opens no confirmation modal.

`ToastManager` reports completion, never mere request submission: “Saved Manual 1”, “Quick-saved”,
“Autosaved — Auto 2”, or a concise failure. Each toast remains for two seconds. Its FIFO is capped at
four entries and coalesces identical messages; errors displace the oldest success when full. A small
pending icon identifies an active storage operation without blocking play or opening a modal.

Application-owned `WorldRevision` increments on every accepted player planning mutation and once
for every completed `EndTurn` use case, and is included in the immutable save snapshot. The internal
archaic batch and five-phase `World.AdvanceTurn` are part of that single turn increment; no
intermediate computer decision creates an externally observable revision.
Workforce draft edits are not planning mutations; only a successfully applied complete vector
increments the revision. A save made while a draft exists therefore serializes the accepted world
allocation and revision, never the UI-local draft. Quick-save and autosave do not resolve or discard
the draft; draft edits alone do not trigger the revision-based autosave fallback. Actual load
requests must pass the dirty-draft guard before starting or joining the storage queue (§4).
`internal/application/autosave.go` requests an autosave after every completed turn. It also requests
one when five minutes of monotonic running time have elapsed since the last successful autosave and
the current revision is newer; this is the fallback for long planning periods. On load, the
scheduler adopts the loaded revision as its baseline and starts a fresh five-minute interval.

Autosave scheduling is bounded: there may be one active storage operation and at most one
`autoNeeded` flag, never one queued save per trigger. If a turn completes while storage is busy,
`autoNeeded` remains set; when storage becomes idle, the scheduler captures the **latest** revision
once. Empty autosave slots are filled in ascending order; afterward the slot with the lowest
`CommitSequence` is overwritten, with slot ID as the tie-breaker. Completion records the operation's
captured revision, not the then-current revision. If play advanced meanwhile, `autoNeeded` remains
set for one follow-up snapshot. Failure leaves the prior autosave valid, shows a toast, and retries
only after another turn or the five-minute fallback—not on every Ebitengine update.

### Wire contract

`SaveState` is a versioned application-layer persistence DTO, not the aggregate and not a
`gameapi.Frame`. To save, `GameService` exports an immutable `domain.State` memento and explicitly
maps its value types into `SaveState` before calling `CampaignRepository`; the repository adapter
only JSON-encodes/decodes and commits or retrieves that DTO. To load, the application validates the
decoded DTO's schema and algorithm compatibility, maps it into a candidate domain memento, and calls
`domain.RestoreWorld` so every aggregate invariant is rechecked. Only then does it swap the live
world. No JSON tag, slot ID, schema version, or migration branch appears in `internal/domain`.

`SaveState.SchemaVersion` starts at `1`. The state includes `WorldSeed`,
`CampaignClockAlgorithm: "four-era-v1"`,
`GeographyAlgorithm: "dispersal-map-v2"`, `ClimateAlgorithm: "hybrid-abrupt-moisture-v1"`,
`NaturalShelterMaskAlgorithm: "authored-ellipse-v1"`,
`TemperatureAlgorithm: "lat-elev-offset-v1"`,
`MacroEventAlgorithm: "bounded-regional-v1"`,
`ExplorationAlgorithm: "sapiens-frontier-v1"`,
`BandAlgorithm: "fixed-half-global-cap-v1"`,
`ArchaicPolicyAlgorithm: "ranked-pressure-v1"`,
`AssignmentAlgorithm: "proportional-basis-points-v1"`,
`FoodStorageAlgorithm: "population-food-turns-v1"`,
`FoodConversionAlgorithm: "normalized-source-v1"`,
`ForagingAlgorithm: "linear-shared-flora-v1"`,
`HuntingAlgorithm: "linear-shared-fauna-v1"`,
`MegafaunaAlgorithm: "linear-megafauna-v1"`,
`HuntingRiskAlgorithm: "linear-share-work-risk-v1"`,
`FaunaProfileAlgorithm: "region-biome-v1"`,
`ShelterAlgorithm: "saturating-share-terrain-v1"`,
`TechnologyOwnershipAlgorithm: "band-local-v1"`,
`ResearchProductionAlgorithm: "saturating-toolcraft-v1"`,
`KnowledgeContactAlgorithm: "co-located-cross-species-v1"`,
`KnowledgeDiffusionAlgorithm: "stacked-acquired-v1"`,
`HeritableStateAlgorithm: "band-six-trait-v1"`,
`GeneticSelectionAlgorithm: "trait-functions-v1"`,
`GeneFlowAlgorithm: "local-reciprocal-v1"`,
`MutationAlgorithm: "rare-emergence-v1"`,
`MovementAlgorithm: "eight-way-no-water-corners-v1"`,
`MovementCostAlgorithm: "destination-vegetation-v1"`,
`PassageAlgorithm: "named-asymmetric-v1"`, `ResourceAlgorithm: "toward-cap-v1"`,
`HazardAlgorithm: "split-v1"`,
`KinSupportAlgorithm: "saturating-kin-acute-v1"`,
`PopulationRoundingAlgorithm: "stochastic-v1"`, the campaign turn and terminal result,
`ExploredTiles[ExplorationWordCount]`, the sorted unique
`SapiensEstablishedRegions`, every non-derived tile and band field including `Species`, acquired-tech
bitset, research target, fixed research-progress vector, fixed
`HeritableState[HeritableTraitCount]`, the fixed
`AllocationBP[AssignmentCount]` vector, `StoredFood` in FU, current `Health`, `LastFoodReport`,
`LastOutcomeReport`, the fixed last-mortality breakdown, and `SpatialActionUsed`, the monotonic `uint64 NextBandID`, queued migrations with
band ID, recorded origin, target, edge kind, and optional
passage ID, queued interbreeding intents with sapiens actor, archaic target, and recorded co-location
tile, each tile's resource stocks and `Degradation`, the bounded chronological event feed,
`RNGAlgorithm: "pcg-splitmix-v1"`, and the serialized `WorldRNG` bytes. That identifier versions the
`SplitMix64` seed expansion, the `WorldRNG.Float64` mapping over raw PCG output, and the marshalled
PCG state together, because all three sit between a world seed and a simulation draw.
`CampaignClockAlgorithm` versions the 80,000/20,000 BP endpoints, four 100-turn era intervals and
their three internal boundaries, their 300/150/100/50-year turn spans, and the piecewise
`CampaignDate`/`CalendarProgress`
conversion. Save only the authoritative campaign turn plus the algorithm ID in world JSON; year and
era are derived. `SlotMetadata` retains its denormalized display year and era, which validation
requires—under its matching supported clock identifier—to equal `CampaignDate(metadata turn)` and
`EraForTurn(metadata turn)` before showing the preview. Changing any boundary, date, or
turn span after release requires a new clock version and an explicit migration because the climate
curve, macro-episode activation, and every displayed date depend on it. No genetics rule reads the
clock: mutation emergence is ungated, so cutting `HbS` removed the clock's only genetics consumer.
`GeographyAlgorithm` versions the grid projection and land/water, region, highland, river, and fixed
base-moisture geography, including §6's numeric elevation catalog, `HighlandElevationKm`, land
clipping, maximum-overlap rule, and checked elevation checksum. Elevation is reconstructed from that
supported geography and never serialized. A geography change therefore also changes every downstream
temperature, highland-biome, orographic, UV, hypoxia, and render-height result that reads it, without
requiring those consumers to duplicate the authored height table under their own identifiers.
`ClimateAlgorithm` versions the LGM/orbital curve, the long-term moisture curve with its exact
precession table, `RegionalAridityWeight`, the epoch thresholds and `EpochHysteresis`, the authored
abrupt-pulse catalog and composition,
12-turn seasonal term, counter-based seed/turn noise, and `BeringiaOpenFraction` with its
attainability check. `TemperatureAlgorithm` separately versions the absolute-temperature function,
the exact 64-row `LatitudeSinSquared` table, and its `EquatorTempC`, `PolarTempC`, and
`LapseRateCPerKm` coefficients. That split matters because
biome history, the water-demand multiplier, endemic-disease row selection, and cold-exposure genetics
all read absolute °C: changing the temperature coefficients silently reclassifies every tile in an
existing save, so it requires its own version bump and migration even when the climate curve is
untouched. Derived temperatures are never serialized. Moisture takes the opposite decision for the same
reason: `ClassifyBiome` is its only consumer, so it needs no identifier of its own and stays inside
`ClimateAlgorithm`. Changing the moisture curve, its precession table, or the aridity weights still
reclassifies every tile in an existing save, so it carries the same version-bump-and-migration
obligation. Derived moisture, `AridityIndex`, and `ClimateEpoch` are never serialized. `BandAlgorithm`
versions the `uint32` population representation and `MaxPopulation`, the fixed 50/50 split,
`MinSplitSourcePopulation`, `MaxBands = 256`, and `MaxArchaicBands`; the sub-cap constrains the computer policy's split branch only, so a loaded world
whose archaic count already exceeds a lowered value remains valid and simply takes no further archaic
splits. `MacroEventAlgorithm` versions eruption
dates, impact masks, warnings, refugium rule, phase-2 resource/habitat factors, phase-5 mortality and
health application, and their strict non-sterilization bounds. These catalogs and episode
summaries are derived from the saved turn, world seed where applicable, and supported geography;
only their mutable outcomes and bounded historical event entries are saved. Loading neither replays
an episode already represented by the saved turn nor restores resource/health state from catalog
defaults.
`ArchaicPolicyAlgorithm` also versions the warning-triggered whole-band escape override and its
strictly-lower-impact selection rule; changing that response after release requires a policy-version
update even when the macro catalog itself is unchanged.
`ExplorationAlgorithm` versions East Africa initialization, adjacent-water coastline reveal, the
3×3 sapiens frontier, eligible named-passage endpoint reveal, monotonicity, and reveal timing. The
fixed bitset is authoritative campaign knowledge and is serialized even though it cannot affect
simulation results. Loading never recomputes it from current band positions, which would erase
historical exploration.
The selected ownership, production, contact, and diffusion models require no species-level research
record, generic research pool, or contact history. `MaxResearchPerTurn` and
`ResearchHalfSaturation` are selected configuration inputs `20` and `25` of
`saturating-toolcraft-v1`; `DiffusionRate = 0.20`
is a Locked constant of `stacked-acquired-v1`. They are configuration, not mutable save data, and
none of these algorithms adds an unbounded work or interaction log.
`HeritableStateAlgorithm` versions the enum membership/order, `[0, 1]` representation, standing
variation, exact split inheritance, and species-identity rule. `GeneticSelectionAlgorithm` versions
the per-trait effect and selection functions, including the UV formula, its season/elevation factors,
and the derived hypoxia input. UV derivation reads `TemperatureAlgorithm`'s exact
`LatitudeSinSquared` table without taking ownership of it; changing that shared table changes both
absolute temperature and genetic selection behavior, so its version/migration review must cover both
consumers. `GeneFlowAlgorithm` versions automatic same-species contact, active co-located
cross-species contact, population-weighted simultaneous mixing, and its three rate constants.
`MutationAlgorithm` versions eligibility, the `MutationProbability` and `MutationEntryFrequency`
arrays, stable draw order, and the absence of a date or species gate. Only each band's six current
values are durable genetic state; selection
pressures, genotype shares, contact pairs, partner means, deltas, and mutation outcomes are not
serialized separately. A planning save must preserve accepted interbreeding intent and spatial-action
use so reload cannot grant a second move or split. No ancestry log, pedigree, or species-level gene
pool is saved. Unknown identifiers are rejected.
`FoodStorageAlgorithm` versions automatic per-band consumption and its normalized deficit in
phase 3, population-scaled capacity, end-of-turn overflow discard, fixed-percentage spoilage at
phase-1 start, and the storage constants listed under `FoodStorageAlgorithm` in Appendix C.
These are versioned configuration, not per-save mutable settings. `StoredFood` stays in FU; capacity, food-turn ratios, and transient harvested,
available, remaining, discarded, or spoiled amounts are never serialized separately. The one
explicit display exception is `LastFoodReport`: it preserves the completed turn's requirement and
deficit against that turn number, as specified in §7; consumption and the unmet fraction are
derived for display and never serialized.
Those historical values are not saved hunger state or inputs to the next meal; lasting health
consequences are represented by persisted `Health`. All saves capture canonical
planning or completed-turn state, never a pre-cap or partially resolved
food intermediate. Loading rejects unsupported versions, invalid configuration (including a
non-finite spoilage rate or one outside `(0, 1)`), non-finite capacity, and reserves outside
`[0, FoodStorageCapacity(b)]` after any explicit supported-version migration. It does not rerun
spoilage or consumption, apply offline decay, finalize a turn, or clip a reserve.
`NaturalShelterMaskAlgorithm` versions the nine-entry ellipse catalog, inclusion equation,
land clipping, maximum-overlap rule, and four permitted ratings. The derived tile ratings are not
serialized; supported geography plus the mask algorithm reconstructs them exactly. Load rejects an
unknown identifier or a reconstructed-mask fixture mismatch rather than silently changing exposure
efficiency in an existing campaign.
`FoodConversionAlgorithm` versions the three-entry `FoodSource` catalog, exact `1.0 FU` base factors,
the Hunting allocation's rate-composition split, source-specific effect channel, stable summation
order, and four-component pro-rata partial-meal attribution used by fatty-acid selection. Source breakdowns and conversion modifiers are transient and never saved; `StoredFood` and
`LastFoodReport` remain aggregate FU. Load rejects an unknown identifier. After release, changing a
source, base factor, split rule, or modifier ownership requires an explicit supported migration.
`Population` is authoritative `uint32` whole-person band state and `BandSave.Population` is a JSON
number decoded directly into `uint32`. The decoder rejects negative, fractional, and overflowing
values before aggregate restoration; it never rounds an invalid save into acceptance. Existing
development saves whose JSON population tokens are whole numbers such as `100` remain wire-compatible,
while old fractional saves are rejected. Slot metadata uses `uint64` sapiens/archaic totals.
`Health` is authoritative `float64` band state in `[0, 1]`, not a value reconstructed from current
food or population. Save it as a JSON number decoded into the typed `float64` field, using a
round-trip representation that preserves its value; do not store a formatted percentage, string,
quantized integer score, or `float32` approximation. It must round-trip unchanged, including in planning and
terminal saves, with finite inclusive `[0, 1]` validation. Loading never rounds, clamps, heals,
applies either health checkpoint, or resets health to the new-game `1.0` value. Missing/invalid state
requires an explicit supported migration or rejection, never an automatic new-game fallback.
`HazardAlgorithm` versions health representation, bounds, initialization, nutritional decline/recovery,
normalized water-deficit damage, endemic/outbreak health channels, the two endemic health rates
and their component-specific health-tech effects, the outbreak's shared-draw linear health loss
and technology/genetics composition points, the combined phase-3 clamp, the phase-5 outbreak checkpoint,
coexistence with direct mortality, and chronic vulnerability `1 + (1 - Health)`. It owns every
constant listed under `HazardAlgorithm` in Appendix C, including the six-row camp/non-camp endemic
health-rate table. It also owns direct quadratic
starvation without health gating/scaling and food-limited positive growth. The
whole-person demographic, macro-loss, and acute-loss checkpoints use `RoundPopulation` under this
contract. The genetics algorithms own the trait factors and burdens that enter these hazard checkpoints.
`ResourceAlgorithm` versions water units, the canonical heat increment, and where the
`AridClimateAdaptation` remaining-heat factor composes; `GeneticSelectionAlgorithm` owns the `0.40`
factor itself. Disease and health-loss rules remain under `HazardAlgorithm`.
These rates and mappings are versioned configuration, not mutable save data. The transient
water-deficit fraction, intermediate health checkpoints, and derived vulnerability are not
serialized. `LastOutcomeReport` is the one display-only exception: it persists `BaseGrowth`, the
signed nutrition contribution, individual health-loss actuals, and population/health endpoints for
the last completed turn. No simulation rule reads those historical values. Current `Health`
continues to persist independently, including any phase-5 outbreak damage. `FoodStorageAlgorithm`
continues to own consumption and the transient deficit fraction. These approved numerical
defaults may be tuned before release.
The starvation base is derived during the next advancing turn, not saved or recomputed on load;
the existing last-mortality fields preserve the applied losses. `LastFoodReport` separately
preserves food actuals, while `LastOutcomeReport` brackets the resulting demographic and health
change without adding a mutable population aggregate or display-induced simulation step. Require
both reports' full value shapes and §7 validity/turn/accounting invariants on every band,
including canonical unavailable reports at new game or after a split. A valid food report's
requirement is historical, so do not validate it against current population or storage capacity.
Both reports round-trip unchanged in planning, completed-turn, and terminal saves on both backends;
no UI-local cache or metadata record substitutes for either. They are included in this
still-unreleased schema v1; the decoder maps an absent `LastOutcomeReport` from an earlier
development save only to its canonical unavailable zero value and never fabricates causes or
endpoints. After release, wire changes require a schema-version update and explicit supported
migration. A migration lacking historical actuals may explicitly produce the applicable canonical
unavailable report, never fabricated zero-shortfall or zero-loss actuals. Absent or malformed data
without such a supported migration is rejected; loading does not infer missing history from current
state.
`AllocationBasisPoints = 10_000` is a closed constant of
`proportional-basis-points-v1`; changing its representation or population/split lifecycle requires a
new assignment-algorithm version and an explicit save migration.
`FaunaProfileAlgorithm` versions the closed region/biome lookup, prey-group
definitions, weights, role support, and profile-specific exploitation inputs. Its identifier is
saved; derived profiles, summaries, and work-risk inputs are not. New-game/load validation checks
complete lookup coverage, group-vector lengths, valid role/group mappings, finite non-negative
weights and finite totals, and the empty-profile contract before using any profile. Unknown
versions are rejected. Because the archaic assignment presets and the collection/hazard modifier
tables are keyed on the profile map, a profile version change must move their identifiers too.
Restoring supported versions, region, climate inputs, and the one fauna
stock reproduces opportunities without saved animal populations or profile overrides.
`HuntingAlgorithm` versions the linear combined terrestrial/aquatic demand formula, group-rate
summation, aquatic catch-share derivation, conservative post-allocation source split, and shared
per-role fauna allocation semantics. Profile contents remain versioned by
`FaunaProfileAlgorithm`, so a change on either side moves both identifiers.
Loading rejects unsupported hunting versions and validates finite non-negative
group-rate inputs/results and the finite `[0, 1]` aquatic share. Rates, shares, demands, per-role
allocations, source splits, and band totals are derived rather than serialized; only their normal
world outcomes persist. This adds no separate catch inventory.
`MegafaunaAlgorithm` versions the linear tracking-worker demand rule, access/rate evaluation,
and higher-per-worker-rate constraint. Profile contents remain under `FaunaProfileAlgorithm`,
shared allocation under `HuntingAlgorithm`, and added work risk under `HuntingRiskAlgorithm` and
the hazard contract. Loading
rejects unsupported versions and invalid rate/access configurations, including a nonzero rate
without accessible megafauna or an available rate not above ordinary terrestrial hunting's.
No additional rate, demand, expedition progress, or catch field is serialized.
`HuntingRiskAlgorithm` versions direct basis-point shares, linear coefficient weighting, the
higher active megafauna coefficient total, and origin capture into uncovered acute components.
`WorkRiskCoeff` values and their ordering remain under `HuntingRiskAlgorithm`; the fauna profile and
technology only gate whether a role has a positive active rate. Acute integration remains under
`HazardAlgorithm`. Loading
rejects unsupported versions and invalid coefficient configuration. Shares, coefficients, and captured work-risk
vectors are derived, not serialized as duplicate allocations or work history.
`ForagingAlgorithm` versions the linear potential-demand rule, shared-flora
allocation semantics, and rate/modifier tables. Its identifier is saved, but derived rates, demands,
and allocations are not; it adds no new mutable band/tile field. Loading rejects unsupported
versions or migrates known older ones explicitly, then uses the ordinary saved inputs to reproduce
future collection. Shelter adds its algorithm identifier but no per-band or per-tile mutable save
field: its share is derived directly from `AllocationBP[Shelter]`, not serialized as a second value
or reconstructed from worker counts. Its transient effort is not serialized, and no
camp level, shelter stock, or maintenance history exists. `NaturalShelter` is reconstructed by tile
ID from the supported geography version, not saved as mutable terrain or band state; changing its
authored mapping after release requires a geography-version migration. `MaxShelterMitigation`,
`ShelterHalfSaturation`, `NaturalShelterEfficiencyBonus`, `CampSecurityScale`, and
`CampHygieneScale` are selected Initial configuration inputs of `saturating-share-terrain-v1`, not mutable save data.
The class coverage mapping, probability-only acute application before the shared cap, and finite
risk-profile partitions, together with the remaining-risk composition rule, are versioned
shelter/hazard configuration, not band-local schedules, hygiene stocks, or infection history.
After loading, each phase reconstructs identical factors from supported
configuration, saved acquired-tech bits and shares, and that phase's tile. The factors and combined display
fractions are derived, not serialized. Other inputs and outcomes use the existing allocation,
population, position, technology, and mortality fields.
`CampaignDate`, campaign era, `CalendarProgress`, `Season`, the four global climate components
(long-term temperature, seasonal, noise, and long-term moisture) with derived `AridityIndex` and
`ClimateEpoch`,
bounded regional abrupt-offset vector, and per-tile total,
warned/current/elapsed macro-episode summaries and per-tile impact factors, `BeringiaOpen`,
`Frame.Passages`, ordinary edges and their step
lengths/directed costs, `ElevationKm`, biome, fauna profiles/`FaunaSummary`, `NaturalShelter`, `BaselineK`,
`EcologicalK`, resource caps, destination membership,
region membership for every tile and band,
per-band `OriginalResearchGainPreview`, `ResearchOptions`, migration-candidate rankings, timeline
labels, and timeline geometry are derived rather than redundantly serialized. The exploration
bitset is the bounded display-state exception because it records past geographic discovery; each
frame derives `Tile.Explored` from it. Queued migrations do
not store their planning-time score or cost. Together these form the complete currently identified
campaign payload: future-affecting mutable state plus bounded historical exploration knowledge.
Saving is allowed during the
planning period after commands have been applied, so queued work must round-trip too. Diffusion
and automatic same-species gene-flow contacts are recomputed from final band positions rather than
serialized; only active player-selected cross-species intent persists during planning. Field Notes
visibility, `MasterVolume`, and `Muted` are separate `UISettings` data, while Field Notes content is
bundled UI data; none appears in world JSON or slot metadata.
Save-browser timestamps and labels live only in metadata and are excluded from campaign-state hashes.
No tile or band `Region` field appears in schema v1; restore and frame projection resolve region
from the supported geography and saved `TileID`. The campaign JSON decoder rejects unknown fields
for a supported schema version, so a stray legacy `Region` member cannot become a silent override;
future additions require the ordinary schema-version/migration path.

Each slot record is a tagged value with common fields `SlotID`, `SlotKind`, `CommitSequence`, and
`Deleted`. When `Deleted == false`, every live metadata field is required: `SchemaVersion`, the
clock identifier, display/revision fields, `Generation`, and world SHA-256. When `Deleted == true`,
all live-only fields—including generation and hash—must be absent; a tombstone is not an empty live
record. The lowercase hex
SHA-256 is also the generation identifier; this makes the world object immutable and content-
addressed without drawing from the simulation RNG. If more than one valid desktop metadata record
survives recovery for a slot, the greatest sequence wins, with metadata-file hash as a deterministic
tie-breaker. The desktop lock-holding process computes the next sequence under its writer mutex as
`max(all valid slot metadata and tombstones) + 1`. IndexedDB reads and increments one global counter
inside the same transaction that commits the generation and slot record. A failed transaction does
not consume a sequence. Sequence overflow fails the operation without mutation. Because sequences
are global rather than per-slot, comparing Auto 1–3 selects their true oldest committed snapshot.

Load first decodes the small version header. A version newer than the executable is rejected without
mutation; known older versions go through explicit migration functions before validation. Validation
checks the checksum, generation, campaign turn in `[0, 400]`, grid dimensions, IDs and references,
enum ranges, finite/non-negative
numeric fields, `Degradation` in `[0, 0.75]`, resource stocks in their recomputed `[0, cap]` ranges,
per-band `StoredFood` in `[0, FoodStorageCapacity(b)]` using the saved population and supported
storage limit, sorted/unique/bounded established regions, and destination/terminal-state consistency.
It requires supported identifiers for all 32 algorithm fields: geography, natural-shelter-mask,
campaign-clock, climate, temperature, macro-event, exploration,
band, archaic-policy, assignment, food-storage, food-conversion, foraging, hunting, megafauna,
hunting-risk,
fauna-profile, shelter,
technology-ownership,
research-production, knowledge-contact, knowledge-diffusion, heritable-state, genetic-selection,
gene-flow, mutation, movement, movement-cost, passage,
resource, hazard, and RNG. Validation also requires `0 <= len(Bands) <= MaxBands`, unique positive
band IDs, and a positive `NextBandID`
greater than every existing ID. Reconstructed natural-shelter ratings must be finite, in `[0, 1]`,
and zero on water; no saved override may replace the versioned map. Every band must have exactly
`AssignmentCount` basis-point entries,
each at most 10,000 and with their `uint32` sum exactly 10,000. Every band's tech bitset must contain
only known enum bits and be transitively prerequisite-closed. Its research target must be absent or
an in-range, unacquired technology whose direct prerequisites are acquired. Each of its `TechCount`
progress values must be finite and lie in `[0, ResearchCost[k]]`; a value equals its threshold if and
only if the corresponding acquired bit is set, and an unacquired technology with unmet prerequisites
must have progress exactly zero. `NextBandID` may equal the maximum `uint64` value in a valid
exhausted-ID save, but `SplitBand` must then reject without mutation.
Every band must also have exactly `HeritableTraitCount = 6` finite heritable values in `[0, 1]`.
A queued migration or interbreeding intent requires `SpatialActionUsed == true`; a false
marker permits neither, and at most one of the two intents may name a sapiens actor. A queued
interbreeding intent must reference distinct living sapiens/archaic bands that remain co-located at
its recorded tile in the externally saveable planning state. A used marker without a queue remains
valid for either descendant of an already-applied split. Completed-turn states contain no spatial
intent and have all markers false.
The exploration bitset must have exactly `ExplorationWordCount` words and include every required
East Africa/coastline initialization tile, every current sapiens 3×3 footprint, and every currently
traversable named-passage far endpoint for a resident sapiens band. Extra in-range set bits are valid
historical discoveries and are never removed or inferred from current occupancy. Archaic positions
create no required bits. A loaded frame exposes the exact validated mask without running reveal
again or consuming RNG.
Validation also checks queued edge/passage identities under the supported movement and movement-cost
algorithms, at most 128 chronologically ordered events with valid kinds/references and finite applied
losses, valid finite non-negative mortality breakdowns, `LastFoodReport` turn and FU bounds, and
`LastOutcomeReport` availability, current-turn endpoints, signed finite growth/nutrition, bounded
health endpoints, and finite non-negative loss components under §7, plus exact PCG state length/format. Only a
fully decoded and validated temporary `World` replaces the running world. Missing or mismatched data
marks the slot damaged in the browser rather than partially loading it.

### Commit protocol

The metadata record is the **commit marker**. Saving slot `N` always preserves this logical order:

1. Serialize and validate `SaveState`; compute its SHA-256 generation.
2. Commit the immutable generation-addressed world record and complete metadata record according to
   the backend-specific atomicity rule below. Completion is reported only after that commit succeeds.
3. Best-effort garbage collection removes unreferenced generations only after the commit. Orphans
   from an interrupted pre-commit write are ignored and cleaned on a later startup/save. Deleting a
   slot commits a higher-sequence tombstone before cleanup, so interruption cannot resurrect an old
   save.

Backends implement those same semantics:

- **Desktop** (`//go:build !js`):
  `os.UserConfigDir()/africa2ice/saves/slot_N_world_<sha256>.json` plus generation-addressed
  `slot_N_meta_<sequence>_<metadata-sha256>.json`. Each new record is written to a unique temporary
  file in the same directory and flushed with `File.Sync`. If the content-addressed world filename
  already exists, validate its exact bytes/hash, discard the duplicate temporary file, and reuse the
  immutable object; a mismatch is corruption and fails the save. Otherwise close and rename the
  world temp to that name. Metadata/tombstone final names are previously unused because their commit
  sequence is new. Sync the directory where the platform supports it. Recovery scans the small
  metadata records and selects the highest valid commit rather than depending on replacement-rename
  atomicity, which is not portable across every desktop OS. A torn or incomplete new record fails
  validation and leaves the prior commit reachable. The asynchronous facade runs file I/O on one
  worker goroutine and reports raw bytes/results back to the application completion FIFO.
- **Web** (`//go:build js`): IndexedDB database `africa2ice`, database schema version `1`, with
  object stores `worlds` (key: lowercase SHA-256 generation; value: JSON bytes), `slots` (key:
  integer slot ID; value: metadata or tombstone), and `control` (key `nextCommitSequence`). Saving
  uses one short-lived `readwrite` transaction spanning all three stores: read and validate the
  global sequence, then enqueue the immutable-world, new-metadata, and incremented-counter `put`
  requests synchronously from that request's success callback while the transaction is active. The
  implementation never `await`s between those requests. Only the transaction's `complete` event
  reports success; an error, quota failure, explicit abort, or failed commit rolls back every write
  and leaves the old slot and counter intact. Loading uses one `readonly` transaction spanning
  `slots` and `worlds` so metadata and its referenced world come from one consistent snapshot.
  Listing slots reads only `slots`.

`onupgradeneeded` is the only place object stores/indexes may change; the IndexedDB database version
is distinct from `SaveState.SchemaVersion`. A connection receiving `versionchange` closes promptly,
releases its Web Lock lease, marks storage temporarily unavailable, and asks the player to reload. A
new client whose upgrade is `blocked` shows a non-modal “close or reload other game tabs” message
rather than hanging the loading screen.

During the loading screen, the app requests the exclusive Web Lock `africa2ice/saves` with
`ifAvailable: true` and, if granted, holds it for the lifetime of the tab. That tab is the sole
writer and may save, delete, and collect unreferenced generations; a second tab remains playable but
its save/delete controls are read-only until it retries and acquires the released lease. If the Web
Locks API is unavailable, IndexedDB writes remain enabled with last-committed-transaction semantics,
but automatic orphan collection is disabled for that session. No Ebitengine `Update()` waits for an
IndexedDB request, transaction event, Web Lock promise, or desktop file operation.

Desktop enforces the same single-writer boundary across processes. At startup `file.go` opens
`saves.lock` in the save directory and attempts a non-blocking exclusive advisory lock held for the
process lifetime: `x/sys/unix.Flock` on Unix-like systems and `x/sys/windows.LockFileEx` on Windows.
Only the lock holder may save, delete, allocate `CommitSequence`, or garbage-collect; another
process may list/read/load complete records but exposes its mutation controls as read-only. The OS
releases the lock after normal close or process death. The process mutex still serializes the lock
holder's worker; under both protections it computes
`max(all valid slot metadata and tombstones) + 1`. Lock acquisition/release, two-process contention,
crash release, and read-only fallback have platform-specific integration tests; a lock error other
than contention makes storage unavailable rather than silently enabling a second writer.

The world/metadata split still lets the save browser preview all seven slots without parsing world
data. Generation addressing plus backend atomicity prevents an overwrite from producing a
new-world/old-metadata pair. Failure-injection tests stop after every file-protocol step and every
IndexedDB request/transaction event; after each failure, a slot must resolve to the complete old
save, the complete new save, or a committed deletion—never a mixture. The IndexedDB fake covers
request error, transaction abort, quota failure, `blocked`, `versionchange`, and completion arriving
on a later game update. Shared tests cover missing world data, truncated/newer metadata, sequence
ties/overflow, global-counter rollback, checksum mismatch, unknown versions, migration, invalid RNG
state, operation FIFO ordering, and deletion interrupted between tombstone publication and cleanup.

---

## 10. Web target

- `build_web.sh --dev` performs a fast `GOOS=js GOARCH=wasm go build` for local iteration.
  `build_web.sh --release` builds with `-trimpath -ldflags="-s -w"`, runs a pinned Binaryen
  `wasm-opt -O3` over the result, and fails if `wasm-opt` is missing. CI uses Binaryen
  **`version_131`**; the Linux x86-64 archive must match SHA-256
  **`b5bf1f0eaf17c63ee588ff7a5954dc8f6ce2c26989051c66f24dfe9ece3e46db`** before extraction.
  Both modes copy
  `$(go env GOROOT)/lib/wasm/wasm_exec.js` into `web/`; Go moved this file out of `misc/wasm`.
- **The size gate measures Brotli, because that is what a browser negotiating with Pages receives.**
  The release job compresses `web/main.wasm` with `brotli -q 11` and enforces
  `brotli bytes <= MaxCompressedWasmBytes`. It additionally records the raw and `gzip -9` sizes,
  because a client that does not send `Accept-Encoding: br` receives the gzip stream, and because the
  gzip number keeps this project's earlier measurements comparable. The `web-release` job installs
  the `brotli` CLI alongside the pinned Binaryen release rather than assuming a runner image ships
  it. Neither number is the exact byte count on the wire: Pages compresses on the fly at an
  unpublished quality, so `brotli -q 11` is a reproducible floor rather than a prediction. The gate
  exists to detect growth against a fixed, reproducible measurement, not to model a CDN.
- **The measured budget ratchets.** The original `5_500_000`-byte ceiling came from a
  pre-implementation dependency skeleton. The first full release build reduced it to the current
  Appendix C value: measured Brotli size plus `CompressedWasmHeadroom`, rounded to the ratchet
  quantum. CI compares both the ceiling and headroom with `tools/wasm_size_budget.env` at the trusted
  pull-request base or pre-push revision and rejects either value if it increased. The original
  ceiling remains only a bootstrap upper bound; it cannot be used to reverse a later reduction.
  Ebitengine, Tetra3D, and `text/v2` dominate transfer size, while `wasm-opt -O3` primarily changes
  decompressed size and startup, so optimizing application and dependency reachability remains the
  route for reclaiming compressed bytes.
- `web/` is the complete deployable root. A release artifact contains `index.html`, the optimized
  `main.wasm`, the matching copied `wasm_exec.js`, and any future runtime assets under that directory;
  it contains no source tree or development-only files. Both compressed streams exist only to measure
  the size gate: Pages receives the optimized `main.wasm`, not a precompressed file whose required
  `Content-Encoding` metadata it cannot infer. Every URL in `index.html` is relative so the build
  works at the repository-project path returned by Pages rather than assuming the site is hosted at
  `/`.
- `web/index.html` hosts a full-window canvas, uses the full game title, and shows a loader until
  the first successful game draw. `pkg/app` invokes one injected outer presentation callback once
  after that draw; `main_js.go` implements it by hiding the loader and setting
  `document.documentElement.dataset.africa2iceReady = "true"`. The callback exposes no frame or
  simulation data, takes no action, and is the stable readiness marker used by the step-2a and
  release browser tests. A separate optional browser-test observer is installed only when the page
  URL contains `?e2e=1` **and** Playwright has injected its callable observer before WASM
  instantiation; either condition alone does nothing. After each published frame it writes the
  bounded `E2ESummary` defined in step 11 to
  `document.documentElement.dataset.africa2iceSummary`; ordinary sessions install no observer. The
  observer receives the already-published immutable frame, never requests another snapshot, and
  cannot emit an action, touch storage, consume RNG, or change a revision. While gameplay is active,
  the page's `keydown` listener calls
  `preventDefault()` for Ctrl+S/Cmd+S so the browser does not open Save Page; the event still
  reaches the game scene, which handles the same shortcut on both targets. A clickable HUD
  quick-save control is the keyboard-independent fallback.
- **Web operational log.** `main_js.go` constructs §3's session before the storage, renderer, audio,
  or UI graph. Its JSONL sink writes to `os.Stdout`; the copied Go `wasm_exec.js` forwards each
  newline-terminated record to `console.log`. No log record enters IndexedDB, the save wire format,
  browser storage, or a network request. The deferred panic guard logs a bounded stack and re-raises
  the panic so the browser retains its ordinary failure semantics.
- **Browser DPI and resize.** `main_js.go` calls `ebiten.RunGameWithOptions` with
  `DisableHiDPI = false`. CSS owns only the canvas's full-window DIP size; JavaScript never assigns
  `canvas.width`/`canvas.height`, multiplies by `window.devicePixelRatio`, or scales input a second
  time. Ebitengine supplies the outside DIP dimensions and monitor scale to §8's `LayoutF` contract.
  Browser zoom, DPR changes, orientation changes, and ordinary resize therefore publish one new
  `render.Viewport`, rebuild only screen-sized presentation targets, and preserve logical HUD layout
  and hit areas.
- **Audio autoplay.** Browsers refuse to start an audio context before a user gesture, so
  `pkg/app` starts the asynchronous `UISettings` read at boot, but constructs and resumes
  `SoundManager` only on the first click. If the read has completed, the manager receives those
  settings before accepting a sound request. If it is still pending, the manager starts at effective
  zero gain and retains only the first requested sound in one optional `PendingFirstSound` slot;
  later requests before settlement are dropped, so a stalled read cannot grow a queue. A successful
  completion applies the stored settings and emits that retained sound only when `Muted == false`; a
  failed read applies the §8 defaults and then emits it. Thus no sound can precede settings
  settlement, and a persisted mute always governs the first emitted sound without blocking the
  browser gesture. Construction failure yields a silent no-op manager, never a crash.
- `-dumpmap` / `-headless` / `-screenshot` / `-turns` / `-terrain-detail` plus verification-only
  `-seed` / `-policy` / `-checkpoint-json` are registered only in
  `main.go` (`!js`), so the wasm binary does not carry `flag` plumbing it cannot use.
- **Performance and transfer size are the two web risks, and §12's step 2a measures both before the
  domain exists.** Mitigations and release floors are in §8. If 96×64 proves too slow in a
  browser, `MaxRenderScale` bounds high-DPI pixel work and the low-detail mesh is the first geometry
  fallback — though low detail removes side walls and lighting, not Tetra3D's per-frame CPU vertex
  transform, so it is a narrower lever than its name suggests. The grid dimension and maximum 32×32
  render-chunk dimension are separate constants, and
  the geography rasterizes at any resolution; changing simulation grid size is a later balance
  decision, not an automatic runtime downgrade. Both risks are properties of the pinned dependency
  set and the release geometry rather than of game code, which is why step 2a measures them on a
  skeleton and step 11 only enforces what step 2a already reported.
- **Release CI is cross-platform and ordered.** `.github/workflows/ci.yml` runs for pull requests and
  pushes to `main`. Its native matrix runs `golangci-lint config verify` and `golangci-lint run`
  before `go vet`, `go test`, and the native desktop build on named runner images `ubuntu-24.04`
  (`amd64`), `macos-15` (`arm64`), and `windows-2025` (`amd64`), asserting `go env GOARCH` before
  tests and using platform-appropriate shell syntax; the Ubuntu job installs Ebitengine's
  required X11/OpenGL development headers. A separate `ubuntu-24.04` `web-release` job installs the pinned
  Binaryen release and `brotli` CLI, runs `build_web.sh --release`, applies the compressed-size gate, validates
  the complete `web/` artifact, runs `tools/run_wasm_go_tests.mjs` and
  `tools/run_wasm_checkpoint.mjs` — the latter uploading `reference-checkpoints.json` for the
  `cross-target-determinism` job — and then runs the pinned
  Playwright/Chromium application smoke against a local server.
  The browser lane must observe the canvas-ready signal, complete one new-game/migration/end-turn/
  quick-save/reload path, and fail on a page error, panic, unexpected `console.error`, timeout, or
  restored-`E2ESummary` mismatch. Playwright, Node, and Chromium are CI/test tools only and are absent
  from `web/`.
  Display-required screenshot comparisons remain outside this matrix unless a dedicated Xvfb lane
  is added.
- **GitHub Pages is the web release destination.** The repository's Pages source is configured once
  as **GitHub Actions**. Step 13 adds the Pages publication wiring only after step 12 and the
  prepublication release-readiness checks are green. On a push to `main` with that wiring present,
  `web-release` runs `configure-pages` and
  `upload-pages-artifact` for the validated contents of `web/`. A `deploy-pages` job has
  `needs: [native, web-release, cross-target-determinism, release-readiness]`, so a failure on any native platform, release
  optimization, browser smoke, artifact/size check, reference run, or automated performance
  regression prevents publication. The deploy job alone receives `pages: write` and
  `id-token: write`, targets the protected `github-pages` environment, exposes the deployment action's
  `page_url`, and uses a `pages` concurrency group with `cancel-in-progress: false`. Pull-request runs
  exercise the release build but skip Pages configuration, artifact upload, environment access, and
  deployment; v1 has no preview deployment or custom-domain requirement.
- Every external GitHub Action is pinned to a reviewed full commit SHA, with its upstream major
  release recorded in a comment so automated dependency updates remain reviewable. No personal
  access token or long-lived deployment credential enters the workflow.

---

## 11. Audio

`synth.go` generates short enveloped PCM waveforms at three distinct pitches, played through
`audio.Context.NewPlayerF32FromBytes`. Constants keep the established event names —
`SFXChoiceClick`, `SFXSaveComplete`, `SFXEventTrigger` — so real `.wav` assets can replace the synth
without an API change. After a successful `game.EndTurn()` use case, `pkg/app` compares event
`(turn, ordinal)` keys and requests at most one `SFXEventTrigger` when that completed turn appended
one or more acute events.
Planning-only snapshot refreshes and loading a frame mark its existing feed as already seen and never
replay old event sounds. Domain and application tests never touch audio.

`SFXChoiceClick` is requested once for an accepted enabled discrete button, list-row, checkbox, or
toggle activation. It is not requested for a disabled/rejected activation, slider drag/update,
keyboard repeat, hover, focus, camera input, or the mute/volume controls. The UI deduplicates by the
accepted input sequence and control ID, so pointer-up plus key activation cannot double-play one
choice. `SFXSaveComplete` is requested once per successful manual or quick-save `StorageOpID` when
its completion is polled; autosaves, list/load/delete operations, request acceptance, failures, and
duplicate completion delivery are silent. These keys are presentation-only and never enter a save
or simulation hash.

`SoundManager` applies one master gain to every player it creates, from the `UISettings`
`MasterVolume` and `Muted` preferences defined in §8. `pkg/app` begins that read at boot and uses
§10's bounded pending-first-sound protocol on both targets if the first gesture arrives before
completion; web requires gesture-gated construction, while desktop shares the ordering contract. No
sound is ever emitted at an unconfigured level or against a persisted mute.
`Muted` is a distinct flag rather than `MasterVolume = 0`, so unmuting restores the previous level
without a second stored field and a muted session still remembers where the slider sat. Changes take
effect on currently playing and subsequently created players within the same frame; neither control
stops playback, alters the event feed's seen-key set, nor requests a sound of its own. Both are
presentation-only in the §8 sense: they consume no `WorldRNG` draw, never reach `SaveState`, slot
metadata, or campaign hashes, and cannot change a simulation outcome.

Future ambient audio may use this same master gain and mute flag without changing the v1
`UISettings` shape. A later requirement for an independently adjustable ambient channel would be a
new preference and would still require the ordinary settings-version migration; this design does not
reserve an unnamed per-channel field.

The `-dumpmap`, `-headless`, and `-screenshot` verification modes need no audio and open no audio
context. `wire.go` selects the no-op `SoundManager` unconditionally in those modes rather than
leaving silence to the incidental absence of user gestures, so the §13 gate requires no sound device
on any runner. This also keeps the silent-fallback path an actual error handler: if CI reached it on
every run, a genuine audio-construction regression would be indistinguishable from normal output.

---

## 12. Build sequence

Test-first at each numbered step and required lettered substep. A substep ends with the complete
headless CI gate green and a reviewable integrated result; steps 4 and 5 are milestones, not licenses
to accumulate one large change.

**Where a test is specified.** Three sections of this document could plausibly hold any given test,
and for a while all three held most of them: §7 stated a rule and its fixtures, §12 restated those
fixtures as build work, and §13 restated them again as verification. The same list written three
times drifts three ways, so each section now answers exactly one question and cites the others
rather than repeating them:

| Section | Question it answers                 | What it therefore contains                                                                                                       |
| ------- | ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| §7      | *What must be true?*                | The rule, and the fixtures that pin the numbers the rule states — the worked examples, bounds, and rejection cases that **are** the contract expressed as assertions |
| §12     | *When do we build it, and on what?* | Step and substep placement, prerequisites, which step owns an artifact, and assertions that exist only at integration or campaign scale |
| §13     | *How do we know it shipped?*        | The CI gate commands, checkpoint output, release margins, and the display- or browser-required checks a headless test cannot make   |

A step therefore names the contracts it implements and adds only what its own scale contributes; it
does not re-enumerate the fixtures those contracts already specify. When a step's assertion is
genuinely new — because it needs several subsystems at once, a full campaign, or a real device — it
belongs here and nowhere else.

**Structure before values.** Several rules in §7 say their tables “must be closed before
implementation,” while later viability and balance passes may tune them. Without a lifecycle rule,
those statements could be read as either forbidding calibration or permitting consumers to start
with unspecified release data. The resolution applies uniformly and is stated once here rather than
per-subsection:

- A table's **structure** — its dimensions, key set, entry shape, ordering, completeness
  requirement, and validation predicate — is closed _before_ the step that implements its consumer,
  and is exercised there against explicit fixture values. This is what “closed before implementation”
  means everywhere in §7. Structure includes constraints between entries, such as the
  accessible-megafauna rate exceeding ordinary terrestrial hunting's, or a fauna profile supporting no role for
  which it has zero prey weight.
- A table's **values** follow Appendix C's lifecycle. **Locked** and **Initial** values are implemented
  with their consumer; **Initial** domain values may first be tuned at the step-5e viability gate and
  may be tuned again in step 12, while **Locked** values may only be validated. Only **Open** rows use
  explicit non-release fixtures until step 12 closes them.

Thus step 5 implements the phase-3 pipeline with the selected C.7 foraging, hunting, megafauna, and
fauna-profile values plus the selected C.8–C.12 values. Appendix C currently has no **Open** rows;
step 5e tests the complete domain model early, while step 12 performs final calibration of every
**Initial** row. Neither pass invents a release value or replaces a placeholder fixture. A subsystem
is implementable once its structure and every input consumed by that
step are present. In step 4 this includes `TemperatureAlgorithm: "lat-elev-offset-v1"` and its exact
latitude table, because biome classification and the cooling-pushes-tundra-south test use absolute
°C; the exact orbital and seasonal tables; the already fixed campaign-clock values; the selected C.7
ecology/resource tables; the Toba presentation marker; and the required active Campanian episode's
exact source-backed `EventYearBP`, on which its activation-boundary assertion depends; and the eight
starting tile IDs derived from the locked anchor catalog once turn-0 habitat is available. Appendix C
makes this timing explicit: the closed
six-entry `FaunaGroup` catalog is mapped into the public API in step 2; the selected initial-stock
configuration and resource tables land in step 4; the selected collection tables, hunting-work
activation/mapping/coefficients, and water-equivalent scoring land with their consumers in step 5. The normalized
stock-unit and conversion values are already selected; step 5 implements and verifies them.

1. **Scaffolding first, so the boundaries exist before any code can violate them** — `go.mod` with
   Ebitengine + Tetra3D, `.golangci.yml`, `internal/archtest`, minimal compiling `doc.go` files for
   `pkg/gameapi`, `internal/domain`, `internal/application`, `internal/adapters/storage`,
   `internal/adapters/logging`,
   `pkg/render`, `pkg/ui`, `pkg/audio`, and `pkg/app`, plus build-tagged stub entrypoints and
   `.github/workflows/ci.yml`. Confirm `golangci-lint config verify`,
   `golangci-lint run`, and both desktop and `GOOS=js GOARCH=wasm` builds pass on the skeleton.
   Architecture rejection fixtures cover `log`/`log/slog` imports from any package other than the
   logging adapter, a logging-adapter import of domain, and any unlisted logging-adapter dependency.
   Include all three of §5's arithmetic rules in `arch_test.go` from the start. Rejection fixtures for the
   floating-product rule cover
   both product forms: the expression cases `a*b + c`, `a + b*c`, `a*b + c*d`, `x += a*b`,
   `p := a*b; r := p+c`, `*p = a*b; r = *p+c`, `float64(a*b + c)`, and the raw chain end
   `float64(a*b) * c`; the compound-assignment cases `x *= y` and `x *= y; r := x + c`, including
   `float32`, named-floating, indexed, and map-indexed left sides that contain no multiplication
   expression and are missed entirely by an expression-only walk; a type-parameter multiplication
   whose type set admits a floating type; plus aliased `math.FMA` and a production `math.Sin` call.
   Acceptance fixtures cover `float64(a*b) + c`, `p := float64(a*b); r := p+c`, type-appropriate
   explicit rewrites for `float32`, `float64`, and named floating types, the chained
   `float64(a*b*c) + d` whose inner product needs no conversion, an integer product and an integer `x *= y`,
   every approved `math` call, and test-only `math.Sin`/`math.Cos` table checks.
   Regression fixtures pin the multiplication exemption's boundary: `float64(a*b) * c` is rejected, while
   `float64(a*b*c)` and `float64(float64(a*b) * c)` are accepted. The first accepted form proves that
   no conversion is required per factor; the second proves that a redundant inner conversion remains
   legal.

   The constant exemption needs its own fixtures, because it is the one case where two source forms
   with identical meaning must both be accepted. Cover a bare `const X = 0.5 * 2.0`, a typed
   `const X float64 = 0.5 * 2.0`, a typed `float32` constant product, a constant product inside a
   composite-literal configuration table, and a constant `*` whose parent is an addition — all
   accepted with no conversion, all reported as constant-folded by `types.Info.Types[expr].Value`. A
   paired rejection fixture keeps the exemption honest: the same expression with one operand replaced
   by a runtime variable must fail, proving the pass tests constant-ness rather than literal syntax.

   The float-to-integer rule gets rejection fixtures for `int(x)`, `int64(x)`, `uint32(x)`, a named
   integer type, a named floating source type, and a conversion inside a `_test.go` file, plus
   acceptance fixtures for `float64(i)` in the other direction, integer-to-integer conversions, and
   the constant conversion `int(2.0)` that the type-checker folds. The one production acceptance
   fixture is `RoundPopulation`'s finite/range-checked `Population(value + rng.Float64())` conversion,
   identified by its shape — a single floating parameter, all four checks present — rather than by
   its name, so a copy under another name is still rejected; the same
   conversion anywhere else, or a helper version missing any check, is rejected. Adding all three gates now
   costs one focused test; adding them after the domain exists means auditing every expression already
   written.
2. **`pkg/gameapi`** — implement Appendix C's closed six-entry `FaunaGroup` catalog and
   `FaunaGroupCount`, then
   define enums including `CampaignEra`, `Frame`/`Tile`/`Band`/`FoodTurnReport`/`OutcomeReport`/`Event`, fixed per-band
   technology/progress and six-value heritable state, `LastFoodReport`, `LastOutcomeReport`, climate/macro summaries, co-located interbreeding
   candidates, and five-role basis-point allocation values, per-tile
   `ElevationKm`, `Biome`, `Explored`, `NaturalShelter`, `LocalTemperatureC`, `MovementCost`, visible macro impact, and fixed-size `FaunaSummary`
   values with the closed `FaunaGroup` enum, derived
   original-research-preview and band/destination mortality-rate preview values, fixed nine-entry `ResearchOptions`, `Interbreed`, the other `Command` values, segregated
   `CampaignUseCases`/`StorageUseCases` and composed `Game` port, and `SlotMetadata`. Depends on
   nothing outside the standard library; every driving adapter is written against it.

   - **2a — Cross-platform walking skeleton.** Replace the step-1 no-op entrypoints with the smallest
     real Ebitengine host on desktop and js/wasm, backed by a fake `gameapi.Game` that returns one
     immutable minimal frame. Add the full-window `web/index.html`, `build_web.sh --dev`, and a
     deliberately minimal Tetra3D scene/camera that clears and draws one visible primitive through
     the eventual `pkg/render` seam; none of these may import the domain. The desktop smoke opens a
     window and exits cleanly. The browser smoke serves `web/`, proves the matching copied
     `wasm_exec.js` instantiates the module, waits for a stable canvas-ready marker, observes no page
     error or unexpected console error, and exits. Introduce the locked browser-test dependency and
     lockfile here; it is tooling only. Keep this slice and evolve it in steps 8, 10, and 11 rather
     than replacing it with a second host. Its purpose is to fail early on graphics, loader, build-tag,
     input-loop, or browser-policy incompatibility before the domain and persistence layers exist.

     **Both web risks are measured here, because both become measurable the moment `go.mod` is
     fixed.** §10 names web performance as the main web risk and this substep exists to fail early on
     it, so it must not defer the two numbers that decide whether the target is viable until step 11,
     by which point the domain, application, render, and UI layers have already been written against
     whatever those numbers turn out to be. Neither measurement needs `internal/domain`, and neither
     gates this substep — a walking skeleton that fails a release threshold has told you something
     useful, not something disqualifying.

     - **Transfer size.** Add `--release` to `build_web.sh` far enough to run the pinned
       `wasm-opt -O3` and record raw, `brotli -q 11`, and `gzip -9` byte counts for the skeleton, and
       for one variant per major dependency (Ebitengine alone; plus `text/v2`; plus Tetra3D). The
       per-dependency attribution is the point: a single total says the budget is missed, while the
       variants say which dependency to argue with. Write the results into `docs/PERFORMANCE.md`
       under a skeleton heading. Step 11 owns the enforced gate; this substep owns knowing the floor.
     - **Frame rate.** Build a synthetic terrain fixture with the release geometry — six 32×32
       chunks, 12,288 top-face triangles, one shared material, one directional light, and Tetra3D's
       default depth pass — driven by the fake frame with no simulation behind it. Run §8's orbit/pan
       script against it at DPR 1 and DPR 2 in both detail modes on the §8 reference machine and
       record median FPS. This is the release render workload minus the game: geometry, chunk count,
       material, and lighting are what §8's floors actually measure, and Tetra3D transforms every
       vertex on the CPU every frame whether or not a snapshot changed. If the skeleton cannot reach
       §8's floors, no later optimization of simulation or projection will rescue them, and the
       chunking, detail-mode, and `MaxRenderScale` decisions must be revisited now rather than after
       step 8 has built on them.

3. **Domain geography** — `internal/domain` value types plus `geo.go`, `geodata.go`, `region.go`,
   `grid.go`, and the concrete `worldgen.go` service. Implement the exact §6 coordinate catalogs and
   integer raster/region/coastline/passage-anchor rules; this step freezes independent land, water,
   region, highland, river, and coastal-mask checksums and may not select alternate geometry. Tests: landmarks
   resolve to land, mid-Atlantic is ocean, Bab-el-Mandeb is a narrow strait, the Campi Flegrei
   epicenter resolves to Frangistan land, Frangistan sits north of
   the Mediterranean, South Asia has destination tiles, the central Yellow River coordinate resolves
   to its non-overlapping basin destination, Sahul and western Alaska are in bounds, Wallacea and the
   Bering Strait remain explicit water gaps, region masks are stable, and generation is deterministic.
   Implement §6's authored elevation model and `BaseMoisture` here, since `worldgen.go` owns both and
   step 4's temperature and vegetation index read them. Elevation fixtures assert the exact ten-entry
   stable catalog, `HighlandElevationKm = 1.0 km`, zero on water/uncovered land, maximum on overlap,
   strict threshold boundaries, all ten features contributing land, feature-order and seed
   invariance, finite `[0, 3]` bounds, and the checked 6,144-value checksum. For moisture, implement
   the 64-entry `ZonalMoisture` table under the same bit-pattern, tolerance-assertion, and checksum
   discipline as `LatitudeSinSquared`, plus river, continentality, and orographic terms. Assert a
   finite `BaseMoisture` in `[0, 1]` for all 6,144 tiles, a profile symmetric in absolute latitude,
   both zonal minima falling within the Saharan/Arabian and Kalahari latitude bands and the maximum
   within the equatorial band, that river-corridor tiles suppress continentality along their length, and that
   the Nile corridor is materially moister than the desert two tiles away. Assert the seed does not
   perturb any of it.
   Natural-shelter fixtures assert the exact nine unique ellipse entries, permitted radii/ratings,
   finite `[0, 1]` derived values, at least one land tile per feature and tier, zero on unmarked
   land/water, maximum rather than sum on overlaps, identical results after feature reordering, the
   checked 6,144-rating hash, and at most one rating per tile. Ratings remain unchanged across seeds,
   biome/climate changes, degradation, occupancy, and reload under the same geography and shelter-mask
   versions, with zero RNG use.
   Grid fixtures assert center/edge/corner neighbor bounds, stable eight-direction order, symmetric
   cardinal/diagonal edges, `1`/`math.Sqrt2` step lengths, and removal of a diagonal when either
   orthogonal corner is water but not merely uninhabitable. Movement-cost fixtures wait for step 4,
   where `V`, `ClassifyBiome`, the curve, and the biome factors all exist.
   Add §6's exact ten-entry starting-anchor catalog and validate stable order, finite in-bounds
   coordinates, the `4/1/2/3` species/region partition, and distinct fractional projected positions.
   Exact tile resolution waits for step 4's turn-0 habitat classifier.
   Exploration fixtures assert the exact initial East Africa land/coastline mask, no other initial
   region land, deterministic 96-word bit ordering, and no reveal from Levant/Frangistan archaic
   starts. A 3×3 center/edge/corner footprint includes water and ignores movement-corner legality;
   named far endpoints appear only when the resident sapiens band can currently traverse them.
4. **Domain campaign clock, climate, biomes + tiles** — deliver this milestone as three separately
   green substeps; each exposes only pure domain APIs and preserves all earlier fixtures:

   - **4a — Clock and physical climate:** campaign-date conversion, season/orbital tables, absolute
     temperature, long-term temperature/moisture, abrupt pulses, and Beringian eligibility.
   - **4b — Habitat and renewable ecology:** vegetation index, biome classification, movement and
     capacity curves, degradation, stock caps/regeneration, and fauna profiles.
   - **4c — Authored scenario derivation:** starting-anchor resolution, macro-event masks and
     activation, initial stocks, and the exhaustive 401-turn/map sweeps that compose 4a and 4b.

   The combined acceptance contract implements §§6–7's clock, climate, habitat, and macro-event
   contracts with their fixtures as specified there: `four-era-v1`'s exact
   80,000/50,000/35,000/25,000/20,000 BP endpoints and 300/150/100/50-year spans;
   `dispersal-map-v2`'s authored elevation catalog and strict highland threshold;
   `lat-elev-offset-v1` with its 64-row table and checksum; the orbital, seasonal, and precession
   tables under that same bit-pattern, tolerance, and checksum discipline; the abrupt-pulse catalog
   and the shared regional climate-response table; `BeringiaOpenFraction` and its attainability
   check; the vegetation index and `ClassifyBiome`; `BaselineKCurve` and `MovementCurve` with their
   `V = 0.60` reference values and the per-biome deviation table; `MaxMovementCost` and its Wallacea
   coupling; reversible degradation; Appendix B's cap indices and regeneration fractions;
   `region-biome-v1`; `bounded-regional-v1`'s Campanian catalog, polygons, and compiled masks
   alongside the no-effect Toba marker; and the two-scale initial-stock rule.

   What this step adds is scale and authoring. Seed-independent `HabitatTemperatureC`,
   `EffectiveMoisture`, `V`, biome, and every composed movement cost are swept exhaustively for all
   6,144 tiles at all 401 turns; `BiomeChurnCap` and `MinBiomeDwellTurns` hold across that exact
   history. `LocalTemperatureC` additionally tests both closed `UnitNoiseV1` extremes and the exact
   `BalanceSeedCorpus`. That exhaustiveness makes the configuration-time gates real rather than
   aspirational without pretending to enumerate every uint64 seed.

   Run the starting-anchor authoring generator here, now that turn-0 `BaselineK` exists: for each of
   §6's eight anchors, filter to unused land tiles in the required region with finite positive,
   seed-independent turn-0 habitat `BaselineK`, rank by squared projected-grid distance
   and then tile ID, and freeze the winner into the checked-in scenario fixture. Assert the exact
   eight IDs, the `4/2/2` species and region partition, no duplicate, zero RNG use, and unchanged
   output across the seed corpus. A missing robust candidate is a build failure, never permission to
   fall back to an uninhabitable or wrong-region tile.

   Three calibration assertions also belong here, because each needs the whole map over the whole
   campaign. Turn 0 must classify East Africa as habitable savanna and woodland while turn 400 pushes
   the tundra boundary strictly south of its turn-0 latitude. The Saharan and Arabian corridor must
   show net desert expansion at turn 400 relative to turn 0, with at least one interval where the
   habitable-tile count rises above the preceding local minimum — proving the precession term is
   visible rather than swamped by the trend. And a threshold table must be rejected when its own
   hysteresis violates the strict headroom policy: at `upper = 0.98` with a `0.02` band the trip
   point equals the clamped maximum and is technically reachable only at exactly `x = 1`; reject it
   because `upper + h < 1` deliberately requires positive entry headroom. Report the
   tile-turn-weighted biome mix at turns 0, 100, 200, 300, and 400 for Appendix B.4's moisture gate,
   and assert that no `internal/domain` rule reads `ClimateEpoch`.

5. **Domain aggregate, WorldRNG, species, bands + turn loop** — deliver this milestone through five
   separately green, integrated substeps:

   - **5a — Aggregate shell and planning invariants:** scenario construction, IDs and caps, owned RNG,
     assignments, spatial-action ownership, command rejection, splitting, and memento round trips.
   - **5b — Subsistence and demographics:** shared flora/fauna/water allocation, food conversion,
     spoilage/storage, shelter, health, growth, chronic mortality, research production, and their
     origin-time captures.
   - **5c — Spatial world behavior:** movement ranking and queues, named passages, exploration,
     establishment, archaic planning, and final-tile revalidation.
   - **5d — Adaptation, incidents, and atomic integration:** technology prerequisites/diffusion,
     genetics/selection/gene flow/interbreeding, macro and acute effects, the complete five-phase
     transition, cleanup/publication, and full deterministic continuation tests.
   - **5e — Mandatory domain viability gate:** complete reference/corpus campaigns and all five
     route policies before application and persistence work begins.

   In this step, “save/reload” means a pure `ExportState`/`RestoreWorld` memento round trip; JSON,
   `gameapi` frames, application revision, and storage-adapter assertions belong to steps 6 and 7.

   Implement §7's scenario, workforce, genetics, exploration, and archaic-policy contracts with
   their fixtures as specified there: the complete `NewWorld` field-by-field initializer, including
   IDs `1`–`8`, `NextBandID = 9`, zero reserves/reports/mortality, exact sapiens/preset assignments,
   `SplitMix64` seed expansion with §7's pinned corpus word pairs, the project-owned `Float64`
   mapping with its `[0, 1)` and endpoint fixtures, and resources; the four/one/two/one starting bands at `Health = 1.0` with an
   empty technology state, `proportional-basis-points-v1`, the six-value heritable vector with its
   standing-variation profiles and per-trait effect and selection functions, the derived
   `UVExposure` and hypoxia inputs, `rare-emergence-v1`, `sapiens-frontier-v1`, and
   `ranked-pressure-v1` with its presets and technology priority.

   Genetics assertions that exist only at aggregate or campaign scale stay here: exact heritable
   vectors survive movement and demographic change, copy exactly to both split descendants, and
   round-trip through `ExportState`/`RestoreWorld`; every habitable tile at all 401 supported turns
   produces finite `UVExposure` in `[0, 1]`, with identical results before and after a memento
   round-trip; and applying each trait through a complete turn reaches only its authored gameplay
   channel. These checks complement rather than restate §7's function and mutation fixtures.

   What this step adds is everything that appears only once a live aggregate exists. The owned PCG
   round-trips and continues its sequence rather than restarting it. Player authority is limited to
   sapiens at the aggregate boundary, not merely omitted from the UI. Split, migration, and
   `Interbreed` share one persisted spatial action: both split descendants are spent, assignment and
   research remain available, the archaic target may still take its own computer action, and a
   memento round trip after either cannot regain an action. Several sapiens actors targeting one
   archaic band must produce a population-weighted result invariant to command and contact order.
   The archaic controller is exercised against live state: a new world stays below
   `MaxArchaicBands`; a restored world already above a lowered sub-cap remains valid and takes no
   further policy split until extinction cleanup drops the count below the cap; and shuffled backing
   storage must yield the identical command batch. An all-`65_535` assignment vector must fail
   validation, proving the `uint32` accumulator cannot wrap into an apparently valid total. Role
   routing is proved end to end: foraging reaches only flora, Hunting & fishing and eligible
   megafauna tracking reach only the one shared fauna stock, water demand stays population-wide and
   role-independent, shelter reaches only mitigation, and toolcraft only original research. Every
   migration candidate exposed to sapiens must target an already explored tile, and no reveal,
   selection, gene-flow, or policy step may consume a draw.

   Implement §7's subsistence contracts here with their fixtures as specified there:
   `linear-shared-flora-v1`, `linear-shared-fauna-v1`, `linear-megafauna-v1`, and
   `linear-share-work-risk-v1`, over `region-biome-v1`'s profile lookup. Their worked allocator
   examples — 200:100 demand against 60 flora, one band's 100 hunting plus 100 megafauna against
   another's 100 with 60 fauna, the `3`-of-`5` aquatic split, rates 1 and 2 with ten workers in
   each role, and the `0.018`/`0.006` work-risk weights — are the fixture list, together with each
   contract's rejection cases and Appendix C locks.

   What this step adds beyond those contracts: the allocator fixtures may supply demands directly,
   but separate integration fixtures must derive megafauna demand from the production rule rather
   than hand-feeding it, so a production regression cannot hide behind a correct allocator.
   Overflow behavior is exercised at real scale — individual products or their totals exceeding
   `float64` range must still yield finite bounded allocations with conserved stock. Backing
   storage and role-evaluation order are shuffled before every allocation assertion, so stable
   iteration cannot conceal a first-band or first-role advantage. Origin capture is proved across a
   whole turn: work-risk weights taken before demographics survive growth, loss, and migration and
   reach the final tile's uncovered components exactly once, while a move or newly completed
   technology grants no second catch. Reloading the same planning state must reproduce the next
   turn's rates, demands, allocations, and work weights with no extra draw.
   Implement §7's `saturating-share-terrain-v1` shelter contract, its four-class coverage table,
   the `RemainingRisk(tech, camp) = (1 - tech) * (1 - camp)` composition, the technology-mitigation
   table, and the genetics effect-coverage factors, with their fixtures as specified there. That
   includes the share-conversion endpoints, the half-saturation point, the diminishing-returns and
   below-immunity bounds, the `0.50`/`0.50` composition leaving `0.25` residual, the
   `0.50` × `0.30` technology product leaving `0.35`, the seasonal and chronic risk-profile locks,
   and every rejection case those contracts enumerate.

   What this step adds: the same captured share must be proved to reach phase-3 origin components
   and phase-5 final-tile components exactly once, with `NaturalShelter` read only for `Exposure`
   and only at the tile where each phase resolves — open-to-sheltered, sheltered-to-open, and
   cancelled-move fixtures make that observable in a way a single-phase unit test cannot. Effort is
   discarded after acute resolution, so a band whose next turn assigns zero shelter receives no
   residual benefit while its independent technology mitigation survives. Split descendants derive
   effort from the copied share against their own terrain rather than inheriting a cached fraction.
   Component values are checked before the shared caps as well as after, so a rebalanced
   distribution under a binding cap is not mistaken for a mitigation regression.
   Implement §7's hazard, movement, and passage contracts here with their fixtures as specified
   there: the acute probability partition and its shared cap, the acute-severity bounds, the
   one-or-two-draw accounting, `destination-vegetation-v1` ordinary edge costs,
   `named-asymmetric-v1` with its `4.00`/`4.50`/`3.00` route costs and eligibility rules, the
   migration attraction score with its water survival-equivalent term, and `ranked-pressure-v1`.

   What this step adds is the five-phase transition ordering itself: macro impact resolved in
   stable episode-then-band order before acute selection in stable band-then-kind order, each
   mortality term applied exactly once, post-macro/acute stress computed on the final tile, a
   climate-driven destination-cost change that reranks the next planning candidates without
   cancelling an already valid queue, cancelled versus successful passage risk, and a new
   destination that an acute event pushes below `MinEstablishedBand` failing to latch while an
   older achievement survives.
   Implement §7's food and health contracts here, with their fixtures as specified there:
   `population-food-turns-v1` storage capacity, end-of-turn overflow discard, and phase-1 spoilage;
   `normalized-source-v1` conversion; automatic consumption and `FoodDeficitFraction`;
   `LastFoodReport` and `LastOutcomeReport`; the health representation with its nutritional decline and recovery, water,
   endemic-disease, and genetic-burden contributions and their single phase-3 clamp; the phase-5
   outbreak health checkpoint; `HealthVulnerability`; the quadratic starvation response; and
   food-limited positive growth. Each of those subsections' worked examples, bounds, rejection
   cases, and Appendix C locks **are** its fixture list; this step adds only what integration
   contributes and does not restate them.

   Two constraints belong to this step rather than to those contracts. Read every Appendix C
   constant through the production configuration values, never by parsing this Markdown manifest.
   And stop at domain state and `ExportState`/`RestoreWorld` round trips: steps 6 and 7 add the
   corresponding frame and `SaveState` projections, so no assertion here may reach for them.

   The integration-scale assertions that exist only here: one advancing turn applies spoilage,
   consumption, health, starvation, growth, mortality, and the overflow cap exactly once each and
   in that order; §7's ordering example holds its `300 = 30 + 100 + 20 + 150` FU accounting across
   a turn with real demographic losses rather than fixture inputs; a health-driven change in
   chronic mortality under a binding shared cap leaves `raw_starvation` unchanged while applied
   deaths differ; a band that dies mid-turn leaves no report, no transient gain, and no reserve; a
   dedicated synthetic acute fixture uses the valid boundary `MinAcuteLoss = MaxAcuteLoss = 1` with
   no severity mitigation: selecting that incident against a positive phase-5 survivor must produce
   exact acute extinction, exercise the same cleanup, and free its band slot;
   and identical planning state reproduces all of it after a memento round trip with no extra draw.
   Aggregate boundaries belong to this step because no single contract owns them. A valid split
   above 90% conserves population and stored food. Starting at 255 bands, one valid split reaches
   256 and the next returns typed `BandLimitReached` without changing population, stored food, IDs,
   queues, or RNG state; phase-5 removal of an extinct band frees that place so a later valid split
   succeeds with the next monotonic ID; and a separate exhausted-ID fixture rejects without mutation
   even below the cap. Low-stress sapiens bands still expose and accept a whole-band migration while
   rejecting `SplitBand`, so crossing `SplitStressThreshold` changes the alert and the split
   permission without changing candidate availability or command authority. Every external command
   with an archaic acting `BandID` returns typed `ComputerControlledBand` without mutation, the
   archaic `TargetBandID` of a valid sapiens `Interbreed` being the sole target exception; the same
   commands issued by the internal authority must still pass every ordinary semantic
   check. Implement §7's technology contracts here with their fixtures as specified there: the
   nine-node DAG with its roots, prerequisite sets, research costs, and `T_tech` factors; the empty
   new-game state; `ResearchTech` validation; `saturating-toolcraft-v1`; the
   `co-located-cross-species-v1` contact geometry; and `stacked-acquired-v1`. Their locks — the
   exact `1.4772347614192902` all-nine product, the half-saturation gain, the one-source 20% and
   five-source completion examples, and the pair-count bounds — are that fixture list.

   What this step adds are the cases that need two bands and a completed turn. With a
   zero-progress recipient, let one eligible source teach a missing prerequisite while another
   already knows `MedicinalKnowledge`: only the prerequisite may gain or complete this turn, and
   the dependent becomes diffusion-eligible only on the next. A band that does not survive phase-5
   cleanup leaves no technology state, while a survivor keeps the gain captured before
   demographics. Original research and diffusion sum once against the frozen snapshot, and a
   combined threshold crossing normalizes exactly, clears the matching target, and reaches the next
   planning frame's stress. Shuffling source and contact-pair discovery order must produce
   identical progress and identical state hashes, and the whole research/contact/diffusion path
   consumes zero RNG.

   - **5e — Mandatory domain viability gate.** Before application or persistence work begins, run the
     complete turn loop through the same fixed seed corpus and deterministic sapiens route policies
     used by the final balance gate. On every corpus seed, prove the reference policy reaches one
     destination by the current turn and establishing-population margins. Across the corpus, also
     prove it retains the establishment and turn-400 survival margins, ends more subdivided than the
     scenario was founded with, and never exceeds `MaxBands`; separately prove Frangistan,
     South Asia, the Yellow River Basin, Sahul, and Beringia are reachable.

     The **subdivision margin** requires the reference policy to end at least one corpus campaign
     holding strictly more established bands than the scenario's founding sapiens count, derived from
     `StartingAnchors` so that editing the scenario cannot silently weaken it. It exists because the
     other margins are all satisfiable by a campaign that never disperses: regional achievements latch
     permanently once earned, so a campaign that touches Beringia and then contracts to its founding
     tiles still reports a victory with the record intact, and total sapiens is blind to how the
     people are distributed, so the founding bands doubling in place clear the survival margin as
     comfortably as forty spread across a continent. Both shapes were produced during calibration, the
     second of them passing every seed with zero variance — which is itself the tell, since a
     stochastic model should not land on the same integer 48 times. Dispersal in this model happens by
     splitting, so a campaign that disperses must end more subdivided than it began. The margin is
     scoped to the reference policy, matching the survival margin, because only the reference policy
     splits its non-route bands and so is the only one that can subdivide by construction. Record extinction and dispersal-failed outcomes as expected possible results,
     not harness failures. This pass may tune only Appendix C **Initial** domain values and must update
     their owning §7 rules, Appendix B, fixtures, and manifest rows together. It may not relax a
     tighten-only margin or change a **Locked** value to manufacture a win. The gate must be green
     before step 6, so a nonviable model is corrected before its wire, storage, and UI contracts make
     that correction expensive. Step 12 reruns and extends this corpus; it is not the first time the
     whole campaign is exercised.

6. **Application use cases + frame projector** — `GameService`, gameapi-to-domain command mapping,
   `WorldRevision`, `EndTurn`, and the `gameapi.Game` implementation. Retain several returned frames
   across more than three later projections, then mutate each older `Frame`, including band
   population, `Health`, `StoredFood`, `HeritableState`,
   `LastFoodReport`, `LastOutcomeReport`, interbreeding candidates, and
   tile `ElevationKm`, `Biome`, `Explored`, `NaturalShelter`, `LocalTemperatureC`, `MovementCost`,
   `FaunaSummary` weights/flags, passage,
   established-region, event-feed, and nested
   migration-candidate slices including destination mortality previews, current-tile mortality previews, `CampaignResult`, and every per-band three-entry `PassageStatuses`
   array, and prove the domain and
   every later `Frame` remain unchanged — the no-recycling isolation guarantee, asserted rather
   than assumed. A backing-address fixture must
   also prove that no published slice is reused by another published frame. Assert every band has at
   most ten candidates and exactly `TechCount` progress
   values, exactly `HeritableTraitCount = 6` bounded trait values, and exactly
   `AssignmentCount = 5` basis-point allocation entries totaling 10,000, the frame
   has at most 256 bands, assignment entries total at most `5 * len(Bands) <= 1_280`, and candidate
   plus progress, heritable, and passage-status values total at most
   `(10 + 9 + 6 + 3) * len(Bands) <= 7_168`. Each passage status must match the shared domain
   predicate, including occupancy/habitability/technology/climate reasons, and each terminal domain
   result must map exactly to the frame enum consumed by the end scene.
   Across all sapiens bands, co-located archaic candidate IDs correspond exactly to eligible
   cross-species pairs, contain no duplicate, and total at most 16,384.
   Each band has exactly one value-typed food report, at most 256 total; it must equal the domain's
   record, including its turn, without consuming RNG or recalculating food.
   Tile fauna summaries must match the domain's current regional profile, contain exactly
   `FaunaGroupCount` weights, and total at most `6_144 * FaunaGroupCount` weights per frame.
   Modifying a summary must not mutate the domain's configuration or later summaries; selection,
   draft editing, and drawing must not resample profiles or reserve fauna.
   Every frame's `Tile.ElevationKm` must exactly match the supported geography's finite `[0, 3]`
   value and remain unchanged across turn-only climate or biome changes; renderer tests must consume
   that projected value rather than a second height map.
   Every frame's `Tile.Explored` values must exactly expand the world's 96-word bitset; mutating a
   returned flag cannot alter the world. Hidden archaic bands and full tile values may remain in the
   isolated frame for bounded rendering, but UI-visible candidate destinations must all be explored.
   `TerrainRevision` starts at `1`. Biome, exploration, or visible macro-impact changes advance it;
   assignment/research-only changes do not; rejected/no-op operations advance nothing and
   publish no frame. A successful load or new campaign with changed rendered inputs advances it.
   Assert the frame exposes no second cache counter, and that band markers rebuild from the
   published frame without one. Renderer fixtures use the coarse terrain signal to compare cache-specific
   inputs, proving a biome-only change does not rebuild the veil and an exploration-only change does
   not rewrite unchanged biome colors. `OriginalResearchGainPreview` must equal the domain formula
   capped at the target's remaining cost,
   remain finite and non-negative, and be zero when original research is inactive.
   Exhaustive count-sentinel fixtures map every closed domain enum to `gameapi` and back where the
   boundary is bidirectional; mappings use switches rather than relying on equal underlying ordinals.
   A table fixture maps every exported typed domain failure to a stable `gameapi.ErrorCode`.
   Command-shape failures stop at the application boundary, while semantic failures are returned by
   `World.PlanPlayer`; neither path partially mutates the aggregate or increments `WorldRevision`.
7. **Application persistence use cases + storage adapters** — domain-memento/`SaveState` mapping,
   schema/version validation, `CampaignRepository`, asynchronous operation/result facade,
   generation-addressed commit protocol, `internal/adapters/storage/file.go`,
   `internal/adapters/storage/indexeddb.go`, and backend-neutral failure injection. This step owns
   both adapter implementations and the shared repository contract suite; js/wasm tests exercise
   IndexedDB through the controllable adapter fake, and step 11 executes the compiled js/wasm Go
   test binary in Chromium against real IndexedDB. Step 11 otherwise owns real-browser composition,
   upgrade/versionchange, Web Lock, and timing integration — it must not introduce a second
   campaign-storage implementation.

   **The wire contract is one rule applied to §9's complete lists, not a partial copy of them.** A
   fully populated state fixture must survive `State` → `SaveState` → JSON → each backend →
   `SaveState` → `State` with exact normalized-DTO equality and the golden future-equivalence result.
   Every field §9's payload enumerates therefore round-trips unchanged. Everything it marks derived
   is reconstructed identically from saved state plus versioned configuration rather than serialized
   — rates, demands, allocations, prey mixes, class effects, remaining-risk factors,
   natural-shelter ratings, endemic health contributions, and the rest.

   Algorithm coverage comes from executable schema rather than parsing this document. Step 7 owns a
   test-only `algorithmCases` table keyed by the exact exported `SaveState` field name for every
   string field whose name ends in `Algorithm`. Each case supplies the current identifier, zero or
   more explicitly supported older identifiers, one unsupported identifier, and the
   algorithm-specific reconstruction/future-equivalence assertion. A reflection sentinel compares
   the exact field-name set with the table and rejects a missing, extra, or duplicate case. The shared
   suite proves that every current identifier round-trips, every unsupported identifier is rejected
   without replacing the running world, and every **declared** older identifier follows its explicit
   migration and reconstruction case. All older-identifier lists are empty while v1 remains
   unreleased; the suite must not invent a migration from a provisional identifier. A separate
   `SchemaVersion` case rejects a schema newer than the executable without conflating ordered schema
   versions with opaque supported-or-unsupported algorithm strings.

   What follows is the work no algorithm contract owns.

   *Isolation and save-shape validation.* Mutating any slice or RNG byte sequence in an exported
   `domain.State`, a mapped `SaveState`, or a repository completion must not reach the live world or
   a later export, and `RestoreWorld` must reject invalid aggregate state before it becomes live.
   Validation covers zero, 256, and oversized band collections; unique positive band IDs; valid and
   exhausted `NextBandID`; prerequisite-closed technology bitsets with their targets and progress;
   six-value heritable vectors; allocation vectors rejected both under and over total; spatial-action
   markers with their intent references and recorded co-location; split-used markers without queues;
   and completed-turn states with every marker cleared. A new-game save must round-trip its empty
   technology state on both backends without inventing a target, progress, acquisition, or a
   difference between species. `Region` is absent from the DTO and reconstructed from each `TileID`;
   reflection rejects reintroducing a serialized override.

   *The commit protocol under failure.* Stop after every file-protocol step and every IndexedDB
   request and transaction event; after each, a slot must resolve to the complete old save, the
   complete new save, or a committed deletion — never a mixture. Cover metadata-only listing, the
   exact slot IDs and kinds, manual/quick/rolling-auto overwrite, delete, and corrupt or unknown
   data. Assert the exact tagged live/tombstone shapes, including absence of generation/hash on a
   tombstone and rejection of missing live fields. On desktop, run two real processes against one
   temporary save directory: exactly one holds `saves.lock` and mutates, the other remains readable
   but read-only, and process death releases the OS lock. Also cover content-addressed world reuse
   only after exact byte/hash validation.

   *Timing, which only save and load expose.* A reload just before Campanian must apply it once on
   the next transition; a post-Campanian reload must preserve depleted stocks, health, macro
   mortality, and event history without applying it again; pre- and post-Toba reloads add only the
   elapsed presentation marker and change no simulation state. Repeated saves, reloads, and elapsed
   real time must not decay a food reserve: the next advancing turn applies spoilage once at phase-1
   start to the loaded reserve and then resolves the same consumption and final reserve, and repeated
   reloads neither skip nor duplicate a food operation nor serialize an intermediate total. An
   asynchronous save retains the snapshot it captured even when another turn completes before it
   commits, and completed-turn and terminal autosaves never capture partially resolved food.

   *Wire shapes with their own hazards.* `Health` round-trips as a JSON number with no narrowing,
   percentage quantization, or healing from load or split, and missing or invalid health is rejected
   rather than defaulted. `LastFoodReport` and `LastOutcomeReport` round-trip exactly — including
   their turn and canonical unavailable states — across new-game, post-split, planning,
   completed-turn, and terminal saves on both backends. Reject wrong-turn, non-finite, negative, and
   over-required food records, unavailable records carrying nonzero fields, mismatched outcome
   endpoints, and invalid signed or loss components, without mutating the running world, replaying a
   meal, or rerunning demographics. The explicit unreleased-development compatibility fixture maps a
   missing outcome report to unavailable; no other missing history is invented. A failed load
   preserves the current reports while a successful one replaces them with that world's. The
   exploration bitset round-trips exactly, including discoveries no longer near any band, and rejects
   a wrong-length mask or missing required initial and current-frontier bits; save and load must
   neither reveal again nor regress it. Also cover bounded event-feed eviction, last-mortality values
   including the macro cause, exact RNG restoration across both acute draw paths, FIFO completion
   ordering, and the golden future-equivalence invariant — identical future results from identical
   inputs and RNG state — before any UI exists.

8. **3D render layer** — evolve step 2a's retained minimal scene into viewport, fonts, palette,
   `scene3d`, `terrain`, `veil`, `orbit`, `picking`,
   `passages`, `markers`, `hud`.
   Implement and screenshot-review §8's exact three `GradeColors` anchor six-tuples through a
   step-8 display test harness with injected frames; this harness is not the public desktop
   `-screenshot` mode. Lock their exact
   RGBA/scalar values plus the two segment-midpoint interpolation fixtures in `palette.go` tests.
   Test chunk bounds, worst-case triangle counts, both detail meshes, collider replacement, orbit
   math, top/wall/boundary triangle→tile lookup, and all open/locked passage-overlay states; verify
   that `EpochGrade` interpolates continuously across all 401 turns with no discontinuity at either
   epoch threshold, that it drives only light, ambient, veil, water, and chrome, and that a changing
   `AridityIndex` alone advances neither `TerrainRevision` nor any terrain or veil rebuild. Assert
   the recurring climate-epoch name renders as text without a date range while campaign era retains
   its range, so color is never the sole cue; run the display
   harness on a machine with a graphics context. Step 10 exposes the retained harness through the
   desktop `-screenshot` and `-turns` flags for the §13 visual smoke test. Assert the marker layer
   renders
   at most one marker for each of the at most 256 bands. Veil tests cover the six-chunk/10,240-
   triangle bound, opaque common-height tops and boundary skirts, exact explored cutouts, no hidden
   biome/elevation/marker/full-passage leakage at low orbit angles, ignored hidden picking, and no
   rebuild on camera, selection, biome-only, or idle frames. Reveal changes rebuild once without RNG.
   Add table-driven viewport tests at `1`, `1.25`, `1.5`, `2`, and `3` device scale: assert the `2`
   cap, ceiling of fractional render dimensions, invalid/non-finite fallback to `1`, and exact
   render-pixel↔DIP round trips within one physical pixel. At `960 × 600` DIPs all required controls
   remain available; one DIP below either bound shows the resize overlay and emits no gameplay or
   scene action. The same DIP point must hit the same HUD
   control and map tile at every scale. A changed viewport advances `ViewportRevision` once,
   reallocates camera and screen-sized presentation targets once, and holds the font cache to the
   current three faces; an identical layout call does none of those things. Neither case may rebuild
   terrain/veil/colliders/markers or change an action, simulation revision, save, hash, or RNG state.
   Screenshot fixtures inject `1×` or `2×` explicitly and record both logical and render dimensions.
   Finish the step by adding the maximum-workload `World.AdvanceTurn` and frame-projection
   benchmarks plus calibration benchmark named in §8, running them repeatedly on
   `NativeBenchmarkReference` (`ubuntu-24.04`, `linux/amd64`), and checking the reviewed normalized
   medians plus allocation counts into
   `testdata/performance_baseline.json`. The baseline
   is recorded only after the benchmark workloads themselves are reviewed for exactly 6,144 tiles,
   256 bands, deterministic inputs, and complete turn/projection work; a fast benchmark that silently
   omits a subsystem is a test defect, not a performance improvement.
9. **Scenes + audio** — scene stack, widgets, grouped save/load scene, bounded two-second toast
   queue, all scenes including the terminal scene, typed `ui.Action` batches, and `SoundManager`
   wired to UI events. Cover the settings scene's master-volume slider and mute checkbox: the
   persisted level applies to the first emitted sound of a session, `Muted` silences without discarding the
   stored `MasterVolume`, unmuting restores exactly that level, out-of-range and hand-edited stored
   values clamp to `[0, 1]`, and neither control emits a `ui.Action`, revision, RNG draw, or sound of
   its own. With a controllable pending settings read, prove the first gesture constructs/resumes the
   manager at zero effective gain, retains at most one sound, emits nothing before settlement,
   discards the retained sound for stored `Muted == true`, and applies stored settings or whole-record
   defaults before any emission. UI copy tests cover every player-visible `gameapi.ErrorCode`, including
   “computer-controlled band,” “band limit reached,” and “missing technology prerequisite,” without
   importing the domain. Verify the §8 stats layout: the persistent top bar sums only sapiens
   populations and shows turn/year/era/season independently of selection; each band header shows that
   band's population/health, with food/mortality/workforce/research below; environmental stats
   appear in the tile inspector. Verify the campaign timeline marker, era, and date at turns 0, 100,
   200, 300, and 400, including exact endpoint placement, all seven fixed 10,000-year labels, and
   the three era boundaries. Confirm early turns move the marker by 300 years while final-era turns
   move it by 50. An early-extinction
   terminal frame stays at its exact fractional position. At minimum width, endpoint/current labels
   remain legible while intermediate labels may be suppressed without removing their tick marks.
   Clicking, dragging, rendering, or resizing the rail emits no action, consumes no RNG, and changes
   no selection, draft, revision, frame, or save payload. Field Notes open/hidden/drawer states do
   not move or cover the rail, and save/load of the same turn reproduces identical timeline content.
   Current macro episodes add non-interactive dated glyphs; player-relevant one-turn warnings appear
   in the top bar and on explored affected tiles without exposing a hidden epicenter or hidden archaic
   impact. Confirm warning/current/elapsed states, text alternatives, no future-catalog disclosure,
   and zero action/RNG/revision/save mutation from their presentation.
   During a warning, each explored migration-candidate row displays its warned impact and explains
   the multiplicative safety factor; zero attraction from danger is not mislabeled as an illegal
   move, and the UI never queues a sapiens escape automatically.
   Active abrupt-climate pulses add only the current-marker accent and explored tiles' regional
   values; no future pulse or hidden-region magnitude is disclosed.
   At new game, only East Africa terrain/coastline is inspectable. Hidden tiles expose no inspector,
   marker, resource, region, shelter, fauna, destination highlight, or far passage endpoint; one-
   endpoint passages show at most the local glyph. Moving or splitting sapiens reveals the specified
   persistent frontier, while archaic motion does not. Previously explored remote tiles retain
   current public stats and markers. An explored archaic inspector omits candidate rows whose
   targets remain hidden without changing the computer's full ranking; exploration never becomes an
   action or temporary LOS mode.
   Use mixed-species and multiple-band fixtures, no selection,
   a completed turn, and a loaded frame. New-game bands show 100% health; split and loaded bands
   show their inherited/restored condition, not a new-game reset. Health shows `100 * Health`
   as a condition percentage,
   with the §7 endpoint/interior examples and no display-rounding writeback or population scaling.
   Current values must agree with that frame, applied mortality must not be replaced by an
   uncapped base, and drafts/drawing must not alter accepted
   stats or consume RNG. Check “Last turn · Turn N” required/consumed/shortfall actuals and unmet
   percentage against the report, separate from current reserves. New-game and split-unavailable
   states show no numeric actuals; small positive shortages cannot be labeled zero through rounding.
   Planning and draft changes preserve history, the next completed turn replaces it, and loaded
   reports display without recomputation. Zero reserves alone must never be rendered as proof of
   starvation. The band inspector shows all six heritable values with their local pressure/effect
   text and no implication that they are technologies or manually assigned bonuses. A co-located
   sapiens band lists every eligible archaic target, explains that the action exchanges the whole
   modeled heritable vector reciprocally, and disables split/migration after acceptance while
   leaving workforce/research usable. Plain co-location presents competition but no automatic gene
   transfer. Test spent-action and invalid/dead-target states without hiding the archaic band's
   independent computer movement. Test that archaic selections
   remain inspectable but show “Computer controlled” and expose no assignment, research, split,
   or migration action. The sapiens inspector labels all five assignment controls with their
   distinct resource, mitigation, or research effect and displays both percentage and derived
   people; the `Shelter` control may read “Shelter & camp care”.

   **Help copy is tested against its owning rule, not rewritten here.** Every player-facing
   explanation of a workforce role, hazard, shelter effect, prey mix, or terrain rating is asserted
   equivalent to the §7 contract it describes, so a rule change that invalidates copy fails a test
   instead of shipping a confident falsehood. What these tests add are the specific misreadings the
   copy must not permit, which do not follow from the rules themselves:

   - **Potential is not harvest.** Foraging, Hunting & fishing, and megafauna help present
     per-worker rates as potential that shared scarcity then limits, never as food already obtained;
     draft edits reserve nothing.
   - **A workforce percentage is not an outcome percentage.** A 20% shelter share is not 20% of
     losses prevented, a work-risk weight is not a final incident probability, and the shelter cap
     bounds labor alone — no copy may present the asymptote as an attainable target.
   - **Prey weights are not populations.** `FaunaSummary` shows opportunity within one shared stock,
     never per-group counts or separately remaining animals; technology changes exploitation rather
     than the animals present.
   - **Caves are terrain efficiency, not protection.** Natural shelter reduces the share needed for
     exposure protection only — never hygiene, camp security, or a count of cave places.
   - **Mitigation composes multiplicatively.** The worked help example is two 50% reductions leaving
     25% risk, not a 100% allocation and not a promise about every normalized event probability.
   - **Absent structure stays absent.** There is no sixth Fishing slider, success button, catch or
     fish inventory, guard or hygiene sub-slider, and no accumulated camp level.

   Selecting an environmentally unsupported role must never silently redistribute an allocation, and
   no frame refresh, tile selection, or permitted turn or load may rebalance a dirty draft or bypass
   its guard.

   Test initialization from the accepted frame;
   below-, above-, and exactly-10,000 totals; the remaining/excess display; and Apply being enabled
   only for an exact, changed vector. Slider edits must produce no action, frame refresh, revision,
   RNG use, or save mutation. Apply must emit exactly one complete `SetAssignment` and refresh the
   frame only after acceptance. An authoritative rejection retains the draft and inline error while
   leaving the accepted world unchanged. Assert the UI holds at most one five-entry draft, a save
   captures the accepted vector, and archaic selections have neither draft nor controls. For valid,
   under-total, and over-total dirty drafts, mouse/keyboard/menu attempts to change selection, close
   the inspector, load/replace the world, leave for title, or end the turn must preserve selection,
   scene, draft, and world while showing one reusable inline message. Assert Discard restores the
   accepted vector with no command, snapshot, revision, or RNG use; editing back to baseline also
   clears dirty state. Successful Apply and Discard must never replay the blocked action; only a
   fresh retry proceeds, and rejected Apply leaves the guard in place. Test retained overlays use
   the same draft/guard, quick-save preserves the draft, and unrelated planning-frame refreshes
   preserve entered shares while updating derived people. Archaic presets use the same closed roles.
   Sapiens and archaic inspectors both show their selected band's population and current health
   in the header, with acquired technologies, active target, progress, and derived
   original-research preview; there is no species-global research display. The research panel
   renders all nine nodes and their exact prerequisite edges in stable enum/topological order.
   On a new game it shows no acquired nodes or selected target and exact zero progress for every node.
   All three roots are unlocked and all six dependent nodes are locked for both species; sapiens root
   controls are enabled for target selection, while the archaic inspector shows the same unlocked
   state read-only.
   Locked nodes identify their missing direct prerequisites, show exact zero progress, and disable
   target selection; unlocking occurs only in the next planning frame after prerequisites complete.
   The panel must explain that basic fire, ordinary stone tools, and baseline survival actions do not
   require DAG acquisition, and must display the selected costs without presenting them as historical dates.
   Test that one or
   many new acute or macro events in a completed turn request one event sound, while planning snapshots and
   loaded historical feeds request none. Choice-sound fixtures cover accepted enabled discrete
   activations and reject disabled/repeated/slider/hover paths; save-sound fixtures key exactly once
   by successful manual/quick-save operation ID and keep autosave/load/delete/failure silent.
   Implement and test Field Notes in this step: visible by default; lower-edge capped/scrollable
   layout; top-bar book button, plain-`F` toggle, and hidden restore tab; stable context priority for
   explicit trait/technology/passage/interbreeding focus, newly established region, current/warned
   macro context, band, region/biome, event, and campaign topics; and all
   three required text blocks, including sourced abrupt-climate, active Campanian, and no-effect Toba
   entries that label gameplay envelopes, warning scale, and demographic uncertainty as abstractions.
   The first completed-turn insertion of Arabia or the Levant focuses the sourced founder-estimate
   entry from the pre/post-frame achievement difference; an already-latched loaded frame does not
   replay it and no founder counter or target appears.
   Fishing entries must distinguish the
   approximately 90 ka Katanda points/fish association, 42 ka Jerimalai pelagic catch, later 23–16 ka
   shell hook, and post-campaign Lake Condah stone-walled traps; no entry may collapse those dates
   into a 40 ka fishhook or stone-trap claim. At minimum width/height it must not cover required inspectors or
   alerts. Catalog completeness and reference tests cover every closed enum/context key. Toggling,
   focusing, and scrolling leave selection, draft, action batch, revision, RNG, and frame untouched.
   Exercise absent/malformed/future-version `UISettings`, local cross-session persistence of all
   three preferences, whole-record defaulting (Field Notes visible, `MasterVolume` `0.5`, `Muted`
   false), required-field presence, null and wrong-type rejection, an obsolete one-field development
   record, `MasterVolume` clamping on read and write, write failure, and campaign load/delete
   independence on desktop and the separate web database. With the initial read pending, all three
   preference controls render disabled and issue no write; installing either a valid result or the
   whole-record defaults enables them atomically. This step owns `ui_settings_file.go` and
   `ui_settings_idb.go`, including their shared record-validation contract; step 11 composes and
   exercises the latter in a real browser rather than implementing another preferences store. Use a
   controllable completion order to prove one active write plus one replaceable `pendingLatest`
   record, revision-tagged stale completions that never roll back live UI, and last-value-wins state.
10. **Ebitengine host/composition adapter + desktop entry** — replace step 2a's fake wiring in
    `pkg/app` and `main.go` with the real application/render/UI/audio graph and add the desktop flags,
    including integration of step 8's retained display harness behind `-screenshot` and `-turns`.
    Add `internal/verification` here, because this is the step that registers
    `-headless`/`-seed`/`-policy`/`-checkpoint-json` and therefore the first that needs a driver
    behind them: the five frame-driven route policies, `CheckpointRecord` with its canonical
    sorted-key encoding, and `ReferenceRun`. Assert that it reaches the game only through
    `gameapi.Game`, that it performs no I/O and reads no wall clock, and that repeated runs of the
    same `(seed, turns, policy)` produce identical bytes. Step 11 adds the js-tagged emission and
    browser harness on top of this same entrypoint.
    Retain step 2a's injected one-shot ready callback and prove it fires only after the first
    completed `Draw`, never from `LayoutF`, `Update`, before that draw completes, or from a later frame. With a counting fake,
    prove one `EndTurn` produces exactly one computer-planning batch followed by one five-phase step
    and one revision increment; no intermediate archaic state is framed or saveable. Also prove
    repeated/ill-ordered `EndTurn` is rejected, idle Ebitengine updates and workforce draft
    edits produce zero commands, snapshots, revisions, or steps, one enabled assignment Apply
    produces exactly one command and one refreshed frame, and an Apply result is required before
    the editor unlocks. One accepted `Interbreed` produces one planning mutation and one refreshed
    frame, spends only its sapiens actor's spatial action, and does not step until `EndTurn`.
    A dirty-draft exit attempt must produce zero `EndTurn`/`BeginLoad` calls and
    no queued load; test Apply plus `EndTurn` in one campaign batch, repeated attempts, rejection,
    Discard, and fresh retry so neither implicit turn advancement nor deferred replay can pass.
    Save completion may arrive after later turns. An accepted load, even while queued, freezes
    simulation-changing input and workforce draft editing; failure restores the old clean editor,
    success clears the old-world selection/draft, and input remains frozen until no load is active
    or queued. An immediately rejected request must neither acquire a freeze nor release one held
    by another pending load. No storage action advances time. With a fake monotonic clock and controllable
    storage, prove the three legal action-batch classes and every forbidden class mixture. A
    structurally invalid batch invokes nothing; a semantic failure in a valid campaign batch stops
    later actions while preserving earlier accepted commands and their latest frame. Also prove
    per-turn/five-minute triggers, oldest-slot rotation, captured-revision accounting,
    trigger coalescing, failure retry, and queue bounds. Prove `-dumpmap`, `-headless`, and
    `-screenshot` each wire the no-op `SoundManager` and open no audio context, so the §13 gate
    cannot come to depend on runner sound hardware. With an injected `DeviceScaleSource`, exercise
    `LayoutF` across resize, fractional scaling, a simulated monitor move, minimization, and restore;
    assert each returned render size and viewport revision and prove no layout call advances the
    simulation, emits an action, touches storage/audio, or consumes RNG. Implement §3's explicit
    `gameapi.Game`, `CampaignRepository`, and `UISettingsStore` log decorators and desktop session
    sink here. Against fakes, prove each wrapped call occurs exactly once, start/end correlation is
    stable, empty `PollStorage` and idle frame methods are silent, returned values/errors are
    unchanged, no logging path asks for an extra snapshot, and decorated and bare state hashes agree.
    Prove an accepted host batch emits one `action.dispatch` per invoked typed action in invocation
    order, while a rejected batch emits one `action.rejected` and invokes/logs none of its members.
    With an injected temp directory and stdout/stderr writers, prove two sessions produce distinct
    `africa2ice-*.jsonl` files and announce their absolute paths before `session.start`; normal close
    produces `session.end`, and entrypoint- and worker-boundary panics each log a bounded stack, close
    the sink, and re-panic with the original value. Inject successful and failing entropy readers to
    cover the 32-hex-character session ID and its warning/fallback without touching `WorldRNG`.
    Inject a smaller byte limit into the sink test to
    prove the production 32 MiB policy wraps in the same file with `log.wrapped`;
    prove oversize records are clipped and marked and create/write failures fall back once to stderr
    without failing a use case. Assert serialized save/frame/RNG payloads and free-form Field Notes
    never appear in fixture logs.
11. **Web target and release-build CI** — replace step 2a's fake `main_js.go` wiring with the real
    graph, add release mode to its dev-only build script, and perform real-browser composition of step 7's
    generation-addressed `internal/adapters/storage/indexeddb.go`, IndexedDB upgrade/versionchange handling,
    session-writer Web Lock lease,
    step 9's independent `africa2ice-ui` preference database,
    `web/index.html`, dev/release modes in `build_web.sh`, pinned Binaryen and `brotli`,
    compressed-size enforcement including §10's one-way ratchet of `MaxCompressedWasmBytes` down to
    this build's measured Brotli size plus `CompressedWasmHeadroom`, relative project-site asset paths, and the native OS/WASM release-build workflow; run
    request/abort/commit timing tests against a
    controllable IndexedDB fake. Add `tools/run_wasm_go_tests.mjs`: it compiles the js-tagged storage
    adapter suite with `GOOS=js GOARCH=wasm go test -c`, serves a minimal harness, runs the Go test
    binary in pinned Playwright Chromium against real IndexedDB, and fails on the Go exit/status.
    Run it in `web-release` before the application smoke; desktop `go test ./...` and cross-builds do
    not prove that this adapter suite executed. Add `tools/run_wasm_checkpoint.mjs` and
    `internal/verification`'s js-tagged checkpoint test on the same harness: it must emit exactly one
    sentinel-delimited canonical `CheckpointRecord` block, upload `reference-checkpoints.json`, and
    produce bytes identical to the native matrix's records for `ReferenceSeed`. This is the artifact
    the `cross-target-determinism` job compares, so a missing, duplicated, or non-canonical block is a
    build failure rather than a skipped comparison. Assert the browser entry leaves `DisableHiDPI` false and the host
    page owns only CSS sizing—no JavaScript `devicePixelRatio`, canvas backing-size assignment, or
    input rescaling. Expand step 2a's pinned Playwright smoke into an automated pre-deployment gate.
    It serves the optimized bundle with `?e2e=1`, waits for the stable ready marker, and fails on a
    page error, panic, or unexpected `console.error`. It then starts a campaign, queues one
    deterministic valid whole-band migration to the first ranked candidate, ends the turn, and
    quick-saves to IndexedDB; this ordinary storage smoke does not assume that the first legal move
    crosses the initial reveal frontier. Before reload it
    captures the observer's canonical `E2ESummary`: turn, year BP, era, `CalendarProgress`, total
    sapiens population, the lowest-ID living sapiens band's ID/tile/population/health/stored FU and
    complete `LastFoodReport` and `LastOutcomeReport`, the sorted explored-tile IDs, and the SHA-256 of one byte per tile
    (`0` or `1`) in ascending tile-ID order. A separate injected frontier fixture places the actor at
    a known reveal boundary, proves that a legal move grows the explored set, and chooses the lowest
    newly explored ID and lowest still-hidden ID as probes. After page reload and load of the
    ordinary quick-save, the gate waits for exact summary equality. A second save/load run of the
    frontier fixture requires its new probe to remain present and hidden probe absent. Summary
    JSON uses stable enum strings and Go's ordinary lossless finite-`float64` JSON representation,
    never formatted UI percentages or labels. The observer is presentation/test data only — never a
    save field, slot record, campaign hash input, log payload, network request, or second game API. Its
    matrix covers DPR `1`,
    `1.25`, `2`, and `3` (capped to `2`), plus browser zoom, orientation, and resize, with stable DIP
    geometry and matching pointer/touch hits.
    A build-tagged sink test injects stdout, parses every emitted line as one JSON object, and proves
    the web path opens no temp file, IndexedDB record, download, or network request. The browser smoke
    check confirms `session.start`, one decorated action, and one induced error are visible in the
    JavaScript console; a callback-boundary fixture confirms its injected `PanicGuard` logs before
    rethrowing. Validate the staged `web/` artifact and §8's automated timeouts. Do not add or enable
    Pages configuration, upload, environment access, or deployment in this step; step 13 does so only
    after final calibration and release readiness have passed.
12. **Balance pass** — use Appendix B and every **Initial** Appendix C row as the accepted starting
    configuration, including any accepted step-5e tuning. Earlier consuming steps have implemented
    those exact values; Appendix C has no remaining **Open** input. Rerun the complete step-5e corpus
    and tune **Initial** values against it; do not
    change **Locked** values. Any accepted tuning must update the owning §7 rule, Appendix B where
    applicable, and Appendix C together before the configuration is called calibrated. Retain food-scaled
    positive growth and the
    selected storage/consumption ordering, fixed recovery, and `1 + (1 - Health)` chronic mapping.
    Retain the combined phase-3 health clamp, phase-5 outbreak damage, and separate direct deaths.
    Start with the selected temperature-responsive water-demand range `[1.0, 1.3]`, including
    seasonal/long-term variation, the 20°C/35°C anchors, and the `0.02`-per-degree linear ramp.
    Validate the step-4 latitude table and locked temperature coefficients against the seed corpus
    without changing them; validate the selected disease health-technology effects. Validate and,
    where its Appendix C rows are **Initial**, tune the complete genetics balance contract here: six starting
    values for every species/start-region profile; all trait effect/selection functions; same-species
    and interbreeding flow rates plus `MaxGeneFlowPerTurn`; mutation eligibility/probability/entry
    values; the selected UV factors and derived UV/hypoxia functions; and integration with food,
    water, health, exposure, and disease without double application. Start endemic health balance with the selected `0.020`/`0.010`
    Riverine Woodland rates, `0.010`/`0.010` Savanna and Coastal Shrubland rates, and
    `0.005`/`0.005` rates for the other three biomes. Retain multiplicative component-specific
    tech/hygiene protection, hygiene on the camp portion only, and no cave bonus to disease health. Retain the outbreak's shared-draw
    linear health loss and explicit technology/genetic factors; hygiene remains probability-only for outbreaks.
    Tune `r`,
    `BaselineKCurve(V)` knots, biome-capacity factors, absolute flora/fauna capacities and source-specific heritable modifiers, resource
    caps/regeneration rates, initial resource-stock fractions and seed-perturbation parameters, degradation
    damage/recovery rates, regional fauna mixes and profile-specific exploitation inputs,
    biome foraging rates, profile-adjusted hunting and megafauna rates with the selected higher-rate
    constraint, collection-tech modifiers, hunting-work risk coefficient vectors with the higher
    active megafauna-total constraint, workforce role
    yields/risks/mitigation, natural-shelter ratings,
    `MaxShelterMitigation`, `ShelterHalfSaturation`, `NaturalShelterEfficiencyBonus`,
    `CampSecurityScale`, `CampHygieneScale`, finite acute base-weight profiles and camp/non-camp partitions, the two
    endemic health-rate inputs and their applicable health-tech effects independently of direct
    mortality, chronic risk, `MaxChronicRate`, acute probability/loss bounds, outbreak health bounds,
    technology mitigation,
    research costs, `MaxResearchPerTurn`, `ResearchHalfSaturation`, tech capacity multipliers,
    ordinary and named-passage movement costs, the water survival-equivalent migration-ranking
    conversion, archaic assignment presets and technology priority, `SplitStressThreshold`,
    `MinEstablishedBand`, `MaxArchaicBands`, `BeringiaOpenFraction`, climate amplitudes and the
    source-backed abrupt-pulse catalog and `MaxAbruptClimateOffset`, plus Campanian impact scales,
    resource/habitat factors, `MaxMacroLoss`, and `MaxRefugiumMitigation`, so the documented sapiens reference policy can establish
    a band in at least one destination and survive through
    turn 400 while the fixed archaic computer policy competes in the same world. Dedicated
    deterministic sapiens route policies must also prove Frangistan, South Asia, the Yellow River
    Basin, Sahul, and Beringia are each independently reachable, using `named-asymmetric-v1` where a
    named crossing is required. Run the sapiens reference policy plus `ranked-pressure-v1` across a
    exact `BalanceSeedCorpus` declared by the determinism contract, never exceeding `MaxBands`; the
    reference seed must win, while extinction and dispersal-failed runs remain possible.
    Validate without retuning the eight starting bands, fixed 50/50 split, authored starting anchors
    and their frozen resolved tile IDs, Toba no-effect marker, Campanian identity/date, and checked-in
    Campanian masks/checksums. Run §8's native benchmarks, automated browser timeouts, and complete
    reference-machine profile. All four DPR/detail floors must pass; low detail may remain a player
    fallback but does not excuse a normal-detail failure. Finish with
    every benchmark, route policy, reference margin, and browser timeout green. This step calibrates
    but does not publish either target.
13. **Release closure and publication** — begin only after step 12 and the complete §13 verification
    suite are green from a clean checkout. Add the final `release-readiness` CI job: it reruns the
    `linux/amd64` reference campaign on `NativeBenchmarkReference`, all five route policies, the
    checked-in native benchmark
    regression comparison, the optimized browser smoke, the compressed-size gate, and a manifest
    check that no Appendix C configuration data row has status **Open** (the status legend and prose
    necessarily contain the word). It depends on the native matrix and `web-release`.
    It also depends on the successful `cross-target-determinism` checkpoint comparison.
    A failure in any prerequisite or readiness check must leave publication jobs unstarted.

    Complete the release surface before enabling delivery: update `README.md` with desktop and web
    launch instructions, controls, save locations, the announced desktop session-log path, browser
    console-log instructions, and the unsigned-native trust prompt; verify `LICENSE`; generate and
    review `THIRD_PARTY_NOTICES.md`; audit every Field Notes citation and bundled-font/dependency
    license; and record §8's reference-machine profile in `docs/PERFORMANCE.md`. A release candidate
    must load the oldest supported save fixture, complete a turn, save again, and reload on desktop
    and web. The release record names the SemVer tag, VCS revision, Go/Binaryen/browser versions,
    clean/modified status, save schema and algorithm versions, raw/Brotli/gzip WASM sizes, and performance
    results. `runtime/debug.ReadBuildInfo` remains the runtime source for module/VCS identity; release
    builds must retain build-VCS metadata rather than replacing it with an unrelated hand-maintained
    version string.

    Only now add and enable §10's `configure-pages`, `upload-pages-artifact`, and `deploy-pages`
    steps in `.github/workflows/ci.yml`, configure the repository's Pages source as GitHub Actions,
    and prove a pull request skips them while a successful `main` push deploys the exact optimized
    artifact that passed `native`, `web-release`, and `release-readiness`. Verify the resulting URL
    with §13's post-deployment smoke. Once enabled, later qualifying `main` pushes retain this
    continuous-deployment behavior; initial roadmap work must never publish merely by reaching step
    11.

    Add `.github/workflows/release.yml` for annotated `vMAJOR.MINOR.PATCH` tags. It rejects a commit
    that is not reachable from `main`, reruns the release-readiness gates on the exact tagged tree,
    builds Linux/Windows on the named x86-64 runners and macOS on `macos-15-intel` plus `macos-15`
    rather than cross-compiling either architecture, asserts `go env GOARCH`, and publishes portable archives
    named with tag/OS/architecture for Linux amd64, Windows amd64, and macOS amd64/arm64, plus one
    `SHA256SUMS` covering the exact uploaded bytes. Extracting an archive yields the executable,
    `LICENSE`, `THIRD_PARTY_NOTICES.md`, and a short run/readme file—no installer or hidden mutable
    dependency. Build and verification jobs retain `contents: read`; only the final publisher gets
    `contents: write`, using the run's `GITHUB_TOKEN` rather than a personal token. V1 archives are
    unsigned; release notes must say so and document the expected OS
    warning. Code signing, notarization, installers, package-manager feeds, and automatic updates
    require a later distribution decision and must not be implied by the presence of an archive.

---

## 13. Verification

### Automated headless/build gate

These checks require no display and no sound device, and cover both targets and the
build-tag-independent architecture gate:

```bash
golangci-lint config verify && golangci-lint run && go vet ./... && go test ./... && go build ./... && GOOS=js GOARCH=wasm go build -o /dev/null .
```

`config verify` catches a typo'd depguard rule that would otherwise silently enforce nothing.
`go test ./...` includes the source-parsing `internal/archtest`, so desktop and `js`-tagged source
violate the build even without a linter present. The final cross-build type-checks the actual web
selection too.

The test suite also treats observability as behavior-preserving architecture, not best-effort prose.
On non-failure sinks it parses every emitted line as one JSON object; it asserts stable event names,
required common fields, start/end correlation, operation-specific identifiers, clipping, file
wrapping, close, panic, and single-warning fallback; and runs identical command/storage fixtures
through bare and decorated ports.
The two paths must return equal results and errors, invoke each wrapped method once, consume the same
RNG draws, and end with the same campaign-state hash. Source-level deny fixtures prove the inner
layers cannot import the logging adapter and no package other than that adapter can import `log` or
`log/slog`.

The lint/vet/test/native-build sequence runs in that order in GitHub Actions on Linux, macOS, and
Windows; a lint failure therefore prevents tests from starting in the affected matrix job. The web
cross-build, optimized release build, artifact checks, browser runtime smoke, and size assertion run
in the separate Ubuntu `web-release` job:

```bash
./build_web.sh --release
test "$(brotli -q 11 -c web/main.wasm | wc -c | tr -d ' ')" -le 5500000
gzip -9 -c web/main.wasm | wc -c    # recorded, not gated: the no-Brotli fallback stream
npm --prefix tools/web-e2e ci
npm --prefix tools/web-e2e run install-browser
node tools/run_wasm_go_tests.mjs
node tools/run_wasm_checkpoint.mjs   # writes reference-checkpoints.json for cross-target-determinism
npm --prefix tools/web-e2e test
```

The artifact check requires exactly one deploy root under `web/`, including non-empty `index.html`,
`main.wasm`, and `wasm_exec.js`; confirms that the HTML uses relative runtime URLs; and rejects a
staged precompressed `main.wasm.gz` or `main.wasm.br`. The browser test starts its own loopback server, uses only the
staged release root, and enforces §8's readiness/action timeouts. It must fail if it accidentally
loads a development server, checked-out source, or network resource other than that loopback origin.

After build step 13 enables publication, a pull request must report `deploy-pages` as skipped and
create no Pages deployment. On a push to `main`, `deploy-pages` must declare
`needs: [native, web-release, cross-target-determinism, release-readiness]`; an intentionally failed native, web-release,
reference-policy, manifest, or benchmark-regression fixture therefore leaves it unstarted. The
`release-readiness` job runs `tools/check_benchmarks.sh` on `NativeBenchmarkReference` and consumes the
exact optimized artifact already tested by `web-release`; it must not rebuild a different deployable
bundle. After a real deployment, a bounded retry smoke check requests the emitted
`page_url` plus its relative `main.wasm` and `wasm_exec.js`, requires successful responses, and
requires the wasm response to use `Content-Type: application/wasm`. This post-deploy smoke reports a
bad publication immediately; it does not replace the pre-deploy artifact checks.

The native matrix and web-release job also run the reference-policy checkpoints with
`ReferenceSeed` and upload `reference-checkpoints.json`. A dependent `cross-target-determinism` job
downloads the Linux amd64, macOS arm64, and Chromium js/wasm records
and requires byte-identical campaign-state hashes and named margin values at turns
`0, 100, 200, 300, 400`. `release-readiness`, Pages deployment, and tagged publication all depend on
that job.

**One Go entrypoint produces all three records.** `internal/verification.ReferenceRun(seed, turns,
policy)` constructs a `GameService`, drives it through the `gameapi.Game` inbound port — frame in,
`gameapi.Command` and `EndTurn` out, exactly the path `pkg/app` uses — and returns the
`CheckpointRecord` slice. It performs no I/O, reads no wall clock, and consumes no entropy, so it is
callable identically from a desktop binary and a browser. This is what makes "the same use case,
not a JavaScript reimplementation" an enforceable property rather than an intention: there is one
function, and the two targets differ only in how its return value reaches a file.

- **Desktop.** `main.go` (`!js`) parses `-headless`/`-turns`/`-seed`/`-policy`/`-checkpoint-json`,
  calls `ReferenceRun`, and writes the encoded records to the named path.
- **Web.** `tools/run_wasm_checkpoint.mjs` compiles `internal/verification`'s js-tagged checkpoint
  test with `GOOS=js GOARCH=wasm go test -c`, serves the minimal harness `tools/run_wasm_go_tests.mjs`
  already uses, runs it in pinned Playwright Chromium, and scrapes the emitted block into
  `reference-checkpoints.json`. It fails on a Go non-zero exit, a page error, a missing block, or
  more than one block. The wasm binary carries no `flag` plumbing, which is why this is a test
  binary rather than a mode of `main_js.go`.

The emission is delimited rather than parsed out of ordinary output. `run_js_test.go` writes the
single canonical line between the exact sentinel lines `AFRICA2ICE-CHECKPOINT-BEGIN` and
`AFRICA2ICE-CHECKPOINT-END` on stdout, which the pinned `wasm_exec.js` forwards to `console.log`.
The checkpoint binary installs no observability session, so nothing else writes to its stdout and the
block cannot interleave with §3's JSONL records; the sentinels remain because a page-level error or a
Chromium warning can still reach the same console.

`CheckpointRecord` encoding is canonical at the producer, not repaired at the aggregator: object keys
sorted, no insignificant whitespace, stable enum strings, and Go's ordinary lossless finite-`float64`
representation — the same rules `E2ESummary` follows, for the same reason. Both targets must emit
byte-identical files before the comparison job runs, so the job's only job is to compare. A
desktop-only fixture additionally runs `ReferenceRun` through both the `main.go` flag path and the
test path and requires identical bytes, so the two callers cannot drift.

The local reproducer is
`go run . -headless -turns 400 -seed 0x9e3779b97f4a7c15 -policy reference -checkpoint-json <path>`,
and `node tools/run_wasm_checkpoint.mjs` reproduces the browser record.

Step 5e's viability gate is a different measurement at a different layer and does not use this
driver: it runs before `gameapi` and `SaveState` exist, so it asserts domain-observable outcomes —
destinations reached, margins, `MaxBands` — against `World` directly, and cannot compute a
campaign-state hash, which §3 makes an application concern. Step 6 adds one equivalence fixture
requiring the frame-driven route policies here and 5e's domain-level route decisions to select the
same command sequence for the same world, so "the same reference policy" stays true across the two
gates.

An annotated SemVer tag additionally exercises `.github/workflows/release.yml` from the tagged
commit. The verification gate downloads every candidate archive, checks it against `SHA256SUMS`,
asserts the expected executable and notice files and no unexpected payload, and starts the executable
on its native runner far enough to emit `session.start` with the tagged VCS revision. Publishing the
GitHub Release is downstream of all archive checks; no job assembles an archive on one OS and labels
it as another.

Every desktop mode, including verification modes, first prints §3's session-log path. Automated
parsers treat that single line as a preamble rather than map or balance output; structured operational
records go to the announced file and do not pollute the deterministic checkpoint format. `main.go`
ships the desktop verification flags specified in §10. The first two modes are genuinely headless:

```bash
go run . -dumpmap
```

Prints the 96×64 grid as ASCII biome codes with a second region-code layer. Africa, Arabia, the
Mediterranean, Frangistan, India, the Yellow River Basin, Sahul, Siberia, the Bering Strait, and
western Alaska should be recognizable at a glance, and every destination must have at least one
habitable land tile — the direct check on the real-geography requirement.

```bash
go run . -headless -turns 400 -seed 0x9e3779b97f4a7c15 -policy reference
```

Runs the documented deterministic sapiens reference policy for the full campaign while
`ranked-pressure-v1` controls archaic bands, with no window. It prints at every 50-turn checkpoint
and every active macro-episode turn: year, campaign
era, explored-tile count, population and band count by species, dominant biome,
`LongTermTempOffset`, `SeasonalTempOffset`, `ClimateNoise`, their summed `GlobalTempOffset`, the
minimum/maximum regional `AbruptClimateOffset`, minimum/maximum `TileTempOffset`, active macro
episode IDs, and `BeringiaOpen`, mean/maximum tile `Degradation`,
cumulative chronic loss, macro-event impact totals, acute incident/loss totals by kind,
technology-holder counts by species, mean heritable values by species, mutation events by
trait, the established-region set, and current sapiens population inside versus outside Africa.
The geographic telemetry is descriptive only: it has no founder-flow target, counter, pass/fail
threshold, or effective-population inference. At every checkpoint
and at termination it asserts `len(Bands) <= MaxBands`, exactly `AssignmentCount` basis-point
allocation entries totaling 10,000, exactly `TechCount` finite non-negative progress values, exactly
`HeritableTraitCount` finite `[0, 1]` values, no pending spatial intent, and a cleared
`SpatialActionUsed` marker per band.

It also reports the **band-budget split**: sapiens and archaic band counts, the archaic count against
`MaxArchaicBands`, and the number of consecutive turns on which no sapiens band could have split
because the global cap was full. `len(Bands) <= MaxBands` is an invariant and must hold; the budget
split is a balance signal and must be read rather than merely asserted. A run in which sapiens are
locked out of splitting for a sustained stretch is a balance failure even though no invariant broke,
and it is exactly the failure `MaxArchaicBands` exists to prevent — so the checkpoint must make it
visible instead of leaving it to be discovered as an unexplained plateau in sapiens expansion. The
new-world reference run asserts total `archaic_band_count <= MaxArchaicBands` at every checkpoint.
Separate restore fixtures cover a valid world already above a lowered sub-cap and assert that its
archaic count cannot increase through policy splitting while it remains at or above the cap. This
tests observable counts and transitions rather than unpersisted band-creation provenance. The
explored mask must be monotonic, include every current sapiens frontier, and remain unchanged by
archaic-only movement. Each band's stored FU must also be finite and in
`[0, FoodStorageCapacity(b)]` after
turn-end finalization, including the terminal checkpoint. It must finish at turn 400 with the exact
long-term LGM offset, an instantaneous offset inside
the documented envelope, the correct Beringian passage state, all degradation values within bounds,
and a terminal result.

**The reference run must win with margin, not marginally.** Cross-target checkpoint equality is a
separate mandatory gate; the margin exists because a reference policy that succeeds only at a
knife-edge is not a robust calibration. §12's tuning target is therefore not "the reference seed
wins" but "the reference seed wins
with room to spare," asserted as explicit headroom rather than a terminal enum value:

- Its first destination region is established by **turn 350 or earlier**, so victory does not depend
  on the last few transitions resolving favorably.
- The establishing band holds at least `2 * MinEstablishedBand` people at that turn, so the
  achievement does not latch on a band sitting exactly at the threshold.
- At turn 400 it retains at least **five** living sapiens bands and total sapiens population of at
  least `10 * MinEstablishedBand`, so one unfavorable acute draw cannot erase the useful survival
  margin even though it cannot produce exact extinction under the selected severity table.

Those margins are themselves balance data: a tuning pass that can only just satisfy them has produced
a knife-edge configuration and should keep tuning. If a future change makes the margins genuinely
unreachable while the campaign is still well-formed, improve the reference policy or select a
representative seed rather than narrowing the assertion back to a bare win. Appendix C records the three margins as **Policy
(tighten only)** for this reason: they are verification thresholds asserted against the tuned run,
not balance defaults §12 may relax to make that run pass.

Every supported target gates on these margins and uploads the checkpoint record. The
`cross-target-determinism` job then requires exact agreement; architecture divergence is a release
failure, not a permitted signal.

The reference policy keeps its route-leading band whole and follows the deterministic time-expanded
route to the nearest destination. On the final approach only, it holds instead of entering that
destination while the planning-frame band population is below
`referenceRouteDeparturePopulation = 5 * MinEstablishedBand / 2 = 50`. The ten-person reserve above
the required 40-person arrival margin covers ordinary same-turn attrition without pretending to be
a mathematical guarantee: `CampaignOutcome.FirstDestinationPopulation` records the largest
post-resolution sapiens band in the destination region newly established on that turn, and the gate
requires the actual value to be at least `2 * MinEstablishedBand` on every corpus seed. The reserve
does not constrain directed reachability policies, ordinary migration, or the route leader after its
first destination; broad low-population holding can strand it on a marginal tile and suppress the
subdivision the reference policy exists to exercise. Non-route reference bands consider migration
only under split pressure and otherwise use the ranked ordinary candidates. This exercises the same
always-present derived score exposed by the UI without making that policy a domain restriction.

Unit fixtures separately cover both loss modes. The separate archaic policy is part of the domain
and must produce the same decisions before and after reload. Separate destination fixtures exercise
every destination because a single 400-turn reference run need not and should not be forced to
establish all five.

### Display-required visual smoke test

The screenshot path boots Ebitengine and Tetra3D and therefore requires a real graphics context. It
is a local/manual check by default, or a separate CI lane configured with a virtual display such as
Xvfb; it is not part of the headless gate above.

```bash
go run . -screenshot /tmp/a2i.png -turns 20
go run . -screenshot /tmp/a2i-low.png -turns 20 -terrain-detail low
```

Boots the real 3D renderer, advances 20 turns through the CLI harness, writes a PNG of the
framebuffer, and exits. Generate and inspect both normal and low-detail screenshots so chunk seams,
lighting, biome colors, and side walls are visible rather than merely compiled; the PNGs are test
outputs, not committed assets.

For a release candidate, run `npm --prefix tools/web-e2e run profile` on the documented reference machine
against the optimized local bundle in normal and low detail at both DPR 1 and DPR 2. The
script performs the fixed 30-second post-warm-up sample and emits machine-readable FPS, frame-gap,
turn-latency, and peak-process-memory results. Copy the results and complete machine/browser identity
to `docs/PERFORMANCE.md`; §8's floors, not a subjective “feels responsive” judgment, decide whether
the candidate passes.

**Interactive desktop:** `go run .` → Title → New Game. This walkthrough covers only what needs a
display, a real input device, or a human judgement. §12's steps own everything a headless test can
assert, so nothing here re-verifies a numeric fixture by hand — a person clicking through allocator
arithmetic proves less than the test that already runs on every push, and takes an afternoon.

*Rendering and camera.* Orbit with right-drag, `Q`/`E`, and scroll; change elevation and pan. Chunk
seams, side walls, lighting, and biome colours must read correctly at every angle in both terrain
detail modes. Hover top and wall geometry and confirm the tile inspector resolves the tile under the
pointer, including across a chunk boundary.

*The veil, which a screenshot cannot replace.* On a new game, orbit low and all the way around:
unexplored Eurasian, Sahul, and Beringian terrain must expose no elevation, coastline, biome colour,
marker, inspector, or full passage line from any angle, and clicks on it must do nothing. Then use
split and completed-migration fixtures to watch successive frontiers open, confirm remote archaic
movement reveals nothing, and reload to confirm the uncovered area comes back exact.

*High-DPI, which only real monitors exercise.* At OS scales `1×`, a fractional scale (`1.25×` or
`1.5×`), and `2×`, keep the same logical window size: controls must retain their DIP size while
text, one-DIP rules, and terrain gain physical-pixel detail. Move the running window between
differently scaled monitors and resize it — every control and tile must stay under the pointer,
labels must not clip, and the viewport may change without a simulation revision or a terrain or veil
rebuild. Record logical size, render size, and render scale beside any screenshot kept for
comparison.

*Layout under real constraints.* Confirm the timeline sits directly beneath the top bar, keeps its
tick spacing and era boundaries, ignores clicks and drags, and is never moved or covered when Field
Notes opens, hides to its tab, or becomes a narrow-window drawer. At minimum width the endpoint and
current-date labels must stay legible while intermediate labels may drop. Field Notes must not cover
a required inspector or alert at any size.

*Audio and feedback.* A triggered acute incident produces one typed feed entry and one sound; a
no-event turn produces neither. A completed turn fills Auto 1 with a two-second toast without
interrupting play. `Ctrl+S`/`Cmd+S` quick-saves to slot 99 without opening a scene.

*Flows worth driving by hand.* Open `ESC` → Save, overwrite Manual 1 with no confirmation modal,
exit to title, and confirm the browser groups three Manual, one Quick, and three Auto rows; load
Manual 1 and confirm the restored turn and pending commands. Then, with a dirty workforce draft,
attempt every exit route a real player has — selecting another band, closing the inspector, ending
the turn, loading, and returning to title from a retained overlay — and confirm each stays put
behind the same inline Apply/Discard message, that an invalid total disables Apply while still
allowing Discard, and that resolving the draft never replays the blocked action.

*Content review, which is a judgement rather than an assertion.* Read the pigmentation, immune,
fishing, Toba, and Campanian Field Notes entries and confirm each separates evidence from game
abstraction, labels its remaining uncertainty, and keeps the Katanda, Jerimalai, shell-hook, and
Lake Condah dates distinct. Confirm no help text presents potential as harvest, a workforce
percentage as an outcome percentage, prey weights as populations, or caves as hygiene protection —
step 9's enumerated misreadings, checked here as prose a person actually reads.

*Scripted date fixtures.* At the turn-100/200/300 fixtures confirm the date stays continuous while
the next turn advances by 150/100/50 years. The Toba date falls inside the transition from turn 20
(`74,000 BP`) to turn 21 (`73,700 BP`); completing that first crossing of `73,880 BP` adds only the
dated `73,880 BP` glyph and its Field Notes entry — no warning, overlay, or simulation
change of any kind. At the pre-Campanian fixture, confirm that only explored affected tiles and
resident sapiens receive the warning, that completing the `39,850 BP` transition applies its
one-turn factors and records macro mortality separately while capacity stays strictly positive, and
that the following turn removes those factors while depleted stocks recover normally.

**Web:** `./build_web.sh --dev && (cd web && python3 -m http.server 8080)` → load
`http://localhost:8080`. Step 11's Playwright gate already drives the DPR `1`/`1.25`/`2`/`3` matrix,
browser zoom, orientation, resize, and the boot/new-game/migration/end-turn/quick-save/reload path on
every push, failing on any page error, panic, or unexpected `console.error`. This pass therefore
covers only what that harness cannot reach or cannot judge.

*Rendering through the WASM path.* Confirm the canvas fills the window, terrain renders, the orbit
camera responds, and both detail modes pick correctly. This repeats the desktop visual review
because the browser is a different rendering path, not because the expectations differ. Confirm the
veil still hides everything beyond East Africa at a shallow orbit and cannot be bypassed by touch
picking.

*Restored pixels, which the semantic observer cannot judge.* Using the same post-migration scenario,
compare the canvas immediately before the quick-save and after reload: the timeline marker/date/era,
the lowest-ID sapiens band's last-turn food labels and actuals after reselecting it, and the
revealed-versus-hidden veil boundary must agree. This is a compact visual confirmation only; step 11
already proves exact `E2ESummary` equality through the real IndexedDB load path.

*Browser policy, which needs a real gesture.* Audio must start only after the first click, never
before. `Ctrl+S`/`Cmd+S` must quick-save without opening the browser's own Save Page dialog.

*Multi-tab behavior, which needs two real tabs.* Open a second tab and confirm its save and delete
controls stay read-only while the first holds the `africa2ice/saves` lease, then become writable
after the first closes and the second retries acquisition. Hide Field Notes, reload, and confirm the
separate `africa2ice-ui` record restores that state without opening, modifying, or locking a
campaign slot. Trigger a database-version upgrade in a development fixture and confirm the old tab
closes its connection and shows the reload message rather than blocking indefinitely.

*Judgement rather than assertion.* Gameplay must stay responsive while a save is in flight, and a
failed or aborted transaction must leave the previous slot usable and the campaign playable.

---

## 14. Out of scope for this build

Direct combat, diplomacy, player control of archaic bands, alternate archaic policies or difficulty
levels, persistent camps or shelter inventories, guard rosters or separate security/hygiene workforce
roles, sanitation stocks or tracked infections/epidemics, individual cave ownership/capacity/occupancy,
separately simulated animal-species populations, selective prey depletion/extinction or replacement,
species-wide research pools, species-specific technology graphs (DAGs), detailed Neanderthal/Denisovan
subspecies mechanics beyond the authored representative anchors, individual genomes/pedigrees/sex-linked inheritance, background random genetic
drift, targeted trait-by-trait breeding, the sickle-cell `HbS` allele with its Hardy-Weinberg
genotype shares, malaria-protection/anemia tradeoff, and region × biome `FalciparumPressure` table,
any other genotype-decomposed trait, additional named loci beyond the six-value catalog,
cloud-, ozone-, surface-reflection-, time-of-day-, or behavior-specific UV modifiers and
hemisphere-aware solar declination,
rain shadow and the prevailing-wind field it would require, lake-effect moisture, monsoon geography
as a fixed field rather than the precession term,
signed regional aridity weights and the glacial pluvials and interhemispheric-seesaw wetting they
would express, any coupling between the seven Greenland Interstadial pulses and moisture, any
wet/dry season, a non-monotonic MIS 4 moisture excursion, a seventh transitional biome between
savanna and semi-arid desert (`BaselineKCurve(V)` now covers that gradient within each band), any domain rule that branches on `ClimateEpoch`,
permanent ecological scarring or land-restoration technology, the full
~30-technology graph (DAG), continental North America beyond western Alaska, streaming
ambient audio beds, remote log collection or upload, a crash-reporting service, a player-facing log
viewer, user-configurable log levels, temporary line-of-sight/espionage fog, randomized scouting, textured and sprite
art, and glTF asset loading (Tetra3D supports it; nothing
here needs it), native installers, code signing/notarization, package-manager feeds, and automatic
desktop updates. V1 still publishes the explicitly documented unsigned portable archives in step 13;
these exclusions prevent those archives from being mistaken for signed installer packages.

---

## Appendix A — Environment facts (verified 2026-08-29)

- Go 1.26.4, darwin/arm64.
- Go 1.26.4's `lib/wasm/wasm_exec.js` implements stdout/stderr writes by buffering to a newline and
  calling `console.log`, so a newline-delimited `slog.JSONHandler` on `os.Stdout` reaches the browser's
  JavaScript console without a `syscall/js` bridge.
- Go 1.26.4's `crypto/rand.Read` documents that it fills the buffer or irrecoverably crashes on a
  reader error. The session-ID fallback therefore uses `io.ReadFull(crypto/rand.Reader, b)` directly
  rather than calling `crypto/rand.Read`; the default js/wasm reader uses Web Crypto.
- **Ebitengine v2.9.10.** The v2.9 API differs from older v2 releases in ways this design depends
  on: `text/v2` replaces the old `text` package (`Draw`, `Measure`, `NewGoTextFaceSource`), and
  `vector` provides `DrawFilledRect` / `StrokeRect` / `DrawFilledCircle`.
- Ebitengine's [`LayoutF`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2@v2.9.10#LayoutFer)
  receives outside dimensions in device-independent pixels and returns the game's logical screen
  dimensions; its actual image dimensions round up. The current monitor exposes
  [`DeviceScaleFactor`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2@v2.9.10#MonitorType.DeviceScaleFactor),
  and [`CursorPosition`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2@v2.9.10#CursorPosition)
  already accounts for the returned screen scale. Browser HiDPI is enabled by the default
  [`RunGameOptions.DisableHiDPI == false`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2@v2.9.10#RunGameOptions).
- **Tetra3D v0.18.0** (released 2026-07-11) requires Ebitengine v2.9.7, satisfied by v2.9.10 — no
  version conflict. API verified: `NewCamera(name, w, h)`, `camera.RenderScene(scene)`,
  `camera.ColorTexture() *ebiten.Image`, `NewMesh` / `AddMeshPart` / `AddVertices` / `NewVertex`,
  `NewModel`, `Mesh.UpdateBounds`, `NewBoundingTriangles`, `camera.MouseRayTest`,
  `MouseRayTestOptions.TestAgainst`, `Material.Shadeless`, and `VertexColorChannel` +
  `VertexActiveColorChannel`. Its v0.18 display index list is `uint16`, so this design deliberately
  keeps every mesh part below 21,845 triangles.
- **golangci-lint v2.12.2**, v2 config format. depguard semantics confirmed: `deny` entries are
  prefix matches unless suffixed `$`; `files` globs must be prefixed `**/`; `list-mode: lax` means
  "allowed unless denied," while `strict` denies every import not explicitly allowed.
- **`rand.PCG` from `math/rand/v2`** implements `MarshalBinary` and `UnmarshalBinary`; `rand.Rand`
  from `math/rand` does not. The save design depends on the PCG source's wire state, not reflection
  over `Rand` internals.
- `js/wasm` is a supported build target. `wasm_exec.js` is at
  `$(go env GOROOT)/lib/wasm/wasm_exec.js`.
- GitHub's [custom Pages workflow documentation](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)
  specifies `configure-pages`, `upload-pages-artifact`, and `deploy-pages`; the deployment job needs
  `pages: write` and `id-token: write` and normally targets the `github-pages` environment. The
  official workflow versions current on the verification date are `configure-pages@v5`,
  `upload-pages-artifact@v4`, and `deploy-pages@v4`; §10 strengthens those examples by requiring
  immutable full-commit pins in the implemented workflow.
- Browser save semantics are grounded in the [Indexed Database API transaction
  lifecycle](https://w3c.github.io/IndexedDB/#transaction-lifecycle), which requires rollback on
  abort and atomic commit, and the [Web Locks specification](https://w3c.github.io/web-locks/),
  whose exclusive mode serializes same-named work across cooperating tabs/workers in the same
  storage bucket.
- **Zero binary assets are required.** `golang.org/x/image/font/gofont/goregular` ships TTF bytes
  as a Go variable and is already a transitive Ebitengine dependency.

---

## Appendix B — Initial resource-balance configuration

**Status: selected initial playtest values, pending balance calibration.** The
mechanics in §7 and versioning in §9 remain authoritative; this appendix is their compact balance
view, not a second runtime source. The values are gameplay parameters, not historical measurements.
They add no runtime field, resource pool, algorithm version, or regional capacity multiplier.

### B.1 Reading the targets

The selected reference capacities are 300 normalized flora units, 300 normalized fauna units, and
250 WU. The selected foraging and ordinary-hunting reference rates are each 2.5 stock units per
applicable worker per turn. A capacity or rate index of `1.00` means that reference value. These
indices are the authored tables used by §7, not extra multipliers stacked onto another biome table.
Flora and fauna remain separate stocks, and their capacity indices cannot be added to each other;
however, an actually allocated unit from either stock shares the selected `1.0 FU` baseline.
Rate indices describe potential collection, never guaranteed harvest or FU.

Flora in these tables uses the §7 definition: plant food people can forage, not all vegetation.
Low flora means little forageable plant food, not necessarily little vegetation for herbivores.
The flora definition and food unit are settled: baseline consumption is one FU per person per turn.
Storage capacity is likewise selected as current population times `FoodStorageTurns`, with excess
food discarded at turn end after consumption and all population changes. Fixed-percentage spoilage
applies to carried-over food at phase-1 start. Phase 3 automatically meets the band's requirement
from actual fresh harvest plus the surviving reserve, carrying any remainder to the final cap.
The storage constants are **Initial** in Appendix C, alongside the nutrition coefficients whose
rules §7 defines. They remain subject to calibration with the resource values here.
Absolute capacities, reference collection rates, initial-stock fractions, seed-perturbation
parameters, and source-specific heritable modifiers are owned by §7 and Appendix C. Water retains its
existing separate stock measured in WU, with baseline demand of one WU per person per game turn
and a gradual `[1.0, 1.3]` multiplier driven by current local heat, including seasonal and long-term
temperature changes. That range and `WaterHealthLossRate = 0.40` are approved initial playtest
settings, with a 20°C baseline, a 35°C cap, and a `0.02`-per-degree linear ramp. The absolute
temperature coefficients and water-capacity table are selected below.

### B.2 Biome × season resource targets

The six land-biome labels follow §7's classifier. Each capacity tuple is ordered
`SeasonWarm / SeasonCooling / SeasonCold / SeasonWarming`, using the existing four-season gameplay
cycle, not calendar months or a new wet/dry-season model. §7's long-term moisture offset does not
add a wet/dry season either, but it does change which biome a tile occupies over the campaign, and
it oscillates. These per-biome indices were selected against a temperature-only biome trajectory.
They are not themselves invalidated by moisture — a savanna tile is still a savanna tile — but three
things about their _use_ are, and B.4's moisture gate must settle all three before these values can
be called calibrated. That gate cannot run until build step 4 supplies `ClassifyBiome`'s thresholds
and the rasterized base-moisture grid; until then these rows remain **Initial**, unrecalibrated, and
the campaign's aggregate resource envelope is unverified.

First, **the mix shifts.** Aggregate resource availability is the tile-turn-weighted sum over
whichever biomes tiles actually occupy. Moisture moves that mix toward Semi-Arid Desert without
changing a single row here, so the campaign gets poorer even at fixed indices. The recalibration
target is the aggregate envelope across the 401 turns, not any individual row.

Second, **Semi-Arid Desert changes role.** Its row was selected when desert was a marginal biome
bands crossed. Under a drying trend it becomes a biome bands must live in for extended stretches,
and a tuple tuned for transit is not automatically survivable for residency. It is the row most
likely to need to move, and it must be revalidated as a residency biome.

Third, **oscillation interacts with regeneration.** When a tile reclassifies to a poorer biome its
stock is immediately over the new cap; when it reclassifies back, stock must regrow from a low base
at the gap-recovery fractions below. Fauna at `0.15` needs about 14 turns to close 90% of a gap, so
a tile that flips faster than that is systematically poorer than either steady state would be. §7's
`MinBiomeDwellTurns = 15` exists to make that impossible. Because it is derived from the fauna
regeneration fraction, the balance pass must recompute it whenever that input changes and recheck
the trajectory; the dwell floor is never tuned independently. Values are for an undegraded tile in the
named current biome. Multiply the reference capacity by the table index, then by
`1 - Degradation` exactly once, as in §7.
Climate still determines the biome; these rows do not paint a fixed biome onto any region.

The selected regeneration fractions are stock-specific: flora `0.30`, fauna `0.15`, and water
`0.50` in every row and season. They mean the fraction of the gap to the current cap recovered per
game turn, not a historical annual growth rate. The foraging-rate index is biome-specific and
identical across all four seasons; seasonal availability acts through caps, not a second
collection-rate penalty.

| Current biome         | Flora cap indices           | Fauna cap indices           | Water cap indices           | Gap recovery (F/A/W) | Foraging index |
| --------------------- | --------------------------- | --------------------------- | --------------------------- | -------------------- | -------------: |
| Savanna               | `1.00 / 0.85 / 0.65 / 0.85` | `1.00 / 0.95 / 0.80 / 0.95` | `1.00 / 0.85 / 0.65 / 0.85` | `0.30 / 0.15 / 0.50` |         `1.00` |
| Riverine Woodland     | `1.50 / 1.30 / 1.00 / 1.30` | `1.20 / 1.10 / 1.00 / 1.10` | `2.00 / 1.80 / 1.50 / 1.80` | `0.30 / 0.15 / 0.50` |         `1.20` |
| Coastal Shrubland     | `0.80 / 0.70 / 0.55 / 0.70` | `1.10 / 1.00 / 0.90 / 1.00` | `1.20 / 1.10 / 0.90 / 1.10` | `0.30 / 0.15 / 0.50` |         `0.75` |
| Semi-Arid Desert      | `0.25 / 0.20 / 0.15 / 0.20` | `0.35 / 0.30 / 0.25 / 0.30` | `0.30 / 0.25 / 0.20 / 0.25` | `0.30 / 0.15 / 0.50` |         `0.40` |
| Mountainous Highlands | `0.50 / 0.35 / 0.20 / 0.35` | `0.65 / 0.55 / 0.45 / 0.55` | `0.90 / 0.80 / 0.70 / 0.80` | `0.30 / 0.15 / 0.50` |         `0.60` |
| Glacial Tundra        | `0.20 / 0.10 / 0.05 / 0.10` | `1.20 / 1.10 / 0.90 / 1.10` | `0.70 / 0.65 / 0.55 / 0.65` | `0.30 / 0.15 / 0.50` |         `0.30` |

These are resource targets, not `BaselineK` values. In particular, §7's low tundra carrying
capacity need not imply an equally low animal-food stock. Actual survival still depends on food
conversion, collection access, water, capacity, technology, and hazards. Water tiles, uninhabitable
tiles, and unavailable resources retain the explicit zero-cap rules rather than inheriting a
positive entry from this table. The positive land-biome entries do not grant habitability.

### B.3 Region hunting indices and representative profile audit

The §7 baseline row and its five exceptions resolve all 78 region/biome pairs. This compact table shows each region's
selected hunting index and one representative mapped profile; it is an audit view, never a fallback
for another biome. A tile always takes capacity and foraging inputs from its **current biome** in
B.2 and its fauna archetype from the complete §7 lookup.

| Region             | Representative current biome | Profile ID | Hunting index | Megafauna supported |
| ------------------ | ---------------------------- | ---------- | ------------: | ------------------- |
| East Africa        | Savanna                      | `OM`       |        `1.00` | Yes                 |
| Rest of Africa     | Savanna                      | `OM`       |        `1.00` | Yes                 |
| Arabia             | Semi-Arid Desert             | `AS`       |        `0.65` | No                  |
| Levant             | Riverine Woodland            | `WR`       |        `0.95` | Yes                 |
| Frangistan         | Glacial Tundra               | `CS`       |        `1.15` | Yes                 |
| Central Asia       | Mountainous Highlands        | `HM`       |        `0.80` | Yes                 |
| South Asia         | Riverine Woodland            | `WR`       |        `1.05` | Yes                 |
| Southeast Asia     | Riverine Woodland            | `WR`       |        `0.85` | Yes                 |
| East Asia          | Riverine Woodland            | `WR`       |        `0.95` | Yes                 |
| Yellow River Basin | Riverine Woodland            | `WR`       |        `1.00` | Yes                 |
| Sahul              | Coastal Shrubland            | `TI`       |        `0.90` | Yes                 |
| Siberia            | Glacial Tundra               | `CS`       |        `1.25` | Yes                 |
| Beringia           | Glacial Tundra               | `BC`       |        `1.15` | Yes                 |

For example, an undegraded Siberian Glacial Tundra tile in `SeasonCold` has caps of
`300 * 0.05 = 15` flora units, `300 * 0.90 = 270` fauna units, and
`250 * 0.55 = 137.5` WU. From zero stock before extraction, regeneration restores `4.5` flora,
`40.5` fauna, and `68.75` WU. Frangistan uses the same `CS` cold-steppe weights with hunting index
`1.15` rather than Siberia's `1.25`; Beringia uses the distinct `BC` mix at index `1.15`. These are
within-resource and within-role comparisons, not guaranteed usable FU or survival.

The cold-region sketch is motivated by evidence of Late Pleistocene Arctic steppe–tundra vegetation
and large herbivores, not an assumption that all cold terrain was barren. That evidence does not
quantify human-forageable food or supply any of the numerical indices above. See Wang et al.,
[Late Quaternary dynamics of Arctic biota from ancient environmental genomics](https://www.nature.com/articles/s41586-021-04016-x)
(2021). The selected archetypes remain broad gameplay abstractions subject to scientific review;
they do not add time-varying species extinction, separately depleted prey, or a new biome.

### B.4 Validation and calibration gates

1. Retain the selected baseline food requirement of one FU per person per game turn, start-of-turn
   spoilage of carried-over food, automatic consumption from actual harvest plus surviving reserves,
   and end-of-turn overflow discard after consumption and all population changes. Retain
   `FoodDeficitFraction` as the proportional shortage measure. Retain the normalized flora/fauna
   harvest units and three `1.0 FU` baseline conversion factors, and validate the locked
   `lat-elev-offset-v1` latitude table and
   `28.0`/`-12.0`/`6.5` coefficients in build step 4. Retain the one-WU baseline water requirement
   per person, modified by current local temperature before allocation. Retain persistent `Health float64` in `[0, 1]`,
   displayed as 0–100% condition, with every new-game band initialized to `1.0`. Retain the signed
   nutritional contribution using Appendix C's `HealthLossRate` and `HealthRecoveryRate`.
   Subtract linear unmet-water damage and endemic-disease damage before one phase-3 clamp, then
   derive chronic vulnerability `1 + (1 - Health)`. Selected outbreaks damage survivors' health
   once in phase 5 without replaying phase 3. Direct water/disease mortality remains separate.
   Retain initial `WaterHealthLossRate = 0.40` and the gradual `[1.0, 1.3]` demand multiplier
   driven by current local temperature, including seasonal/long-term changes, with the selected
   20°C/35°C anchors and `0.02`-per-degree linear ramp. Retain separate camp and non-camp endemic
   health rates with component-specific technology, hygiene on the camp term only, no cave bonus,
   and multiplicative remaining-risk composition before summing. Retain initial outbreak-health
   bounds `0.05` and `0.15`, interpolated linearly using the same raw severity draw as direct deaths,
   with only explicit health-technology/genetic mitigation and no camp/cave reduction of the hit. Retain the
   selected six-biome endemic health-rate table. Validate the shared `lat-elev-offset-v1` latitude
   table and locked coefficients without changing them, and retain the disease
   health-technology effects; the heritable-effect coverage table remains separate.
   Retain the squared starvation response using the post-reserve deficit fraction, applied
   directly without a health threshold or multiplier, using Appendix C's `StarvationCoefficient`.
   Scale only positive logistic growth by `1 - FoodDeficitFraction`; leave non-positive growth
   unscaled by the fed fraction, its magnitude being governed separately by
   `MaxCrowdingDeclineFraction`. Start from Appendix C's storage constants without changing
   the population-scaled capacity, spoilage model, or checkpoints. Those are approved playtest
   defaults, not final calibration. Validate the selected absolute capacities, collection rates,
   starting-stock fractions, and seed-perturbation parameters. Keep the
   campaign's compressed time scale distinct from literal monthly food budgets.
2. Validate the selected indices and regeneration fractions. Require every valid region/biome
   mapping, the eight fixed prey-group vectors, role support, and acquired-technology
   access/modifier tables; the representative rows above must not become a default for missing pairs.
   Retain the step-5 hunting-risk activation/event-kind mapping and validate its selected coefficient vectors.
3. Validate finite non-negative capacities and rates, positive regeneration fractions no greater than
   one, valid empty profiles, and the existing megafauna-versus-ordinary-terrestrial rate ordering. Keep one fauna
   stock, one application of degradation, and no regional capacity multiplier or seasonal yield
   penalty. Do not interpret archetype names as extra resources or runtime rules.
4. Exercise the existing conservation, zero-stock recovery, climate/season transition, contention,
   access, and save/reload tests with the approved tables. Balance fixtures should compare actual
   usable food per worker rather than stock indices, and include the selected low-forage/cold hunting
   case without guaranteeing survival or removing hunting danger. Run the §12 destination-policy
   and seed-corpus checks before calling the values calibrated.
   **Moisture gate.** These tables cannot be called calibrated until they are revalidated against the
   moisture-driven biome trajectory rather than the temperature-only one. This gate requires build
   step 4's `ClassifyBiome` thresholds and rasterized base moisture and cannot run before them.
   Report the tile-turn-weighted biome mix at turns 0, 100, 200, 300, and 400 and the aggregate
   flora/fauna/water envelope each mix implies, so the campaign's resource trajectory is a reviewed
   number rather than an emergent surprise. Revalidate the Semi-Arid Desert row as a residency biome:
   a band resident in desert through a full precession half-cycle must have a survivable but
   pressured food and water budget at the selected indices, without that outcome being guaranteed.
   Recompute the derived `MinBiomeDwellTurns` from any accepted fauna-regeneration change, confirm it
   still exceeds the fauna 90%-gap-closure time, and verify that no oscillating tile ends a half-cycle further from its cap than a
   steady tile of the same biome. Because §13's three reference-run margins are Policy
   (tighten-only), any failure here must be answered in the moisture model — amplitude, weights,
   dwell floor, or churn cap — or in these indices, never by relaxing a margin.
5. Implement the accepted tables in the owning resource, foraging, hunting, megafauna, and fauna
   configuration contracts, plus the food-storage contract for its limit, overflow, and spoilage
   rules; update any affected versions and explicit save migrations after release.
   The appendix is not a second configuration source, and its indices are not extra serialized state.

---

## Appendix C — Configuration manifest

**This appendix is the authoritative manifest for release configuration and structural prerequisites.**
For a scalar row, the Value column records the current selected value or derivation. For a composite
table, catalog, enum, or policy, it records the authoritative owning section or artifact until the
entry is closed. §§6–13 remain authoritative for behavior and may repeat a current manifest value
when a rule, worked example, derived bound, test expectation, or wire expectation needs it. Algorithm
and schema version identifiers remain in their owning contracts; they are dependencies the C.14
checklist may require changing, not duplicate manifest entries. All repeated values must agree with
this manifest; the manifest does not pretend they disappear.

This exists because a value restated across its rule, fixtures, §9's wire contract, §12's build step,
and §13's verification takes several consistent edits to change, and drift is silent when one is
missed. Appendix B is the compact balance-table view referenced by the owning §7 contracts and this
manifest; it is not a second runtime configuration source. An Initial value remains subject to the
§12 balance pass even though its design selection is closed.

### C.1 Status vocabulary

| Status      | Meaning                                                                                                                                                                                                                                                                                                                                                                                                       |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Locked**  | Structural or policy. Not a balance knob; changing it requires the owning contract's version/change process and any applicable migration, not a tuning decision.                                                                                                                                                                                                                                              |
| **Initial** | Approved initial playtest default. Real and implementable now; §12's balance pass may retune it before release.                                                                                                                                                                                                                                                                                               |
| **Policy**  | Approved default or threshold that its owning UI, platform, persistence, or verification contract validates, rather than one the simulation seed corpus tunes. A verification threshold is asserted _against_ runs, not derived _from_ them, so it is not a balance knob even when it is stated in simulation terms. Some Policy rows carry a direction rule; where one exists it is recorded beside the row. |
| **Step N**  | Selected and locked during the named build step, ahead of the balance pass, because it defines public structure or because the consuming step's own assertions require the real value. §12 validates without retuning.                                                                                                                                                                                        |
| **Derived** | Computed only from other manifest entries. Never tuned independently; readiness and lifecycle follow its inputs.                                                                                                                                                                                                                                                                                              |
| **Open**    | Not yet selected. Every Open row is a §12 gate; its shape and validation predicate must close before its consuming build step under §12's structure-before-values rule.                                                                                                                                                                                                                                       |

An Open row is not a licence to implement around the gap. Before its consumer is implemented, the
owning section or paired Step N entry closes the value's type, dimensions, bounds, and validation;
only the release value remains Open. An Open row with no such structural owner is a manifest defect.
Earlier fixtures use explicit values that are never release data.

### C.2 Structural bounds and closed counts

| Constant                                 | Value                      | Status  | Owning contract                |
| ---------------------------------------- | -------------------------- | ------- | ------------------------------ |
| `MaxBands`                               | `256`                      | Locked  | `BandAlgorithm`                |
| `MaxArchaicBands`                        | `96`                       | Initial | `BandAlgorithm`                |
| `AssignmentCount`                        | `5`                        | Locked  | `AssignmentAlgorithm`          |
| `AllocationBasisPoints`                  | `10_000`                   | Locked  | `AssignmentAlgorithm`          |
| `TechCount`                              | `9`                        | Locked  | `TechnologyOwnershipAlgorithm` |
| `HeritableTraitCount`                    | `6`                        | Locked  | `HeritableStateAlgorithm`      |
| `ShelterProtectionClassCount`            | `4`                        | Locked  | `ShelterAlgorithm`             |
| `MaxGridNeighbors`                       | `8`                        | Locked  | `MovementAlgorithm`            |
| `PassageCount`                           | `3`                        | Locked  | `PassageAlgorithm`             |
| `MaxPassageEdgesPerTile`                 | `2`                        | Locked  | `PassageAlgorithm`             |
| `MaxEvents`                              | `128`                      | Locked  | world event feed               |
| `MaxClimatePulses`                       | `32`                       | Locked  | `ClimateAlgorithm`             |
| `MaxMacroEpisodes`                       | `16`                       | Locked  | `MacroEventAlgorithm`          |
| `FaunaGroup` catalog / `FaunaGroupCount` | §7 six-entry catalog / `6` | Locked  | `FaunaProfileAlgorithm`        |
| `FoodSourceCount`                        | `3`                        | Locked  | `FoodConversionAlgorithm`      |
| `RegionCount`                            | `13`                       | Locked  | `GeographyAlgorithm`           |
| `BiomeCount`                             | `6` land biomes            | Locked  | `GeographyAlgorithm`           |
| New-world PCG seed expansion             | two `SplitMix64(state)` applications from `state = seed`; §7 pins the corpus expansions | Locked | `RNGAlgorithm` |
| `SplitMix64` constants                   | `0x9e3779b97f4a7c15` / `0xbf58476d1ce4e5b9` / `0x94d049bb133111eb`; shifts `30` / `27` / `31` | Locked | `RNGAlgorithm` |
| `WorldRNG.Float64` mapping               | `float64(float64(pcg.Uint64()>>11) * 0x1p-53)`; project-owned, no `*rand.Rand` | Locked | `RNGAlgorithm` |

### C.3 Geography and campaign clock

| Constant                                    | Value                                                                          | Status  | Owning contract               |
| ------------------------------------------- | ------------------------------------------------------------------------------ | ------- | ----------------------------- |
| Grid dimensions                             | `96 × 64` = `6_144` tiles                                                      | Locked  | `GeographyAlgorithm`          |
| Longitude span                              | `20°W` at `x=0` to `200°E` at `x=95`                                           | Locked  | `GeographyAlgorithm`          |
| Latitude span                               | `72°N` at `y=0` to `48°S` at `y=63`                                            | Locked  | `GeographyAlgorithm`          |
| `PolygonCoordinateScale`                    | `10` signed integer units per degree                                             | Locked  | `GeographyAlgorithm` + `MacroEventAlgorithm` |
| Land/water/region/highland/river catalogs   | §6 exact coordinate tables and raster rules; checked-in independent checksums     | Locked  | `GeographyAlgorithm`          |
| `ZonalMoisture` table                       | `64` checked-in exact values from §6's eight anchors                           | Step 3  | `GeographyAlgorithm`          |
| `RiverCorridorBonus` / `RiverAdjacentBonus` | `0.35` / `0.20`                                                                | Initial | `GeographyAlgorithm`          |
| `ContinentalityMax` / `ContinentalityRange` | `0.20` / `8` tiles                                                             | Initial | `GeographyAlgorithm`          |
| `OrographicBonus`                           | `0.10`                                                                         | Initial | `GeographyAlgorithm`          |
| Authored elevation catalog                  | §6 ten height-valued highland polygons; `1.25–3.0 km`, maximum on overlap       | Initial | `GeographyAlgorithm`          |
| `HighlandElevationKm`                       | `1.0 km`; strict `ElevationKm > HighlandElevationKm` classification             | Initial | `GeographyAlgorithm`          |
| `ExplorationWordCount`                      | `6_144 / 64 = 96`                                                              | Derived | `ExplorationAlgorithm`        |
| Campaign length                             | `400` turn transitions                                                         | Locked  | `CampaignClockAlgorithm`      |
| Era boundaries                              | turns `100`, `200`, `300`                                                      | Locked  | `CampaignClockAlgorithm`      |
| Years per turn, by era                      | `300` / `150` / `100` / `50`                                                   | Locked  | `CampaignClockAlgorithm`      |
| Campaign date range                         | 80,000 BP to 20,000 BP                                                         | Locked  | `CampaignClockAlgorithm`      |
| Seasonal cycle length                       | `12` turns, four three-turn phases                                             | Locked  | `ClimateAlgorithm`            |
| Natural-shelter coverage and rating catalog | §6 nine-entry authored ellipse catalog; ratings `0` / `0.25` / `0.50` / `1.00` | Initial | `NaturalShelterMaskAlgorithm` |

### C.4 Food and storage — `FoodStorageAlgorithm`, `FoodConversionAlgorithm`

| Constant                                            | Value                                                       | Status  |
| --------------------------------------------------- | ----------------------------------------------------------- | ------- |
| `FoodStorageTurns`                                  | `3`                                                         | Initial |
| `FoodSpoilageRate`                                  | `0.10`                                                      | Initial |
| Baseline food requirement                           | `1` FU per person per turn                                  | Locked  |
| Flora/fauna stock-unit and food-conversion contract | §7 normalized harvest conversion and extraction rules       | Locked  |
| Flora/fauna stock-unit scales                       | normalized harvest units with equal baseline edible yield   | Locked  |
| Flora → FU base conversion factor                   | `1.0`                                                       | Locked  |
| Fauna → FU base conversion factors                  | `1.0` each for `Animal` and `Aquatic`                       | Locked  |
| New-game `StoredFood`                               | `0` FU for every starting band                               | Locked  |
| Partial-meal source attribution                     | §7 four-component pro-rata transient accounting              | Locked  |

### C.5 Health, hazard, and demographics — `HazardAlgorithm`

| Constant                              | Value                              | Status  |
| ------------------------------------- | ---------------------------------- | ------- |
| `HealthLossRate`                      | `0.20`                             | Initial |
| `HealthRecoveryRate`                  | `0.05`                             | Initial |
| `WaterHealthLossRate`                 | `0.40`                             | Initial |
| `StarvationCoefficient`               | `0.10`                             | Initial |
| `MinOutbreakHealthLoss`               | `0.05`                             | Initial |
| `MaxOutbreakHealthLoss`               | `0.15`                             | Initial |
| `MaxChronicRate`                      | `0.25`                             | Initial |
| `SeasonalMortalityScale`              | `0.10`                             | Initial |
| `ChronicMortalityScale`               | `0.10`                             | Initial |
| `AcuteProbabilityScale`               | `0.10`                             | Initial |
| `KinContactHalfSaturation`            | `1.0`                              | Initial |
| `MaxKinAcuteReduction`                | `0.40`                             | Initial |
| Endemic camp/non-camp health rates    | six-biome table in §7              | Initial |
| Health bounds and new-game value      | `[0, 1]`, new game `1.0`           | Locked  |
| `HealthVulnerability` mapping         | `1 + (1 - Health)`, range `[1, 2]` | Locked  |
| `MaxAcuteProbability`                 | `0.25`                             | Initial |
| `MinAcuteLoss[k]` / `MaxAcuteLoss[k]` | §7 five-kind acute-severity table  | Initial |
| `r` — logistic growth coefficient     | `0.020` per game turn              | Initial |
| `MaxCrowdingDeclineFraction`          | `0.25`                             | Initial |
| Seasonal and chronic risk profiles    | §7 six-biome risk-profile tables   | Initial |
| Technology mitigation effect tables   | §7 channel-specific mitigation table | Initial |

### C.6 Water and temperature — `ResourceAlgorithm`, `TemperatureAlgorithm`

| Constant                      | Value                                      | Status  |
| ----------------------------- | ------------------------------------------ | ------- |
| `BaseWaterPerPerson`          | `1.0` WU                                   | Locked  |
| Water demand slope            | `0.02` per °C                              | Initial |
| Water demand anchors          | `20 °C` baseline, `35 °C` cap              | Initial |
| Water demand multiplier range | `[1.0, 1 + 0.02 · (35 − 20)] = [1.0, 1.3]` | Derived |
| `AridClimateAdaptation` heat reduction | `0.40` of the heat increment at trait value `1` | Initial |
| `EquatorTempC`                | `28.0 °C`                                  | Locked  |
| `PolarTempC`                  | `-12.0 °C`                                 | Locked  |
| `LapseRateCPerKm`             | `6.5 °C/km`                                | Locked  |
| `ElevationKm` input           | §6 / C.3 authored `0–3.0 km` geography     | Derived |
| `LatitudeSinSquared[64]`      | §7 checked-in exact bit-pattern table      | Locked  |

### C.7 Ecology and resources — `ResourceAlgorithm`, `MovementCostAlgorithm`

| Constant                                                    | Value                                                                      | Status  |
| ----------------------------------------------------------- | -------------------------------------------------------------------------- | ------- |
| `MaxDegradation`                                            | `0.75`                                                                     | Locked  |
| Degradation `damage_rate`                                   | `0.06`                                                                     | Initial |
| Degradation `recovery_rate`                                 | `0.03`                                                                     | Initial |
| `BaselineKCurve(V)` knot table                              | §7 eight-knot, within-band piecewise-linear curve with one `0.45` step     | Initial |
| `CanopyClosureV`                                            | `0.90`; shared knot owned jointly by `ResourceAlgorithm` and `MovementCostAlgorithm`; changing it moves both identifiers | Initial |
| `BiomeCapacityFactor[CoastalShrubland, MountainousHighlands]` | `1.15` / `0.65`                                                         | Initial |
| `BiomeCapacityFactor[vegetation-classified biomes]`          | exactly `1.00` for all four entries; not tunable                         | Locked  |
| `VegetationColdCutoffC` / `VegetationWarmthC`               | `-5.0` / `10.0`                                                            | Initial |
| `TundraSplitTempC`                                          | `0.0`                                                                      | Initial |
| `V` biome thresholds                                        | `0.30` / `0.45` / `0.75`                                                   | Initial |
| Reference resource capacities                               | flora `300`, fauna `300`, water `250`                                      | Initial |
| `ResourceCapIndex_s`, per biome × season                    | Appendix B six-by-four F/A/W table                                         | Initial |
| `regen_rate_s`, per stock                                   | flora `0.30`, fauna `0.15`, water `0.50`                                   | Initial |
| Initial-stock initialization and seed-perturbation contract | §7 two-scale, counter-based rule                                           | Locked  |
| Initial resource-stock fractions, per biome × stock         | §7 “Initial resource abundance” six-by-three table                         | Initial |
| Initial-stock seed-perturbation parameters                  | region `0.08`, tile `0.04`                                                 | Initial |
| `ForagingRate`, per biome × technology                      | §7 “Linear foraging and shared flora” biome rates and `PlantKnowledge ×1.30` | Initial |
| `HuntingRate` / `MegafaunaRate` tables                      | §7 “Linear hunting and shared fauna” reference/access/technology tables and “Linear megafauna tracking” `1.75×` multiplier | Initial |
| Region × biome fauna profiles and regional hunting indices  | §7 “Regional fauna profiles” eight archetypes, baseline row + five regional exceptions, and 13 indices | Initial |

### C.8 Workforce and shelter — `ShelterAlgorithm`, `HuntingRiskAlgorithm`

| Constant                                                                 | Value                                                          | Status  |
| ------------------------------------------------------------------------ | -------------------------------------------------------------- | ------- |
| `SapiensInitialAllocationBP`                                             | `3500 / 3000 / 1500 / 500 / 1500` in assignment order         | Initial |
| `MaxShelterMitigation`                                                   | `0.60`                                                         | Initial |
| `ShelterHalfSaturation`                                                  | `0.25`                                                         | Initial |
| `NaturalShelterEfficiencyBonus`                                          | `1.00`                                                         | Initial |
| `CampSecurityScale`                                                      | `0.75`                                                         | Initial |
| `CampHygieneScale`                                                       | `0.40`                                                         | Initial |
| `WorkRiskCoeff` vectors                                                  | §7 selected five-kind Hunting/Megafauna table                  | Initial |
| Hunting/megafauna work-risk activation and event-kind mapping            | positive share + positive profile/technology-adjusted rate; §7 | Locked  |
| Acute biome bases, season factors, passage weights, and class partitions | §7 selected complete tables                                    | Initial |

### C.9 Technology and knowledge

| Constant                      | Value                                      | Status  | Owning contract                |
| ----------------------------- | ------------------------------------------ | ------- | ------------------------------ |
| `DiffusionRate`               | `0.20`                                     | Locked  | `KnowledgeDiffusionAlgorithm`  |
| `MaxResearchPerTurn`          | `20`                                       | Initial | `ResearchProductionAlgorithm`  |
| `ResearchHalfSaturation`      | `25` workers                               | Initial | `ResearchProductionAlgorithm`  |
| `ResearchCost[TechCount]`     | §7 selected nine-entry `70`–`150` table    | Initial | `TechnologyOwnershipAlgorithm` |
| `T_tech` capacity multipliers | §7 selected nine-entry `1.02`–`1.08` table | Initial | `TechnologyOwnershipAlgorithm` |

### C.10 Genetics

| Constant                                       | Value                                      | Status  | Owning contract             |
| ---------------------------------------------- | ------------------------------------------ | ------- | --------------------------- |
| `SameSpeciesGeneFlowRate`                      | `0.025`                                    | Initial | `GeneFlowAlgorithm`         |
| `InterbreedGeneFlowRate`                       | `0.10`                                     | Initial | `GeneFlowAlgorithm`         |
| `MaxGeneFlowPerTurn`                           | `0.20`                                     | Initial | `GeneFlowAlgorithm`         |
| `GeneticSeverityMitigation`                    | `0`; no acute severity channel in v1       | Locked  | `GeneticSelectionAlgorithm` |
| `MutationEntryFrequency[HeritableTraitCount]`  | §7 selected six-entry table                | Initial | `MutationAlgorithm`         |
| `MutationProbability[HeritableTraitCount]`     | §7 selected six-entry table                | Initial | `MutationAlgorithm`         |
| Standing variation, per species × start region | §7 selected three-profile table            | Initial | `HeritableStateAlgorithm`   |
| Trait effect and selection functions           | §7 selected six-function table             | Initial | `GeneticSelectionAlgorithm` |
| `SeasonUVFactor[SeasonCount]`                  | `1.00 / 0.85 / 0.70 / 0.85`                | Initial | `GeneticSelectionAlgorithm` |
| `UVAltitudeGainPerKm`                          | `0.10`                                     | Initial | `GeneticSelectionAlgorithm` |
| `UVExposure`                                   | §7 latitude × season × altitude derivation | Derived | `GeneticSelectionAlgorithm` |

### C.11 Climate and macro-episodes

| Constant                                                                   | Value                                             | Status  | Owning contract              |
| -------------------------------------------------------------------------- | ------------------------------------------------- | ------- | ---------------------------- |
| `BeringiaOpenFraction`                                                     | `0.85`                                            | Initial | `ClimateAlgorithm`           |
| `BeringiaOpenTemp`                                                         | `−BeringiaOpenFraction · LGM_cooling`             | Derived | `ClimateAlgorithm`           |
| `LGM_cooling`                                                              | `6.1°C`                                           | Initial | `ClimateAlgorithm`           |
| `orbital_amplitude`                                                        | `1.5°C`                                           | Initial | `ClimateAlgorithm`           |
| `seasonal_amplitude`                                                       | `2.0°C`                                           | Initial | `ClimateAlgorithm`           |
| `noise_amplitude`                                                          | `0.35°C`                                          | Initial | `ClimateAlgorithm`           |
| `MaxAbruptClimateOffset`                                                   | `2.5°C`; sampled cap across all supported turns and regions | Initial | `ClimateAlgorithm`           |
| Orbital table                                                              | `401` checked-in exact values                     | Step 4  | `ClimateAlgorithm`           |
| Seasonal table                                                             | `12` checked-in exact values                      | Step 4  | `ClimateAlgorithm`           |
| Abrupt-pulse catalog and regional weights                                  | §7 seven selected GI entries and 13-region vector; max sampled weighted pulse `2.43°C` at GI-14 turn 87 | Initial | `ClimateAlgorithm`           |
| `aridification_amplitude`                                                  | `0.35`                                            | Initial | `ClimateAlgorithm`           |
| `precession_amplitude`                                                     | `0.10`                                            | Initial | `ClimateAlgorithm`           |
| `precession_period`                                                        | `21_000` years                                    | Initial | `ClimateAlgorithm`           |
| `precession_phase`                                                         | `2.020` rad                                       | Initial | `ClimateAlgorithm`           |
| Precession table                                                           | `401` checked-in exact values                     | Step 4  | `ClimateAlgorithm`           |
| `RegionalAridityWeight`                                                    | §7 selected complete 13-region vector             | Initial | `ClimateAlgorithm`           |
| Epoch thresholds on `AridityIndex`                                         | `0.25` / `0.96`                                   | Initial | `ClimateAlgorithm`           |
| `EpochHysteresis`                                                          | `0.02`                                            | Initial | `ClimateAlgorithm`           |
| `BiomeChurnCap`                                                            | `12` biome changes per tile across 401 turns      | Initial | `ClimateAlgorithm`           |
| `MinBiomeDwellTurns`                                                       | `ceil(ln(0.10) / ln(1 − regen_rate_fauna))` = `15` at fauna `0.15` | Derived | `ClimateAlgorithm`           |
| Epoch grade anchor palettes                                                | §8 exact naturalistic three-anchor table          | Locked  | presentation/palette         |
| Toba timeline/Field Notes marker                                           | `73,880 BP`; no simulation effect or warning      | Locked  | presentation/context catalog |
| Required active v1 macro episode                                           | Campanian Ignimbrite, `39,850 BP`                 | Locked  | `MacroEventAlgorithm`        |
| `MaxMacroLoss`                                                             | `0.45`                                            | Initial | `MacroEventAlgorithm`        |
| `MaxRefugiumMitigation`                                                    | `0.20`                                            | Initial | `MacroEventAlgorithm`        |
| Campanian zone intensities, resource/habitat factors, direct/health scales | §7 selected tables                                | Initial | `MacroEventAlgorithm`        |
| Campanian epicenter, five polygons, and precedence                         | §7 exact evidence-shaped multi-lobe catalog       | Locked  | `MacroEventAlgorithm`        |
| Campanian tile masks and checksums                                         | generated from the Locked polygon catalog         | Step 4  | `MacroEventAlgorithm`        |

### C.12 Movement, achievement, and spatial policy

| Constant                                                   | Value                                                                                           | Status  | Owning contract            |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ------- | -------------------------- |
| `SplitStressThreshold`                                     | `0.67`                                                                                          | Initial | `BandAlgorithm`            |
| `MinEstablishedBand`                                       | `20`                                                                                            | Initial | `BandAlgorithm`            |
| `referenceRouteDeparturePopulation`                        | `5 × MinEstablishedBand / 2 = 50`                                                               | Derived | reference route policy     |
| `MaxPopulation`                                            | `2^32 - 1` whole people (`uint32`)                                                              | Locked  | `BandAlgorithm`            |
| New-game band populations                                  | four sapiens `120` in East Africa; one archaic `120` in the Levant; two archaic `60` in Frangistan; Denisovan-representative archaics `12` in Siberia, `12` in Southeast Asia, and `90` in East Asia | Locked  | scenario contract          |
| New-game geographic anchors                                | §6 exact ten-entry Afar-to-Harbin catalog                                                        | Locked  | scenario contract          |
| New-game starting tile IDs                                 | deterministic nearest valid tiles generated from the anchors and frozen                         | Step 4  | scenario contract          |
| Split ratio                                                | `50/50`; odd whole-person remainder stays with source                                           | Locked  | `BandAlgorithm`            |
| `MinSplitSourcePopulation`                                 | `2 × MinEstablishedBand = 40`                                                                   | Derived | `BandAlgorithm`            |
| `DestinationRegions`                                       | `{Frangistan, SouthAsia, YellowRiverBasin, Sahul, Beringia}`                                    | Locked  | campaign outcome contract  |
| Cardinal / diagonal step length                            | `1` / `math.Sqrt2`                                                                              | Locked  | `MovementAlgorithm`        |
| `MovementCurve(V)` knots                                   | §7 six-knot translation of the intent table's movement grades                                   | Initial | `MovementCostAlgorithm`    |
| `BiomeMovementFactor[CoastalShrubland, MountainousHighlands]` | `1.25` / `2.50`                                                                            | Initial | `MovementCostAlgorithm`    |
| `BiomeMovementFactor[vegetation-classified biomes]`          | exactly `1.00` for all four entries; not tunable                                            | Locked  | `MovementCostAlgorithm`    |
| `MaxMovementCost`                                          | `2.75`, selected within `math.Sqrt2 * MaxMovementCost < 4.00`                                   | Initial | `MovementCostAlgorithm`    |
| Named-passage route costs                                  | north Wallacea `4.00`; south Wallacea `4.50`; Beringia `3.00`                                   | Initial | `PassageAlgorithm`         |
| Water survival-equivalent conversion for migration ranking | `min(EcologicalK, WaterStock / EffectiveWaterDemandPerPerson)` plus §7 validation               | Locked  | migration scoring contract |
| `ArchaicAssignmentPreset[region, biome]`                   | §7 eight profile presets expanded through the resolved profile map                              | Initial | `ArchaicPolicyAlgorithm`   |
| `ArchaicTechPriority`                                      | §7 selected nine-entry order                                                                    | Initial | `ArchaicPolicyAlgorithm`   |

`SplitStressThreshold` previously appeared as the bare literal `0.9` in nine semantic places. It gates
whether `SplitBand` is legal for the player and whether the archaic policy takes a spatial action, so
it is a balance knob with no name — exactly the kind of value this manifest exists to hold. Step 5e's
whole-campaign viability pass selected `0.67`, allowing stressed populations to divide before
whole-tile crowding turns a viable outward route into synchronized decline, while leaving bands large
enough that band size carries information about the ground they stand on.

It cannot go much higher. Because band size tracks `SplitStressThreshold · BaselineK`, and most
regions have a mean `BaselineK` between 80 and 160, a threshold at or above `0.8` produces bands too
large to relocate: on arriving at any ordinary tile such a band exceeds its capacity, takes the full
`MaxCrowdingDeclineFraction` every turn, and is gone within about ten. At `0.8` the corpus collapses
to a single band of roughly twelve people per seed. The ceiling is therefore set by the map's
capacity distribution rather than by the demographic constants, and it is selected together with `r`
above.

V1 intentionally adds no founder-flow counter, `EverExitedAfrica` lineage flag, or `2,000`–`5,000`
gameplay target. The first sapiens establishment in Arabia or the Levant triggers a sourced Field
Note about founder-population and effective-population estimates and their uncertainty. Existing
route-neutral regional achievements record geographic expansion. Balance telemetry reports current
sapiens population inside versus outside the two African regions at checkpoints, but neither value
is a goal, a saved statistic, nor a victory condition.

### C.13 Presentation, platform, persistence, observability, and verification policy

These are presentation, platform, persistence, observability, and verification policies rather than
simulation balance values. Most consume no save contract; save-slot IDs are part of §9's persistence
interface, while operational-log files are diagnostic output and are never read back into the game.

The three reference-run margins are **Policy**, not Initial, because §13 asserts them against the
tuned run rather than tuning them from it. They also carry a **direction rule**: a deliberate policy
revision may only _tighten_ an effective threshold; §12 must never relax one to rescue a failing run.
For the destination deadline, tighter means an earlier turn. For band/population minima and their
multipliers, tighter means a larger effective minimum. This rule applies after resolving dependencies:
if §12 lowers `MinEstablishedBand`, it must raise the affected multipliers enough that the effective
`2 × MinEstablishedBand` and `10 × MinEstablishedBand` policy minima do not decrease from their
currently approved values. When a margin is genuinely unreachable while the campaign is still
well-formed, §13's remedy is to improve the reference policy or select a representative seed — fix
the run, not the test. Recording these as retunable balance defaults would invite exactly the edit
the rule forbids, and it is the knife-edge outcome §13 exists to prevent.

`MaxCompressedWasmBytes` carries the same direction rule for a different reason. It is the one
Policy row that began from a *dependency-skeleton* measurement rather than this game. The first full
build has now ratcheted it to `4_050_000` bytes, including the selected headroom and quantum. CI
compares the live ceiling and headroom with the trusted base revision, so the direction rule is
mechanical rather than an appeal to reviewers. A ceiling that can only fall is a growth detector;
one that may rise on demand is a number that records whatever the build happens to weigh.

| Constant                                   | Value                                                          | Status                                          | Where |
| ------------------------------------------ | -------------------------------------------------------------- | ----------------------------------------------- | ----- |
| Save slot IDs                              | Manual `1`–`3`; quick `99`; autosave `101`–`103`               | Locked                                          | §9    |
| Autosave interval fallback                 | `5` minutes of monotonic running time                          | Policy                                          | §9    |
| Toast display duration                     | `2` seconds                                                    | Policy                                          | §9    |
| Toast FIFO capacity                        | `4` entries                                                    | Policy                                          | §9    |
| `UISettings.SchemaVersion`                 | `1`; all three preference fields required                      | Locked                                          | §8    |
| `UISettings` `FieldNotesVisible` default   | `true`                                                         | Policy                                          | §8    |
| `UISettings` `MasterVolume` default        | `0.5`                                                          | Policy                                          | §8    |
| `UISettings` `MasterVolume` range          | `[0, 1]`                                                       | Locked                                          | §8    |
| `UISettings` `Muted` default               | `false`                                                        | Policy                                          | §8    |
| Go toolchain                              | `1.26.4`                                                       | Locked                                          | §10/§13 |
| Binaryen toolchain                        | `version_131`; Linux x86-64 SHA-256 `b5bf1f0eaf17c63ee588ff7a5954dc8f6ce2c26989051c66f24dfe9ece3e46db` | Locked | §10 |
| Compressed-wasm measurement                | `brotli -q 11`; raw and `gzip -9` recorded alongside, not gated | Locked                                         | §10   |
| `MaxCompressedWasmBytes`                   | `4_050_000` bytes                                              | Policy (tighten only: smaller ceiling)          | §10   |
| `CompressedWasmHeadroom`                   | `500_000` bytes above the step-11 measured release build       | Policy (tighten only: smaller headroom)         | §10   |
| Automated browser ready timeout            | `10` seconds on the optimized loopback-served bundle           | Policy                                          | §8/§10 |
| Automated browser scripted checkpoint timeout | `5` seconds each                                             | Policy                                          | §8/§10 |
| Native benchmark regression gate           | no more than `25%` normalized-time regression or bytes/op increase from reviewed calibrated baseline | Policy              | §8/§13 |
| `NativeBenchmarkReference`                 | GitHub-hosted `ubuntu-24.04`, `linux/amd64`; record reported image version | Policy                                  | §8/§13 |
| `ReferenceSeed`                            | `0x9e3779b97f4a7c15`                                          | Locked                                          | §7/§13 |
| `BalanceSeedCorpus`                        | `{0, 1, 2, 3, 0x9e3779b97f4a7c15, 0xd1b54a32d192ed03, 0x94d049bb133111eb, 0xffffffffffffffff}` | Locked | §7/§12 |
| Cross-target checkpoint gate               | identical turn `0/100/200/300/400` hashes and margins on Linux amd64, macOS arm64, Chromium js/wasm | Locked | §7/§13 |
| Native CI runner matrix                     | `ubuntu-24.04` amd64 / `macos-15` arm64 / `windows-2025` amd64; assert architecture | Policy | §10/§13 |
| Interactive reference machine/workload     | §8 Mac mini `Mac16,10`, M4/16 GB/macOS 26.6.1; exact checked-in 5 s warm-up + 30 s workload | Policy | §8/§13 |
| Reference-browser performance floors       | §8: DPR1 normal `20` median FPS; DPR1 low `30`; DPR2 normal `15`; DPR2 low `20`; p95 frame gap `<= 150 ms`; `EndTurn <= 2 s` | Policy | §8/§13 |
| GitHub Pages source                        | GitHub Actions repository-project site; no custom domain       | Locked                                          | §10   |
| GitHub Pages publish trigger               | With step-13 publication wiring present: successful push to `main` after `native` + `web-release` + `cross-target-determinism` + `release-readiness`; no PR | Locked | §10/§12 |
| Native release trigger and payload         | Annotated SemVer tag; four unsigned portable OS/architecture archives + `SHA256SUMS` | Locked                 | §12   |
| `MaxTrianglesPerPart`                      | `21_000`                                                       | Locked                                          | §8    |
| Terrain chunk dimensions                   | `32 × 32`, six chunks                                          | Locked                                          | §8    |
| Terrain-detail launch default              | Normal                                                         | Policy                                          | §8    |
| `MaxRenderScale`                           | `2.0`                                                          | Policy                                          | §8    |
| Minimum gameplay viewport / narrow breakpoint | `960 × 600 DIPs` / `1,100 DIPs`                            | Policy                                          | §8    |
| High-DPI coordinate contract               | HUD layout/hits in DIPs; scene target/picking in render pixels | Locked                                          | §8    |
| Browser `DisableHiDPI`                     | `false`                                                        | Locked                                          | §10   |
| Operational-log target sinks               | Desktop: new temp JSONL file/session; web: JS console          | Locked                                          | §3    |
| Operational `session_id`                   | 128 random bits; documented timestamp/counter fallback         | Policy                                          | §3    |
| Operational-log minimum level              | `Debug` on desktop and web                                     | Policy                                          | §3    |
| `MaxOperationalLogRecordBytes`             | `64 * 1024` bytes                                              | Policy                                          | §3    |
| `MaxDesktopSessionLogBytes`                | `32 * 1024 * 1024` bytes; truncate/mark/reuse the same file    | Policy                                          | §3    |
| Reference-run margin: first destination by | turn `350`                                                     | Policy (tighten only: earlier)                  | §13   |
| Reference-run margin: establishing band    | `2 × MinEstablishedBand`                                       | Policy (tighten only: larger effective minimum) | §13   |
| Reference-run margin: turn-400 survival    | `5` bands, `10 × MinEstablishedBand` people                    | Policy (tighten only: larger effective minima)  | §13   |

### C.14 Using this manifest in the balance pass

There are currently no **Open** rows: no release value waits until the step-12 balance pass for its
first selection. **Step N** rows intentionally remain scheduled artifacts until their named earlier
build step closes them; they are not placeholder values for step 12. Any future **Open** row is a §12
step-12 gate, and all such rows together are the complete list of release values still to select
during that pass—§12's prose names the areas to tune, but this manifest says whether selection is
finished. Every **Initial** row contains at least one independently selected simulation or balance
value that step 5e and/or §12 may retune and that §12 must validate against the seed corpus. A **Policy** row passes its owning UI,
platform, persistence, or verification contract instead, and where it carries a direction rule that
rule binds §12 as well. A **Step N** row must hold its real structure or value
before that build step completes,
and §12 validates it without changing it. A **Derived** row is recomputed from its dependencies and
is never tuned separately. **Locked** rows are out of scope for balancing; a change follows its
owning contract's version/change process, with §9 migration when saved state or persistence changes.

The manifest is release-complete when no Open row remains, every Initial row has been validated against
the seed corpus with the §13 margins satisfied, every Policy row has passed its owning verification,
and every Step N entry is closed. Before accepting any
value change, search the document and implementation for the constant name and old literal, then
update dependent formulas, examples, derived bounds, algorithm/version identifiers, fixtures, wire
expectations, and required migrations. The manifest is the change's starting point and completeness
checklist, not a claim that those dependencies update themselves.
