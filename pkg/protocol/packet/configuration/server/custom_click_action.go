package server

import (
	"git.konjactw.dev/falloutBot/go-mc/nbt"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
)

//codec:gen
type ConfigCustomClickAction struct {
	Action string         `mc:"Identifier"`
	Data   nbt.RawMessage `mc:"NBT"`
}

func (*ConfigCustomClickAction) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigCustomClickAction
}

func init() {
	registerPacket(packetid.ServerboundConfigCustomClickAction, func() ServerboundPacket {
		return &ConfigCustomClickAction{}
	})
}
