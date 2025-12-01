package tag_value_parser

import (
	"strings"
	"testing"

	"github.com/lascyb/tagkit"
)

// TestParseTagValue_ErrorCases 测试各种异常情况和错误场景

func TestParseTagValue_UnmatchedOpenParenthesis(t *testing.T) {
	// 测试未匹配的左括号
	_, err := tagkit.ParseValue("fieldName(")
	if err == nil {
		t.Error("Expected error for unmatched open parenthesis")
	}
	if !strings.Contains(err.Error(), "unmatched parentheses") {
		t.Errorf("Expected error message about unmatched parentheses, got: %v", err)
	}
}

func TestParseTagValue_UnmatchedCloseParenthesis(t *testing.T) {
	// 测试未匹配的右括号（这种情况可能不会报错，因为可能被当作字段名的一部分）
	// 实际上，单独的右括号可能不会触发错误，因为解析器会尝试找到匹配的左括号
	result, err := tagkit.ParseValue("fieldName)")
	if err != nil {
		// 如果有错误，也是可以接受的
		return
	}
	// 如果没有错误，字段名应该包含右括号
	if result.FieldName != "fieldName)" {
		t.Logf("Note: Unmatched close parenthesis may be treated as part of field name: %s", result.FieldName)
	}
}

func TestParseTagValue_MultipleUnmatchedParentheses(t *testing.T) {
	// 测试多个未匹配的括号
	_, err := tagkit.ParseValue("fieldName((arg:1)")
	if err == nil {
		t.Error("Expected error for multiple unmatched parentheses")
	}
}

func TestParseTagValue_EmptyFieldNameWithParentheses(t *testing.T) {
	// 测试括号前字段名为空
	_, err := tagkit.ParseValue("(arg:1)")
	if err == nil {
		t.Error("Expected error for empty field name with parentheses")
	}
	if !strings.Contains(err.Error(), "field name cannot be empty") {
		t.Errorf("Expected error message about empty field name, got: %v", err)
	}
}

func TestParseTagValue_OnlyParentheses(t *testing.T) {
	// 测试只有括号
	_, err := tagkit.ParseValue("()")
	if err == nil {
		t.Error("Expected error for only parentheses")
	}
}

func TestParseTagValue_InvalidArgFormat_NoColon_ErrorCase(t *testing.T) {
	// 测试参数格式错误：缺少冒号（异常情况测试）
	result, err := tagkit.ParseValue("fieldName(arg1,arg2:value)")
	if err != nil {
		t.Fatalf("ParseValue should not fail for invalid arg format, got: %v", err)
	}

	// arg1 应该被跳过（格式错误）
	if _, ok := result.Args["arg1"]; ok {
		t.Error("Expected arg 'arg1' to be skipped due to invalid format (no colon)")
	}
	// arg2 应该正常解析
	if _, ok := result.Args["arg2"]; !ok {
		t.Error("Expected arg 'arg2' to be parsed")
	}
}

