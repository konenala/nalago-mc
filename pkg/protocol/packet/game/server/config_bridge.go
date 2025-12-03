package server

import (
	"io"

	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
)

type ClientInformation struct{}
type CookieResponse struct{}
type ResourcePack struct{}

func (*ClientInformation) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigClientInformation
}
func (*ClientInformation) ReadFrom(io.Reader) (int64, error) { return 0, nil }
func (ClientInformation) WriteTo(io.Writer) (int64, error)   { return 0, nil }

func (*CookieResponse) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigCookieResponse
}
func (*CookieResponse) ReadFrom(io.Reader) (int64, error) { return 0, nil }
func (CookieResponse) WriteTo(io.Writer) (int64, error)   { return 0, nil }

func (*ResourcePack) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundConfigResourcePack
}
func (*ResourcePack) ReadFrom(io.Reader) (int64, error) { return 0, nil }
func (ResourcePack) WriteTo(io.Writer) (int64, error)   { return 0, nil }

func init() {
	registerPacket(packetid.ServerboundConfigClientInformation, func() ServerboundPacket { return &ClientInformation{} })
	registerPacket(packetid.ServerboundConfigCookieResponse, func() ServerboundPacket { return &CookieResponse{} })
	registerPacket(packetid.ServerboundConfigResourcePack, func() ServerboundPacket { return &ResourcePack{} })
}
