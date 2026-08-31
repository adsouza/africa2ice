package verification

import (
	"fmt"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// EncodeMap renders two 96x64 ASCII layers from a frame without I/O.
func EncodeMap(frame *gameapi.Frame) string {
	if frame == nil || len(frame.Tiles) == 0 {
		return ""
	}
	width, height := 0, 0
	for _, tile := range frame.Tiles {
		width = max(width, tile.X+1)
		height = max(height, tile.Y+1)
	}
	biomes := make([]byte, width*height)
	regions := make([]byte, width*height)
	for index := range biomes {
		biomes[index], regions[index] = '~', '~'
	}
	for _, tile := range frame.Tiles {
		if tile.X < 0 || tile.X >= width || tile.Y < 0 || tile.Y >= height {
			continue
		}
		index := tile.Y*width + tile.X
		if tile.Land {
			biomes[index] = biomeCode(tile.Biome)
			regions[index] = regionCode(tile.Region)
		}
	}
	var output strings.Builder
	fmt.Fprintf(&output, "biomes %dx%d\n", width, height)
	writeLayer(&output, biomes, width, height)
	output.WriteString("regions\n")
	writeLayer(&output, regions, width, height)
	return output.String()
}

func writeLayer(output *strings.Builder, layer []byte, width, height int) {
	for y := 0; y < height; y++ {
		output.Write(layer[y*width : (y+1)*width])
		output.WriteByte('\n')
	}
}

func biomeCode(biome gameapi.Biome) byte {
	codes := [...]byte{'W', 'V', 'C', 'M', 'D', 'T'}
	if biome >= gameapi.BiomeCount {
		return '?'
	}
	return codes[biome]
}

func regionCode(region gameapi.Region) byte {
	codes := [...]byte{'e', 'a', 'r', 'l', 'f', 'c', 'i', 's', 'E', 'y', 'u', 'S', 'b'}
	if region >= gameapi.RegionCount {
		return '?'
	}
	return codes[region]
}
