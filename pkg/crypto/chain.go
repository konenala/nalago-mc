package crypto

import (
	"sync"
)

// MessageChain tracks the chain of signed messages for acknowledgment
// This is required for Minecraft 1.19+ chat system
type MessageChain struct {
	mu              sync.RWMutex
	lastSignature   []byte
	pendingMessages []SignedMessage
	maxChainLength  int
}

// SignedMessage represents a signed chat message in the chain
type SignedMessage struct {
	MessageID int32
	Signature []byte
}

// NewMessageChain creates a new message chain tracker
func NewMessageChain() *MessageChain {
	return &MessageChain{
		pendingMessages: make([]SignedMessage, 0),
		maxChainLength:  20, // Minecraft tracks last 20 messages
	}
}

// AddMessage adds a signed message to the chain
func (c *MessageChain) AddMessage(messageID int32, signature []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update last signature
	if signature != nil && len(signature) > 0 {
		c.lastSignature = make([]byte, len(signature))
		copy(c.lastSignature, signature)
	}

	// Add to pending messages
	c.pendingMessages = append(c.pendingMessages, SignedMessage{
		MessageID: messageID,
		Signature: signature,
	})

	// Keep only last N messages
	if len(c.pendingMessages) > c.maxChainLength {
		c.pendingMessages = c.pendingMessages[len(c.pendingMessages)-c.maxChainLength:]
	}
}

// GetLastSignature returns the signature of the last sent message
func (c *MessageChain) GetLastSignature() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastSignature == nil {
		return nil
	}

	// Return a copy to prevent external modification
	sig := make([]byte, len(c.lastSignature))
	copy(sig, c.lastSignature)
	return sig
}

// GetPendingMessages returns the list of pending messages for acknowledgment
func (c *MessageChain) GetPendingMessages() []SignedMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy
	messages := make([]SignedMessage, len(c.pendingMessages))
	copy(messages, c.pendingMessages)
	return messages
}

// Clear clears the message chain
func (c *MessageChain) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastSignature = nil
	c.pendingMessages = make([]SignedMessage, 0)
}

// Reset resets the chain to empty state
func (c *MessageChain) Reset() {
	c.Clear()
}
