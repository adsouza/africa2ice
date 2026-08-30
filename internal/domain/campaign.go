package domain

const MaxCampaignTurn = 400

type CampaignDateValue struct {
	Turn             int
	YearBP           int
	Era              CampaignEra
	YearsPerTurn     int
	CalendarProgress float64
}

func CampaignDate(turn int) (CampaignDateValue, error) {
	if turn < 0 || turn > MaxCampaignTurn {
		return CampaignDateValue{}, ErrInvalidTurn
	}
	var era CampaignEra
	var startTurn, startYear, yearsPerTurn int
	switch {
	case turn < 100:
		era, startTurn, startYear, yearsPerTurn = EraEarly, 0, 80_000, 300
	case turn < 200:
		era, startTurn, startYear, yearsPerTurn = EraMiddle, 100, 50_000, 150
	case turn < 300:
		era, startTurn, startYear, yearsPerTurn = EraLate, 200, 35_000, 100
	default:
		era, startTurn, startYear, yearsPerTurn = EraFinal, 300, 25_000, 50
	}
	year := startYear - yearsPerTurn*(turn-startTurn)
	progress := float64(80_000-year) / 60_000
	return CampaignDateValue{Turn: turn, YearBP: year, Era: era, YearsPerTurn: yearsPerTurn, CalendarProgress: progress}, nil
}

func SeasonForTurn(turn int) (Season, error) {
	if turn < 0 || turn > MaxCampaignTurn {
		return 0, ErrInvalidTurn
	}
	return Season((turn % 12) / 3), nil
}
