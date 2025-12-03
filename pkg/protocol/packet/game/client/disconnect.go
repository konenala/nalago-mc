package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/chat"
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

type Disconnect struct {
	Reason chat.Message
}

func (*Disconnect) PacketID() packetid.ClientboundPacketID { return 0 }
func (*Disconnect) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (Disconnect) WriteTo(io.Writer) (int64, error)        { return 0, nil }
