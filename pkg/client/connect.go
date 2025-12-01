package client

import (
	"context"
	"errors"
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
	var p pk.Packet
	for {
		err = b.conn.ReadPacket(&p)

		switch packetid.ClientboundPacketID(p.ID) {
		case packetid.ClientboundConfigDisconnect:
			var reason chat.Message
			err = p.Scan(&reason)
			if err != nil {
				return err
			}
			return errors.New("kicked: " + reason.String())
		case packetid.ClientboundConfigFinishConfiguration:
			err = b.conn.WritePacket(pk.Marshal(
				packetid.ServerboundConfigFinishConfiguration,
			))
			return err
		case packetid.ClientboundConfigKeepAlive:
			var keepAliveID pk.Long
			err = p.Scan(&keepAliveID)
			if err != nil {
				return err
			}
			err = b.conn.WritePacket(pk.Marshal(packetid.ServerboundConfigKeepAlive, keepAliveID))
			if err != nil {
				return err
			}
		case packetid.ClientboundConfigPing:
			var pingID pk.Int
			err = p.Scan(&pingID)
			if err != nil {
				return err
			}
			err = b.conn.WritePacket(pk.Marshal(packetid.ServerboundConfigPong, pingID))
			if err != nil {
				return err
			}

		case packetid.ClientboundConfigSelectKnownPacks:
			err = b.conn.WritePacket(pk.Marshal(packetid.ServerboundConfigSelectKnownPacks, pk.VarInt(0)))
			if err != nil {
				return err
			}
		case packetid.ClientboundConfigResourcePackPush:
			var pkt client.AddResourcePack
			err = p.Scan(&pkt)
			if err != nil {
				return err
			}
			u := pk.UUID(pkt.UUID)
			if err = b.conn.WritePacket(pk.Marshal(packetid.ServerboundConfigResourcePack, u, pk.VarInt(3))); err != nil { // accepted
				return err
			}
			if err = b.conn.WritePacket(pk.Marshal(packetid.ServerboundConfigResourcePack, u, pk.VarInt(4))); err != nil { // downloaded
				return err
			}
			if err = b.conn.WritePacket(pk.Marshal(packetid.ServerboundConfigResourcePack, u, pk.VarInt(0))); err != nil { // successfully_loaded
				return err
			}
		case packetid.ClientboundConfigResourcePackPop:
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
			continue
		default:
			continue
		}
	}
}
