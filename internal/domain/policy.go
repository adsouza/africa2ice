package domain

import "github.com/adsouza/africa2ice/pkg/gameapi"

const (
	MinEstablishedBand                Population = gameapi.MinEstablishedBand
	MinSplitSourcePopulation          Population = gameapi.MinSplitSourcePopulation
	SplitStressThreshold                         = gameapi.SplitStressThreshold
	MaxBands                                     = gameapi.MaxBands
	referenceRouteDeparturePopulation Population = gameapi.ReferenceRouteDeparturePopulation
)

func referenceRouteStepCost(capacity float64) uint8 {
	return uint8(gameapi.ReferenceRouteStepCost(capacity))
}
