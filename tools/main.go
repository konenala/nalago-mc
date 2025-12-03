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
}

// PacketDef 封包定義
type PacketDef struct {
	Name       string
	StructName string
	Fields     []PacketField
	PacketID   string
	Imports    map[string]bool
}

var (
	protocolFile = flag.String("protocol", "", "Path to protocol.json")
	outputDir    = flag.String("output", "", "Output directory")
	direction    = flag.String("direction", "client", "client or server")
	verbose      = flag.Bool("v", false, "Verbose output")
	genCodec     = flag.Bool("codec", true, "Generate ReadFrom/WriteTo methods")
)

func main() {
	flag.Parse()

	if *protocolFile == "" || *outputDir == "" {
		log.Fatal("Usage: enhanced-generator -protocol <protocol.json> -output <dir> -direction <client|server>")
	}

	// 讀取 protocol.json
	data, err := os.ReadFile(*protocolFile)
	if err != nil {
		log.Fatalf("讀取協議文件失敗: %v", err)
	}

	var protocol Protocol
	if err := json.Unmarshal(data, &protocol); err != nil {
		log.Fatalf("解析協議文件失敗: %v", err)
	}

	// 根據方向選擇封包類型
	var packetTypes map[string]interface{}
	if *direction == "client" {
		packetTypes = protocol.Play.ToClient.Types
	} else {
		packetTypes = protocol.Play.ToServer.Types
	}

	// 解析所有封包
	packets := parsePackets(packetTypes, protocol.Types)

	if *verbose {
		log.Printf("解析了 %d 個封包", len(packets))
	}

	// 生成代碼
	if err := generatePackets(packets, *outputDir, *direction); err != nil {
		log.Fatalf("生成代碼失敗: %v", err)
	}

	fmt.Printf("✅ 成功生成 %d 個封包定義到 %s\n", len(packets), *outputDir)
}

func parsePackets(packetTypes map[string]interface{}, globalTypes map[string]interface{}) []PacketDef {
	var packets []PacketDef

	for name, def := range packetTypes {
		if !strings.HasPrefix(name, "packet_") {
			continue
		}

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
		Name:       name,
		StructName: structName,
		Fields:     parseFields(fields, globalTypes),
		Imports:    make(map[string]bool),
	}

	// 收集需要的導入
	packet.collectImports()

	return packet
}

func parseFields(fields []interface{}, globalTypes map[string]interface{}) []PacketField {
	var result []PacketField

	for _, f := range fields {
		fieldMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := fieldMap["name"].(string)
		fieldType := fieldMap["type"]

		field := parseFieldType(name, fieldType, globalTypes)
		if field != nil {
			result = append(result, *field)
		}
	}

	return result
}

func parseFieldType(name string, fieldType interface{}, globalTypes map[string]interface{}) *PacketField {
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
				field.Optional = true
				field.NeedsPointer = true
				if len(t) > 1 {
					if innerType, ok := t[1].(string); ok {
						field.GoType = "*" + mapType(innerType)
						field.ReadCode = generateOptionalReadCode(field.Name, innerType)
						field.WriteCode = generateOptionalWriteCode(field.Name, innerType)
					}
				}

			case "array":
				// 數組類型
				field.IsArray = true
				if len(t) > 1 {
					if arrayDef, ok := t[1].(map[string]interface{}); ok {
						countType, _ := arrayDef["countType"].(string)
						if countType == "" {
							countType = "varint"
						}
						field.ArrayCount = countType

						if arrayType, ok := arrayDef["type"].(string); ok {
							field.ArrayType = arrayType
							field.GoType = "[]" + mapType(arrayType)
							field.ReadCode = generateArrayReadCode(field.Name, arrayType, countType)
							field.WriteCode = generateArrayWriteCode(field.Name, arrayType, countType)
						} else if _, ok := arrayDef["type"].([]interface{}); ok {
							// 複雜數組元素類型
							field.GoType = "[]interface{}"
							field.Comment = "// TODO: Complex array element type"
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
						}
					}
				}

			case "switch":
				// Switch 類型（複雜）
				field.GoType = "interface{}"
				field.Comment = "// TODO: Implement switch type"

			case "container":
				// 嵌套 container
				field.GoType = "interface{}"
				field.Comment = "// TODO: Implement nested container"

			default:
				// 未知複雜類型
				field.GoType = "interface{}"
				field.Comment = fmt.Sprintf("// TODO: Implement %s type", typeName)
			}
		}
	}

	return field
}

