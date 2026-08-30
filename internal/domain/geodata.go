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

func c(lon, lat float64) coordinate {
	return coordinate{lon10: int64(lon * 10), lat10: int64(lat * 10)}
}
func g(lon, lat float64) GeoPoint { return GeoPoint{Longitude: lon, Latitude: lat} }

var landPolygons = []polygonFeature{
	{"Africa", []coordinate{c(-17, 37), c(10, 37), c(25, 33), c(35, 30), c(52, 12), c(44, -12), c(34, -35), c(18, -35), c(10, -25), c(0, -5), c(-17, 15)}},
	{"Arabia", []coordinate{c(34, 32), c(58, 30), c(57, 16), c(44, 12), c(34, 26)}},
	{"LevantAnatolia", []coordinate{c(25, 42), c(45, 42), c(45, 30), c(34, 27), c(28, 32)}},
	{"Frangistan", []coordinate{c(-10, 36), c(25, 36), c(45, 42), c(40, 60), c(20, 70), c(-10, 60)}},
	{"Eurasia", []coordinate{c(25, 36), c(45, 42), c(60, 31), c(92, 20), c(125, 18), c(160, 48), c(180, 58), c(176, 70), c(35, 70)}},
	{"SouthAsiaPeninsula", []coordinate{c(64, 26), c(91, 25), c(86, 7), c(76, 6), c(67, 18)}},
	{"SoutheastAsia", []coordinate{c(90, 25), c(122, 23), c(132, 7), c(126, -9), c(104, -10), c(96, 7)}},
	{"Sahul", []coordinate{c(112, -10), c(154, -8), c(160, -44), c(113, -45)}},
	{"NortheastSiberia", []coordinate{c(155, 50), c(180, 52), c(188, 66), c(176, 72), c(158, 70)}},
	{"WesternAlaska", []coordinate{c(190, 52), c(200, 54), c(200, 72), c(184, 70), c(184, 61)}},
}

var waterPolygons = []polygonFeature{
	{"Mediterranean", []coordinate{c(-6, 36), c(0, 42), c(10, 45), c(20, 44), c(30, 41), c(37, 36), c(32, 31), c(20, 31), c(10, 35), c(0, 35)}},
	{"RedSea", []coordinate{c(32, 29), c(37, 30), c(44, 13), c(39, 12), c(34, 22)}},
	{"PersianGulf", []coordinate{c(47, 31), c(57, 30), c(57, 24), c(49, 24)}},
	{"Caspian", []coordinate{c(46, 47), c(55, 47), c(55, 36), c(47, 36)}},
	{"BlackSea", []coordinate{c(27, 47), c(42, 47), c(42, 40), c(28, 40)}},
	{"NorthWallaceaGap", []coordinate{c(119, 4), c(137, 4), c(137, -7), c(119, -7)}},
	{"SouthWallaceaGap", []coordinate{c(121, -7), c(139, -7), c(139, -16), c(121, -16)}},
	{"BeringStrait", []coordinate{c(178, 68), c(193, 68), c(193, 61), c(178, 61)}},
}

var highlands = []highlandFeature{
	{polygonFeature{"Atlas", []coordinate{c(-10, 36), c(11, 36), c(11, 28), c(-10, 28)}}, 1.50},
	{polygonFeature{"EthiopianHighlands", []coordinate{c(33, 15), c(43, 15), c(43, 4), c(33, 4)}}, 2.00},
	{polygonFeature{"Zagros", []coordinate{c(43, 38), c(57, 38), c(57, 27), c(43, 27)}}, 1.50},
	{polygonFeature{"Caucasus", []coordinate{c(37, 46), c(51, 46), c(51, 39), c(37, 39)}}, 2.00},
	{polygonFeature{"Himalaya", []coordinate{c(69, 37), c(101, 37), c(101, 26), c(69, 26)}}, 3.00},
	{polygonFeature{"Alps", []coordinate{c(4, 49), c(17, 49), c(17, 43), c(4, 43)}}, 2.00},
	{polygonFeature{"Urals", []coordinate{c(54, 68), c(69, 68), c(69, 50), c(54, 50)}}, 1.25},
	{polygonFeature{"Altai", []coordinate{c(79, 54), c(99, 54), c(99, 43), c(79, 43)}}, 2.00},
	{polygonFeature{"NewGuineaCentralRange", []coordinate{c(130, 0), c(151, 0), c(151, -11), c(130, -11)}}, 2.50},
	{polygonFeature{"AlaskaRange", []coordinate{c(184, 69), c(200, 69), c(200, 56), c(184, 56)}}, 2.50},
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
