package crypto

import "os"

var chatSignDebug bool

func init() {
	chatSignDebug = os.Getenv("CHAT_DEBUG") == "1" || os.Getenv("CHAT_DEBUG") == "true"
}

// SetChatSignDebug 由外部設定簽名 debug 開關。
func SetChatSignDebug(v bool) { chatSignDebug = v }
