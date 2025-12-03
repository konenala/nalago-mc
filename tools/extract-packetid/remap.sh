#!/usr/bin/env bash
set -euo pipefail
SRC_JAR="/mnt/d/PrismLauncher-Windows-MinGW-w64-Portable-9.4/libraries/com/mojang/minecraft/1.21.10/minecraft-1.21.10-client.jar"
MAPPINGS_TXT="/mnt/d/PrismLauncher-Windows-MinGW-w64-Portable-9.4/libraries/com/mojang/minecraft/1.21.10/1.21.10client.txt"
TINY_JAR="/mnt/d/PrismLauncher-Windows-MinGW-w64-Portable-9.4/libraries/com/mojang/minecraft/1.21.10/tiny-remapper-0.8.10-fat.jar"
OUT_JAR="/tmp/minecraft-1.21.10-client-deobf.jar"

# tiny-remapper需要 .tiny 格式映射，mojmap txt 需先轉 tiny；本環境無轉換工具（如 SpecialSource 或 mapping-io）。
# 目前無法直接用 txt 餵 tiny-remapper，避免白跑，先提示錯誤。
echo "需要將 mojmap txt 轉成 tiny 映射後才能跑 tiny-remapper；本機無轉換工具，請提供 .tiny 或允許下載 mapping-io/SpecialSource" >&2
exit 1
