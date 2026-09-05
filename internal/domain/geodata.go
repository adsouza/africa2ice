package domain

type scaledPoint struct{ x, y int64 }

type coordinate struct {
	lon10 int64
	lat10 int64
}

type polygonFeature struct {
	id     string
	points []coordinate
}

type highlandFeature struct {
	polygonFeature
	heightKm float64
}

type riverFeature struct {
	id     string
	points []GeoPoint
}

type shelterFeature struct {
	id     string
	cx, cy int
	rx, ry int
	rating float64
}

// c records an authored coordinate as exact integer tenths of a degree. Call
// sites spell the degrees and the *10, so the scaling is an untyped constant
// expression the compiler folds exactly: 12.7*10 is precisely 127, where the
// earlier int64(lon*10) truncated a runtime float product and was correct only
// by luck of the target's rounding. §5 forbids that conversion for this reason.
func c(lon10, lat10 int64) coordinate {
	return coordinate{lon10: lon10, lat10: lat10}
}
func g(lon, lat float64) GeoPoint { return GeoPoint{Longitude: lon, Latitude: lat} }

var landPolygons = []polygonFeature{
	{"Africa", []coordinate{c(-17*10, 37*10), c(10*10, 37*10), c(25*10, 33*10), c(35*10, 30*10), c(52*10, 12*10), c(44*10, -12*10), c(34*10, -35*10), c(18*10, -35*10), c(10*10, -25*10), c(0*10, -5*10), c(-17*10, 15*10)}},
	{"Arabia", []coordinate{c(34*10, 32*10), c(58*10, 30*10), c(57*10, 16*10), c(44*10, 12*10), c(34*10, 26*10)}},
	{"LevantAnatolia", []coordinate{c(25*10, 42*10), c(45*10, 42*10), c(45*10, 30*10), c(34*10, 27*10), c(28*10, 32*10)}},
	{"Frangistan", []coordinate{c(-10*10, 36*10), c(25*10, 36*10), c(45*10, 42*10), c(40*10, 60*10), c(20*10, 70*10), c(-10*10, 60*10)}},
	{"Eurasia", []coordinate{c(25*10, 36*10), c(45*10, 42*10), c(60*10, 31*10), c(92*10, 20*10), c(125*10, 18*10), c(160*10, 48*10), c(180*10, 58*10), c(176*10, 70*10), c(35*10, 70*10)}},
	{"SouthAsiaPeninsula", []coordinate{c(64*10, 26*10), c(91*10, 25*10), c(86*10, 7*10), c(76*10, 6*10), c(67*10, 18*10)}},
	{"SoutheastAsia", []coordinate{c(90*10, 25*10), c(122*10, 23*10), c(132*10, 7*10), c(126*10, -9*10), c(104*10, -10*10), c(96*10, 7*10)}},
	{"Sahul", []coordinate{c(112*10, -10*10), c(154*10, -8*10), c(160*10, -44*10), c(113*10, -45*10)}},
	{"NortheastSiberia", []coordinate{c(155*10, 50*10), c(180*10, 52*10), c(188*10, 66*10), c(176*10, 72*10), c(158*10, 70*10)}},
	{"WesternAlaska", []coordinate{c(190*10, 52*10), c(200*10, 54*10), c(200*10, 72*10), c(184*10, 70*10), c(184*10, 61*10)}},
}

var waterPolygons = []polygonFeature{
	{"Mediterranean", []coordinate{c(-6*10, 36*10), c(0*10, 42*10), c(10*10, 45*10), c(20*10, 44*10), c(30*10, 41*10), c(37*10, 36*10), c(32*10, 31*10), c(20*10, 31*10), c(10*10, 35*10), c(0*10, 35*10)}},
	{"RedSea", []coordinate{c(32*10, 29*10), c(37*10, 30*10), c(44*10, 13*10), c(39*10, 12*10), c(34*10, 22*10)}},
	{"PersianGulf", []coordinate{c(47*10, 31*10), c(57*10, 30*10), c(57*10, 24*10), c(49*10, 24*10)}},
	{"Caspian", []coordinate{c(46*10, 47*10), c(55*10, 47*10), c(55*10, 36*10), c(47*10, 36*10)}},
	{"BlackSea", []coordinate{c(27*10, 47*10), c(42*10, 47*10), c(42*10, 40*10), c(28*10, 40*10)}},
	{"NorthWallaceaGap", []coordinate{c(119*10, 4*10), c(137*10, 4*10), c(137*10, -7*10), c(119*10, -7*10)}},
	{"SouthWallaceaGap", []coordinate{c(121*10, -7*10), c(139*10, -7*10), c(139*10, -16*10), c(121*10, -16*10)}},
	{"BeringStrait", []coordinate{c(178*10, 68*10), c(193*10, 68*10), c(193*10, 61*10), c(178*10, 61*10)}},
}

