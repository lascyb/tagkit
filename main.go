package tagkit

import (
	"strconv"
	"strings"

	"github.com/lascyb/tagkit/core"
)

// ============ 结果结构 ============

// TagValue 解析结果根结构；字段唯一（仅允许一个 name(...) 或 (args)），字段名与参数直接展开于此，外加标记位。
//
// 零值约定：存在字段调用时 Args 为非 nil map（无参字段为空 map），否则 Args、ArgKeys、Variables 均为 nil；
// Flags 总是非 nil。可用 HasField 判断是否存在字段调用
type TagValue struct {
	Name      string              // 唯一字段名，无字段或匿名字段时为空
	Args      map[string]ArgValue // 参数表（按名查找）
	Variables []VariableDetail    // 变量详情，按参数书写顺序排列
	Flags     []FlagInfo          // 标记位，按书写顺序排列；同名标记允许重复出现
	ArgKeys   []string            // v1.1.0：参数名，按书写顺序排列
}

// VariableDetail 变量的详细描述，含类型结构、默认值、数组维度等；用于代码生成或校验。
// Name 为空表示匿名变量（$ 或 $:Type），见 IsAnonymous
type VariableDetail struct {
	Name           string
	VarType        string         // 声明的类型（如 [Int!]!），未声明时为空
	TypeStruct     *core.TypeNode // 声明的类型结构，未声明时为 nil
	DefaultValue   interface{}    // 默认值的 Go 值（见 Literal.Value）
	HasDefault     bool
	IsArrayDefault bool
	ArrayType      string   // 数组默认值的元素类别：string / number / bool / null，空数组为空串
	Dimension      int      // 数组默认值的维度；非数组为 0
	Key            string   // 对应的参数名
	Default        *Literal // v1.1.0：默认值的结构化表示，无默认值时为 nil
}

// ArgValue 单个参数的值表示，可能是变量或字面量；用于 TagValue.Args 的 value
type ArgValue struct {
	Type           string      // ArgTypeLiteral 或 ArgTypeVariable
	Value          interface{} // 字面量：Go 值（见 Literal.Value）；变量：规范写法，如 $age:Int=18
	VarName        string
	VarType        string
	HasDefault     bool
	DefaultVal     interface{}
	IsArrayDefault bool
	Dimension      int
	Kind           ValueKind // v1.1.0：字面量种类；变量为 KindVariable
	Literal        *Literal  // v1.1.0：字面量的结构化表示；变量为 nil
	Default        *Literal  // v1.1.0：变量默认值的结构化表示；无默认值或字面量参数为 nil
}

// FlagInfo 单个标记位的结果，对应顶层 name（裸标记）或 name=value（键值标记）。
// IsBoolean 表示「裸标记、未写 =value」（此时 Value 为 true），并不表示取值类型为布尔：
// ,x=true 的 IsBoolean 为 false、Value 为 true；可使用语义更清晰的 HasValue
type FlagInfo struct {
	Name      string
	IsBoolean bool
	Value     interface{}
	ValueType string    // ValueTypeNumber / ValueTypeString / ValueTypeBool / ValueTypeNull / ValueTypeArray
	Kind      ValueKind // v1.1.0：取值种类；裸标记为 KindBool
	Literal   *Literal  // v1.1.0：取值的结构化表示；裸标记为 nil
}

