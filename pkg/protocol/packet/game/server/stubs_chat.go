package server

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
)

type SignedSignatures struct {
	Name      string
	Signature []byte
}

type Chat struct {
	Message      string
	Timestamp    int64
	Salt         int64
	HasSignature bool
	Signature    []byte
	Offset       int32
	Checksum     int8
	Acknowledged []byte
}

func (*Chat) PacketID() packetid.ServerboundPacketID { return 0 }
func (*Chat) ReadFrom(io.Reader) (int64, error)      { return 0, nil }
func (Chat) WriteTo(io.Writer) (int64, error)        { return 0, nil }
