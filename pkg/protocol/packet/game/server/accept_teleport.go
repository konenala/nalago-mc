package server

import (
	"io"

	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
)

type AcceptTeleportation struct {
	TeleportID int32
}

func (*AcceptTeleportation) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundAcceptTeleportation
}
func (p *AcceptTeleportation) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p AcceptTeleportation) WriteTo(w io.Writer) (n int64, err error) {
	return pk.VarInt(p.TeleportID).WriteTo(w)
}

func init() {
	registerPacket(packetid.ServerboundAcceptTeleportation, func() ServerboundPacket { return &AcceptTeleportation{} })
}
