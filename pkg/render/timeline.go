package render

import (
	"fmt"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const (
	timelineStartYearBP = 80_000
	timelineEndYearBP   = 20_000
)

type timelineTick struct {
	YearBP   int
	Progress float64
}

type timelineState struct {
	Progress        float64
	CurrentLabel    string
	CurrentOnMajor  bool
	ShowTobaContext bool
	PulseDirection  int
}

var timelineMajorTicks = [...]timelineTick{
	{YearBP: 80_000, Progress: 0},
	{YearBP: 70_000, Progress: 1.0 / 6.0},
	{YearBP: 60_000, Progress: 2.0 / 6.0},
	{YearBP: 50_000, Progress: 3.0 / 6.0},
	{YearBP: 40_000, Progress: 4.0 / 6.0},
	{YearBP: 30_000, Progress: 5.0 / 6.0},
	{YearBP: 20_000, Progress: 1},
}

var timelineEraBoundaries = [...]timelineTick{
	{YearBP: 50_000, Progress: 0.5},
	{YearBP: 35_000, Progress: 0.75},
	{YearBP: 25_000, Progress: 11.0 / 12.0},
}

func deriveTimelineState(frame *gameapi.Frame) timelineState {
	if frame == nil {
		return timelineState{}
	}
	progress := frame.CalendarProgress
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		progress = 0
	}
	progress = clampRender(progress)
	state := timelineState{
		Progress:        progress,
		CurrentLabel:    formatTimelineYear(frame.YearBP),
		ShowTobaContext: frame.YearBP <= 73_880,
	}
	for _, tick := range timelineMajorTicks {
		if frame.YearBP == tick.YearBP {
			state.CurrentOnMajor = true
			break
		}
	}
	for _, offset := range frame.Climate.RegionalAbrupt {
		switch {
		case offset > 0:
			state.PulseDirection = 1
		case offset < 0 && state.PulseDirection == 0:
			state.PulseDirection = -1
		}
	}
	return state
}

func formatTimelineYear(yearBP int) string {
	return fmt.Sprintf("%d,%03d BP", yearBP/1_000, yearBP%1_000)
}

func timelineProgressForYear(yearBP int) float64 {
	return clampRender(float64(timelineStartYearBP-yearBP) / float64(timelineStartYearBP-timelineEndYearBP))
}

func timelinePosition(left, right float32, progress float64) float32 {
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		progress = 0
	}
	progress = clampRender(progress)
	return left + (right-left)*float32(progress)
}
