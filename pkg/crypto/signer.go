package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// chatSignDebug 定義於 debug.go，可由 SetChatSignDebug 設定

// ChatSigner handles signing chat messages for Minecraft 1.19+
type ChatSigner struct {
	privateKey   *rsa.PrivateKey
	publicKey    *rsa.PublicKey
	playerUUID   uuid.UUID
	sessionID    uuid.UUID
	sessionIndex int32
	expiresAt    time.Time
}

// NewChatSigner creates a new chat signer with the provided private key
func NewChatSigner(privateDER []byte, playerUUID uuid.UUID, sessionID uuid.UUID) (*ChatSigner, error) {
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
		privateKey:   rsaKey,
		publicKey:    &rsaKey.PublicKey,
		playerUUID:   playerUUID,
		sessionID:    sessionID,
		sessionIndex: 0,                              // Start at 0, increments with each message
		expiresAt:    time.Now().Add(24 * time.Hour), // Keys typically valid for 24h
	}, nil
}

// SignChatMessage signs a chat message following Minecraft's protocol
// acknowledgements: array of previous message signatures (lastSeen)
// Returns the signature bytes or an error
func (s *ChatSigner) SignChatMessage(message string, timestamp int64, salt int64, acknowledgements [][]byte) ([]byte, error) {
	// Encode the message for signing following Minecraft's format (mineflayer compatible)
	encoded := s.encodeMessageForSigning(message, timestamp, salt, acknowledgements)

	// Sign with SHA256 and RSA
	hashed := sha256.Sum256(encoded)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	// Increment session index after each message (mineflayer chat.js:462)
	s.sessionIndex++

	if chatSignDebug {
		fmt.Printf("[CHAT-SIGN] msg='%s' ts=%d salt=%d idx=%d acks=%d payload=%s siglen=%d\n",
			message, timestamp, salt, s.sessionIndex-1, len(acknowledgements), hex.EncodeToString(encoded), len(signature))
	}

	return signature, nil
}

// encodeMessageForSigning encodes a message in the exact format Minecraft expects
// 參照 mineflayer/lib/plugins/chat.js client.signMessage (useChatSessions 分支)：
// concat('i32',1,'UUID',player,'UUID',session,'i32',index,'i64',salt,'i64',timestamp/1000,
//
//	'i32',length,'pstring',message,'i32',ackCount,'buffer',concat(acks))
//
// 注意：`pstring` 會再寫入 VarInt 長度，因此必須同時包含 i32 length 與 VarInt 長度，兩者缺一會導致簽章驗證失敗。
func (s *ChatSigner) encodeMessageForSigning(message string, timestamp int64, salt int64, acknowledgements [][]byte) []byte {
	messageBytes := []byte(message)
	messageLen := len(messageBytes)

	// Calculate total size
	// Version(4) + PlayerUUID(16) + SessionUUID(16) + Index(4) + Salt(8) + Timestamp(8) +
	// MessageLength(4) + VarIntLen(messageLen) + Message(len) +
	// AckCount(4) + VarIntLen(ackBytes) + AckBytes
	ackDataSize := 0
	for _, ack := range acknowledgements {
		ackDataSize += len(ack)
	}

	ackVarLen := varIntSize(ackDataSize) // even when 0, VarInt(0) is 1 byte
	varIntLen := varIntSize(messageLen)
	size := 4 + 16 + 16 + 4 + 8 + 8 + 4 + varIntLen + messageLen + 4 + ackVarLen + ackDataSize
	buf := make([]byte, size)
	offset := 0

	// Write version = 1 (int32, big endian)
	binary.BigEndian.PutUint32(buf[offset:], 1)
	offset += 4

	// Write player UUID (16 bytes)
	copy(buf[offset:], s.playerUUID[:])
	offset += 16

	// Write session UUID (16 bytes)
	copy(buf[offset:], s.sessionID[:])
	offset += 16

	// Write session index (int32, big endian) - current value before increment
	binary.BigEndian.PutUint32(buf[offset:], uint32(s.sessionIndex))
	offset += 4

	// Write salt (int64, big endian)
	binary.BigEndian.PutUint64(buf[offset:], uint64(salt))
	offset += 8

	// Write timestamp / 1000 (int64, big endian) - convert milliseconds to seconds
	binary.BigEndian.PutUint64(buf[offset:], uint64(timestamp/1000))
	offset += 8

	// Write message length (int32, big endian)
	binary.BigEndian.PutUint32(buf[offset:], uint32(messageLen))
	offset += 4

	// Write VarInt length (pstring prefix)
	offset += putVarInt(buf[offset:], messageLen)

	// Write message bytes
	copy(buf[offset:], messageBytes)
	offset += messageLen

	// Write acknowledgements count (int32, big endian)
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(acknowledgements)))
	offset += 4

	// Write acknowledgements total length (VarInt) then concatenated signatures
	offset += putVarInt(buf[offset:], ackDataSize)
	for _, ack := range acknowledgements {
		copy(buf[offset:], ack)
		offset += len(ack)
	}

	return buf
}

// varIntSize 回傳寫出該數值所需的位元組數
func varIntSize(value int) int {
	if value == 0 {
		return 1
	}
	size := 0
	for x := value; x != 0; x >>= 7 {
		size++
	}
	return size
}

// putVarInt 以 VarInt 方式寫入 value，返回寫入的位元組數
func putVarInt(dst []byte, value int) int {
	size := 0
	for {
		temp := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			temp |= 0x80
		}
		dst[size] = temp
		size++
		if value == 0 {
			break
		}
	}
	return size
}

// IsExpired checks if the chat keys have expired
func (s *ChatSigner) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

// SessionID returns the session ID
func (s *ChatSigner) SessionID() uuid.UUID {
	return s.sessionID
}
