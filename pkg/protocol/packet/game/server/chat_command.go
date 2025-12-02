package server

import (
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

//codec:gen
type ChatCommand struct {
	Command            string
	Timestamp          int64
	Salt               int64
	ArgumentSignatures []SignedSignatures
	Offset             int32 `mc:"VarInt"`
	Checksum           int8  `mc:"Byte"`
	Acknowledged       []byte
}

func (*ChatCommand) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundChatCommand
}

func init() {
	registerPacket(packetid.ServerboundChatCommand, func() ServerboundPacket {
		return &ChatCommand{}
	})
}
