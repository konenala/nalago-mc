package client

import (
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

// 步骤 1: 手动定义结构体（根据 protocol.json 的定义）
// 步骤 2: 加上 // codec:gen 注释
// 步骤 3: 运行 packetizer 自动生成 ReadFrom/WriteTo

// AdvancementEntry 表示一个成就项
// codec:gen
type AdvancementEntry struct {
	Key   string `mc:"String"`
	Value AdvancementData
}

// AdvancementData 成就数据
// codec:gen
type AdvancementData struct {
	ParentId    *string      `mc:"Option"` // 可选的父成就 ID
	DisplayData *DisplayData `mc:"Option"` // 可选的显示数据
	// ... 其他字段根据需要添加
}

// DisplayData 显示数据
// codec:gen
type DisplayData struct {
	Title       pk.NBT  `mc:"NBT"`
	Description pk.NBT  `mc:"NBT"`
	Icon        pk.Slot `mc:"Slot"`
	// ... 其他字段
}

// ProgressEntry 进度项
// codec:gen
type ProgressEntry struct {
	Key   string `mc:"String"`
	Value ProgressData
}

// ProgressData 进度数据
// codec:gen
type ProgressData struct {
	// 根据实际需要定义
	Criteria []CriterionProgress `mc:"Array"`
}

// CriterionProgress 标准进度
// codec:gen
type CriterionProgress struct {
	Identifier string `mc:"String"`
	Achieved   *int64 `mc:"Option"` // 可选的时间戳
}

// 然后修改 Advancements 使用这些类型
// （把自动生成的替换掉）
