# tagkit

Go 库：解析 struct tag 中的「字段调用 + 标记位」语法，得到结构化结果，便于代码生成、校验或配置消费。

**版本**：v1.1.0（v1 全新重构后的首个功能版本，含行为变更，见 [行为变更](#行为变更v110) 与 [CHANGELOG](CHANGELOG.md)）

## 安装

```bash
go get github.com/lascyb/tagkit
```

## 语法概览

一条 tag value 由**至多一个字段调用**和**若干标记位**组成，逗号分隔：

- **字段**：`name(arg1:val1, ...)` 或匿名字段 `(arg1:val1,...)`，字段名可为空。无参字段可写 `name()` 或直接 `name`
  （无前导逗号时，裸标识符视为无参字段）。
- **标记位**：布尔标记 `name` 或键值 `name=value`。**存在赋值（name=value）时允许省略前导逗号**；仅布尔标记（裸 `name`）时需该标识符前有逗号才解析为标记位，故「仅布尔标记」时需前导逗号，如 `,inline`、`,verbose`；`limit=10` 可写于开头。
- **唯一性**：整条 tag 中只允许一个字段调用。
- **同名标记**：允许重复，按书写顺序保留，如 `directive='a',directive='b'`；`Flag(name)` 返回第一个，`FlagsNamed(name)` 返回全部。

### 逗号规则

| 位置           | 规则                                                                        |
|--------------|---------------------------------------------------------------------------|
| 顶层元素之间       | **必须**有逗号：`name(a:1),inline`；`a b`、`x=1 y=2`、`f(a:1)b` 均报错                  |
| 顶层前导/连续/末尾逗号 | 允许，视为空位：`,verbose`、`,,flag`、`name(),`                                       |
| 参数之间         | **必须**有逗号：`f(a:1 b:2)` 报错；允许末尾逗号 `f(a:1,)`；不允许前导或连续逗号 `f(,a:1)`、`f(a:1,,b:2)` |
| 数组元素之间       | **必须**有逗号：`[1 2]` 报错；允许末尾逗号 `[1,2,]`                                       |

空白（空格、制表符、换行、回车）可出现在任意 Token 之间。

### 参数值形式

| 形式            | 示例                                                                       |
|---------------|--------------------------------------------------------------------------|
| 字符串           | `label:'hello'`、`label:"hello"`、`s:'it\'s'`                                |
| 枚举（裸标识符）      | `sortKey:CREATED_AT`、`label:hello`                                        |
| 数字            | `age:18`、`n:-1`、`scale:1.5`、`big:1e6`、`r:2.5E-3`                           |
| 布尔 / null     | `enabled:true`、`off:false`、`cursor:null`                                  |
| 列表            | `ids:[1,2,3]`、`tags:[A,'b']`、`m:[[1,2],[3,4]]`、`e:[]`                       |
| 变量（类型与默认值均可选） | `$age:Int=18`, `$ids:[Int!]!=[1,2,3]`, `$name`（无类型）                        |
| 匿名变量          | `$:String`、`key:$`（变量名为空，`VariableDetail.IsAnonymous()` 为 true；不写类型时类型为空，不做推断） |
| 嵌套数组          | `[[1,2],[3,4]]`, `[[[1,2]],[[3,4]]]`                                     |

标记位的取值（`name=value`）可以是上表中除变量以外的任意字面量或数组。

### 字面量规则

- **标识符**（字段名、参数名、标记名、变量名、类型名、枚举值）：仅 ASCII，`[A-Za-z_][A-Za-z0-9_]*`（GraphQL Name 规则）。
  非 ASCII 字符（如 `中文`、`ä`）只能出现在字符串中，否则报 `unexpected character`。
- **字符串**：单引号或双引号包裹，可包含任意字符（含中文）。转义：`\\` → `\`、`\'` → `'`、`\"` → `"`；
  其余反斜杠序列（如 `\n`、`\t`）**原样保留**（反斜杠和后续字符都保留）。未闭合的字符串（包括以反斜杠结尾）报错。
- **数字**：遵循 GraphQL IntValue/FloatValue：`-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?`。
    - 支持负数 `-1`、小数 `1.5`、指数 `1e3`、`2.5E-3`；
    - 不支持 `.5`、`1.`、前导零 `007`、`+1`、`1_000`、`0x10`，均报错；
    - 不含小数点和指数的数字解析为 `int`，超出 `int` 范围报错（大整数 ID 请写成字符串 `'12345678901234567890'`）；
      其余解析为 `float64`，超出范围（如 `1e999`）报错。
- **true / false**：布尔值；不能用作字段名、参数名或标记名。
- **null**：空值，Go 值为 `nil`，`Kind` 为 `null`。
- **枚举**：其余裸标识符（如 `CREATED_AT`、`True`），Go 值为 `string`（与 v1.0.x 相同），`Kind` 为 `enum`，据此与带引号的字符串区分。
- **数组**：同一层的非 null 元素必须同为标量或同为数组；非空子数组的维度必须相同；所有非 null 标量必须属于同一类别
  （字符串与枚举同属一类、数字、布尔）。`null` 可出现在任意位置，如 `[1,null]`、`[[1],null]`。

### 变量默认值校验

同时声明类型与默认值时（`$v:Type=default`），默认值必须与类型相符，否则报错：

- `null` 只能用于可空类型：`$v:Int=null` 合法，`$v:Int!=null`、`$v:[Int!]=[1,null]` 报错；
- 列表类型的默认值必须是数组且维度一致（`[]` 可用于任意列表类型）：`$v:[[Int]]=[[1],[2]]` 合法，`$v:[Int]=1`、`$v:[Int]=[[1]]` 报错；
- 内置标量：`Int` 为 32 位整数，`Float` 为数字，`Boolean` 为布尔，`String` 为字符串或裸标识符，`ID` 为字符串、裸标识符或整数；
- 其他类型（自定义标量、枚举等）不校验标量取值，如 `$v:Sort=CREATED_AT`、`$v:[ID]=[1,'x']`。

未声明类型时（`$v=default`）只做上面的数组一致性校验。

### 示例

```
name(age:$age:Int=18, first:10),inline=3,union=false
tags(items:$items:[String]=[a,b,c], count:$count:Int=5),flatten
products(first:10, sortKey:CREATED_AT, query:'tag:new', after:null),alias=items
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
	input := `query(ids:$ids:[Int]=[1,2,3], sortKey:CREATED_AT, names:['foo','bar']),cache=true`
	result, err := tagkit.ParseTagValue(input)
	if err != nil {
		panic(err) // *tagkit.SyntaxError，含出错位置 Pos
	}

	// 唯一字段：名称与参数直接展开在 TagValue 上；OrderedArgs 按书写顺序遍历
	fmt.Println("字段名:", result.Name)
	for _, a := range result.OrderedArgs() {
		if a.Type == tagkit.ArgTypeVariable {
			fmt.Printf("  参数 %s = $%s (%s)", a.Name, a.VarName, a.VarType)
			if a.HasDefault {
				fmt.Printf(" 默认 %v", a.DefaultVal)
			}
			fmt.Println()
		} else {
			fmt.Printf("  参数 %s = %v (%s)\n", a.Name, a.Value, a.Kind) // Kind：string / enum / number / bool / null / list
		}
	}
	if f, ok := result.Flag("cache"); ok {
		fmt.Println("cache =", f.Value)
	}
	fmt.Println(result.String()) // 规范写法，可被重新解析
}
```

## API

### 解析入口

- **ParseTagValue**(input string) (\*TagValue, error)  
  解析整条 tag value；出错时返回 `*SyntaxError`（若存在多个字段调用也会报错）。

- **SyntaxError**：`Pos`（出错位置，输入中的字节偏移，从 0 开始）、`Msg`（错误描述）；`Error()` 为 `<Msg> at position <Pos>`，
  可用 `errors.As` 取得。

### 结果结构

- **TagValue**
    - `Name`：唯一字段名，无字段时为空。
    - `Args`：参数表，key 为参数名，value 为 `ArgValue`（字面量或变量）。存在字段调用时为非 nil map，否则为 nil。
    - `ArgKeys`：参数名，按书写顺序排列（v1.1.0）。
    - `Variables`：变量详情列表（类型、默认值、数组维度等），按参数书写顺序排列。
    - `Flags`：标记位列表，按书写顺序排列，总是非 nil。
    - 方法（v1.1.0）：`HasField()`、`Arg(name)`、`OrderedArgs()`、`Flag(name)`、`HasFlag(name)`、`FlagsNamed(name)`、`String()`。
      `String()` 输出规范写法（字段总带括号且在最前，每个标记位前带逗号，字符串用单引号），满足 `ParseTagValue(tv.String())` 与 `tv` 完全相等。

- **ArgValue**：单个参数的值
    - `Type`：`ArgTypeLiteral`（`"literal"`）或 `ArgTypeVariable`（`"variable"`）。
    - 字面量：`Value` 为解析后的 Go 值（`string`、`int`、`float64`、`bool`、`nil`、`[]interface{}`），
      `Kind` 为 `KindString` / `KindEnum` / `KindNumber` / `KindBool` / `KindNull` / `KindList`，`Literal` 为结构化表示，
      列表的 `Dimension` 为维度。
    - 变量：`Value` 为规范写法（如 `$age:Int=18`），`Kind` 为 `KindVariable`，`VarName`、`VarType`（未设置类型时为空）、
      `HasDefault`、`DefaultVal`、`Default`（结构化默认值）、`IsArrayDefault`、`Dimension` 等。

- **Literal**（v1.1.0）：`Kind`、`Value`（Go 值）、`Text`（规范写法，如 `'it\'s'`、`CREATED_AT`、`1.5e3`、`[A,'b']`）、
  `Elems`（列表元素，每个元素同样带 `Kind`，可区分列表中的字符串与枚举），方法 `Dimension()`。

- **VariableDetail**：变量的完整描述（名称、类型、默认值、数组维度、对应参数 key 等）。未设置类型时 `VarType` 为空字符串、
  `TypeStruct` 为 nil；`ArrayType` 为数组默认值的元素类别（`string` / `number` / `bool` / `null`，空数组为空串，
  `ID` 等类型混用整数与字符串时为 `mixed`），`Dimension` 为维度；`Default` 为结构化默认值（v1.1.0）；`IsAnonymous()` 判断匿名变量。

- **FlagInfo**：单个标记
    - `IsBoolean`：是否为裸标记（未写 `=value`，此时 `Value` 为 `true`）。注意 `,x=true` 的 `IsBoolean` 为 false；
      语义更清晰的写法是 `HasValue()`（v1.1.0）。
    - `Value` / `ValueType`：键值标记时的取值与类型（`ValueTypeNumber` / `ValueTypeString` / `ValueTypeBool` / `ValueTypeNull` /
      `ValueTypeArray`；枚举的 `ValueType` 仍为 `"string"`，用 `Kind` 区分）。
    - `Kind` / `Literal`（v1.1.0）：取值种类与结构化表示，裸标记的 `Kind` 为 `KindBool`、`Literal` 为 nil。

### core 包

`github.com/lascyb/tagkit/core` 暴露词法分析器、解析器与 AST（`Lexer`、`Parser`、`FieldNode` 等），供需要直接处理语法树的场景使用。
v1.1.0 新增：`FieldNode.ArgList`（有序参数）、`LiteralNode.Kind`、各节点的 `Pos`、`TypeNode.ListDepth()` / `BaseName()`、
`ArrayNode.Depth()`、`ArrayElemType`、`Unescape` / `Quote`、`SyntaxError`。
`TagValue.Variables[].TypeStruct` 引用了 `core.TypeNode`，因此 core 目前是公开 API 的一部分；计划在 v2 中将其移入 `internal/`，v1 内保持不变。

## 行为变更（v1.1.0）

以下写法在 v1.0.x 中被**静默接受（常常得到错误结果）**，v1.1.0 起返回 `*SyntaxError`：

- 无法识别的字符不再被跳过：`f(a:@#1)`、`f(a:1)%%,inline`、`,omit-empty` 报错（v1.0.x 分别得到 `1`、成功、两个元素）。
- 负数被正确解析：`f(n:-1)` 得到 `-1`（v1.0.x 得到 `1`）。
- 非法数字报错：`.5`、`1.`、`007`、`1_000`、`0x10`（v1.0.x 分别得到 `5`、`1`、`7` 或报出误导性的错误）；超出 `int` 范围的整数报错（v1.0.x 返回字符串）。
- 非 ASCII 标识符报错：`f(ä:1)`、`中文(a:1)`（v1.0.x 得到非法 UTF-8 键或误导性的错误）；字符串内的非 ASCII 字符不受影响。
- 元素、参数、数组元素之间必须有逗号：`f(a:1 b:2)`、`x=1 y=2`、`[1 2]` 报错。
- 重复的参数名报错：`f(a:1,a:2)`（v1.0.x 静默保留最后一个）。同名**标记位**仍然允许并全部保留。
- 类型写法错误报错：`$v:`、`$v:[Int`（v1.0.x 得到类型 `Unknown`）、`$v:[Int!!]`（v1.0.x 吞成 `[Int!]`）。
- 数组元素不一致报错：`[1,[2]]`、`['x',1]`、`[[1],[[2]]]`。
- 变量默认值与声明类型不符报错：`$v:Int='x'`、`$v:Int!=null`、`$v:[Int]=[[1]]`、`$v:Int=3000000000` 等。
- 未闭合的字符串（含以反斜杠结尾，如 `'\`、`'abc`）报错（v1.0.6 会 panic 或静默接受）。

以下是**结果变化**（不报错，但取值不同）：

- 转义被还原：`'it\'s'` 的值为 `it's`（v1.0.x 为 `it\'s`）；`\\`、`\"` 同理；其余反斜杠序列保持不变。
- 列表参数不再丢失：`f(ids:[1,2])` 的 `Args["ids"]` 为 `[]interface{}{1,2}`（v1.0.x 中该参数不存在）。
- `null` 的 Go 值为 `nil`，`FlagInfo.ValueType` 为 `"null"`（v1.0.x 为字符串 `"null"`、`ValueType` 为 `"string"`）。
- 数组元素中允许 `true` / `false` / `null`（v1.0.x 报错）。
- `Variables` 按参数书写顺序排列（v1.0.x 随机）；变量的 `Value`（Raw）对字符串默认值正确转义，可被重新解析。
- `VariableDetail.ArrayType` 统一为元素类别，不再随维度变化：二维 `[[1,2]]` 由 `number[]` 改为 `number`，空数组由 `[]` 改为空串；维度看 `Dimension`。
- 错误信息统一为 `<描述> at position <字节偏移>`，类型为 `*SyntaxError`（v1.0.x 混用 `pos` / `position` 的纯文本错误）。

## 要求

- Go 1.18+

## License

见仓库内 [LICENSE](LICENSE) 文件。
