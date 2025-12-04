package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// Protocol JSON 結構
type Protocol struct {
	Types map[string]interface{} `json:"types"`
	Play  struct {
		ToClient struct {
			Types map[string]interface{} `json:"types"`
		} `json:"toClient"`
		ToServer struct {
			Types map[string]interface{} `json:"types"`
		} `json:"toServer"`
	} `json:"play"`
}

// PacketField 封包欄位
type PacketField struct {
	Name             string
	Type             string
	GoType           string
	MCTag            string
	Optional         bool
	IsArray          bool
	ArrayType        string
	ArrayCount       string
	Comment          string
	ReadCode         []string
	WriteCode        []string
	NeedsPointer     bool
	IsStruct         bool           // 是否是子结构体
	StructName       string         // 子结构体名称
	ConditionalField string         // 條件字段（匿名 switch 展平）
	ConditionalValue string         // 條件值
	FlagMap          map[string]int // 若為 bitflags，旗標名稱 → 位元序
	NeedsParent      bool           // 該字段生成需要父層上下文（compareTo 包含 ../）
	IsMapper         bool           // 是否為 mapper 類型
	MapperBase       string         // mapper 的底層型別（如 varint）
}

// StructDef 结构体定义（包括嵌套的子结构体）
type StructDef struct {
	Name        string
	Fields      []PacketField
	Parent      string
	NeedsParent bool
}

// PacketDef 封包定義
type PacketDef struct {
	Name         string
	StructName   string
	Fields       []PacketField
	PacketID     string
	Imports      map[string]bool
	ImportList   []string
	SubStructs   []StructDef // 子结构体
	GenerateInit bool        // 是否生成 init 函数
}

var (
	protocolFile = flag.String("protocol", "", "Path to protocol.json")
	outputDir    = flag.String("output", "", "Output directory")
	direction    = flag.String("direction", "client", "client or server")
	verbose      = flag.Bool("v", false, "Verbose output")
	genCodec     = flag.Bool("codec", true, "Generate ReadFrom/WriteTo methods")
	packetidPkg  = flag.String("packetid", "git.konjactw.dev/falloutBot/go-mc/data/packetid", "packetid import path")
)

// 全局变量：收集所有生成的结构体名称，避免重复
var generatedStructs = make(map[string]bool)
var structCounter = make(map[string]int)
var structFlagMaps = make(map[string]map[string]map[string]int) // struct -> field -> flagName->pos
var structParent = make(map[string]string)
var structFields = make(map[string][]PacketField) // struct -> fields
var skipPackets = map[string]bool{}

func main() {
	flag.Parse()

	if *protocolFile == "" || *outputDir == "" {
		log.Fatal("Usage: enhanced-generator -protocol <protocol.json> -output <dir> -direction <client|server>")
	}

	if *verbose {
		log.Printf("📖 讀取協議文件: %s", *protocolFile)
	}

	// 讀取 protocol.json
	data, err := os.ReadFile(*protocolFile)
	if err != nil {
		log.Fatalf("❌ 讀取協議文件失敗: %v", err)
	}

	var protocol Protocol
	if err := json.Unmarshal(data, &protocol); err != nil {
		log.Fatalf("❌ 解析協議文件失敗: %v", err)
	}

	// 根據方向選擇封包類型
	var packetTypes map[string]interface{}
	var playTypes map[string]interface{}
	dirName := "Client"
	if *direction == "client" {
		packetTypes = protocol.Play.ToClient.Types
		playTypes = protocol.Play.ToClient.Types
	} else {
		packetTypes = protocol.Play.ToServer.Types
		playTypes = protocol.Play.ToServer.Types
		dirName = "Server"
	}

	// 合併 globalTypes: protocol.Types + play.toClient/toServer.types
	globalTypes := make(map[string]interface{})
	for k, v := range protocol.Types {
		globalTypes[k] = v
	}
	for k, v := range playTypes {
		globalTypes[k] = v
	}

	if *verbose {
		log.Printf("🔄 解析 %s 封包定義...", dirName)
		log.Printf("📚 加載了 %d 個全局類型定義", len(globalTypes))
	}

	// 解析所有封包
	packets := parsePackets(packetTypes, globalTypes)

	if *verbose {
		log.Printf("📊 解析統計:")
		log.Printf("  - 總封包數: %d", len(packets))

		// 统计子结构体
		totalSubStructs := 0
		for _, p := range packets {
			totalSubStructs += len(p.SubStructs)
		}
		log.Printf("  - 生成的子結構體: %d", totalSubStructs)
	}

	// 生成代碼
	if err := generatePackets(packets, *outputDir, *direction); err != nil {
		log.Fatalf("❌ 生成代碼失敗: %v", err)
	}

	fmt.Printf("✅ 成功生成 %d 個封包定義到 %s\n", len(packets), *outputDir)
}

func parsePackets(packetTypes map[string]interface{}, globalTypes map[string]interface{}) []PacketDef {
	var packets []PacketDef

	for name, def := range packetTypes {
		if !strings.HasPrefix(name, "packet_") {
			continue
		}
		if skipPackets[name] {
			if *verbose {
				log.Printf("⚠️  跳過 %s (在 skip list)", name)
			}
			continue
		}

		// 重置结构体计数器
		generatedStructs = make(map[string]bool)
		structCounter = make(map[string]int)
		structFlagMaps = make(map[string]map[string]map[string]int)
		structParent = make(map[string]string)
		structFields = make(map[string][]PacketField)

		packet := parsePacket(name, def, globalTypes)
		if packet != nil {
			packets = append(packets, *packet)
		}
	}

	// 排序
	sort.Slice(packets, func(i, j int) bool {
		return packets[i].Name < packets[j].Name
	})

	return packets
}

func parsePacket(name string, def interface{}, globalTypes map[string]interface{}) *PacketDef {
	// packet_xxx → Xxx
	structName := toPascalCase(strings.TrimPrefix(name, "packet_"))

	container, ok := def.([]interface{})
	if !ok || len(container) < 2 {
		if *verbose {
			log.Printf("⚠️  跳過 %s: 不是 container 類型", name)
		}
		return nil
	}

	if containerType, ok := container[0].(string); !ok || containerType != "container" {
		return nil
	}

	fields, ok := container[1].([]interface{})
	if !ok {
		return nil
	}

	packet := &PacketDef{
		Name:         name,
		StructName:   structName,
		Fields:       []PacketField{},
		Imports:      make(map[string]bool),
		SubStructs:   []StructDef{},
		GenerateInit: true,
	}
	structParent[structName] = ""

	// 解析字段，收集子结构体
	packet.Fields = parseFields(fields, globalTypes, structName, packet)

	// 收集需要的導入
	packet.collectImports()
	packet.buildImportList()

	if *verbose {
		log.Printf("  ✓ %s (%d 欄位, %d 子結構)", structName, len(packet.Fields), len(packet.SubStructs))
		if structName == "SetCreativeSlot" {
			log.Printf("  🔍 SetCreativeSlot SubStructs:")
			for i, s := range packet.SubStructs {
				log.Printf("    [%d] %s with %d fields", i, s.Name, len(s.Fields))
			}
			log.Printf("  🔍 SetCreativeSlot Fields:")
			for i, f := range packet.Fields {
				log.Printf("    [%d] %s: %s (IsStruct=%v)", i, f.Name, f.GoType, f.IsStruct)
			}
		}
	}

	return packet
}

// parseAnonymousField 處理匿名字段（anon: true），將其展平到父結構
func parseAnonymousField(fieldType interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef, parentFields []PacketField) []PacketField {
	var result []PacketField

	switch t := fieldType.(type) {
	case []interface{}:
		if len(t) > 0 {
			typeName, ok := t[0].(string)
			if !ok {
				if *verbose {
					log.Printf("    ❌ 匿名字段類型名稱不是字符串")
				}
				return result
			}

			if *verbose {
				log.Printf("    📝 匿名字段類型: %s", typeName)
			}

			switch typeName {
			case "switch":
				// 匿名 switch：展平所有分支的字段
				if len(t) > 1 {
					if switchConfig, ok := t[1].(map[string]interface{}); ok {
						compareField, _ := switchConfig["compareTo"].(string)

						if *verbose {
							log.Printf("    📝 switch compareTo: %s", compareField)
						}

						// 收集所有可能的字段（從 default 分支）
						// 注意：default 是 switchConfig 的鍵，不是 fields 的鍵
						if defaultBranch, exists := switchConfig["default"]; exists {
							if *verbose {
								log.Printf("    📝 找到 default 分支")
							}
							if branchDef, ok := defaultBranch.([]interface{}); ok && len(branchDef) > 0 {
								if branchType, ok := branchDef[0].(string); ok && branchType == "container" {
									if *verbose {
										log.Printf("    📝 default 分支是 container")
									}
									if len(branchDef) > 1 {
										if branchFields, ok := branchDef[1].([]interface{}); ok {
											if *verbose {
												log.Printf("    📝 解析 container 的 %d 個字段", len(branchFields))
											}
											// 遞迴解析 container 的字段
											expandedFields := parseFields(branchFields, globalTypes, parentName, packet)

											if *verbose {
												log.Printf("    📝 展開了 %d 個字段", len(expandedFields))
											}

											// 為每個字段添加條件讀寫（基於 compareField）
											for i := range expandedFields {
												expandedFields[i].ConditionalField = compareField
												expandedFields[i].ConditionalValue = "!= 0" // 默認條件
												// 包裝 ReadCode 和 WriteCode 為條件代碼
												expandedFields[i].ReadCode = wrapConditionalCode(expandedFields[i].ReadCode, compareField, "!= 0")
												expandedFields[i].WriteCode = wrapConditionalCode(expandedFields[i].WriteCode, compareField, "!= 0")
											}
											result = append(result, expandedFields...)
										} else {
											if *verbose {
												log.Printf("    ❌ branchFields 不是 []interface{}")
											}
										}
									} else {
										if *verbose {
											log.Printf("    ❌ branchDef 長度不足")
										}
									}
								} else {
									if *verbose {
										log.Printf("    ❌ default 分支不是 container，是: %v", branchType)
									}
								}
							} else {
								if *verbose {
									log.Printf("    ❌ defaultBranch 不是 []interface{}")
								}
							}
						} else {
							if *verbose {
								log.Printf("    ❌ 沒有找到 default 分支")
							}
						}
					} else {
						if *verbose {
							log.Printf("    ❌ switchConfig 不是 map")
						}
					}
				} else {
					if *verbose {
						log.Printf("    ❌ switch 定義長度不足")
					}
				}

			case "container":
				// 匿名 container：直接展平所有字段
				if len(t) > 1 {
					if containerFields, ok := t[1].([]interface{}); ok {
						result = parseFields(containerFields, globalTypes, parentName, packet)
					}
				}
			}
		}
	}

	return result
}

// wrapConditionalCode 將代碼包裝在條件語句中
func wrapConditionalCode(code []string, compareField, condition string) []string {
	if len(code) == 0 {
		return code
	}

	wrapped := []string{
		fmt.Sprintf("if p.%s %s {", toPascalCase(compareField), condition),
	}

	for _, line := range code {
		if strings.HasPrefix(line, "//") {
			wrapped = append(wrapped, line)
		} else {
			wrapped = append(wrapped, "\t"+line)
		}
	}

	wrapped = append(wrapped, "}")
	return wrapped
}

func parseFields(fields []interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef) []PacketField {
	var result []PacketField

	for _, f := range fields {
		fieldMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := fieldMap["name"].(string)
		isAnon, _ := fieldMap["anon"].(bool)

		// 處理匿名字段（展平到父結構）
		if name == "" || isAnon {
			if isAnon {
				if *verbose {
					log.Printf("  🔄 檢測到匿名字段 in %s", parentName)
				}
				fieldType := fieldMap["type"]
				// 展平匿名字段
				anonFields := parseAnonymousField(fieldType, globalTypes, parentName, packet, result)
				if *verbose {
					log.Printf("  ✅ 展平了 %d 個匿名字段 in %s", len(anonFields), parentName)
				}
				result = append(result, anonFields...)
				continue
			}
			if *verbose {
				log.Printf("⚠️  跳過無名稱欄位 in %s", parentName)
			}
			continue
		}
		fieldType := fieldMap["type"]

		field := parseFieldType(name, fieldType, globalTypes, parentName, packet, result)
		if field != nil {
			result = append(result, *field)
		}
	}

	// 記錄當前結構的欄位資訊以便父層查詢
	if parentName != "" {
		structFields[parentName] = result
	}

	return result
}

