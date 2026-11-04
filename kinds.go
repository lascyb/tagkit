package tagkit

import (
	"strings"

	"github.com/lascyb/tagkit/core"
)

// SyntaxError 解析错误，含出错位置（字节偏移）与描述；可通过 errors.As 取得。等同于 core.SyntaxError
type SyntaxError = core.SyntaxError

// ArgValue.Type 的取值
const (
	ArgTypeLiteral  = "literal"  // 字面量参数
	ArgTypeVariable = "variable" // 变量参数
)

// FlagInfo.ValueType 的取值（兼容 v1.0.x：裸标识符/枚举的 ValueType 仍为 "string"）
const (
	ValueTypeNumber = "number"
	ValueTypeString = "string"
	ValueTypeBool   = "bool"
	ValueTypeNull   = "null"
	ValueTypeArray  = "array"
)

// ValueKind 值的种类（v1.1.0 新增），精确区分带引号字符串、裸标识符（枚举）、数字、布尔、null 与列表
type ValueKind string

const (
	KindString   ValueKind = "string"   // 带引号的字符串 'abc' / "abc"
	KindEnum     ValueKind = "enum"     // 裸标识符 CREATED_AT（Value 仍为 string，兼容 v1.0.x）
	KindNumber   ValueKind = "number"   // 数字：Value 为 int 或 float64
	KindBool     ValueKind = "bool"     // true / false
	KindNull     ValueKind = "null"     // null：Value 为 nil
	KindList     ValueKind = "list"     // 数组：Value 为 []interface{}
	KindVariable ValueKind = "variable" // 变量参数（仅用于 ArgValue.Kind）
)

// Literal 字面量的结构化表示（v1.1.0 新增），保留种类信息（包括数组内每个元素），便于区分字符串与枚举、输出原样写法
type Literal struct {
	Kind  ValueKind
	Value interface{} // Go 值：string / int / float64 / bool / nil / []interface{}
	Text  string      // 规范写法，可被重新解析：字符串为单引号并转义 \ 与 '，数字保留原文（如 1.5e3），数组为 [a,'b',1]
	Elems []*Literal  // Kind 为 KindList 时的元素
}

// Dimension 返回数组维度：非数组为 0，[1] 为 1，[[1]] 为 2，[] 为 1
func (l *Literal) Dimension() int {
	if l == nil || l.Kind != KindList {
		return 0
	}
	deepest := 0
	for _, e := range l.Elems {
		if d := e.Dimension(); d > deepest {
			deepest = d
		}
	}
	return deepest + 1
}

// valueType 返回兼容 v1.0.x 的 FlagInfo.ValueType
func (l *Literal) valueType() string {
	switch l.Kind {
	case KindList:
		return ValueTypeArray
	case KindEnum:
		return ValueTypeString
	}
	return string(l.Kind)
}

// ============ 便捷方法 ============

// HasField 是否存在字段调用（name(...)、(...) 或无前导逗号的裸标识符）
func (t *TagValue) HasField() bool {
	return t.Args != nil
}

// Arg 按名返回参数
func (t *TagValue) Arg(name string) (ArgValue, bool) {
	a, ok := t.Args[name]
	return a, ok
}

// NamedArg 带参数名的参数，用于按书写顺序遍历
type NamedArg struct {
	Name string
	ArgValue
}

// OrderedArgs 按书写顺序返回全部参数
func (t *TagValue) OrderedArgs() []NamedArg {
	out := make([]NamedArg, 0, len(t.ArgKeys))
	for _, k := range t.ArgKeys {
		out = append(out, NamedArg{Name: k, ArgValue: t.Args[k]})
	}
	return out
}

// Flag 返回第一个名为 name 的标记位
func (t *TagValue) Flag(name string) (FlagInfo, bool) {
	for _, f := range t.Flags {
		if f.Name == name {
			return f, true
		}
	}
	return FlagInfo{}, false
}

// HasFlag 是否存在名为 name 的标记位（裸标记或键值标记均可）
func (t *TagValue) HasFlag(name string) bool {
	_, ok := t.Flag(name)
	return ok
}

// FlagsNamed 按书写顺序返回所有名为 name 的标记位（同名标记允许重复，如多个 directive='...'）
func (t *TagValue) FlagsNamed(name string) []FlagInfo {
	var out []FlagInfo
	for _, f := range t.Flags {
		if f.Name == name {
			out = append(out, f)
		}
	}
	return out
}

// HasValue 是否为键值标记（name=value）；裸标记返回 false
func (f FlagInfo) HasValue() bool {
	return !f.IsBoolean
}

// IsAnonymous 是否为匿名变量（$ 或 $:Type）
func (v VariableDetail) IsAnonymous() bool {
	return v.Name == ""
}

// String 将 TagValue 序列化为规范的 tag 写法，满足 ParseTagValue(t.String()) 与 t 相等：
// 字段（若有）总是带括号输出在最前，参数按 ArgKeys 顺序；每个标记位前都带逗号，如 name(a:1,b:$b:Int),inline,limit=10
func (t *TagValue) String() string {
	var b strings.Builder
	if t.Args != nil {
		b.WriteString(t.Name)
		b.WriteByte('(')
		keys := t.ArgKeys
		if len(keys) != len(t.Args) {
			keys = sortedKeys(t.Args)
		}
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(k)
			b.WriteByte(':')
			b.WriteString(t.Args[k].text())
		}
		b.WriteByte(')')
	}
	for _, f := range t.Flags {
		b.WriteByte(',')
		b.WriteString(f.Name)
		if !f.IsBoolean {
			b.WriteByte('=')
			if f.Literal != nil {
				b.WriteString(f.Literal.Text)
			} else {
				b.WriteString(formatValue(f.Value))
			}
		}
	}
	return b.String()
}

// text 返回参数值的规范写法
func (a ArgValue) text() string {
	if a.Type == ArgTypeVariable {
		if s, ok := a.Value.(string); ok {
			return s
		}
	}
	if a.Literal != nil {
		return a.Literal.Text
	}
	return formatValue(a.Value)
}
