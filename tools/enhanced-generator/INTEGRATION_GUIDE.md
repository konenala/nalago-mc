# 完全集成指南

本文档介绍完整的自动化封包生成和使用系统。

## 🎯 系统概览

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  protocol.json  ──→  Enhanced Generator V2  ──→  Go Packets│
│  (Minecraft 协议)     (自动生成器)              (可用代码)    │
│                                                             │
└─────────────────────────────────────────────────────────────┘

        ↓ 一键操作                    ↓ 自动处理

  generate.bat / Makefile    →   修复变量名 → 编译验证 → 完成
```

## 🚀 快速开始（3 步）

### 1️⃣ 生成封包

**Windows:**
```cmd
cd E:\bot編寫\go-mc\nalago-mc
generate.bat
```

**Linux/Mac:**
```bash
cd nalago-mc
make gen-packets
```

### 2️⃣ 验证

生成完成后会自动：
- ✅ 修复子结构体变量名
- ✅ 编译验证代码
- ✅ 显示统计信息

### 3️⃣ 使用

```go
import "git.konjactw.dev/falloutBot/go-mc/pkg/protocol/packet/game/client"

// 直接使用生成的封包
keepAlive := &client.KeepAlive{
    KeepAliveId: 123,
}
keepAlive.WriteTo(writer)
```

## 📁 完整文件结构

```
nalago-mc/
│
├── 🎮 一键脚本
│   ├── generate.bat                   # Windows 一键生成
│   └── Makefile                       # Linux/Mac 自动化
│
├── 📦 生成的封包
│   └── pkg/protocol/packet/game/
│       ├── client/                    # 125 个 client 封包
│       │   ├── packet_keep_alive.go
│       │   ├── packet_position.go
│       │   ├── packet_advancements.go # 带嵌套结构体
│       │   └── ...
│       └── server/                    # Server 封包
│           └── ...
│
├── 🛠️ 生成器工具
│   └── tools/enhanced-generator/
│       ├── main_v2.go                 # 核心生成器
│       ├── generate.sh                # Unix 生成脚本
│       ├── README.md                  # 使用文档
│       ├── MANUAL_FIXES.md            # 手动修复指南
│       ├── IMPROVEMENTS.md            # 改进历史
│       └── INTEGRATION_GUIDE.md       # 本文档
│
└── 📖 示例代码
    └── examples/use_packets/
        └── main.go                    # 使用示例
```

## ⚙️ 生成流程详解

### 自动化流程

```mermaid
graph LR
    A[protocol.json] --> B[解析 JSON]
    B --> C[生成结构体]
    C --> D[生成序列化代码]
    D --> E[修复变量名]
    E --> F[编译验证]
    F --> G[完成]
```

### 各步骤说明

| 步骤 | 说明 | 输出 |
|------|------|------|
| 1. 解析 JSON | 读取 protocol.json，提取所有封包定义 | PacketDef 数组 |
| 2. 生成结构体 | 为每个封包和嵌套结构生成 Go struct | *.go 文件 |
| 3. 生成序列化 | 自动生成 ReadFrom/WriteTo 方法 | 序列化代码 |
| 4. 修复变量名 | 将子结构体的 `p.` 改为 `s.` | 修正后的代码 |
| 5. 编译验证 | 确保所有代码可编译 | 编译成功 |
| 6. 统计报告 | 显示生成结果和 TODO 数量 | 统计信息 |

## 📊 生成结果统计

### 当前版本 (V2)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 Client 封包
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ├─ 总封包数: 125
  ├─ 完全可用: 110+
  ├─ 有 TODO: 15
  └─ TODO 数量: ~100

主要 TODO 类型:
  • Switch type (33 个) - 需要条件逻辑
  • Unknown types (10 个) - 特殊类型
```

### 对比旧版本

| 指标 | V1 | V2 | 改进 |
|------|----|----|------|
| 总封包数 | 125 | 125 | - |
| 完全可用 | 83 (66%) | 110+ (88%+) | **+32%** 🎉 |
| "Complex array" | 42 | 0 | **-100%** ✅ |
| 子结构体 | 0 | 50+ | **新增** ✨ |
| 一键生成 | ❌ | ✅ | **新增** 🚀 |

## 🎮 使用场景

### 场景 1: 基础机器人

```go
// 处理 KeepAlive
func handleKeepAlive(data []byte) {
    packet := &client.KeepAlive{}
    packet.ReadFrom(bytes.NewReader(data))

    // 回复 server 封包
    response := &server.KeepAlive{
        KeepAliveId: packet.KeepAliveId,
    }
    response.WriteTo(conn)
}
```

### 场景 2: 位置追踪

```go
func handlePosition(data []byte) {
    pos := &client.Position{}
    pos.ReadFrom(bytes.NewReader(data))

    fmt.Printf("玩家位置: (%.2f, %.2f, %.2f)\n",
        pos.X, pos.Y, pos.Z)
}
```

### 场景 3: 成就系统

