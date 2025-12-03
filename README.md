# TagKit

一个 Go 语言结构体标签解析工具包，提供强大的结构体字段标签解析与结构化处理能力。

## 背景

TagKit 是在开发「将 Go 结构体自动转换为 GraphQL Schema / 查询」工具过程中抽离出来的通用 tag 解析库，最初用于处理结构体字段上的 GraphQL 风格标签，但语法设计为通用格式，可复用于其他需要解析"字段+参数+标记位"风格标签的场景。

## 快速开始

### 安装

```bash
go get github.com/lascyb/tagkit
```

### 基本使用

```go
package main

import (
    "fmt"
    "reflect"
    "github.com/lascyb/tagkit"
)

type User struct {
    Nodes []Node `graphql:"nodes(first:10,after:$cursor),inline,union=UserConnection"`
    // graphql 是 tag 的名称，可以自定义
}

type Node struct {
    ID int
}

func main() {
    // 从结构体标签获取 tag 值
    t := reflect.TypeOf(User{})
    field, _ := t.FieldByName("Nodes")
    tagValue := field.Tag.Get("graphql") // nodes(first:10,after:$cursor),inline,union=UserConnection
    
    // 解析 tag 值
    result, err := tagkit.ParseValue(tagValue)
    if err != nil {
        panic(err)
    }

    fmt.Printf("字段名: %s\n", result.FieldName)
    fmt.Printf("参数: %v\n", result.Args)
    fmt.Printf("标记位: %v\n", result.Flags)
}
```

## 使用示例

### 基础用法

```go
// 1. 只有字段名
result, _ := tagkit.ParseValue("fieldName")
// result.FieldName = "fieldName"

// 2. 字段名 + 参数
result, _ := tagkit.ParseValue("fieldName(first:10,after:$cursor)")
// result.Args["first"].Value = "10"
// result.Args["after"].Placeholder = true

// 3. 字段名 + 标记位
result, _ := tagkit.ParseValue("fieldName,inline,union")
// slices.Contains(result.Flags, "inline") = true

// 4. 完整格式
result, _ := tagkit.ParseValue("name(arg:1),inline,union=unionTypeName")
// result.FieldName = "name"
// result.FlagValues["union"] = "unionTypeName"

// 5. 只有标记位
result, _ := tagkit.ParseValue(",inline,union")
// result.FieldName = ""
```

### 高级用法

更多高级用法和详细示例，请参考 [文档目录](./docs/)：

- [自定义解析器](./docs/custom-parser.md) - 创建和配置自定义解析器
- [链式调用](./docs/method-chaining.md) - 使用链式调用简化代码
- [全局配置](./docs/global-config.md) - 配置全局默认解析器

**快速示例**:

```go
// 自定义解析器
parser := tagkit.NewParser()
parser.SetFieldNameValidatorByRegex(regexp.MustCompile(`^[a-z_]+$`))
result, _ := parser.ParseValue("field_name")

// 链式调用
result, _ := tagkit.NewParser().
    SetFieldNameValidatorByRegex(regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)).
    ParseValue("FieldName")
```

## 核心类型

```go
// 解析结果
type TagValue struct {
    FieldName  string            // 字段名
    Args       map[string]*Arg   // 参数列表
    Flags      []string          // 布尔标记位列表
    FlagValues map[string]string // 带值的标记位
}

// 参数元数据
type Arg struct {
    Name        string // 参数名
    Value       string // 参数值
    Placeholder bool   // 是否为占位符（$ 开头）
    CustomName  string // 自定义变量名
}
```

## 语法规则

TagKit 支持的语法格式：

```
[字段名[(参数名:参数值,...)]] [,标记位1[=标记位值]][,标记位2[=标记位值]...]
```

**字段名**: 默认允许字母、数字、下划线、中划线（可通过自定义验证器修改）

**参数格式**: `参数名:参数值`，多个参数用逗号分隔

**标记位格式**:
- 布尔标记位：`flagName`
- 带值标记位：`flagName=value`

**占位符**: `$var` 或 `$`

详细语法说明请参考 [完整文档](./docs/README.md)。

## API 参考

### 全局函数

```go
// 解析 tag 值（使用默认解析器）
func ParseValue(value string) (*TagValue, error)

// 配置默认解析器
func SetFieldNameValidator(validator FieldNameValidator) *Parser
func SetFieldNameValidatorByRegex(regex *regexp.Regexp) *Parser
```

### Parser 方法

```go
// 创建解析器
parser := tagkit.NewParser()

// 设置验证器
parser.SetFieldNameValidator(validator) *Parser
parser.SetFieldNameValidatorByRegex(regex) *Parser

// 解析
parser.ParseValue(value) (*TagValue, error)
```

完整 API 文档请参考 [文档目录](./docs/README.md)。

## 错误处理

解析器会在以下情况返回错误：

- 未匹配的括号
- 括号前字段名为空
- 字段名包含非法字符
- 标记位存在等号但名称为空

## 更多文档

- [文档目录](./docs/README.md) - 详细的使用文档和示例
- [自定义解析器](./docs/custom-parser.md) - 创建自定义解析器
- [链式调用](./docs/method-chaining.md) - 链式调用用法
- [全局配置](./docs/global-config.md) - 全局配置说明

## 测试

```bash
go test ./test/...
```

## 许可证

[MIT License](LICENSE) © [lascyb](https://github.com/lascyb)

## 贡献

欢迎提交 Issue 和 Pull Request！
