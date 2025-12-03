package player

import "os"

var chatDebug bool

func init() {
	chatDebug = os.Getenv("CHAT_DEBUG") == "1" || os.Getenv("CHAT_DEBUG") == "true"
}

// SetChatDebug 由外部設定聊天簽名相關的 debug 開關。
func SetChatDebug(v bool) { chatDebug = v }
