package world

import (
	"bytes"
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-gl/mathgl/mgl64"
	"golang.org/x/exp/constraints"

	"git.konjactw.dev/falloutBot/go-mc/level"
	"git.konjactw.dev/falloutBot/go-mc/level/block"
	"git.konjactw.dev/falloutBot/go-mc/nbt"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"

	"git.konjactw.dev/patyhank/minego/pkg/bot"
	"git.konjactw.dev/patyhank/minego/pkg/protocol"
	"git.konjactw.dev/patyhank/minego/pkg/protocol/metadata"
	cp "git.konjactw.dev/patyhank/minego/pkg/protocol/packet/game/client"

	"github.com/google/uuid"
)

type World struct {
	c bot.Client

	columns             map[level.ChunkPos]*level.Chunk
	entities            map[int32]*Entity
	parsedBlockEntities map[protocol.Position]level.BlockEntityData

	entityLock sync.Mutex
	chunkLock  sync.Mutex
	beLock     sync.Mutex
}

func NewWorld(c bot.Client) *World {
	w := &World{
		c:                   c,
		columns:             make(map[level.ChunkPos]*level.Chunk),
		entities:            make(map[int32]*Entity),
		parsedBlockEntities: make(map[protocol.Position]level.BlockEntityData),
	}

	bot.AddHandler(c, func(ctx context.Context, p *cp.MapChunk) {
		w.chunkLock.Lock()
		defer w.chunkLock.Unlock()
		w.beLock.Lock()
		defer w.beLock.Unlock()

		pos2d := level.ChunkPos{p.X, p.Z}
		chunk := level.EmptyChunk(24)
		_ = chunk.PutData(p.ChunkData)
		// 高度圖
		for _, hm := range p.Heightmaps {
			chunk.HeightMaps = append(chunk.HeightMaps, level.HeightMap{
				Type: int32(mapHeightmapType(hm.Type)),
				Data: toLongSlice(hm.Data),
			})
		}
		// 方塊實體
		for _, be := range p.BlockEntities {
			var raw map[string]interface{}
			buf := &bytes.Buffer{}
			_, _ = be.NbtData.WriteTo(buf)
			dataBytes := buf.Bytes()
			if err := nbt.Unmarshal(dataBytes, &raw); err == nil {
				var x, z int32
				var y int16
				switch v := raw["x"].(type) {
				case int32:
					x = v
				case int:
					x = int32(v)
				}
				y = be.Y
				switch v := raw["z"].(type) {
				case int32:
					z = v
				case int:
					z = int32(v)
				}
				beType := block.EntityType(be.Type)
				lb := level.BlockEntity{
					XZ:   int8(((x & 0xF) << 4) | (z & 0xF)),
					Y:    y,
					Type: beType,
					Data: nbt.RawMessage{Type: 0, Data: dataBytes},
				}
				chunk.BlockEntity = append(chunk.BlockEntity, lb)

				// 解析 NBT 資料
				if parsedData, err := parseBlockEntityData(beType, lb.Data); err == nil {
					pos := protocol.Position{int32(x), int32(y), int32(z)}
					w.parsedBlockEntities[pos] = parsedData
				}
			}
		}
		// 光照
		applyLightMask := func(mask []int64, lights [][]uint8, setter func(sec *level.Section, data []byte)) {
			lightIdx := 0
			maxSec := len(chunk.Sections)
			for bit := 0; bit < maxSec && lightIdx < len(lights); bit++ {
				if bitSet(mask, bit) {
					sec := &chunk.Sections[bit]
					data := make([]byte, len(lights[lightIdx]))
					copy(data, lights[lightIdx])
					setter(sec, data)
					lightIdx++
				}
			}
		}
		applyLightMask(p.SkyLightMask, p.SkyLight, func(sec *level.Section, data []byte) { sec.SkyLight = data })
		applyLightMask(p.BlockLightMask, p.BlockLight, func(sec *level.Section, data []byte) { sec.BlockLight = data })
		for bit := 0; bit < len(chunk.Sections); bit++ {
			if bitSet(p.EmptySkyLightMask, bit) {
				chunk.Sections[bit].SkyLight = nil
			}
			if bitSet(p.EmptyBlockLightMask, bit) {
				chunk.Sections[bit].BlockLight = nil
			}
		}

		w.columns[pos2d] = chunk
	})

	bot.AddHandler(c, func(ctx context.Context, p *cp.UnloadChunk) {
		w.chunkLock.Lock()
		defer w.chunkLock.Unlock()
		w.beLock.Lock()
		defer w.beLock.Unlock()

		pos2d := level.ChunkPos{p.ChunkX, p.ChunkZ}

		// 移除被卸載 chunk 中的 block entities
		if chunk, ok := w.columns[pos2d]; ok {
			for _, be := range chunk.BlockEntity {
				x, z := be.UnpackXZ()
				posX := p.ChunkX*16 + int32(x)
				posZ := p.ChunkZ*16 + int32(z)
				pos := protocol.Position{posX, int32(be.Y), posZ}
				delete(w.parsedBlockEntities, pos)
			}
		}

		delete(w.columns, pos2d)
	})
	bot.AddHandler(c, func(ctx context.Context, p *cp.Respawn) {
		w.chunkLock.Lock()
		defer w.chunkLock.Unlock()

		w.columns = make(map[level.ChunkPos]*level.Chunk)
		w.parsedBlockEntities = make(map[protocol.Position]level.BlockEntityData)
	})

	bot.AddHandler(c, func(ctx context.Context, p *cp.SpawnEntity) {
		w.entityLock.Lock()
		defer w.entityLock.Unlock()

		uid, _ := uuid.FromBytes(p.ObjectUUID[:])
		w.entities[p.EntityId] = NewEntity(
			p.EntityId,
			uid,
			int32(p.Type),
			mgl64.Vec3{p.X, p.Y, p.Z},
			mgl64.Vec2{pk.Angle(p.Pitch).ToDeg(), pk.Angle(p.Yaw).ToDeg()},
		)
	})
	bot.AddHandler(c, func(ctx context.Context, p *cp.EntityDestroy) {
		w.entityLock.Lock()
		defer w.entityLock.Unlock()
		for _, d := range p.EntityIds {
			e, ok := w.entities[d]
			if ok {
				bot.PublishEvent(c, EntityRemoveEvent{Entity: e})
				delete(w.entities, d)
			}
		}
	})
	bot.AddHandler(c, func(ctx context.Context, p *cp.EntityMetadata) {
		w.entityLock.Lock()
		defer w.entityLock.Unlock()
		e, ok := w.entities[p.EntityId]
		if ok {
			if e.metadata == nil {
				e.metadata = make(map[uint8]metadata.Metadata)
			}
			for u, entityMetadata := range p.Metadata.Data {
				e.metadata[u] = entityMetadata
			}
		}
	})
	// EntityEquipment 新版格式較複雜，此處暫不處理裝備更新避免解析錯誤。
	bot.AddHandler(c, func(ctx context.Context, p *cp.SyncEntityPosition) {
		w.entityLock.Lock()
		defer w.entityLock.Unlock()
		if e, ok := w.entities[p.EntityId]; ok {
			e.SetPosition(mgl64.Vec3{p.X, p.Y, p.Z})
		}
	})

	bot.AddHandler(c, func(ctx context.Context, p *cp.EntityLook) {
		w.entityLock.Lock()
		defer w.entityLock.Unlock()
		if e, ok := w.entities[p.EntityId]; ok {
			e.SetRotation(mgl64.Vec2{pk.Angle(p.Yaw).ToDeg(), pk.Angle(p.Pitch).ToDeg()})
		}
	})

	bot.AddHandler(c, func(ctx context.Context, p *cp.EntityMoveLook) {
		w.entityLock.Lock()
		defer w.entityLock.Unlock()
		if e, ok := w.entities[p.EntityId]; ok {
			currentPos := e.Position()
			newPos := currentPos.Add(mgl64.Vec3{float64(p.DX) / 4096.0, float64(p.DY) / 4096.0, float64(p.DZ) / 4096.0})
			e.SetPosition(newPos)
			e.SetRotation(mgl64.Vec2{pk.Angle(p.Yaw).ToDeg(), pk.Angle(p.Pitch).ToDeg()})
		}
	})

	// 單方塊更新
	bot.AddHandler(c, func(ctx context.Context, p *cp.BlockChange) {
		w.chunkLock.Lock()
		defer w.chunkLock.Unlock()

		chunkX, chunkZ, lx, ly, lz := decodePackedBlockPos(int64(p.Location))
		pos2d := level.ChunkPos{int32(chunkX), int32(chunkZ)}
		chunk, ok := w.columns[pos2d]
		if !ok {
			return
		}
		sectionY := sectionIndex(int32(ly >> 4))
		if sectionY < 0 || sectionY >= len(chunk.Sections) {
			return
		}
		section := chunk.Sections[sectionY]
		blockIdx := (int(ly&15) << 8) | (int(lz) << 4) | int(lx)
		section.SetBlock(blockIdx, level.BlocksState(p.Type))
	})

	// 多方塊更新 (每個 record 為 chunk 內 12bit 位置 + 狀態)
	bot.AddHandler(c, func(ctx context.Context, p *cp.MultiBlockChange) {
		w.chunkLock.Lock()
		defer w.chunkLock.Unlock()

		chunkX, chunkY, chunkZ := decodeChunkSectionPos(int64(p.ChunkCoordinates))
		pos2d := level.ChunkPos{chunkX, chunkZ}
		chunk, ok := w.columns[pos2d]
		if !ok {
			return
		}
		secIdx := sectionIndex(chunkY)
		if secIdx < 0 || secIdx >= len(chunk.Sections) {
			return
		}
		section := chunk.Sections[secIdx]
		for _, rec := range p.Records {
			stateID := rec >> 12
			lpos := rec & 0xFFF
			lx := (lpos >> 8) & 0xF
			lz := (lpos >> 4) & 0xF
			ly := lpos & 0xF
			blockIdx := (int(ly&15) << 8) | (int(lz) << 4) | int(lx)
			section.SetBlock(blockIdx, level.BlocksState(stateID))
		}
	})

	// 光照更新，僅保存數據，不計算
	bot.AddHandler(c, func(ctx context.Context, p *cp.UpdateLight) {
		w.chunkLock.Lock()
		defer w.chunkLock.Unlock()

		pos2d := level.ChunkPos{p.ChunkX, p.ChunkZ}
		chunk, ok := w.columns[pos2d]
		if !ok {
			return
		}

		applyLightMask := func(mask []int64, lights [][]uint8, setter func(sec *level.Section, data []byte)) {
			lightIdx := 0
			maxSec := len(chunk.Sections)
			for bit := 0; bit < maxSec && lightIdx < len(lights); bit++ {
				if bitSet(mask, bit) {
					sec := &chunk.Sections[bit]
					data := make([]byte, len(lights[lightIdx]))
					copy(data, lights[lightIdx])
					setter(sec, data)
					lightIdx++
				}
			}
		}

		applyLightMask(p.SkyLightMask, p.SkyLight, func(sec *level.Section, data []byte) {
			sec.SkyLight = make([]byte, len(data))
			copy(sec.SkyLight, data)
		})
		applyLightMask(p.BlockLightMask, p.BlockLight, func(sec *level.Section, data []byte) {
			sec.BlockLight = make([]byte, len(data))
			copy(sec.BlockLight, data)
		})

		// 將空光照標記的 section 清空，以符合協議
		for bit := 0; bit < len(chunk.Sections); bit++ {
			if bitSet(p.EmptySkyLightMask, bit) {
				chunk.Sections[bit].SkyLight = nil
			}
			if bitSet(p.EmptyBlockLightMask, bit) {
				chunk.Sections[bit].BlockLight = nil
			}
		}
	})

	return w
}

