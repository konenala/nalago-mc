package client

import (
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
)

type ConfigUpdateTags struct {
	client.UpdateTags
}

func (*ConfigUpdateTags) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigUpdateTags
}

func init() {
	registerPacket(packetid.ClientboundConfigUpdateTags, func() ClientboundPacket {
		return &ConfigUpdateTags{}
	})
}
