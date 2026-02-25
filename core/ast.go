package core

import (
	"fmt"
	"strings"
)

// ============ AST 节点 ============

// Node 抽象语法树节点接口，所有解析得到的语法单元均实现此接口；用于统一遍历与序列化
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

// String 将 TypeNode 序列化为 [Inner]! 或 Name! 等形式，与输入语法一致
func (t *TypeNode) String() string {
	if t.IsList {
		inner := t.InnerType.String()
		if t.InnerType.IsNonNull {
			inner = inner[:len(inner)-1]
		}
		s := "[" + inner + "]"
		if t.IsNonNull {
			s += "!"
		}
		return s
	}
	s := t.Name
	if t.IsNonNull {
		s += "!"
	}
	return s
}

// FieldNode 字段调用节点，表示 name(key:value,...) 形式的调用；用于描述带参数的指令或配置项
type FieldNode struct {
	Name string
	Args map[string]Node
}

// String 将 FieldNode 格式化为 name(k1:v1,k2:v2,...)，便于调试与错误提示
func (f *FieldNode) String() string {
	var args []string
	for k, v := range f.Args {
		args = append(args, fmt.Sprintf("%s:%s", k, v.String()))
	}
	return fmt.Sprintf("%s(%s)", f.Name, strings.Join(args, ","))
}

// VariableNode 变量节点，表示 $name:Type=default 形式的参数；用于字段调用中的形参
type VariableNode struct {
	Name         string
	VarType      *TypeNode
	DefaultValue Node
	Raw          string
}

// String 将 VariableNode 格式化为 $name:Type 或 $name:Type=default
func (v *VariableNode) String() string {
	s := fmt.Sprintf("$%s:%s", v.Name, v.VarType.String())
	if v.DefaultValue != nil {
		s += "=" + v.DefaultValue.String()
	}
	return s
}

// LiteralNode 字面量节点，表示数字、字符串、布尔等常量值；用于参数默认值与标记位取值
type LiteralNode struct {
	Value    string
	Type     string
	IsString bool
}

// String 将 LiteralNode 序列化为原始值，字符串带引号
func (l *LiteralNode) String() string {
	if l.IsString {
		return fmt.Sprintf("'%s'", l.Value)
	}
	return l.Value
}

// ArrayNode 数组节点，支持嵌套 [e1,e2] 或 [[...]]；用于默认值中的数组字面量
type ArrayNode struct {
	Elements []Node
}

// String 将 ArrayNode 序列化为 [e1,e2,...] 形式
func (a *ArrayNode) String() string {
	var elems []string
	for _, e := range a.Elements {
		elems = append(elems, e.String())
	}
	return "[" + strings.Join(elems, ",") + "]"
}

// FlagNode 标记位节点，表示 name 或 name=value；用于布尔开关或键值型配置
type FlagNode struct {
	Name  string
	Value Node
}

// String 将 FlagNode 格式化为 name 或 name=value
func (f *FlagNode) String() string {
	if f.Value == nil {
		return f.Name
	}
	return fmt.Sprintf("%s=%s", f.Name, f.Value.String())
}
