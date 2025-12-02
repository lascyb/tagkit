package tag_value_parser

import (
	"testing"

	"github.com/lascyb/tagkit"
)

func TestParseTagValue_OnlyFieldName(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}
	if len(result.Args) != 0 {
		t.Errorf("Expected no args, got %d", len(result.Args))
	}
	if len(result.Flags) != 0 {
		t.Errorf("Expected no flags, got %d", len(result.Flags))
	}
}

func TestParseTagValue_FieldNameWithArgs(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName(first:10,after:$cursor)")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}
	if len(result.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(result.Args))
	}

	arg1, ok := result.Args["first"]
	if !ok {
		t.Error("Expected arg 'first' not found")
	} else if arg1.Value != "10" {
		t.Errorf("Expected arg 'first' value '10', got '%s'", arg1.Value)
	}

	arg2, ok := result.Args["after"]
	if !ok {
		t.Error("Expected arg 'after' not found")
	} else if arg2.Value != "$cursor" {
		t.Errorf("Expected arg 'after' value '$cursor', got '%s'", arg2.Value)
	} else if !arg2.Placeholder {
		t.Error("Expected arg 'after' to be placeholder")
	} else if arg2.CustomName != "cursor" {
		t.Errorf("Expected arg 'after' CustomName 'cursor', got '%s'", arg2.CustomName)
	}
}

func TestParseTagValue_OnlyFlags(t *testing.T) {
	result, err := tagkit.ParseValue(",inline,union")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "" {
		t.Errorf("Expected empty FieldName, got '%s'", result.FieldName)
	}
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' not set")
	}
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' not set")
	}
}

func TestParseTagValue_FieldNameAndFlags(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName,inline")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' not set")
	}
}

func TestParseTagValue_FullFormat(t *testing.T) {
	result, err := tagkit.ParseValue("name(arg:1,arg2:$var),inline,union")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "name" {
		t.Errorf("Expected FieldName 'name', got '%s'", result.FieldName)
	}
	if len(result.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(result.Args))
	}
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' not set")
	}
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' not set")
	}
}

func TestParseTagValue_FlagsWithValues(t *testing.T) {
	result, err := tagkit.ParseValue("name(arg:1),union=unionTypeName,handle=A|B|C")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "name" {
		t.Errorf("Expected FieldName 'name', got '%s'", result.FieldName)
	}
	if len(result.Args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(result.Args))
	}

	// 标记位只要存在就应该先标记为 true
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' to be true (flag exists)")
	}
	if !result.Flags["handle"] {
		t.Error("Expected flag 'handle' to be true (flag exists)")
	}

	// 然后再检查值是否存在
	if result.FlagValues["union"] != "unionTypeName" {
		t.Errorf("Expected FlagValues['union'] 'unionTypeName', got '%s'", result.FlagValues["union"])
	}
	if result.FlagValues["handle"] != "A|B|C" {
		t.Errorf("Expected FlagValues['handle'] 'A|B|C', got '%s'", result.FlagValues["handle"])
	}
}

func TestParseTagValue_MultipleFlags(t *testing.T) {
	// 注意：根据需求文档 3.6，如果只有标记位且无字段名，应该以逗号开头
	// 但这里测试的是 "inline,union,final" 这种格式，根据需求应该是字段名为空
	// 实际上，如果用户想要只有标记位，应该使用 ",inline,union,final"
	// 但为了兼容性，我们也可以支持 "inline,union,final" 作为只有标记位的情况
	// 这里我们假设第一个部分是字段名（为了保持一致性）
	// 如果用户想要只有标记位，应该使用 ",inline,union,final"

	// 测试以逗号开头的格式（只有标记位）
	result, err := tagkit.ParseValue(",inline,union,final")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "" {
		t.Errorf("Expected empty FieldName, got '%s'", result.FieldName)
	}
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' not set")
	}
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' not set")
	}
	if !result.Flags["final"] {
		t.Error("Expected flag 'final' not set")
	}
}

func TestParseTagValue_ComplexArgs(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName(first:10,after:$cursor,filter:{name:\"test\"})")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}
	if len(result.Args) < 2 {
		t.Errorf("Expected at least 2 args, got %d", len(result.Args))
	}
}

func TestParseTagValue_EmptyString(t *testing.T) {
	result, err := tagkit.ParseValue("")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "" {
		t.Errorf("Expected empty FieldName, got '%s'", result.FieldName)
	}
	if len(result.Args) != 0 {
		t.Errorf("Expected no args, got %d", len(result.Args))
	}
	if len(result.Flags) != 0 {
		t.Errorf("Expected no flags, got %d", len(result.Flags))
	}
}

func TestParseTagValue_WhitespaceOnly(t *testing.T) {
	result, err := tagkit.ParseValue("   ")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "" {
		t.Errorf("Expected empty FieldName, got '%s'", result.FieldName)
	}
}

