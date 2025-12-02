package player

import (
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// chatState 模擬 vanilla/mineflayer 的 lastSeen 結構，維持 pending 計數與 checksum。
// 支援簽名聊天，追蹤最近 20 條消息的完整簽名用於 acknowledgment。
// 完全符合 Minecraft 協議和 mineflayer 實現 (chat.js:556-569)
type chatState struct {
	pending  int32
	seen     int32
	lastSeen [20][]byte // Track last 20 seen message signatures for acknowledgment
}

// NextOffset 回傳目前 pending 做為 offset。
func (s *chatState) NextOffset() int32 {
	return s.pending
}

// ResetPending 在成功送出聊天後歸零。
func (s *chatState) ResetPending() {
	s.pending = 0
}

// AckBitset 產生 20bit 固定長度的 ack bitset。
// 每個 bit 代表對應 lastSeen 消息是否被確認。
// 參考 mineflayer: chat.js:340-362 (getAcknowledgements)
func (s *chatState) AckBitset() pk.FixedBitSet {
	bitset := pk.NewFixedBitSet(20)
	// Mark all seen messages as acknowledged
	for i := 0; i < 20; i++ {
		if len(s.lastSeen[i]) > 0 { // Check if signature exists
			bitset.Set(i, true)
		}
	}
	return bitset
}

// Checksum 計算 lastSeen 的 checksum。
// 使用 Java Arrays.hashCode 算法，與 vanilla 客戶端一致。
// 參考 mineflayer: minecraft-protocol/src/datatypes/checksums.js
func (s *chatState) Checksum() int8 {
	// If no messages seen, return 1 (avoid 0)
	if s.seen == 0 {
		return 1
	}

	// Java Arrays.hashCode algorithm
	// checksum = 31 * checksum + sigHash for each signature
	// sigHash = 31 * sigHash + byte for each byte in signature
	checksum := uint32(1)
	for _, sig := range s.lastSeen {
		if len(sig) > 0 {
			// Calculate hash of this signature
			sigHash := uint32(1)
			for _, b := range sig {
				sigHash = (31*sigHash + uint32(b)) & 0xffffffff
			}
			checksum = (31*checksum + sigHash) & 0xffffffff
		}
	}

	// Return lower 8 bits, avoid 0
	result := int8(checksum & 0xff)
	if result == 0 {
		return 1
	}
	return result
}

// IncSeen 供客戶端收到聊天時增加計數並更新 lastSeen 列表。
// 類似 mineflayer LastSeenMessages.push() (chat.js:575)
// signature: 消息簽名，如果沒有簽名則傳入 nil
func (s *chatState) IncSeen(signature []byte) int32 {
	s.seen++
	s.pending++ // CRITICAL: increment pending count (mineflayer chat.js:566)

	// Shift lastSeen array and add new message at front
	for i := 19; i > 0; i-- {
		s.lastSeen[i] = s.lastSeen[i-1]
	}

	// Store the signature (nil if unsigned)
	s.lastSeen[0] = signature
	return s.seen
}
