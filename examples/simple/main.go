package main

import (
	"context"
	"fmt"
	"log"

	"git.konjactw.dev/patyhank/minego/pkg/auth"
	"git.konjactw.dev/patyhank/minego/pkg/bot"
	"git.konjactw.dev/patyhank/minego/pkg/client"
	"git.konjactw.dev/patyhank/minego/pkg/game/player"
)

func main() {
	// 1. 創建離線驗證
	authProvider := &auth.OfflineAuth{
		Username: "SimpleBot",
	}

	// 2. 創建客戶端
	c := client.NewClient(&bot.ClientOptions{
		AuthProvider: authProvider,
	})

	// 3. 設置事件處理器
	setupEventHandlers(c)

	// 4. 連接到伺服器
	ctx := context.Background()
	serverAddr := "localhost:25565" // 修改為你的伺服器地址

	log.Printf("正在連接到 %s...", serverAddr)
	if err := c.Connect(ctx, serverAddr, nil); err != nil {
		log.Fatalf("連接失敗: %v", err)
	}

	log.Println("連接成功！正在進入遊戲...")

	// 5. 處理遊戲（阻塞直到斷線）
	if err := c.HandleGame(ctx); err != nil {
		log.Printf("遊戲結束: %v", err)
	}
}

func setupEventHandlers(c bot.Client) {
	// 訂閱聊天訊息事件
	bot.SubscribeEvent(c, func(e player.MessageEvent) error {
		msg := e.Message.String()
		fmt.Printf("[CHAT] %s\n", msg)
		return nil
	})

	log.Println("事件處理器已設置")
}
