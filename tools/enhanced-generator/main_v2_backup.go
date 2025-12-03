package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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
	Name         string
	Type         string
	GoType       string
	MCTag        string
	Optional     bool
	IsArray      bool
	ArrayType    string
	ArrayCount   string
	Comment      string
	ReadCode     []string
	WriteCode    []string
	NeedsPointer bool
	IsStruct     bool   // 是否是子结构体
	StructName   string // 子结构体名称
}

// StructDef 结构体定义（包括嵌套的子结构体）
type StructDef struct {
	Name   string
	Fields []PacketField
}

// PacketDef 封包定義
type PacketDef struct {
	Name         string
	StructName   string
	Fields       []PacketField
	PacketID     string
	Imports      map[string]bool
	SubStructs   []StructDef // 子结构体
	GenerateInit bool        // 是否生成 init 函数
}

var (
	protocolFile = flag.String("protocol", "", "Path to protocol.json")
	outputDir    = flag.String("output", "", "Output directory")
	direction    = flag.String("direction", "client", "client or server")
	verbose      = flag.Bool("v", false, "Verbose output")
	genCodec     = flag.Bool("codec", true, "Generate ReadFrom/WriteTo methods")
)

// 全局变量：收集所有生成的结构体名称，避免重复
var generatedStructs = make(map[string]bool)
var structCounter = make(map[string]int)

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
	dirName := "Client"
	if *direction == "client" {
		packetTypes = protocol.Play.ToClient.Types
	} else {
		packetTypes = protocol.Play.ToServer.Types
		dirName = "Server"
	}

	if *verbose {
		log.Printf("🔄 解析 %s 封包定義...", dirName)
	}

	// 解析所有封包
	packets := parsePackets(packetTypes, protocol.Types)

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

		// 重置结构体计数器
		generatedStructs = make(map[string]bool)
		structCounter = make(map[string]int)

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

	// 解析字段，收集子结构体
	packet.Fields = parseFields(fields, globalTypes, structName, packet)

	// 收集需要的導入
	packet.collectImports()

	if *verbose {
		log.Printf("  ✓ %s (%d 欄位, %d 子結構)", structName, len(packet.Fields), len(packet.SubStructs))
	}

	return packet
}

func parseFields(fields []interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef) []PacketField {
	var result []PacketField

	for _, f := range fields {
		fieldMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := fieldMap["name"].(string)
		fieldType := fieldMap["type"]

		field := parseFieldType(name, fieldType, globalTypes, parentName, packet)
		if field != nil {
			result = append(result, *field)
		}
	}

	return result
}

