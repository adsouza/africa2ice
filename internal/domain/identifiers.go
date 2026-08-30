package domain

type BandID uint64
type TileID uint16
type RegionID uint8
type PassageID uint8

const InvalidTileID TileID = ^TileID(0)
