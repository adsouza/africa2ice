package render

import (
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestTimelineMajorTicksAndEraBoundariesUseCalendarDates(t *testing.T) {
	wantMajorYears := [...]int{80_000, 70_000, 60_000, 50_000, 40_000, 30_000, 20_000}
	if len(timelineMajorTicks) != len(wantMajorYears) {
		t.Fatalf("major ticks = %d", len(timelineMajorTicks))
	}
	for index, tick := range timelineMajorTicks {
		if tick.YearBP != wantMajorYears[index] || math.Abs(tick.Progress-float64(index)/6) > 1e-12 {
			t.Fatalf("major tick %d = %#v", index, tick)
		}
	}
	wantEra := [...]timelineTick{
		{YearBP: 50_000, Progress: 0.5},
		{YearBP: 35_000, Progress: 0.75},
		{YearBP: 25_000, Progress: 11.0 / 12.0},
	}
	if timelineEraBoundaries != wantEra {
		t.Fatalf("era boundaries = %#v, want %#v", timelineEraBoundaries, wantEra)
	}
}

func TestTimelineStateReadsProjectedCalendarValues(t *testing.T) {
	frame := &gameapi.Frame{
		Turn: 1, YearBP: 70_000, CalendarProgress: 0.61,
		Climate: gameapi.ClimateSummary{RegionalAbrupt: [gameapi.RegionCount]float64{gameapi.EastAfrica: -0.2}},
	}
	state := deriveTimelineState(frame)
	if state.Progress != 0.61 || state.CurrentLabel != "70,000 BP" || !state.CurrentOnMajor || !state.ShowTobaContext || state.PulseDirection != -1 {
		t.Fatalf("timeline state = %#v", state)
	}

	frame.YearBP = 75_000
	frame.CalendarProgress = 0.2
	frame.Climate.RegionalAbrupt[gameapi.Arabia] = 0.1
	state = deriveTimelineState(frame)
	if state.CurrentOnMajor || state.ShowTobaContext || state.PulseDirection != 1 {
		t.Fatalf("non-major warm-pulse state = %#v", state)
	}
}

func TestTimelinePositionAndYearProgressClampPresentationInputs(t *testing.T) {
	tests := []struct {
		name     string
		progress float64
		want     float32
	}{
		{name: "start", progress: 0, want: 20},
		{name: "middle", progress: 0.5, want: 120},
		{name: "end", progress: 1, want: 220},
		{name: "below", progress: -1, want: 20},
		{name: "above", progress: 2, want: 220},
		{name: "nan", progress: math.NaN(), want: 20},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := timelinePosition(20, 220, test.progress); got != test.want {
				t.Fatalf("position = %v, want %v", got, test.want)
			}
		})
	}
	for year, want := range map[int]float64{80_000: 0, 50_000: 0.5, 20_000: 1, 90_000: 0, 10_000: 1} {
		if got := timelineProgressForYear(year); math.Abs(got-want) > 1e-12 {
			t.Fatalf("year %d progress = %v, want %v", year, got, want)
		}
	}
}