func (w *World) GetBlockEntity(pos protocol.Position) (level.BlockEntityData, bool) {
	w.beLock.Lock()
	defer w.beLock.Unlock()
	be, ok := w.parsedBlockEntities[pos]
	return be, ok
}

func parseBlockEntityData(entityType block.EntityType, data nbt.RawMessage) (level.BlockEntityData, error) {
	var v level.BlockEntityData
	var err error

	dataBytes := data.Data

	if int(entityType) >= len(block.EntityList) {
		return nil, fmt.Errorf("unknown block entity type: %d", entityType)
	}
	id := block.EntityList[entityType].ID()

	switch id {
	case "minecraft:sign", "minecraft:hanging_sign":
		var sign level.SignEntity
		err = nbt.Unmarshal(dataBytes, &sign)
		v = sign
	case "minecraft:chest", "minecraft:trapped_chest":
		var chest level.ChestEntity
		err = nbt.Unmarshal(dataBytes, &chest)
		v = chest
	default:
		var base level.BaseBlockEntity
		err = nbt.Unmarshal(dataBytes, &base)
		v = base
	}

	if err != nil {
		return nil, fmt.Errorf("nbt unmarshal error for %s: %w", id, err)
	}
	return v, nil
}

func (w *World) GetBlock(pos protocol.Position) (block.Block, error) {
	w.chunkLock.Lock()
	defer w.chunkLock.Unlock()
	chunkX := pos[0] >> 4
	chunkZ := pos[2] >> 4
	pos2d := level.ChunkPos{chunkX, chunkZ}

	chunk, ok := w.columns[pos2d]
	if !ok {
		return nil, errors.New("chunk not loaded")
	}

	blockX := pos[0] & 15
	blockZ := pos[2] & 15
	blockY := pos[1] & 15
	blockIdx := (blockY << 8) | (blockZ << 4) | blockX
	sectionY := pos[1] >> 4
	if sectionY < 0 || int(sectionY) >= len(chunk.Sections) {
		return nil, errors.New("invalid section Y coordinate")
	}
	blockStateId := chunk.Sections[sectionY].GetBlock(int(blockIdx))
	return block.StateList[blockStateId], nil
}

