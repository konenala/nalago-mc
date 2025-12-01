package client

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// PlayerChat 對應 1.21 player_chat（clientbound），手寫解碼以取得簽章。
type PlayerChat struct {
	GlobalIndex      int32
	SenderUUID       pk.UUID
	Index            int32
	HasSignature     bool
	Signature        []byte
	PreviousMessages []PrevMsg
}

type PrevMsg struct {
	ID        int32
	HasSig    bool
	Signature []byte
}

func (*PlayerChat) PacketID() packetid.ClientboundPacketID {
	return packetid.ClientboundPlayerChat
}

// ReadFrom 只解需要的欄位（簽章），其餘略讀。
func (p *PlayerChat) ReadFrom(r io.Reader) (n int64, err error) {
	var temp int64
	// globalIndex
	temp, err = (*pk.VarInt)(&p.GlobalIndex).ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	// sender UUID
	temp, err = (*pk.UUID)(&p.SenderUUID).ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	// index
	temp, err = (*pk.VarInt)(&p.Index).ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	// signature option
	var hasSig pk.Boolean
	temp, err = hasSig.ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	p.HasSignature = bool(hasSig)
	if p.HasSignature {
		temp, err = (*pk.ByteArray)(&p.Signature).ReadFrom(r)
		n += temp
		if err != nil {
			return n, err
		}
	}
	// plainMessage (string)
	var plain pk.String
	temp, err = plain.ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	// timestamp
	var ts pk.Long
	temp, err = ts.ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	// salt
	var salt pk.Long
	temp, err = salt.ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	// previousMessages array
	var arrLen pk.VarInt
	temp, err = arrLen.ReadFrom(r)
	n += temp
	if err != nil {
		return n, err
	}
	for i := 0; i < int(arrLen); i++ {
		var msgID pk.VarInt
		temp2, err2 := msgID.ReadFrom(r)
		n += temp2
		if err2 != nil {
			return n, err2
		}
		var opt pk.Boolean
		temp2, err2 = opt.ReadFrom(r)
		n += temp2
		if err2 != nil {
			return n, err2
		}
		var sig []byte
		if opt {
			temp2, err2 = (*pk.ByteArray)(&sig).ReadFrom(r)
			n += temp2
			if err2 != nil {
				return n, err2
			}
		}
		p.PreviousMessages = append(p.PreviousMessages, PrevMsg{ID: int32(msgID), HasSig: bool(opt), Signature: sig})
	}
	// 之後的 unsignedChatContent / filterType / mask / type / networkName / networkTargetName 都略讀
	var rest pk.PluginMessageData
	temp, err = rest.ReadFrom(r)
	n += temp
	if err != nil && err != io.EOF {
		return n, err
	}
	return n, nil
}

func (p PlayerChat) WriteTo(w io.Writer) (n int64, err error) {
	return 0, io.ErrUnexpectedEOF // 不用寫回伺服器
}

func init() {
	registerPacket(func() ClientboundPacket {
		return &PlayerChat{}
	})
}
