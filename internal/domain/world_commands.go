package domain

func (world *World) SetAssignment(id BandID, allocation [AssignmentCount]AssignmentBP, player bool) error {
	if world.result != CampaignOngoing {
		return ErrCampaignComplete
	}
	index := world.bandIndex(id)
	if index < 0 {
		return ErrBandNotFound
	}
	if player && world.bands[index].Species != HomoSapiens {
		return ErrComputerControlledBand
	}
	return world.bands[index].SetAssignment(allocation)
}

func (world *World) Research(id BandID, technology Technology, player bool) error {
	if world.result != CampaignOngoing {
		return ErrCampaignComplete
	}
	index := world.bandIndex(id)
	if index < 0 {
		return ErrBandNotFound
	}
	if player && world.bands[index].Species != HomoSapiens {
		return ErrComputerControlledBand
	}
	return world.bands[index].Technology.Select(technology)
}

func (world *World) QueueMigration(id BandID, destination TileID, player bool) error {
	if world.result != CampaignOngoing {
		return ErrCampaignComplete
	}
	index := world.bandIndex(id)
	if index < 0 {
		return ErrBandNotFound
	}
	band := &world.bands[index]
	if player && band.Species != HomoSapiens {
		return ErrComputerControlledBand
	}
	if band.SpatialActionUsed {
		return ErrSpatialActionUsed
	}
	if player && !world.IsExplored(destination) {
		return ErrInvalidMigration
	}
	if destination >= TileCount || world.habitat[destination].BaselineK <= 0 {
		return ErrInvalidMigration
	}
	valid := false
	for _, edge := range world.grid.OrdinaryEdges(band.TileID) {
		if edge.To == destination {
			valid = true
			break
		}
	}
	usesPassage := false
	passageID := PassageID(0)
	if !valid {
		passage, ok := passageForEdge(band.TileID, destination)
		if !ok || passageAvailability(*band, passage, world.habitat, world.climate.LongTermTempOffset, false) != PassageAvailable {
			return ErrInvalidMigration
		}
		usesPassage, passageID = true, passage.ID
	}
	band.QueuedOrigin, band.QueuedMigration = band.TileID, destination
	band.QueuedPassage, band.QueueUsesPassage = passageID, usesPassage
	band.HasQueuedMigration, band.SpatialActionUsed = true, true
	return nil
}

func (world *World) Split(id BandID, destination TileID, player bool) error {
	if world.result != CampaignOngoing {
		return ErrCampaignComplete
	}
	index := world.bandIndex(id)
	if index < 0 {
		return ErrBandNotFound
	}
	band := world.bands[index]
	if player && band.Species != HomoSapiens {
		return ErrComputerControlledBand
	}
	if band.SpatialActionUsed {
		return ErrSpatialActionUsed
	}
	if len(world.bands) >= MaxBands {
		return ErrBandLimitReached
	}
	if world.nextBandID == 0 || world.nextBandID == ^BandID(0) {
		return ErrBandIDExhausted
	}
	if world.BandStress(id) <= SplitStressThreshold {
		return ErrSplitStressTooLow
	}
	adjacent := false
	for _, edge := range world.grid.OrdinaryEdges(band.TileID) {
		if edge.To == destination {
			adjacent = true
			break
		}
	}
	if !adjacent {
		return ErrSplitDestinationNotAdjacent
	}
	// Checked before the terrain is inspected, and enforced here rather than
	// left to the caller: Split reveals its destination, so without this the
	// aggregate would hand a player the fog-of-war gate QueueMigration applies.
	if player && !world.IsExplored(destination) {
		return ErrSplitDestinationUnexplored
	}
	if destination >= TileCount || world.habitat[destination].BaselineK <= 0 {
		return ErrSplitDestinationUninhabitable
	}
	left, right, err := splitBand(band, world.nextBandID)
	if err != nil {
		return err
	}
	right.TileID = destination
	world.bands[index] = left
	world.bands = append(world.bands, right)
	world.nextBandID++
	world.revealFromSapiens()
	return nil
}

func (world *World) Interbreed(id, target BandID, player bool) error {
	if world.result != CampaignOngoing {
		return ErrCampaignComplete
	}
	index, targetIndex := world.bandIndex(id), world.bandIndex(target)
	if index < 0 || targetIndex < 0 {
		return ErrBandNotFound
	}
	actor, other := &world.bands[index], world.bands[targetIndex]
	if player && actor.Species != HomoSapiens {
		return ErrComputerControlledBand
	}
	if actor.SpatialActionUsed {
		return ErrSpatialActionUsed
	}
	if actor.Species != HomoSapiens || other.Species != ArchaicHominin || actor.TileID != other.TileID || actor.ID == other.ID {
		return ErrInvalidInterbreedTarget
	}
	actor.InterbreedTarget, actor.HasInterbreedTarget, actor.SpatialActionUsed = target, true, true
	return nil
}

func (world *World) BandStress(id BandID) float64 {
	index := world.bandIndex(id)
	if index < 0 {
		return 0
	}
	band := world.bands[index]
	habitat := world.habitat[band.TileID]
	denominator := float64(habitat.BaselineK * band.Technology.CapacityMultiplier())
	if denominator <= 0 {
		return 0
	}
	var total uint64
	for _, resident := range world.bands {
		if resident.TileID == band.TileID {
			total += uint64(resident.Population)
		}
	}
	return float64(total) / denominator
}

func (world *World) PassageStatus(id BandID, passageID PassageID) PassageAvailability {
	index := world.bandIndex(id)
	if index < 0 || passageID >= PassageCount {
		return PassageNotAtEndpoint
	}
	return passageAvailability(world.bands[index], passageCatalog[passageID], world.habitat, world.climate.LongTermTempOffset, true)
}
