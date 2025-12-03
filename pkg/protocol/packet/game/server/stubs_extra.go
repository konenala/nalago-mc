package server

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/slot"
)

// 手動補齊目前程式碼會用到但生成器未輸出的封包。

type ContainerClick struct {
	WindowID    int32
	StateID     int32
	Slot        int16
	Button      int8
	Mode        int32
	CarriedItem slot.Slot
}

func (*ContainerClick) PacketID() packetid.ServerboundPacketID { return 0 }
func (*ContainerClick) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (ContainerClick) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type ContainerClose struct{ WindowID int32 }

func (*ContainerClose) PacketID() packetid.ServerboundPacketID { return 0 }
func (*ContainerClose) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (ContainerClose) WriteTo(io.Writer) (int64, error)        { return 0, nil }

// Configuration phase aliases
type ClientInformation struct{}
type CookieResponse struct{}
type ResourcePack struct{}

func (*ClientInformation) PacketID() packetid.ServerboundPacketID { return 0 }
func (*ClientInformation) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (ClientInformation) WriteTo(io.Writer) (int64, error)        { return 0, nil }
func (*CookieResponse) PacketID() packetid.ServerboundPacketID    { return 0 }
func (*CookieResponse) ReadFrom(io.Reader) (int64, error)         { return 0, nil }
func (CookieResponse) WriteTo(io.Writer) (int64, error)           { return 0, nil }
func (*ResourcePack) PacketID() packetid.ServerboundPacketID      { return 0 }
func (*ResourcePack) ReadFrom(io.Reader) (int64, error)           { return 0, nil }
func (ResourcePack) WriteTo(io.Writer) (int64, error)             { return 0, nil }
