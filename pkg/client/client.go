package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	mcnet "git.konjactw.dev/falloutBot/go-mc/net"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

	"git.konjactw.dev/patyhank/minego/pkg/auth"
	"git.konjactw.dev/patyhank/minego/pkg/bot"
	"git.konjactw.dev/patyhank/minego/pkg/crypto"
	"git.konjactw.dev/patyhank/minego/pkg/game/inventory"
	"git.konjactw.dev/patyhank/minego/pkg/game/player"
	"git.konjactw.dev/patyhank/minego/pkg/game/world"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/server"
)

// handshakeProtocol is the protocol number used in handshake. Default 772.
// Use SetProtocolVersion to override before calling Connect.
var handshakeProtocol int32 = 772
var disableAbilities bool

// handshakeExtra 用於附加到握手主機欄位（例如 Proxy Forwarding Secret）。
// 留空則不附加。
var handshakeExtra string

// SetProtocolVersion overrides the handshake protocol version (must be >0).
func SetProtocolVersion(v int32) {
	if v > 0 {
		handshakeProtocol = v
	}
}

// SetHandshakeExtra 設定握手時附加在 host 後的額外字串（通常為代理轉發密鑰）。
// 若伺服器未要求，請保持空字串避免斷線。
func SetHandshakeExtra(extra string) {
	handshakeExtra = extra
}

// SetDisableAbilities 控制是否攔截 PlayerAbilities 封包送出
func SetDisableAbilities(disable bool) {
	disableAbilities = disable
}

type botClient struct {
	conn          *mcnet.Conn
	packetHandler *packetHandler
	eventHandler  bot.EventHandler
	connected     bool
	authProvider  auth.Provider
	recorder      *packetRecorder
	recState      string
	chatKeys      *ChatKeys

	inventory *inventory.Manager
	world     *world.World
	player    *player.Player

	chatSessionSent bool
}

// packetRecorder 簡易登入階段封包記錄器
type packetRecorder struct {
	max     int
	timeout time.Duration
	file    string
	mu      sync.Mutex
	buf     []packetLog
	start   time.Time
	wrote   bool
}

type packetLog struct {
	Ts    int64  `json:"ts"`
	Dir   string `json:"dir"`
	ID    int32  `json:"id"`
	State string `json:"state"`
	Data  string `json:"data"` // hex 編碼的原始封包內容
}

func newPacketRecorderFromEnv() *packetRecorder {
	if os.Getenv("DUMP_LOGIN_PKT") != "1" {
		return nil
	}
	max := 500
	if v, err := strconv.Atoi(os.Getenv("LOGIN_DUMP_MAX")); err == nil && v > 0 {
		max = v
	}
	timeout := 30 * time.Second
	if v, err := strconv.Atoi(os.Getenv("LOGIN_DUMP_TIMEOUT")); err == nil && v > 0 {
		timeout = time.Duration(v) * time.Millisecond
	}
	file := os.Getenv("LOGIN_DUMP_FILE")
	if file == "" {
		file = "login_packets.json"
	}
	rec := &packetRecorder{max: max, timeout: timeout, file: file, start: time.Now()}
	go func() {
		time.Sleep(timeout)
		rec.Flush()
	}()
	return rec
}

func (p *packetRecorder) logPacket(dir, state string, pkt pk.Packet) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.wrote || len(p.buf) >= p.max {
		return
	}
	entry := packetLog{
		Ts:    time.Now().UnixMilli(),
		Dir:   dir,
		ID:    pkt.ID,
		State: state,
		Data:  fmt.Sprintf("%x", pkt.Data),
	}
	p.buf = append(p.buf, entry)
}

func (p *packetRecorder) Flush() {
	p.mu.Lock()
	if p.wrote || len(p.buf) == 0 {
		p.mu.Unlock()
		return
	}
	p.wrote = true
	data, _ := json.MarshalIndent(p.buf, "", "  ")
	p.mu.Unlock()
	_ = os.WriteFile(p.file, data, 0o644)
}

func (b *botClient) Player() bot.Player {
	return b.player
}

func (b *botClient) Close(ctx context.Context) error {
	if err := b.conn.Close(); err != nil {
		return err
	}
	if b.recorder != nil {
		b.recorder.Flush()
	}

	return nil
}

func (b *botClient) IsConnected() bool {
	return b.connected
}

func (b *botClient) WritePacket(ctx context.Context, packet server.ServerboundPacket) error {
	if disableAbilities {
		if packet.PacketID() == packetid.ServerboundPlayerAbilities {
			return nil
		}
	}
	pkt := pk.Marshal(packet.PacketID(), packet)
	b.logPacket("out", pkt)
	if err := b.conn.WritePacket(pkt); err != nil {
		return err
	}
	return nil
}

func (b *botClient) PacketHandler() bot.PacketHandler {
	return b.packetHandler
}

func (b *botClient) EventHandler() bot.EventHandler {
	return b.eventHandler
}

func (b *botClient) World() bot.World {
	return b.world
}

func (b *botClient) Inventory() bot.InventoryHandler {
	return b.inventory
}

func (b *botClient) logPacket(dir string, pkt pk.Packet) {
	if b.recorder == nil {
		return
	}
	b.recorder.logPacket(dir, b.recState, pkt)
}

