// Code generated bootstrap for enhanced-generator; DO NOT EDIT.
package server

import (
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
	"io"
)

// ServerboundPacket 定義 client->server 封包介面。
type ServerboundPacket interface {
	PacketID() packetid.ServerboundPacketID
	ReadFrom(r io.Reader) (int64, error)
	WriteTo(w io.Writer) (int64, error)
}

var registry = make(map[packetid.ServerboundPacketID]func() ServerboundPacket)

func registerPacket(id packetid.ServerboundPacketID, ctor func() ServerboundPacket) {
	registry[id] = ctor
}

func New(id packetid.ServerboundPacketID) ServerboundPacket {
	if ctor, ok := registry[id]; ok {
		return ctor()
	}
	return nil
}