func parseFieldType(name string, fieldType interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef, parentFields []PacketField) *PacketField {
	field := &PacketField{
		Name:      toPascalCase(name),
		ReadCode:  []string{},
		WriteCode: []string{},
	}

	switch t := fieldType.(type) {
	case string:
		// 檢查是否為 globalTypes 中定義的複雜類型
		if typeDef, exists := globalTypes[t]; exists {
			// 檢查是否為原生類型（不應該遞迴展開）
			if typeDefStr, ok := typeDef.(string); ok && typeDefStr == "native" {
				// 原生類型，直接映射
				field.Type = t
				field.GoType = mapType(t)
				field.MCTag = getMCTag(t)
				field.ReadCode = generateReadCode(field.Name, t, false)
				field.WriteCode = generateWriteCode(field.Name, t, false)
				return field
			}

			if *verbose {
				log.Printf("🔄 展開類型 %s 於字段 %s.%s", t, parentName, name)
			}
			// 遞迴展開 globalTypes 中的類型定義
			return parseFieldType(name, typeDef, globalTypes, parentName, packet, parentFields)
		}

		// 簡單類型
		field.Type = t
		field.GoType = mapType(t)
		field.MCTag = getMCTag(t)
		field.ReadCode = generateReadCode(field.Name, t, false)
		field.WriteCode = generateWriteCode(field.Name, t, false)

	case []interface{}:
		// 複雜類型
		if len(t) > 0 {
			typeName, ok := t[0].(string)
			if !ok {
				return nil
			}

			field.Type = typeName

			switch typeName {
			case "option":
				// Optional 類型
				return parseOptionalField(name, t, globalTypes, parentName, packet)

			case "array":
				// 數組類型
				return parseArrayField(name, t, globalTypes, parentName, packet)

			case "container":
				// 嵌套 container - 生成子结构体
				return parseContainerField(name, t, globalTypes, parentName, packet)

			case "bitfield":
				// Bitfield 类型 - 使用 int32/uint32
				field.GoType = "int32"
				field.Comment = "// Bitfield - see protocol spec for bit layout"
				field.ReadCode = []string{
					fmt.Sprintf("temp, err = (*pk.Int)(&p.%s).ReadFrom(r)", field.Name),
					"n += temp",
					"if err != nil { return n, err }",
				}
				field.WriteCode = []string{
					fmt.Sprintf("temp, err = pk.Int(p.%s).WriteTo(w)", field.Name),
					"n += temp",
					"if err != nil { return n, err }",
				}

			case "switch":
				// Switch 類型 - 根據 compareTo 產生對應欄位
				return generateSwitchField(name, t, parentFields, parentName)

			case "bitflags":
				// 位元旗標，依據底層 type（通常 u8）
				baseType := "u8"
				if len(t) > 1 {
					if def, ok := t[1].(map[string]interface{}); ok {
						if bt, ok := def["type"].(string); ok {
							baseType = bt
						}
					}
				}
				field.GoType = mapType(baseType)
				field.Comment = "// Bitflags"
				field.ReadCode = generateReadCode(field.Name, baseType, false)
				field.WriteCode = generateWriteCode(field.Name, baseType, false)
				// 收集旗標位置
				if len(t) > 1 {
					if def, ok := t[1].(map[string]interface{}); ok {
						if flags, ok := def["flags"].([]interface{}); ok {
							field.FlagMap = make(map[string]int)
							for i, f := range flags {
								if s, ok := f.(string); ok {
									field.FlagMap[s] = i
								}
							}
							// 記錄於全域表供 ../ 查詢
							if structFlagMaps[parentName] == nil {
								structFlagMaps[parentName] = make(map[string]map[string]int)
							}
							structFlagMaps[parentName][field.Name] = field.FlagMap
						}
					}
				}

			case "buffer":
				// ByteArray
				field.GoType = "[]byte"
				field.MCTag = "`mc:\"ByteArray\"`"
				if len(t) > 1 {
					if bufDef, ok := t[1].(map[string]interface{}); ok {
						if countType, ok := bufDef["countType"].(string); ok && countType == "varint" {
							field.ReadCode = []string{"temp, err = (*pk.ByteArray)(&p." + field.Name + ").ReadFrom(r)", "n += temp", "if err != nil { return n, err }"}
							field.WriteCode = []string{"temp, err = (*pk.ByteArray)(&p." + field.Name + ").WriteTo(w)", "n += temp", "if err != nil { return n, err }"}
						} else if countType == "restBuffer" {
							field.GoType = "pk.PluginMessageData"
							field.ReadCode = []string{"temp, err = (*pk.PluginMessageData)(&p." + field.Name + ").ReadFrom(r)", "n += temp", "if err != nil && err != io.EOF { return n, err }"}
							field.WriteCode = []string{"temp, err = (*pk.PluginMessageData)(&p." + field.Name + ").WriteTo(w)", "n += temp", "if err != nil { return n, err }"}
						} else if _, ok := bufDef["count"]; ok {
							field.ReadCode = []string{"temp, err = (*pk.ByteArray)(&p." + field.Name + ").ReadFrom(r)", "n += temp", "if err != nil { return n, err }"}
							field.WriteCode = []string{"temp, err = (*pk.ByteArray)(&p." + field.Name + ").WriteTo(w)", "n += temp", "if err != nil { return n, err }"}
						}
					}
				} else {
					field.ReadCode = []string{"temp, err = (*pk.ByteArray)(&p." + field.Name + ").ReadFrom(r)", "n += temp", "if err != nil { return n, err }"}
					field.WriteCode = []string{"temp, err = (*pk.ByteArray)(&p." + field.Name + ").WriteTo(w)", "n += temp", "if err != nil { return n, err }"}
				}

			case "pstring", "string":
				// 包含額外屬性的 pstring 仍以字串處理
				field.Type = typeName
				field.GoType = mapType(typeName)
				field.MCTag = getMCTag(typeName)
				field.ReadCode = generateReadCode(field.Name, typeName, false)
				field.WriteCode = generateWriteCode(field.Name, typeName, false)

			case "entityMetadataLoop":
				field.Type = typeName
				field.GoType = mapType(typeName)
				field.ReadCode = generateReadCode(field.Name, typeName, false)
				field.WriteCode = generateWriteCode(field.Name, typeName, false)

			case "topBitSetTerminatedArray":
				field.Type = typeName
				field.GoType = mapType(typeName)
				field.ReadCode = generateReadCode(field.Name, typeName, false)
				field.WriteCode = generateWriteCode(field.Name, typeName, false)
			case "registryEntryHolder", "registryEntryHolderSet":
				field.Type = typeName
				field.GoType = mapType(typeName)
				field.MCTag = getMCTag(typeName)
				field.ReadCode = generateReadCode(field.Name, typeName, false)
				field.WriteCode = generateWriteCode(field.Name, typeName, false)

			case "mapper":
				// 數值映射到字串的型別
				if len(t) > 1 {
					if mapperDef, ok := t[1].(map[string]interface{}); ok {
						baseType, _ := mapperDef["type"].(string)
						field.IsMapper = true
						field.MapperBase = baseType
						mappings, _ := mapperDef["mappings"].(map[string]interface{})
						field.GoType = "string"
						field.Comment = "// Mapper to string"

						// 生成讀取代碼：讀取 baseType，再 switch 映射到字串
						pkType := mapTypeToPkType(baseType)
						field.ReadCode = []string{
							fmt.Sprintf("var mapperVal pk.%s", pkType),
							fmt.Sprintf("temp, err = mapperVal.ReadFrom(r)"),
							"n += temp",
							"if err != nil { return n, err }",
							"switch mapperVal {",
						}
						field.WriteCode = []string{
							"switch p." + field.Name + " {",
						}

						for k, v := range mappings {
							// k 是數字字串
							field.ReadCode = append(field.ReadCode,
								fmt.Sprintf("case %s:", k),
								fmt.Sprintf("	p.%s = \"%v\"", field.Name, v),
							)
							field.WriteCode = append(field.WriteCode,
								fmt.Sprintf("case \"%v\":", v),
								fmt.Sprintf("	temp, err = pk.%s(%s).WriteTo(w)", pkType, k),
								"	n += temp",
								"	if err != nil { return n, err }",
							)
						}
						field.ReadCode = append(field.ReadCode,
							"default:",
							fmt.Sprintf("	return n, fmt.Errorf(\"unknown mapper value %%d for %s\", mapperVal)", field.Name),
							"}",
						)
						field.WriteCode = append(field.WriteCode,
							"default:",
							fmt.Sprintf("	return n, fmt.Errorf(\"unknown %s value %%v\", p.%s)", field.Name, field.Name),
							"}",
						)
					}
				}

			default:
				// 未知複雜類型
				field.GoType = "interface{}"
				field.Comment = fmt.Sprintf("// TODO: Implement %s type", typeName)
				field.ReadCode = []string{fmt.Sprintf("// TODO: Read %s", field.Name)}
				field.WriteCode = []string{fmt.Sprintf("// TODO: Write %s", field.Name)}
			}
		}
	}

	return field
}

// 解析 Optional 字段
func parseOptionalField(name string, t []interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef) *PacketField {
	field := &PacketField{
		Name:         toPascalCase(name),
		Optional:     true,
		NeedsPointer: true,
		ReadCode:     []string{},
		WriteCode:    []string{},
	}

	if len(t) > 1 {
		innerType := t[1]

		// 检查内部类型
		switch inner := innerType.(type) {
		case string:
			// 檢查是否為 globalTypes 中定義的複雜類型
			if typeDef, exists := globalTypes[inner]; exists {
				// 遞迴展開為複雜類型的 optional
				baseField := parseFieldType("temp", typeDef, globalTypes, parentName, packet, []PacketField{})
				if baseField != nil && baseField.IsStruct {
					// 如果是結構體，轉換為可選的結構體
					field.GoType = "*" + baseField.StructName
					field.IsStruct = true
					field.StructName = baseField.StructName
					field.NeedsParent = baseField.NeedsParent
					field.ReadCode = generateOptionalStructReadCode(field.Name, baseField.StructName, baseField.NeedsParent)
					field.WriteCode = generateOptionalStructWriteCode(field.Name, baseField.StructName, baseField.NeedsParent)
					return field
				}
			}

			// 简单类型的 optional
			field.GoType = "*" + mapType(inner)
			field.ReadCode = generateOptionalReadCode(field.Name, inner)
			field.WriteCode = generateOptionalWriteCode(field.Name, inner)

		case []interface{}:
			// 复杂类型的 optional（如 option[container]）
			if len(inner) > 0 {
				if innerTypeName, ok := inner[0].(string); ok {
					if innerTypeName == "container" {
						// option[container] - 生成可选的子结构体
						subStructName := generateSubStructName(parentName, field.Name)
						subStruct := parseContainerToStruct(subStructName, inner, globalTypes, parentName, packet)
						if subStruct != nil {
							packet.SubStructs = append(packet.SubStructs, *subStruct)
							field.GoType = "*" + subStructName
							field.IsStruct = true
							field.StructName = subStructName
							field.NeedsParent = subStruct.NeedsParent

							// 生成 optional struct 的读写代码
							field.ReadCode = generateOptionalStructReadCode(field.Name, subStructName, subStruct.NeedsParent)
							field.WriteCode = generateOptionalStructWriteCode(field.Name, subStructName, subStruct.NeedsParent)
						}
					} else if innerTypeName == "option" {
						// option[option[T]] 簡化為 *interface{}
						field.GoType = "*interface{}"
						field.Comment = "// TODO: nested option type"
						field.ReadCode = []string{
							fmt.Sprintf("var has%s pk.Boolean", field.Name),
							fmt.Sprintf("temp, err = has%s.ReadFrom(r)", field.Name),
							"n += temp",
							"if err != nil { return n, err }",
							fmt.Sprintf("if has%s {", field.Name),
							"	var v interface{}",
							"	// TODO: Read nested option payload",
							fmt.Sprintf("	p.%s = &v", field.Name),
							"}",
						}
						field.WriteCode = []string{
							fmt.Sprintf("if p.%s != nil {", field.Name),
							"	temp, err = pk.Boolean(true).WriteTo(w)",
							"	n += temp",
							"	if err != nil { return n, err }",
							"	// TODO: Write nested option payload",
							"} else {",
							"	temp, err = pk.Boolean(false).WriteTo(w)",
							"	n += temp",
							"	if err != nil { return n, err }",
							"}",
						}
					} else if innerTypeName == "array" {
						// option[array] -> *[]T (支援簡單型別或 container)
						if len(inner) > 1 {
							if arrDef, ok := inner[1].(map[string]interface{}); ok {
								countType, _ := arrDef["countType"].(string)
								if elemStr, ok := arrDef["type"].(string); ok && elemStr != "" {
									// 簡單元素
									field.GoType = "*[]" + mapType(elemStr)
									readLines := generateValueReadLines(elemStr, "v")
									writeLines := generateValueWriteLines(elemStr, "(*p."+field.Name+")[i]")
									generateOptionalArrayRW(field, countType, readLines, writeLines)
								} else if elemArr, ok := arrDef["type"].([]interface{}); ok && len(elemArr) > 0 {
									if elemName, ok := elemArr[0].(string); ok && elemName == "container" {
										// 生成子結構
										subStructName := generateSubStructName(parentName, field.Name+"Entry")
										subStruct := parseContainerToStruct(subStructName, elemArr, globalTypes, parentName, packet)
										if subStruct != nil {
											packet.SubStructs = append(packet.SubStructs, *subStruct)
											field.GoType = "*[]" + subStructName
											readLines := []string{
												"	var v " + subStructName,
												func() string {
													if subStruct.NeedsParent {
														return "	temp, err = v.ReadFromWithParent(r, p)"
													}
													return "	temp, err = v.ReadFrom(r)"
												}(),
												"	n += temp",
												"	if err != nil { return n, err }",
											}
											writeLines := []string{
												func() string {
													if subStruct.NeedsParent {
														return "	temp, err = (*p." + field.Name + ")[i].WriteToWithParent(w, &p)"
													}
													return "	temp, err = (*p." + field.Name + ")[i].WriteTo(w)"
												}(),
												"	n += temp",
												"	if err != nil { return n, err }",
											}
											field.NeedsParent = subStruct.NeedsParent
											generateOptionalArrayRW(field, countType, readLines, writeLines)
										}
									}
								}
							}
						}
					} else if innerTypeName == "buffer" || innerTypeName == "restBuffer" {
						// option[buffer]
						field.GoType = "*[]byte"
						if innerTypeName == "restBuffer" {
							field.GoType = "*pk.PluginMessageData"
						}
						field.ReadCode = generateOptionalReadCode(field.Name, innerTypeName)
						field.WriteCode = generateOptionalWriteCode(field.Name, innerTypeName)
					} else if innerTypeName == "pstring" || innerTypeName == "string" {
						field.GoType = "*string"
						field.ReadCode = generateOptionalReadCode(field.Name, innerTypeName)
						field.WriteCode = generateOptionalWriteCode(field.Name, innerTypeName)
					} else {
						// 其他複合型（array/switch等）暫回退 interface{}
						field.GoType = "*interface{}"
						field.Comment = "// TODO: Optional complex type"
						field.ReadCode = []string{"// TODO: Read optional complex type"}
						field.WriteCode = []string{"// TODO: Write optional complex type"}
					}
				}
			}
		default:
			// 無法解析
			field.GoType = "*interface{}"
			field.Comment = "// TODO: Optional unknown type"
			field.ReadCode = []string{"// TODO: Read optional unknown type"}
			field.WriteCode = []string{"// TODO: Write optional unknown type"}
		}
	}

	return field
}

