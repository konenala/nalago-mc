package client

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type ConfigCustomPayload struct {
	Channel string `mc:"Identifier"`
	Data    []byte `mc:"ByteArray"`
}

func (*ConfigCustomPayload) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigCustomPayload
}

func init() {
	registerPacket(packetid.ClientboundConfigCustomPayload, func() ClientboundPacket {
		return &ConfigCustomPayload{}
	})
}
