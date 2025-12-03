package server

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type ConfigKeepAlive struct {
	ID int64
}

func (*ConfigKeepAlive) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigKeepAlive
}

func init() {
	registerPacket(packetid.ServerboundConfigKeepAlive, func() ServerboundPacket {
		return &ConfigKeepAlive{}
	})
}
