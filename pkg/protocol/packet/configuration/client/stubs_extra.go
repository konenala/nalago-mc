package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

// 補齊 configuration 階段缺少的封包結構，僅供編譯使用。

type CustomReportDetails struct{}

func (*CustomReportDetails) PacketID() packetid.ClientboundPacketID { return 0 }
func (*CustomReportDetails) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (CustomReportDetails) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type RemoveResourcePack struct{}

func (*RemoveResourcePack) PacketID() packetid.ClientboundPacketID { return 0 }
func (*RemoveResourcePack) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (RemoveResourcePack) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type AddResourcePack struct{}

func (*AddResourcePack) PacketID() packetid.ClientboundPacketID { return 0 }
func (*AddResourcePack) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (AddResourcePack) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type ServerLinks struct{}

func (*ServerLinks) PacketID() packetid.ClientboundPacketID { return 0 }
func (*ServerLinks) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (ServerLinks) WriteTo(io.Writer) (int64, error)        { return 0, nil }

type UpdateTags struct{}

func (*UpdateTags) PacketID() packetid.ClientboundPacketID { return 0 }
func (*UpdateTags) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (UpdateTags) WriteTo(io.Writer) (int64, error)        { return 0, nil }
