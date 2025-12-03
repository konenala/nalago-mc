package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/chat"
	"git.konjactw.dev/falloutBot/go-mc/level"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/metadata"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/slot"
	"github.com/google/uuid"
)

// Extra manual stubs for packets not emitted by generator but referenced in codebase.

type LevelChunkWithLight struct {
	Pos  level.ChunkPos
	Data *level.Chunk
}

func (*LevelChunkWithLight) PacketID() packetid.ClientboundPacketID { return 0 }
func (*LevelChunkWithLight) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (LevelChunkWithLight) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type ForgetLevelChunk struct{ Pos level.ChunkPos }

func (*ForgetLevelChunk) PacketID() packetid.ClientboundPacketID { return 0 }
func (*ForgetLevelChunk) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (ForgetLevelChunk) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type AddEntity struct {
	ID         int32
	UUID       uuid.UUID
	Type       int32
	X, Y, Z    float64
	XRot, YRot float32
}

func (*AddEntity) PacketID() packetid.ClientboundPacketID { return 0 }
func (*AddEntity) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (AddEntity) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type RemoveEntities struct{ EntityIDs []int32 }

func (*RemoveEntities) PacketID() packetid.ClientboundPacketID { return 0 }
func (*RemoveEntities) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (RemoveEntities) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type SetEntityMetadata struct {
	EntityID int32
	Metadata struct{ Data map[uint8]metadata.Metadata }
}

func (*SetEntityMetadata) PacketID() packetid.ClientboundPacketID { return 0 }
func (*SetEntityMetadata) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (SetEntityMetadata) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type EquipmentEntry struct {
	Slot int8
	Item slot.Slot
}

type SetEquipment struct {
	EntityID  int32
	Equipment []EquipmentEntry
}

func (*SetEquipment) PacketID() packetid.ClientboundPacketID { return 0 }
func (*SetEquipment) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (SetEquipment) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type UpdateEntityPosition struct {
	EntityID               int32
	DeltaX, DeltaY, DeltaZ int16
}

func (*UpdateEntityPosition) PacketID() packetid.ClientboundPacketID { return 0 }
func (*UpdateEntityPosition) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (UpdateEntityPosition) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type UpdateEntityRotation struct {
	EntityID   int32
	Yaw, Pitch float32
}

func (*UpdateEntityRotation) PacketID() packetid.ClientboundPacketID { return 0 }
func (*UpdateEntityRotation) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (UpdateEntityRotation) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type UpdateEntityPositionAndRotation struct {
	EntityID               int32
	DeltaX, DeltaY, DeltaZ int16
	Yaw, Pitch             float32
}

func (*UpdateEntityPositionAndRotation) PacketID() packetid.ClientboundPacketID { return 0 }
func (*UpdateEntityPositionAndRotation) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (UpdateEntityPositionAndRotation) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type BlockUpdate struct {
	Position   struct{ X, Y, Z int32 }
	BlockState int32
}

func (*BlockUpdate) PacketID() packetid.ClientboundPacketID { return 0 }
func (*BlockUpdate) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (BlockUpdate) WriteTo(io.Writer) (int64, error)        { return 0, nil }

// UpdateSectionsBlocks minimal stub
type UpdateSectionsBlocks struct{}

func (*UpdateSectionsBlocks) PacketID() packetid.ClientboundPacketID { return 0 }
func (*UpdateSectionsBlocks) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (UpdateSectionsBlocks) WriteTo(io.Writer) (int64, error)        { return 0, nil }
func (UpdateSectionsBlocks) ToSectionPos() (int32, int32, int32)     { return 0, 0, 0 }
func (UpdateSectionsBlocks) ParseBlocks() map[[3]int]int32           { return map[[3]int]int32{} }

// Inventory related
type SetContainerContent struct {
	WindowID int32
	StateID  int32
	Slots    []slot.Slot
}

func (*SetContainerContent) PacketID() packetid.ClientboundPacketID { return 0 }
func (*SetContainerContent) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (SetContainerContent) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type ContainerSetSlot struct {
	ContainerID int32
	StateID     int32
	Slot        int16
	ItemStack   slot.Slot
}

func (*ContainerSetSlot) PacketID() packetid.ClientboundPacketID { return 0 }
func (*ContainerSetSlot) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (ContainerSetSlot) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type CloseContainer struct{ WindowID int32 }

func (*CloseContainer) PacketID() packetid.ClientboundPacketID { return 0 }
func (*CloseContainer) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (CloseContainer) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type OpenScreen struct {
	WindowID    int32
	WindowType  int32
	WindowTitle chat.Message
}

func (*OpenScreen) PacketID() packetid.ClientboundPacketID { return 0 }
func (*OpenScreen) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (OpenScreen) WriteTo(io.Writer) (int64, error)        { return 0, nil }

// Config phase aliases (used by configuration packets referencing game/client)
type CustomReportDetails struct{}
type RemoveResourcePack struct{}
type AddResourcePack struct{ UUID pk.UUID }
type ServerLinks struct{}
type UpdateTags struct{}

func (*CustomReportDetails) PacketID() packetid.ClientboundPacketID { return 0 }
func (*CustomReportDetails) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (CustomReportDetails) WriteTo(io.Writer) (int64, error)        { return 0, nil }
func (*RemoveResourcePack) PacketID() packetid.ClientboundPacketID  { return 0 }
func (*RemoveResourcePack) ReadFrom(io.Reader) (int64, error)       { return 0, nil }
func (RemoveResourcePack) WriteTo(io.Writer) (int64, error)         { return 0, nil }
func (*AddResourcePack) PacketID() packetid.ClientboundPacketID     { return 0 }
func (*AddResourcePack) ReadFrom(io.Reader) (int64, error)          { return 0, nil }
func (AddResourcePack) WriteTo(io.Writer) (int64, error)            { return 0, nil }
func (*ServerLinks) PacketID() packetid.ClientboundPacketID         { return 0 }
func (*ServerLinks) ReadFrom(io.Reader) (int64, error)              { return 0, nil }
func (ServerLinks) WriteTo(io.Writer) (int64, error)                { return 0, nil }
func (*UpdateTags) PacketID() packetid.ClientboundPacketID          { return 0 }
func (*UpdateTags) ReadFrom(io.Reader) (int64, error)               { return 0, nil }
func (UpdateTags) WriteTo(io.Writer) (int64, error)                 { return 0, nil }
