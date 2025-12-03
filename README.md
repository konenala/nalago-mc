# nalago-mc

**Minecraft 1.19+ Go 客戶端協議實現 (L2 層)**

nalago-mc 是一個 Minecraft 協議層實現，提供完整的封包處理、連線管理、聊天簽名等功能。設計為高階 bot 框架的基礎層。

## 特性

- ✅ **完整協議支持**: 支援 Minecraft 1.21.0-1.21.10 (協議 767-774)
- ✅ **聊天簽名**: 完整實現 Minecraft 1.19+ 的 RSA 聊天簽名系統
- ✅ **聊天確認**: 自動處理 lastSeen 追蹤和 acknowledgement
- ✅ **世界管理**: 區塊加載、方塊更新、實體追蹤
- ✅ **背包系統**: 完整的物品欄操作支援
- ✅ **事件系統**: 基於泛型的事件發射器
- ✅ **Microsoft 驗證**: 支援 Microsoft 帳號登入

## 安裝

```bash
go get git.konjactw.dev/patyhank/minego
```

## 快速開始

### 基本連線

```go
package main

import (
    "context"
    "fmt"
    "log"

    "git.konjactw.dev/patyhank/minego/pkg/auth"
    "git.konjactw.dev/patyhank/minego/pkg/bot"
    "git.konjactw.dev/patyhank/minego/pkg/client"
)

func main() {
    // 離線模式驗證
    authProvider := &auth.OfflineAuth{
        Username: "TestBot",
    }

    // 創建客戶端
    c := client.NewClient(&bot.ClientOptions{
        AuthProvider: authProvider,
    })

    // 連接到伺服器
    ctx := context.Background()
    if err := c.Connect(ctx, "localhost:25565", nil); err != nil {
        log.Fatal(err)
    }

    // 處理遊戲封包
    if err := c.HandleGame(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### 聊天與指令

```go
// 訂閱聊天事件
bot.SubscribeEvent(c, func(e player.MessageEvent) error {
    fmt.Printf("收到消息: %s\n", e.Message.String())
    return nil
})

// 發送聊天訊息
c.Player().Chat("Hello, world!")

// 發送指令
c.Player().Command("gamemode creative")
```

### 玩家移動

```go
// 飛行到指定位置
pos := mgl64.Vec3{100, 64, 200}
c.Player().FlyTo(pos)

// 看向指定位置
target := mgl64.Vec3{150, 70, 250}
c.Player().LookAt(target)
```

### 方塊操作

```go
// 挖掘方塊
blockPos := protocol.Position{100, 64, 200}
c.Player().BreakBlock(blockPos)

// 放置方塊
c.Player().PlaceBlock(blockPos)
```

### 背包操作

```go
// 打開容器
container, err := c.Player().OpenContainer(protocol.Position{100, 64, 200})
if err != nil {
    log.Fatal(err)
}

// 點擊物品槽
inv := c.Inventory().(*inventory.Manager)
inv.Click(container.ID(), 0, inventory.LeftClick, 0)
```

## 架構

nalago-mc 作為 L2 層，位於 5 層架構中：

```
L5: 全能bot (應用層)
    ↓
L4: mineflayer-go (高階 API)
    ↓
L2: nalago-mc (協議層) ← 本庫
    ↓
L1: go-mc-core (底層協議)

L3: prismarine-go (數據模型，獨立)
```

### 核心組件

- **pkg/client**: 客戶端連線管理
- **pkg/auth**: 驗證系統 (Microsoft/Offline)
- **pkg/game/player**: 玩家控制與聊天
- **pkg/game/world**: 世界狀態管理
- **pkg/game/inventory**: 背包系統
- **pkg/crypto**: 聊天簽名與加密
- **pkg/protocol**: 封包定義

## 聊天系統

nalago-mc 實現了完整的 Minecraft 1.19+ 聊天簽名系統：

### 聊天簽名

```go
// 自動從 Microsoft 帳號取得簽名金鑰
// SetChatProfileKeys 會在登入時自動設置
client.SetChatProfileKeys(keys)

// 發送已簽名的聊天訊息
player.Chat("Hello!") // 自動簽名並發送
```

### 聊天確認機制

系統自動處理：
- **lastSeen 追蹤**: 追蹤最近 20 條已簽名訊息
- **Acknowledgement Bitset**: 20-bit 確認位元組
- **Checksum**: Java Arrays.hashCode 算法
- **Auto-ACK**: 當 pending > 64 自動發送確認封包

詳見 [docs/CHAT_SYSTEM.md](docs/CHAT_SYSTEM.md)

## Microsoft 驗證

```go
import "git.konjactw.dev/patyhank/minego/pkg/auth"

// 使用 Microsoft 帳號
authProvider := &auth.OnlineAuth{
    AccessToken: "eyJ0eXAiOiJKV1QiLCJhbG...",
    Profile: auth.Profile{
        Name: "PlayerName",
        UUID: uuid.MustParse("..."),
    },
}

client := client.NewClient(&bot.ClientOptions{
    AuthProvider: authProvider,
})
```

## 配置選項

```go
type ClientOptions struct {
    AuthProvider auth.Provider  // 驗證提供者
}

type ConnectOptions struct {
    FakeHost string  // 虛擬主機（用於代理）
}
```

## 進階用法

### 自定義封包處理

```go
// 註冊泛型封包處理器
c.PacketHandler().AddGenericPacketHandler(func(ctx context.Context, pk client.ClientboundPacket) {
    fmt.Printf("收到封包: %T\n", pk)
})

// 處理特定封包類型
bot.AddHandler(c, func(ctx context.Context, p *client.PlayerPosition) {
    fmt.Printf("位置更新: X=%.2f, Y=%.2f, Z=%.2f\n", p.X, p.Y, p.Z)
})
```

### 事件系統

```go
// 訂閱事件
bot.SubscribeEvent(c, func(e player.MessageEvent) error {
    // 處理聊天訊息
    return nil
})

bot.SubscribeEvent(c, func(e inventory.ContainerOpenEvent) error {
    // 處理容器開啟
    return nil
})

// 發佈自定義事件
bot.PublishEvent(c, MyCustomEvent{Data: "example"})
```

## 文檔

- [快速開始](docs/GETTING_STARTED.md)
- [API 參考](docs/API.md)
- [聊天系統詳解](docs/CHAT_SYSTEM.md)
- [示例代碼](examples/)

## 版本

- **當前版本**: v0.1.0
- **支援 Minecraft**: 1.21.0 - 1.21.10
- **支援協議**: 767 - 774

## 相關項目

- [prismarine-go](https://github.com/user/prismarine-go) - Minecraft 數據模型 (L3)
- [go-mc](https://github.com/Tnze/go-mc) - 底層協議庫 (L1)
- [mineflayer](https://github.com/PrismarineJS/mineflayer) - Node.js 參考實現

## 授權

MIT License

## 貢獻

歡迎提交 Issue 和 Pull Request！

### 開發指南

```bash
# 編譯
go build ./...

# 測試
go test ./...

# 在工作區模式下開發
cd E:\bot編寫\go-mc
go work sync
```

## 致謝

- 基於 [Tnze/go-mc](https://github.com/Tnze/go-mc) 底層協議
- 參考 [PrismarineJS/mineflayer](https://github.com/PrismarineJS/mineflayer) 聊天系統實現
- 感謝 Minecraft 協議文檔 [wiki.vg](https://wiki.vg)
