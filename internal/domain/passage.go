package domain

const (
	NorthWallacea PassageID = iota
	SouthWallacea
	BeringStrait
	PassageCount
)

type Passage struct {
	ID              PassageID
	From            TileID
	To              TileID
	Cost            float64
	RequiredTech    Technology
	HasRequiredTech bool
	ClimateGated    bool
}

var passageCatalog = [PassageCount]Passage{
	{ID: NorthWallacea, From: 3804, To: 4197, Cost: 4.00, RequiredTech: CoastalNavigation, HasRequiredTech: true},
	{ID: SouthWallacea, From: 4284, To: 4389, Cost: 4.50, RequiredTech: CoastalNavigation, HasRequiredTech: true},
	{ID: BeringStrait, From: 469, To: 476, Cost: 3.00, ClimateGated: true},
}

type PassageAvailability uint8

const (
	PassageNotAtEndpoint PassageAvailability = iota
	PassageSpatialActionSpent
	PassageDestinationUninhabitable
	PassageBeringiaClosed
	PassageNeedsCoastalNavigation
	PassageAvailable
)

func Passages() [PassageCount]Passage { return passageCatalog }

func passageDestination(passage Passage, origin TileID) (TileID, bool) {
	switch origin {
	case passage.From:
		return passage.To, true
	case passage.To:
		return passage.From, true
	default:
		return InvalidTileID, false
	}
}

func passageForEdge(origin, destination TileID) (Passage, bool) {
	for _, passage := range passageCatalog {
		other, atEndpoint := passageDestination(passage, origin)
		if atEndpoint && other == destination {
			return passage, true
		}
	}
	return Passage{}, false
}

func PassageBetween(origin, destination TileID) (Passage, bool) {
	return passageForEdge(origin, destination)
}

func passageAvailability(band Band, passage Passage, habitat *Habitat, longTermTemp float64, checkSpatial bool) PassageAvailability {
	destination, atEndpoint := passageDestination(passage, band.TileID)
	if !atEndpoint {
		return PassageNotAtEndpoint
	}
	if checkSpatial && band.SpatialActionUsed {
		return PassageSpatialActionSpent
	}
	if destination >= TileCount || habitat[destination].BaselineK <= 0 {
		return PassageDestinationUninhabitable
	}
	if passage.ClimateGated && !BeringiaOpen(longTermTemp) {
		return PassageBeringiaClosed
	}
	if passage.HasRequiredTech && !band.Technology.Has(passage.RequiredTech) {
		return PassageNeedsCoastalNavigation
	}
	return PassageAvailable
}

func ValidatePassages(grid *Grid) error {
	incident := [TileCount]uint8{}
	for index, passage := range passageCatalog {
		if passage.ID != PassageID(index) || passage.From >= TileCount || passage.To >= TileCount || passage.From == passage.To || passage.Cost <= 0 {
			return ErrInvalidValue
		}
		from, fromOK := grid.Tile(passage.From)
		to, toOK := grid.Tile(passage.To)
		if !fromOK || !toOK || !from.Land || !to.Land {
			return ErrInvalidValue
		}
		for _, edge := range grid.OrdinaryEdges(passage.From) {
			if edge.To == passage.To {
				return ErrInvalidValue
			}
		}
		incident[passage.From]++
		incident[passage.To]++
		if incident[passage.From] > 2 || incident[passage.To] > 2 {
			return ErrInvalidValue
		}
	}
	return nil
}
