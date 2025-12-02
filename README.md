## TagKit

一个 Go 语言结构体标签解析工具包，提供强大的结构体字段标签解析与结构化处理能力。

## 背景

TagKit 是在开发「将 Go 结构体自动转换为 GraphQL Schema / 查询」工具过程中抽离出来的通用 tag 解析库，
最初用于处理结构体字段上的 GraphQL 风格标签，但语法设计为通用格式，可复用于其他需要解析“字段+参数+标记位”风格标签的场景。

## 功能特性

- ✅ 解析字段名
- ✅ 解析参数列表（支持字面量和占位符）
- ✅ 解析布尔标记位
- ✅ 解析带值的标记位并结构化存储
- ✅ 支持嵌套括号和复杂值
- ✅ 完整的字段名验证
- ✅ 健壮的错误处理
- ✅ 统一的 TagValue 结构便于后续流程处理和编排

## 安装

```bash
go get github.com/lascyb/tagkit
```

## 快速开始

```go
package main

import (
    "fmt"
    "github.com/lascyb/tagkit"
)

func main() {
    // 解析完整的 tag 值
    result, err := tagkit.ParseValue("nodes(first:10,after:$cursor),inline,union=UserConnection")
    if err != nil {
        panic(err)
    }

    fmt.Printf("字段名: %s\n", result.FieldName)
    fmt.Printf("参数数量: %d\n", len(result.Args))
    fmt.Printf("标记位: %v\n", result.Flags)
}
```

## 使用示例

### 1. 只有字段名

```go
result, _ := tagkit.ParseValue("fieldName")
// result.FieldName = "fieldName"
// result.Args = {}
// result.Flags = {}
```

### 2. 字段名 + 参数

```go
result, _ := tagkit.ParseValue("fieldName(first:10,after:$cursor)")
// result.FieldName = "fieldName"
// result.Args["first"].Value = "10"
// result.Args["after"].Value = "$cursor"
// result.Args["after"].Placeholder = true
// result.Args["after"].CustomName = "cursor"
```

### 3. 字段名 + 标记位

```go
result, _ := tagkit.ParseValue("fieldName,inline,union")
// result.FieldName = "fieldName"
// slices.Contains(result.Flags, "inline") = true
// slices.Contains(result.Flags, "union") = true
```

### 4. 完整格式

```go
result, _ := tagkit.ParseValue("name(arg:1,arg2:$var),inline,union=unionTypeName,handle=A|B|C")
// result.FieldName = "name"
// result.Args["arg"].Value = "1"
// result.Args["arg2"].Value = "$var"
// slices.Contains(result.Flags, "inline") = true
// slices.Contains(result.Flags, "union") = true
// result.FlagValues["union"] = "unionTypeName"
// result.FlagValues["handle"] = "A|B|C"
```

### 5. 只有标记位

```go
result, _ := tagkit.ParseValue(",inline,union")
// result.FieldName = ""
// slices.Contains(result.Flags, "inline") = true
// slices.Contains(result.Flags, "union") = true
```

### 6. 复杂参数值

```go
result, _ := tagkit.ParseValue("fieldName(filter:{name:\"test\",age:18})")
// result.FieldName = "fieldName"
// result.Args["filter"].Value = "{name:\"test\",age:18}"
```

### 7. 标记位值包含括号

```go
result, _ := tagkit.ParseValue("fieldName,handle=func(a,b,c),other=value")
// slices.Contains(result.Flags, "handle") = true
// result.FlagValues["handle"] = "func(a,b,c)"
// result.FlagValues["other"] = "value"
```

## API 文档

### TagValue

解析结果结构体：

```go
type TagValue struct {
    FieldName  string              // 字段名（如果为空，表示使用默认字段名）
    Args       map[string]*ArgMeta // 参数列表（key: 参数名）
    Flags      []string            // 布尔标记位列表
    FlagValues map[string]string   // 带值的标记位（key: 标记位名称, value: 标记位的值）
}
```

### ArgMeta

参数元数据结构体：

```go
type ArgMeta struct {
    Name        string // 参数名（如 "first", "end2", "arg"）
    Value       string // 参数值（字面量，如 "1", "true"）
    Placeholder bool   // 是否为占位符（$ 开头）
    CustomName  string // 自定义变量名（如果指定了 $arg1，则为 "arg1"；如果为 $，则为空）
}
```

### ParseValue

解析 tag 值字符串：

```go
func ParseValue(value string) (*TagValue, error)
```

**参数**:
- `value`: tag 的值字符串（如 `"name(arg:1,arg2:$var),inline,union=unionTypeName"`）

**返回**:
- `*TagValue`: 解析结果
- `error`: 错误信息（如果解析失败）

**使用示例**:
```go
import "slices"

result, _ := tagkit.ParseValue("fieldName,inline,union")
if slices.Contains(result.Flags, "inline") {
    // 处理 inline 标记位
}
```

## 语法规则

### 语法格式

TagKit 支持的完整语法格式如下：

```
[字段名[(参数名:参数值,参数名:参数值,...)]] [,标记位1[=标记位值]][,标记位2[=标记位值]...]
```

### 字段名

- 允许的字符：字母（A-Z, a-z）、数字（0-9）、下划线（_）、中划线（-）
- 不允许包含空格、点号、特殊符号等

### 参数格式

- 格式：`参数名:参数值`
- 多个参数用逗号分隔：`arg1:value1,arg2:value2`
- 支持占位符：
  - `$var` - 自定义变量名
  - `$` - 默认占位符（CustomName 为空）

### 标记位格式

- 布尔标记位：`flagName`
- 带值标记位：`flagName=value`
- 多个标记位用逗号分隔：`inline,union=unionTypeName`

### 完整语法

```
tagValue := fieldName? (args)? (flags)?
fieldName := [A-Za-z0-9_-]+
args := '(' arg (',' arg)* ')'
arg := name ':' value
name := [A-Za-z0-9_-]+
value := 任意字符串（支持嵌套括号）
flags := ',' flag (',' flag)*
flag := flagName ('=' flagValue)?
flagName := [A-Za-z0-9_-]+
flagValue := 任意字符串（支持嵌套括号和逗号）
```

## 错误处理

解析器会在以下情况返回错误：

- 未匹配的括号：`fieldName(` 或 `fieldName(arg:1))`
- 括号前字段名为空：`(arg:1)`
- 字段名包含非法字符：`field name`（包含空格）

## 测试

运行测试：

```bash
go test ./test/...
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

