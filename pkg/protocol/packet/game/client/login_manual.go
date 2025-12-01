package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// LoginManual 以「吃掉全部 payload」的方式解碼 1.21.x 的 clientbound login 封包，避免未對齊欄位導致 EOF 噪音。
// 內容目前未使用，後續若需要可改為完整欄位定義。
type LoginManual struct {
	Raw pk.PluginMessageData
}

func (*LoginManual) PacketID() packetid.ClientboundPacketID { return packetid.ClientboundLogin }

func (l *LoginManual) ReadFrom(r io.Reader) (n int64, err error) {
	return l.Raw.ReadFrom(r)
}

func (LoginManual) WriteTo(w io.Writer) (n int64, err error) { return 0, io.ErrUnexpectedEOF }

func init() {
	registerPacket(func() ClientboundPacket { return &LoginManual{} })
}