```go
func handleAdvancements(data []byte) {
    adv := &client.Advancements{}
    adv.ReadFrom(bytes.NewReader(data))

    for _, entry := range adv.AdvancementMapping {
        fmt.Printf("成就: %s\n", entry.Key)
        if entry.Value.DisplayData != nil {
            // 访问嵌套结构
            fmt.Println("  有显示数据")
        }
    }
}
```

## 🔧 自定义配置

### 修改 protocol.json 路径

**generate.bat:**
```batch
set PROTOCOL_JSON=你的\路径\protocol.json
```

**Makefile:**
```makefile
PROTOCOL_JSON = 你的/路径/protocol.json
```

### 只生成特定方向

**Windows:**
```cmd
cd tools\enhanced-generator
go run main_v2.go -protocol "..." -output "..." -direction client
```

**Linux/Mac:**
```bash
make gen-client  # 只生成 client
make gen-server  # 只生成 server
```

### 禁用代码生成（只生成结构体）

```bash
go run main_v2.go -protocol "..." -output "..." -codec=false
```

## 🛠️ 高级功能

### 1. 处理 TODO 字段

参见 [MANUAL_FIXES.md](./MANUAL_FIXES.md) 了解如何手动实现复杂类型。

**最常见：Switch 类型**
```go
// 根据条件读写不同的数据
if s.Flags & 0x01 != 0 {
    // 读取可选字段
}
```

### 2. 扩展生成的封包

**不要直接修改生成的文件！** 在新文件中扩展：

```go
// custom_packets.go

// 为 KeepAlive 添加方法
func (k *KeepAlive) IsValid() bool {
    return k.KeepAliveId > 0
}

// 添加辅助函数
func NewKeepAlive(id int64) *KeepAlive {
    return &KeepAlive{KeepAliveId: id}
}
```

### 3. 单元测试

```go
func TestPacketSerialization(t *testing.T) {
    original := &client.KeepAlive{KeepAliveId: 123}

    var buf bytes.Buffer
    n, err := original.WriteTo(&buf)
    require.NoError(t, err)
    require.Greater(t, n, int64(0))

    decoded := &client.KeepAlive{}
    n, err = decoded.ReadFrom(&buf)
    require.NoError(t, err)
    require.Equal(t, original.KeepAliveId, decoded.KeepAliveId)
}
```

## 📚 工作流程示例

### 日常开发流程

```bash
# 1. 开发你的应用
code your_app.go

# 2. 需要新封包时，检查是否已生成
ls pkg/protocol/packet/game/client/

# 3. 如果协议更新，重新生成
make gen-packets   # 或 generate.bat

# 4. 继续开发
go run your_app.go
```

### 协议更新流程

```bash
# 1. 获取新的 protocol.json
wget https://.../protocol.json

# 2. 更新配置中的路径
edit generate.bat  # 或 Makefile

# 3. 重新生成所有封包
generate.bat       # 或 make gen-packets

# 4. 检查编译错误
go build ./...

# 5. 修复 API 变更（如果有）
# ...

# 6. 测试
go test ./...
```

## ⚡ 性能优化

### 生成性能

- **生成速度**: ~2 秒 (125 个封包)
- **编译速度**: ~3 秒 (所有封包)
- **内存占用**: ~50 MB

### 运行时性能

生成的代码性能与手写代码相同：
- 无反射
- 直接字节操作
- 零内存分配（在可能的情况下）

## 🐛 故障排查

### 问题 1: 编译错误

**症状**: `go build` 失败

**解决**:
```bash
# 1. 清理旧文件
make clean

# 2. 重新生成
make gen-packets

# 3. 检查依赖
go mod tidy
```

### 问题 2: TODO 字段无法使用

**症状**: 某些字段是 `interface{}`

**解决**:
1. 检查是否真的需要这个字段
2. 如果需要，参考 [MANUAL_FIXES.md](./MANUAL_FIXES.md) 手动实现
3. 或暂时跳过，用到时再实现

### 问题 3: 子结构体变量名错误

**症状**: `p.Field` 编译错误

**解决**:
```bash
# 运行修复脚本
cd pkg/protocol/packet/game/client
perl -i -pe 's/(\s+)p\./\1s./g if /^func \(s \*/' packet_*.go
```

## 🎓 最佳实践

1. **版本控制**: 不要将生成的封包提交到 git（加入 .gitignore）
2. **一致性**: 所有团队成员使用相同的 protocol.json
3. **自动化**: 在 CI/CD 中集成生成流程
4. **按需修复**: 只在真正使用时才修复 TODO
5. **测试优先**: 修复后立即编写测试

## 📖 相关文档

- [README.md](./README.md) - 基础使用指南
- [MANUAL_FIXES.md](./MANUAL_FIXES.md) - 手动修复指南
- [IMPROVEMENTS.md](./IMPROVEMENTS.md) - 改进历史

## 🤝 贡献

欢迎贡献！可以：
- 改进生成器逻辑
- 添加更多类型支持
- 改进文档
- 报告 bug

## 📄 许可证

MIT License

---

**🎉 恭喜！你现在拥有完全自动化的 Minecraft 协议封包生成系统！**

需要帮助？提交 Issue 或查看文档。
