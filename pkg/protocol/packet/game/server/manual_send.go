package server

import (
	"io"

	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/slot"
)

type SignedSignatures struct {
	Name      string
	Signature []byte
}

// Chat command signed (serverbound) used by player.Command
// Matches go-mc packet id ServerboundChatCommandSigned.
type ChatCommandSigned struct {
	Command            string
	Timestamp          int64
	Salt               int64
	ArgumentSignatures []SignedSignatures
	MessageCount       int32
	Acknowledged       []byte
	Checksum           int8
}

func (*ChatCommandSigned) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundChatCommandSigned
}
func (p *ChatCommandSigned) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p ChatCommandSigned) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.String(p.Command).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Long(p.Timestamp).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Long(p.Salt).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.VarInt(len(p.ArgumentSignatures)).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	for _, a := range p.ArgumentSignatures {
		temp, err = pk.String(a.Name).WriteTo(w)
		n += temp
		if err != nil {
			return
		}
		temp, err = pk.ByteArray(a.Signature).WriteTo(w)
		n += temp
		if err != nil {
			return
		}
	}

	temp, err = pk.VarInt(p.MessageCount).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.ByteArray(p.Acknowledged).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Byte(p.Checksum).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

// Plain chat (serverbound) for signed/unsigned chat
// Matches go-mc packet id ServerboundChat.
type Chat struct {
	Message      string
	Timestamp    int64
	Salt         int64
	HasSignature bool
	Signature    []byte
	Offset       int32
	Checksum     int8
	Acknowledged []byte
}

func (*Chat) PacketID() packetid.ServerboundPacketID { return packetid.ServerboundChat }
func (p *Chat) ReadFrom(r io.Reader) (int64, error)  { return 0, nil }
func (p Chat) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.String(p.Message).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Long(p.Timestamp).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Long(p.Salt).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Boolean(p.HasSignature).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	if p.HasSignature {
		temp, err = pk.ByteArray(p.Signature).WriteTo(w)
		n += temp
		if err != nil {
			return
		}
	}
	temp, err = pk.VarInt(p.Offset).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Byte(p.Checksum).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.ByteArray(p.Acknowledged).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

// ChatAck acknowledges seen messages (used when pending >64)
type ChatAck struct {
	MessageCount int32
}

func (*ChatAck) PacketID() packetid.ServerboundPacketID { return packetid.ServerboundChatAck }
func (p *ChatAck) ReadFrom(r io.Reader) (int64, error)  { return 0, nil }
func (p ChatAck) WriteTo(w io.Writer) (n int64, err error) {
	return pk.VarInt(p.MessageCount).WriteTo(w)
}

// Movement packets

type MovePlayerPos struct {
	X, FeetY, Z float64
	Flags       byte
}

func (*MovePlayerPos) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundMovePlayerPos
}
func (p *MovePlayerPos) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p MovePlayerPos) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.Double(p.X).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Double(p.FeetY).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Double(p.Z).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Byte(p.Flags).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

type MovePlayerPosRot struct {
	X, FeetY, Z float64
	Yaw, Pitch  float32
	Flags       byte
}

func (*MovePlayerPosRot) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundMovePlayerPosRot
}
func (p *MovePlayerPosRot) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p MovePlayerPosRot) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.Double(p.X).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Double(p.FeetY).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Double(p.Z).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.Yaw).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.Pitch).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Byte(p.Flags).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

type MovePlayerRot struct {
	Yaw, Pitch float32
	Flags      byte
}

func (*MovePlayerRot) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundMovePlayerRot
}
func (p *MovePlayerRot) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p MovePlayerRot) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.Float(p.Yaw).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.Pitch).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Byte(p.Flags).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

// PlayerAction used for digging

type PlayerAction struct {
	Status   int32
	Sequence int32
	Location pk.Position
	Face     int32
}

func (*PlayerAction) PacketID() packetid.ServerboundPacketID { return packetid.ServerboundPlayerAction }
func (p *PlayerAction) ReadFrom(r io.Reader) (int64, error)  { return 0, nil }
func (p PlayerAction) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.VarInt(p.Status).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.VarInt(p.Sequence).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = p.Location.WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.VarInt(p.Face).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

// UseItemOn

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

func (*UseItemOn) PacketID() packetid.ServerboundPacketID { return packetid.ServerboundUseItemOn }
func (p *UseItemOn) ReadFrom(r io.Reader) (int64, error)  { return 0, nil }
func (p UseItemOn) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.VarInt(p.Hand).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = p.Location.WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.VarInt(p.Face).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.CursorX).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.CursorY).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.CursorZ).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Boolean(p.InsideBlock).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Boolean(p.WorldBorderHit).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.VarInt(p.Sequence).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

// UseItem (hand swing / use)
type UseItem struct {
	Hand     int32
	Sequence int32
	Yaw      float32
	Pitch    float32
}

func (*UseItem) PacketID() packetid.ServerboundPacketID { return packetid.ServerboundUseItem }
func (p *UseItem) ReadFrom(r io.Reader) (int64, error)  { return 0, nil }
func (p UseItem) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.VarInt(p.Hand).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.VarInt(p.Sequence).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.Yaw).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Float(p.Pitch).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

// Inventory actions

type ContainerClick struct {
	WindowID    int32
	StateID     int32
	Slot        int16
	Button      int8
	Mode        int32
	CarriedItem slot.Slot
}

func (*ContainerClick) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundContainerClick
}
func (p *ContainerClick) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p ContainerClick) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
	temp, err = pk.Byte(p.WindowID).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.VarInt(p.StateID).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Short(p.Slot).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.Byte(p.Button).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = pk.VarInt(p.Mode).WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	temp, err = p.CarriedItem.WriteTo(w)
	n += temp
	if err != nil {
		return
	}
	return
}

type ContainerClose struct {
	WindowID int32
}

func (*ContainerClose) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundContainerClose
}
func (p *ContainerClose) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p ContainerClose) WriteTo(w io.Writer) (n int64, err error) {
	return pk.Byte(p.WindowID).WriteTo(w)
}

// KeepAlive

type KeepAlive struct {
	ID int64
}

func (*KeepAlive) PacketID() packetid.ServerboundPacketID { return packetid.ServerboundKeepAlive }
func (p *KeepAlive) ReadFrom(r io.Reader) (int64, error)  { return 0, nil }
func (p KeepAlive) WriteTo(w io.Writer) (n int64, err error) {
	return pk.Long(p.ID).WriteTo(w)
}
