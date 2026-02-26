# tagkit

Go 库：解析 struct tag 中的「字段调用 + 标记位」语法，得到结构化结果，便于代码生成、校验或配置消费。

**版本**：v1（全新重构）

## 安装

```bash
go get github.com/lascyb/tagkit
```

## 语法概览

一条 tag value 由**至多一个字段调用**和**若干标记位**组成，逗号分隔：

- **字段**：`name(arg1:val1, ...)` 或匿名字段 `(arg1:val1,...)`，字段名可为空。无参字段可写 `name()` 或直接 `name`
  （无前导逗号时，裸标识符视为无参字段）。
- **标记位**：布尔标记 `name` 或键值 `name=value`。**存在赋值（name=value）时允许省略前导逗号**；仅布尔标记（裸 `name`）时需该标识符前有逗号才解析为标记位，故「仅布尔标记」时需前导逗号，如 `,inline`、`,verbose`；`limit=10` 可写于开头。
- **唯一性**：整条 tag 中只允许一个字段调用；字段与标记之间、标记与标记之间逗号不可省略。

### 参数值形式

| 形式            | 示例                                                                 |
|---------------|--------------------------------------------------------------------|
| 字面量           | `age:18`, `scale:1.5`, `enabled:true`, `label:hello`               |
| 变量（类型与默认值均可选） | `$age:Int=18`, `$ids:[Int!]!=[1,2,3]`, `$name`（无类型）, `key:$`（匿名变量） |
| 匿名变量          | `$:String`、`key:$`（仅 `$`，变量名为空；类型不写时默认推断）                          |
| 字符串数组         | `['a','b','c']` 或 `[a,b,c]`（有单引号以单引号为准，无则按逗号分割）                    |
| 嵌套数组          | `[[1,2],[3,4]]`, `[[[1,2]],[[3,4]]]`                               |

### 示例

```
name(age:$age:Int=18, first:10),inline=3,union=false
tags(items:$items:[String]=[a,b,c], count:$count:Int=5),flatten
(age:18,sex:$:String),inline
single2
limit=10
,verbose
,flag=false
```

## 用法

```go
package main

import (
	"fmt"
	"github.com/lascyb/tagkit"
)

func main() {
	input := `query(ids:$ids:[Int]=[1,2,3], names:$names:[String]=['foo','bar']),cache=true`
	result, err := tagkit.ParseTagValue(input)
	if err != nil {
		panic(err)
	}

	// 唯一字段：名称与参数直接展开在 TagValue 上
	fmt.Println("字段名:", result.Name)
	for k, v := range result.Args {
		if v.Type == "variable" {
			fmt.Printf("  参数 %s = $%s (%s)", k, v.VarName, v.VarType)
			if v.HasDefault {
				fmt.Printf(" 默认 %v", v.DefaultVal)
			}
			fmt.Println()
		} else {
			fmt.Printf("  参数 %s = %v\n", k, v.Value)
		}
	}
	for _, f := range result.Flags {
		if f.IsBoolean {
			fmt.Printf("标记 %s = true\n", f.Name)
		} else {
			fmt.Printf("标记 %s = %v\n", f.Name, f.Value)
		}
	}
}
```

## API

### 解析入口

- **ParseTagValue**(input string) (\*TagValue, error)  
  解析整条 tag value；若存在多个字段调用则返回错误。

### 结果结构

- **TagValue**
    - `Name`：唯一字段名，无字段时为空。
    - `Args`：参数表，key 为参数名，value 为 `ArgValue`（字面量或变量）。
    - `Variables`：变量详情列表（类型、默认值、数组维度等），便于代码生成或校验。
    - `Flags`：标记位列表。

- **ArgValue**：单个参数的值
    - `Type`：`"literal"` 或 `"variable"`。
    - 字面量：`Value` 为解析后的 Go 值（int、float64、bool、string 等）。
    - 变量：`VarName`、`VarType`（未设置类型时为空）、`HasDefault`、`DefaultVal`、`IsArrayDefault`、`Dimension` 等。

- **VariableDetail**：变量的完整描述（名称、类型、默认值、数组维度、对应参数 key 等）。未设置类型时 `VarType` 为空字符串、
  `TypeStruct` 为 nil。

- **FlagInfo**：单个标记
    - `IsBoolean`：是否为布尔标记（无 `=value`）。
    - `Value` / `ValueType`：键值标记时的取值与类型。

## 要求

- Go 1.21+

## License

见仓库内 [LICENSE](LICENSE) 文件。
