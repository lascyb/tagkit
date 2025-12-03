package tagkit

import (
	"fmt"
	"regexp"
	"strings"
)

// Parser tag value 解析控制器
type Parser struct {
	// FieldNameValidator 字段名验证函数，如果为 nil 则使用默认验证规则
	FieldNameValidator FieldNameValidator
}

// NewParser 创建新的解析控制器
// 返回: 解析控制器实例（使用默认配置）
// 所有配置通过 Set 方法设置
func NewParser() *Parser {
	return &Parser{
		FieldNameValidator: nil, // 使用默认验证规则
	}
}

// SetFieldNameValidator 设置字段名验证器
// validator: 字段名验证函数，如果为 nil 则使用默认验证规则
// 返回: Parser 实例（支持链式调用）
func (p *Parser) SetFieldNameValidator(validator FieldNameValidator) *Parser {
	p.FieldNameValidator = validator
	return p
}

// SetFieldNameValidatorByRegex 通过正则表达式设置字段名验证器
// regex: 预编译的正则表达式对象
// 返回: Parser 实例（支持链式调用）
func (p *Parser) SetFieldNameValidatorByRegex(regex *regexp.Regexp) *Parser {
	if regex == nil {
		// 如果传入 nil，则使用默认验证规则
		p.SetFieldNameValidator(nil)
		return p
	}
	validator := &RegexValidator{regex: regex}
	p.SetFieldNameValidator(validator.Validate)
	return p
}

// ParseValue 解析 tag value 字符串
// value: tag 的值（如 "name(arg:1,arg2:$var),inline,union=unionTypeName"）
// 返回: 解析结果
func (p *Parser) ParseValue(value string) (*TagValue, error) {
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
		return p.parseFlagsOnly(value, result)
	}

	// 2. 查找第一个逗号的位置（用于区分字段名部分和标记位部分）
	firstCommaIdx := strings.Index(value, ",")

	// 3. 判断字段名部分是否包含括号（参数列表）
	if firstCommaIdx >= 0 {
		// 有逗号，检查第一个逗号之前是否有括号
		fieldPart := value[:firstCommaIdx]
		if strings.Contains(fieldPart, "(") {
			// 第一个逗号之前有括号，说明是字段名+参数
			return p.parseWithParentheses(value, result)
		}
		// 第一个逗号之前没有括号，可能是字段名+标记位，或者只有标记位
		// 需要判断：如果第一个部分看起来像标记位（没有等号且是常见标记位），可能是只有标记位
		// 但为了简化，我们假设第一个部分是字段名
		return p.parseFieldNameAndFlags(value, result)
	}

	// 没有逗号，检查整个字符串是否包含括号
	if strings.Contains(value, "(") {
		// 有括号但没有逗号，说明是字段名+参数，没有标记位
		return p.parseWithParentheses(value, result)
	}
	// 没有括号也没有逗号，可能是字段名，也可能是单个标记位
	// 为了兼容性，假设是字段名（如果用户想要只有标记位，应该以逗号开头）
	if err := p.validateFieldName(value); err != nil {
		return nil, err
	}
	result.FieldName = value
	return result, nil
}

// parseFlagsOnly 解析只有标记位的情况（如 ",inline,union"）
func (p *Parser) parseFlagsOnly(tagValue string, result *TagValue) (*TagValue, error) {
	// 去掉开头的逗号
	flagsStr := strings.TrimPrefix(tagValue, ",")
	return p.parseFlags(flagsStr, result)
}

// parseWithParentheses 解析包含括号的情况（如 "name(arg:1,arg2:$var),inline,union"）
func (p *Parser) parseWithParentheses(tagValue string, result *TagValue) (*TagValue, error) {
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
	if err := p.validateFieldName(fieldName); err != nil {
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
		return p.parseFlags(flagsStr, result)
	}

	return result, nil
}

// parseFieldNameAndFlags 解析字段名和标记位（无括号，如 "fieldName,inline,union"）
func (p *Parser) parseFieldNameAndFlags(tagValue string, result *TagValue) (*TagValue, error) {
	// 查找第一个逗号
	commaIdx := strings.Index(tagValue, ",")
	if commaIdx < 0 {
		if err := p.validateFieldName(tagValue); err != nil {
			return nil, err
		}
		result.FieldName = tagValue
		return result, nil
	}

	// 提取字段名
	fieldName := strings.TrimSpace(tagValue[:commaIdx])
	if fieldName != "" {
		if err := p.validateFieldName(fieldName); err != nil {
			return nil, err
		}
	}
	result.FieldName = fieldName

	// 提取标记位
	flagsStr := strings.TrimSpace(tagValue[commaIdx+1:])
	return p.parseFlags(flagsStr, result)
}

// parseFlags 解析标记位字符串
func (p *Parser) parseFlags(flagsStr string, result *TagValue) (*TagValue, error) {
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
		if idx := strings.Index(flag, "="); idx >= 0 {
			flagName := strings.TrimSpace(flag[:idx])
			if flagName == "" {
				// 没有名称的标记位直接报错（例如 "=value"）
				return nil, fmt.Errorf("parseFlags: flag name is required in %q", flag)
			}
			result.Flags = append(result.Flags, flagName)
			result.FlagValues[flagName] = strings.TrimSpace(flag[idx+1:])
		} else {
			result.Flags = append(result.Flags, flag)
		}
	}

	return result, nil
}

// validateFieldName 验证字段名是否符合规则
// 如果 Parser 配置了自定义验证器，则使用自定义验证器；否则使用默认验证器
func (p *Parser) validateFieldName(fieldName string) error {
	if p.FieldNameValidator != nil {
		return p.FieldNameValidator(fieldName)
	}
	return defaultFieldNameValidator(fieldName)
}
