# 聊天系統詳解

nalago-mc 實現了完整的 Minecraft 1.19+ 聊天簽名與確認系統，100% 兼容 mineflayer。

## 概述

Minecraft 1.19 引入了聊天簽名機制，要求所有聊天訊息使用 RSA 簽名。這個系統包含三個主要部分：

1. **訊息簽名** - 使用 RSA-SHA256 簽名每條訊息
2. **lastSeen 追蹤** - 追蹤最近 20 條已簽名訊息
3. **Acknowledgement** - 確認已接收的訊息

## 架構

```
┌──────────────────────────────────────────────────────────┐
│  Client (nalago-mc)                                      │
│                                                          │
│  ┌────────────┐   ┌──────────────┐   ┌──────────────┐ │
│  │ ChatSigner │──→│  chatState   │──→│ Chat Packet  │ │
│  │  (crypto)  │   │ (lastSeen)   │   │ (protocol)   │ │
│  └────────────┘   └──────────────┘   └──────────────┘ │
│       ↓                  ↓                    ↓         │
│  Private Key      20 Signatures         Signature      │
│  Session UUID     Offset/Checksum       Bitset         │
└──────────────────────────────────────────────────────────┘
                             ↓
                      [ Network ]
                             ↓
┌──────────────────────────────────────────────────────────┐
│  Minecraft Server                                        │
│  - 驗證簽名                                               │
│  - 檢查 lastSeen                                         │
│  - 驗證 acknowledgement                                  │
└──────────────────────────────────────────────────────────┘
```

## 1. 訊息簽名 (ChatSigner)

### 簽名格式

根據 mineflayer 實現 (chat.js:456-463)，簽名數據格式：

```
┌─────────────────────────────────────────────────────────┐
│ Version (int32, 4 bytes)          = 1                   │
├─────────────────────────────────────────────────────────┤
│ Player UUID (16 bytes)            = 玩家 UUID           │
├─────────────────────────────────────────────────────────┤
│ Session UUID (16 bytes)           = 會話 UUID           │
├─────────────────────────────────────────────────────────┤
│ Session Index (int32, 4 bytes)    = 訊息編號（遞增）     │
├─────────────────────────────────────────────────────────┤
│ Salt (int64, 8 bytes)             = 隨機鹽值             │
├─────────────────────────────────────────────────────────┤
│ Timestamp (int64, 8 bytes)        = 時間戳/1000（秒）    │
├─────────────────────────────────────────────────────────┤
│ Message Length (int32, 4 bytes)   = 訊息長度             │
├─────────────────────────────────────────────────────────┤
│ Message VarInt Length             = VarInt(訊息長度)     │
├─────────────────────────────────────────────────────────┤
│ Message (variable)                = 訊息內容 UTF-8       │
├─────────────────────────────────────────────────────────┤
│ Acknowledgements Count (int32)    = 確認數量             │
├─────────────────────────────────────────────────────────┤
│ Acknowledgements VarInt Length    = VarInt(總長度)       │
├─────────────────────────────────────────────────────────┤
│ Acknowledgements (variable)       = 串接的簽名           │
└─────────────────────────────────────────────────────────┘
```

### 實現

```go
// pkg/crypto/signer.go

func (s *ChatSigner) SignChatMessage(
    message string,
    timestamp int64,
    salt int64,
    acknowledgements [][]byte,
) ([]byte, error) {
    // 1. 編碼訊息
    encoded := s.encodeMessageForSigning(message, timestamp, salt, acknowledgements)

    // 2. SHA256 雜湊
    hashed := sha256.Sum256(encoded)

    // 3. RSA 簽名
    signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hashed[:])

    // 4. 遞增 session index
    s.sessionIndex++

    return signature, nil
}
```

### 關鍵細節

1. **Timestamp 除以 1000**: Minecraft 使用秒而非毫秒
2. **VarInt 長度**: 訊息和 acknowledgements 都需要 VarInt 長度前綴
3. **Session Index**: 每條訊息遞增，用於防重放攻擊
4. **Acknowledgements**: 必須包含在簽名數據中

## 2. lastSeen 追蹤 (chatState)

### 數據結構

