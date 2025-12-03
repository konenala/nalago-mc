package server

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type ConfigFinishConfiguration struct {
}

func (*ConfigFinishConfiguration) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigFinishConfiguration
}

func init() {
	registerPacket(packetid.ServerboundConfigFinishConfiguration, func() ServerboundPacket {
		return &ConfigFinishConfiguration{}
	})
}
