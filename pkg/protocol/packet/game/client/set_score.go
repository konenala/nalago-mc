package client

import "git.konjactw.dev/falloutBot/go-mc/nbt"

//codec:gen
type UpdateScore struct {
	ItemName        string
	ScoreName       string
	Value           int32 `mc:"VarInt"`
	HasDisplayName  bool
	DisplayName     nbt.RawMessage `mc:"NBT"`
	HasNumberFormat bool
	NumberFormat    int32 `mc:"VarInt"`
	HasStyling      bool
	Styling         nbt.RawMessage `mc:"NBT"`
}
