package server

import (
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

//codec:gen
type Chat struct {
	Message      string
	Timestamp    int64
	Salt         int64
	HasSignature bool
	//opt:optional:HasSignature
	Signature    []byte `mc:"ByteArray"`
	Offset       int32  `mc:"VarInt"`
	Checksum     int8   `mc:"Byte"`
	Acknowledged []byte
}

func (*Chat) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundChat
}

func init() {
	registerPacket(packetid.ServerboundChat, func() ServerboundPacket {
		return &Chat{}
	})
}
