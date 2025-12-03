package client

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type ConfigPing struct {
	ID int32
}

func (*ConfigPing) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigPing
}

func init() {
	registerPacket(packetid.ClientboundConfigPing, func() ClientboundPacket {
		return &ConfigPing{}
	})
}
