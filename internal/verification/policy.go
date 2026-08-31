// Package verification drives deterministic reference campaigns through the
// public game port. It deliberately knows nothing about domain types.
package verification

import (
	"container/heap"
	"fmt"
	"sort"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const (
	minSplitPopulation               = 40
	referenceDestinationReserve      = 50
	verificationSplitStressThreshold = 0.67
)

// Policy is one of the five release-verification routes. The reference route
// covers Frangistan; the other four exercise the longer destination paths.
type Policy struct {
	Name      string
	Target    gameapi.Region
	Reference bool
}

var policies = [...]Policy{
	{Name: "reference", Target: gameapi.Frangistan, Reference: true},
	{Name: "toward-south-asia", Target: gameapi.SouthAsia},
	{Name: "toward-yellow-river", Target: gameapi.YellowRiverBasin},
	{Name: "toward-sahul", Target: gameapi.Sahul},
	{Name: "toward-beringia", Target: gameapi.Beringia},
}

// PolicyNames returns the accepted CLI names in stable order.
func PolicyNames() []string {
	result := make([]string, len(policies))
	for index, policy := range policies {
		result[index] = policy.Name
	}
	return result
}

// ParsePolicy resolves a stable CLI name.
func ParsePolicy(name string) (Policy, error) {
	for _, policy := range policies {
		if name == policy.Name {
			return policy, nil
		}
	}
	return Policy{}, fmt.Errorf("unknown policy %q (choose %s)", name, strings.Join(PolicyNames(), ", "))
}

func researchCommands(frame *gameapi.Frame, policy Policy) []gameapi.Command {
	bands := append([]gameapi.Band(nil), frame.Bands...)
	sort.Slice(bands, func(i, j int) bool { return bands[i].ID < bands[j].ID })
	commands := make([]gameapi.Command, 0, len(bands))
	for _, band := range bands {
		if band.Species != gameapi.HomoSapiens || band.HasResearchTarget {
			continue
		}
		if technology, ok := nextResearch(band, policy.Target); ok {
			commands = append(commands, gameapi.ResearchTech{BandID: band.ID, Tech: technology})
		}
	}
	return commands
}

func spatialCommands(frame *gameapi.Frame, policy Policy) []gameapi.Command {
	bands := append([]gameapi.Band(nil), frame.Bands...)
	sort.Slice(bands, func(i, j int) bool { return bands[i].ID < bands[j].ID })
	commands := make([]gameapi.Command, 0, len(bands))
	distances := routeDistances(frame, policy.Target, !policy.Reference)
	leader, hasLeader := routeLeader(bands, distances)
	if !hasLeader {
		return commands
	}
	for _, band := range bands {
		if band.Species != gameapi.HomoSapiens || band.SpatialActionUsed {
			continue
		}
		if band.ID != leader.ID {
			if policy.Reference && band.Population >= minSplitPopulation && band.Stress > verificationSplitStressThreshold {
				if candidate, ok := mostAttractiveOrdinary(band.MigrationCandidates); ok {
					commands = append(commands, gameapi.SplitBand{BandID: band.ID, Destination: candidate.TileID})
				}
			}
			continue
		}
		if candidate, ok := routeChoice(frame, band, policy, distances); ok {
			commands = append(commands, gameapi.QueueMigration{BandID: band.ID, TileID: candidate.TileID})
		}
	}
	return commands
}

func nextResearch(band gameapi.Band, target gameapi.Region) (gameapi.Tech, bool) {
	priority := [...]gameapi.Tech{
		gameapi.Firecraft, gameapi.HaftedTools, gameapi.Campcraft,
		gameapi.PlantKnowledge, gameapi.MedicinalKnowledge,
		gameapi.TailoredClothing, gameapi.CordageAndNets,
		gameapi.Trapping, gameapi.CoastalNavigation,
	}
	if target == gameapi.Sahul {
		priority = [...]gameapi.Tech{
			gameapi.Firecraft, gameapi.HaftedTools, gameapi.CordageAndNets,
			gameapi.CoastalNavigation, gameapi.Campcraft, gameapi.PlantKnowledge,
			gameapi.MedicinalKnowledge, gameapi.TailoredClothing, gameapi.Trapping,
		}
	}
	for _, technology := range priority {
		if band.ResearchOptions[technology].Available {
			return technology, true
		}
	}
	return 0, false
}

func routeLeader(bands []gameapi.Band, distances []int) (gameapi.Band, bool) {
	best := gameapi.Band{}
	bestDistance := unreachableDistance
	found := false
	for _, band := range bands {
		if band.Species != gameapi.HomoSapiens {
			continue
		}
		distance := distanceAt(distances, band.TileID)
		if !found || distance < bestDistance || distance == bestDistance &&
			(band.Population > best.Population || band.Population == best.Population && band.ID < best.ID) {
			best, bestDistance, found = band, distance, true
		}
	}
	return best, found
}

const unreachableDistance = int(^uint(0) >> 1)

func routeDistances(frame *gameapi.Frame, target gameapi.Region, habitatWeighted bool) []int {
	distances := make([]int, len(frame.Tiles))
	for index := range distances {
		distances[index] = unreachableDistance
	}
	byCoordinate := make(map[[2]int]gameapi.TileID, len(frame.Tiles))
	queue := &routeQueue{}
	for _, tile := range frame.Tiles {
		if !tile.Land || tile.EcologicalK <= 0 {
			continue
		}
		byCoordinate[[2]int{tile.X, tile.Y}] = tile.ID
		if tile.Region == target {
			distances[int(tile.ID)] = 0
			heap.Push(queue, routeNode{tile: tile.ID})
		}
	}
	passagePeers := make(map[gameapi.TileID][]gameapi.TileID, len(frame.Passages)*2)
	for _, passage := range frame.Passages {
		passagePeers[passage.From] = append(passagePeers[passage.From], passage.To)
		passagePeers[passage.To] = append(passagePeers[passage.To], passage.From)
	}
	for queue.Len() > 0 {
		node := heap.Pop(queue).(routeNode)
		current := node.tile
		if int(current) >= len(frame.Tiles) {
			continue
		}
		if node.cost != distances[int(current)] {
			continue
		}
		tile := frame.Tiles[current]
		neighbors := passagePeers[current]
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				neighbor, ok := byCoordinate[[2]int{tile.X + dx, tile.Y + dy}]
				if !ok {
					continue
				}
				if dx != 0 && dy != 0 {
					_, horizontal := byCoordinate[[2]int{tile.X + dx, tile.Y}]
					_, vertical := byCoordinate[[2]int{tile.X, tile.Y + dy}]
					if !horizontal || !vertical {
						continue
					}
				}
				neighbors = append(neighbors, neighbor)
			}
		}
		for _, neighbor := range neighbors {
			if int(neighbor) >= len(distances) {
				continue
			}
			step := 1
			if habitatWeighted {
				step = habitatStepCost(frame.Tiles[current].EcologicalK)
			}
			cost := distances[int(current)] + step
			if cost >= distances[int(neighbor)] {
				continue
			}
			distances[int(neighbor)] = cost
			heap.Push(queue, routeNode{tile: neighbor, cost: cost})
		}
	}
	return distances
}