```go
// pkg/game/player/chat_state.go

type seenMsg struct {
    signature []byte  // 訊息簽名
    pending   bool    // 是否待確認
}

type chatState struct {
    pending  int32                      // 待確認數量
    offset   int                        // 環形緩衝區偏移
    lastSeen [20]*seenMsg              // 最近 20 條訊息
}
```

### 環形緩衝區

```
初始狀態:
offset=0, pending=0
[nil, nil, nil, ..., nil]  (20 個)

收到第一條訊息 (sig1):
offset=1, pending=1
[sig1, nil, nil, ..., nil]

收到第二條訊息 (sig2):
offset=2, pending=2
[sig1, sig2, nil, ..., nil]

...

收到第 21 條訊息 (sig21):
offset=1, pending=21 (會自動發送 ACK)
[sig21, sig2, sig3, ..., sig20]  ← sig1 被覆蓋
```

### 實現

```go
func (s *chatState) IncSeen(signature []byte) {
    // 複製簽名
    copied := make([]byte, len(signature))
    copy(copied, signature)

    // 寫入環形緩衝區
    s.lastSeen[s.offset] = &seenMsg{
        signature: copied,
        pending:   true,
    }

    // 遞增
    s.offset = (s.offset + 1) % 20
    s.pending++
}
```

## 3. Acknowledgement 系統

### Bitset 格式

20-bit 固定長度位元組（3 bytes），每個 bit 對應一個 lastSeen 訊息：

```
Byte 0: [b7 b6 b5 b4 b3 b2 b1 b0]  ← 訊息 0-7
Byte 1: [b7 b6 b5 b4 b3 b2 b1 b0]  ← 訊息 8-15
Byte 2: [-- -- -- -- b3 b2 b1 b0]  ← 訊息 16-19
```

### GetAcknowledgements 實現

```go
func (s *chatState) GetAcknowledgements() ([][]byte, pk.FixedBitSet) {
    bitset := pk.NewFixedBitSet(20)
    acknowledgements := make([][]byte, 0, 20)

    // 從 offset 開始遍歷（最舊到最新）
    for i := 0; i < 20; i++ {
        idx := (s.offset + i) % 20
        msg := s.lastSeen[idx]

        if msg != nil {
            bitset.Set(i, true)
            acknowledgements = append(acknowledgements, msg.signature)
            msg.pending = false  // 標記已確認
        }
    }

    return acknowledgements, bitset
}
```

### Checksum 算法

使用 Java `Arrays.hashCode` 算法：

```go
func (s *chatState) Checksum() int8 {
    var checksum int32 = 1

    // 對每個簽名計算 hash
    for i := 0; i < 20; i++ {
        idx := (s.offset + i) % 20
        msg := s.lastSeen[idx]

        if msg != nil && len(msg.signature) > 0 {
            // 簽名的 hash
            var sigHash int32 = 1
            for _, b := range msg.signature {
                sigHash = 31*sigHash + int32(b)
            }
            // 累積到 checksum
            checksum = 31*checksum + sigHash
        }
    }

    // 取低 8 位，避免 0
    result := byte(checksum & 0xFF)
    if result == 0 {
        result = 1
    }
    return int8(result)
}
```

## 4. 完整流程

### 發送訊息

```go
func (p *Player) Chat(msg string) error {
    // 1. 生成時間戳和鹽值
    ts := time.Now().UnixMilli()
    salt := rand.Int63()

    // 2. 獲取 acknowledgements
    acknowledgements, ackBitset := p.chat.GetAcknowledgements()

    // 3. 創建封包
    chatPacket := &server.Chat{
        Message:      msg,
        Timestamp:    ts,
        Salt:         salt,
        Offset:       p.chat.NextOffset(),    // pending 計數
        Checksum:     p.chat.Checksum(),      // lastSeen checksum
        Acknowledged: ackBitset,              // 20-bit bitset
    }

    // 4. 簽名（如果有 signer）
    if p.signer != nil && !p.signer.IsExpired() {
        signature, _ := p.signer.SignChatMessage(msg, ts, salt, acknowledgements)
        chatPacket.HasSignature = true
        chatPacket.Signature = signature
    }

    // 5. 重置 pending
    p.chat.ResetPending()

    // 6. 發送
    return p.c.WritePacket(context.Background(), chatPacket)
}
```

### 接收訊息

