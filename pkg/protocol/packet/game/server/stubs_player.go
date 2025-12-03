package server

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

type ChatAck struct {
	MessageCount int32
}

func (*ChatAck) PacketID() packetid.ServerboundPacketID { return 0 }
func (*ChatAck) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (ChatAck) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type AcceptTeleportation struct {
	TeleportID int32
}

func (*AcceptTeleportation) PacketID() packetid.ServerboundPacketID { return 0 }
func (*AcceptTeleportation) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (AcceptTeleportation) WriteTo(io.Writer) (int64, error)        { return 0, nil }
