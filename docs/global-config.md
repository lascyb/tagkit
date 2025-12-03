# 全局配置默认解析器

TagKit 提供了全局函数来配置默认解析器，这样可以在应用启动时统一设置验证规则，之后所有使用全局 `ParseValue` 函数的地方都会应用这些规则。

## 基本用法

### 设置验证规则

使用 `SetFieldNameValidatorByRegex` 为全局默认解析器设置验证规则：

```go
import (
    "regexp"
    "github.com/lascyb/tagkit"
)

// 在应用启动时设置全局验证规则
func init() {
    regex := regexp.MustCompile(`^[a-z_]+$`)
    tagkit.SetFieldNameValidatorByRegex(regex)
}

// 之后所有使用全局 ParseValue 的地方都会应用这个规则
func main() {
    result, err := tagkit.ParseValue("field_name")
    // ...
}
```

### 设置自定义验证函数

使用 `SetFieldNameValidator` 设置自定义验证函数：

```go
func init() {
    tagkit.SetFieldNameValidator(func(fieldName string) error {
        if len(fieldName) < 3 {
            return fmt.Errorf("field name must be at least 3 characters")
        }
        return nil
    })
}
```

## 重置为默认规则

将验证器设置为 `nil` 可以恢复默认验证规则：

```go
// 恢复默认验证规则
tagkit.SetFieldNameValidator(nil)
```

## 使用场景

### 场景 1: 应用级统一配置

在应用启动时统一配置验证规则，确保整个应用使用一致的验证标准：

```go
package main

import (
    "log"
    "regexp"
    "github.com/lascyb/tagkit"
)

func init() {
    // 根据环境或配置设置验证规则
    if isProduction() {
        // 生产环境：严格验证
        regex := regexp.MustCompile(`^[a-z_]+$`)
        tagkit.SetFieldNameValidatorByRegex(regex)
    } else {
        // 开发环境：允许更多字符
        regex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
        tagkit.SetFieldNameValidatorByRegex(regex)
    }
}

func main() {
    // 所有使用 tagkit.ParseValue 的地方都会应用上面的规则
    result, err := tagkit.ParseValue("field_name")
    if err != nil {
        log.Fatal(err)
    }
    // ...
}
```

### 场景 2: 动态切换验证规则

根据运行时条件动态切换验证规则：

```go
func switchToStrictMode() {
    regex := regexp.MustCompile(`^[a-z_]+$`)
    tagkit.SetFieldNameValidatorByRegex(regex)
}

func switchToRelaxedMode() {
    regex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
    tagkit.SetFieldNameValidatorByRegex(regex)
}

func resetToDefault() {
    tagkit.SetFieldNameValidator(nil)
}
```

### 场景 3: 测试环境配置

在测试中临时修改全局配置：

```go
func TestWithCustomValidator(t *testing.T) {
    // 保存原始配置
    originalValidator := getCurrentValidator() // 需要自己实现获取当前验证器的方法
    
    // 设置测试用的验证规则
    tagkit.SetFieldNameValidator(func(fieldName string) error {
        // 测试用的宽松规则
        return nil
    })
    
    // 执行测试
    result, err := tagkit.ParseValue("test_field")
    // ...
    
    // 恢复原始配置
    tagkit.SetFieldNameValidator(originalValidator)
}
```

## 与实例解析器的区别

| 特性 | 全局配置 | 实例解析器 |
|------|---------|-----------|
| 作用范围 | 全局，影响所有 `ParseValue()` 调用 | 仅影响该实例的调用 |
| 配置方式 | `SetFieldNameValidator()` | `parser.SetFieldNameValidator()` |
| 使用场景 | 应用级统一配置 | 特定场景的独立配置 |
| 线程安全 | 需要注意并发访问 | 每个实例独立，更安全 |

## 注意事项

1. **线程安全**：如果在多 goroutine 环境下频繁修改全局配置，建议使用实例解析器或添加同步机制。

2. **配置影响范围**：全局配置会影响所有使用 `tagkit.ParseValue()` 的代码，修改前需要谨慎考虑。

3. **测试隔离**：在测试中修改全局配置可能会影响其他测试，建议使用实例解析器或确保测试后恢复配置。

## 完整示例

```go
package main

import (
    "fmt"
    "regexp"
    "github.com/lascyb/tagkit"
)

func main() {
    // 设置全局验证规则：只允许小写字母和下划线
    regex := regexp.MustCompile(`^[a-z_]+$`)
    tagkit.SetFieldNameValidatorByRegex(regex)
    
    // 使用全局 ParseValue 函数
    result, err := tagkit.ParseValue("field_name")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Success: %s\n", result.FieldName)
    }
    
    // 尝试解析无效的字段名
    _, err = tagkit.ParseValue("FieldName")
    if err != nil {
        fmt.Printf("Expected error: %v\n", err)
    }
    
    // 重置为默认验证规则
    tagkit.SetFieldNameValidator(nil)
    
    // 现在可以使用默认规则
    result2, err := tagkit.ParseValue("FieldName123")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Success with default: %s\n", result2.FieldName)
    }
}
```

