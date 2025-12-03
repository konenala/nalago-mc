package server

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type LoginCookieResponse struct {
	Key        string
	HasPayload bool
	//opt:optional:HasPayload
	Payload []byte `mc:"ByteArray"`
}

func (*LoginCookieResponse) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundLoginCookieResponse
}

func init() {
	registerPacket(packetid.ServerboundLoginCookieResponse, func() ServerboundPacket {
		return &LoginCookieResponse{}
	})
}
