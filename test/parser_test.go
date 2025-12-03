package test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/lascyb/tagkit"
)

// TestNewParser 测试创建新的解析器
func TestNewParser(t *testing.T) {
	parser := tagkit.NewParser()
	if parser == nil {
		t.Fatal("NewParser() returned nil")
	}

	// 使用默认验证器应该能解析标准字段名
	result, err := parser.ParseValue("fieldName")
	if err != nil {
		t.Fatalf("ParseValue failed with default validator: %v", err)
	}
	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}
}

// TestParser_SetFieldNameValidator 测试设置自定义验证器
func TestParser_SetFieldNameValidator(t *testing.T) {
	parser := tagkit.NewParser()

	// 设置自定义验证器：只允许长度至少为3的字段名
	customValidator := func(fieldName string) error {
		if fieldName == "" {
			return nil
		}
		if len(fieldName) < 3 {
			return fmt.Errorf("field name '%s' must be at least 3 characters", fieldName)
		}
		return nil
	}
	parser.SetFieldNameValidator(customValidator)

	// 测试短字段名（应该失败）
	_, err := parser.ParseValue("ab")
	if err == nil {
		t.Error("Expected error for short field name, got nil")
	}

	// 测试长字段名（应该通过）
	result, err := parser.ParseValue("abc")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}
	if result.FieldName != "abc" {
		t.Errorf("Expected FieldName 'abc', got '%s'", result.FieldName)
	}
}

// TestParser_SetFieldNameValidatorByRegex 测试通过正则表达式设置验证器
func TestParser_SetFieldNameValidatorByRegex(t *testing.T) {
	parser := tagkit.NewParser()

	// 设置只允许小写字母和下划线的验证规则
	regex := regexp.MustCompile(`^[a-z_]+$`)
	parser.SetFieldNameValidatorByRegex(regex)

	// 测试小写字母（应该通过）
	result, err := parser.ParseValue("field_name")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}
	if result.FieldName != "field_name" {
		t.Errorf("Expected FieldName 'field_name', got '%s'", result.FieldName)
	}

	// 测试包含大写字母（应该失败）
	_, err = parser.ParseValue("FieldName")
	if err == nil {
		t.Error("Expected error for field name with uppercase, got nil")
	}

	// 测试包含数字（应该失败）
	_, err = parser.ParseValue("field123")
	if err == nil {
		t.Error("Expected error for field name with digits, got nil")
	}
}

// TestParser_SetFieldNameValidatorByRegex_Nil 测试传入 nil 正则表达式
func TestParser_SetFieldNameValidatorByRegex_Nil(t *testing.T) {
	parser := tagkit.NewParser()

	// 传入 nil 应该使用默认验证规则
	parser.SetFieldNameValidatorByRegex(nil)

	// 应该能解析标准字段名
	result, err := parser.ParseValue("fieldName123")
	if err != nil {
		t.Fatalf("ParseValue failed with nil regex (should use default): %v", err)
	}
	if result.FieldName != "fieldName123" {
		t.Errorf("Expected FieldName 'fieldName123', got '%s'", result.FieldName)
	}
}

// TestParser_ChainCalls 测试链式调用
func TestParser_ChainCalls(t *testing.T) {
	// 测试链式调用
	regex := regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
	parser := tagkit.NewParser().
		SetFieldNameValidatorByRegex(regex)

	// 测试首字母大写的字段名（应该通过）
	result, err := parser.ParseValue("FieldName")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}
	if result.FieldName != "FieldName" {
		t.Errorf("Expected FieldName 'FieldName', got '%s'", result.FieldName)
	}

	// 测试小写开头的字段名（应该失败）
	_, err = parser.ParseValue("fieldName")
	if err == nil {
		t.Error("Expected error for lowercase field name, got nil")
	}
}

// TestParser_SetFieldNameValidator_Chain 测试 SetFieldNameValidator 链式调用
func TestParser_SetFieldNameValidator_Chain(t *testing.T) {
	// 测试 SetFieldNameValidator 也支持链式调用
	customValidator := func(fieldName string) error {
		return nil // 允许所有字段名
	}

	parser := tagkit.NewParser().
		SetFieldNameValidator(customValidator)

	// 应该能解析任何字段名（包括特殊字符）
	result, err := parser.ParseValue("field@name")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}
	if result.FieldName != "field@name" {
		t.Errorf("Expected FieldName 'field@name', got '%s'", result.FieldName)
	}
}