func parseFieldType(name string, fieldType interface{}, globalTypes map[string]interface{}, parentName string, packet *PacketDef) *PacketField {
	field := &PacketField{
		Name:      toPascalCase(name),
		ReadCode:  []string{},
		WriteCode: []string{},
	}

	switch t := fieldType.(type) {
	case string:
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
				// Switch 類型 - 使用 interface{}，标注需要手动实现
				field.GoType = "interface{}"
				field.Comment = "// TODO: Switch type - conditional field based on other field value"
				field.ReadCode = []string{"// TODO: Implement switch field read"}
				field.WriteCode = []string{"// TODO: Implement switch field write"}

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
						}
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

							// 生成 optional struct 的读写代码
							field.ReadCode = generateOptionalStructReadCode(field.Name, subStructName)
							field.WriteCode = generateOptionalStructWriteCode(field.Name, subStructName)
						}
					}
				}
			}
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
				// 简单类型数组
				field.ArrayType = elemType
				field.GoType = "[]" + mapType(elemType)
				field.ReadCode = generateArrayReadCode(field.Name, elemType, countType)
				field.WriteCode = generateArrayWriteCode(field.Name, elemType, countType)

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

								// 生成结构体数组的读写代码
								field.ReadCode = generateStructArrayReadCode(field.Name, subStructName, countType)
								field.WriteCode = generateStructArrayWriteCode(field.Name, subStructName, countType)
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

		// 生成嵌套结构体的读写代码
		field.ReadCode = []string{
			fmt.Sprintf("temp, err = p.%s.ReadFrom(r)", field.Name),
			"n += temp",
			"if err != nil { return n, err }",
		}
		field.WriteCode = []string{
			fmt.Sprintf("temp, err = p.%s.WriteTo(w)", field.Name),
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

	subStruct := &StructDef{
		Name:   structName,
		Fields: parseFields(fields, globalTypes, structName, packet),
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
func generateOptionalStructReadCode(fieldName, structName string) []string {
	return []string{
		fmt.Sprintf("var has%s pk.Boolean", fieldName),
		fmt.Sprintf("temp, err = has%s.ReadFrom(r)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("if has%s {", fieldName),
		fmt.Sprintf("	p.%s = &%s{}", fieldName, structName),
		fmt.Sprintf("	temp, err = p.%s.ReadFrom(r)", fieldName),
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成 optional struct 的写入代码
func generateOptionalStructWriteCode(fieldName, structName string) []string {
	return []string{
		fmt.Sprintf("if p.%s != nil {", fieldName),
		"	temp, err = pk.Boolean(true).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		fmt.Sprintf("	temp, err = p.%s.WriteTo(w)", fieldName),
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
func generateStructArrayReadCode(fieldName, structName, countType string) []string {
	countVar := strings.ToLower(fieldName[:1]) + fieldName[1:] + "Count"
	return []string{
		fmt.Sprintf("var %s pk.VarInt", countVar),
		fmt.Sprintf("temp, err = %s.ReadFrom(r)", countVar),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("p.%s = make([]%s, %s)", fieldName, structName, countVar),
		fmt.Sprintf("for i := 0; i < int(%s); i++ {", countVar),
		fmt.Sprintf("	temp, err = p.%s[i].ReadFrom(r)", fieldName),
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成结构体数组的写入代码
func generateStructArrayWriteCode(fieldName, structName, countType string) []string {
	return []string{
		fmt.Sprintf("temp, err = pk.VarInt(len(p.%s)).WriteTo(w)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("for i := range p.%s {", fieldName),
		fmt.Sprintf("	temp, err = p.%s[i].WriteTo(w)", fieldName),
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成嵌套数组的读取代码（array[array[T]]）
func generateNestedArrayReadCode(fieldName, innerType, countType string) []string {
	countVar := strings.ToLower(fieldName[:1]) + fieldName[1:] + "Count"
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
	switch innerType {
	case "string", "pstring":
		return fmt.Sprintf("		var elem pk.String\n		temp, err = elem.ReadFrom(r)\n		p.%s[i][j] = string(elem)", fieldName)
	case "varint", "varlong":
		return fmt.Sprintf("		var elem pk.VarInt\n		temp, err = elem.ReadFrom(r)\n		p.%s[i][j] = int32(elem)", fieldName)
	default:
		return fmt.Sprintf("		temp, err = (*pk.%s)(&p.%s[i][j]).ReadFrom(r)", mapTypeToPk(innerType), fieldName)
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
	switch innerType {
	case "string", "pstring":
		return fmt.Sprintf("		temp, err = pk.String(p.%s[i][j]).WriteTo(w)", fieldName)
	case "varint", "varlong":
		return fmt.Sprintf("		temp, err = pk.VarInt(p.%s[i][j]).WriteTo(w)", fieldName)
	default:
		return fmt.Sprintf("		temp, err = pk.%s(p.%s[i][j]).WriteTo(w)", mapTypeToPk(innerType), fieldName)
	}
}

func mapTypeToPk(t string) string {
	mapping := map[string]string{
		"i8":   "Byte",
		"i16":  "Short",
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
		"varint":             "int32",
		"varlong":            "int64",
		"optvarint":          "*int32",
		"i8":                 "int8",
		"i16":                "int16",
		"i32":                "int32",
		"i64":                "int64",
		"u8":                 "uint8",
		"u16":                "uint16",
		"u32":                "uint32",
		"u64":                "uint64",
		"f32":                "float32",
		"f64":                "float64",
		"bool":               "bool",
		"string":             "string",
		"pstring":            "string",
		"UUID":               "pk.UUID",
		"buffer":             "[]byte",
		"ByteArray":          "[]byte",
		"restBuffer":         "pk.PluginMessageData",
		"entityMetadataLoop": "pk.Metadata",
		"entityMetadata":     "pk.Metadata",
		"nbt":                "pk.NBT",
		"anonymousNbt":       "pk.NBT",
		"anonOptionalNbt":    "*pk.NBT",
		"optionalNbt":        "*pk.NBT",
		"position":           "pk.Position",
		"slot":               "pk.Slot",
		"Slot":               "pk.Slot",
		"component":          "pk.Component",
		"textComponent":      "pk.Component",
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
	case "string", "pstring":
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
	varName := strings.ToLower(fieldName[:1]) + fieldName[1:]
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
	case "i16", "u16":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Short)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i32", "u32":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Int)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i64", "u64":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Long)(&p.%s).ReadFrom(r)", fieldName),
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
	case "string", "pstring":
		code = []string{
			fmt.Sprintf("var %s pk.String", varName),
			fmt.Sprintf("temp, err = %s.ReadFrom(r)", varName),
			"n += temp",
			"if err != nil { return n, err }",
			fmt.Sprintf("p.%s = string(%s)", fieldName, varName),
		}
	case "UUID", "position", "slot", "Slot", "nbt", "anonymousNbt", "component", "textComponent":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.%s)(&p.%s).ReadFrom(r)", mapTypeToPkType(typeName), fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	default:
		code = []string{fmt.Sprintf("// TODO: Read %s (%s)", fieldName, typeName)}
	}

	return code
}

func mapTypeToPkType(t string) string {
	mapping := map[string]string{
		"UUID":          "UUID",
		"position":      "Position",
		"slot":          "Slot",
		"Slot":          "Slot",
		"nbt":           "NBT",
		"anonymousNbt":  "NBT",
		"component":     "Component",
		"textComponent": "Component",
	}
	if mapped, ok := mapping[t]; ok {
		return mapped
	}
	return t
}

// 生成寫入代碼
func generateWriteCode(fieldName, typeName string, optional bool) []string {
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
	case "i16", "u16":
		code = []string{
			fmt.Sprintf("temp, err = pk.Short(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i32", "u32":
		code = []string{
			fmt.Sprintf("temp, err = pk.Int(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "i64", "u64":
		code = []string{
			fmt.Sprintf("temp, err = pk.Long(p.%s).WriteTo(w)", fieldName),
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
	case "string", "pstring":
		code = []string{
			fmt.Sprintf("temp, err = pk.String(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}
	case "UUID", "position", "slot", "Slot", "nbt", "anonymousNbt", "component", "textComponent":
		code = []string{
			fmt.Sprintf("temp, err = p.%s.WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
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
	case "string", "pstring":
		code = append(code,
			"	var elem pk.String",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	val = string(elem)",
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
	case "string", "pstring":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.String(*p.%s).WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
	default:
		code = append(code,
			fmt.Sprintf("	temp, err = p.%s.WriteTo(w)", fieldName),
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
	countVar := strings.ToLower(fieldName[:1]) + fieldName[1:] + "Count"
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
	default:
		code = append(code,
			fmt.Sprintf("	temp, err = (*pk.%s)(&p.%s[i]).ReadFrom(r)", mapTypeToPkType(arrayType), fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
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
	default:
		code = append(code,
			fmt.Sprintf("	temp, err = p.%s[i].WriteTo(w)", fieldName),
			"	n += temp",
			"	if err != nil { return n, err }",
		)
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
	p.Imports["io"] = true
	p.Imports["git.konjactw.dev/falloutBot/go-mc/data/packetid"] = true
	p.Imports["pk \"git.konjactw.dev/falloutBot/go-mc/net/packet\""] = true
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

	// 模板 - 支持子结构体
	tmpl := template.Must(template.New("packet").Funcs(template.FuncMap{
		"join": func(lines []string) string {
			return strings.Join(lines, "\n\t")
		},
	}).Parse(`// Code generated by enhanced-generator v2 from protocol.json; DO NOT EDIT manually.
// To regenerate: go run main_v2.go -protocol <path> -output <dir> -direction {{.Direction}}

package {{.Package}}

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
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
func (s *{{.Name}}) ReadFrom(r io.Reader) (n int64, err error) {
	var temp int64
{{range .Fields}}
	{{join .ReadCode}}
{{end}}
	return n, nil
}

// WriteTo writes the data to the writer.
func (s {{.Name}}) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
{{range .Fields}}
	{{join .WriteCode}}
{{end}}
	return n, nil
}
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
	var temp int64
{{range .Fields}}
	{{join .ReadCode}}
{{end}}
	return n, nil
}

// WriteTo writes the packet data to the writer.
func (p {{.StructName}}) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
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
			GenerateCodec bool
			SubStructs    []StructDef
			GenerateInit  bool
			Direction     string
		}{
			Package:       packageName,
			StructName:    packet.StructName,
			Fields:        packet.Fields,
			PacketIDType:  packetIDType,
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
