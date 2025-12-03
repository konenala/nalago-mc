#!/bin/bash
# 完整的封包生成脚本 - 一键生成所有封包

set -e  # 遇到错误立即退出

# 配置
PROTOCOL_JSON="E:/bot編寫/go-mc/minecraft-data-pc-1_21_10/data/pc/1.21.10/protocol.json"
OUTPUT_BASE="../../pkg/protocol/packet/game"
GENERATOR="main_v2.go"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Minecraft Protocol Packet Generator"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 检查依赖
if [ ! -f "$PROTOCOL_JSON" ]; then
    echo "❌ 找不到 protocol.json: $PROTOCOL_JSON"
    exit 1
fi

if [ ! -f "$GENERATOR" ]; then
    echo "❌ 找不到生成器: $GENERATOR"
    exit 1
fi

echo "📋 配置信息:"
echo "  Protocol: $PROTOCOL_JSON"
echo "  Generator: $GENERATOR"
echo "  Output: $OUTPUT_BASE"
echo ""

# 生成 Client 封包
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📦 生成 Client 封包 (服务器 → 客户端)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
go run "$GENERATOR" \
    -protocol "$PROTOCOL_JSON" \
    -output "$OUTPUT_BASE/client" \
    -direction client \
    -codec=true \
    -v

echo ""

# 生成 Server 封包
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📦 生成 Server 封包 (客户端 → 服务器)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
go run "$GENERATOR" \
    -protocol "$PROTOCOL_JSON" \
    -output "$OUTPUT_BASE/server" \
    -direction server \
    -codec=true \
    -v

echo ""

# 修复变量名
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔧 修复子结构体变量名"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

fix_vars() {
    local dir=$1
    echo "  修复 $dir ..."

    for file in "$dir"/packet_*.go; do
        if [ -f "$file" ]; then
            # 修复：在子结构体方法中，p. -> s.
            perl -i -pe '
                if (/^func \(s \*.*\) ReadFrom/ .. /^}/) {
                    s/(\s+)(temp, err = \(\*pk\.\w+\)\(&)p\./\1\2s./g;
                    s/(\s+)p\.(\w+) = /\1s.\2 = /g;
                    s/(\s+)(temp, err = )p\./\1\2s./g;
                    s/(\s+var \w+ pk\.\w+\s+temp, err = \w+\.ReadFrom\(r\)\s+.*\s+)p\.(\w+) =/\1s.\2 =/g;
                }
                if (/^func \(s .*\) WriteTo/ .. /^}/) {
                    s/(\s+)(temp, err = pk\.\w+\()p\./\1\2s./g;
                    s/(\s+)(temp, err = )p\./\1\2s./g;
                }
            ' "$file"
        fi
    done
}

fix_vars "$OUTPUT_BASE/client"
fix_vars "$OUTPUT_BASE/server"

echo "✅ 修复完成"
echo ""

# 统计生成结果
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 生成统计"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

count_stats() {
    local dir=$1
    local name=$2

    local total=$(find "$dir" -name "packet_*.go" | wc -l)
    local with_todo=$(grep -l "// TODO" "$dir"/packet_*.go 2>/dev/null | wc -l)
    local todo_count=$(grep -c "// TODO" "$dir"/packet_*.go 2>/dev/null | awk '{s+=$1} END {print s}')

    echo ""
    echo "📦 $name:"
    echo "  ├─ 总封包数: $total"
    echo "  ├─ 完全可用: $((total - with_todo))"
    echo "  ├─ 有 TODO: $with_todo"
    echo "  └─ TODO 数量: $todo_count"
}

count_stats "$OUTPUT_BASE/client" "Client 封包"
count_stats "$OUTPUT_BASE/server" "Server 封包"

echo ""

# 编译验证
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 编译验证"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

cd "$OUTPUT_BASE/client"
if go build .; then
    echo "✅ Client 封包编译成功"
else
    echo "❌ Client 封包编译失败"
    exit 1
fi

cd ../server
if go build .; then
    echo "✅ Server 封包编译成功"
else
    echo "❌ Server 封包编译失败"
    exit 1
fi

cd -

echo ""

# 分析 TODO
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚠️  需要手动处理的类型"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

analyze_todos() {
    local dir=$1
    echo ""
    echo "📁 $(basename $dir):"
    grep -r "// TODO" "$dir" 2>/dev/null | \
        sed 's/.*TODO: //' | \
        sort | uniq -c | sort -rn | head -10 | \
        awk '{printf "  • %s (%d 个)\n", substr($0, index($0,$2)), $1}'
}

analyze_todos "$OUTPUT_BASE/client"
analyze_todos "$OUTPUT_BASE/server"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 生成完成！"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "💡 提示:"
echo "  • 大部分封包可以直接使用"
echo "  • Switch 类型需要手动实现条件逻辑"
echo "  • 查看 MANUAL_FIXES.md 了解如何处理 TODO"
echo ""