// TestParser_InstanceVsGlobal 测试实例解析器和全局函数的区别
func TestParser_InstanceVsGlobal(t *testing.T) {
	// 创建自定义解析器
	regex := regexp.MustCompile(`^[a-z]+$`)
	parser := tagkit.NewParser().
		SetFieldNameValidatorByRegex(regex)

	// 使用自定义解析器（应该只允许小写字母）
	_, err := parser.ParseValue("FieldName")
	if err == nil {
		t.Error("Expected error for uppercase in custom parser, got nil")
	}

	// 使用全局函数（应该使用默认验证规则，允许大小写字母）
	result, err := tagkit.ParseValue("FieldName")
	if err != nil {
		t.Fatalf("Global ParseValue failed: %v", err)
	}
	if result.FieldName != "FieldName" {
		t.Errorf("Expected FieldName 'FieldName', got '%s'", result.FieldName)
	}
}

// TestParser_ComplexUsage 测试复杂使用场景
func TestParser_ComplexUsage(t *testing.T) {
	// 创建解析器并设置验证规则
	regex := regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
	parser := tagkit.NewParser().
		SetFieldNameValidatorByRegex(regex)

	// 测试完整的 tag value 解析
	result, err := parser.ParseValue("field_name(arg:1,arg2:$var),inline,union=type")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "field_name" {
		t.Errorf("Expected FieldName 'field_name', got '%s'", result.FieldName)
	}
	if len(result.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(result.Args))
	}
	if len(result.Flags) != 2 {
		t.Errorf("Expected 2 flags, got %d", len(result.Flags))
	}
}

// TestParser_ResetValidator 测试重置验证器
func TestParser_ResetValidator(t *testing.T) {
	parser := tagkit.NewParser()

	// 设置严格验证规则
	strictRegex := regexp.MustCompile(`^[a-z]+$`)
	parser.SetFieldNameValidatorByRegex(strictRegex)

	// 测试应该失败
	_, err := parser.ParseValue("FieldName")
	if err == nil {
		t.Error("Expected error with strict validator, got nil")
	}

	// 重置为 nil（使用默认验证规则）
	parser.SetFieldNameValidator(nil)

	// 现在应该能通过
	result, err := parser.ParseValue("FieldName")
	if err != nil {
		t.Fatalf("ParseValue failed after reset: %v", err)
	}
	if result.FieldName != "FieldName" {
		t.Errorf("Expected FieldName 'FieldName', got '%s'", result.FieldName)
	}
}

// TestSetFieldNameValidator_Global 测试全局 SetFieldNameValidator 函数
func TestSetFieldNameValidator_Global(t *testing.T) {
	// 设置默认解析器的验证器
	customValidator := func(fieldName string) error {
		if fieldName == "" {
			return nil
		}
		if len(fieldName) < 3 {
			return fmt.Errorf("field name '%s' must be at least 3 characters", fieldName)
		}
		return nil
	}
	tagkit.SetFieldNameValidator(customValidator)

	// 使用全局 ParseValue 应该应用自定义验证规则
	_, err := tagkit.ParseValue("ab")
	if err == nil {
		t.Error("Expected error for short field name with global validator, got nil")
	}

	result, err := tagkit.ParseValue("abc")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}
	if result.FieldName != "abc" {
		t.Errorf("Expected FieldName 'abc', got '%s'", result.FieldName)
	}

	// 重置为默认验证规则
	tagkit.SetFieldNameValidator(nil)
}

// TestSetFieldNameValidatorByRegex_Global 测试全局 SetFieldNameValidatorByRegex 函数
func TestSetFieldNameValidatorByRegex_Global(t *testing.T) {
	// 设置默认解析器的正则验证规则
	regex := regexp.MustCompile(`^[a-z_]+$`)
	tagkit.SetFieldNameValidatorByRegex(regex)

	// 使用全局 ParseValue 应该应用正则验证规则
	result, err := tagkit.ParseValue("field_name")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}
	if result.FieldName != "field_name" {
		t.Errorf("Expected FieldName 'field_name', got '%s'", result.FieldName)
	}

	// 测试包含大写字母（应该失败）
	_, err = tagkit.ParseValue("FieldName")
	if err == nil {
		t.Error("Expected error for field name with uppercase with global regex validator, got nil")
	}

	// 重置为默认验证规则
	tagkit.SetFieldNameValidatorByRegex(nil)
}

// TestSetFieldNameValidator_Global_Chain 测试全局 Set 函数的链式调用
func TestSetFieldNameValidator_Global_Chain(t *testing.T) {
	// 测试链式调用
	regex := regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
	parser := tagkit.SetFieldNameValidatorByRegex(regex)

	// 应该返回默认解析器实例
	if parser == nil {
		t.Fatal("SetFieldNameValidatorByRegex returned nil")
	}

	// 使用全局 ParseValue 应该应用正则验证规则
	result, err := tagkit.ParseValue("FieldName")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}
	if result.FieldName != "FieldName" {
		t.Errorf("Expected FieldName 'FieldName', got '%s'", result.FieldName)
	}

	// 重置为默认验证规则
	tagkit.SetFieldNameValidator(nil)
}