var highlands = []highlandFeature{
	{polygonFeature{"Atlas", []coordinate{c(-10*10, 36*10), c(11*10, 36*10), c(11*10, 28*10), c(-10*10, 28*10)}}, 1.50},
	{polygonFeature{"EthiopianHighlands", []coordinate{c(33*10, 15*10), c(43*10, 15*10), c(43*10, 4*10), c(33*10, 4*10)}}, 2.00},
	{polygonFeature{"Zagros", []coordinate{c(43*10, 38*10), c(57*10, 38*10), c(57*10, 27*10), c(43*10, 27*10)}}, 1.50},
	// The Caucasus is authored as two massifs rather than one, because between
	// the Black Sea and the Caspian the land narrows to a few tile columns and a
	// continuous 2 km front across all of them seals the only eastern route out
	// of the Levant. Highland tiles are cold-limited: at 2 km the lapse rate
	// plus LGMCooling drives ThermalSuitability under VegetationColdCutoffC, so
	// such a front does not merely slow a band down, it reaches BaselineK = 0
	// and stays there for the rest of the campaign. The gap between the lobes is
	// the Colchis corridor on the Black Sea shore, the same low coastal approach
	// that carries the historical route, and it is a deliberate pass in exactly
	// the sense the escarpment catalog means it.
	{polygonFeature{"CaucasusWest", []coordinate{c(37*10, 46*10), c(41.5*10, 46*10), c(41.5*10, 39*10), c(37*10, 39*10)}}, 2.00},
	{polygonFeature{"CaucasusEast", []coordinate{c(43.5*10, 46*10), c(51*10, 46*10), c(51*10, 39*10), c(43.5*10, 39*10)}}, 2.00},
	{polygonFeature{"HimalayaTibetanPlateau", []coordinate{c(69*10, 37*10), c(105*10, 37*10), c(105*10, 34*10), c(101*10, 34*10), c(101*10, 26*10), c(69*10, 26*10)}}, 3.00},
	{polygonFeature{"Alps", []coordinate{c(4*10, 49*10), c(17*10, 49*10), c(17*10, 43*10), c(4*10, 43*10)}}, 2.00},
	{polygonFeature{"Urals", []coordinate{c(54*10, 68*10), c(69*10, 68*10), c(69*10, 50*10), c(54*10, 50*10)}}, 1.25},
	{polygonFeature{"Altai", []coordinate{c(79*10, 54*10), c(99*10, 54*10), c(99*10, 43*10), c(79*10, 43*10)}}, 2.00},
	{polygonFeature{"NewGuineaCentralRange", []coordinate{c(130*10, 0*10), c(151*10, 0*10), c(151*10, -11*10), c(130*10, -11*10)}}, 2.50},
	{polygonFeature{"AlaskaRange", []coordinate{c(184*10, 69*10), c(200*10, 69*10), c(200*10, 56*10), c(184*10, 56*10)}}, 2.50},
}

var rivers = []riverFeature{
	{"Nile", []GeoPoint{g(31, -3), g(32, 5), g(31, 15), g(32, 24), g(31, 31)}},
	{"Niger", []GeoPoint{g(-11, 11), g(-5, 14), g(2, 13), g(7, 10), g(6, 5)}},
	{"Congo", []GeoPoint{g(27, -11), g(25, -4), g(20, 1), g(15, -4), g(12, -6)}},
	{"Zambezi", []GeoPoint{g(24, -12), g(28, -17), g(33, -18), g(36, -19)}},
	{"TigrisEuphrates", []GeoPoint{g(39, 38), g(42, 34), g(45, 31), g(48, 29)}},
	{"Indus", []GeoPoint{g(74, 34), g(72, 29), g(69, 24)}},
	{"Ganges", []GeoPoint{g(78, 30), g(84, 27), g(90, 24)}},
	{"Danube", []GeoPoint{g(9, 48), g(19, 47), g(27, 45), g(29, 44)}},
	{"Yellow", []GeoPoint{g(96, 35), g(103, 37), g(111, 35), g(119, 37)}},
	{"Amur", []GeoPoint{g(121, 50), g(132, 49), g(141, 52)}},
	{"Lena", []GeoPoint{g(106, 53), g(118, 61), g(126, 69)}},
	{"Yukon", []GeoPoint{g(191, 61), g(197, 64), g(200, 63)}},
}

var shelters = []shelterFeature{
	{"EastAfricanRift", 24, 36, 3, 10, 0.50},
	{"SouthernAfricanEscarpment", 21, 50, 6, 5, 0.50},
	{"AtlasLevantAnatoliaKarst", 27, 18, 10, 4, 0.50},
	{"ZagrosCentralAsianUplands", 39, 20, 10, 5, 0.50},
	{"SouthSoutheastAsianLimestone", 54, 30, 12, 6, 1.00},
	{"YellowRiverEastAsianUplands", 59, 19, 7, 5, 0.25},
	{"AltaiSouthernSiberianCaves", 62, 10, 10, 4, 0.50},
	{"SahulEscarpments", 73, 44, 6, 5, 0.50},
	{"BeringianAlaskanUplands", 91, 5, 4, 4, 0.25},
}

func polygonContains(points []coordinate, point scaledPoint) bool {
	const common = int64(5_985)
	inside := false
	for i, current := range points {
		previous := points[(i+len(points)-1)%len(points)]
		ax, ay := previous.lon10*common, previous.lat10*common
		bx, by := current.lon10*common, current.lat10*common
		cross := (point.x-ax)*(by-ay) - (point.y-ay)*(bx-ax)
		if cross == 0 && point.x >= min64(ax, bx) && point.x <= max64(ax, bx) && point.y >= min64(ay, by) && point.y <= max64(ay, by) {
			return true
		}
		if (ay > point.y) == (by > point.y) {
			continue
		}
		left := (point.x - ax) * (by - ay)
		right := (point.y - ay) * (bx - ax)
		if (by > ay && left < right) || (by < ay && left > right) {
			inside = !inside
		}
	}
	return inside
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func insideFeature(id string, x, y int) bool {
	point := scaledTileCenter(x, y)
	for _, feature := range landPolygons {
		if feature.id == id {
			return polygonContains(feature.points, point)
		}
	}
	return false
}
