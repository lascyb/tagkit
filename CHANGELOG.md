# Changelog

## v1.1.1（2026-11-04）

- 文档：CHANGELOG 中 v1.1.0 的标题由「未发布」改为发布日期。代码与 v1.1.0 相同。

## v1.1.0（2026-11-04）

本版本修复了 v1.0.x 对非法输入过于宽容、会静默改值的问题，并补充有序参数、值种类等 API。部分以前能通过的写法现在会报错，
完整列表见 README 的「行为变更（v1.1.0）」一节，以下为摘要。

### 修复

- 词法分析器遇到以反斜杠结尾的未闭合字符串（如 `'\`）时 panic，现返回 `unterminated string` 错误（PR #1）。
- 无法识别的字符不再被静默跳过（`f(n:-1)` 曾得到 `1`、`f(a:.5)` 曾得到 `5`）。
- 列表字面量参数 `f(ids:[1,2])` 不再丢失。
- 参数、变量顺序与 `String()` 输出不再随机。
- 裸标识符（枚举）、带引号字符串与 `null` 可以区分。
- 字符串转义 `\\` `\'` `\"` 被还原；`Raw` / `String()` 输出可被重新解析（Parse → String → Parse 结果相同）。
- 标识符限定 ASCII，不再产生非法 UTF-8 键。
- 重复参数报错；类型写法错误报错（不再得到 `Unknown`，`[Int!!]` 不再被吞成 `[Int!]`）。
- 数组元素支持 `true` / `false` / `null`；数组维度与元素类别一致性校验。
- 变量默认值按声明类型校验（维度、可空性、内置标量）。
- 数字遵循 GraphQL 规则：支持负数与指数，`.5`、`1.`、`007`、超出 `int` 范围的整数报错。
- 元素、参数、数组元素之间缺少逗号时报错，错误位置指向出错的 Token。

### 新增 API（均为增量，原有字段与语义保持不变）

- `TagValue.ArgKeys`、`TagValue.OrderedArgs()`、`TagValue.Arg()`、`TagValue.HasField()`、`TagValue.String()`。
- `TagValue.Flag()`、`TagValue.HasFlag()`、`TagValue.FlagsNamed()`、`FlagInfo.HasValue()`、`VariableDetail.IsAnonymous()`。
- `ValueKind` 与 `KindString` / `KindEnum` / `KindNumber` / `KindBool` / `KindNull` / `KindList` / `KindVariable`；
  `ArgValue.Kind` / `Literal` / `Default`、`FlagInfo.Kind` / `Literal`、`VariableDetail.Default`。
- `Literal`（结构化字面量，列表元素保留各自种类）。
- 常量 `ArgTypeLiteral`、`ArgTypeVariable`、`ValueTypeNumber`、`ValueTypeString`、`ValueTypeBool`、`ValueTypeNull`、`ValueTypeArray`。
- `SyntaxError{Pos, Msg}`（`tagkit.SyntaxError` 为 `core.SyntaxError` 的别名），所有解析错误均为该类型。
- core：`FieldNode.ArgList`、`ArgNode`、`LiteralNode.Kind`、`LiteralKind`、节点 `Pos`、`TypeNode.ListDepth()` / `BaseName()`、
  `ArrayNode.Depth()`、`ArrayElemType()`、`Unescape()`、`Quote()`。

### 行为变更

- 以前被静默接受的非法输入现在报错：非法字符、非 ASCII 标识符、非法数字、缺少逗号、重复参数、错误的类型写法、
  不一致的数组、与类型不符的默认值、未闭合的字符串。
- `null` 的 Go 值由字符串 `"null"` 改为 `nil`，`FlagInfo.ValueType` 为 `"null"`。
- 字符串转义被还原（`'it\'s'` → `it's`）。
- `VariableDetail.ArrayType` 统一为元素类别（二维数组由 `number[]` 改为 `number`，空数组由 `[]` 改为空串）。
- 错误信息格式统一为 `<描述> at position <字节偏移>`。

### 其他

- `go.mod` 的 Go 版本由 1.25 降为 1.18（实际所需的最低版本，模糊测试需要 1.18），与 README 一致。
- 性能：去掉 `tokenTypeName` 每次调用的切片分配、AST 序列化改用 `strings.Builder`、移除未使用的括号栈。
- 测试：表驱动精确比较替换只打印的测试；新增往返测试、错误位置测试、带不变式的模糊测试、Example 与 Benchmark；覆盖率 99%。
- 已知限制：`core` 包目前是公开 API 的一部分（`VariableDetail.TypeStruct` 引用 `core.TypeNode`），计划在 v2 中移入 `internal/`。