func TestParseTagValue_EmptyArgs(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName()")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}
	if len(result.Args) != 0 {
		t.Errorf("Expected no args, got %d", len(result.Args))
	}
}

func TestParseTagValue_DuplicateArgNames(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName(arg:1,arg:2)")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if len(result.Args) != 1 {
		t.Errorf("Expected 1 arg (duplicate should be overwritten), got %d", len(result.Args))
	}
	if result.Args["arg"].Value != "2" {
		t.Errorf("Expected last arg value '2', got '%s'", result.Args["arg"].Value)
	}
}

func TestParseTagValue_PlaceholderDollar(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName(arg:$)")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	arg, ok := result.Args["arg"]
	if !ok {
		t.Error("Expected arg 'arg' not found")
	} else if !arg.Placeholder {
		t.Error("Expected arg to be placeholder")
	} else if arg.CustomName != "" {
		t.Errorf("Expected empty CustomName, got '%s'", arg.CustomName)
	}
}

func TestParseTagValue_UnmatchedParentheses(t *testing.T) {
	_, err := tagkit.ParseValue("fieldName(")
	if err == nil {
		t.Error("Expected error for unmatched parentheses")
	}
}

func TestParseTagValue_InvalidArgFormat(t *testing.T) {
	// 参数格式错误（缺少冒号），应该跳过该参数
	result, err := tagkit.ParseValue("fieldName(arg1,arg2:value)")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	// arg1 应该被跳过（格式错误）
	if _, ok := result.Args["arg1"]; ok {
		t.Error("Expected arg 'arg1' to be skipped due to invalid format")
	}
	// arg2 应该正常解析
	if _, ok := result.Args["arg2"]; !ok {
		t.Error("Expected arg 'arg2' to be parsed")
	}
}

func TestParseTagValue_EmptyArgValue(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName(arg:)")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	arg, ok := result.Args["arg"]
	if !ok {
		t.Error("Expected arg 'arg' not found")
	} else if arg.Value != "" {
		t.Errorf("Expected empty value, got '%s'", arg.Value)
	}
}

func TestParseTagValue_FlagWithEmptyValue(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName,union=,handle=value")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	// 标记位只要存在就应该先标记为 true
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' to be true (flag exists, even with empty value)")
	}
	if !result.Flags["handle"] {
		t.Error("Expected flag 'handle' to be true (flag exists)")
	}

	// union 标记位的值为空，不应该在 FlagValues 中
	if _, exists := result.FlagValues["union"]; exists {
		t.Error("Expected 'union' not in FlagValues (empty value should not be stored)")
	}

	// handle 标记位有值，应该在 FlagValues 中
	if result.FlagValues["handle"] != "value" {
		t.Errorf("Expected FlagValues['handle'] 'value', got '%s'", result.FlagValues["handle"])
	}
}

func TestParseTagValue_FlagExistsButNoValue(t *testing.T) {
	// 测试标记位存在但没有值的情况（只有 flagName=，没有值）
	result, err := tagkit.ParseValue("fieldName,union=")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	// 标记位存在，应该被设置为 true
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' to be true (flag exists)")
	}

	// 值不存在，不应该在 FlagValues 中
	if _, exists := result.FlagValues["union"]; exists {
		t.Error("Expected 'union' not in FlagValues (no value provided)")
	}
}

func TestParseTagValue_MixedFlags(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName,inline,union=unionTypeName,final")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	// 所有标记位都应该被设置为 true
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' to be true")
	}
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' to be true (flag exists)")
	}
	if !result.Flags["final"] {
		t.Error("Expected flag 'final' to be true")
	}

	// union 标记位有值，应该存储在 FlagValues 中
	if result.FlagValues["union"] != "unionTypeName" {
		t.Errorf("Expected FlagValues['union'] 'unionTypeName', got '%s'", result.FlagValues["union"])
	}

	// inline 和 final 没有值，不应该在 FlagValues 中
	if _, exists := result.FlagValues["inline"]; exists {
		t.Error("Expected 'inline' not in FlagValues (no value)")
	}
	if _, exists := result.FlagValues["final"]; exists {
		t.Error("Expected 'final' not in FlagValues (no value)")
	}
}

func TestParseTagValue_ComplexExample(t *testing.T) {
	result, err := tagkit.ParseValue("nodes(first:10,after:$cursor),inline,union=UserConnection,handle=A|B|C")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "nodes" {
		t.Errorf("Expected FieldName 'nodes', got '%s'", result.FieldName)
	}
	if len(result.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(result.Args))
	}

	// 所有标记位都应该被设置为 true
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' to be true")
	}
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' to be true (flag exists)")
	}
	if !result.Flags["handle"] {
		t.Error("Expected flag 'handle' to be true (flag exists)")
	}

	// 检查值是否正确
	if result.FlagValues["union"] != "UserConnection" {
		t.Errorf("Expected FlagValues['union'] 'UserConnection', got '%s'", result.FlagValues["union"])
	}
	if result.FlagValues["handle"] != "A|B|C" {
		t.Errorf("Expected FlagValues['handle'] 'A|B|C', got '%s'", result.FlagValues["handle"])
	}

	// inline 没有值，不应该在 FlagValues 中
	if _, exists := result.FlagValues["inline"]; exists {
		t.Error("Expected 'inline' not in FlagValues (no value)")
	}
}

