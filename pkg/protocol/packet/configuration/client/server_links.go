package client

import (
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
)

type ConfigServerLinks struct {
	client.ServerLinks
}

func (*ConfigServerLinks) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigServerLinks
}

func init() {
	registerPacket(packetid.ClientboundConfigServerLinks, func() ClientboundPacket {
		return &ConfigServerLinks{}
	})
}
