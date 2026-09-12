package application

import "github.com/adsouza/africa2ice/pkg/gameapi"

// Independent schematic rings informed by basin contours, not scaled copies.
// Malawi: ILEC bathymetry AFR-13-01 and the shallow southern shelf described
// by Eccles (1974); early low ~350m, later low ~200m below modern. Chronology:
// Scholz/Cohen (2007). Lisan: Bartov (2002), with the mapped -165m highstand
// reproduced in Ron et al. (2006), fig. 1. See docs/LAKES.md for the precise
// distinction between source observations and our authored simplifications.
var changingLakeFeatures = []lakeFeature{
	{"Lake Malawi / Nyasa", gameapi.LakeStageStart(gameapi.MalawiEarlyLow), gameapi.MalawiEarlyLow, []gameapi.LakePoint{
		{X: 34.04, Y: -9.85}, {X: 34.3, Y: -10.12}, {X: 34.43, Y: -10.6}, {X: 34.48, Y: -11.15},
		{X: 34.6, Y: -11.65}, {X: 34.68, Y: -12.05}, {X: 34.59, Y: -12.42}, {X: 34.67, Y: -12.85},
		{X: 34.58, Y: -12.9}, {X: 34.42, Y: -12.4}, {X: 34.25, Y: -11.85}, {X: 34.36, Y: -11.5},
		{X: 34.24, Y: -10.85}, {X: 34.06, Y: -10.25},
	}},
	{"Lake Malawi / Nyasa", gameapi.LakeStageStart(gameapi.MalawiRecovered), gameapi.MalawiRecovered, []gameapi.LakePoint{
		{X: 33.91, Y: -9.5}, {X: 34.18, Y: -9.62}, {X: 34.44, Y: -10.05}, {X: 34.52, Y: -10.6},
		{X: 34.62, Y: -11.18}, {X: 34.8, Y: -11.58}, {X: 34.95, Y: -11.7}, {X: 34.96, Y: -12.08},
		{X: 34.8, Y: -12.5}, {X: 34.93, Y: -13.05}, {X: 34.94, Y: -13.48}, {X: 35.16, Y: -13.85},
		{X: 35.28, Y: -14.4}, {X: 35.08, Y: -14.17}, {X: 34.93, Y: -14.02}, {X: 34.88, Y: -14.32},
		{X: 34.65, Y: -14.2}, {X: 34.62, Y: -13.8}, {X: 34.36, Y: -13.35}, {X: 34.28, Y: -12.8},
		{X: 34.06, Y: -12.22}, {X: 34.08, Y: -11.9}, {X: 34.29, Y: -11.4}, {X: 34.19, Y: -10.88},
		{X: 34.08, Y: -10.4}, {X: 33.9, Y: -10.0},
	}},
	{"Lake Malawi / Nyasa", gameapi.LakeStageStart(gameapi.MalawiGlacialLow), gameapi.MalawiGlacialLow, []gameapi.LakePoint{
		{X: 34.01, Y: -9.68}, {X: 34.25, Y: -9.88}, {X: 34.43, Y: -10.24}, {X: 34.49, Y: -10.78},
		{X: 34.59, Y: -11.28}, {X: 34.79, Y: -11.68}, {X: 34.89, Y: -11.78}, {X: 34.88, Y: -12.06},
		{X: 34.73, Y: -12.5}, {X: 34.86, Y: -13.03}, {X: 34.87, Y: -13.4}, {X: 34.6, Y: -13.43},
		{X: 34.49, Y: -13.03}, {X: 34.35, Y: -12.65}, {X: 34.17, Y: -12.17}, {X: 34.19, Y: -11.85},
		{X: 34.34, Y: -11.44}, {X: 34.23, Y: -10.85}, {X: 34.13, Y: -10.35}, {X: 34.0, Y: -10.0},
	}},
	// The usual lower lake occupied the Dead Sea basin and lower Jordan Valley,
	// below the sill to the northern basin. The highstand adds that northern arm.
	{"Lake Lisan", gameapi.LakeStageStart(gameapi.LisanInitial), gameapi.LisanInitial, []gameapi.LakePoint{
		{X: 35.54, Y: 32.18}, {X: 35.61, Y: 32.14}, {X: 35.57, Y: 31.85}, {X: 35.57, Y: 31.55},
		{X: 35.52, Y: 31.28}, {X: 35.43, Y: 31.05}, {X: 35.31, Y: 30.98}, {X: 35.3, Y: 31.16},
		{X: 35.36, Y: 31.4}, {X: 35.39, Y: 31.67}, {X: 35.48, Y: 31.88},
	}},
	{"Lake Lisan", gameapi.LakeStageStart(gameapi.LisanHigh), gameapi.LisanHigh, []gameapi.LakePoint{
		{X: 35.55, Y: 32.9}, {X: 35.64, Y: 32.88}, {X: 35.66, Y: 32.76}, {X: 35.6, Y: 32.57},
		{X: 35.61, Y: 32.4}, {X: 35.64, Y: 32.22}, {X: 35.62, Y: 31.94}, {X: 35.63, Y: 31.73},
		{X: 35.6, Y: 31.51}, {X: 35.6, Y: 31.3}, {X: 35.49, Y: 31.01}, {X: 35.31, Y: 30.79},
		{X: 35.24, Y: 30.84}, {X: 35.26, Y: 31.13}, {X: 35.31, Y: 31.39}, {X: 35.35, Y: 31.68},
		{X: 35.42, Y: 31.95}, {X: 35.47, Y: 32.23}, {X: 35.49, Y: 32.45}, {X: 35.51, Y: 32.6},
		{X: 35.5, Y: 32.78},
	}},
	// A partial retreat within the campaign, not the much smaller Holocene sea.
	{"Lake Lisan", gameapi.LakeStageStart(gameapi.LisanDeclining), gameapi.LisanDeclining, []gameapi.LakePoint{
		{X: 35.54, Y: 32.51}, {X: 35.59, Y: 32.49}, {X: 35.61, Y: 32.23}, {X: 35.58, Y: 31.93},
		{X: 35.59, Y: 31.71}, {X: 35.57, Y: 31.48}, {X: 35.56, Y: 31.28}, {X: 35.44, Y: 31.04},
		{X: 35.33, Y: 30.89}, {X: 35.28, Y: 30.96}, {X: 35.3, Y: 31.15}, {X: 35.34, Y: 31.4},
		{X: 35.37, Y: 31.68}, {X: 35.46, Y: 31.96}, {X: 35.5, Y: 32.25},
	}},
}
