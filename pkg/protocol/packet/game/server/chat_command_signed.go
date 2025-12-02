package server

import (
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

//codec:gen
type SignedSignatures struct {
	ArgumentName string
	Signature    []byte `mc:"ByteArray"`
}

//codec:gen
type ChatCommandSigned struct {
	Command            string
	Timestamp          int64
	Salt               int64
	ArgumentSignatures []SignedSignatures
	Offset             int32 `mc:"VarInt"`
	Checksum           int8  `mc:"Byte"`
	Acknowledged       []byte
}

func (*ChatCommandSigned) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundChatCommandSigned
}

func init() {
	registerPacket(packetid.ServerboundChatCommandSigned, func() ServerboundPacket {
		return &ChatCommandSigned{}
	})
}
