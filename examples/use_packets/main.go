package main

import (
	"bytes"
	"fmt"
	"log"

	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	"git.konjactw.dev/falloutBot/go-mc/pkg/protocol/packet/game/client"
)

func main() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🎮 Nalago-MC 封包使用示例")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 示例 1: 简单封包 - KeepAlive
	exampleKeepAlive()

	// 示例 2: 复杂封包 - Position
	examplePosition()

	// 示例 3: 嵌套结构体 - Advancements
	exampleAdvancements()

	fmt.Println()
	fmt.Println("✅ 所有示例运行完成！")
}

// 示例 1: 简单封包
func exampleKeepAlive() {
	fmt.Println("📦 示例 1: KeepAlive 封包")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 创建封包
	original := &client.KeepAlive{
		KeepAliveId: 12345,
	}
	fmt.Printf("原始数据: ID = %d\n", original.KeepAliveId)

	// 序列化
	var buf bytes.Buffer
	n, err := original.WriteTo(&buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("序列化: 写入 %d 字节\n", n)
	fmt.Printf("字节: %v\n", buf.Bytes())

	// 反序列化
	decoded := &client.KeepAlive{}
	n, err = decoded.ReadFrom(&buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("反序列化: 读取 %d 字节\n", n)
	fmt.Printf("解码数据: ID = %d\n", decoded.KeepAliveId)

	// 验证
	if decoded.KeepAliveId == original.KeepAliveId {
		fmt.Println("✅ 序列化/反序列化成功！")
	} else {
		fmt.Println("❌ 数据不匹配！")
	}
	fmt.Println()
}

// 示例 2: 包含多个字段的封包
func examplePosition() {
	fmt.Println("📦 示例 2: Position 封包")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 创建封包
	original := &client.Position{
		X:          100.5,
		Y:          64.0,
		Z:          -50.25,
		Yaw:        180.0,
		Pitch:      45.0,
		Flags:      0,
		TeleportId: 1,
		// 其他字段...
	}
	fmt.Printf("原始位置: (%.2f, %.2f, %.2f)\n", original.X, original.Y, original.Z)
	fmt.Printf("视角: Yaw=%.2f, Pitch=%.2f\n", original.Yaw, original.Pitch)

	// 序列化
	var buf bytes.Buffer
	n, err := original.WriteTo(&buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("序列化: 写入 %d 字节\n", n)

	// 反序列化
	decoded := &client.Position{}
	n, err = decoded.ReadFrom(&buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("反序列化: 读取 %d 字节\n", n)
	fmt.Printf("解码位置: (%.2f, %.2f, %.2f)\n", decoded.X, decoded.Y, decoded.Z)

	fmt.Println("✅ 复杂封包序列化成功！")
	fmt.Println()
}

// 示例 3: 带嵌套结构体的封包
func exampleAdvancements() {
	fmt.Println("📦 示例 3: Advancements 封包（嵌套结构体）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 创建带嵌套结构的封包
	title := pk.NBT{Type: 8, Data: "成就标题"}
	desc := pk.NBT{Type: 8, Data: "成就描述"}

	displayData := &client.AdvancementsAdvancementMappingEntryValueDisplayData{
		Title:       title,
		Description: desc,
		Icon:        pk.Slot{Present: false},
		FrameType:   0,
		Flags:       0,
		XCord:       0,
		YCord:       0,
	}

	value := client.AdvancementsAdvancementMappingEntryValue{
		ParentId:          nil,
		DisplayData:       displayData,
		Requirements:      [][]string{{"requirement1"}},
		SendsTelemtryData: false,
	}

	entry := client.AdvancementsAdvancementMappingEntry{
		Key:   "minecraft:story/root",
		Value: value,
	}

	original := &client.Advancements{
		Reset:              true,
		AdvancementMapping: []client.AdvancementsAdvancementMappingEntry{entry},
		Identifiers:        []string{"minecraft:story/root"},
		ProgressMapping:    []client.AdvancementsProgressMappingEntry{},
		ShowAdvancements:   true,
	}

	fmt.Printf("成就数量: %d\n", len(original.AdvancementMapping))
	fmt.Printf("成就 Key: %s\n", original.AdvancementMapping[0].Key)
	fmt.Println("包含嵌套的 DisplayData 结构体 ✓")

	// 序列化
	var buf bytes.Buffer
	n, err := original.WriteTo(&buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("序列化: 写入 %d 字节\n", n)

	// 反序列化
	decoded := &client.Advancements{}
	n, err = decoded.ReadFrom(&buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("反序列化: 读取 %d 字节\n", n)
	fmt.Printf("解码成就数量: %d\n", len(decoded.AdvancementMapping))

	if len(decoded.AdvancementMapping) > 0 {
		fmt.Printf("解码 Key: %s\n", decoded.AdvancementMapping[0].Key)
	}

	fmt.Println("✅ 嵌套结构体序列化成功！")
	fmt.Println()
}
