package server

import "git.konjactw.dev/patyhank/minego/pkg/protocol/packetid"

//codec:gen
type LoginKey struct {
	SharedSecret []byte `mc:"ByteArray"`
	VerifyToken  []byte `mc:"ByteArray"`
}

func (*LoginKey) PacketID() packetid.ServerboundPacketID {
	return packetid.ServerboundLoginKey
}

func init() {
	registerPacket(packetid.ServerboundLoginKey, func() ServerboundPacket {
		return &LoginKey{}
	})
}