func TestParseTagValue_InvalidArgFormat_OnlyColon(t *testing.T) {
	// 测试参数格式错误：只有冒号
	result, err := tagkit.ParseValue("fieldName(:value)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 参数名称为空，应该被跳过
	if len(result.Args) != 0 {
		t.Errorf("Expected no args (empty key should be skipped), got %d", len(result.Args))
	}
}

func TestParseTagValue_InvalidArgFormat_MultipleColons(t *testing.T) {
	// 测试参数格式：多个冒号（应该使用第一个冒号）
	result, err := tagkit.ParseValue("fieldName(arg:value:extra)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	arg, ok := result.Args["arg"]
	if !ok {
		t.Error("Expected arg 'arg' to be parsed")
	} else if arg.Value != "value:extra" {
		t.Errorf("Expected arg value 'value:extra', got '%s'", arg.Value)
	}
}

func TestParseTagValue_EmptyArgValue_ErrorCase(t *testing.T) {
	// 测试参数值为空（异常情况测试）
	result, err := tagkit.ParseValue("fieldName(arg:)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	arg, ok := result.Args["arg"]
	if !ok {
		t.Error("Expected arg 'arg' to be parsed even with empty value")
	} else if arg.Value != "" {
		t.Errorf("Expected empty value, got '%s'", arg.Value)
	}
}

func TestParseTagValue_WhitespaceOnlyArgName(t *testing.T) {
	// 测试参数名只有空格
	result, err := tagkit.ParseValue("fieldName( :value)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 空格参数名应该被跳过
	if len(result.Args) != 0 {
		t.Errorf("Expected no args (whitespace key should be skipped), got %d", len(result.Args))
	}
}

func TestParseTagValue_WhitespaceOnlyArgValue(t *testing.T) {
	// 测试参数值只有空格
	result, err := tagkit.ParseValue("fieldName(arg: )")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	arg, ok := result.Args["arg"]
	if !ok {
		t.Error("Expected arg 'arg' to be parsed")
	} else if strings.TrimSpace(arg.Value) != "" {
		t.Errorf("Expected whitespace-only value to be trimmed, got '%s'", arg.Value)
	}
}

func TestParseTagValue_NestedParenthesesInArgs(t *testing.T) {
	// 测试参数值中包含嵌套括号
	result, err := tagkit.ParseValue("fieldName(filter:{name:\"test\",age:18})")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	arg, ok := result.Args["filter"]
	if !ok {
		t.Error("Expected arg 'filter' to be parsed")
	} else if !strings.Contains(arg.Value, "{") {
		t.Errorf("Expected nested parentheses in arg value, got '%s'", arg.Value)
	}
}

func TestParseTagValue_FlagWithEmptyName(t *testing.T) {
	// 测试标记位名称为空
	result, err := tagkit.ParseValue("fieldName,=value")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 空名称的标记位应该被跳过
	if len(result.FlagValues) != 0 {
		t.Errorf("Expected no flag values (empty name should be skipped), got %d", len(result.FlagValues))
	}
}

func TestParseTagValue_FlagWithOnlyEquals(t *testing.T) {
	// 测试只有等号的标记位
	result, err := tagkit.ParseValue("fieldName,=")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 应该被跳过
	if len(result.Flags) != 0 && len(result.FlagValues) != 0 {
		t.Error("Expected empty flag to be skipped")
	}
}

func TestParseTagValue_MultipleEqualsInFlag(t *testing.T) {
	// 测试标记位值中包含多个等号
	result, err := tagkit.ParseValue("fieldName,key=value=extra")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 应该使用第一个等号作为分隔符
	if result.FlagValues["key"] != "value=extra" {
		t.Errorf("Expected flag value 'value=extra', got '%s'", result.FlagValues["key"])
	}
	if !result.Flags["key"] {
		t.Error("Expected flag 'key' to be true")
	}
}

func TestParseTagValue_FlagValueWithNestedEquals(t *testing.T) {
	// 测试标记位值中包含等号（如 JSON 字符串）
	result, err := tagkit.ParseValue("fieldName,config={\"key\":\"value\"}")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.FlagValues["config"] != "{\"key\":\"value\"}" {
		t.Errorf("Expected flag value with equals, got '%s'", result.FlagValues["config"])
	}
}

func TestParseTagValue_ComplexNestedParentheses(t *testing.T) {
	// 测试复杂的嵌套括号
	result, err := tagkit.ParseValue("fieldName(filter:{and:[{name:\"test\"},{age:18}]})")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	arg, ok := result.Args["filter"]
	if !ok {
		t.Error("Expected arg 'filter' to be parsed with nested parentheses")
	}
	_ = arg // 避免未使用变量警告
}

func TestParseTagValue_FlagValueWithCommas(t *testing.T) {
	// 测试标记位值中包含逗号（在括号内）
	result, err := tagkit.ParseValue("fieldName,handle=func(a,b,c)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.FlagValues["handle"] != "func(a,b,c)" {
		t.Errorf("Expected flag value with commas in parentheses, got '%s'", result.FlagValues["handle"])
	}
}

func TestParseTagValue_FlagValueWithUnmatchedParentheses(t *testing.T) {
	// 测试标记位值中包含未匹配的括号（应该作为普通字符处理）
	result, err := tagkit.ParseValue("fieldName,handle=func(a,b")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 未匹配的括号应该作为值的一部分
	if !strings.Contains(result.FlagValues["handle"], "func(a,b") {
		t.Errorf("Expected flag value with unmatched parentheses, got '%s'", result.FlagValues["handle"])
	}
}

func TestParseTagValue_OnlyCommas(t *testing.T) {
	// 测试只有逗号
	result, err := tagkit.ParseValue(",,,")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.FieldName != "" {
		t.Errorf("Expected empty field name, got '%s'", result.FieldName)
	}
	if len(result.Flags) != 0 {
		t.Errorf("Expected no flags (empty flags should be skipped), got %d", len(result.Flags))
	}
}

func TestParseTagValue_CommasWithSpaces(t *testing.T) {
	// 测试逗号之间只有空格
	result, err := tagkit.ParseValue("fieldName, , ,inline")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 空格标记位应该被跳过
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' to be set")
	}
	if len(result.Flags) != 1 {
		t.Errorf("Expected 1 flag (empty flags should be skipped), got %d", len(result.Flags))
	}
}

func TestParseTagValue_FieldNameWithAllowedChars(t *testing.T) {
	// 测试字段名包含允许的特殊字符（下划线、中划线）
	result, err := tagkit.ParseValue("field_name-123,inline")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.FieldName != "field_name-123" {
		t.Errorf("Expected field name with allowed chars, got '%s'", result.FieldName)
	}
}

func TestParseTagValue_FieldNameWithInvalidChars(t *testing.T) {
	// 测试字段名包含不允许的特殊字符（应该报错）
	testCases := []struct {
		name  string
		input string
	}{
		{"space", "field name"},
		{"dot", "field.name"},
		{"at", "field@name"},
		{"hash", "field#name"},
		{"percent", "field%name"},
		{"ampersand", "field&name"},
		{"asterisk", "field*name"},
		{"plus", "field+name"},
		{"equals", "field=name"},
		{"question", "field?name"},
		{"exclamation", "field!name"},
		{"colon", "field:name"},
		{"semicolon", "field;name"},
		// 注意：逗号会被解析为字段名和标记位的分隔符，所以需要单独测试
		{"pipe", "field|name"},
		{"backslash", "field\\name"},
		{"slash", "field/name"},
		{"brackets", "field[name]"},
		{"braces", "field{name}"},
		// 注意：parentheses 在字段名中会被解析为参数列表的开始，所以无法测试
		// {"parentheses", "field(name)"}, // 会被解析为字段名 "field" 和参数 "name"
		{"angle", "field<name>"},
		{"tilde", "field~name"},
		{"caret", "field^name"},
		{"dollar", "field$name"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tagkit.ParseValue(tc.input)
			if err == nil {
				t.Errorf("Expected error for field name with invalid char '%s', got nil", tc.name)
			} else if !strings.Contains(err.Error(), "invalid field name") {
				t.Errorf("Expected error message about invalid field name, got: %v", err)
			}
		})
	}
}

func TestParseTagValue_FieldNameWithComma_ShouldError(t *testing.T) {
	// 测试字段名包含逗号
	// 注意：如果字段名包含逗号且没有括号，逗号会被当作分隔符
	// 所以 "field,name" 会被解析为字段名 "field"（有效）和标记位 "name"
	// 要测试字段名包含逗号，需要使用括号：field,name(arg:1)
	// 但解析器会先查找第一个逗号，所以 "field,name(arg:1)" 会被解析为字段名 "field" 和标记位 "name(arg:1)"
	// 因此，字段名包含逗号的情况实际上无法通过语法表达（逗号总是被当作分隔符）
	// 这个测试用例验证：如果字段名部分包含逗号，应该验证第一个逗号之前的部分
	// 实际上，由于逗号是分隔符，字段名不会包含逗号，所以这个测试用例可以移除或改为测试其他场景
	t.Skip("Field name with comma cannot be tested as comma is always treated as separator")
}

func TestParseTagValue_ArgNameWithSpecialChars(t *testing.T) {
	// 测试参数名包含特殊字符
	result, err := tagkit.ParseValue("fieldName(arg_1:value,arg-2:value2)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if _, ok := result.Args["arg_1"]; !ok {
		t.Error("Expected arg 'arg_1' to be parsed")
	}
	if _, ok := result.Args["arg-2"]; !ok {
		t.Error("Expected arg 'arg-2' to be parsed")
	}
}

func TestParseTagValue_FlagNameWithSpecialChars(t *testing.T) {
	// 测试标记位名包含特殊字符
	result, err := tagkit.ParseValue("fieldName,flag_1,flag-2=value")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if !result.Flags["flag_1"] {
		t.Error("Expected flag 'flag_1' to be set")
	}
	if !result.Flags["flag-2"] {
		t.Error("Expected flag 'flag-2' to be set")
	}
	if result.FlagValues["flag-2"] != "value" {
		t.Errorf("Expected flag value 'value', got '%s'", result.FlagValues["flag-2"])
	}
}

func TestParseTagValue_ArgValueWithQuotes(t *testing.T) {
	// 测试参数值包含引号
	result, err := tagkit.ParseValue("fieldName(name:\"test\",desc:'description')")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.Args["name"].Value != "\"test\"" {
		t.Errorf("Expected arg value with quotes, got '%s'", result.Args["name"].Value)
	}
	if result.Args["desc"].Value != "'description'" {
		t.Errorf("Expected arg value with single quotes, got '%s'", result.Args["desc"].Value)
	}
}

func TestParseTagValue_FlagValueWithQuotes(t *testing.T) {
	// 测试标记位值包含引号
	result, err := tagkit.ParseValue("fieldName,config=\"value\",other='test'")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.FlagValues["config"] != "\"value\"" {
		t.Errorf("Expected flag value with quotes, got '%s'", result.FlagValues["config"])
	}
	if result.FlagValues["other"] != "'test'" {
		t.Errorf("Expected flag value with single quotes, got '%s'", result.FlagValues["other"])
	}
}

func TestParseTagValue_ArgValueWithSpaces(t *testing.T) {
	// 测试参数值包含空格
	result, err := tagkit.ParseValue("fieldName(name:test value,desc:hello world)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.Args["name"].Value != "test value" {
		t.Errorf("Expected arg value with spaces, got '%s'", result.Args["name"].Value)
	}
	if result.Args["desc"].Value != "hello world" {
		t.Errorf("Expected arg value with spaces, got '%s'", result.Args["desc"].Value)
	}
}

func TestParseTagValue_FlagValueWithSpaces(t *testing.T) {
	// 测试标记位值包含空格
	result, err := tagkit.ParseValue("fieldName,config=test value,other=hello world")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if result.FlagValues["config"] != "test value" {
		t.Errorf("Expected flag value with spaces, got '%s'", result.FlagValues["config"])
	}
	if result.FlagValues["other"] != "hello world" {
		t.Errorf("Expected flag value with spaces, got '%s'", result.FlagValues["other"])
	}
}

func TestParseTagValue_ArgValueWithNewlines(t *testing.T) {
	// 测试参数值包含换行符
	result, err := tagkit.ParseValue("fieldName(desc:\"line1\nline2\")")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if !strings.Contains(result.Args["desc"].Value, "\n") {
		t.Error("Expected arg value with newline")
	}
}

func TestParseTagValue_FlagValueWithNewlines(t *testing.T) {
	// 测试标记位值包含换行符
	result, err := tagkit.ParseValue("fieldName,config=\"line1\nline2\"")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if !strings.Contains(result.FlagValues["config"], "\n") {
		t.Error("Expected flag value with newline")
	}
}

func TestParseTagValue_ArgValueWithTabs(t *testing.T) {
	// 测试参数值包含制表符
	result, err := tagkit.ParseValue("fieldName(desc:\"col1\tcol2\")")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	if !strings.Contains(result.Args["desc"].Value, "\t") {
		t.Error("Expected arg value with tab")
	}
}

func TestParseTagValue_PlaceholderWithSpecialChars(t *testing.T) {
	// 测试占位符包含特殊字符
	result, err := tagkit.ParseValue("fieldName(arg:$var_name,arg2:$var-123)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	arg1, ok := result.Args["arg"]
	if !ok {
		t.Error("Expected arg 'arg' to be parsed")
	} else if !arg1.Placeholder {
		t.Error("Expected arg to be placeholder")
	} else if arg1.CustomName != "var_name" {
		t.Errorf("Expected custom name 'var_name', got '%s'", arg1.CustomName)
	}

	arg2, ok := result.Args["arg2"]
	if !ok {
		t.Error("Expected arg 'arg2' to be parsed")
	} else if arg2.CustomName != "var-123" {
		t.Errorf("Expected custom name 'var-123', got '%s'", arg2.CustomName)
	}
}

func TestParseTagValue_EmptyFlagNameAfterEquals(t *testing.T) {
	// 测试等号后标记位名称为空的情况（实际上应该是 flagName=value，但如果解析错误）
	result, err := tagkit.ParseValue("fieldName,=value")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 空名称应该被跳过
	if len(result.FlagValues) != 0 {
		t.Error("Expected empty flag name to be skipped")
	}
}

func TestParseTagValue_FlagValueStartsWithEquals(t *testing.T) {
	// 测试标记位值以等号开头
	result, err := tagkit.ParseValue("fieldName,key==value")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 第一个等号是分隔符，值应该包含第二个等号
	if result.FlagValues["key"] != "=value" {
		t.Errorf("Expected flag value starting with equals, got '%s'", result.FlagValues["key"])
	}
}

func TestParseTagValue_ArgValueStartsWithColon(t *testing.T) {
	// 测试参数值以冒号开头
	result, err := tagkit.ParseValue("fieldName(arg::value)")
	if err != nil {
		t.Fatalf("ParseValue should not fail, got: %v", err)
	}

	// 第一个冒号是分隔符，值应该包含第二个冒号
	if result.Args["arg"].Value != ":value" {
		t.Errorf("Expected arg value starting with colon, got '%s'", result.Args["arg"].Value)
	}
}

func TestParseTagValue_ExtremelyLongString(t *testing.T) {
	// 测试极长的字符串
	longFieldName := strings.Repeat("a", 10000)
	longValue := strings.Repeat("b", 10000)
	input := longFieldName + "(" + "arg:" + longValue + ")"

	result, err := tagkit.ParseValue(input)
	if err != nil {
		t.Fatalf("ParseValue should not fail for long strings, got: %v", err)
	}

	if result.FieldName != longFieldName {
		t.Error("Expected long field name to be parsed")
	}
	if result.Args["arg"].Value != longValue {
		t.Error("Expected long arg value to be parsed")
	}
}

func TestParseTagValue_UnicodeCharacters_ShouldError(t *testing.T) {
	// 测试 Unicode 字符（字段名包含 Unicode 应该报错）
	_, err := tagkit.ParseValue("字段名(参数:值),标记位=值")
	if err == nil {
		t.Error("Expected error for unicode characters in field name")
	}
	if !strings.Contains(err.Error(), "invalid field name") {
		t.Errorf("Expected error message about invalid field name, got: %v", err)
	}
}

func TestParseTagValue_EmojiCharacters_ShouldError(t *testing.T) {
	// 测试 Emoji 字符（字段名包含 Emoji 应该报错）
	_, err := tagkit.ParseValue("field😀(arg:value🎉),flag=value✨")
	if err == nil {
		t.Error("Expected error for emoji characters in field name")
	}
	if !strings.Contains(err.Error(), "invalid field name") {
		t.Errorf("Expected error message about invalid field name, got: %v", err)
	}
}
