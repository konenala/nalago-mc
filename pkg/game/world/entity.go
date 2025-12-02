package world

import (
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"

	prismarineEntity "github.com/konjacbot/prismarine-go/entity"
	prismarineWorld "github.com/konjacbot/prismarine-go/world"

	"git.konjactw.dev/patyhank/minego/pkg/protocol/metadata"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/slot"
)

// Entity wraps prismarine-go Entity with protocol-specific data
type Entity struct {
	*prismarineEntity.Entity // Embed L3 entity
	metadata                 map[uint8]metadata.Metadata
	equipment                map[int8]slot.Slot
}

// Converter functions between protocol and L3 types
func vec3ToVec3d(v mgl64.Vec3) prismarineWorld.Vec3d {
	return prismarineWorld.Vec3d{X: v.X(), Y: v.Y(), Z: v.Z()}
}

func vec3dToVec3(v prismarineWorld.Vec3d) mgl64.Vec3 {
	return mgl64.Vec3{v.X, v.Y, v.Z}
}

func vec2ToVec2(v mgl64.Vec2) prismarineWorld.Vec2 {
	return prismarineWorld.Vec2{X: float32(v.X()), Y: float32(v.Y())}
}

func vec2ToVec2d(v prismarineWorld.Vec2) mgl64.Vec2 {
	return mgl64.Vec2{float64(v.X), float64(v.Y)}
}

// NewEntity creates a new Entity wrapping a prismarine-go Entity
func NewEntity(eid int32, uuid uuid.UUID, entityType int32, pos mgl64.Vec3, rot mgl64.Vec2) *Entity {
	return &Entity{
		Entity: &prismarineEntity.Entity{
			EID:      eid,
			UUID:     [16]byte(uuid),
			Type:     entityType,
			Position: vec3ToVec3d(pos),
			Rotation: vec2ToVec2(rot),
			Metadata: make(map[uint8]interface{}),
		},
		metadata:  make(map[uint8]metadata.Metadata),
		equipment: make(map[int8]slot.Slot),
	}
}

// bot.Entity interface implementation - returns protocol types
func (e *Entity) ID() int32 {
	return e.Entity.EID
}

func (e *Entity) UUID() uuid.UUID {
	return uuid.UUID(e.Entity.UUID)
}

func (e *Entity) Type() int32 {
	return e.Entity.Type
}

func (e *Entity) Position() mgl64.Vec3 {
	return vec3dToVec3(e.Entity.Position)
}

func (e *Entity) Rotation() mgl64.Vec2 {
	return vec2ToVec2d(e.Entity.Rotation)
}

func (e *Entity) Metadata() map[uint8]metadata.Metadata {
	return e.metadata
}

func (e *Entity) Equipment() map[int8]slot.Slot {
	return e.equipment
}

func (e *Entity) SetPosition(pos mgl64.Vec3) {
	e.Entity.Position = vec3ToVec3d(pos)
}

func (e *Entity) SetRotation(rot mgl64.Vec2) {
	e.Entity.Rotation = vec2ToVec2(rot)
}
