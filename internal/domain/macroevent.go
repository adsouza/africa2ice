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

var directCampanianPlain = []coordinate{c(12.7*10, 41.5*10), c(14.8*10, 42.3*10), c(16.5*10, 41.2*10), c(15.8*10, 39.8*10), c(12.8*10, 40.0*10)}
var proximalSouthernItaly = []coordinate{c(9.0*10, 44.5*10), c(14.0*10, 45.0*10), c(19.0*10, 43.0*10), c(20.0*10, 39.0*10), c(17.0*10, 36.0*10), c(12.0*10, 36.5*10), c(9.0*10, 40.0*10)}
var wideCampanianPolygons = [][]coordinate{
	{c(13.5*10, 40.0*10), c(18.0*10, 42.0*10), c(27.0*10, 40.0*10), c(34.0*10, 36.0*10), c(32.0*10, 31.0*10), c(22.0*10, 30.0*10), c(15.0*10, 34.0*10), c(13.0*10, 38.0*10)},
	{c(15.0*10, 44.0*10), c(20.0*10, 48.0*10), c(28.0*10, 49.0*10), c(31.0*10, 46.0*10), c(30.0*10, 42.0*10), c(25.0*10, 39.0*10), c(19.0*10, 39.0*10), c(15.0*10, 41.0*10)},
	{c(28.0*10, 48.0*10), c(33.0*10, 52.5*10), c(41.5*10, 53.0*10), c(44.0*10, 50.0*10), c(42.0*10, 46.0*10), c(35.0*10, 43.5*10), c(30.0*10, 45.0*10)},
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
	refugium := float64(0.20 * clamp01(tile.NaturalShelter))
	impact.LossFraction = float64(impact.Intensity * 0.40 * (1 - refugium))
	if impact.LossFraction > 0.45 {
		impact.LossFraction = 0.45
	}
	impact.HealthLoss = float64(impact.Intensity * 0.25 * (1 - refugium))
	return impact
}
