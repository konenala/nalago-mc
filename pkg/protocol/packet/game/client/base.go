// Code generated bootstrap for enhanced-generator; DO NOT EDIT.
package client

import (
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	"io"
)

// ClientboundPacket 定義 game->client 封包介面。
// 目前 ReadFrom/WriteTo 由生成檔填入 stub，避免編譯錯。
type ClientboundPacket interface {
	PacketID() packetid.ClientboundPacketID
	ReadFrom(r io.Reader) (int64, error)
	WriteTo(w io.Writer) (int64, error)
}

var registry = make(map[packetid.ClientboundPacketID]func() ClientboundPacket)

// ClientboundPackets 對外暴露的封包註冊表，用於客戶端解碼路由。
var ClientboundPackets = registry

// registerPacket 由生成的檔案在 init() 呼叫，註冊封包構造函式。
func registerPacket(id packetid.ClientboundPacketID, ctor func() ClientboundPacket) {
	registry[id] = ctor
}

// New 建立封包實例，若未註冊回傳 nil。
func New(id packetid.ClientboundPacketID) ClientboundPacket {
	if ctor, ok := registry[id]; ok {
		return ctor()
	}
	return nil
}
