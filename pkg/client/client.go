package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"golang.org/x/sync/errgroup"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	mcnet "git.konjactw.dev/falloutBot/go-mc/net"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"

	"git.konjactw.dev/patyhank/minego/pkg/auth"
	"git.konjactw.dev/patyhank/minego/pkg/bot"
	"git.konjactw.dev/patyhank/minego/pkg/game/inventory"
	"git.konjactw.dev/patyhank/minego/pkg/game/player"
	"git.konjactw.dev/patyhank/minego/pkg/game/world"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/server"
)

// handshakeProtocol is the protocol number used in handshake. Default 772.
// Use SetProtocolVersion to override before calling Connect.
var handshakeProtocol int32 = 772

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

type botClient struct {
	conn          *mcnet.Conn
	packetHandler *packetHandler
	eventHandler  bot.EventHandler
	connected     bool
	authProvider  auth.Provider

	inventory *inventory.Manager
	world     *world.World
	player    *player.Player
}

func (b *botClient) Player() bot.Player {
	return b.player
}

func (b *botClient) Close(ctx context.Context) error {
	if err := b.conn.Close(); err != nil {
		return err
	}

	return nil
}

func (b *botClient) IsConnected() bool {
	return b.connected
}

func (b *botClient) WritePacket(ctx context.Context, packet server.ServerboundPacket) error {
	err := b.conn.WritePacket(pk.Marshal(packet.PacketID(), packet))
	if err != nil {
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
	b.conn, err = dialer.DialMCContext(ctx, addr)
	if err != nil {
		return err
	}

	if options != nil && options.FakeHost != "" {
		host = options.FakeHost
	}

	err = b.handshake(host, port)
	if err != nil {
		return err
	}

	err = b.login()
	if err != nil {
		return err
	}

	err = b.configuration()
	if err != nil {
		return err
	}

	b.connected = true

	return nil
}

func (b *botClient) HandleGame(ctx context.Context) error {
	return b.handlePackets(ctx)
}

func (b *botClient) handshake(host string, port uint64) error {
	// 若設定了附加字串，依照部分代理/轉發插件要求附加於 host（以 NUL 分隔）。
	if handshakeExtra != "" {
		host = host + "\x00" + handshakeExtra
	}

	return b.conn.WritePacket(pk.Marshal(
		0,
		pk.VarInt(handshakeProtocol),
		pk.String(host),
		pk.UnsignedShort(port),
		pk.VarInt(2), // to game state
	))
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
	}

	if options.AuthProvider == nil {
		c.authProvider = &auth.OfflineAuth{Username: "Steve"}
	}

	c.world = world.NewWorld(c)
	c.inventory = inventory.NewManager(c)
	c.player = player.New(c)

	return c
}
