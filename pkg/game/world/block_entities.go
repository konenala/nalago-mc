package world

import (
	"git.konjactw.dev/falloutBot/go-mc/level"
)

// ParsedBlockEntity is a wrapper around level.BlockEntity that contains
// the parsed NBT data in a structured format.
type ParsedBlockEntity struct {
	level.BlockEntity
	ParsedData level.BlockEntityData
}
