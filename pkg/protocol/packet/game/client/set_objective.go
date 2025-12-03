package client

import "git.konjactw.dev/falloutBot/go-mc/nbt"

//codec:gen
type ObjectivesData struct {
	DisplayText     nbt.RawMessage `mc:"NBT"`
	HasNumberFormat bool
	NumberFormat    int32 `mc:"VarInt"`
	HasStyling      bool
	Styling         nbt.RawMessage `mc:"NBT"`
}

type ObjectivesCreateData struct {
	ObjectivesData
}

type ObjectivesUpdateData struct {
	ObjectivesData
}

//codec:gen
type UpdateObjectives struct {
	ObjectiveName string
	Mode          int8
	//opt:enum:Mode:0
	Create ObjectivesCreateData
	//opt:enum:Mode:2
	Update ObjectivesUpdateData
}
