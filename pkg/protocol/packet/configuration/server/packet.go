package server

import (
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
)

type ServerboundPacket interface {
	pk.Field
	PacketID() packetid.ServerboundPacketID
}

type serverPacketCreator func() ServerboundPacket

var packetRegistry = make(map[packetid.ServerboundPacketID]serverPacketCreator)

func registerPacket(id packetid.ServerboundPacketID, creator serverPacketCreator) {
	packetRegistry[id] = creator
}
