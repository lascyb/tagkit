package tagkit

import "regexp"

// defaultParser 默认解析器实例（使用默认字段名验证规则）
var defaultParser = NewParser()

// ParseValue 解析 tag value 字符串（全局函数，使用默认解析器）
// value: tag 的值（如 "name(arg:1,arg2:$var),inline,union=unionTypeName"）
// 返回: 解析结果
func ParseValue(value string) (*TagValue, error) {
	return defaultParser.ParseValue(value)
}

// SetFieldNameValidator 为默认解析器设置字段名验证器
// validator: 字段名验证函数，如果为 nil 则使用默认验证规则
// 返回: 默认解析器实例（支持链式调用）
func SetFieldNameValidator(validator FieldNameValidator) *Parser {
	return defaultParser.SetFieldNameValidator(validator)
}

// SetFieldNameValidatorByRegex 为默认解析器通过正则表达式设置字段名验证器
// regex: 预编译的正则表达式对象
// 返回: 默认解析器实例（支持链式调用）
func SetFieldNameValidatorByRegex(regex *regexp.Regexp) *Parser {
	return defaultParser.SetFieldNameValidatorByRegex(regex)
}
