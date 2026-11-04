package core

import (
	"sort"
	"strings"
)

// ============ AST 节点 ============

// Node 抽象语法树节点接口，所有解析得到的语法单元均实现此接口；String() 输出可被重新解析的规范写法
type Node interface {
	String() string
}

// TypeNode 类型定义节点，表示标量/列表/非空等类型信息；用于变量声明与类型推断
type TypeNode struct {
	Name      string
	IsNonNull bool
	IsList    bool
	InnerType *TypeNode
}

// String 将 TypeNode 序列化为 [Inner]! 或 Name! 等形式，与输入语法一致；列表内层非空保留，如 [String!]!
func (t *TypeNode) String() string {
	var b strings.Builder
	t.writeTo(&b)
	return b.String()
}

func (t *TypeNode) writeTo(b *strings.Builder) {
	if t.IsList {
		b.WriteByte('[')
		if t.InnerType != nil {
			t.InnerType.writeTo(b)
		}
		b.WriteByte(']')
	} else {
		b.WriteString(t.Name)
	}
	if t.IsNonNull {
		b.WriteByte('!')
	}
}

// ListDepth 返回列表嵌套层数：Int → 0，[Int] → 1，[[Int!]]! → 2
func (t *TypeNode) ListDepth() int {
	d := 0
	for n := t; n != nil && n.IsList; n = n.InnerType {
		d++
	}
	return d
}

// BaseName 返回去掉列表与非空修饰后的基础类型名：[[Int!]]! → Int
func (t *TypeNode) BaseName() string {
	n := t
	for n != nil && n.IsList {
		n = n.InnerType
	}
	if n == nil {
		return ""
	}
	return n.Name
}

// ArgNode 字段调用中的单个参数 key:value，按书写顺序保存在 FieldNode.ArgList 中
type ArgNode struct {
	Name  string
	Value Node
	Pos   int // 参数名的位置（字节偏移）
}

// FieldNode 字段调用节点，表示 name(key:value,...) 形式的调用；用于描述带参数的指令或配置项。
// Args 便于按名查找（保留以兼容 v1.0.x），ArgList 保留书写顺序（v1.1.0 新增）
type FieldNode struct {
	Name    string
	Args    map[string]Node
	ArgList []*ArgNode
	Pos     int // 字段名（匿名字段为左括号）的位置
}

// String 将 FieldNode 格式化为 name(k1:v1,k2:v2,...)，参数按书写顺序输出；
// 手动构造且未设置 ArgList 时按参数名字典序输出，保证结果稳定
func (f *FieldNode) String() string {
	var b strings.Builder
	b.WriteString(f.Name)
	b.WriteByte('(')
	if len(f.ArgList) > 0 {
		for i, a := range f.ArgList {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(a.Name)
			b.WriteByte(':')
			b.WriteString(a.Value.String())
		}
	} else {
		keys := make([]string, 0, len(f.Args))
		for k := range f.Args {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(k)
			b.WriteByte(':')
			b.WriteString(f.Args[k].String())
		}
	}
	b.WriteByte(')')
	return b.String()
}

// VariableNode 变量节点，表示 $name 或 $name:Type=default 形式的参数；Type 可选，未设置时 VarType 为 nil。
// Name 为空表示匿名变量（$ 或 $:Type）。Raw 为规范写法（等于 String()），可被重新解析
type VariableNode struct {
	Name         string
	VarType      *TypeNode
	DefaultValue Node
	Raw          string
	Pos          int // $ 的位置
}

// String 将 VariableNode 格式化为 $name、$name:Type 或 $name:Type=default；类型未设置时不输出 :Type
func (v *VariableNode) String() string {
	var b strings.Builder
	b.WriteByte('$')
	b.WriteString(v.Name)
	if v.VarType != nil {
		b.WriteByte(':')
		v.VarType.writeTo(&b)
	}
	if v.DefaultValue != nil {
		b.WriteByte('=')
		b.WriteString(v.DefaultValue.String())
	}
	return b.String()
}

// LiteralKind 字面量种类（v1.1.0 新增）；零值 LiteralUnspecified 表示手动构造且未设置种类
type LiteralKind int

const (
	LiteralUnspecified LiteralKind = iota // 未指定（兼容手动构造的节点，按 IsString 序列化）
	LiteralString                         // 带引号的字符串 'abc' / "abc"
	LiteralEnum                           // 裸标识符 CREATED_AT（GraphQL 枚举值）
	LiteralNumber                         // 数字 10、-1、1.5、1e3
	LiteralBool                           // true / false
	LiteralNull                           // null
)

// String 返回种类名：string、enum、number、bool、null
func (k LiteralKind) String() string {
	switch k {
	case LiteralString:
		return "string"
	case LiteralEnum:
		return "enum"
	case LiteralNumber:
		return "number"
	case LiteralBool:
		return "bool"
	case LiteralNull:
		return "null"
	default:
		return "unspecified"
	}
}

// LiteralNode 字面量节点，表示数字、字符串、布尔、枚举、null 等常量值；用于参数值、默认值与标记位取值。
//   - Value：字符串为还原转义后的内容；数字为原文（如 1.5e3）；其余为原文。
//   - Type：兼容 v1.0.x，取值 number / string / bool / null；裸标识符（枚举）仍为 "string"。
//   - IsString：兼容 v1.0.x，带引号字符串与裸标识符均为 true。
//   - Kind：v1.1.0 新增，精确区分 string / enum / number / bool / null。
type LiteralNode struct {
	Value    string
	Type     string
	IsString bool
	Kind     LiteralKind
	Pos      int
}

// String 将 LiteralNode 序列化为规范写法：字符串加单引号并转义 \ 与 '，其余原样输出
func (l *LiteralNode) String() string {
	switch l.Kind {
	case LiteralString:
		return Quote(l.Value)
	case LiteralUnspecified:
		if l.IsString {
			return Quote(l.Value)
		}
	}
	return l.Value
}

// ArrayNode 数组节点，支持嵌套 [e1,e2] 或 [[...]]；用于参数值、默认值与标记位中的数组字面量
type ArrayNode struct {
	Elements []Node
	Pos      int // 左方括号的位置
}

// String 将 ArrayNode 序列化为 [e1,e2,...] 形式
func (a *ArrayNode) String() string {
	var b strings.Builder
	b.WriteByte('[')
	for i, e := range a.Elements {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(e.String())
	}
	b.WriteByte(']')
	return b.String()
}

// Depth 返回数组嵌套层数：[1] → 1，[[1]] → 2，[] → 1
func (a *ArrayNode) Depth() int {
	deepest := 0
	for _, e := range a.Elements {
		if sub, ok := e.(*ArrayNode); ok {
			if d := sub.Depth(); d > deepest {
				deepest = d
			}
		}
	}
	return deepest + 1
}

// FlagNode 标记位节点，表示 name 或 name=value；用于布尔开关或键值型配置
type FlagNode struct {
	Name  string
	Value Node
	Pos   int
}

// String 将 FlagNode 格式化为 name 或 name=value
func (f *FlagNode) String() string {
	if f.Value == nil {
		return f.Name
	}
	return f.Name + "=" + f.Value.String()
}
