package world

// decodeChunkXZ 由 packed chunkCoordinates 解析 chunkX, chunkZ。
// 依照 wiki.vg MultiBlockChange: chunkXZ 長度 64bit，前 22bit 為 chunkX，後 22bit 為 chunkZ。
func decodeChunkXZ(packed int64) (int64, int64) {
	chunkX := packed >> 42
	chunkZ := packed << 22 >> 42
	return chunkX, chunkZ
}

// decodePackedBlockPos 將 BlockChange 的 packed location 解析為 chunkX, chunkZ, localX, localY, localZ。
// 使用與 legacy 64-bit BlockPos 相同的布局。
func decodePackedBlockPos(pos int64) (chunkX, chunkZ, lx, ly, lz int64) {
	x := pos >> 38
	y := (pos << 52) >> 52
	z := (pos << 26) >> 38
	chunkX = x >> 4
	chunkZ = z >> 4
	lx = x & 15
	ly = y & 255
	lz = z & 15
	return
}

// bitSet 檢查 bitset 中某位是否為 1。
func bitSet(mask []int64, idx int) bool {
	if idx < 0 {
		return false
	}
	word := idx / 64
	bit := uint(idx % 64)
	if word >= len(mask) {
		return false
	}
	return (mask[word] & (1 << bit)) != 0
}

// chunkSectionPos long: 21 bits each for x,y,z (two's complement)
func decodeChunkSectionPos(v int64) (x, y, z int32) {
	x = int32(v >> 42)
	y = int32((v << 22) >> 43) // shift to keep sign
	z = int32((v << 43) >> 43)
	return
}

// sectionIndex 把 chunk section Y 轉為 slice index，假設 chunk 覆蓋 -64..319 (24 sections)
func sectionIndex(chunkY int32) int {
	idx := int(chunkY + 4) // -64/16 = -4
	if idx < 0 || idx >= 24 {
		return -1
	}
	return idx
}