func (w *World) SetBlock(pos protocol.Position, blk block.Block) error {
	w.chunkLock.Lock()
	defer w.chunkLock.Unlock()

	chunkX := pos[0] >> 4
	chunkZ := pos[2] >> 4
	pos2d := level.ChunkPos{chunkX, chunkZ}

	chunk, ok := w.columns[pos2d]
	if !ok {
		return errors.New("chunk not loaded")
	}

	blockX := pos[0] & 15
	blockZ := pos[2] & 15
	sectionY := pos[1] >> 4
	blockY := pos[1] & 15

	if sectionY < 0 || int(sectionY) >= len(chunk.Sections) {
		return errors.New("invalid section Y coordinate")
	}

	section := chunk.Sections[sectionY]

	blockIdx := (blockY << 8) | (blockZ << 4) | blockX
	section.SetBlock(int(blockIdx), block.ToStateID[blk])
	return nil
}

func (w *World) GetNearbyBlocks(pos protocol.Position, radius int32) ([]block.Block, error) {
	var blocks []block.Block

	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			for dz := -radius; dz <= radius; dz++ {
				blk, err := w.GetBlock(protocol.Position{pos[0] + dx, pos[1] + dy, pos[2] + dz})
				if err != nil {
					continue
				}
				blocks = append(blocks, blk)
			}
		}
	}

	return blocks, nil
}

