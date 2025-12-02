package tagkit

import (
	"fmt"
	"regexp"
	"strings"
)

// TagValue tag value 解析结果
type TagValue struct {
	FieldName  string            // 字段名（如果为空，表示使用默认字段名）
	Args       map[string]*Arg   // 参数列表（key: 参数名）
	Flags      []string          // 布尔标记位列表
	FlagValues map[string]string // 带值的标记位（key: 标记位名称, value: 标记位的值）
}

// Arg 表示 GraphQL 参数元数据
type Arg struct {
	Name        string // 参数名（如 "first", "end2", "arg"）
	Value       string // 参数值（字面量，如 "1", "true"）
	Placeholder bool   // 是否为占位符（$ 开头）
	CustomName  string // 自定义变量名（如果指定了 $arg1，则为 "arg1"；如果为 $，则为空）
}

// ParseValue  解析 tag value 字符串
// value: tag 的值（如 "name(arg:1,arg2:$var),inline,union=unionTypeName"）
// 返回: 解析结果
func ParseValue(value string) (*TagValue, error) {
	result := &TagValue{
		FieldName:  "",
		Args:       make(map[string]*Arg),
		Flags:      []string{},
		FlagValues: make(map[string]string),
	}

	// 预处理：去除首尾空格
	value = strings.TrimSpace(value)
	if value == "" {
		return result, nil
	}

	// 判断格式并解析
	// 1. 如果以逗号开头 → 无字段名，只有标记位
	if strings.HasPrefix(value, ",") {
		return parseFlagsOnly(value, result)
	}

	// 2. 查找第一个逗号的位置（用于区分字段名部分和标记位部分）
	firstCommaIdx := strings.Index(value, ",")

	// 3. 判断字段名部分是否包含括号（参数列表）
	if firstCommaIdx >= 0 {
		// 有逗号，检查第一个逗号之前是否有括号
		fieldPart := value[:firstCommaIdx]
		if strings.Contains(fieldPart, "(") {
			// 第一个逗号之前有括号，说明是字段名+参数
			return parseWithParentheses(value, result)
		}
		// 第一个逗号之前没有括号，可能是字段名+标记位，或者只有标记位
		// 需要判断：如果第一个部分看起来像标记位（没有等号且是常见标记位），可能是只有标记位
		// 但为了简化，我们假设第一个部分是字段名
		return parseFieldNameAndFlags(value, result)
	}

	// 没有逗号，检查整个字符串是否包含括号
	if strings.Contains(value, "(") {
		// 有括号但没有逗号，说明是字段名+参数，没有标记位
		return parseWithParentheses(value, result)
	}
	// 没有括号也没有逗号，可能是字段名，也可能是单个标记位
	// 为了兼容性，假设是字段名（如果用户想要只有标记位，应该以逗号开头）
	if err := validateFieldName(value); err != nil {
		return nil, err
	}
	result.FieldName = value
	return result, nil
}

// parseFlagsOnly 解析只有标记位的情况（如 ",inline,union"）
func parseFlagsOnly(tagValue string, result *TagValue) (*TagValue, error) {
	// 去掉开头的逗号
	flagsStr := strings.TrimPrefix(tagValue, ",")
	return parseFlags(flagsStr, result)
}

// parseWithParentheses 解析包含括号的情况（如 "name(arg:1,arg2:$var),inline,union"）
func parseWithParentheses(tagValue string, result *TagValue) (*TagValue, error) {
	// 查找第一个左括号
	openIdx := strings.Index(tagValue, "(")
	if openIdx < 0 {
		return result, fmt.Errorf("unmatched parentheses in tag value")
	}

	// 找到匹配的右括号
	closeIdx := findMatchingCloseParen(tagValue, openIdx)
	if closeIdx < 0 {
		return result, fmt.Errorf("unmatched parentheses in tag value")
	}

	// 提取字段名
	fieldName := strings.TrimSpace(tagValue[:openIdx])
	if fieldName == "" {
		return result, fmt.Errorf("field name cannot be empty when parentheses are present")
	}
	if err := validateFieldName(fieldName); err != nil {
		return nil, err
	}
	result.FieldName = fieldName

	// 提取参数列表
	argsStr := strings.TrimSpace(tagValue[openIdx+1 : closeIdx])
	if argsStr != "" {
		args, err := parseArgs(argsStr)
		if err != nil {
			// 参数解析错误，记录但不中断
			// 可以在这里记录警告
		} else {
			result.Args = args
		}
	}

	// 提取标记位（右括号之后的部分）
	flagsStr := strings.TrimSpace(tagValue[closeIdx+1:])
	if flagsStr != "" {
		// 去掉开头的逗号（如果有）
		flagsStr = strings.TrimPrefix(flagsStr, ",")
		return parseFlags(flagsStr, result)
	}

	return result, nil
}

// parseFieldNameAndFlags 解析字段名和标记位（无括号，如 "fieldName,inline,union"）
func parseFieldNameAndFlags(tagValue string, result *TagValue) (*TagValue, error) {
	// 查找第一个逗号
	commaIdx := strings.Index(tagValue, ",")
	if commaIdx < 0 {
		if err := validateFieldName(tagValue); err != nil {
			return nil, err
		}
		result.FieldName = tagValue
		return result, nil
	}

	// 提取字段名
	fieldName := strings.TrimSpace(tagValue[:commaIdx])
	if fieldName != "" {
		if err := validateFieldName(fieldName); err != nil {
			return nil, err
		}
	}
	result.FieldName = fieldName

	// 提取标记位
	flagsStr := strings.TrimSpace(tagValue[commaIdx+1:])
	return parseFlags(flagsStr, result)
}

// parseFlags 解析标记位字符串
func parseFlags(flagsStr string, result *TagValue) (*TagValue, error) {
	if flagsStr == "" {
		return result, nil
	}

	// 使用智能分割，考虑标记位值中可能包含括号
	flags := splitFlags(flagsStr)
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}

		// 分割标记位名称和值
		var flagName, flagValue string
		if idx := strings.Index(flag, "="); idx >= 0 {
			flagName = strings.TrimSpace(flag[:idx])
			flagValue = strings.TrimSpace(flag[idx+1:])
		} else {
			flagName = flag
		}

		if flagName != "" {
			result.Flags = append(result.Flags, flagName)
			// 始终设置 FlagValues（即使没有值，值为空字符串）
			result.FlagValues[flagName] = flagValue
		}
	}

	return result, nil
}

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

// 预编译字段名验证正则表达式
// 字段名规则：下划线_，大小写字母A-Z,a-z，数字0-9和中划线-
var fieldNameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validateFieldName 验证字段名是否符合规则
// 字段名允许的字符：下划线_，大小写字母A-Z,a-z，数字0-9和中划线-
// 其他字符应该报错
// 使用正则表达式进行验证，更精准
func validateFieldName(fieldName string) error {
	if fieldName == "" {
		return nil // 空字段名是允许的（表示使用默认字段名）
	}

	// 使用正则表达式验证，更精准
	if !fieldNameRegex.MatchString(fieldName) {
		return fmt.Errorf("invalid field name '%s': field name can only contain letters (A-Z, a-z), digits (0-9), underscore (_), and hyphen (-)", fieldName)
	}

	return nil
}
