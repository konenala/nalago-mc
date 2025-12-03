package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

// PlayerPosition stub used by player.go
type PlayerPosition struct {
	X, Y, Z float64
	YRot    float32
	XRot    float32
	Flags   byte
	ID      int32
}

func (*PlayerPosition) PacketID() packetid.ClientboundPacketID { return 0 }
func (*PlayerPosition) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (PlayerPosition) WriteTo(io.Writer) (int64, error)        { return 0, nil }
