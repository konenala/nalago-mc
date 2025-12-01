package client

import (
	"git.konjactw.dev/falloutBot/go-mc/yggdrasil/user"
	"github.com/google/uuid"
)

// ChatKeys 保存聊天簽名所需的 session 與密鑰資料。
type ChatKeys struct {
	SessionID uuid.UUID
	Public    user.PublicKey
	// 私鑰的 PKCS#8 DER 編碼（後續簽章用）
	PrivateDER []byte
}

var chatKeys *ChatKeys

// SetChatProfileKeys 設定聊天簽名所需的密鑰資料（全域共用）。
func SetChatProfileKeys(k *ChatKeys) {
	chatKeys = k
}
