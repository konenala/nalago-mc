package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/chat"
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// PlayerChat is used by player.go for signed chat tracking.
type PlayerChat struct {
	GlobalIndex         int32
	SenderUuid          pk.UUID
	Index               int32
	Signature           []byte
	HasSignature        bool
	PlainMessage        string
	Timestamp           int64
	Salt                int64
	PreviousMessages    pk.ByteArray
	UnsignedChatContent pk.ByteArray
	FilterType          int32
	FilterTypeMask      pk.ByteArray
	FieldType           pk.ByteArray
	NetworkName         pk.NBTField
	NetworkTargetName   pk.NBTField
}

func (*PlayerChat) PacketID() packetid.ClientboundPacketID      { return packetid.ClientboundPlayerChat }
func (p *PlayerChat) ReadFrom(r io.Reader) (n int64, err error) { return 0, nil }
func (p PlayerChat) WriteTo(w io.Writer) (n int64, err error)   { return 0, nil }

// SystemChatMessage is used for overlay/system chat display.
type SystemChatMessage struct {
	Content chat.Message
	Overlay bool
}

func (*SystemChatMessage) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundSystemChat
}
func (p *SystemChatMessage) ReadFrom(r io.Reader) (n int64, err error) { return 0, nil }
func (p SystemChatMessage) WriteTo(w io.Writer) (n int64, err error)   { return 0, nil }

// Disconnect wraps kick reason as chat.Message for player.go
type Disconnect struct {
	Reason chat.Message
}

func (*Disconnect) PacketID() packetid.ClientboundPacketID      { return packetid.ClientboundDisconnect }
func (p *Disconnect) ReadFrom(r io.Reader) (n int64, err error) { return 0, nil }
func (p Disconnect) WriteTo(w io.Writer) (n int64, err error)   { return 0, nil }

func init() {
	registerPacket(packetid.ClientboundPlayerChat, func() ClientboundPacket { return &PlayerChat{} })
	registerPacket(packetid.ClientboundSystemChat, func() ClientboundPacket { return &SystemChatMessage{} })
	registerPacket(packetid.ClientboundDisconnect, func() ClientboundPacket { return &Disconnect{} })
}
