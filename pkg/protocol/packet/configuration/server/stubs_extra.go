package server

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

type ClientInformation struct{}

type CookieResponse struct{}

type ResourcePack struct{}

func (*ClientInformation) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigClientInformation
}
func (p *ClientInformation) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p ClientInformation) WriteTo(w io.Writer) (n int64, err error) {
	return 0, nil
}

func (*CookieResponse) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigCookieResponse
}
func (p *CookieResponse) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p CookieResponse) WriteTo(w io.Writer) (n int64, err error) {
	return 0, nil
}

func (*ResourcePack) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigResourcePack
}
func (p *ResourcePack) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
func (p ResourcePack) WriteTo(w io.Writer) (n int64, err error) {
	return 0, nil
}

func init() {
	registerPacket(packetid.ServerboundConfigClientInformation, func() ServerboundPacket { return &ClientInformation{} })
	registerPacket(packetid.ServerboundConfigCookieResponse, func() ServerboundPacket { return &CookieResponse{} })
	registerPacket(packetid.ServerboundConfigResourcePack, func() ServerboundPacket { return &ResourcePack{} })
}
