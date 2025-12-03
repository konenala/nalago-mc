package main

import (
	"fmt"
	"strings"
)

// SwitchConfig Switch 类型的配置
type SwitchConfig struct {
	CompareTo string                 // 比较的字段路径，如 "flags/has_background_texture"
	Fields    map[string]interface{} // 条件值 -> 类型映射
	Default   string                 // 默认类型
}

// parseSwitchConfig 解析 Switch 配置
func parseSwitchConfig(switchDef []interface{}) *SwitchConfig {
	if len(switchDef) < 2 {
		return nil
	}

	configMap, ok := switchDef[1].(map[string]interface{})
	if !ok {
		return nil
	}

	config := &SwitchConfig{}

	if compareTo, ok := configMap["compareTo"].(string); ok {
		config.CompareTo = compareTo
	}

	if fields, ok := configMap["fields"].(map[string]interface{}); ok {
		config.Fields = fields
	}

	if defaultType, ok := configMap["default"].(string); ok {
		config.Default = defaultType
	}

	return config
}

// generateSwitchField 生成 Switch 字段的智能处理
func generateSwitchField(fieldName string, switchDef []interface{}, parentFields []PacketField) *PacketField {
	config := parseSwitchConfig(switchDef)
	if config == nil {
		// 无法解析，回退到 interface{}
		return &PacketField{
			Name:      fieldName,
			GoType:    "interface{}",
			Comment:   "// TODO: Switch type - conditional field based on other field value",
			ReadCode:  []string{"// TODO: Implement switch field read"},
			WriteCode: []string{"// TODO: Implement switch field write"},
		}
	}

	// 分析 compareTo 路径
	compareField, bitFlag := parseSwitchCompareTo(config.CompareTo, parentFields)
	if compareField == "" {
		// 无法找到比较字段，回退
		return generateFallbackSwitchField(fieldName, config)
	}

	// 判断 Switch 类型
	if len(config.Fields) == 1 && config.Default == "void" {
		// 简单的 optional 类型
		return generateOptionalSwitchField(fieldName, config, compareField, bitFlag)
	}

	// 复杂的 union 类型
	return generateUnionSwitchField(fieldName, config, compareField)
}

// parseSwitchCompareTo 解析 compareTo 路径
func parseSwitchCompareTo(compareTo string, parentFields []PacketField) (fieldName string, bitFlag int) {
	// 支持两种格式:
	// 1. "fieldName" - 直接字段
	// 2. "flags/bit_name" - 位字段

	parts := strings.Split(compareTo, "/")
	if len(parts) == 1 {
		// 直接字段
		return toPascalCase(parts[0]), -1
	}

	// 位字段
	fieldName = toPascalCase(parts[0])

	// 查找字段定义以确定位位置
	for _, f := range parentFields {
		if f.Name == fieldName {
			// 这里简化处理：假设位标志按顺序定义
			// 实际应该解析 bitfield 定义
			return fieldName, 0 // 简化：返回第一个位
		}
	}

	return fieldName, 0
}

// generateOptionalSwitchField 生成简单的 optional switch
func generateOptionalSwitchField(fieldName string, config *SwitchConfig, compareField string, bitFlag int) *PacketField {
	// 获取唯一的字段类型
	var innerType string
	for _, t := range config.Fields {
		if typeStr, ok := t.(string); ok {
			innerType = typeStr
			break
		}
	}

	if innerType == "" {
		return generateFallbackSwitchField(fieldName, config)
	}

	field := &PacketField{
		Name:     toPascalCase(fieldName),
		GoType:   "*" + mapType(innerType),
		Optional: true,
		Comment:  fmt.Sprintf("// Optional field, present when %s flag is set", compareField),
	}

	// 生成读取代码
	if bitFlag >= 0 {
		// 位标志检查
		field.ReadCode = generateBitFlagOptionalRead(field.Name, compareField, bitFlag, innerType)
		field.WriteCode = generateBitFlagOptionalWrite(field.Name, compareField, bitFlag, innerType)
	} else {
		// 直接字段检查
		field.ReadCode = generateDirectOptionalRead(field.Name, compareField, innerType)
		field.WriteCode = generateDirectOptionalWrite(field.Name, compareField, innerType)
	}

	return field
}

