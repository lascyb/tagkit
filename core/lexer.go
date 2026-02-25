package core

import "unicode"

// Lexer 词法分析器，将输入字符串逐字符扫描并产出 Token 流；用于解析 tag 语法前的分词
type Lexer struct {
	input string
	pos   int
	ch    byte
}

// NewLexer 根据输入字符串创建 Lexer 并定位到首个字符
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar 前进一个字符并更新 ch，到达末尾时 ch 为 0
func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.pos]
	}
	l.pos++
}

// peekChar 返回当前字符但不前进位置，用于前瞻
func (l *Lexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// skipWhitespace 跳过连续空白字符
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier 从当前字符起读取标识符（字母/数字/下划线），返回不含前导字符的片段
func (l *Lexer) readIdentifier() string {
	start := l.pos - 1
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

// readNumber 从当前字符起读取整数或浮点数字面量
func (l *Lexer) readNumber() string {
	start := l.pos - 1
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[start : l.pos-1]
}

// readString 从当前引号起读取字符串字面量，支持 \" 转义，不消费结束引号后的字符由调用方处理
func (l *Lexer) readString() string {
	quote := l.ch
	l.readChar()
	start := l.pos - 1

	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}

	str := l.input[start : l.pos-1]
	if l.ch == quote {
		l.readChar()
	}
	return str
}

// NextToken 跳过空白后返回下一个 Token，到达输入末尾返回 TokenEOF
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	pos := l.pos - 1

	switch l.ch {
	case 0:
		return Token{Type: TokenEOF, Value: "", Pos: pos}
	case '(':
		l.readChar()
		return Token{Type: TokenLParen, Value: "(", Pos: pos}
	case ')':
		l.readChar()
		return Token{Type: TokenRParen, Value: ")", Pos: pos}
	case '[':
		l.readChar()
		return Token{Type: TokenLBracket, Value: "[", Pos: pos}
	case ']':
		l.readChar()
		return Token{Type: TokenRBracket, Value: "]", Pos: pos}
	case '!':
		l.readChar()
		return Token{Type: TokenBang, Value: "!", Pos: pos}
	case ':':
		l.readChar()
		return Token{Type: TokenColon, Value: ":", Pos: pos}
	case ',':
		l.readChar()
		return Token{Type: TokenComma, Value: ",", Pos: pos}
	case '=':
		l.readChar()
		return Token{Type: TokenEqual, Value: "=", Pos: pos}
	case '$':
		l.readChar()
		return Token{Type: TokenDollar, Value: "$", Pos: pos}
	case '"', '\'':
		return Token{Type: TokenString, Value: l.readString(), Pos: pos}
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			if ident == "true" || ident == "false" {
				return Token{Type: TokenBool, Value: ident, Pos: pos}
			}
			return Token{Type: TokenIdent, Value: ident, Pos: pos}
		} else if isDigit(l.ch) {
			return Token{Type: TokenNumber, Value: l.readNumber(), Pos: pos}
		}
		l.readChar()
		return l.NextToken()
	}
}

// isLetter 判断字符是否为字母或下划线
func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

// isDigit 判断字符是否为十进制数字
func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}
