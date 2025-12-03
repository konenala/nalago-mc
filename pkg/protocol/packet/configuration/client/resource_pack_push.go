package client

import (
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
)

type ConfigResourcePackPush struct {
	client.AddResourcePack
}

func (*ConfigResourcePackPush) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigResourcePackPush
}

func init() {
	registerPacket(packetid.ClientboundConfigResourcePackPush, func() ClientboundPacket {
		return &ConfigResourcePackPush{}
	})
}
