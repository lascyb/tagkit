package tagkit

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

// FieldNameValidator 字段名验证函数类型
// fieldName: 待验证的字段名
// 返回: 如果验证失败返回错误，否则返回 nil
type FieldNameValidator func(fieldName string) error
