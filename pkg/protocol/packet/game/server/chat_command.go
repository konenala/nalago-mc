package server

import (
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

//codec:gen
type ChatCommand struct {
	Command            string
	Timestamp          int64
	Salt               int64
	ArgumentSignatures []SignedSignatures
	Offset             int32          `mc:"VarInt"`
	Checksum           int8           `mc:"Byte"`
	Acknowledged       pk.FixedBitSet `mc:"FixedBitSet" size:"20"`
}

func (*ChatCommand) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundChatCommand
}

func init() {
	registerPacket(packetid.ServerboundChatCommand, func() ServerboundPacket {
		return &ChatCommand{}
	})
}