// 解析 Array 字段
func parseArrayField(name string, t []interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef) *PacketField {
	field := &PacketField{
		Name:      toPascalCase(name),
		IsArray:   true,
		ReadCode:  []string{},
		WriteCode: []string{},
	}

	if len(t) > 1 {
		if arrayDef, ok := t[1].(map[string]interface{}); ok {
			countType, _ := arrayDef["countType"].(string)
			if countType == "" {
				countType = "varint"
			}
			field.ArrayCount = countType

			arrayElemType := arrayDef["type"]

			// 检查数组元素类型
			switch elemType := arrayElemType.(type) {
			case string:
				// 檢查是否為 globalTypes 中定義的複雜類型
				if typeDef, exists := globalTypes[elemType]; exists {
					// 遞迴展開為複雜類型的數組
					baseField := parseFieldType("temp", typeDef, globalTypes, parentName, packet, []PacketField{})
					if baseField != nil && baseField.IsStruct {
						// 如果是結構體，生成結構體數組
						field.GoType = "[]" + baseField.StructName
						field.IsStruct = true
						field.StructName = baseField.StructName
						field.NeedsParent = baseField.NeedsParent
						field.ReadCode = generateStructArrayReadCode(field.Name, baseField.StructName, countType, baseField.NeedsParent)
						field.WriteCode = generateStructArrayWriteCode(field.Name, baseField.StructName, countType, baseField.NeedsParent)
						return field
					}
				}

				// 简单类型数组
				field.ArrayType = elemType
				field.GoType = "[]" + mapType(elemType)
				if mapType(elemType) == "interface{}" {
					field.Comment = "// TODO: Array element type " + elemType + " unsupported"
				} else {
					field.ReadCode = generateArrayReadCode(field.Name, elemType, countType)
					field.WriteCode = generateArrayWriteCode(field.Name, elemType, countType)
				}

			case []interface{}:
				// 复杂类型数组
				if len(elemType) > 0 {
					if elemTypeName, ok := elemType[0].(string); ok {
						if elemTypeName == "container" {
							// array[container] - 生成结构体数组
							subStructName := generateSubStructName(parentName, field.Name+"Entry")
							subStruct := parseContainerToStruct(subStructName, elemType, globalTypes, parentName, packet)
							if subStruct != nil {
								packet.SubStructs = append(packet.SubStructs, *subStruct)
								field.GoType = "[]" + subStructName
								field.IsStruct = true
								field.StructName = subStructName
								field.NeedsParent = subStruct.NeedsParent

								// 生成结构体数组的读写代码
								field.ReadCode = generateStructArrayReadCode(field.Name, subStructName, countType, subStruct.NeedsParent)
								field.WriteCode = generateStructArrayWriteCode(field.Name, subStructName, countType, subStruct.NeedsParent)
							}
						} else if elemTypeName == "array" {
							// array[array] - 嵌套数组
							// 简化处理：内层如果是简单类型，生成 [][]type
							if len(elemType) > 1 {
								if innerArrayDef, ok := elemType[1].(map[string]interface{}); ok {
									if innerType, ok := innerArrayDef["type"].(string); ok {
										field.GoType = "[][]" + mapType(innerType)
										field.ReadCode = generateNestedArrayReadCode(field.Name, innerType, countType)
										field.WriteCode = generateNestedArrayWriteCode(field.Name, innerType, countType)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 如果没有成功解析，使用默认值
	if field.GoType == "" {
		field.GoType = "[]interface{}"
		field.Comment = "// TODO: Complex array element type"
	}

	return field
}

// 解析 Container 字段（嵌套结构体）
func parseContainerField(name string, t []interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef) *PacketField {
	field := &PacketField{
		Name:      toPascalCase(name),
		IsStruct:  true,
		ReadCode:  []string{},
		WriteCode: []string{},
	}

	subStructName := generateSubStructName(parentName, field.Name)
	subStruct := parseContainerToStruct(subStructName, t, globalTypes, parentName, packet)

	if subStruct != nil {
		packet.SubStructs = append(packet.SubStructs, *subStruct)
		field.GoType = subStructName
		field.StructName = subStructName
		field.NeedsParent = subStruct.NeedsParent

		// 生成嵌套结构体的读写代码
		readCall := fmt.Sprintf("p.%s.ReadFrom(r)", field.Name)
		if subStruct.NeedsParent {
			readCall = fmt.Sprintf("p.%s.ReadFromWithParent(r, p)", field.Name)
		}
		writeCall := fmt.Sprintf("p.%s.WriteTo(w)", field.Name)
		if subStruct.NeedsParent {
			writeCall = fmt.Sprintf("p.%s.WriteToWithParent(w, &p)", field.Name)
		}
		field.ReadCode = []string{
			fmt.Sprintf("temp, err = %s", readCall),
			"n += temp",
			"if err != nil { return n, err }",
		}
		field.WriteCode = []string{
			fmt.Sprintf("temp, err = %s", writeCall),
			"n += temp",
			"if err != nil { return n, err }",
		}
	} else {
		field.GoType = "interface{}"
		field.Comment = "// TODO: Failed to parse nested container"
	}

	return field
}

// 将 container 解析为结构体
func parseContainerToStruct(structName string, containerDef []interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef) *StructDef {
	if len(containerDef) < 2 {
		return nil
	}

	fields, ok := containerDef[1].([]interface{})
	if !ok {
		return nil
	}

	structParent[structName] = parentName

	childFields := parseFields(fields, globalTypes, structName, packet)
	needsParent := false
	for _, f := range childFields {
		if f.NeedsParent {
			needsParent = true
			break
		}
	}
	subStruct := &StructDef{
		Name:        structName,
		Fields:      childFields,
		Parent:      parentName,
		NeedsParent: needsParent,
	}

	return subStruct
}

// 生成子结构体名称（避免重复）
func generateSubStructName(parentName, fieldName string) string {
	baseName := parentName + fieldName

	// 如果已经生成过，添加序号
	if generatedStructs[baseName] {
		structCounter[baseName]++
		return fmt.Sprintf("%s%d", baseName, structCounter[baseName])
	}

	generatedStructs[baseName] = true
	return baseName
}

// 生成 optional struct 的读取代码
func generateOptionalStructReadCode(fieldName, structName string, needsParent bool) []string {
	return []string{
		fmt.Sprintf("var has%s pk.Boolean", fieldName),
		fmt.Sprintf("temp, err = has%s.ReadFrom(r)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("if has%s {", fieldName),
		fmt.Sprintf("	p.%s = &%s{}", fieldName, structName),
		func() string {
			if needsParent {
				return fmt.Sprintf("	temp, err = p.%s.ReadFromWithParent(r, p)", fieldName)
			}
			return fmt.Sprintf("	temp, err = p.%s.ReadFrom(r)", fieldName)
		}(),
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成 optional struct 的写入代码
func generateOptionalStructWriteCode(fieldName, structName string, needsParent bool) []string {
	return []string{
		fmt.Sprintf("if p.%s != nil {", fieldName),
		"	temp, err = pk.Boolean(true).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		func() string {
			if needsParent {
				return fmt.Sprintf("	temp, err = p.%s.WriteToWithParent(w, &p)", fieldName)
			}
			return fmt.Sprintf("	temp, err = p.%s.WriteTo(w)", fieldName)
		}(),
		"	n += temp",
		"	if err != nil { return n, err }",
		"} else {",
		"	temp, err = pk.Boolean(false).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成结构体数组的读取代码
func generateStructArrayReadCode(fieldName, structName, countType string, needsParent bool) []string {
	countVar := safeIdent(strings.ToLower(fieldName[:1]) + fieldName[1:] + "Count")
	return []string{
		fmt.Sprintf("var %s pk.%s", countVar, mapTypeToPkType(countType)),
		fmt.Sprintf("temp, err = %s.ReadFrom(r)", countVar),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("p.%s = make([]%s, %s)", fieldName, structName, countVar),
		fmt.Sprintf("for i := 0; i < int(%s); i++ {", countVar),
		func() string {
			if needsParent {
				return fmt.Sprintf("	temp, err = p.%s[i].ReadFromWithParent(r, p)", fieldName)
			}
			return fmt.Sprintf("	temp, err = p.%s[i].ReadFrom(r)", fieldName)
		}(),
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成结构体数组的写入代码
func generateStructArrayWriteCode(fieldName, structName, countType string, needsParent bool) []string {
	return []string{
		fmt.Sprintf("temp, err = pk.%s(len(p.%s)).WriteTo(w)", mapTypeToPkType(countType), fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("for i := range p.%s {", fieldName),
		func() string {
			if needsParent {
				return fmt.Sprintf("	temp, err = p.%s[i].WriteToWithParent(w, &p)", fieldName)
			}
			return fmt.Sprintf("	temp, err = p.%s[i].WriteTo(w)", fieldName)
		}(),
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成嵌套数组的读取代码（array[array[T]]）
func generateNestedArrayReadCode(fieldName, innerType, countType string) []string {
	countVar := safeIdent(strings.ToLower(fieldName[:1]) + fieldName[1:] + "Count")
	innerGoType := mapType(innerType)

	return []string{
		fmt.Sprintf("var %s pk.VarInt", countVar),
		fmt.Sprintf("temp, err = %s.ReadFrom(r)", countVar),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("p.%s = make([][]%s, %s)", fieldName, innerGoType, countVar),
		fmt.Sprintf("for i := 0; i < int(%s); i++ {", countVar),
		"	var innerCount pk.VarInt",
		"	temp, err = innerCount.ReadFrom(r)",
		"	n += temp",
		"	if err != nil { return n, err }",
		fmt.Sprintf("	p.%s[i] = make([]%s, innerCount)", fieldName, innerGoType),
		"	for j := 0; j < int(innerCount); j++ {",
		generateInnerArrayReadLine(fieldName, innerType),
		"		n += temp",
		"		if err != nil { return n, err }",
		"	}",
		"}",
	}
}

func generateInnerArrayReadLine(fieldName, innerType string) string {
	goType := mapType(innerType)
	pkType := mapTypeToPkType(innerType)

	switch innerType {
	case "string", "pstring":
		return fmt.Sprintf("		var elem pk.String\n		temp, err = elem.ReadFrom(r)\n		p.%s[i][j] = string(elem)", fieldName)
	case "varint", "varlong":
		return fmt.Sprintf("		var elem pk.VarInt\n		temp, err = elem.ReadFrom(r)\n		p.%s[i][j] = %s(elem)", fieldName, goType)
	default:
		// 使用 pk type 读取
		if pkType != innerType {
			return fmt.Sprintf("		var elem pk.%s\n		temp, err = elem.ReadFrom(r)\n		p.%s[i][j] = %s(elem)", pkType, fieldName, goType)
		}
		return fmt.Sprintf("		temp, err = p.%s[i][j].ReadFrom(r)", fieldName)
	}
}

// 生成嵌套数组的写入代码
func generateNestedArrayWriteCode(fieldName, innerType, countType string) []string {
	return []string{
		fmt.Sprintf("temp, err = pk.VarInt(len(p.%s)).WriteTo(w)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("for i := range p.%s {", fieldName),
		fmt.Sprintf("	temp, err = pk.VarInt(len(p.%s[i])).WriteTo(w)", fieldName),
		"	n += temp",
		"	if err != nil { return n, err }",
		fmt.Sprintf("	for j := range p.%s[i] {", fieldName),
		generateInnerArrayWriteLine(fieldName, innerType),
		"		n += temp",
		"		if err != nil { return n, err }",
		"	}",
		"}",
	}
}

func generateInnerArrayWriteLine(fieldName, innerType string) string {
	pkType := mapTypeToPkType(innerType)

	switch innerType {
	case "string", "pstring":
		return fmt.Sprintf("		temp, err = pk.String(p.%s[i][j]).WriteTo(w)", fieldName)
	case "varint", "varlong":
		return fmt.Sprintf("		temp, err = pk.VarInt(p.%s[i][j]).WriteTo(w)", fieldName)
	default:
		// 使用 pk type 写入
		if pkType != innerType {
			return fmt.Sprintf("		temp, err = pk.%s(p.%s[i][j]).WriteTo(w)", pkType, fieldName)
		}
		return fmt.Sprintf("		temp, err = p.%s[i][j].WriteTo(w)", fieldName)
	}
}

func mapTypeToPk(t string) string {
	mapping := map[string]string{
		"i8":   "Byte",
		"u8":   "UnsignedByte",
		"i16":  "Short",
		"u16":  "UnsignedShort",
		"i32":  "Int",
		"i64":  "Long",
		"f32":  "Float",
		"f64":  "Double",
		"bool": "Boolean",
	}
	if mapped, ok := mapping[t]; ok {
		return mapped
	}
	return "Int"
}

// 類型映射
func mapType(t string) string {
	mapping := map[string]string{
		"void":                     "struct{}",    // 佔位型別
		"native":                   "interface{}", // minecraft-data 標記的原生型別
		"varint":                   "int32",
		"varlong":                  "int64",
		"optvarint":                "*int32",
		"i8":                       "int8",
		"i16":                      "int16",
		"i32":                      "int32",
		"i64":                      "int64",
		"u8":                       "uint8",
		"ContainerID":              "int8",
		"packedChunkPos":           "int64",
		"PositionUpdateRelatives":  "int32",
		"soundSource":              "int32",
		"u16":                      "uint16",
		"u32":                      "uint32",
		"u64":                      "uint64",
		"f32":                      "float32",
		"f64":                      "float64",
		"bool":                     "bool",
		"string":                   "string",
		"pstring":                  "string",
		"Key":                      "string",
		"CriterionIdentifier":      "string",
		"UUID":                     "pk.UUID",
		"buffer":                   "[]byte",
		"ByteArray":                "[]byte",
		"restBuffer":               "pk.PluginMessageData",
		"topBitSetTerminatedArray": "pk.PluginMessageData",
		"bitflags":                 "uint8",
		"registryEntryHolder":      "string",
		"registryEntryHolderSet":   "[]string",
		"ItemFireworkExplosion":    "pk.NBTField",
		"ItemSoundHolder":          "pk.NBTField",
		"nbt":                      "pk.NBTField",
		"anonymousNbt":             "pk.NBTField",
		"anonOptionalNbt":          "pk.NBTField",
		"optionalNbt":              "*pk.NBTField",
		"position":                 "pk.Position",
		"slot":                     "slot.Slot",
		"Slot":                     "slot.Slot",
		"component":                "pk.Component",
		"textComponent":            "pk.Component",
		"entityMetadata":           "metadata.EntityMetadata",
		"entityMetadataLoop":       "metadata.EntityMetadata",
		"vec3f64":                  "[3]float64",
		"vec3f":                    "[3]float32",
		"vec3i":                    "[3]int32",
		"HashedSlot":               "slot.HashedSlot",
		"MovementFlags":            "uint8",       // bitflags
		"game_profile":             "pk.NBTField", // 簡化：使用 NBT 讀寫
		"chat_session":             "pk.NBTField", // 簡化：使用 NBT 讀寫
		"IDSet":                    "[]int32",     // VarInt 長度 + VarInt 元素
	}

	if mapped, ok := mapping[t]; ok {
		return mapped
	}

	if *verbose {
		log.Printf("⚠️  未映射的類型: %s", t)
	}
	return "interface{}"
}

func getMCTag(t string) string {
	switch t {
	case "varint", "varlong", "optvarint":
		return "`mc:\"VarInt\"`"
	case "string", "pstring", "Key", "CriterionIdentifier", "registryEntryHolder":
		return "`mc:\"String\"`"
	case "buffer":
		return "`mc:\"ByteArray\"`"
	case "nbt", "anonymousNbt":
		return "`mc:\"NBT\"`"
	default:
		return ""
	}
}

// 生成讀取代碼
func generateReadCode(fieldName, typeName string, optional bool) []string {
	if mapType(typeName) == "interface{}" {
		return []string{"// TODO: Read " + fieldName + " (unsupported type " + typeName + ")"}
	}
	varName := safeIdent(strings.ToLower(fieldName[:1]) + fieldName[1:])
	// 避免與 package 名稱衝突導致誤判 import
	if varName == "metadata" {
		varName = "metadataVal"
	}
	var code []string

	switch typeName {
	case "varint", "varlong":
		code = []string{
			fmt.Sprintf("var %s pk.VarInt", varName),
			fmt.Sprintf("temp, err = %s.ReadFrom(r)", varName),
			"n += temp",
			"if err != nil { return n, err }",
			fmt.Sprintf("p.%s = int32(%s)", fieldName, varName),
		}
	case "i8":
		code = []string{
			fmt.Sprintf("var %s int8", varName),
			fmt.Sprintf("temp, err = (*pk.Byte)(&%s).ReadFrom(r)", varName),
			"n += temp",
			"if err != nil { return n, err }",
			fmt.Sprintf("p.%s = %s", fieldName, varName),
		}
	case "u8":
		code = []string{
			fmt.Sprintf("var %s pk.UnsignedByte", varName),
			fmt.Sprintf("temp, err = %s.ReadFrom(r)", varName),
			"n += temp",
			"if err != nil { return n, err }",
			fmt.Sprintf("p.%s = uint8(%s)", fieldName, varName),
		}
	case "i16", "u16":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.%s)(&p.%s).ReadFrom(r)", mapTypeToPkType(typeName), fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i32", "u32":
		if typeName == "u32" {
			code = []string{
				"	var elem pk.Int",
				"	temp, err = elem.ReadFrom(r)",
				"	n += temp",
				"	if err != nil { return n, err }",
				fmt.Sprintf("	p.%s = uint32(elem)", fieldName),
			}
		} else {
			code = []string{
				fmt.Sprintf("temp, err = (*pk.Int)(&p.%s).ReadFrom(r)", fieldName),
				"n += temp",
				"if err != nil { return n, err }",
			}
		}
	case "i64", "u64":
		if typeName == "u64" {
			code = []string{
				"	var elem pk.Long",
				"	temp, err = elem.ReadFrom(r)",
				"	n += temp",
				"	if err != nil { return n, err }",
				fmt.Sprintf("	p.%s = uint64(elem)", fieldName),
			}
		} else {
			code = []string{
				fmt.Sprintf("temp, err = (*pk.Long)(&p.%s).ReadFrom(r)", fieldName),
				"n += temp",
				"if err != nil { return n, err }",
			}
		}
	case "bitflags":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.UnsignedByte)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "f32":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Float)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "f64":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Double)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "bool":
		code = []string{
			fmt.Sprintf("var %s pk.Boolean", varName),
			fmt.Sprintf("temp, err = %s.ReadFrom(r)", varName),
			"n += temp",
			"if err != nil { return n, err }",
			fmt.Sprintf("p.%s = bool(%s)", fieldName, varName),
		}
	case "string", "pstring", "Key", "CriterionIdentifier", "registryEntryHolder":
		code = []string{
			fmt.Sprintf("var %s pk.String", varName),
			fmt.Sprintf("temp, err = %s.ReadFrom(r)", varName),
			"n += temp",
			"if err != nil { return n, err }",
			fmt.Sprintf("p.%s = string(%s)", fieldName, varName),
		}
	case "buffer":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.ByteArray)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "restBuffer", "topBitSetTerminatedArray":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.PluginMessageData)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil && err != io.EOF { return n, err }",
		}
	case "registryEntryHolderSet":
		code = []string{
			"var count pk.VarInt",
			"temp, err = count.ReadFrom(r)",
			"n += temp",
			"if err != nil { return n, err }",
			"if count < 0 { return n, fmt.Errorf(\"negative registryEntryHolderSet length\") }",
			"p." + fieldName + " = make([]string, count)",
			"for i := int32(0); i < int32(count); i++ {",
			"	var s pk.String",
			"	temp, err = s.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	p." + fieldName + "[i] = string(s)",
			"}",
		}
	case "ItemFireworkExplosion", "ItemSoundHolder":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.NBTField)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "UUID", "position", "nbt", "anonymousNbt", "anonOptionalNbt", "optionalNbt", "component", "textComponent":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.%s)(&p.%s).ReadFrom(r)", mapTypeToPkType(typeName), fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "game_profile", "chat_session":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.%s)(&p.%s).ReadFrom(r)", mapTypeToPkType(typeName), fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "slot", "Slot":
		code = []string{
			fmt.Sprintf("temp, err = (*slot.Slot)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "entityMetadata", "entityMetadataLoop":
		code = []string{
			fmt.Sprintf("temp, err = (*metadata.EntityMetadata)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "vec3f64":
		code = []string{
			fmt.Sprintf("var _%s [3]float64", varName),
			"	for i := 0; i < 3; i++ {",
			"		var d pk.Double",
			"		temp, err = d.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		_" + varName + "[i] = float64(d)",
			"	}",
			fmt.Sprintf("p.%s = _%s", fieldName, varName),
		}
	default:
		code = []string{fmt.Sprintf("// TODO: Read %s (%s)", fieldName, typeName)}
	}

	return code
}

func mapTypeToPkType(t string) string {
	mapping := map[string]string{
		"nbt":                 "NBTField",
		"anonymousNbt":        "NBTField",
		"anonOptionalNbt":     "NBTField",
		"optionalNbt":         "NBTField",
		"varint":              "VarInt",
		"varlong":             "VarLong",
		"i8":                  "Byte",
		"u8":                  "UnsignedByte",
		"i16":                 "Short",
		"u16":                 "UnsignedShort",
		"i32":                 "Int",
		"u32":                 "Int",
		"i64":                 "Long",
		"u64":                 "Long",
		"f32":                 "Float",
		"f64":                 "Double",
		"bool":                "Boolean",
		"UUID":                "UUID",
		"position":            "Position",
		"component":           "Component",
		"textComponent":       "Component",
		"game_profile":        "NBTField",
		"chat_session":        "NBTField",
		"string":              "String",
		"pstring":             "String",
		"Key":                 "String",
		"CriterionIdentifier": "String",
		"buffer":              "ByteArray",
		"bitflags":            "UnsignedByte",
	}
	if mapped, ok := mapping[t]; ok {
		return mapped
	}
	return t
}

var goKeywords = map[string]bool{
	"break": true, "default": true, "func": true, "interface": true, "select": true,
	"case": true, "defer": true, "go": true, "map": true, "struct": true,
	"chan": true, "else": true, "goto": true, "package": true, "switch": true,
	"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
	"continue": true, "for": true, "import": true, "return": true, "var": true,
}

// safeIdent 避免與關鍵字或保留名衝突的區域變數名稱
func safeIdent(name string) string {
	if name == "" {
		return "v"
	}
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	lower := strings.ToLower(name)
	if goKeywords[name] || goKeywords[lower] {
		name = "_" + name
	}
	if name == "p" || name == "s" || name == "v" {
		name = name + "Var"
	}
	return name
}

// SwitchConfig Switch 類型的配置
type SwitchConfig struct {
	CompareTo string                 // 比較的字段路徑，如 "flags/has_background_texture"
	Fields    map[string]interface{} // 條件值 -> 類型映射
	Default   string                 // 預設類型
}

// 解析 switch 配置
func parseSwitchConfig(switchDef []interface{}) *SwitchConfig {
	if len(switchDef) < 2 {
		return nil
	}

	configMap, ok := switchDef[1].(map[string]interface{})
	if !ok {
		return nil
	}

	cfg := &SwitchConfig{}
	if compareTo, ok := configMap["compareTo"].(string); ok {
		cfg.CompareTo = compareTo
	}
	if fields, ok := configMap["fields"].(map[string]interface{}); ok {
		cfg.Fields = fields
	}
	if def, ok := configMap["default"].(string); ok {
		cfg.Default = def
	}
	return cfg
}

// 產生 switch 欄位
func generateSwitchField(fieldName string, switchDef []interface{}, parentFields []PacketField, parentStruct string) *PacketField {
	cfg := parseSwitchConfig(switchDef)
	if cfg == nil {
		return generateFallbackSwitchField(fieldName, nil)
	}

	compareField, bitFlag, fromParent, ownerStruct := parseSwitchCompareTo(cfg.CompareTo, parentFields, parentStruct)
	if compareField == "" {
		return generateFallbackSwitchField(fieldName, cfg)
	}
	var compareFieldType string
	var compareFieldIsPointer bool
	var compareFieldIsMapper bool
	if pf := findStructField(ownerStruct, parentStruct, parentFields, compareField); pf != nil {
		compareFieldIsPointer = strings.HasPrefix(pf.GoType, "*")
		compareFieldType = strings.TrimPrefix(pf.GoType, "*")
		compareFieldIsMapper = pf.IsMapper
	}
	// 單一條件 + default void → 可視為 optional
	if len(cfg.Fields) == 1 && cfg.Default == "void" {
		res := generateOptionalSwitchField(fieldName, cfg, compareField, compareFieldType, bitFlag, fromParent, compareFieldIsMapper)
		res.NeedsParent = fromParent
		return res
	}

	// 其他情況視為 union
	res := generateUnionSwitchField(fieldName, cfg, compareField, compareFieldType, compareFieldIsPointer, fromParent, compareFieldIsMapper)
	if fromParent {
		res.NeedsParent = true
	}
	return res
}

// 依 compareTo 尋找父欄位
func findParentField(parentFields []PacketField, name string) *PacketField {
	for i := range parentFields {
		if parentFields[i].Name == name {
			return &parentFields[i]
		}
	}
	return nil
}

func parentFieldExists(parentFields []PacketField, name string) bool {
	for _, f := range parentFields {
		if f.Name == name {
			return true
		}
	}
	return false
}

func findStructField(structName, currentStruct string, currentFields []PacketField, name string) *PacketField {
	if structName == currentStruct {
		return findParentField(currentFields, name)
	}
	if fields, ok := structFields[structName]; ok {
		for i := range fields {
			if fields[i].Name == name {
				return &fields[i]
			}
		}
	}
	return nil
}

// 解析 compareTo（支援 "../action/add_player" 或 "flags/0"）
func parseSwitchCompareTo(compareTo string, parentFields []PacketField, currentStruct string) (fieldName string, bitFlag int, fromParent bool, ownerStruct string) {
	bitFlag = -1

	parts := strings.Split(compareTo, "/")
	var tokens []string
	upLevels := 0
	for _, p := range parts {
		if p == "" {
			continue
		}
		if p == ".." {
			upLevels++
			continue
		}
		tokens = append(tokens, p)
	}
	if len(tokens) == 0 {
		return "", -1, false, ""
	}

	ownerStruct = currentStruct
	for i := 0; i < upLevels; i++ {
		if parent, ok := structParent[ownerStruct]; ok && parent != "" {
			ownerStruct = parent
		} else {
			ownerStruct = ""
			break
		}
	}
	fromParent = upLevels > 0
	fieldName = toPascalCase(tokens[0])

	if ownerStruct == "" {
		return "", -1, fromParent, ownerStruct
	}

	// 確認欄位存在
	if findStructField(ownerStruct, currentStruct, parentFields, fieldName) == nil && !fromParent {
		return "", -1, fromParent, ownerStruct
	}

	// 若有第二段，可能是位序或旗標名稱
	if len(tokens) > 1 {
		if pos, err := strconv.Atoi(tokens[1]); err == nil && pos >= 0 {
			return fieldName, pos, fromParent, ownerStruct
		}
		if m, ok := structFlagMaps[ownerStruct]; ok {
			if flags, ok := m[fieldName]; ok {
				if idx, ok := flags[tokens[1]]; ok {
					return fieldName, idx, fromParent, ownerStruct
				}
			}
		}
	}

	return fieldName, bitFlag, fromParent, ownerStruct
}

// 生成 optional switch
func generateOptionalSwitchField(fieldName string, cfg *SwitchConfig, compareField, compareFieldType string, bitFlag int, fromParent bool, compareFieldIsMapper bool) *PacketField {
	var innerType string
	var compareVal interface{}
	var innerDef interface{}
	for k, v := range cfg.Fields {
		switch t := v.(type) {
		case string:
			innerType = t
			compareVal = k
			innerDef = v
		case []interface{}:
			if len(t) > 0 {
				if tn, ok := t[0].(string); ok {
					innerType = tn
					compareVal = k
					innerDef = v
				}
			}
		}
		if innerType != "" {
			break
		}
	}
	if innerType == "" || compareField == "" {
		return generateFallbackSwitchField(fieldName, cfg)
	}

	field := &PacketField{
		Name:        toPascalCase(fieldName),
		Optional:    true,
		NeedsParent: fromParent,
		Comment:     fmt.Sprintf("// Optional，當 %s 符合條件時出現", compareField),
	}

	// 特殊處理：switch → array 型別
	if arr, ok := innerDef.([]interface{}); ok && len(arr) > 1 {
		if arrName, ok2 := arr[0].(string); ok2 && arrName == "array" {
			if arrDef, ok3 := arr[1].(map[string]interface{}); ok3 {
				countType, _ := arrDef["countType"].(string)
				elemStr, elemIsStr := arrDef["type"].(string)
				if elemIsStr && countType != "" {
					elemGo := mapType(elemStr)
					field.GoType = "*[]" + elemGo
					compareLiteral := buildCompareLiteral(compareVal, compareFieldType, fromParent && isNumericType(compareFieldType))
					if compareLiteral == "" {
						return generateFallbackSwitchField(fieldName, cfg)
					}
					field.ReadCode = generateDirectOptionalArrayRead(field.Name, compareField, compareLiteral, elemStr, countType, fromParent)
					field.WriteCode = generateDirectOptionalArrayWrite(field.Name, elemStr, countType)
					return field
				}
			}
		}
	}

	field.GoType = "*" + mapType(innerType)

	preferNumericBool := fromParent && isNumericType(compareFieldType)
	_ = compareFieldIsMapper
	compareLiteral := buildCompareLiteral(compareVal, compareFieldType, preferNumericBool)
	if compareLiteral == "" {
		return generateFallbackSwitchField(fieldName, cfg)
	}
	if bitFlag >= 0 {
		field.ReadCode = generateBitFlagOptionalRead(field.Name, compareField, bitFlag, innerType, fromParent)
		field.WriteCode = generateBitFlagOptionalWrite(field.Name, compareField, bitFlag, innerType, fromParent)
	} else {
		field.ReadCode = generateDirectOptionalRead(field.Name, compareField, compareLiteral, innerType, fromParent)
		field.WriteCode = generateDirectOptionalWrite(field.Name, innerType, fromParent)
	}
	return field
}

// 基於位元旗標的 optional 讀取
func generateBitFlagOptionalRead(fieldName, flagField string, bitPos int, innerType string, fromParent bool) []string {
	code := []string{
		fmt.Sprintf("// 旗標存在才讀取 %s", fieldName),
	}
	if fromParent {
		code = append(code, fmt.Sprintf("if parent != nil && parent.%s & (1 << %d) != 0 {", flagField, bitPos))
	} else {
		code = append(code, fmt.Sprintf("if p.%s & (1 << %d) != 0 {", flagField, bitPos))
	}
	code = append(code, fmt.Sprintf("	var val %s", mapType(innerType)))

	code = append(code, generateValueReadLines(innerType, "val")...)
	code = append(code,
		fmt.Sprintf("	p.%s = &val", fieldName),
		"}",
	)
	return code
}

// 基於位元旗標的 optional 寫入
func generateBitFlagOptionalWrite(fieldName, flagField string, bitPos int, innerType string, fromParent bool) []string {
	_ = fromParent
	code := []string{
		fmt.Sprintf("// 若欄位存在則寫入 %s", fieldName),
		fmt.Sprintf("if p.%s != nil {", fieldName),
	}
	code = append(code, generateValueWriteLines(innerType, "*p."+fieldName)...)
	code = append(code,
		"	if err != nil { return n, err }",
		"}",
	)
	return code
}

// 基於直接欄位比較的 optional 讀取
func generateDirectOptionalRead(fieldName, compareField, compareLiteral, innerType string, fromParent bool) []string {
	code := []string{
		fmt.Sprintf("// 當 %s == %s 時讀取 %s", compareField, compareLiteral, fieldName),
	}
	if fromParent {
		code = append(code, fmt.Sprintf("if parent != nil && parent.%s == %s {", compareField, compareLiteral))
	} else {
		code = append(code, fmt.Sprintf("if p.%s == %s {", compareField, compareLiteral))
	}
	code = append(code, fmt.Sprintf("	var val %s", mapType(innerType)))
	code = append(code, generateValueReadLines(innerType, "val")...)
	code = append(code,
		fmt.Sprintf("	p.%s = &val", fieldName),
		"}",
	)
	return code
}

// 基於直接欄位比較的 optional 寫入
func generateDirectOptionalWrite(fieldName, innerType string, fromParent bool) []string {
	_ = fromParent
	code := []string{
		fmt.Sprintf("if p.%s != nil {", fieldName),
	}
	code = append(code, generateValueWriteLines(innerType, "*p."+fieldName)...)
	code = append(code,
		"	if err != nil { return n, err }",
		"}",
	)
	return code
}

// 基於直接欄位比較的 optional「array」讀取
func generateDirectOptionalArrayRead(fieldName, compareField, compareLiteral, elemType, countType string, fromParent bool) []string {
	code := []string{
		fmt.Sprintf("// 當 %s == %s 時讀取 %s (array)", compareField, compareLiteral, fieldName),
	}
	cond := fmt.Sprintf("if p.%s == %s {", compareField, compareLiteral)
	if fromParent {
		cond = fmt.Sprintf("if parent != nil && parent.%s == %s {", compareField, compareLiteral)
	}
	code = append(code, cond)
	code = append(code,
		fmt.Sprintf("	var cnt pk.%s", mapTypeToPkType(countType)),
		"	temp, err = cnt.ReadFrom(r)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"	arr := make([]"+mapType(elemType)+", cnt)",
		"	for i := 0; i < int(cnt); i++ {",
	)
	readLines := generateValueReadLines(elemType, "v")
	// 確保有變數宣告
	if !strings.Contains(strings.Join(readLines, "\n"), "var v ") {
		readLines = append([]string{"\t\tvar v " + mapType(elemType)}, readLines...)
	}
	for _, l := range readLines {
		code = append(code, "		"+strings.TrimLeft(l, "\t"))
	}
	code = append(code,
		"		arr[i] = v",
		"	}",
		fmt.Sprintf("	p.%s = &arr", fieldName),
		"}",
	)
	return code
}

// 基於直接欄位比較的 optional「array」寫入
func generateDirectOptionalArrayWrite(fieldName, elemType, countType string) []string {
	code := []string{
		fmt.Sprintf("if p.%s != nil {", fieldName),
		fmt.Sprintf("	temp, err = pk.%s(len(*p.%s)).WriteTo(w)", mapTypeToPkType(countType), fieldName),
		"	n += temp",
		"	if err != nil { return n, err }",
		"	for i := range *p." + fieldName + " {",
	}
	writeLines := generateValueWriteLines(elemType, "(*p."+fieldName+")[i]")
	for _, l := range writeLines {
		code = append(code, "		"+strings.TrimLeft(l, "\t"))
	}
	code = append(code,
		"		if err != nil { return n, err }",
		"	}",
		"}",
	)
	return code
}

// 產生 option[array] 的讀寫片段，readLines/writeLines 為單元素讀寫
func generateOptionalArrayRW(field *PacketField, countType string, readLines, writeLines []string) {
	elemType := strings.TrimPrefix(field.GoType, "*[]")
	needsDecl := true
	for _, line := range readLines {
		if strings.Contains(line, "var v ") && strings.Contains(line, elemType) {
			needsDecl = false
			break
		}
	}
	if needsDecl {
		readLines = append([]string{"\t\tvar v " + elemType}, readLines...)
	}
	field.ReadCode = []string{
		fmt.Sprintf("var has%s pk.Boolean", field.Name),
		fmt.Sprintf("temp, err = has%s.ReadFrom(r)", field.Name),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("if has%s {", field.Name),
		fmt.Sprintf("	var cnt pk.%s", mapTypeToPkType(countType)),
		"	temp, err = cnt.ReadFrom(r)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"	arr := make([]" + strings.TrimPrefix(field.GoType, "*[]") + ", cnt)",
		"	for i := 0; i < int(cnt); i++ {",
	}
	field.ReadCode = append(field.ReadCode, readLines...)
	field.ReadCode = append(field.ReadCode,
		"		arr[i] = v",
		"	}",
		fmt.Sprintf("	p.%s = &arr", field.Name),
		"}",
	)

	field.WriteCode = []string{
		fmt.Sprintf("if p.%s != nil {", field.Name),
		"	temp, err = pk.Boolean(true).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"	temp, err = pk." + mapTypeToPkType(countType) + "(len(*p." + field.Name + ")).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"	for i := range *p." + field.Name + " {",
		"		_ = i",
	}
	field.WriteCode = append(field.WriteCode, writeLines...)
	field.WriteCode = append(field.WriteCode,
		"	}",
		"} else {",
		"	temp, err = pk.Boolean(false).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	)
}

// 生成 union switch 欄位
func generateUnionSwitchField(fieldName string, cfg *SwitchConfig, compareField, compareFieldType string, compareFieldIsPointer bool, fromParent bool, compareFieldIsMapper bool) *PacketField {
	comment := fmt.Sprintf("// Switch 基於 %s：\n", compareField)
	for v, t := range cfg.Fields {
		comment += fmt.Sprintf("//   %v -> %v\n", v, t)
	}
	if cfg.Default != "" {
		comment += fmt.Sprintf("//   default -> %s", cfg.Default)
	}

	return &PacketField{
		Name:        toPascalCase(fieldName),
		GoType:      "interface{}",
		NeedsParent: fromParent,
		Comment:     comment,
		ReadCode:    generateUnionReadCode(fieldName, cfg, compareField, compareFieldType, compareFieldIsPointer, fromParent, compareFieldIsMapper),
		WriteCode:   generateUnionWriteCode(fieldName, cfg),
	}
}

func generateUnionReadCode(fieldName string, cfg *SwitchConfig, compareField, compareFieldType string, compareFieldIsPointer bool, fromParent bool, compareFieldIsMapper bool) []string {
	targetField := toPascalCase(fieldName)

	// 如果 compareField 是指针类型，需要解引用
	switchExpr := fmt.Sprintf("p.%s", compareField)
	if compareFieldIsPointer {
		switchExpr = fmt.Sprintf("*p.%s", compareField)
	}
	// 若來自父層（../），以外部變數名稱表示
	if fromParent {
		switchExpr = "parent." + compareField
	}

	preferNumericBool := fromParent && isNumericType(compareFieldType)
	_ = compareFieldIsMapper

	code := []string{
		fmt.Sprintf("switch %s {", switchExpr),
	}

	for rawKey, rawType := range cfg.Fields {
		typeStr, ok := rawType.(string)
		if !ok {
			continue
		}
		valueLiteral := buildCompareLiteral(rawKey, compareFieldType, preferNumericBool)
		if valueLiteral == "" {
			continue
		}

		code = append(code,
			fmt.Sprintf("case %s:", valueLiteral),
			fmt.Sprintf("	var val %s", mapType(typeStr)),
		)
		code = append(code, generateValueReadLines(typeStr, "val")...)
		code = append(code, fmt.Sprintf("	p.%s = val", targetField))
	}

	if cfg.Default != "" && cfg.Default != "void" {
		code = append(code,
			"default:",
			fmt.Sprintf("	var val %s", mapType(cfg.Default)),
		)
		code = append(code, generateValueReadLines(cfg.Default, "val")...)
		code = append(code, fmt.Sprintf("	p.%s = val", targetField))
	} else {
		code = append(code,
			"default:",
			"	// 無對應負載",
		)
	}

	code = append(code, "}")
	return code
}

func generateUnionWriteCode(fieldName string, cfg *SwitchConfig) []string {
	targetField := toPascalCase(fieldName)
	code := []string{
		fmt.Sprintf("switch v := p.%s.(type) {", targetField),
	}

	seen := map[string]bool{}
	for _, rawType := range cfg.Fields {
		typeStr, ok := rawType.(string)
		if !ok {
			continue
		}
		goType := mapType(typeStr)
		if seen[goType] {
			continue
		}
		seen[goType] = true

		code = append(code, fmt.Sprintf("case %s:", goType))
		code = append(code, generateValueWriteLines(typeStr, "v")...)
		code = append(code, "	if err != nil { return n, err }")
	}

	if cfg.Default != "" && cfg.Default != "void" {
		goType := mapType(cfg.Default)
		if !seen[goType] {
			code = append(code, fmt.Sprintf("case %s:", goType))
			code = append(code, generateValueWriteLines(cfg.Default, "v")...)
			code = append(code, "	if err != nil { return n, err }")
		}
	}

	code = append(code,
		"default:",
		fmt.Sprintf("	return n, fmt.Errorf(\"unsupported switch type for %s: %%T\", v)", targetField),
		"}",
	)
	return code
}

func isNumericType(t string) bool {
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}

// 建立比較值字面
func buildCompareLiteral(v interface{}, compareFieldType string, preferNumericBool bool) string {
	switch val := v.(type) {
	case bool:
		if preferNumericBool {
			if val {
				return "1"
			}
			return "0"
		}
		return fmt.Sprintf("%t", val)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%f", val)
	case string:
		if compareFieldType == "string" || compareFieldType == "interface{}" {
			// string 类型或 interface{} (mapper) 类型，需要加引号
			return fmt.Sprintf("%q", val)
		}
		// 非字串就直接回傳原字串（假設為數值/枚舉）
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// 產生針對指定型別的讀取片段，賦值給 target 變數
func generateValueReadLines(typeName, target string) []string {
	switch typeName {
	case "string", "pstring", "Key", "CriterionIdentifier", "registryEntryHolder":
		return []string{
			"	var elem pk.String",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	" + target + " = string(elem)",
		}
	case "buffer":
		return []string{
			"	temp, err = (*pk.ByteArray)(&" + target + ").ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		}
	case "restBuffer", "topBitSetTerminatedArray":
		return []string{
			"	temp, err = (*pk.PluginMessageData)(&" + target + ").ReadFrom(r)",
			"	n += temp",
			"	if err != nil && err != io.EOF { return n, err }",
		}
	case "registryEntryHolderSet":
		return []string{
			"	var count pk.VarInt",
			"	temp, err = count.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	if count < 0 { return n, fmt.Errorf(\"negative registryEntryHolderSet length\") }",
			"	" + target + " = make([]string, count)",
			"	for i := int32(0); i < int32(count); i++ {",
			"		var s pk.String",
			"		temp, err = s.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		" + target + "[i] = string(s)",
			"	}",
		}
	case "ItemFireworkExplosion", "ItemSoundHolder":
		return []string{
			"	temp, err = (*pk.NBTField)(&" + target + ").ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		}
	case "varint", "varlong":
		return []string{
			"	var elem pk.VarInt",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	" + target + " = int32(elem)",
		}
	case "u8":
		return []string{
			"	var elem pk.UnsignedByte",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	" + target + " = uint8(elem)",
		}
	case "bool":
		return []string{
			"	var elem pk.Boolean",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	" + target + " = bool(elem)",
		}
	case "slot", "Slot":
		return []string{
			"	temp, err = (*slot.Slot)(&" + target + ").ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		}
	case "entityMetadata", "entityMetadataLoop":
		return []string{
			"	temp, err = (*metadata.EntityMetadata)(&" + target + ").ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		}
	case "vec3f64":
		return []string{
			"	for i := 0; i < 3; i++ {",
			"		var d pk.Double",
			"		temp, err = d.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		" + target + "[i] = float64(d)",
			"	}",
		}
	case "vec3f":
		return []string{
			"	for i := 0; i < 3; i++ {",
			"		var f pk.Float",
			"		temp, err = f.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		" + target + "[i] = float32(f)",
			"	}",
		}
	case "vec3i":
		return []string{
			"	for i := 0; i < 3; i++ {",
			"		var v pk.VarInt",
			"		temp, err = v.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		" + target + "[i] = int32(v)",
			"	}",
		}
	case "game_profile", "chat_session":
		return []string{
			"	temp, err = (*pk.NBTField)(&" + target + ").ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		}
	case "IDSet":
		return []string{
			"	var cnt pk.VarInt",
			"	temp, err = cnt.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	if cnt < 0 { return n, fmt.Errorf(\"negative IDSet length\") }",
			"	" + target + " = make([]int32, cnt)",
			"	for i := int32(0); i < int32(cnt); i++ {",
			"		var vi pk.VarInt",
			"		temp, err = vi.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		" + target + "[i] = int32(vi)",
			"	}",
		}
	case "option":
		return []string{
			"	var present pk.Boolean",
			"	temp, err = present.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	if present {",
			"		// option payload omitted (unknown inner type)",
			"		var dummy interface{}",
			"		" + target + " = dummy",
			"	} else {",
			"		" + target + " = nil",
			"	}",
		}
	case "void":
		return []string{}
	default:
		// 檢查是否為 interface{} 類型 (複雜類型的 placeholder)
		goType := mapType(typeName)
		if goType == "interface{}" {
			return []string{
				"	// TODO: Read " + typeName + " type",
			}
		}
		return []string{
			"	temp, err = (*pk." + mapTypeToPkType(typeName) + ")(&" + target + ").ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		}
	}
}

// 產生針對指定型別的寫入片段，資料來自 valueExpr
func generateValueWriteLines(typeName, valueExpr string) []string {
	switch typeName {
	case "string", "pstring", "Key", "CriterionIdentifier", "registryEntryHolder":
		return []string{
			"	temp, err = pk.String(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "buffer":
		return []string{
			"	temp, err = pk.ByteArray(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "restBuffer", "topBitSetTerminatedArray":
		return []string{
			"	temp, err = pk.PluginMessageData(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "registryEntryHolderSet":
		return []string{
			"	temp, err = pk.VarInt(len(" + valueExpr + ")).WriteTo(w)",
			"	n += temp",
			"	for i := range " + valueExpr + " {",
			"		temp, err = pk.String(" + valueExpr + "[i]).WriteTo(w)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		}
	case "ItemFireworkExplosion", "ItemSoundHolder":
		return []string{
			"	temp, err = pk.NBTField(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "varint", "varlong":
		return []string{
			"	temp, err = pk.VarInt(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "u8":
		return []string{
			"	temp, err = pk.UnsignedByte(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "bool":
		return []string{
			"	temp, err = pk.Boolean(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "slot", "Slot":
		return []string{
			"	temp, err = (" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "entityMetadata", "entityMetadataLoop":
		return []string{
			"	temp, err = (" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "vec3f64":
		return []string{
			"	for i := 0; i < 3; i++ {",
			"		temp, err = pk.Double(" + valueExpr + "[i]).WriteTo(w)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		}
	case "vec3f":
		return []string{
			"	for i := 0; i < 3; i++ {",
			"		temp, err = pk.Float(" + valueExpr + "[i]).WriteTo(w)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		}
	case "vec3i":
		return []string{
			"	for i := 0; i < 3; i++ {",
			"		temp, err = pk.VarInt(" + valueExpr + "[i]).WriteTo(w)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		}
	case "game_profile", "chat_session":
		return []string{
			"	temp, err = pk.NBTField(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	case "IDSet":
		return []string{
			"	temp, err = pk.VarInt(len(" + valueExpr + ")).WriteTo(w)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	for i := range " + valueExpr + " {",
			"		temp, err = pk.VarInt(" + valueExpr + "[i]).WriteTo(w)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		}
	case "option":
		return []string{
			"	if " + valueExpr + " != nil {",
			"		temp, err = pk.Boolean(true).WriteTo(w)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		// option payload omitted (unknown inner type)",
			"	} else {",
			"		temp, err = pk.Boolean(false).WriteTo(w)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		}
	case "void":
		return []string{}
	default:
		// 檢查是否為 interface{} 類型 (複雜類型的 placeholder)
		goType := mapType(typeName)
		if goType == "interface{}" {
			return []string{
				"	// TODO: Write " + typeName + " type",
			}
		}
		return []string{
			"	temp, err = pk." + mapTypeToPkType(typeName) + "(" + valueExpr + ").WriteTo(w)",
			"	n += temp",
		}
	}
}

// 回退 switch 欄位（無法解析時）
func generateFallbackSwitchField(fieldName string, cfg *SwitchConfig) *PacketField {
	return &PacketField{
		Name:      toPascalCase(fieldName),
		GoType:    "interface{}",
		Comment:   "// TODO: Switch type - conditional field based on other field value",
		ReadCode:  []string{"// TODO: Implement switch field read"},
		WriteCode: []string{"// TODO: Implement switch field write"},
	}
}

// 生成寫入代碼
func generateWriteCode(fieldName, typeName string, optional bool) []string {
	if mapType(typeName) == "interface{}" {
		return []string{fmt.Sprintf("// TODO: Write %s (unsupported type %s)", fieldName, typeName)}
	}
	var code []string

	switch typeName {
	case "varint", "varlong":
		code = []string{
			fmt.Sprintf("temp, err = pk.VarInt(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i8":
		code = []string{
			fmt.Sprintf("temp, err = pk.Byte(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "u8":
		code = []string{
			fmt.Sprintf("temp, err = pk.UnsignedByte(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i16", "u16":
		code = []string{
			fmt.Sprintf("temp, err = pk.%s(p.%s).WriteTo(w)", mapTypeToPkType(typeName), fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i32", "u32":
		if typeName == "u32" {
			code = []string{
				fmt.Sprintf("temp, err = pk.Int(int32(p.%s)).WriteTo(w)", fieldName),
				"n += temp",
				"if err != nil { return n, err }",
			}
		} else {
			code = []string{
				fmt.Sprintf("temp, err = pk.Int(p.%s).WriteTo(w)", fieldName),
				"n += temp",
				"if err != nil { return n, err }",
			}
		}
	case "i64", "u64":
		if typeName == "u64" {
			code = []string{
				fmt.Sprintf("temp, err = pk.Long(int64(p.%s)).WriteTo(w)", fieldName),
				"n += temp",
				"if err != nil { return n, err }",
			}
		} else {
			code = []string{
				fmt.Sprintf("temp, err = pk.Long(p.%s).WriteTo(w)", fieldName),
				"n += temp",
				"if err != nil { return n, err }",
			}
		}
	case "bitflags":
		code = []string{
			fmt.Sprintf("temp, err = pk.UnsignedByte(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "f32":
		code = []string{
			fmt.Sprintf("temp, err = pk.Float(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "f64":
		code = []string{
			fmt.Sprintf("temp, err = pk.Double(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "bool":
		code = []string{
			fmt.Sprintf("temp, err = pk.Boolean(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "string", "pstring", "Key", "CriterionIdentifier", "registryEntryHolder":
		code = []string{
			fmt.Sprintf("temp, err = pk.String(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "buffer":
		code = []string{
			fmt.Sprintf("temp, err = pk.ByteArray(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "restBuffer", "topBitSetTerminatedArray":
		code = []string{
			fmt.Sprintf("temp, err = pk.PluginMessageData(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "registryEntryHolderSet":
		code = []string{
			"temp, err = pk.VarInt(len(p." + fieldName + ")).WriteTo(w)",
			"n += temp",
			"if err != nil { return n, err }",
			"for i := range p." + fieldName + " {",
			"	temp, err = pk.String(p." + fieldName + "[i]).WriteTo(w)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"}",
		}
	case "ItemFireworkExplosion", "ItemSoundHolder":
		code = []string{
			fmt.Sprintf("temp, err = pk.NBTField(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "UUID", "position", "nbt", "anonymousNbt", "anonOptionalNbt", "optionalNbt", "component", "textComponent", "game_profile", "chat_session":
		code = []string{
			fmt.Sprintf("temp, err = p.%s.WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "slot", "Slot":
		code = []string{
			fmt.Sprintf("temp, err = p.%s.WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "entityMetadata", "entityMetadataLoop":
		code = []string{
			fmt.Sprintf("temp, err = p.%s.WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "vec3f64":
		code = []string{
			fmt.Sprintf("for i := 0; i < 3; i++ {"),
			fmt.Sprintf("	temp, err = pk.Double(p.%s[i]).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
			"}",
		}
	default:
		code = []string{fmt.Sprintf("// TODO: Write %s (%s)", fieldName, typeName)}
	}

	return code
}

// 生成 Optional 讀取代碼
func generateOptionalReadCode(fieldName, innerType string) []string {
	goType := mapType(innerType)
	code := []string{
		fmt.Sprintf("var has%s pk.Boolean", fieldName),
		fmt.Sprintf("temp, err = has%s.ReadFrom(r)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("if has%s {", fieldName),
		fmt.Sprintf("	var val %s", goType),
	}

	switch innerType {
	case "varint", "varlong":
		code = append(code,
			"	var elem pk.VarInt",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			fmt.Sprintf("	val = %s(elem)", goType),
		)
	case "string", "pstring", "Key", "CriterionIdentifier", "registryEntryHolder":
		code = append(code,
			"	var elem pk.String",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	val = string(elem)",
		)
	case "buffer":
		code = append(code,
			"	temp, err = (*pk.ByteArray)(&val).ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "restBuffer":
		code = append(code,
			"	temp, err = (*pk.PluginMessageData)(&val).ReadFrom(r)",
			"	n += temp",
			"	if err != nil && err != io.EOF { return n, err }",
		)
	case "u8":
		code = append(code,
			"	var elem pk.UnsignedByte",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	val = uint8(elem)",
		)
	case "slot", "Slot":
		code = append(code,
			"	temp, err = (*slot.Slot)(&val).ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "HashedSlot":
		code = append(code,
			"	temp, err = (&val).ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "entityMetadata", "entityMetadataLoop":
		code = append(code,
			"	temp, err = (*metadata.EntityMetadata)(&val).ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "vec3f64":
		code = append(code,
			"	for i := 0; i < 3; i++ {",
			"		var d pk.Double",
			"		temp, err = d.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		val[i] = float64(d)",
			"	}",
		)
	case "vec3f":
		code = append(code,
			"	for i := 0; i < 3; i++ {",
			"		var f pk.Float",
			"		temp, err = f.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		val[i] = float32(f)",
			"	}",
		)
	case "vec3i":
		code = append(code,
			"	for i := 0; i < 3; i++ {",
			"		var v pk.VarInt",
			"		temp, err = v.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			"		val[i] = int32(v)",
			"	}",
		)
	default:
		code = append(code,
			"	temp, err = (*pk."+mapTypeToPkType(innerType)+")(&val).ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	}

	code = append(code,
		fmt.Sprintf("	p.%s = &val", fieldName),
		"}",
	)
	return code
}

// 生成 Optional 寫入代碼
func generateOptionalWriteCode(fieldName, innerType string) []string {
	code := []string{
		fmt.Sprintf("if p.%s != nil {", fieldName),
		"	temp, err = pk.Boolean(true).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
	}

	switch innerType {
	case "varint", "varlong":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.VarInt(*p.%s).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "string", "pstring", "Key", "CriterionIdentifier", "registryEntryHolder":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.String(*p.%s).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "buffer":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.ByteArray(*p.%s).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "restBuffer":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.PluginMessageData(*p.%s).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "u8":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.UnsignedByte(*p.%s).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "slot", "Slot":
		code = append(code,
			fmt.Sprintf("	temp, err = p.%s.WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "HashedSlot":
		code = append(code,
			fmt.Sprintf("	temp, err = p.%s.WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "entityMetadata", "entityMetadataLoop":
		code = append(code,
			fmt.Sprintf("	temp, err = p.%s.WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "vec3f64":
		code = append(code,
			"	for i := 0; i < 3; i++ {",
			fmt.Sprintf("		temp, err = pk.Double(p.%s[i]).WriteTo(w)", fieldName),
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		)
	case "vec3f":
		code = append(code,
			"	for i := 0; i < 3; i++ {",
			fmt.Sprintf("		temp, err = pk.Float((*p.%s)[i]).WriteTo(w)", fieldName),
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		)
	case "vec3i":
		code = append(code,
			"	for i := 0; i < 3; i++ {",
			fmt.Sprintf("		temp, err = pk.VarInt((*p.%s)[i]).WriteTo(w)", fieldName),
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		)
	default:
		code = append(code,
			fmt.Sprintf("	temp, err = pk.%s(*p.%s).WriteTo(w)", mapTypeToPkType(innerType), fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	}

	code = append(code,
		"} else {",
		"	temp, err = pk.Boolean(false).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	)
	return code
}

// 生成 Array 讀取代碼
func generateArrayReadCode(fieldName, arrayType, countType string) []string {
	countVar := safeIdent(strings.ToLower(fieldName[:1]) + fieldName[1:] + "Count")
	goType := mapType(arrayType)

	code := []string{
		fmt.Sprintf("var %s pk.VarInt", countVar),
		fmt.Sprintf("temp, err = %s.ReadFrom(r)", countVar),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("p.%s = make([]%s, %s)", fieldName, goType, countVar),
		fmt.Sprintf("for i := 0; i < int(%s); i++ {", countVar),
	}

	switch arrayType {
	case "varint", "varlong":
		code = append(code,
			"	var elem pk.VarInt",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			fmt.Sprintf("	p.%s[i] = %s(elem)", fieldName, goType),
		)
	case "string", "pstring":
		code = append(code,
			"	var elem pk.String",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			fmt.Sprintf("	p.%s[i] = string(elem)", fieldName),
		)
	case "slot", "Slot":
		code = append(code,
			fmt.Sprintf("	temp, err = (*slot.Slot)(&p.%s[i]).ReadFrom(r)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "entityMetadata", "entityMetadataLoop":
		code = append(code,
			fmt.Sprintf("	temp, err = (*metadata.EntityMetadata)(&p.%s[i]).ReadFrom(r)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "vec3f64":
		code = append(code,
			"	for j := 0; j < 3; j++ {",
			"		var d pk.Double",
			"		temp, err = d.ReadFrom(r)",
			"		n += temp",
			"		if err != nil { return n, err }",
			fmt.Sprintf("		p.%s[i][j] = float64(d)", fieldName),
			"	}",
		)
	default:
		// 对于基础类型，使用 pk.Type 读取
		pkType := mapTypeToPkType(arrayType)
		if pkType != arrayType {
			// 已知的基础类型
			code = append(code,
				fmt.Sprintf("	var elem pk.%s", pkType),
				"	temp, err = elem.ReadFrom(r)",
				"	n += temp",
				"	if err != nil { return n, err }",
				fmt.Sprintf("	p.%s[i] = %s(elem)", fieldName, goType),
			)
		} else {
			// 复杂类型，假设有 ReadFrom 方法
			code = append(code,
				fmt.Sprintf("	temp, err = p.%s[i].ReadFrom(r)", fieldName),
				"	n += temp",
				"	if err != nil { return n, err }",
			)
		}
	}

	code = append(code, "}")
	return code
}

// 生成 Array 寫入代碼
func generateArrayWriteCode(fieldName, arrayType, countType string) []string {
	code := []string{
		fmt.Sprintf("temp, err = pk.VarInt(len(p.%s)).WriteTo(w)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("for i := range p.%s {", fieldName),
	}

	switch arrayType {
	case "varint", "varlong":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.VarInt(p.%s[i]).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "string", "pstring":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.String(p.%s[i]).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "slot", "Slot":
		code = append(code,
			fmt.Sprintf("	temp, err = p.%s[i].WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "entityMetadata", "entityMetadataLoop":
		code = append(code,
			fmt.Sprintf("	temp, err = p.%s[i].WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	case "vec3f64":
		code = append(code,
			"	for j := 0; j < 3; j++ {",
			fmt.Sprintf("		temp, err = pk.Double(p.%s[i][j]).WriteTo(w)", fieldName),
			"		n += temp",
			"		if err != nil { return n, err }",
			"	}",
		)
	default:
		// 对于基础类型，使用 pk.Type 包装
		pkType := mapTypeToPkType(arrayType)
		if pkType != arrayType {
			// 已知的基础类型
			code = append(code,
				fmt.Sprintf("	temp, err = pk.%s(p.%s[i]).WriteTo(w)", pkType, fieldName),
				"	n += temp",
				"	if err != nil { return n, err }",
			)
		} else {
			// 复杂类型，假设有 WriteTo 方法
			code = append(code,
				fmt.Sprintf("	temp, err = p.%s[i].WriteTo(w)", fieldName),
				"	n += temp",
				"	if err != nil { return n, err }",
			)
		}
	}

	code = append(code, "}")
	return code
}

// 命名轉換
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// 收集需要的導入
func (p *PacketDef) collectImports() {
	p.Imports["\"io\""] = true
	p.Imports["\"git.konjactw.dev/patyhank/minego/pkg/protocol/packetid\""] = true

	needSlot := false
	needMetadata := false
	needFmt := false
	needPk := false

	hasPkgCall := func(line, pkg string) bool {
		idx := strings.Index(line, pkg+".")
		if idx == -1 {
			return false
		}
		// 確保是套件呼叫（點後為大寫通常為型別/函式）
		pos := idx + len(pkg) + 1
		if pos >= len(line) {
			return false
		}
		ch := line[pos]
		return ch >= 'A' && ch <= 'Z'
	}

	checkCodeUsage := func(code []string) {
		for _, line := range code {
			if hasPkgCall(line, "fmt") {
				needFmt = true
			}
			if hasPkgCall(line, "pk") {
				needPk = true
			}
			// slot 匹配限定型別/結構使用，避免區域變數同名誤判
			if strings.Contains(line, "slot.Slot") || strings.Contains(line, "slot.HashedSlot") {
				needSlot = true
			}
			if strings.Contains(line, "metadata.") {
				needMetadata = true
			}
		}
	}

	addFieldImports := func(fields []PacketField) {
		for _, f := range fields {
			if strings.Contains(f.GoType, "slot.") {
				needSlot = true
			}
			if strings.Contains(f.GoType, "metadata.") {
				needMetadata = true
			}
			if strings.Contains(f.GoType, "pk.") {
				needPk = true
			}
			checkCodeUsage(f.ReadCode)
			checkCodeUsage(f.WriteCode)
		}
	}

	addFieldImports(p.Fields)
	for _, s := range p.SubStructs {
		addFieldImports(s.Fields)
	}

	// 检查是否使用了 fmt
	for _, f := range p.Fields {
		checkCodeUsage(f.ReadCode)
		checkCodeUsage(f.WriteCode)
		if needFmt {
			break
		}
	}
	if !needFmt {
		for _, s := range p.SubStructs {
			for _, f := range s.Fields {
				checkCodeUsage(f.ReadCode)
				checkCodeUsage(f.WriteCode)
				if needFmt {
					break
				}
			}
			if needFmt {
				break
			}
		}
	}

	if needPk {
		p.Imports["pk \"git.konjactw.dev/falloutBot/go-mc/net/packet\""] = true
	}
	if needSlot {
		p.Imports["\"git.konjactw.dev/patyhank/minego/pkg/protocol/slot\""] = true
	} else {
		delete(p.Imports, "\"git.konjactw.dev/patyhank/minego/pkg/protocol/slot\"")
	}
	if needMetadata {
		p.Imports["\"git.konjactw.dev/patyhank/minego/pkg/protocol/metadata\""] = true
	} else {
		delete(p.Imports, "\"git.konjactw.dev/patyhank/minego/pkg/protocol/metadata\"")
	}
	if needFmt {
		p.Imports["\"fmt\""] = true
	}
}

func (p *PacketDef) buildImportList() {
	p.ImportList = p.ImportList[:0]
	for k := range p.Imports {
		p.ImportList = append(p.ImportList, k)
	}
	sort.Strings(p.ImportList)
}

// 生成代碼
func generatePackets(packets []PacketDef, outputDir, direction string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	packageName := direction
	packetIDType := "Clientbound"
	if direction == "server" {
		packetIDType = "Serverbound"
	}
	// 先產出 base.go（放接口 + 註冊表）
	if err := generateBaseFile(outputDir, packageName, packetIDType); err != nil {
		return err
	}

	// 模板 - 支持子结构体
	tmpl := template.Must(template.New("packet").Funcs(template.FuncMap{
		"join": func(lines []string) string {
			return strings.Join(lines, "\n\t")
		},
		"hasParent":  func(s StructDef) bool { return s.NeedsParent },
		"parentType": func(s StructDef) string { return s.Parent },
	}).Parse(`// Code generated by enhanced-generator v2 from protocol.json; DO NOT EDIT manually.
// To regenerate: go run main_v2.go -protocol <path> -output <dir> -direction {{.Direction}}

package {{.Package}}

import (
{{- range .ImportList }}
	{{.}}
{{- end }}
)
{{range .SubStructs}}
// {{.Name}} is a sub-structure used in the packet.
type {{.Name}} struct {
{{- range .Fields}}
{{if .Comment}}	{{.Comment}}
{{end}}	{{.Name}} {{.GoType}}{{if .MCTag}} {{.MCTag}}{{end}}
{{- end}}
}

{{if $.GenerateCodec}}
// ReadFrom reads the data from the reader.
func (p *{{.Name}}) ReadFrom(r io.Reader) (n int64, err error) {
{{- if hasParent . }}
	// Parent context required; fallback calls ReadFromWithParent with nil (may error if accessed).
	return p.ReadFromWithParent(r, nil)
{{- else }}
{{- if gt (len .Fields) 0 }}
	var temp int64
	_ = temp
{{- end }}
{{range .Fields}}
	{{join .ReadCode}}
{{end}}
	return n, nil
{{- end }}
}

{{if hasParent .}}
// ReadFromWithParent reads the data from the reader with parent context.
func (p *{{.Name}}) ReadFromWithParent(r io.Reader, parent *{{parentType .}}) (n int64, err error) {
{{- if gt (len .Fields) 0 }}
	var temp int64
	_ = temp
{{- end }}
	_ = parent
{{range .Fields}}
	{{join .ReadCode}}
{{end}}
	return n, nil
}
{{end}}

// WriteTo writes the data to the writer.
func (p {{.Name}}) WriteTo(w io.Writer) (n int64, err error) {
{{- if hasParent . }}
	return p.WriteToWithParent(w, nil)
{{- else }}
{{- if gt (len .Fields) 0 }}
	var temp int64
	_ = temp
{{- end }}
{{range .Fields}}
	{{join .WriteCode}}
{{end}}
	return n, nil
{{- end }}
}
{{if hasParent .}}
// WriteToWithParent writes the data with parent context.
func (p {{.Name}}) WriteToWithParent(w io.Writer, parent *{{parentType .}}) (n int64, err error) {
{{- if gt (len .Fields) 0 }}
	var temp int64
	_ = temp
{{- end }}
	_ = parent
{{range .Fields}}
	{{join .WriteCode}}
{{end}}
	return n, nil
}
{{end}}
{{end}}
{{end}}

// {{.StructName}} represents the {{.PacketIDType}} {{.StructName}} packet.
{{if .Comment}}// {{.Comment}}{{end}}
type {{.StructName}} struct {
{{- range .Fields}}
{{if .Comment}}	{{.Comment}}
{{end}}	{{.Name}} {{.GoType}}{{if .MCTag}} {{.MCTag}}{{end}}
{{- end}}
}

// PacketID returns the packet ID for this packet.
func (*{{.StructName}}) PacketID() packetid.{{.PacketIDType}}PacketID {
	return packetid.{{.PacketIDType}}{{.StructName}}
}

{{if .GenerateCodec}}
// ReadFrom reads the packet data from the reader.
func (p *{{.StructName}}) ReadFrom(r io.Reader) (n int64, err error) {
{{- if gt (len .Fields) 0 }}
	var temp int64
	_ = temp
{{- end }}
{{range .Fields}}
	{{join .ReadCode}}
{{end}}
	return n, nil
}

// WriteTo writes the packet data to the writer.
func (p {{.StructName}}) WriteTo(w io.Writer) (n int64, err error) {
{{- if gt (len .Fields) 0 }}
	var temp int64
	_ = temp
{{- end }}
{{range .Fields}}
	{{join .WriteCode}}
{{end}}
	return n, nil
}
{{end}}
{{if .GenerateInit}}
func init() {
	registerPacket(packetid.{{.PacketIDType}}{{.StructName}}, func() {{.PacketIDType}}Packet {
		return &{{.StructName}}{}
	})
}
{{end}}
`))

	// 為每個封包生成文件
	for _, packet := range packets {
		filename := filepath.Join(outputDir, strings.ToLower(packet.Name)+".go")

		f, err := os.Create(filename)
		if err != nil {
			return err
		}

		data := struct {
			Package       string
			StructName    string
			Fields        []PacketField
			PacketIDType  string
			Comment       string
			ImportList    []string
			GenerateCodec bool
			SubStructs    []StructDef
			GenerateInit  bool
			Direction     string
		}{
			Package:       packageName,
			StructName:    packet.StructName,
			Fields:        packet.Fields,
			PacketIDType:  packetIDType,
			ImportList:    packet.ImportList,
			GenerateCodec: *genCodec,
			SubStructs:    packet.SubStructs,
			GenerateInit:  packet.GenerateInit,
			Direction:     direction,
		}

		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			return err
		}
		f.Close()

		if *verbose {
			log.Printf("✅ 生成: %s", filename)
		}
	}

	return nil
}

// generateBaseFile 會在 client/ 或 server/ 目錄下生成 base.go
// 內容就是 ClientboundPacket / ServerboundPacket 介面 + map + registerPacket。
func generateBaseFile(outputDir, packageName, packetIDType string) error {
	filename := filepath.Join(outputDir, "packet.go")

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// 根據方向決定名字
	var (
		packetInterfaceName string
		packetIDTypeName    string
		creatorTypeName     string
		mapName             string
	)

	if packetIDType == "Clientbound" {
		packetInterfaceName = "ClientboundPacket"
		packetIDTypeName = "ClientboundPacketID"
		creatorTypeName = "ClientboundPacketCreator"
		mapName = "ClientboundPackets"
	} else {
		packetInterfaceName = "ServerboundPacket"
		packetIDTypeName = "ServerboundPacketID"
		creatorTypeName = "ServerboundPacketCreator"
		mapName = "ServerboundPackets"
	}

	// 寫入檔案內容
	_, err = fmt.Fprintf(f, `// Code generated by enhanced-generator v2; DO NOT EDIT.

package %s

import (
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
	packetid "%s"
)

// %s 定義所有遊戲階段封包介面（%s）
type %s interface {
	pk.Field
	PacketID() packetid.%s
}

type %s func() %s

// %s 供外部透過 ID 生成封包實例
var %s = make(map[packetid.%s]%s)

func registerPacket(id packetid.%s, creator %s) {
	%s[id] = creator
}
`, packageName, *packetidPkg,
		packetInterfaceName, packetIDType, // 註解用
		packetInterfaceName, packetIDTypeName,
		creatorTypeName, packetInterfaceName,
		mapName, mapName, packetIDTypeName, creatorTypeName,
		packetIDTypeName, creatorTypeName,
		mapName,
	)

	return err
}
