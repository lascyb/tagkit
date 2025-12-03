# 链式调用

TagKit 的 Parser 方法支持链式调用，可以让代码更加简洁和易读。

## 基本用法

所有 `Set` 方法都返回 `*Parser` 实例，支持链式调用：

```go
parser := tagkit.NewParser().
    SetFieldNameValidatorByRegex(regexp.MustCompile(`^[a-z_]+$`))

result, err := parser.ParseValue("field_name")
```

## 完整链式调用示例

可以在一条语句中完成创建、配置和解析：

```go
result, err := tagkit.NewParser().
    SetFieldNameValidatorByRegex(regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)).
    ParseValue("FieldName")
```

## 多个配置的链式调用

如果需要设置多个配置，可以继续链式调用：

```go
// 注意：当前版本只有一个配置项，但未来扩展时可以这样使用
parser := tagkit.NewParser().
    SetFieldNameValidatorByRegex(regexp.MustCompile(`^[a-z_]+$`))

// 然后使用解析器
result, err := parser.ParseValue("field_name")
```

## 与全局函数结合

全局的 `Set` 函数也支持链式调用：

```go
// 配置全局默认解析器并立即使用
result, err := tagkit.SetFieldNameValidatorByRegex(
    regexp.MustCompile(`^[a-z_]+$`),
).ParseValue("field_name")
```

## 实际应用场景

### 场景 1: 临时使用特定验证规则

```go
// 创建一个临时解析器，使用特定的验证规则
result, err := tagkit.NewParser().
    SetFieldNameValidatorByRegex(regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)).
    ParseValue("FieldName")
```

### 场景 2: 批量解析不同规则的数据

```go
// 解析 PascalCase 格式的字段
pascalParser := tagkit.NewParser().
    SetFieldNameValidatorByRegex(regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`))

// 解析 snake_case 格式的字段
snakeParser := tagkit.NewParser().
    SetFieldNameValidatorByRegex(regexp.MustCompile(`^[a-z_]+$`))

// 使用不同的解析器处理不同的数据
pascalResult, _ := pascalParser.ParseValue("FieldName")
snakeResult, _ := snakeParser.ParseValue("field_name")
```

### 场景 3: 函数式编程风格

```go
func parseWithValidator(pattern string, value string) (*tagkit.TagValue, error) {
    return tagkit.NewParser().
        SetFieldNameValidatorByRegex(regexp.MustCompile(pattern)).
        ParseValue(value)
}

// 使用
result, err := parseWithValidator(`^[a-z_]+$`, "field_name")
```

## 注意事项

1. **链式调用不会修改原解析器**：每次调用 `Set` 方法都会返回同一个解析器实例，但配置会应用到该实例上。

2. **链式调用顺序**：配置方法必须在 `ParseValue` 之前调用。

3. **错误处理**：如果链式调用中的任何一步返回错误，后续的调用不会执行。

## 完整示例

```go
package main

import (
    "fmt"
    "regexp"
    "github.com/lascyb/tagkit"
)

func main() {
    // 方式1: 分步链式调用
    parser := tagkit.NewParser().
        SetFieldNameValidatorByRegex(regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`))
    
    result1, err := parser.ParseValue("FieldName")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Result: %s\n", result1.FieldName)
    }
    
    // 方式2: 完整链式调用
    result2, err := tagkit.NewParser().
        SetFieldNameValidatorByRegex(regexp.MustCompile(`^[a-z_]+$`)).
        ParseValue("field_name")
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Result: %s\n", result2.FieldName)
    }
}
```