// generateBitFlagOptionalRead 生成基于位标志的可选读取
func generateBitFlagOptionalRead(fieldName, flagField string, bitPos int, innerType string) []string {
	code := []string{
		fmt.Sprintf("// Read %s if flag is set", fieldName),
		fmt.Sprintf("if p.%s & (1 << %d) != 0 {", flagField, bitPos),
		fmt.Sprintf("	var val %s", mapType(innerType)),
	}

	// 根据类型生成读取代码
	switch innerType {
	case "string", "pstring":
		code = append(code,
			"	var elem pk.String",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	val = string(elem)",
		)
	case "varint":
		code = append(code,
			"	var elem pk.VarInt",
			"	temp, err = elem.ReadFrom(r)",
			"	n += temp",
			"	if err != nil { return n, err }",
			"	val = int32(elem)",
		)
	default:
		code = append(code,
			fmt.Sprintf("	temp, err = (*pk.%s)(&val).ReadFrom(r)", mapTypeToPkType(innerType)),
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

// generateBitFlagOptionalWrite 生成基于位标志的可选写入
func generateBitFlagOptionalWrite(fieldName, flagField string, bitPos int, innerType string) []string {
	code := []string{
		fmt.Sprintf("// Write %s if present", fieldName),
		fmt.Sprintf("if p.%s != nil {", fieldName),
	}

	// 根据类型生成写入代码
	switch innerType {
	case "string", "pstring":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.String(*p.%s).WriteTo(w)", fieldName),
		)
	case "varint":
		code = append(code,
			fmt.Sprintf("	temp, err = pk.VarInt(*p.%s).WriteTo(w)", fieldName),
		)
	default:
		code = append(code,
			fmt.Sprintf("	temp, err = (*pk.%s)(p.%s).WriteTo(w)", mapTypeToPkType(innerType), fieldName),
		)
	}

	code = append(code,
		"	n += temp",
		"	if err != nil { return n, err }",
		"}",
	)

	return code
}

// generateDirectOptionalRead 生成基于直接字段的可选读取
func generateDirectOptionalRead(fieldName, compareField string, innerType string) []string {
	// 类似于上面，但使用直接比较
	return []string{
		fmt.Sprintf("// Read %s based on %s", fieldName, compareField),
		fmt.Sprintf("if p.%s != 0 { // TODO: Check correct condition", compareField),
		"	var val " + mapType(innerType),
		"	// TODO: Read val based on type",
		fmt.Sprintf("	p.%s = &val", fieldName),
		"}",
	}
}

// generateDirectOptionalWrite 生成基于直接字段的可选写入
func generateDirectOptionalWrite(fieldName, compareField string, innerType string) []string {
	return []string{
		fmt.Sprintf("if p.%s != nil {", fieldName),
		"	// TODO: Write based on type",
		"}",
	}
}

// generateUnionSwitchField 生成 union 类型的 switch
func generateUnionSwitchField(fieldName string, config *SwitchConfig, compareField string) *PacketField {
	// 为复杂的 union 类型生成 interface{}，但添加详细注释
	comment := fmt.Sprintf("// Switch field based on %s:\n", compareField)
	for value, fieldType := range config.Fields {
		comment += fmt.Sprintf("//   %s -> %v\n", value, fieldType)
	}
	if config.Default != "" {
		comment += fmt.Sprintf("//   default -> %s", config.Default)
	}

	return &PacketField{
		Name:      toPascalCase(fieldName),
		GoType:    "interface{}",
		Comment:   comment,
		ReadCode:  []string{"// TODO: Implement union switch read"},
		WriteCode: []string{"// TODO: Implement union switch write"},
	}
}

// generateFallbackSwitchField 生成回退的 switch 字段
func generateFallbackSwitchField(fieldName string, config *SwitchConfig) *PacketField {
	return &PacketField{
		Name:      toPascalCase(fieldName),
		GoType:    "interface{}",
		Comment:   "// TODO: Switch type - conditional field based on other field value",
		ReadCode:  []string{"// TODO: Implement switch field read"},
		WriteCode: []string{"// TODO: Implement switch field write"},
	}
}
