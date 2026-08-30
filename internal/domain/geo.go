package domain

const (
	MapWidth  = 96
	MapHeight = 64
	TileCount = MapWidth * MapHeight

	MinLongitude = -20.0
	MaxLongitude = 200.0
	MaxLatitude  = 72.0
	MinLatitude  = -48.0
)

type GeoPoint struct {
	Longitude float64
	Latitude  float64
}

type GridPoint struct {
	X float64
	Y float64
}

func TileIDAt(x, y int) (TileID, error) {
	if x < 0 || x >= MapWidth || y < 0 || y >= MapHeight {
		return InvalidTileID, ErrInvalidCoordinate
	}
	return TileID(y*MapWidth + x), nil
}

func TileXY(id TileID) (int, int, error) {
	if id >= TileCount {
		return 0, 0, ErrInvalidCoordinate
	}
	value := int(id)
	return value % MapWidth, value / MapWidth, nil
}

func GeoAt(x, y int) (GeoPoint, error) {
	if _, err := TileIDAt(x, y); err != nil {
		return GeoPoint{}, err
	}
	lon := MinLongitude + float64(x)*(MaxLongitude-MinLongitude)/float64(MapWidth-1)
	lat := MaxLatitude - float64(y)*(MaxLatitude-MinLatitude)/float64(MapHeight-1)
	return GeoPoint{Longitude: lon, Latitude: lat}, nil
}

func ProjectGeo(point GeoPoint) (GridPoint, error) {
	if point.Longitude < MinLongitude || point.Longitude > MaxLongitude || point.Latitude < MinLatitude || point.Latitude > MaxLatitude {
		return GridPoint{}, ErrInvalidCoordinate
	}
	x := (point.Longitude - MinLongitude) * float64(MapWidth-1) / (MaxLongitude - MinLongitude)
	y := (MaxLatitude - point.Latitude) * float64(MapHeight-1) / (MaxLatitude - MinLatitude)
	return GridPoint{X: x, Y: y}, nil
}

// scaledTileCenter returns exact rational tile-center coordinates multiplied by
// the common denominator for both projection axes and by ten units per degree.
func scaledTileCenter(x, y int) scaledPoint {
	const common = int64(5_985) // lcm(95, 63)
	return scaledPoint{
		x: -200*common + int64(x)*2_200*63,
		y: 720*common - int64(y)*1_200*95,
	}
}
