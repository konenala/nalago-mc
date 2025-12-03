package server

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
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
