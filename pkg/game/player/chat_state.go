package player

import (
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// chatState 模擬 vanilla/mineflayer 的 lastSeen 結構，維持 pending 計數與 checksum。
// 目前僅支援無簽名聊天，checksum 與 ack 都維持最小合法值。
type chatState struct {
	pending int32
	seen    int32
}

// NextOffset 回傳目前 pending 做為 offset。
func (s *chatState) NextOffset() int32 {
	return s.pending
}

// ResetPending 在成功送出聊天後歸零。
func (s *chatState) ResetPending() {
	s.pending = 0
}

// AckBitset 產生 20bit 固定長度的 ack bitset（全 0）。
func (s *chatState) AckBitset() pk.FixedBitSet {
	return pk.NewFixedBitSet(20)
}

// Checksum 依照 mineflayer 的 computeChatChecksum 規則：
// - 沒有 lastSeen 時回傳 1（vanilla 會避免 0）
// - 我們目前未追蹤 lastSeen，因此固定回 1。
func (s *chatState) Checksum() int8 {
	return 1
}

// IncSeen 供客戶端收到聊天時增加計數並回傳目前累計。
func (s *chatState) IncSeen() int32 {
	s.seen++
	return s.seen
}
