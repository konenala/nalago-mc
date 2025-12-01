package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"git.konjactw.dev/falloutBot/go-mc/chat"
	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"

	"git.konjactw.dev/patyhank/minego/pkg/auth"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
)

func (b *botClient) login() error {
	a := &auth.Auth{
		Conn:     b.conn,
		Provider: b.authProvider,
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	return a.HandleLogin(ctx)
}

func (b *botClient) configuration() (err error) {
	// 進入 configuration 狀態後先送一次 ClientInformation，否則部分伺服器會在發送 registry/feature 前直接踢人
	pkt := pk.Marshal(
		packetid.ServerboundConfigClientInformation,
		pk.String("en_us"),    // Location
		pk.Byte(10),           // ViewDistance
		pk.VarInt(0),          // ChatMode: enabled
		pk.Boolean(true),      // ChatColor
		pk.UnsignedByte(0x7f), // DisplayedSkinParts (全開)
		pk.VarInt(1),          // MainHand: right
		pk.Boolean(false),     // EnableTextFiltering
		pk.Boolean(true),      // AllowListing
		pk.VarInt(0),          // ParticleStatus: all
	)
	b.logPacket("out", pkt)
	if err = b.conn.WritePacket(pkt); err != nil {
		return err
	}

	// 送出 brand custom payload（minecraft:brand）
	{
		buf := &bytes.Buffer{}
		_, _ = pk.String("vanilla").WriteTo(buf)
		pkt = pk.Marshal(
			packetid.ServerboundConfigCustomPayload,
			pk.Identifier("minecraft:brand"),
			pk.ByteArray(buf.Bytes()),
		)
		b.logPacket("out", pkt)
		if err = b.conn.WritePacket(pkt); err != nil {
			return err
		}
	}

	var p pk.Packet
	for {
		err = b.conn.ReadPacket(&p)
		b.logPacket("in", p)

		switch packetid.ClientboundPacketID(p.ID) {
		case packetid.ClientboundConfigDisconnect:
			var reason chat.Message
			err = p.Scan(&reason)
			if err != nil {
				return err
			}
			fmt.Printf("[CONFIG] Disconnected: %s\n", reason.String())
			return errors.New("kicked: " + reason.String())
		case packetid.ClientboundConfigFinishConfiguration:
			pkt = pk.Marshal(packetid.ServerboundConfigFinishConfiguration)
			b.logPacket("out", pkt)
			err = b.conn.WritePacket(pkt)
			return err
		case packetid.ClientboundConfigKeepAlive:
			var keepAliveID pk.Long
			err = p.Scan(&keepAliveID)
			if err != nil {
				return err
			}
			pkt = pk.Marshal(packetid.ServerboundConfigKeepAlive, keepAliveID)
			b.logPacket("out", pkt)
			err = b.conn.WritePacket(pkt)
			if err != nil {
				return err
			}
		case packetid.ClientboundConfigPing:
			var pingID pk.Int
			err = p.Scan(&pingID)
			if err != nil {
				return err
			}
			pkt = pk.Marshal(packetid.ServerboundConfigPong, pingID)
			b.logPacket("out", pkt)
			err = b.conn.WritePacket(pkt)
			if err != nil {
				return err
			}

		case packetid.ClientboundConfigSelectKnownPacks:
			pkt = pk.Marshal(packetid.ServerboundConfigSelectKnownPacks, pk.VarInt(0))
			b.logPacket("out", pkt)
			err = b.conn.WritePacket(pkt)
			if err != nil {
				return err
			}
			time.Sleep(2 * time.Millisecond)
		case packetid.ClientboundConfigResourcePackPush:
			var pkt client.AddResourcePack
			err = p.Scan(&pkt)
			if err != nil {
				return err
			}
			u := pk.UUID(pkt.UUID)
			pktOut := pk.Marshal(packetid.ServerboundConfigResourcePack, u, pk.VarInt(3))
			b.logPacket("out", pktOut)
			if err = b.conn.WritePacket(pktOut); err != nil { // accepted
				return err
			}
			pktOut = pk.Marshal(packetid.ServerboundConfigResourcePack, u, pk.VarInt(4))
			b.logPacket("out", pktOut)
			if err = b.conn.WritePacket(pktOut); err != nil { // downloaded
				return err
			}
			pktOut = pk.Marshal(packetid.ServerboundConfigResourcePack, u, pk.VarInt(0))
			b.logPacket("out", pktOut)
			if err = b.conn.WritePacket(pktOut); err != nil { // successfully_loaded
				return err
			}
		case packetid.ClientboundConfigResourcePackPop:
			time.Sleep(2 * time.Millisecond)
			continue
		case packetid.ClientboundConfigUpdateEnabledFeatures,
			packetid.ClientboundConfigRegistryData,
			packetid.ClientboundConfigUpdateTags,
			packetid.ClientboundConfigCustomPayload,
			packetid.ClientboundConfigServerLinks,
			packetid.ClientboundConfigCustomReportDetails,
			packetid.ClientboundConfigResetChat,
			packetid.ClientboundConfigClearDialog,
			packetid.ClientboundConfigShowDialog,
			packetid.ClientboundConfigStoreCookie,
			packetid.ClientboundConfigTransfer,
			packetid.ClientboundConfigCookieRequest:
			// 模擬玩家節奏，避免短時間大量回應；亦避免未處理封包阻塞
			time.Sleep(5 * time.Millisecond)
			continue
		default:
			continue
		}
	}
}