func habitatStepCost(capacity float64) int {
	switch {
	case capacity >= 100:
		return 1
	case capacity >= 75:
		return 2
	case capacity >= 50:
		return 4
	case capacity >= 25:
		return 8
	case capacity >= 10:
		return 16
	default:
		return 32
	}
}

type routeNode struct {
	tile gameapi.TileID
	cost int
}

type routeQueue []routeNode

func (queue routeQueue) Len() int { return len(queue) }
func (queue routeQueue) Less(i, j int) bool {
	return queue[i].cost < queue[j].cost || queue[i].cost == queue[j].cost && queue[i].tile < queue[j].tile
}
func (queue routeQueue) Swap(i, j int)   { queue[i], queue[j] = queue[j], queue[i] }
func (queue *routeQueue) Push(value any) { *queue = append(*queue, value.(routeNode)) }
func (queue *routeQueue) Pop() any {
	old := *queue
	last := old[len(old)-1]
	*queue = old[:len(old)-1]
	return last
}

func distanceAt(distances []int, tile gameapi.TileID) int {
	if int(tile) >= len(distances) {
		return unreachableDistance
	}
	return distances[int(tile)]
}

func routeChoice(frame *gameapi.Frame, band gameapi.Band, policy Policy, distances []int) (gameapi.MigrationCandidate, bool) {
	currentDistance := distanceAt(distances, band.TileID)
	if currentDistance == 0 || currentDistance == unreachableDistance {
		return gameapi.MigrationCandidate{}, false
	}
	best := gameapi.MigrationCandidate{}
	bestDistance := unreachableDistance
	found := false
	for _, candidate := range band.MigrationCandidates {
		if candidate.EcologicalK <= 0 {
			continue
		}
		distance := distanceAt(distances, candidate.TileID)
		if distance >= currentDistance {
			continue
		}
		if policy.Reference && band.Population < referenceDestinationReserve && tileRegion(frame, candidate.TileID) == policy.Target {
			continue
		}
		if !found || distance < bestDistance || distance == bestDistance && betterCandidate(candidate, best) {
			best, bestDistance, found = candidate, distance, true
		}
	}
	return best, found
}

func tileRegion(frame *gameapi.Frame, id gameapi.TileID) gameapi.Region {
	if int(id) >= len(frame.Tiles) {
		return gameapi.RegionCount
	}
	return frame.Tiles[id].Region
}

func mostAttractiveOrdinary(candidates []gameapi.MigrationCandidate) (gameapi.MigrationCandidate, bool) {
	best := gameapi.MigrationCandidate{}
	found := false
	for _, candidate := range candidates {
		if candidate.RequiresPassage || candidate.EcologicalK <= 0 {
			continue
		}
		if !found || betterCandidate(candidate, best) {
			best, found = candidate, true
		}
	}
	return best, found
}

func betterCandidate(candidate, incumbent gameapi.MigrationCandidate) bool {
	if candidate.Attraction != incumbent.Attraction {
		return candidate.Attraction > incumbent.Attraction
	}
	return candidate.TileID < incumbent.TileID
}
