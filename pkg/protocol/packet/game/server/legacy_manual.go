package server

import (
	"fmt"
	"io"

	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"
)

// PlayerAction 為舊名稱封包，手動實作以兼容現有 bot 流程。
type PlayerAction struct {
	Status   int32
	Sequence int32
	Location pk.Position
	Face     int32
}

func (*PlayerAction) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundPlayerAction
}

func (p *PlayerAction) ReadFrom(r io.Reader) (int64, error) {
	return 0, fmt.Errorf("PlayerAction ReadFrom not implemented")
}

func (p PlayerAction) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64

	temp, err = pk.VarInt(p.Status).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.VarInt(p.Sequence).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = p.Location.WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.Byte(p.Face).WriteTo(w)
	n += temp
	return
}

// UseItemOn 封包舊名稱，手動兼容。
type UseItemOn struct {
	Hand        int32
	Location    pk.Position
	Face        int32
	CursorX     float32
	CursorY     float32
	CursorZ     float32
	InsideBlock bool
	Sequence    int32
}

func (*UseItemOn) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundUseItemOn
}

func (p *UseItemOn) ReadFrom(r io.Reader) (int64, error) {
	return 0, fmt.Errorf("UseItemOn ReadFrom not implemented")
}

func (p UseItemOn) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64

	temp, err = pk.VarInt(p.Hand).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = p.Location.WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.VarInt(p.Face).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.Float(p.CursorX).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.Float(p.CursorY).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.Float(p.CursorZ).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.Boolean(p.InsideBlock).WriteTo(w)
	n += temp
	if err != nil {
		return
	}

	temp, err = pk.VarInt(p.Sequence).WriteTo(w)
	n += temp
	return
}

// ---- 關鍵：讓它們接到新的 ServerboundPacket 註冊系統 ----

// 編譯期檢查：確認這兩個 struct 符合 ServerboundPacket 介面
var (
	_ ServerboundPacket = (*PlayerAction)(nil)
	_ ServerboundPacket = (*UseItemOn)(nil)
)

// 在 init() 裡註冊到 ServerboundPackets map
func init() {
	registerPacket(packetid.ServerboundPlayerAction, func() ServerboundPacket {
		return &PlayerAction{}
	})
	registerPacket(packetid.ServerboundUseItemOn, func() ServerboundPacket {
		return &UseItemOn{}
	})
}