func TestParseTagValue_WithSpaces(t *testing.T) {
	result, err := tagkit.ParseValue("  fieldName  ,  inline  ,  union  ")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}
	if !result.Flags["inline"] {
		t.Error("Expected flag 'inline' not set")
	}
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' not set")
	}
}

func TestParseTagValue_ArgsWithSpaces(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName( first : 10 , after : $cursor )")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if len(result.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(result.Args))
	}
	if result.Args["first"].Value != "10" {
		t.Errorf("Expected arg 'first' value '10', got '%s'", result.Args["first"].Value)
	}
	if result.Args["after"].Value != "$cursor" {
		t.Errorf("Expected arg 'after' value '$cursor', got '%s'", result.Args["after"].Value)
	}
}

func TestParseTagValue_FlagValueWithParentheses(t *testing.T) {
	result, err := tagkit.ParseValue("fieldName,handle=func(a,b),other=value")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "fieldName" {
		t.Errorf("Expected FieldName 'fieldName', got '%s'", result.FieldName)
	}

	// 所有标记位都应该被设置为 true
	if !result.Flags["handle"] {
		t.Error("Expected flag 'handle' to be true (flag exists)")
	}
	if !result.Flags["other"] {
		t.Error("Expected flag 'other' to be true (flag exists)")
	}

	// 检查值是否正确
	if result.FlagValues["handle"] != "func(a,b)" {
		t.Errorf("Expected FlagValues['handle'] 'func(a,b)', got '%s'", result.FlagValues["handle"])
	}
	if result.FlagValues["other"] != "value" {
		t.Errorf("Expected FlagValues['other'] 'value', got '%s'", result.FlagValues["other"])
	}
}

func TestParseTagValue_FlagValueWithComplexParentheses(t *testing.T) {
	result, err := tagkit.ParseValue("name(arg:1),handle=func(a,b,c),union=unionTypeName")
	if err != nil {
		t.Fatalf("ParseValue failed: %v", err)
	}

	if result.FieldName != "name" {
		t.Errorf("Expected FieldName 'name', got '%s'", result.FieldName)
	}
	if len(result.Args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(result.Args))
	}

	// 所有标记位都应该被设置为 true
	if !result.Flags["handle"] {
		t.Error("Expected flag 'handle' to be true (flag exists)")
	}
	if !result.Flags["union"] {
		t.Error("Expected flag 'union' to be true (flag exists)")
	}

	// 检查值是否正确
	if result.FlagValues["handle"] != "func(a,b,c)" {
		t.Errorf("Expected FlagValues['handle'] 'func(a,b,c)', got '%s'", result.FlagValues["handle"])
	}
	if result.FlagValues["union"] != "unionTypeName" {
		t.Errorf("Expected FlagValues['union'] 'unionTypeName', got '%s'", result.FlagValues["union"])
	}
}

// 辅助函数：检查两个 map 是否相等
func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestParseTagValue_AllCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected *tagkit.TagValue
		hasError bool
	}{
		{
			name:  "only field name",
			input: "fieldName",
			expected: &tagkit.TagValue{
				FieldName:  "fieldName",
				Args:       make(map[string]*tagkit.ArgMeta),
				Flags:      make(map[string]bool),
				FlagValues: make(map[string]string),
			},
			hasError: false,
		},
		{
			name:  "field name with args",
			input: "fieldName(arg:1)",
			expected: &tagkit.TagValue{
				FieldName: "fieldName",
				Args: map[string]*tagkit.ArgMeta{
					"arg": {Name: "arg", Value: "1", Placeholder: false, CustomName: ""},
				},
				Flags:      make(map[string]bool),
				FlagValues: make(map[string]string),
			},
			hasError: false,
		},
		{
			name:  "only flags",
			input: ",inline,union",
			expected: &tagkit.TagValue{
				FieldName: "",
				Args:      make(map[string]*tagkit.ArgMeta),
				Flags: map[string]bool{
					"inline": true,
					"union":  true,
				},
				FlagValues: make(map[string]string),
			},
			hasError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tagkit.ParseValue(tc.input)
			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.FieldName != tc.expected.FieldName {
				t.Errorf("FieldName: expected '%s', got '%s'", tc.expected.FieldName, result.FieldName)
			}
			if len(result.Args) != len(tc.expected.Args) {
				t.Errorf("Args count: expected %d, got %d", len(tc.expected.Args), len(result.Args))
			}
			if !mapsEqual(result.Flags, tc.expected.Flags) {
				t.Errorf("Flags: expected %v, got %v", tc.expected.Flags, result.Flags)
			}
		})
	}
}