```go
bot.AddHandler(c, func(ctx context.Context, p *client.PlayerChat) {
    // 1. 只追蹤有簽名的訊息
    if p.HasSignature && len(p.Signature) > 0 {
        pl.chat.IncSeen(p.Signature)

        // 2. 自動 ACK（當 pending > 64）
        if pl.chat.Pending() > 64 {
            c.WritePacket(ctx, &server.ChatAck{
                MessageCount: pl.chat.Pending(),
            })
            pl.chat.ResetPending()
        }
    }
})
```

## 5. 關鍵實現細節

### 與 mineflayer 的對應

| mineflayer (chat.js) | nalago-mc | 說明 |
|---------------------|-----------|------|
| `client._lastSeenMessages` | `chatState` | lastSeen 狀態 |
| `LastSeenMessages.push()` | `IncSeen()` | 追蹤新訊息 |
| `pending++` (line 566) | `pending++` | 增加計數 |
| `getAcknowledgements()` | `GetAcknowledgements()` | 取得確認 |
| `computeChatChecksum()` | `Checksum()` | 計算校驗和 |
| `pending = 0` (line 415) | `ResetPending()` | 重置計數 |
| Auto-ACK (line 227-232) | Auto-ACK (player.go:76-81) | 自動確認 |

### 為什麼需要這些機制？

1. **簽名**: 防止訊息偽造
2. **Session Index**: 防重放攻擊
3. **lastSeen**: 證明客戶端確實看到了訊息
4. **Acknowledgement**: 告訴伺服器哪些訊息已確認
5. **Checksum**: 快速驗證 lastSeen 完整性
6. **Auto-ACK**: 避免 pending 無限增長

## 6. 疑難排解

### "An internal error occurred"

**原因**: 簽名驗證失敗

**檢查**:
1. ✅ 簽名格式是否正確（包含所有欄位）
2. ✅ Timestamp 是否除以 1000
3. ✅ Acknowledgements 是否包含在簽名數據中
4. ✅ Session Index 是否正確遞增
5. ✅ VarInt 長度是否正確編碼

### 第二條訊息失敗

**原因**: pending 沒有重置

**修復**: 確保在發送後調用 `ResetPending()`

```go
// ✅ 正確
p.chat.ResetPending()
return p.c.WritePacket(ctx, chatPacket)

// ❌ 錯誤
return p.c.WritePacket(ctx, chatPacket)
// 忘記 ResetPending()
```

### Checksum 不匹配

**原因**: lastSeen 順序錯誤

**修復**: 使用環形緩衝區正確遍歷

```go
// ✅ 正確 - 從 offset 開始
for i := 0; i < 20; i++ {
    idx := (s.offset + i) % 20
    // ...
}

// ❌ 錯誤 - 直接遍歷
for i := 0; i < 20; i++ {
    // ...
}
```

## 7. 調試工具

### 啟用聊天調試

```go
// pkg/crypto/debug.go
package crypto

var chatSignDebug = true  // 啟用簽名調試

// pkg/game/player/debug.go
package player

var chatDebug = true  // 啟用聊天調試
```

### 調試輸出

```
[CHAT-SIGN] msg='hello' ts=1234567890 salt=123 idx=0 acks=0 payload=01000... siglen=256
[CHAT-DBG] msg='hello' signed=true offset=0 checksum=42 ack=000000 siglen=256
```

## 8. 參考資料

- [Mineflayer 實現](https://github.com/PrismarineJS/mineflayer/blob/master/lib/plugins/chat.js)
- [minecraft-protocol chat.js](https://github.com/PrismarineJS/node-minecraft-protocol/blob/master/src/client/chat.js)
- [Minecraft Protocol Wiki](https://wiki.vg/Protocol#Chat_Message_.28serverbound.29)
- [Java Arrays.hashCode](https://docs.oracle.com/javase/8/docs/api/java/util/Arrays.html#hashCode-byte:A-)

## 9. 總結

nalago-mc 的聊天系統實現：

✅ 100% 兼容 mineflayer
✅ 完整 RSA-SHA256 簽名
✅ 正確的 lastSeen 追蹤
✅ 自動 acknowledgement
✅ Java Arrays.hashCode checksum
✅ 自動 ACK 機制（pending > 64）

所有實現細節都經過與 mineflayer 對比驗證，確保協議兼容性。
