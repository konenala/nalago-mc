#!/usr/bin/env bash
set -euo pipefail
MAP_IO="/mnt/d/PrismLauncher-Windows-MinGW-w64-Portable-9.4/libraries/com/mojang/minecraft/1.21.10/mapping-io-0.8.0.jar"
MAPPINGS_TXT="/mnt/d/PrismLauncher-Windows-MinGW-w64-Portable-9.4/libraries/com/mojang/minecraft/1.21.10/1.21.10client.txt"
OUT_TINY="/tmp/1.21.10client.tiny"
java -jar "$MAP_IO" convert "$MAPPINGS_TXT" "$OUT_TINY" --from mojang --to named --input-format mojang --output-format tiny
