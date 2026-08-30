package domain

type MacroEpisode uint8

const (
	CampanianIgnimbrite MacroEpisode = iota
	MacroEpisodeCount
)

const CampanianYearBP = 39_850

type MacroZone uint8

const (
	MacroUnaffected MacroZone = iota
	MacroWide
	MacroProximal
	MacroDirect
)

type MacroImpact struct {
	Active        bool
	Zone          MacroZone
	Intensity     float64
	FloraFactor   float64
	FaunaFactor   float64
	WaterFactor   float64
	HabitatFactor float64
	LossFraction  float64
	HealthLoss    float64
}

var directCampanianPlain = []coordinate{c(12.7, 41.5), c(14.8, 42.3), c(16.5, 41.2), c(15.8, 39.8), c(12.8, 40.0)}
var proximalSouthernItaly = []coordinate{c(9.0, 44.5), c(14.0, 45.0), c(19.0, 43.0), c(20.0, 39.0), c(17.0, 36.0), c(12.0, 36.5), c(9.0, 40.0)}
var wideCampanianPolygons = [][]coordinate{
	{c(13.5, 40.0), c(18.0, 42.0), c(27.0, 40.0), c(34.0, 36.0), c(32.0, 31.0), c(22.0, 30.0), c(15.0, 34.0), c(13.0, 38.0)},
	{c(15.0, 44.0), c(20.0, 48.0), c(28.0, 49.0), c(31.0, 46.0), c(30.0, 42.0), c(25.0, 39.0), c(19.0, 39.0), c(15.0, 41.0)},
	{c(28.0, 48.0), c(33.0, 52.5), c(41.5, 53.0), c(44.0, 50.0), c(42.0, 46.0), c(35.0, 43.5), c(30.0, 45.0)},
}

func MacroEpisodeActive(turn int) bool {
	if turn <= 0 || turn > MaxCampaignTurn {
		return false
	}
	previous, previousErr := CampaignDate(turn - 1)
	current, currentErr := CampaignDate(turn)
	return previousErr == nil && currentErr == nil && previous.YearBP > CampanianYearBP && CampanianYearBP >= current.YearBP
}

func MacroEpisodeWarned(turn int) bool {
	return turn >= 0 && turn < MaxCampaignTurn && MacroEpisodeActive(turn+1)
}

func CampanianZone(id TileID) MacroZone {
	if id >= TileCount {
		return MacroUnaffected
	}
	x, y, _ := TileXY(id)
	point := scaledTileCenter(x, y)
	if polygonContains(directCampanianPlain, point) {
		return MacroDirect
	}
	if polygonContains(proximalSouthernItaly, point) {
		return MacroProximal
	}
	for _, polygon := range wideCampanianPolygons {
		if polygonContains(polygon, point) {
			return MacroWide
		}
	}
	return MacroUnaffected
}

func MacroImpactAt(tile TileGeography, turn int) MacroImpact {
	impact := MacroImpact{FloraFactor: 1, FaunaFactor: 1, WaterFactor: 1, HabitatFactor: 1}
	if !MacroEpisodeActive(turn) {
		return impact
	}
	impact.Active, impact.Zone = true, CampanianZone(tile.ID)
	switch impact.Zone {
	case MacroDirect:
		impact.Intensity, impact.FloraFactor, impact.FaunaFactor, impact.WaterFactor, impact.HabitatFactor = 1.00, 0.25, 0.35, 0.75, 0.50
	case MacroProximal:
		impact.Intensity, impact.FloraFactor, impact.FaunaFactor, impact.WaterFactor, impact.HabitatFactor = 0.55, 0.50, 0.60, 0.85, 0.70
	case MacroWide:
		impact.Intensity, impact.FloraFactor, impact.FaunaFactor, impact.WaterFactor, impact.HabitatFactor = 0.20, 0.80, 0.85, 0.95, 0.90
	default:
		return impact
	}
	refugium := 0.20 * clamp01(tile.NaturalShelter)
	impact.LossFraction = impact.Intensity * 0.40 * (1 - refugium)
	if impact.LossFraction > 0.45 {
		impact.LossFraction = 0.45
	}
	impact.HealthLoss = impact.Intensity * 0.25 * (1 - refugium)
	return impact
}