// ParseTagValue 解析 tag 字符串为主入口，返回 TagValue 或解析错误（*SyntaxError）；字段唯一，多个字段调用时返回错误
func ParseTagValue(input string) (*TagValue, error) {
	nodes, err := core.NewParser(input).Parse()
	if err != nil {
		return nil, err
	}

	// 预先统计标记位数量，避免切片扩容
	nFlags := 0
	for _, node := range nodes {
		if _, ok := node.(*core.FlagNode); ok {
			nFlags++
		}
	}
	result := &TagValue{Flags: make([]FlagInfo, 0, nFlags)}
	for _, node := range nodes {
		switch n := node.(type) {
		case *core.FieldNode:
			if result.Args != nil {
				return nil, &SyntaxError{Pos: n.Pos, Msg: "field must be unique: multiple field calls not allowed"}
			}
			result.Name = n.Name
			result.Args = make(map[string]ArgValue, len(n.ArgList))
			result.ArgKeys = make([]string, 0, len(n.ArgList))
			nVars := 0
			for _, a := range n.ArgList {
				if _, ok := a.Value.(*core.VariableNode); ok {
					nVars++
				}
			}
			result.Variables = make([]VariableDetail, 0, nVars)
			for _, a := range n.ArgList {
				result.ArgKeys = append(result.ArgKeys, a.Name)
				switch val := a.Value.(type) {
				case *core.VariableNode:
					arg, detail := convertVariable(a.Name, val)
					result.Args[a.Name] = arg
					result.Variables = append(result.Variables, detail)
				default:
					lit := newLiteral(val)
					arg := ArgValue{Type: ArgTypeLiteral, Value: lit.Value, Kind: lit.Kind, Literal: lit}
					if lit.Kind == KindList {
						arg.Dimension = lit.Dimension()
					}
					result.Args[a.Name] = arg
				}
			}

		case *core.FlagNode:
			flag := FlagInfo{Name: n.Name}
			if n.Value == nil {
				flag.IsBoolean = true
				flag.Value = true
				flag.ValueType = ValueTypeBool
				flag.Kind = KindBool
			} else {
				lit := newLiteral(n.Value)
				flag.Value = lit.Value
				flag.Kind = lit.Kind
				flag.Literal = lit
				flag.ValueType = lit.valueType()
			}
			result.Flags = append(result.Flags, flag)
		}
	}
	return result, nil
}

// convertVariable 将变量节点转换为 ArgValue 与 VariableDetail
func convertVariable(key string, val *core.VariableNode) (ArgValue, VariableDetail) {
	varTypeStr := ""
	if val.VarType != nil {
		varTypeStr = val.VarType.String()
	}
	arg := ArgValue{
		Type:    ArgTypeVariable,
		Value:   val.Raw,
		VarName: val.Name,
		VarType: varTypeStr,
		Kind:    KindVariable,
	}
	detail := VariableDetail{
		Name:       val.Name,
		VarType:    varTypeStr,
		TypeStruct: val.VarType,
		Key:        key,
	}
	if val.DefaultValue != nil {
		def := newLiteral(val.DefaultValue)
		arg.HasDefault, detail.HasDefault = true, true
		arg.DefaultVal, detail.DefaultValue = def.Value, def.Value
		arg.Default, detail.Default = def, def
		if arr, ok := val.DefaultValue.(*core.ArrayNode); ok {
			dim := arr.Depth()
			arg.IsArrayDefault, detail.IsArrayDefault = true, true
			arg.Dimension, detail.Dimension = dim, dim
			detail.ArrayType = core.ArrayElemType(arr)
		}
	}
	return arg, detail
}

// newLiteral 将字面量或数组节点转换为 Literal
func newLiteral(n core.Node) *Literal {
	switch v := n.(type) {
	case *core.LiteralNode:
		return &Literal{Kind: literalKind(v), Value: convertLiteral(v), Text: v.String()}
	case *core.ArrayNode:
		elems := make([]*Literal, len(v.Elements))
		values := make([]interface{}, len(v.Elements))
		for i, e := range v.Elements {
			elems[i] = newLiteral(e)
			values[i] = elems[i].Value
		}
		return &Literal{Kind: KindList, Value: values, Text: v.String(), Elems: elems}
	}
	return nil
}

// literalKind 将 core.LiteralKind 映射为 ValueKind
func literalKind(l *core.LiteralNode) ValueKind {
	switch l.Kind {
	case core.LiteralString:
		return KindString
	case core.LiteralEnum:
		return KindEnum
	case core.LiteralNumber:
		return KindNumber
	case core.LiteralBool:
		return KindBool
	case core.LiteralNull:
		return KindNull
	}
	return ValueKind(l.Type)
}

// convertLiteral 将 LiteralNode 转为 Go 类型：整数→int、含小数点或指数→float64、bool→bool、null→nil、
// 字符串与枚举→string。数字范围已在解析阶段校验
func convertLiteral(l *core.LiteralNode) interface{} {
	switch l.Kind {
	case core.LiteralNumber:
		if strings.ContainsAny(l.Value, ".eE") {
			f, _ := strconv.ParseFloat(l.Value, 64)
			return f
		}
		i, _ := strconv.Atoi(l.Value)
		return i
	case core.LiteralBool:
		return l.Value == "true"
	case core.LiteralNull:
		return nil
	default:
		return l.Value
	}
}
