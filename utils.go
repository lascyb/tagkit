package tagkit

import (
	"strings"
)

// splitFlags 分割标记位字符串，考虑标记位值中可能包含括号和逗号
// 例如：handle=func(a,b),other=value,inline
func splitFlags(flagsStr string) []string {
	var flags []string
	var current strings.Builder
	depth := 0

	for i, r := range flagsStr {
		switch r {
		case '=':
			current.WriteRune(r)
		case '(':
			depth++
			current.WriteRune(r)
		case ')':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				// 顶层逗号，分割标记位
				flags = append(flags, current.String())
				current.Reset()
			} else {
				// 括号内的逗号，保留（在标记位的值中）
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}

		// 最后一个字符
		if i == len(flagsStr)-1 {
			flags = append(flags, current.String())
		}
	}

	return flags
}

// splitArgs 分割参数字符串，考虑括号嵌套（包括圆括号和花括号）
func splitArgs(argsStr string) []string {
	var parts []string
	var current strings.Builder
	parenDepth := 0 // 圆括号深度
	braceDepth := 0 // 花括号深度

	for i, r := range argsStr {
		switch r {
		case '(':
			parenDepth++
			current.WriteRune(r)
		case ')':
			parenDepth--
			current.WriteRune(r)
		case '{':
			braceDepth++
			current.WriteRune(r)
		case '}':
			braceDepth--
			current.WriteRune(r)
		case ',':
			if parenDepth == 0 && braceDepth == 0 {
				// 顶层逗号，分割
				parts = append(parts, current.String())
				current.Reset()
			} else {
				// 括号或花括号内的逗号，保留
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}

		// 最后一个字符
		if i == len(argsStr)-1 {
			parts = append(parts, current.String())
		}
	}

	return parts
}

// findMatchingCloseParen 查找匹配的右括号
// s: 字符串
// openIdx: 左括号的位置
// 返回: 右括号的位置，如果未找到返回 -1
func findMatchingCloseParen(s string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '(' {
		return -1
	}

	depth := 1
	for i := openIdx + 1; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// parseArgs 解析参数字符串
// argsStr: 参数字符串（如 "first:10, end2:2, arg:$, arg1:$arg1"）
// 返回: Arg map
func parseArgs(argsStr string) (map[string]*Arg, error) {
	args := make(map[string]*Arg)
	if argsStr == "" {
		return args, nil
	}

	// 按逗号分割参数（需要考虑括号嵌套，但这里参数已经在括号内，所以直接分割即可）
	parts := splitArgs(argsStr)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 按冒号分割键值对
		idx := strings.Index(part, ":")
		if idx < 0 {
			// 参数格式错误，跳过
			continue
		}

		key := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])

		if key == "" {
			continue
		}

		argMeta := &Arg{
			Name:        key,
			Value:       value,
			Placeholder: strings.HasPrefix(value, "$"),
			CustomName:  "",
		}

		// 处理占位符
		if argMeta.Placeholder {
			if value == "$" {
				argMeta.CustomName = ""
			} else {
				// 提取自定义变量名（去掉 $ 前缀）
				argMeta.CustomName = strings.TrimPrefix(value, "$")
			}
		}

		args[key] = argMeta
	}

	return args, nil
}
