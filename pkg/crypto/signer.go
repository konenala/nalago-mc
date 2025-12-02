package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ChatSigner handles signing chat messages for Minecraft 1.19+
type ChatSigner struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	sessionID  uuid.UUID
	expiresAt  time.Time
}

// NewChatSigner creates a new chat signer with the provided private key
func NewChatSigner(privateDER []byte, sessionID uuid.UUID) (*ChatSigner, error) {
	// Parse PKCS#8 DER encoded private key
	privateKey, err := x509.ParsePKCS8PrivateKey(privateDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}

	return &ChatSigner{
		privateKey: rsaKey,
		publicKey:  &rsaKey.PublicKey,
		sessionID:  sessionID,
		expiresAt:  time.Now().Add(24 * time.Hour), // Keys typically valid for 24h
	}, nil
}

// SignChatMessage signs a chat message following Minecraft's protocol
// Returns the signature bytes or an error
func (s *ChatSigner) SignChatMessage(message string, timestamp int64, salt int64) ([]byte, error) {
	// Encode the message for signing following Minecraft's format
	encoded := s.encodeMessageForSigning(message, timestamp, salt)

	// Sign with SHA256 and RSA
	hashed := sha256.Sum256(encoded)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return signature, nil
}

// encodeMessageForSigning encodes a message in the exact format Minecraft expects
// Based on wiki.vg protocol documentation and mineflayer implementation
func (s *ChatSigner) encodeMessageForSigning(message string, timestamp int64, salt int64) []byte {
	// Calculate total size needed
	// Format: [salt(8)] + [timestamp(8)] + [message_length(4)] + [message(variable)]
	messageBytes := []byte(message)
	size := 8 + 8 + 4 + len(messageBytes)

	buf := make([]byte, size)
	offset := 0

	// Write salt (8 bytes, big endian)
	binary.BigEndian.PutUint64(buf[offset:], uint64(salt))
	offset += 8

	// Write timestamp (8 bytes, big endian)
	binary.BigEndian.PutUint64(buf[offset:], uint64(timestamp))
	offset += 8

	// Write message length (4 bytes, big endian)
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(messageBytes)))
	offset += 4

	// Write message bytes
	copy(buf[offset:], messageBytes)

	return buf
}

// IsExpired checks if the chat keys have expired
func (s *ChatSigner) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

// SessionID returns the session ID
func (s *ChatSigner) SessionID() uuid.UUID {
	return s.sessionID
}
