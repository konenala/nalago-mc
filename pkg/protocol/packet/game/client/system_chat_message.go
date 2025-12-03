package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/chat"
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

// SystemChatMessage matches code expectations (wrapper around generated SystemChat).
type SystemChatMessage struct {
	Content chat.Message
	Overlay bool
}

func (*SystemChatMessage) PacketID() packetid.ClientboundPacketID { return 0 }
func (*SystemChatMessage) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (SystemChatMessage) WriteTo(io.Writer) (int64, error)        { return 0, nil }
