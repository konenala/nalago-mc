# 任務清單（重置 Context）

- [x] 準備工具與依賴：確認 1.21.10 client jar、mojmap txt、tiny-remapper；補齊 mapping-io 所需 annotations 依賴並可執行 CLI。
- [x] 轉換映射：使用 mapping-io 將 mojmap txt 轉為 tiny 格式供 remap 使用。（產物：`.cache/1.21.10/1.21.10client.tiny`）
- [x] 去混淆客戶端 jar：以 tiny-remapper 套用 tiny 映射，產出可讀的 named/deobf jar。（產物：`.cache/1.21.10/minecraft-1.21.10-client-named.jar`）
- [x] 導出封包 ID：撰寫/執行 extractor（針對 ProtocolInfoBuilder / GameProtocols）取得 protocol 773 各階段 PacketType → ID 對應，輸出 JSON。（產物：`.cache/1.21.10/packet_ids.json`，工具：`tools/packetid_extractor/ExtractPacketIds.java`）
- [x] 回填並生成程式：依據 JSON 產生 `packetid.go` 並加入官方名 + 舊名別名。
- [x] 測試驗證：執行 `go test ./pkg/protocol/packet/game/{client,server}`，必要時進行對真服連線檢查。
- [ ] 清除生成封包程式中的 TODO：補齊未映射型別（ItemFireworkExplosion、ItemSoundHolder、pstring/Key/CriterionIdentifier 等），讓生成程式輸出無 TODO。
- [ ] 重新生成 client/server 封包並驗證編譯：`go test ./pkg/protocol/packet/game/{client,server}`。
