package player

import (
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

const chatStateCapacity = 20

type seenMsg struct {
	signature []byte
	pending   bool
}

type chatState struct {
	pending  int32
	offset   int
	lastSeen [chatStateCapacity]*seenMsg
}

// NextOffset 取得目前待回報的訊息數，對應封包 offset/messageCount。
func (s *chatState) NextOffset() int32 {
	return s.pending
}

// ResetPending 將 pending 計數清零（送出 chat/ack 後呼叫）。
func (s *chatState) ResetPending() {
	s.pending = 0
}

// Pending 回傳目前尚未回報的訊息數。
func (s *chatState) Pending() int32 {
	return s.pending
}

// IncSeen 新增一筆已接收的簽名訊息到 lastSeen 視窗。
func (s *chatState) IncSeen(signature []byte) {
	if len(signature) == 0 {
		return
	}
	// 環形緩衝區寫入最新訊息簽章
	copied := make([]byte, len(signature))
	copy(copied, signature)
	s.lastSeen[s.offset] = &seenMsg{signature: copied, pending: true}
	s.offset = (s.offset + 1) % chatStateCapacity
	s.pending++
}

func (s *chatState) GetAcknowledgements() (acknowledgements [][]byte, bitset pk.FixedBitSet) {
	bitset = pk.NewFixedBitSet(chatStateCapacity)
	acknowledgements = make([][]byte, 0, chatStateCapacity)

	for i := 0; i < chatStateCapacity; i++ {
		idx := (s.offset + i) % chatStateCapacity
		msg := s.lastSeen[idx]
		if msg == nil {
			continue
		}
		bitset.Set(i, true)
		if len(msg.signature) > 0 {
			ackSig := make([]byte, len(msg.signature))
			copy(ackSig, msg.signature)
			acknowledgements = append(acknowledgements, ackSig)
		}
		// 標記已處理，避免重複 pending 統計
		msg.pending = false
	}
	return acknowledgements, bitset
}

// Checksum 計算 lastSeen 的校驗值，對應 Java Arrays.hashCode。
func (s *chatState) Checksum() int8 {
	// 對應 minecraft-protocol computeChatChecksum (Java Arrays.hashCode)
	var checksum int32 = 1
	for i := 0; i < chatStateCapacity; i++ {
		idx := (s.offset + i) % chatStateCapacity
		msg := s.lastSeen[idx]
		if msg == nil || len(msg.signature) == 0 {
			continue
		}
		var sigHash int32 = 1
		for _, b := range msg.signature {
			sigHash = 31*sigHash + int32(b)
		}
		checksum = 31*checksum + sigHash
	}
	result := byte(checksum & 0xFF)
	if result == 0 {
		result = 1
	}
	return int8(result)
}
