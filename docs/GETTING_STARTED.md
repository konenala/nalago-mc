# nalago-mc 快速開始

本教學將引導您使用 nalago-mc 創建第一個 Minecraft bot。

## 安裝

```bash
go get git.konjactw.dev/patyhank/minego
```

## 第一個 Bot

### 1. 基本連線

創建 `main.go`：

```go
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
        Username: "MyBot",
    }

    // 2. 創建客戶端
    c := client.NewClient(&bot.ClientOptions{
        AuthProvider: authProvider,
    })

    // 3. 訂閱事件
    setupEventHandlers(c)

    // 4. 連接伺服器
    ctx := context.Background()
    if err := c.Connect(ctx, "localhost:25565", nil); err != nil {
        log.Fatal(err)
    }

    // 5. 處理遊戲
    if err := c.HandleGame(ctx); err != nil {
        log.Fatal(err)
    }
}

func setupEventHandlers(c bot.Client) {
    // 聊天訊息
    bot.SubscribeEvent(c, func(e player.MessageEvent) error {
        fmt.Printf("[CHAT] %s\n", e.Message.String())
        return nil
    })
}
```

### 2. 運行

```bash
go run main.go
```

## 常見任務

### 發送聊天訊息

```go
func setupEventHandlers(c bot.Client) {
    bot.SubscribeEvent(c, func(e player.MessageEvent) error {
        msg := e.Message.String()
        fmt.Printf("[CHAT] %s\n", msg)

        // 回應特定訊息
        if msg == "hi bot" {
            c.Player().Chat("Hello!")
        }

        return nil
    })
}
```

### 自動回應指令

```go
func setupEventHandlers(c bot.Client) {
    bot.SubscribeEvent(c, func(e player.MessageEvent) error {
        msg := e.Message.String()

        // 解析玩家說話 "[玩家名] hi"
        if strings.Contains(msg, "!pos") {
            // 獲取並回報位置
            pos := c.Player().Entity().Position()
            c.Player().Chat(fmt.Sprintf("我在 %.1f, %.1f, %.1f",
                pos.X(), pos.Y(), pos.Z()))
        }

        return nil
    })
}
```

### 移動到指定位置

```go
import "github.com/go-gl/mathgl/mgl64"

func main() {
    // ... 連線代碼 ...

    // 等待生成後移動
    time.Sleep(2 * time.Second)

    // 飛行到 (100, 70, 200)
    targetPos := mgl64.Vec3{100, 70, 200}
    if err := c.Player().FlyTo(targetPos); err != nil {
        log.Printf("移動失敗: %v", err)
    }
}
```

### 挖掘方塊

```go
import "git.konjactw.dev/patyhank/minego/pkg/protocol"

func breakBlockAt(c bot.Client, x, y, z int) error {
    pos := protocol.Position{x, y, z}
    return c.Player().BreakBlock(pos)
}
```

### 操作容器

```go
func openChest(c bot.Client) error {
    // 打開箱子
    chestPos := protocol.Position{100, 64, 200}
    container, err := c.Player().OpenContainer(chestPos)
    if err != nil {
        return err
    }

    // 取得物品欄管理器
    inv := c.Inventory().(*inventory.Manager)

    // 左鍵點擊第一個格子
    inv.Click(container.ID(), 0, inventory.LeftClick, 0)

    // 等待動作完成
    time.Sleep(100 * time.Millisecond)

    return nil
}
```

## Microsoft 驗證

### 1. 取得 Access Token

使用 Microsoft 裝置碼登入流程（需要另外實現或使用現有工具）。

### 2. 使用 Access Token

```go
import "github.com/google/uuid"

func main() {
    // Microsoft 線上驗證
    authProvider := &auth.OnlineAuth{
        AccessToken: "eyJ0eXAiOiJKV1Qi...", // 你的 token
        Profile: auth.Profile{
            Name: "PlayerName",
            UUID: uuid.MustParse("12345678-1234-1234-1234-123456789abc"),
        },
    }

    c := client.NewClient(&bot.ClientOptions{
        AuthProvider: authProvider,
    })

    // ... 其餘連線代碼 ...
}
```

### 3. 聊天簽名金鑰

```go
import "git.konjactw.dev/patyhank/minego/pkg/client"

// 設置 Profile Keys（從 Microsoft API 取得）
client.SetChatProfileKeys(&client.ChatKeys{
    PrivateDER: privateKeyDER,  // PKCS#8 DER 格式
    PublicDER:  publicKeyDER,   // X.509 DER 格式
    SessionID:  sessionUUID,
    ExpiresAt:  expiryTime,
})
```

## 進階配置

### 虛擬主機（代理伺服器）

```go
// 連接到 BungeeCord/Velocity 代理
err := c.Connect(ctx, "proxy.example.com:25565", &bot.ConnectOptions{
    FakeHost: "lobby.example.com",
})
```

### 協議版本

```go
import "git.konjactw.dev/patyhank/minego/pkg/client"

// 設置協議版本（1.21.0 = 767）
client.SetProtocolVersion(767)
```

### 自定義封包處理

```go
import "git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"

// 處理所有封包
c.PacketHandler().AddGenericPacketHandler(func(ctx context.Context, pk client.ClientboundPacket) {
    switch p := pk.(type) {
    case *client.PlayerPosition:
        fmt.Printf("位置更新: %.2f, %.2f, %.2f\n", p.X, p.Y, p.Z)
    case *client.PlayerChat:
        fmt.Printf("玩家聊天: %s\n", p.PlainMessage)
    }
})

// 或使用類型安全的處理器
bot.AddHandler(c, func(ctx context.Context, p *client.PlayerPosition) {
    fmt.Printf("Y 座標: %.2f\n", p.Y)
})
```

## 錯誤處理

### 連線錯誤

```go
if err := c.Connect(ctx, "localhost:25565", nil); err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        log.Fatal("伺服器未運行")
    } else if strings.Contains(err.Error(), "timeout") {
        log.Fatal("連線超時")
    } else {
        log.Fatalf("連線失敗: %v", err)
    }
}
```

### 遊戲中斷線

```go
bot.AddHandler(c, func(ctx context.Context, p *client.Disconnect) {
    reason := p.Reason.String()
    log.Printf("被踢出: %s", reason)

    // 可以在這裡實現重連邏輯
})
```

## 完整範例

查看 [examples/](../examples/) 目錄：

- `simple/` - 基本連線範例
- `chat_bot/` - 聊天機器人
- `builder/` - 自動建造
- `miner/` - 自動挖礦

## 下一步

- 閱讀 [API 參考](API.md) 了解所有可用方法
- 查看 [聊天系統詳解](CHAT_SYSTEM.md) 深入了解聊天簽名
- 探索 [examples/](../examples/) 中的完整範例

## 疑難排解

### "An internal error occurred in your connection"

這通常是聊天簽名問題。確認：
1. 使用正確的 Microsoft Access Token
2. Profile Keys 正確設置
3. 伺服器支援簽名聊天（`enforce-secure-profile=true`）

### "Invalid session"

Microsoft Token 已過期，需要重新驗證。

### EOF 錯誤

檢查：
1. 協議版本是否匹配伺服器版本
2. 網路連線是否穩定
3. 伺服器是否允許離線模式（如果使用 OfflineAuth）

## 需要幫助？

- 查看 [API 文檔](API.md)
- 提交 [GitHub Issue](https://github.com/user/nalago-mc/issues)