func (b *botClient) sendPlayBrand() error {
	buf := &bytes.Buffer{}
	_, _ = pk.String("vanilla").WriteTo(buf)
	pkt := pk.Marshal(
		packetid.ServerboundCustomPayload,
		pk.Identifier("minecraft:brand"),
		pk.ByteArray(buf.Bytes()),
	)
	b.logPacket("out", pkt)
	return b.conn.WritePacket(pkt)
}

func (b *botClient) setRecState(state string) {
	if b.recorder == nil {
		return
	}
	b.recState = state
}

func (b *botClient) Connect(ctx context.Context, addr string, options *bot.ConnectOptions) error {
	host, portStr, err := net.SplitHostPort(addr)
	var port uint64
	if err != nil {
		var addrErr *net.AddrError
		const missingPort = "missing port in address"
		if errors.As(err, &addrErr) && addrErr.Err == missingPort {
			host = addr
			port = 25565
		} else {
			return err
		}
	} else {
		port, err = strconv.ParseUint(portStr, 0, 16)
		if err != nil {
			return err
		}
	}

	var dialer mcnet.MCDialer = &mcnet.DefaultDialer
	if options != nil && options.Proxy != nil {
		dialer, err = socks5(options.Proxy)
		if err != nil {
			return err
		}
	}
	baseConn, err := dialer.DialMCContext(ctx, addr)
	if err != nil {
		return err
	}

	b.conn = baseConn
	b.recState = "handshaking"

	if options != nil && options.FakeHost != "" {
		host = options.FakeHost
	}

	err = b.handshake(host, port)
	if err != nil {
		return err
	}
	b.setRecState("login")

	err = b.login()
	if err != nil {
		return err
	}
	b.setRecState("configuration")

	err = b.configuration()
	if err != nil {
		return err
	}
	// play 階段再送一次 brand custom payload，貼近原版/ mineflayer
	if err := b.sendPlayBrand(); err != nil {
		return err
	}
	b.setRecState("play")

	b.connected = true

	return nil
}

func (b *botClient) HandleGame(ctx context.Context) error {
	// 進入 play 前先送 chat_session_update（若有密鑰）
	b.sendChatSessionUpdate()

	// 初始化 chat signer（用於 Minecraft 1.19+ 聊天簽名）
	if b.chatKeys != nil && len(b.chatKeys.PrivateDER) > 0 {
		// Get player UUID from auth provider
		profile := b.authProvider.FetchProfile(ctx)
		signer, err := crypto.NewChatSigner(
			b.chatKeys.PrivateDER,
			profile.UUID,
			b.chatKeys.SessionID,
		)
		if err == nil {
			b.player.SetChatSigner(signer)
		}
		// 如果創建失敗，靜默忽略（回退到 unsigned chat）
	}

	return b.handlePackets(ctx)
}

func (b *botClient) handshake(host string, port uint64) error {
	// 若設定了附加字串，依照部分代理/轉發插件要求附加於 host（以 NUL 分隔）。
	if handshakeExtra != "" {
		host = host + "\x00" + handshakeExtra
	}
	pkt := pk.Marshal(
		0,
		pk.VarInt(handshakeProtocol),
		pk.String(host),
		pk.UnsignedShort(port),
		pk.VarInt(2), // to game state
	)
	b.logPacket("out", pkt)
	return b.conn.WritePacket(pkt)
}

func (b *botClient) handlePackets(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(15)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var p pk.Packet
			if err := b.conn.ReadPacket(&p); err != nil {
				return err
			}
			b.logPacket("in", p)
			pktID := packetid.ClientboundPacketID(p.ID)
			if pktID == packetid.ClientboundStartConfiguration {
				err := b.conn.WritePacket(pk.Marshal(packetid.ServerboundConfigurationAcknowledged))
				if err != nil {
					return err
				}
				err = b.configuration()
				if err != nil {
					return err
				}
				continue
			}

			hs, ok := b.packetHandler.rawMap[pktID]
			for _, h := range hs {
				group.Go(func() error {
					h(ctx, p)
					return nil
				})
			}

			creator, ok := client.ClientboundPackets[pktID]
			if !ok {
				continue
			}
			pkt := creator()
			_, err := pkt.ReadFrom(bytes.NewReader(p.Data))
			if err != nil {
				fmt.Printf("Decoding: 0x%x %s %s\n", p.ID, pktID.String(), err.Error())
				continue
			}
			b.packetHandler.HandlePacket(ctx, pkt)
		}
	}
}

func NewClient(options *bot.ClientOptions) bot.Client {
	c := &botClient{
		packetHandler: newPacketHandler(),
		eventHandler:  NewEventHandler(),
		authProvider:  options.AuthProvider,
		recorder:      newPacketRecorderFromEnv(),
		chatKeys:      chatKeys,
	}

	if options.AuthProvider == nil {
		c.authProvider = &auth.OfflineAuth{Username: "Steve"}
	}

	c.world = world.NewWorld(c)
	c.inventory = inventory.NewManager(c)
	c.player = player.New(c)

	return c
}

func (b *botClient) sendChatSessionUpdate() {
	if b.chatSessionSent || b.chatKeys == nil || b.conn == nil {
		return
	}
	pkt := pk.Marshal(
		packetid.ServerboundChatSessionUpdate,
		pk.UUID(b.chatKeys.SessionID),
		b.chatKeys.Public,
	)
	b.logPacket("out", pkt)
	_ = b.conn.WritePacket(pkt)
	b.chatSessionSent = true
}
