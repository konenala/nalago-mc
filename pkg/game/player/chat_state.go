package player

// This file is a dependency for other packages but its chat logic is superseded by
// the implementation in all-in-one-bot/internal/mcclient/lastseen.go.
// This minimal implementation is to ensure the project compiles.

import (
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

const chatStateCapacity = 20

type chatState struct {
	pending  int32
	seen     int32
	lastSeen [chatStateCapacity][]byte
}

func (s *chatState) NextOffset() int32 {
	return s.pending
}

func (s *chatState) ResetPending() {
	s.pending = 0
}

func (s *chatState) Pending() int32 {
	return s.pending
}

func (s *chatState) IncSeen(signature []byte) {
	s.seen++
	s.pending++
	for i := chatStateCapacity - 1; i > 0; i-- {
		s.lastSeen[i] = s.lastSeen[i-1]
	}
	s.lastSeen[0] = signature
}

func (s *chatState) GetAcknowledgements() (acknowledgements [][]byte, bitset pk.FixedBitSet) {
	return nil, pk.NewFixedBitSet(chatStateCapacity)
}

func (s *chatState) Checksum() int8 {
	return 1
}
