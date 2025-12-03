package client

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type ConfigFinishConfiguration struct {
}

func (*ConfigFinishConfiguration) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigFinishConfiguration
}

func init() {
	registerPacket(packetid.ClientboundConfigFinishConfiguration, func() ClientboundPacket {
		return &ConfigFinishConfiguration{}
	})
}
