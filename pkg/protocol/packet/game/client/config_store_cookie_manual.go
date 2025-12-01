package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// ConfigStoreCookieManual 全量吞掉 store_cookie 封包，避免 EOF 噪音；如需解析可改為對應欄位。
type ConfigStoreCookieManual struct {
	Raw pk.PluginMessageData
}

func (*ConfigStoreCookieManual) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundConfigStoreCookie
}

func (c *ConfigStoreCookieManual) ReadFrom(r io.Reader) (n int64, err error) {
	return c.Raw.ReadFrom(r)
}

func (ConfigStoreCookieManual) WriteTo(w io.Writer) (n int64, err error) {
	return 0, io.ErrUnexpectedEOF
}

func init() {
	registerPacket(func() ClientboundPacket { return &ConfigStoreCookieManual{} })
}
