package server

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type LoginCustomQueryAnswer struct {
	MessageID int32 `mc:"VarInt"`
	HasData   bool
	//opt:optional:HasData
	Data []byte `mc:"ByteArray"`
}

func (*LoginCustomQueryAnswer) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundLoginCustomQueryAnswer
}

func init() {
	registerPacket(packetid.ServerboundLoginCustomQueryAnswer, func() ServerboundPacket {
		return &LoginCustomQueryAnswer{}
	})
}
