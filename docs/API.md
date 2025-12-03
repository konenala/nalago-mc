# nalago-mc API 參考

完整的 API 文檔，包含所有公開接口。

## 目錄

- [Client](#client)
- [Player](#player)
- [World](#world)
- [Inventory](#inventory)
- [Auth](#auth)
- [Event System](#event-system)

## Client

### bot.Client

主要客戶端接口。

```go
type Client interface {
    Connect(ctx context.Context, addr string, opts *ConnectOptions) error
    HandleGame(ctx context.Context) error
    Close(ctx context.Context) error

    IsConnected() bool
    Player() *player.Player
    World() *world.World
    Inventory() InventoryHandler
    PacketHandler() *PacketHandler

    WritePacket(ctx context.Context, pk ServerboundPacket) error
}
```

#### Connect

連接到 Minecraft 伺服器。

```go
func (c *Client) Connect(ctx context.Context, addr string, opts *ConnectOptions) error
```

**參數**:
- `ctx`: Context，用於取消操作
- `addr`: 伺服器地址 (格式: `host:port`)
- `opts`: 連線選項 (可為 nil)

**ConnectOptions**:
```go
type ConnectOptions struct {
    FakeHost string  // 虛擬主機（用於代理伺服器）
}
```

**範例**:
```go
// 基本連線
err := client.Connect(ctx, "localhost:25565", nil)

// 使用虛擬主機
err := client.Connect(ctx, "proxy.example.com:25565", &bot.ConnectOptions{
    FakeHost: "lobby.example.com",
})
```

#### HandleGame

處理遊戲封包（阻塞直到斷線）。

```go
func (c *Client) HandleGame(ctx context.Context) error
```

**範例**:
```go
if err := client.HandleGame(ctx); err != nil {
    log.Printf("遊戲結束: %v", err)
}
```

#### WritePacket

發送封包到伺服器。

```go
func (c *Client) WritePacket(ctx context.Context, pk ServerboundPacket) error
```

**範例**:
```go
packet := &server.Chat{
    Message: "Hello!",
    // ...
}
client.WritePacket(ctx, packet)
```

---

## Player

### player.Player

玩家控制接口。

```go
type Player struct {
    // 私有欄位
}
```

#### 移動方法

##### FlyTo

直線飛行到指定位置。

```go
func (p *Player) FlyTo(pos mgl64.Vec3) error
```

**參數**:
- `pos`: 目標位置 (Vec3{X, Y, Z})

**範例**:
```go
target := mgl64.Vec3{100, 70, 200}
err := player.FlyTo(target)
```

##### WalkTo

使用 A* 演算法步行到指定位置。

```go
func (p *Player) WalkTo(pos mgl64.Vec3) error
```

**注意**: 需要地形數據，僅在已加載區塊中有效。

##### LookAt

看向指定位置。

```go
func (p *Player) LookAt(target mgl64.Vec3) error
```

**範例**:
```go
entity := world.GetEntity(123)
player.LookAt(entity.Position())
```

#### 方塊操作

##### BreakBlock

挖掘方塊。

```go
func (p *Player) BreakBlock(pos protocol.Position) error
```

**參數**:
- `pos`: 方塊位置 `protocol.Position{X, Y, Z}`

**範例**:
```go
blockPos := protocol.Position{100, 64, 200}
err := player.BreakBlock(blockPos)
```

##### PlaceBlock

放置方塊。

```go
func (p *Player) PlaceBlock(pos protocol.Position) error
```

##### PlaceBlockWithArgs

指定面和游標位置放置方塊。

```go
func (p *Player) PlaceBlockWithArgs(pos protocol.Position, face int32, cursor mgl64.Vec3) error
```

**參數**:
- `pos`: 目標方塊位置
- `face`: 面向 (0=下, 1=上, 2=北, 3=南, 4=西, 5=東)
- `cursor`: 游標位置 (0.0-1.0)

**範例**:
```go
// 在方塊上方中心放置
player.PlaceBlockWithArgs(
    protocol.Position{100, 64, 200},
    1,  // 上面
    mgl64.Vec3{0.5, 0.5, 0.5},
)
```

#### 聊天方法

##### Chat

發送聊天訊息。

```go
func (p *Player) Chat(msg string) error
```

**範例**:
```go
player.Chat("Hello, world!")
```

**注意**: 如果訊息以 `/` 開頭，會自動轉為指令。

##### Command

發送指令（不含 `/` 前綴）。

```go
func (p *Player) Command(msg string) error
```

**範例**:
```go
player.Command("gamemode creative")
player.Command("tp 100 64 200")
```

#### 容器操作

##### OpenContainer

打開指定位置的容器。

```go
func (p *Player) OpenContainer(pos protocol.Position) (bot.Container, error)
```

**返回**: Container 接口

**範例**:
```go
chestPos := protocol.Position{100, 64, 200}
container, err := player.OpenContainer(chestPos)
if err != nil {
    return err
}

// 使用 container
id := container.ID()
```

##### UseItem

使用手中物品。

```go
func (p *Player) UseItem(hand int8) error
```

**參數**:
- `hand`: 0=主手, 1=副手

#### 實體訪問

##### Entity

獲取玩家實體。

```go
func (p *Player) Entity() bot.Entity
```

**範例**:
```go
entity := player.Entity()
pos := entity.Position()
fmt.Printf("位置: %.2f, %.2f, %.2f\n", pos.X(), pos.Y(), pos.Z())
```

---

## World

### world.World

世界狀態管理。

```go
type World struct {
    // 私有欄位
}
```

#### GetBlock

獲取指定位置的方塊。

```go
func (w *World) GetBlock(pos protocol.Position) (*Block, error)
```

**返回**: Block 資訊

**範例**:
```go
block, err := world.GetBlock(protocol.Position{100, 64, 200})
if err != nil {
    return err
}
fmt.Printf("方塊: %s (ID: %d)\n", block.ID(), block.StateID())
```

#### Entities

獲取所有實體。

```go
func (w *World) Entities() []bot.Entity
```

**範例**:
```go
for _, entity := range world.Entities() {
    fmt.Printf("實體 #%d 在 %v\n", entity.ID(), entity.Position())
}
```

---

## Inventory

### inventory.Manager

背包管理器。

```go
type Manager struct {
    // 私有欄位
}
```

#### Click

點擊物品槽。

```go
func (m *Manager) Click(windowID int8, slot int16, mode int32, button int32) error
```

**參數**:
- `windowID`: 視窗 ID (0=玩家背包)
- `slot`: 物品槽編號
- `mode`: 點擊模式
  - `0` = 左/右鍵
  - `1` = Shift 點擊
  - `2` = 數字鍵
  - `4` = 丟棄
- `button`: 按鈕
  - `0` = 左鍵
  - `1` = 右鍵

**範例**:
```go
inv := client.Inventory().(*inventory.Manager)

// 左鍵點擊第一格
inv.Click(0, 0, 0, 0)

// 丟棄第 36 格物品
inv.Click(0, 36, 4, 1)
```

#### CurrentContainerID

獲取當前打開的容器 ID。

```go
func (m *Manager) CurrentContainerID() int8
```

#### Container

獲取當前容器。

```go
func (m *Manager) Container() bot.Container
```

---

## Auth

### auth.Provider

驗證提供者接口。

```go
type Provider interface {
    Authenticate(ctx context.Context, conn *net.Conn, content client.LoginHello) error
    FetchProfile(ctx context.Context) *Profile
}
```

### OfflineAuth

離線模式驗證。

```go
type OfflineAuth struct {
    Username string
}
```

**範例**:
```go
auth := &auth.OfflineAuth{
    Username: "TestBot",
}
```

### OnlineAuth

Microsoft 線上驗證。

```go
type OnlineAuth struct {
    AccessToken string
    Profile     Profile
}

type Profile struct {
    Name string
    UUID uuid.UUID
}
```

**範例**:
```go
auth := &auth.OnlineAuth{
    AccessToken: "eyJ0eXAiOiJKV1Qi...",
    Profile: auth.Profile{
        Name: "PlayerName",
        UUID: uuid.MustParse("12345678-1234-1234-1234-123456789abc"),
    },
}
```

---

## Event System

### 訂閱事件

使用泛型訂閱事件。

```go
func SubscribeEvent[T any](c bot.Client, handler func(T) error) Subscription
```

**範例**:
```go
import "git.konjactw.dev/patyhank/minego/pkg/game/player"

// 訂閱聊天事件
bot.SubscribeEvent(client, func(e player.MessageEvent) error {
    fmt.Printf("聊天: %s\n", e.Message.String())
    return nil
})
```

### 發佈事件

```go
func PublishEvent[T any](c bot.Client, event T)
```

**範例**:
```go
bot.PublishEvent(client, MyCustomEvent{Data: "example"})
```

### 封包處理器

#### AddHandler

類型安全的封包處理器。

```go
func AddHandler[T client.ClientboundPacket](c bot.Client, handler func(ctx context.Context, p T))
```

**範例**:
```go
bot.AddHandler(client, func(ctx context.Context, p *client.PlayerPosition) {
    fmt.Printf("位置更新: %.2f, %.2f, %.2f\n", p.X, p.Y, p.Z)
})
```

#### AddGenericPacketHandler

通用封包處理器。

```go
func (h *PacketHandler) AddGenericPacketHandler(handler func(ctx context.Context, pk client.ClientboundPacket))
```

**範例**:
```go
client.PacketHandler().AddGenericPacketHandler(func(ctx context.Context, pk client.ClientboundPacket) {
    switch p := pk.(type) {
    case *client.PlayerChat:
        fmt.Printf("玩家聊天: %s\n", p.PlainMessage)
    case *client.SystemChatMessage:
        fmt.Printf("系統訊息: %s\n", p.Content.String())
    }
})
```

---

## 常用事件類型

### player.MessageEvent

聊天訊息事件。

```go
type MessageEvent struct {
    Message chat.Message
}
```

### inventory.ContainerOpenEvent

容器開啟事件。

```go
type ContainerOpenEvent struct {
    WindowID int8
    Title    chat.Message
}
```

---

## 協議相關

### SetProtocolVersion

設置協議版本號。

```go
func SetProtocolVersion(version int32)
```

**支援版本**:
- `767`: Minecraft 1.21.0
- `768`: Minecraft 1.21.4
- `769`: Minecraft 1.21.5
- `770`: Minecraft 1.21.6
- `771`: Minecraft 1.21.7
- `772`: Minecraft 1.21.8
- `773`: Minecraft 1.21.9
- `774`: Minecraft 1.21.10

**範例**:
```go
client.SetProtocolVersion(767)  // 1.21.0
```

### SetChatProfileKeys

設置聊天簽名金鑰。

```go
func SetChatProfileKeys(keys *ChatKeys)

type ChatKeys struct {
    PrivateDER []byte      // PKCS#8 DER 格式私鑰
    PublicDER  []byte      // X.509 DER 格式公鑰
    SessionID  uuid.UUID   // 會話 UUID
    ExpiresAt  time.Time   // 過期時間
}
```

---

## 類型定義

### protocol.Position

方塊位置。

```go
type Position [3]int

// 範例
pos := protocol.Position{100, 64, 200}
x, y, z := pos[0], pos[1], pos[2]
```

### mgl64.Vec3

3D 向量（浮點數）。

```go
type Vec3 struct {
    X, Y, Z float64
}

// 範例
pos := mgl64.Vec3{100.5, 64.0, 200.5}
```

### chat.Message

聊天訊息（支援格式化）。

```go
type Message interface {
    String() string
}
```

---

## 錯誤處理

常見錯誤類型：

```go
var (
    ErrNotConnected     = errors.New("not connected")
    ErrAlreadyConnected = errors.New("already connected")
    ErrTimeout          = errors.New("timeout")
    ErrInvalidPacket    = errors.New("invalid packet")
)
```

---

## 完整範例

查看 [examples/](../examples/) 目錄獲取完整可運行範例：

- `simple/` - 基本連線
- `chat_bot/` - 聊天機器人
- `builder/` - 自動建造
- `miner/` - 自動挖礦

## 更多資源

- [快速開始](GETTING_STARTED.md)
- [聊天系統詳解](CHAT_SYSTEM.md)
- [GitHub Issues](https://github.com/user/nalago-mc/issues)