// 類型映射（完整版本）
func mapType(t string) string {
	mapping := map[string]string{
		// 整數類型
		"varint":    "int32",
		"varlong":   "int64",
		"optvarint": "*int32",
		"i8":        "int8",
		"i16":       "int16",
		"i32":       "int32",
		"i64":       "int64",
		"u8":        "uint8",
		"u16":       "uint16",
		"u32":       "uint32",
		"u64":       "uint64",
		// 浮點類型
		"f32": "float32",
		"f64": "float64",
		// 其他基本類型
		"bool":       "bool",
		"string":     "string",
		"pstring":    "string",
		"UUID":       "pk.UUID",
		"buffer":     "[]byte",
		"ByteArray":  "[]byte",
		"restBuffer": "pk.PluginMessageData",
		// 特殊類型
		"entityMetadataLoop": "pk.Metadata",
		"entityMetadata":     "pk.Metadata",
		"nbt":                "pk.NBT",
		"anonymousNbt":       "pk.NBT",
		"anonOptionalNbt":    "*pk.NBT",
		"optionalNbt":        "*pk.NBT",
		// 位置相關
		"position": "pk.Position",
		// 向量
		"vec2f":   "pk.Vec2f",
		"vec3f":   "pk.Vec3f",
		"vec3f64": "pk.Vec3d",
		"vec3i":   "pk.Vec3i",
		// 角度
		"angle": "pk.Angle",
		// 組件
		"component":         "pk.Component",
		"textComponent":     "pk.Component",
		"formattedChatComp": "pk.Component",
		// Slot
		"slot":         "pk.Slot",
		"Slot":         "pk.Slot",
		"optionalSlot": "*pk.Slot",
		// 容器和聲音
		"ContainerID":       "int8",
		"soundSource":       "int32",
		"Particle":          "pk.Particle",
		"ItemSoundHolder":   "interface{}",
		"RecipeDisplay":     "interface{}",
		"RecipeBookSetting": "interface{}",
		"ChatTypesHolder":   "interface{}",
		"SpawnInfo":         "interface{}",
		// 其他類型
		"PositionUpdateRelatives": "int32",
		"command_node":            "interface{}",
		"chunkBlockEntity":        "interface{}",
		"previousMessages":        "interface{}",
		"bitfield":                "uint32",
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
	case "angle":
		return "`mc:\"Angle\"`"
	default:
		return ""
	}
}

// 生成讀取代碼（增強版）
func generateReadCode(fieldName, typeName string, optional bool) []string {
	varName := safeVarName(strings.ToLower(fieldName[:1]) + fieldName[1:])
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

	case "f32", "f64":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Float)(&p.%s).ReadFrom(r)", fieldName),
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

	case "UUID":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.UUID)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}

	case "buffer":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.ByteArray)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}

	case "angle":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Angle)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}

	case "position":
		code = []string{
			fmt.Sprintf("temp, err = (*pk.Position)(&p.%s).ReadFrom(r)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}

	default:
		code = []string{fmt.Sprintf("// TODO: Read %s (%s)", fieldName, typeName)}
	}

	return code
}

