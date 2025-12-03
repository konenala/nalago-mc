#!/bin/bash
# 修复子结构体中的变量名（p. -> s.）

OUTPUT_DIR="../../pkg/protocol/packet/game/client"

echo "修复子结构体变量名..."

# 对每个文件，找到子结构体的 ReadFrom/WriteTo 方法，并替换 p. 为 s.
for file in "$OUTPUT_DIR"/*.go; do
    # 使用 awk 处理：在子结构体的方法内替换 p. 为 s.
    awk '
    /^func \(s \*.*\) ReadFrom/ { in_substruct=1 }
    /^func \(s .*\) WriteTo/ { in_substruct=1 }
    /^func \(p \*.*\) ReadFrom/ { in_substruct=0 }
    /^func \(p .*\) WriteTo/ { in_substruct=0 }
    /^func \(\*.*\) PacketID/ { in_substruct=0 }
    /^type / { in_substruct=0 }
    {
        if (in_substruct && /\tp\./) {
            gsub(/\tp\./, "\ts.")
        } else if (in_substruct && /^[ \t]+p\./) {
            gsub(/^([ \t]+)p\./, "\\1s.")
        }
        print
    }
    ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
done

echo "✅ 修复完成！"
