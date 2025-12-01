package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// UpdateAdvancementsManual 目前僅吞掉 payload，避免未對齊欄位產生 EOF 日誌。
// TODO: 若後續需要進度資料，可依 1.21.8 協議補齊欄位。
type UpdateAdvancementsManual struct {
	Raw pk.PluginMessageData
}

func (*UpdateAdvancementsManual) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundUpdateAdvancements
}

func (u *UpdateAdvancementsManual) ReadFrom(r io.Reader) (n int64, err error) {
	return u.Raw.ReadFrom(r)
}

func (UpdateAdvancementsManual) WriteTo(w io.Writer) (n int64, err error) {
	return 0, io.ErrUnexpectedEOF
}

func init() {
	registerPacket(func() ClientboundPacket { return &UpdateAdvancementsManual{} })
}
