//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"git.konjactw.dev/patyhank/minego/pkg/auth"
	"git.konjactw.dev/patyhank/minego/pkg/bot"
	"git.konjactw.dev/patyhank/minego/pkg/client"
	"git.konjactw.dev/patyhank/minego/pkg/game/player"
)

func main() {
	// 創建聊天機器人
	authProvider := &auth.OfflineAuth{
		Username: "ChatBot",
	}

	c := client.NewClient(&bot.ClientOptions{
		AuthProvider: authProvider,
	})

	// 設置聊天處理器
	setupChatHandlers(c)

	// 連接
	ctx := context.Background()
	if err := c.Connect(ctx, "localhost:25565", nil); err != nil {
		log.Fatal(err)
	}

	log.Println("聊天機器人已啟動")
	if err := c.HandleGame(ctx); err != nil {
		log.Printf("已斷線: %v", err)
	}
}

func setupChatHandlers(c bot.Client) {
	bot.SubscribeEvent(c, func(e player.MessageEvent) error {
		msg := e.Message.String()
		fmt.Printf("[CHAT] %s\n", msg)

		// 簡單的指令處理
		handleCommand(c, msg)

		return nil
	})
}

func handleCommand(c bot.Client, msg string) {
	// 移除顏色碼
	msg = stripColorCodes(msg)

	// 處理指令
	switch {
	case strings.Contains(msg, "!hello"):
		c.Player().Chat("Hello! I'm a bot!")

	case strings.Contains(msg, "!time"):
		c.Player().Chat(fmt.Sprintf("Current time: %d", 0))

	case strings.Contains(msg, "!pos"):
		pos := c.Player().Entity().Position()
		c.Player().Chat(fmt.Sprintf("I'm at %.1f, %.1f, %.1f",
			pos.X(), pos.Y(), pos.Z()))

	case strings.Contains(msg, "!help"):
		c.Player().Chat("Commands: !hello, !time, !pos, !help")
	}
}

func stripColorCodes(s string) string {
	// 簡單移除 § 顏色碼
	result := strings.Builder{}
	skip := false
	for _, r := range s {
		if skip {
			skip = false
			continue
		}
		if r == '§' {
			skip = true
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
