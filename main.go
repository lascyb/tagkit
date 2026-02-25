package tagkit

import (
	"fmt"
	"strconv"
	"strings"
	"tagkit/core"
)

// ============ 结果结构 ============

// TagValue 解析结果根结构；字段唯一（仅允许一个 name(...) 或 (args)），字段名与参数直接展开于此，外加标记位
type TagValue struct {
	Name      string              // 唯一字段名，无字段时为空
	Args      map[string]ArgValue // 参数表
	Variables []VariableDetail    // 变量详情
	Flags     []FlagInfo
}

// VariableDetail 变量的详细描述，含类型结构、默认值、数组维度等；用于代码生成或校验
type VariableDetail struct {
	Name           string
	VarType        string
	TypeStruct     *core.TypeNode
	DefaultValue   interface{}
	HasDefault     bool
	IsArrayDefault bool
	ArrayType      string
	Dimension      int // 数组维度
	Key            string
}

// ArgValue 单个参数的值表示，可能是变量或字面量；用于 TagValue.Args 的 value
type ArgValue struct {
	Type           string
	Value          interface{}
	VarName        string
	VarType        string
	HasDefault     bool
	DefaultVal     interface{}
	IsArrayDefault bool
	Dimension      int
}

// FlagInfo 单个标记位的结果，布尔标记或 name=value；对应顶层 name 或 name=value
type FlagInfo struct {
	Name      string
	IsBoolean bool
	Value     interface{}
	ValueType string
}

// ParseTagValue 解析 tag 字符串为主入口，返回 TagValue 或解析错误
func ParseTagValue(input string) (*TagValue, error) {
	parser := core.NewParser(input)
	nodes, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	result := &TagValue{
		Flags: make([]FlagInfo, 0),
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *core.FieldNode:
			if result.Args != nil {
				return nil, fmt.Errorf("field must be unique: multiple field calls not allowed")
			}
			result.Name = n.Name
			result.Args = make(map[string]ArgValue)
			result.Variables = make([]VariableDetail, 0)

			for k, v := range n.Args {
				switch val := v.(type) {
				case *core.VariableNode:
					defaultVal, hasDefault, isArray, arrayType, dimension := extractDefault(val.DefaultValue)

					result.Args[k] = ArgValue{
						Type:           "variable",
						Value:          val.Raw,
						VarName:        val.Name,
						VarType:        val.VarType.String(),
						HasDefault:     hasDefault,
						DefaultVal:     defaultVal,
						IsArrayDefault: isArray,
						Dimension:      dimension,
					}

					result.Variables = append(result.Variables, VariableDetail{
						Name:           val.Name,
						VarType:        val.VarType.String(),
						TypeStruct:     val.VarType,
						DefaultValue:   defaultVal,
						HasDefault:     hasDefault,
						IsArrayDefault: isArray,
						ArrayType:      arrayType,
						Dimension:      dimension,
						Key:            k,
					})

				case *core.LiteralNode:
					result.Args[k] = ArgValue{
						Type:  "literal",
						Value: convertLiteral(val),
					}
				}
			}

		case *core.FlagNode:
			flag := FlagInfo{Name: n.Name}
			if n.Value == nil {
				flag.IsBoolean = true
				flag.Value = true
				flag.ValueType = "bool"
			} else {
				flag.IsBoolean = false
				switch val := n.Value.(type) {
				case *core.LiteralNode:
					flag.Value = convertLiteral(val)
					flag.ValueType = val.Type
				case *core.ArrayNode:
					flag.Value = convertArray(val)
					flag.ValueType = "array"
				}
			}
			result.Flags = append(result.Flags, flag)
		}
	}

	return result, nil
}

// extractDefault 从变量的默认值节点提取 Go 值、是否存在、是否数组、数组元素类型与维度
func extractDefault(n core.Node) (interface{}, bool, bool, string, int) {
	if n == nil {
		return nil, false, false, "", 0
	}

	switch val := n.(type) {
	case *core.LiteralNode:
		return convertLiteral(val), true, false, "", 0
	case *core.ArrayNode:
		arr := convertArray(val)
		arrayType, dimension := inferArrayType(val)
		return arr, true, true, arrayType, dimension
	default:
		return nil, false, false, "", 0
	}
}

// convertLiteral 将 LiteralNode 转为 Go 类型：number→int/float64、bool→bool、其余→string
func convertLiteral(l *core.LiteralNode) interface{} {
	switch l.Type {
	case "number":
		if strings.Contains(l.Value, ".") {
			if f, err := strconv.ParseFloat(l.Value, 64); err == nil {
				return f
			}
		}
		if i, err := strconv.Atoi(l.Value); err == nil {
			return i
		}
		return l.Value
	case "bool":
		return l.Value == "true"
	default:
		return l.Value
	}
}

// convertArray 将 ArrayNode 递归转为 []interface{}，元素为字面量或嵌套切片
func convertArray(a *core.ArrayNode) interface{} {
	result := make([]interface{}, len(a.Elements))
	for i, e := range a.Elements {
		switch v := e.(type) {
		case *core.LiteralNode:
			result[i] = convertLiteral(v)
		case *core.ArrayNode:
			result[i] = convertArray(v)
		}
	}
	return result
}

// inferArrayType 推断数组类型和维度
func inferArrayType(a *core.ArrayNode) (string, int) {
	if len(a.Elements) == 0 {
		return "[]", 1
	}
	return inferNodeType(a.Elements[0])
}

// inferNodeType 根据节点类型推断类型名与维度：字面量返回类型与 1，数组递归推断并维度+1
func inferNodeType(n core.Node) (string, int) {
	switch v := n.(type) {
	case *core.LiteralNode:
		return v.Type, 1
	case *core.ArrayNode:
		innerType, innerDim := inferArrayType(v)
		baseType := strings.TrimSuffix(innerType, "[]")
		return baseType + "[]", innerDim + 1
	default:
		return "unknown", 1
	}
}
