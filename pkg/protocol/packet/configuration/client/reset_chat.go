package client

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type ConfigResetChat struct {
}

func (*ConfigResetChat) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigResetChat
}

func init() {
	registerPacket(packetid.ClientboundConfigResetChat, func() ClientboundPacket {
		return &ConfigResetChat{}
	})
}
