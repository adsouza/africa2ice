package render

import (
	"fmt"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// EndSceneX, EndSceneY, EndSceneWidth and EndSceneHeight are the terminal
// panel's DIP geometry, exported so pkg/hud can align its own widgets (the
// New Campaign button) to the same rectangle and a test can assert
// containment inside the map area rather than duplicating the numbers. The
// panel sits at 60-844: centred on 452, the centre of the map area, 40 px
// clear of the map's right edge and 64 px clear of the HUD column that
// starts at x 908 -- fixing the bug where the panel's old 240-1040 span ran
// under that column. It is only 16 px narrower than before, so no drawn line
// needs re-wrapping.
const (
	EndSceneX      = 60.0
	EndSceneY      = 70.0
	EndSceneWidth  = 784.0
	EndSceneHeight = 508.0
	endSceneTextX  = EndSceneX + 44
)

// EndScene contains UI-authored terminal presentation content for the drawing
// adapter. It is transient and is never simulation or save state.
type EndScene struct {
	Visible            bool
	Result             gameapi.CampaignResult
	Title              string
	Subtitle           string
	Epilogue           string
	Destinations       string
	Turn               int
	YearBP             int
	SapiensPopulation  uint64
	ArchaicPopulation  uint64
	SapiensBands       int
	ArchaicBands       int
	RegionsEstablished int
	DestinationCount   int
}

func (scene *MapScene) drawEndScene(screen logicalCanvas, ending EndScene) {
	if !ending.Visible {
		return
	}
	accent := endSceneAccent(ending.Result)

	vector.FillRect(screen, 0, 0, 1280, 720, color.RGBA{R: 2, G: 5, B: 7, A: 205}, false)
	vector.FillRect(screen, EndSceneX, EndSceneY, EndSceneWidth, EndSceneHeight, color.RGBA{R: 20, G: 29, B: 35, A: 252}, false)
	vector.StrokeRect(screen, EndSceneX, EndSceneY, EndSceneWidth, EndSceneHeight, 2, accent, false)
	scene.drawText(screen, ending.Title, endSceneTextX, 105, 25, accent)
	scene.drawText(screen, ending.Subtitle, endSceneTextX, 151, 15, color.RGBA{R: 226, G: 231, B: 227, A: 255})
	vector.StrokeLine(screen, endSceneTextX, 208, EndSceneX+EndSceneWidth-44, 208, 1, color.RGBA{R: 87, G: 104, B: 110, A: 255}, false)

	scene.drawText(screen, "CAMPAIGN RECORD", endSceneTextX, 225, 12, accent)
	scene.drawText(screen, fmt.Sprintf("Turn %d  ·  %d BP  ·  %d regions established", ending.Turn, ending.YearBP, ending.RegionsEstablished), endSceneTextX, 249, 14, color.White)
	scene.drawText(screen, fmt.Sprintf("Homo sapiens   %d people in %d bands", ending.SapiensPopulation, ending.SapiensBands), endSceneTextX, 278, 14, color.RGBA{R: 245, G: 202, B: 92, A: 255})
	scene.drawText(screen, fmt.Sprintf("Archaic hominins   %d people in %d bands", ending.ArchaicPopulation, ending.ArchaicBands), endSceneTextX, 304, 14, color.RGBA{R: 201, G: 137, B: 119, A: 255})

	scene.drawText(screen, fmt.Sprintf("DESTINATIONS ESTABLISHED  ·  %d/5", ending.DestinationCount), endSceneTextX, 344, 12, accent)
	scene.drawText(screen, ending.Destinations, endSceneTextX, 368, 14, color.White)
	scene.drawText(screen, "EPILOGUE", endSceneTextX, 419, 12, accent)
	scene.drawText(screen, ending.Epilogue, endSceneTextX, 443, 14, color.RGBA{R: 226, G: 231, B: 227, A: 255})
}

func endSceneAccent(result gameapi.CampaignResult) color.RGBA {
	accent := color.RGBA{R: 203, G: 172, B: 104, A: 255}
	switch result {
	case gameapi.Victory:
		accent = color.RGBA{R: 121, G: 195, B: 137, A: 255}
	case gameapi.Extinction:
		accent = color.RGBA{R: 232, G: 112, B: 92, A: 255}
	}
	return accent
}
