package client

import (
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
)

type ConfigResourcePackPop struct {
	client.RemoveResourcePack
}

func (*ConfigResourcePackPop) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigResourcePackPop
}

func init() {
	registerPacket(packetid.ClientboundConfigResourcePackPop, func() ClientboundPacket {
		return &ConfigResourcePackPop{}
	})
}
