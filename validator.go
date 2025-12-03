package tagkit

import (
	"fmt"
	"regexp"
)

// 预编译字段名验证正则表达式
// 字段名规则：下划线_，大小写字母A-Z,a-z，数字0-9和中划线-
var fieldNameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// defaultFieldNameValidator 默认字段名验证函数
// 字段名允许的字符：下划线_，大小写字母A-Z,a-z，数字0-9和中划线-
// 其他字符应该报错
func defaultFieldNameValidator(fieldName string) error {
	if fieldName == "" {
		return nil // 空字段名是允许的（表示使用默认字段名）
	}

	// 使用正则表达式验证，更精准
	if !fieldNameRegex.MatchString(fieldName) {
		return fmt.Errorf("invalid field name '%s': field name can only contain letters (A-Z, a-z), digits (0-9), underscore (_), and hyphen (-)", fieldName)
	}

	return nil
}

// RegexValidator 基于正则表达式的字段名验证器
type RegexValidator struct {
	regex *regexp.Regexp
}

// Validate 验证字段名是否符合正则表达式规则
func (v *RegexValidator) Validate(fieldName string) error {
	if fieldName == "" {
		return nil // 空字段名是允许的（表示使用默认字段名）
	}

	if !v.regex.MatchString(fieldName) {
		return fmt.Errorf("invalid field name '%s': does not match pattern '%s'", fieldName, v.regex.String())
	}

	return nil
}
