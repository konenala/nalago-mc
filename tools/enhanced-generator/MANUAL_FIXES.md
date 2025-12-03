# 手动修复指南

本文档说明如何处理生成器无法自动处理的复杂类型。

## 🎯 需要手动处理的类型

### 1. Switch 类型 (最常见)

**问题描述：**
Switch 类型根据另一个字段的值决定读写什么数据，类似于 union/variant。

**示例：**
```go
// TODO: Switch type - conditional field based on other field value
BackgroundTexture interface{}
```

**JSON 定义：**
```json
{
  "name": "backgroundTexture",
  "type": [
    "switch",
    {
      "compareTo": "flags/has_background_texture",
      "fields": {
        "1": "string"
      },
      "default": "void"
    }
  ]
}
```

**修复方法：**

#### 方法 1：使用条件读写（推荐）

```go
// 结构体定义
type DisplayData struct {
    Flags             int32
    BackgroundTexture *string  // 改为可选指针
}

// ReadFrom 方法
func (s *DisplayData) ReadFrom(r io.Reader) (n int64, err error) {
    var temp int64

    // 先读取 Flags
    temp, err = (*pk.Int)(&s.Flags).ReadFrom(r)
    n += temp
    if err != nil { return n, err }

    // 根据 flag 决定是否读取 BackgroundTexture
    if s.Flags & 0x01 != 0 {  // has_background_texture bit
        var val string
        var elem pk.String
        temp, err = elem.ReadFrom(r)
        n += temp
        if err != nil { return n, err }
        val = string(elem)
        s.BackgroundTexture = &val
    }

    return n, nil
}

// WriteTo 方法
func (s DisplayData) WriteTo(w io.Writer) (n int64, err error) {
    var temp int64

    // 先写入 Flags
    temp, err = pk.Int(s.Flags).WriteTo(w)
    n += temp
    if err != nil { return n, err }

    // 根据 flag 决定是否写入 BackgroundTexture
    if s.BackgroundTexture != nil {
        temp, err = pk.String(*s.BackgroundTexture).WriteTo(w)
        n += temp
        if err != nil { return n, err }
    }

    return n, nil
}
```

#### 方法 2：使用原始字节（简单但不类型安全）

```go
type DisplayData struct {
    Flags int32
    BackgroundTextureRaw []byte  // 存储原始数据
}

func (s *DisplayData) ReadFrom(r io.Reader) (n int64, err error) {
    // ... 读取 Flags ...

    // 根据条件读取剩余数据
    if s.Flags & 0x01 != 0 {
        temp, err = (*pk.String)(&s.BackgroundTextureRaw).ReadFrom(r)
        n += temp
        if err != nil { return n, err }
    }

    return n, nil
}

// 提供辅助方法解析
func (s *DisplayData) GetBackgroundTexture() string {
    if len(s.BackgroundTextureRaw) == 0 {
        return ""
    }
    return string(s.BackgroundTextureRaw)
}
```

### 2. Mapper 类型

**问题描述：**
Mapper 类型是键值对的映射，通常是 Map[string]X。

**示例：**
```go
// TODO: Implement mapper type
Data interface{}
```

**修复方法：**
```go
type Packet struct {
    Data map[string]string  // 或其他具体类型
}

func (p *Packet) ReadFrom(r io.Reader) (n int64, err error) {
    // 读取 map 长度
    var count pk.VarInt
    temp, err = count.ReadFrom(r)
    n += temp
    if err != nil { return n, err }

    p.Data = make(map[string]string, count)

    for i := 0; i < int(count); i++ {
        // 读取 key
        var key pk.String
        temp, err = key.ReadFrom(r)
        n += temp
        if err != nil { return n, err }

        // 读取 value
        var value pk.String
        temp, err = value.ReadFrom(r)
        n += temp
        if err != nil { return n, err }

        p.Data[string(key)] = string(value)
    }

    return n, nil
}
```

### 3. 特殊类型

#### ContainerID
```go
// 生成器已映射为 int8，但如果遇到 TODO：
WindowId int8  // 0-127 的容器 ID
```

#### RecipeDisplay
```go
// 复杂的显示信息，建议使用 json.RawMessage
type RecipeDisplay struct {
    Raw json.RawMessage
}

func (r *RecipeDisplay) ReadFrom(reader io.Reader) (n int64, err error) {
    // 读取 NBT 或其他格式
    var nbt pk.NBT
    temp, err := nbt.ReadFrom(reader)
    n += temp
    if err != nil { return n, err }

    r.Raw, _ = json.Marshal(nbt)
    return n, nil
}
```

## 🛠️ 通用修复步骤

### 步骤 1：找到 TODO
```bash
cd pkg/protocol/packet/game/client
grep -rn "// TODO" .
```

### 步骤 2：查看 protocol.json
在 `protocol.json` 中找到对应字段的定义，了解其真实结构。

### 步骤 3：选择修复方法
- **简单条件** → 使用条件读写（方法 1）
- **复杂逻辑** → 使用原始字节 + 辅助函数（方法 2）
- **不常用的封包** → 可以暂时留 `interface{}`，用到时再实现

### 步骤 4：测试
```go
// 创建测试验证序列化/反序列化
func TestPacket(t *testing.T) {
    original := &MyPacket{
        Field: "test",
    }

    // 序列化
    var buf bytes.Buffer
    _, err := original.WriteTo(&buf)
    if err != nil {
        t.Fatal(err)
    }

    // 反序列化
    decoded := &MyPacket{}
    _, err = decoded.ReadFrom(&buf)
    if err != nil {
        t.Fatal(err)
    }

    // 验证
    if decoded.Field != original.Field {
        t.Errorf("expected %v, got %v", original.Field, decoded.Field)
    }
}
```

## 📊 优先级建议

根据使用频率，建议按以下顺序修复：

### 高优先级（常用封包）
1. **KeepAlive** ✅ 已完成
2. **Position** ✅ 已完成
3. **Chat** - 检查是否有 switch 字段
4. **EntityMetadata** - 检查是否有 switch 字段
5. **ChunkData** - 可能有复杂字段

### 中优先级（偶尔使用）
- Commands（DeclareCommands）- 有 command_node
- Recipes（DeclareRecipes）- 有 RecipeDisplay
- Advancements - 有部分 switch

### 低优先级（很少使用）
- Statistics
- Debug packets
- 不常见的实体效果

## 💡 最佳实践

1. **不要过度完善** - 只在真正需要时才修复
2. **先测试** - 修复后立即编写单元测试
3. **参考现有代码** - 看 `manual_recv.go` 里的手动实现
4. **保持简单** - 优先使用可选指针而不是复杂的 union
5. **添加注释** - 说明字段的条件和用途

## 🔗 相关资源

- [Minecraft Protocol Wiki](https://wiki.vg/Protocol)
- [minecraft-data 文档](https://github.com/PrismarineJS/minecraft-data)
- [go-mc 示例](https://github.com/Tnze/go-mc/tree/master/net/packet)

## ❓ 常见问题

**Q: 所有 TODO 都必须修复吗？**
A: 不是！大部分封包已经完全可用。只在你的应用真正用到某个封包时再修复。

**Q: Switch 类型太复杂，有简化方法吗？**
A: 可以暂时用 `json.RawMessage` 存储原始数据，需要时再解析。

**Q: 如何知道某个字段的条件逻辑？**
A: 查看 `protocol.json` 的 `compareTo` 字段，或参考 [wiki.vg](https://wiki.vg/Protocol)。

**Q: 修复后如何验证？**
A: 编写单元测试，或连接真实 Minecraft 服务器测试。
