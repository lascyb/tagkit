# 自定义解析器实例

TagKit 支持创建自定义解析器实例，可以为不同的使用场景配置不同的验证规则。

## 创建解析器

使用 `NewParser()` 创建新的解析器实例：

```go
parser := tagkit.NewParser()
```

## 设置自定义验证函数

通过 `SetFieldNameValidator` 方法设置自定义验证函数：

```go
parser := tagkit.NewParser()

parser.SetFieldNameValidator(func(fieldName string) error {
    if fieldName == "" {
        return nil // 空字段名是允许的
    }
    
    // 自定义验证逻辑
    if len(fieldName) < 3 {
        return fmt.Errorf("field name '%s' must be at least 3 characters", fieldName)
    }
    
    // 禁止某些关键字
    if fieldName == "class" || fieldName == "type" {
        return fmt.Errorf("field name '%s' is a reserved keyword", fieldName)
    }
    
    return nil
})

// 使用解析器
result, err := parser.ParseValue("fieldName")
```

## 使用正则表达式验证

通过 `SetFieldNameValidatorByRegex` 方法使用正则表达式设置验证规则：

```go
parser := tagkit.NewParser()

// 只允许小写字母和下划线
regex := regexp.MustCompile(`^[a-z_]+$`)
parser.SetFieldNameValidatorByRegex(regex)

// 使用解析器
result, err := parser.ParseValue("field_name")
if err != nil {
    // 如果字段名包含大写字母或数字，会返回错误
    log.Fatal(err)
}
```

## 常见验证规则示例

### 只允许小写字母和下划线

```go
regex := regexp.MustCompile(`^[a-z_]+$`)
parser.SetFieldNameValidatorByRegex(regex)
```

### 首字母必须大写（PascalCase）

```go
regex := regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
parser.SetFieldNameValidatorByRegex(regex)
```

### 首字母必须小写（camelCase）

```go
regex := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
parser.SetFieldNameValidatorByRegex(regex)
```

### 允许中文字段名

```go
parser.SetFieldNameValidator(func(fieldName string) error {
    if fieldName == "" {
        return nil
    }
    // 允许中文、字母、数字、下划线、中划线
    matched, _ := regexp.MatchString(`^[\p{Han}a-zA-Z0-9_-]+$`, fieldName)
    if !matched {
        return fmt.Errorf("invalid field name: %s", fieldName)
    }
    return nil
})
```

## 重置验证规则

将验证器设置为 `nil` 可以恢复默认验证规则：

```go
parser.SetFieldNameValidator(nil)
```

## 完整示例

```go
package main

import (
    "fmt"
    "regexp"
    "github.com/lascyb/tagkit"
)

func main() {
    // 创建解析器并设置验证规则
    parser := tagkit.NewParser()
    
    // 设置只允许小写字母和下划线的验证规则
    regex := regexp.MustCompile(`^[a-z_]+$`)
    parser.SetFieldNameValidatorByRegex(regex)
    
    // 解析有效的字段名
    result, err := parser.ParseValue("field_name")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Printf("FieldName: %s\n", result.FieldName)
    
    // 尝试解析无效的字段名（包含大写字母）
    _, err = parser.ParseValue("FieldName")
    if err != nil {
        fmt.Printf("Expected error: %v\n", err)
    }
}
```

