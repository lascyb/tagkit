package core

import (
	"strconv"
	"strings"
)

// 数组元素的标量类别：字符串与枚举同属 string 类别（兼容 [a,b,c] 写法），null 可出现在任意位置
const (
	familyNone   = ""
	familyString = "string"
	familyNumber = "number"
	familyBool   = "bool"
	familyNull   = "null"
	familyMixed  = "mixed"
)

// literalFamily 返回字面量所属的标量类别
func literalFamily(l *LiteralNode) string {
	switch l.Kind {
	case LiteralString, LiteralEnum:
		return familyString
	case LiteralNumber:
		return familyNumber
	case LiteralBool:
		return familyBool
	case LiteralNull:
		return familyNull
	}
	if l.IsString {
		return familyString
	}
	return l.Type
}

// ArrayElemType 返回数组（含嵌套）中非 null 标量元素的类别：string（含枚举）/ number / bool；
// 全为 null 时返回 null，没有任何标量（如空数组）时返回空串；类别不一致时（仅可能出现在声明为 ID 或自定义类型的
// 变量默认值中，如 [1,'x']）返回 mixed
func ArrayElemType(a *ArrayNode) string {
	family := familyNone
	sawNull := false
	var walk func(*ArrayNode)
	walk = func(a *ArrayNode) {
		for _, e := range a.Elements {
			switch v := e.(type) {
			case *ArrayNode:
				walk(v)
			case *LiteralNode:
				switch f := literalFamily(v); {
				case f == familyNull:
					sawNull = true
				case family == familyNone:
					family = f
				case family != f:
					family = familyMixed
				}
			}
		}
	}
	walk(a)
	if family == familyNone && sawNull {
		return familyNull
	}
	return family
}

// checkArray 校验数组元素一致：同一层的非 null 元素要么都是标量、要么都是数组；非空子数组的维度必须相同；
// 所有非 null 标量属于同一类别（字符串/枚举、数字、布尔）。返回标量类别
func checkArray(a *ArrayNode) (string, error) {
	family := familyNone
	sawNull := false
	scalars, arrays := false, false
	depth := 0
	for _, e := range a.Elements {
		switch v := e.(type) {
		case *LiteralNode:
			f := literalFamily(v)
			if f == familyNull {
				sawNull = true
				continue
			}
			if arrays {
				return "", errorf(v.Pos, "array mixes scalars and lists")
			}
			scalars = true
			if family != familyNone && family != f {
				return "", errorf(v.Pos, "array mixes %s and %s elements", family, f)
			}
			family = f
		case *ArrayNode:
			if scalars {
				return "", errorf(v.Pos, "array mixes scalars and lists")
			}
			arrays = true
			sub, err := checkArray(v)
			if err != nil {
				return "", err
			}
			if len(v.Elements) > 0 {
				d := v.Depth()
				if depth != 0 && d != depth {
					return "", errorf(v.Pos, "array elements have different dimensions (%d and %d)", depth, d)
				}
				depth = d
			}
			if sub == familyNull {
				sawNull = true
				continue
			}
			if sub != familyNone {
				if family != familyNone && family != sub {
					return "", errorf(v.Pos, "array mixes %s and %s elements", family, sub)
				}
				family = sub
			}
		}
	}
	if family == familyNone && sawNull {
		family = familyNull
	}
	return family, nil
}

// checkDefault 校验变量默认值与声明类型是否相符：
//   - null 只能用于可空类型；
//   - 列表类型的默认值必须是数组且维度一致（[] 可用于任意列表类型），非列表类型不能用数组；
//   - 内置标量：Int 为 32 位整数、Float 为数字、Boolean 为布尔、String 为字符串或裸标识符、ID 为字符串/裸标识符/整数；
//   - 其他类型（自定义标量、枚举）不校验标量取值。
func checkDefault(t *TypeNode, v Node) error {
	switch n := v.(type) {
	case *LiteralNode:
		if n.Kind == LiteralNull {
			if t.IsNonNull {
				return errorf(n.Pos, "default value null is not allowed for non-null type %s", t)
			}
			return nil
		}
		if t.IsList {
			return errorf(n.Pos, "default value for list type %s must be a list", t)
		}
		return checkScalar(t, n)
	case *ArrayNode:
		if !t.IsList {
			return errorf(n.Pos, "list default value is not allowed for non-list type %s", t)
		}
		for _, e := range n.Elements {
			if err := checkDefault(t.InnerType, e); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkScalar 校验单个标量默认值与内置标量类型是否相符
func checkScalar(t *TypeNode, l *LiteralNode) error {
	isInt := l.Kind == LiteralNumber && !strings.ContainsAny(l.Value, ".eE")
	ok := true
	switch t.Name {
	case "Int":
		if isInt {
			if _, err := strconv.ParseInt(l.Value, 10, 32); err != nil {
				return errorf(l.Pos, "default value %s out of range for Int (32-bit)", l.Value)
			}
		}
		ok = isInt
	case "Float":
		ok = l.Kind == LiteralNumber
	case "Boolean":
		ok = l.Kind == LiteralBool
	case "String":
		ok = l.Kind == LiteralString || l.Kind == LiteralEnum
	case "ID":
		ok = l.Kind == LiteralString || l.Kind == LiteralEnum || isInt
	}
	if !ok {
		return errorf(l.Pos, "default value %s does not match type %s", l.String(), t)
	}
	return nil
}
