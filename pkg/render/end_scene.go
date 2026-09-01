package render

import (
	"fmt"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const (
	newCampaignButtonX      = 490
	newCampaignButtonY      = 494
	newCampaignButtonWidth  = 300
	newCampaignButtonHeight = 46
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

// NewCampaignButtonContains reports whether a logical-screen point activates
// the terminal scene's new-campaign control.
func NewCampaignButtonContains(x, y int) bool {
	return x >= newCampaignButtonX && x < newCampaignButtonX+newCampaignButtonWidth &&
		y >= newCampaignButtonY && y < newCampaignButtonY+newCampaignButtonHeight
}

func (scene *MapScene) drawEndScene(screen logicalCanvas, ending EndScene) {
	if !ending.Visible {
		return
	}
	accent := endSceneAccent(ending.Result)

	vector.FillRect(screen, 0, 0, 1280, 720, color.RGBA{R: 2, G: 5, B: 7, A: 205}, false)
	vector.FillRect(screen, 240, 70, 800, 508, color.RGBA{R: 20, G: 29, B: 35, A: 252}, false)
	vector.StrokeRect(screen, 240, 70, 800, 508, 2, accent, false)
	scene.drawText(screen, ending.Title, 284, 105, 25, accent)
	scene.drawText(screen, ending.Subtitle, 284, 151, 15, color.RGBA{R: 226, G: 231, B: 227, A: 255})
	vector.StrokeLine(screen, 284, 208, 996, 208, 1, color.RGBA{R: 87, G: 104, B: 110, A: 255}, false)

	scene.drawText(screen, "CAMPAIGN RECORD", 284, 225, 12, accent)
	scene.drawText(screen, fmt.Sprintf("Turn %d  ·  %d BP  ·  %d regions established", ending.Turn, ending.YearBP, ending.RegionsEstablished), 284, 249, 14, color.White)
	scene.drawText(screen, fmt.Sprintf("Homo sapiens   %d people in %d bands", ending.SapiensPopulation, ending.SapiensBands), 284, 278, 14, color.RGBA{R: 245, G: 202, B: 92, A: 255})
	scene.drawText(screen, fmt.Sprintf("Archaic hominins   %d people in %d bands", ending.ArchaicPopulation, ending.ArchaicBands), 284, 304, 14, color.RGBA{R: 201, G: 137, B: 119, A: 255})

	scene.drawText(screen, fmt.Sprintf("DESTINATIONS ESTABLISHED  ·  %d/5", ending.DestinationCount), 284, 344, 12, accent)
	scene.drawText(screen, ending.Destinations, 284, 368, 14, color.White)
	scene.drawText(screen, "EPILOGUE", 284, 419, 12, accent)
	scene.drawText(screen, ending.Epilogue, 284, 443, 14, color.RGBA{R: 226, G: 231, B: 227, A: 255})

	vector.FillRect(screen, newCampaignButtonX, newCampaignButtonY, newCampaignButtonWidth, newCampaignButtonHeight, color.RGBA{R: 48, G: 66, B: 70, A: 255}, false)
	vector.StrokeRect(screen, newCampaignButtonX, newCampaignButtonY, newCampaignButtonWidth, newCampaignButtonHeight, 2, accent, false)
	scene.drawText(screen, "NEW CAMPAIGN   [N]", 548, 507, 16, color.White)
	scene.drawText(screen, "Ctrl/Cmd+S saves this final state", 509, 551, 11, color.RGBA{R: 177, G: 190, B: 188, A: 255})
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
