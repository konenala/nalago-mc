package server

import (
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

//codec:gen
type SignedSignatures struct {
	ArgumentName string
	Signature    []byte
}

//codec:gen
type ChatCommandSigned struct {
	Command            string
	Timestamp          int64
	Salt               int64
	ArgumentSignatures []SignedSignatures
	MessageCount       int32 `mc:"VarInt"`
	Acknowledged       []byte
	Checksum           int8 `mc:"Byte"`
}

func (*ChatCommandSigned) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundChatCommandSigned
}

func init() {
	registerPacket(packetid.ServerboundChatCommandSigned, func() ServerboundPacket {
		return &ChatCommandSigned{}
	})
}
