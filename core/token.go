package core

import "fmt"

// ============ 词法分析器 ============

// TokenType 词法单元类型枚举，用于标识源码中的各类符号与字面量
type TokenType int

const (
	TokenEOF      TokenType = iota // {TokenEOF: 文件结束}
	TokenIdent                     // {TokenIdent: 标识符}
	TokenVariable                  // {TokenVariable: 变量}
	TokenLParen                    // {TokenLParen: 左圆括号 (}
	TokenRParen                    // {TokenRParen: 右圆括号 )}
	TokenLBracket                  // {TokenLBracket: 左方括号 [}
	TokenRBracket                  // {TokenRBracket: 右方括号 ]}
	TokenBang                      // {TokenBang: 非空标记 !}
	TokenColon                     // {TokenColon: 冒号 :}
	TokenComma                     // {TokenComma: 逗号 ,}
	TokenEqual                     // {TokenEqual: 等号 =}
	TokenDollar                    // {TokenDollar: 变量前缀 $}
	TokenNumber                    // {TokenNumber: 数字字面量}
	TokenString                    // {TokenString: 字符串字面量 "string" 或 'string'}
	TokenBool                      // {TokenBool: 布尔字面量 true/false}
)

// Token 单个词法单元，包含类型、原始文本值及在输入中的位置；用于词法分析输出与解析器消费
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// tokenTypeName 返回 TokenType 的可读名称，用于调试与错误信息
func tokenTypeName(t TokenType) string {
	names := []string{"EOF", "IDENT", "VAR", "(", ")", "[", "]", "!", ":", ",", "=", "$", "NUM", "STR", "BOOL"}
	if int(t) < len(names) {
		return names[t]
	}
	return "UNKNOWN"
}

// String 将 Token 格式化为 "类型(值)" 字符串，便于日志与调试
func (t Token) String() string {
	return fmt.Sprintf("%s(%q)", tokenTypeName(t.Type), t.Value)
}
