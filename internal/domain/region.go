package domain

var regionNames = [RegionCount]string{
	"East Africa", "Rest of Africa", "Arabia", "Levant", "Frangistan", "Central Asia", "South Asia",
	"Southeast Asia", "East Asia", "Yellow River Basin", "Sahul", "Siberia", "Beringia",
}

func (r Region) String() string {
	if r >= RegionCount {
		return "Unknown Region"
	}
	return regionNames[r]
}

func resolveRegion(x, y int, point GeoPoint) (Region, bool) {
	if point.Longitude >= 165 && point.Latitude >= 50 {
		return Beringia, true
	}
	if point.Longitude >= 96 && point.Longitude <= 122 && point.Latitude >= 30 && point.Latitude <= 42 {
		return YellowRiverBasin, true
	}
	if insideFeature("Africa", x, y) && point.Longitude >= 28 && point.Latitude >= -15 && point.Latitude <= 18 {
		return EastAfrica, true
	}
	if insideFeature("Africa", x, y) {
		return RestOfAfrica, true
	}
	if insideFeature("Arabia", x, y) {
		return Arabia, true
	}
	if point.Longitude >= 25 && point.Longitude <= 45 && point.Latitude >= 28 && point.Latitude <= 42 {
		return Levant, true
	}
	if point.Longitude < 45 && point.Latitude >= 35 {
		return Frangistan, true
	}
	if insideFeature("Sahul", x, y) {
		return Sahul, true
	}
	if point.Longitude >= 90 && point.Longitude <= 140 && point.Latitude < 30 {
		return SoutheastAsia, true
	}
	if point.Longitude >= 60 && point.Longitude < 96 && point.Latitude < 32 {
		return SouthAsia, true
	}
	if point.Latitude >= 50 {
		return Siberia, true
	}
	if point.Longitude >= 45 && point.Longitude < 96 && point.Latitude >= 30 {
		return CentralAsia, true
	}
	return EastAsia, true
}
