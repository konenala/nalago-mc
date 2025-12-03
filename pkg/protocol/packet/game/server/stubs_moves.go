package server

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

type MovePlayerPosRot struct {
	X, FeetY, Z float64
	Yaw, Pitch  float32
	Flags       byte
}

func (*MovePlayerPosRot) PacketID() packetid.ServerboundPacketID { return 0 }
func (*MovePlayerPosRot) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (MovePlayerPosRot) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type MovePlayerPos struct {
	X, FeetY, Z float64
	Flags       byte
}

func (*MovePlayerPos) PacketID() packetid.ServerboundPacketID { return 0 }
func (*MovePlayerPos) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (MovePlayerPos) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type MovePlayerRot struct {
	Yaw, Pitch float32
	Flags      byte
}

func (*MovePlayerRot) PacketID() packetid.ServerboundPacketID { return 0 }
func (*MovePlayerRot) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (MovePlayerRot) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type PlayerAction struct {
	Status   int32
	Sequence int32
	Location pk.Position
	Face     int32
}

func (*PlayerAction) PacketID() packetid.ServerboundPacketID { return 0 }
func (*PlayerAction) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (PlayerAction) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type UseItemOn struct {
	Hand           int32
	Location       pk.Position
	Face           int32
	CursorX        float32
	CursorY        float32
	CursorZ        float32
	InsideBlock    bool
	WorldBorderHit bool
	Sequence       int32
}

func (*UseItemOn) PacketID() packetid.ServerboundPacketID { return 0 }
func (*UseItemOn) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (UseItemOn) WriteTo(io.Writer) (int64, error)        { return 0, nil }