func (w *World) FindNearbyBlock(pos protocol.Position, radius int32, blk block.Block) (protocol.Position, error) {
	visited := make(map[protocol.Position]bool)
	queue := list.New()
	start := pos
	queue.PushBack(start)
	visited[start] = true

	// Direction vectors for 6-way adjacent blocks
	dirs := []protocol.Position{
		{1, 0, 0}, {-1, 0, 0},
		{0, 1, 0}, {0, -1, 0},
		{0, 0, 1}, {0, 0, -1},
	}
	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(protocol.Position)

		// Skip if beyond the radius
		if abs(current[0]-pos[0]) > radius || abs(current[1]-pos[1]) > radius || abs(current[2]-pos[2]) > radius {
			continue
		}

		// Check if current block matches target
		if currentBlock, err := w.GetBlock(current); err == nil {
			if currentBlock == blk {
				return current, nil
			}
		}

		// Check all 6 adjacent blocks
		for _, dir := range dirs {
			next := protocol.Position{
				current[0] + dir[0],
				current[1] + dir[1],
				current[2] + dir[2],
			}

			if !visited[next] {
				visited[next] = true
				queue.PushBack(next)
			}
		}
	}

	return protocol.Position{}, errors.New("block not found")
}

func (w *World) Entities() []bot.Entity {
	w.entityLock.Lock()
	defer w.entityLock.Unlock()
	var entities []bot.Entity
	for _, e := range w.entities {
		entities = append(entities, e)
	}
	return entities
}

func (w *World) GetEntity(id int32) bot.Entity {
	w.entityLock.Lock()
	defer w.entityLock.Unlock()
	return w.entities[id]
}

func (w *World) GetNearbyEntities(radius int32) []bot.Entity {
	w.entityLock.Lock()
	defer w.entityLock.Unlock()

	selfPos := w.c.Player().Entity().Position()
	var entities []bot.Entity

	for _, e := range w.entities {
		sqr := e.Position().Sub(selfPos).LenSqr()
		if sqr <= float64(radius*radius) {
			entities = append(entities, e)
		}
	}
	return entities
}

func (w *World) GetEntitiesByType(entityType int32) []bot.Entity {
	w.entityLock.Lock()
	defer w.entityLock.Unlock()

	var entities []bot.Entity
	for _, e := range w.entities {
		if e.Type() == entityType {
			entities = append(entities, e)
		}
	}
	return entities
}

// mapHeightmapType 對應 minecraft-data 的 heightmap 名稱
func mapHeightmapType(t string) int {
	switch t {
	case "world_surface_wg":
		return 0
	case "world_surface":
		return 1
	case "ocean_floor_wg":
		return 2
	case "ocean_floor":
		return 3
	case "motion_blocking":
		return 4
	case "motion_blocking_no_leaves":
		return 5
	default:
		return -1
	}
}

func toLongSlice(in []int64) []pk.Long {
	out := make([]pk.Long, len(in))
	for i, v := range in {
		out[i] = pk.Long(v)
	}
	return out
}

func abs[T constraints.Signed | constraints.Float](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
