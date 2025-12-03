# TagKit 高级用法文档

本文档包含 TagKit 的高级用法和详细示例。

## 文档索引

- [自定义解析器实例](./custom-parser.md) - 如何创建和配置自定义解析器
- [链式调用](./method-chaining.md) - 使用链式调用简化代码
- [全局配置](./global-config.md) - 配置全局默认解析器

## 快速导航

### 自定义解析器

如果你需要为不同的场景使用不同的验证规则，可以创建自定义解析器实例：

```go
parser := tagkit.NewParser()
parser.SetFieldNameValidatorByRegex(regexp.MustCompile(`^[a-z_]+$`))
```

详细说明请参考 [自定义解析器实例](./custom-parser.md)。

### 链式调用

TagKit 支持链式调用，让代码更简洁：

```go
result, err := tagkit.NewParser().
SetFieldNameValidatorByRegex(regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)).
ParseValue("FieldName")
```

详细说明请参考 [链式调用](./method-chaining.md)。

### 全局配置

在应用启动时统一配置验证规则：

```go
tagkit.SetFieldNameValidatorByRegex(regexp.MustCompile(`^[a-z_]+$`))
```

详细说明请参考 [全局配置](./global-config.md)。

## 选择使用场景

| 场景             |     推荐方案      |
|:---------------|:-------------:|
| 应用级统一验证规则      |     全局配置      |
| 不同场景需要不同规则     |   自定义解析器实例    |
| 临时使用特定规则       | 链式调用 + 自定义解析器 |
| 多 goroutine 环境 | 自定义解析器实例（更安全） |