// 生成寫入代碼（增強版）
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

	case "f32", "f64":
		code = []string{
			fmt.Sprintf("temp, err = pk.Float(p.%s).WriteTo(w)", fieldName),
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

	case "UUID":
		code = []string{
			fmt.Sprintf("temp, err = p.%s.WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}

	case "buffer":
		code = []string{
			fmt.Sprintf("temp, err = pk.ByteArray(p.%s).WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}

	case "angle":
		code = []string{
			fmt.Sprintf("temp, err = p.%s.WriteTo(w)", fieldName),
			"n += temp",
			"if err != nil { return n, err }",
		}

	case "position":
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
	return []string{
		fmt.Sprintf("var has%s pk.Boolean", fieldName),
		fmt.Sprintf("temp, err = has%s.ReadFrom(r)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("if has%s {", fieldName),
		fmt.Sprintf("	var val %s", mapType(innerType)),
		"	// TODO: Read value",
		fmt.Sprintf("	p.%s = &val", fieldName),
		"}",
	}
}

// 生成 Optional 寫入代碼
func generateOptionalWriteCode(fieldName, innerType string) []string {
	return []string{
		fmt.Sprintf("if p.%s != nil {", fieldName),
		"	temp, err = pk.Boolean(true).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"	// TODO: Write value",
		"} else {",
		"	temp, err = pk.Boolean(false).WriteTo(w)",
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	}
}

// 生成 Array 讀取代碼
func generateArrayReadCode(fieldName, arrayType, countType string) []string {
	countVar := strings.ToLower(fieldName) + "Count"
	return []string{
		fmt.Sprintf("var %s pk.VarInt", countVar),
		fmt.Sprintf("temp, err = %s.ReadFrom(r)", countVar),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("p.%s = make([]%s, %s)", fieldName, mapType(arrayType), countVar),
		fmt.Sprintf("for i := 0; i < int(%s); i++ {", countVar),
		fmt.Sprintf("	// TODO: Read array element of type %s", arrayType),
		"}",
	}
}

// 生成 Array 寫入代碼
func generateArrayWriteCode(fieldName, arrayType, countType string) []string {
	return []string{
		fmt.Sprintf("temp, err = pk.VarInt(len(p.%s)).WriteTo(w)", fieldName),
		"n += temp",
		"if err != nil { return n, err }",
		fmt.Sprintf("for _, item := range p.%s {", fieldName),
		fmt.Sprintf("	// TODO: Write array element of type %s", arrayType),
		"}",
	}
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

// safeVarName 避免 Go 關鍵字 / 保留字衝突
func safeVarName(name string) string {
	keywords := map[string]struct{}{
		"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
		"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {},
		"chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {},
		"const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {},
		"continue": {}, "for": {}, "import": {}, "return": {}, "var": {},
	}
	if _, ok := keywords[name]; ok {
		return "_" + name
	}
	return name
}

// 收集需要的導入
func (p *PacketDef) collectImports() {
	p.Imports["io"] = true
	p.Imports["git.konjactw.dev/falloutBot/go-mc/data/packetid"] = true
	p.Imports["pk \"git.konjactw.dev/falloutBot/go-mc/net/packet\""] = true
}

// 生成代碼
func generatePackets(packets []PacketDef, outputDir, direction string) error {
	// 確保輸出目錄存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	packageName := direction
	packetIDType := "Clientbound"
	if direction == "server" {
		packetIDType = "Serverbound"
	}

	// 模板
	tmpl := template.Must(template.New("packet").Funcs(template.FuncMap{
		"join": func(lines []string) string {
			return strings.Join(lines, "\n\t")
		},
	}).Parse(`// Code generated by enhanced-generator from protocol.json; DO NOT EDIT manually.
// To regenerate: make gen-packets

package {{.Package}}

import (
	"io"

	"git.konjactw.dev/falloutBot/go-mc/data/packetid"
	pk "git.konjactw.dev/falloutBot/go-mc/net/packet"
)

{{if .Comment}}// {{.Comment}}{{end}}
type {{.StructName}} struct {
{{- range .Fields}}
{{if .Comment}}	{{.Comment}}{{end}}
	{{.Name}} {{.GoType}}{{if .MCTag}} {{.MCTag}}{{end}}
{{- end}}
}

func (*{{.StructName}}) PacketID() packetid.{{.PacketIDType}}PacketID {
	return packetid.{{.PacketIDType}}{{.StructName}}
}

{{if .GenerateCodec}}
// ReadFrom reads the packet from the reader
func (p *{{.StructName}}) ReadFrom(r io.Reader) (n int64, err error) {
	var temp int64
{{range .Fields}}
	{{join .ReadCode}}
{{end}}
	return n, nil
}

// WriteTo writes the packet to the writer
func (p {{.StructName}}) WriteTo(w io.Writer) (n int64, err error) {
	var temp int64
{{range .Fields}}
	{{join .WriteCode}}
{{end}}
	return n, nil
}
{{end}}

func init() {
	registerPacket(packetid.{{.PacketIDType}}{{.StructName}}, func() {{.PacketIDType}}Packet {
		return &{{.StructName}}{}
	})
}
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
		}{
			Package:       packageName,
			StructName:    packet.StructName,
			Fields:        packet.Fields,
			PacketIDType:  packetIDType,
			GenerateCodec: *genCodec,
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
