package domain

import "math"

const (
	EquatorTempC    = 28.0
	PolarTempC      = -12.0
	LapseRateCPerKm = 6.5
)

var latitudeSinSquaredBits = [MapHeight]uint64{
	0x3fecf1bbcdcbfa53, 0x3fec4a74158e4a2f, 0x3feb9544cbb00e73, 0x3fead2fae966c0f9,
	0x3fea04723af1d0d9, 0x3fe92a9466f29d8c, 0x3fe84657e6187012, 0x3fe758beec4b7af9,
	0x3fe662d644925232, 0x3fe565b420fc4f3e, 0x3fe46276dfe8e430, 0x3fe35a43c80fe878,
	0x3fe24e45bcb9602e, 0x3fe13fabeb9c1614, 0x3fe02fa875e18e67, 0x3fde3ede2ba6aed1,
	0x3fdc20678573045b, 0x3fda0652a8e83614, 0x3fd7f30050b38c23, 0x3fd5e8c991c869db,
	0x3fd3e9fd335ffd89, 0x3fd1f8dd12a0fd1b, 0x3fd0179b94e387ab, 0x3fcc90b256e97ad5,
	0x3fc91a43d753a662, 0x3fc5cfd67bf6d9d2, 0x3fc2b5234d8ff741, 0x3fbf9b5aacfb5b04,
	0x3fba397b5357b339, 0x3fb54abf56256b74, 0x3fb0d4bb3bfc87a2, 0x3fa9b8f5f40dc3c0,
	0x3fa2ccf87cd44e1d, 0x3f99daa56e9c1737, 0x3f9041536ab3d483, 0x3f81b1adc52afe06,
	0x3f6d4c8a96d7cf3f, 0x3f472c089c8c8ec9, 0x3f072d585328c0f8, 0x3f5a0f9238cbbd73,
	0x3f75def43bbd3a1e, 0x3f871715440988a4, 0x3f93d581ca184804, 0x3f9e4c41f536698c,
	0x3fa571fa56f71d4c, 0x3facc72d7fd4c1e0, 0x3fb28eb7651d66bd, 0x3fb735a7de80573d,
	0x3fbc5324de251bf8, 0x3fc0f0b27c7b7208, 0x3fc3ed0f99531496, 0x3fc71b490cf2d72d,
	0x3fca77c5b676a005, 0x3fcdfeb81ece67aa, 0x3fd0d611630288e1, 0x3fd2bdee536f58d9,
	0x3fd4b4caf5d971a5, 0x3fd6b86e679815a8, 0x3fd8c69151c1d424, 0x3fdadce07d1a75be,
	0x3fdcf8ff73706a54, 0x3fdf188b2b6ff16a, 0x3fe09c8e5df330f3, 0x3fe1ac2609b3c577,
}

func LatitudeSinSquared(row int) (float64, error) {
	if row < 0 || row >= MapHeight {
		return 0, ErrInvalidCoordinate
	}
	return math.Float64frombits(latitudeSinSquaredBits[row]), nil
}

func SeaLevelTemperatureC(row int) (float64, error) {
	factor, err := LatitudeSinSquared(row)
	if err != nil {
		return 0, err
	}
	return EquatorTempC - float64((EquatorTempC-PolarTempC)*factor), nil
}

func HabitatTemperatureC(row int, elevationKm, habitatOffsetC float64) (float64, error) {
	seaLevel, err := SeaLevelTemperatureC(row)
	if err != nil {
		return 0, err
	}
	return seaLevel - float64(LapseRateCPerKm*elevationKm) + habitatOffsetC, nil
}

func LocalTemperatureC(row int, elevationKm, habitatOffsetC, climateNoiseC float64) (float64, error) {
	temperature, err := HabitatTemperatureC(row, elevationKm, habitatOffsetC)
	if err != nil {
		return 0, err
	}
	return temperature + climateNoiseC, nil
}
