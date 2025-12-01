package slot

import (
	"io"

	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

type Component interface {
	Type() ComponentID
	ID() string

	pk.Field
}

type ComponentID int32
type componentCreator func() Component

var components = make(map[ComponentID]componentCreator)

func ComponentFromID(id ComponentID) Component {
	if c, ok := components[id]; ok && c != nil {
		return c()
	}
	// 未知元件：回傳可安全跳過的空元件，避免解碼 panic
	return &UnknownComponent{id: id}
}

func RegisterComponent(c componentCreator) {
	components[c().Type()] = c
}

// UnknownComponent 佔位，避免對未知 component id 造成 panic。
// 目前假設該 component 無額外 payload，ReadFrom/WriteTo 為 no-op。
type UnknownComponent struct {
	id ComponentID
}

func (u *UnknownComponent) Type() ComponentID { return u.id }
func (u *UnknownComponent) ID() string        { return "" }

func (u *UnknownComponent) WriteTo(w io.Writer) (int64, error)  { return 0, nil }
func (u *UnknownComponent) ReadFrom(r io.Reader) (int64, error) { return 0, nil }
