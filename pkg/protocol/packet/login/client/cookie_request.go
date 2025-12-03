package client

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type LoginCookieRequest struct {
	Key string `mc:"Identifier"`
}

func (*LoginCookieRequest) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundLoginCookieRequest
}

func init() {
	registerPacket(packetid.ClientboundLoginCookieRequest, func() ClientboundPacket {
		return &LoginCookieRequest{}
	})
}
