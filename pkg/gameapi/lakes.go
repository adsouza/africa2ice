package gameapi

// LakeStage names the small set of research-informed presentation shorelines.
// These stages affect overlays only, not the simulation's terrain or resources.
type LakeStage uint8

const (
	LakeUnstaged LakeStage = iota
	MalawiEarlyLow
	MalawiRecovered
	MalawiGlacialLow
	LisanInitial
	LisanHigh
	LisanDeclining
)

// LakeStageStart is the representative date at which an authored shoreline
// becomes active. Keeping it here lets notes and map projection share a clock.
func LakeStageStart(stage LakeStage) int {
	switch stage {
	case MalawiEarlyLow:
		return 80000
	case MalawiRecovered:
		return 60000
	case MalawiGlacialLow:
		return 35000
	case LisanInitial:
		return 70000
	case LisanHigh:
		return 27000
	case LisanDeclining:
		return 23000
	default:
		return 0
	}
}

func LakeStageAt(name string, yearBP int) LakeStage {
	switch name {
	case "Lake Malawi / Nyasa":
		switch {
		case yearBP <= LakeStageStart(MalawiGlacialLow):
			return MalawiGlacialLow
		case yearBP <= LakeStageStart(MalawiRecovered):
			return MalawiRecovered
		default:
			return MalawiEarlyLow
		}
	case "Lake Lisan":
		switch {
		case yearBP <= LakeStageStart(LisanDeclining):
			return LisanDeclining
		case yearBP <= LakeStageStart(LisanHigh):
			return LisanHigh
		case yearBP <= LakeStageStart(LisanInitial):
			return LisanInitial
		}
	}
	return LakeUnstaged
}
