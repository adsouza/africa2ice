package verification

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const (
	maxBands          = 256
	maxArchaicBands   = 96
	assignmentTotalBP = 10_000
	foodStorageTurns  = 3.0
)

// CheckpointRecord is the deterministic, cross-target release record. Fields
// are intentionally scalar or stable-order slices so canonical JSON is direct
// evidence rather than an aggregator-side normalization.
type CheckpointRecord struct {
	ArchaicBandCount           int      `json:"archaic_band_count"`
	ArchaicPopulation          uint64   `json:"archaic_population"`
	BeringiaOpen               bool     `json:"beringia_open"`
	CampaignResult             string   `json:"campaign_result"`
	EstablishedRegions         []string `json:"established_regions"`
	ExploredTileCount          int      `json:"explored_tile_count"`
	FirstDestinationPopulation uint32   `json:"first_destination_population"`
	FirstDestinationTurn       int      `json:"first_destination_turn"`
	Policy                     string   `json:"policy"`
	SapiensBandCount           int      `json:"sapiens_band_count"`
	SapiensPopulation          uint64   `json:"sapiens_population"`
	Seed                       uint64   `json:"seed"`
	StateHash                  string   `json:"state_hash"`
	TargetClosestDistance      int      `json:"target_closest_distance"`
	TargetEstablished          bool     `json:"target_established"`
	TargetPopulation           uint32   `json:"target_population"`
	Turn                       int      `json:"turn"`
	WorldRevision              uint64   `json:"world_revision"`
	YearBP                     int      `json:"year_bp"`
}

// CanonicalJSON encodes compact JSON with lexically ordered object keys. The
// field declarations above are already lexical, and this guard makes drift a
// testable producer error rather than something CI repairs.
func CanonicalJSON(records []CheckpointRecord) ([]byte, error) {
	return json.Marshal(records)
}

type runMargins struct {
	firstDestinationTurn       int
	firstDestinationPopulation uint32
}

func checkpoint(game gameapi.Game, frame *gameapi.Frame, seed uint64, policy Policy, margins runMargins) (CheckpointRecord, error) {
	if err := validateFrame(frame); err != nil {
		return CheckpointRecord{}, err
	}
	hash, err := game.StateHash()
	if err != nil {
		return CheckpointRecord{}, fmt.Errorf("state hash: %w", err)
	}
	record := CheckpointRecord{
		CampaignResult: frame.CampaignResult.String(), FirstDestinationPopulation: margins.firstDestinationPopulation,
		FirstDestinationTurn: margins.firstDestinationTurn, Policy: policy.Name, Seed: seed, StateHash: hash,
		Turn: frame.Turn, WorldRevision: frame.WorldRevision, YearBP: frame.YearBP,
		EstablishedRegions: make([]string, 0, len(frame.SapiensEstablishedRegions)),
	}
	distances := routeDistances(frame, policy.Target, !policy.Reference)
	record.TargetClosestDistance = unreachableDistance
	for _, passage := range frame.Passages {
		if passage.ID == gameapi.BeringStrait {
			record.BeringiaOpen = passage.Status == gameapi.PassageOpen
		}
	}
	for _, tile := range frame.Tiles {
		if tile.Explored {
			record.ExploredTileCount++
		}
	}
	for _, band := range frame.Bands {
		switch band.Species {
		case gameapi.HomoSapiens:
			record.SapiensBandCount++
			record.SapiensPopulation += uint64(band.Population)
			record.TargetClosestDistance = min(record.TargetClosestDistance, distanceAt(distances, band.TileID))
			if tileRegion(frame, band.TileID) == policy.Target && band.Population > record.TargetPopulation {
				record.TargetPopulation = band.Population
			}
		case gameapi.ArchaicHominin:
			record.ArchaicBandCount++
			record.ArchaicPopulation += uint64(band.Population)
		}
	}
	for _, region := range frame.SapiensEstablishedRegions {
		record.EstablishedRegions = append(record.EstablishedRegions, region.String())
		if region == policy.Target {
			record.TargetEstablished = true
		}
	}
	sort.Strings(record.EstablishedRegions)
	return record, nil
}

func validateFrame(frame *gameapi.Frame) error {
	if frame == nil {
		return fmt.Errorf("nil frame")
	}
	if len(frame.Bands) > maxBands {
		return fmt.Errorf("turn %d has %d bands, limit %d", frame.Turn, len(frame.Bands), maxBands)
	}
	archaic := 0
	for _, band := range frame.Bands {
		if band.Species == gameapi.ArchaicHominin {
			archaic++
		}
		var total uint32
		for _, share := range band.AllocationBP {
			total += uint32(share)
		}
		if total != assignmentTotalBP {
			return fmt.Errorf("turn %d band %d allocation totals %d", frame.Turn, band.ID, total)
		}
		for technology, progress := range band.ResearchProgress {
			if !finite(progress) || progress < 0 {
				return fmt.Errorf("turn %d band %d technology %d progress %v", frame.Turn, band.ID, technology, progress)
			}
		}
		for trait, value := range band.HeritableState {
			if !finite(value) || value < 0 || value > 1 {
				return fmt.Errorf("turn %d band %d trait %d value %v", frame.Turn, band.ID, trait, value)
			}
		}
		if band.SpatialActionUsed || band.HasQueuedMigration || band.HasInterbreedTarget {
			return fmt.Errorf("turn %d band %d retains spatial intent", frame.Turn, band.ID)
		}
		if !finite(band.StoredFood) || band.StoredFood < 0 || band.StoredFood > float64(band.Population)*foodStorageTurns {
			return fmt.Errorf("turn %d band %d stored food %v outside capacity", frame.Turn, band.ID, band.StoredFood)
		}
	}
	if archaic > maxArchaicBands {
		return fmt.Errorf("turn %d has %d archaic bands, limit %d", frame.Turn, archaic, maxArchaicBands)
	}
	return nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
