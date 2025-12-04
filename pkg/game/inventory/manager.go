package inventory

import (
	"context"
	"fmt"

	"git.konjactw.dev/falloutBot/go-mc/chat"
	"git.konjactw.dev/falloutBot/go-mc/level/item"
	"git.konjactw.dev/patyhank/minego/pkg/bot"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/server"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/slot"
)

// Manager 管理inventory和container
type Manager struct {
	c                  bot.Client
	inventory          *Container
	container          *Container
	cursor             *slot.Slot
	currentContainerID int32
}

func NewManager(c bot.Client) *Manager {
	m := &Manager{
		c:                  c,
		inventory:          NewContainerWithSize(c, 0, 45),
		currentContainerID: 0,
	}

	bot.AddHandler(c, func(ctx context.Context, p *client.WindowItems) {
		if p.WindowId == 0 {
			m.inventory.SetSlots(convertWindowItems(p.Items))
		} else if m.container != nil {
			m.container.SetSlots(convertWindowItems(p.Items))
		}
		m.c.Player().UpdateStateID(p.StateId)
	})
	bot.AddHandler(c, func(ctx context.Context, p *client.SetSlot) {
		if p.WindowId == 0 {
			m.inventory.SetSlot(int(p.Slot), convertSetSlotItem(p.Item))
		} else if m.container != nil {
			m.container.SetSlot(int(p.Slot), convertSetSlotItem(p.Item))
		}
		m.c.Player().UpdateStateID(p.StateId)
	})
	bot.AddHandler(c, func(ctx context.Context, p *client.CloseWindow) {
		if p.WindowId == m.currentContainerID {
			m.currentContainerID = -1
			if m.container != nil {
				m.container = nil
			}
		}
	})
	bot.AddHandler(c, func(ctx context.Context, p *client.OpenWindow) {
		m.currentContainerID = p.WindowId
		m.container = NewContainer(c, p.WindowId)
		go bot.PublishEvent(m.c, ContainerOpenEvent{
			WindowID: p.WindowId,
			Type:     p.InventoryType,
			Title:    chat.Message{Text: fmt.Sprintf("%v", p.WindowTitle)},
		})
	})

	return m
}

func (m *Manager) Inventory() bot.Container {
	return m.inventory
}

func (m *Manager) Container() bot.Container {
	return m.container
}
func (m *Manager) Cursor() *slot.Slot {
	return m.cursor
}

func (m *Manager) CurrentContainerID() int32 {
	return m.currentContainerID
}

func (m *Manager) Close() {
	if m.currentContainerID != -1 {
		_ = m.c.WritePacket(context.Background(), &server.CloseWindow{WindowId: m.currentContainerID})
		m.currentContainerID = -1
	} else {
		_ = m.c.WritePacket(context.Background(), &server.CloseWindow{WindowId: 0})
		m.currentContainerID = -1
	}
}

// Click 點擊容器slot
func (m *Manager) Click(id int32, slotIndex int16, mode int32, button int32) error {
	clickPacket := &server.WindowClick{
		WindowId:    id,
		StateId:     m.c.Player().StateID(),
		Slot:        slotIndex,
		MouseButton: int8(button),
		Mode:        mode,
	}
	return m.c.WritePacket(context.Background(), clickPacket)
}

// convertWindowItems 轉換新協議的物品格式為 slot.Slot
func convertWindowItems(items []client.WindowItemsTemp) []slot.Slot {
	res := make([]slot.Slot, len(items))
	for i, it := range items {
		res[i] = slot.Slot{Count: it.ItemCount, ItemID: item.ID(it.ItemId)}
	}
	return res
}

// convertSetSlotItem 將 SetSlotItem 轉換為 slot.Slot
func convertSetSlotItem(it client.SetSlotItem) slot.Slot {
	return slot.Slot{Count: it.ItemCount, ItemID: item.ID(it.ItemId)}
}
