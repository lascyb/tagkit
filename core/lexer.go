package core

import (
	"unicode/utf8"
)

// Lexer 词法分析器，将输入字符串逐字符扫描并产出 Token 流；用于解析 tag 语法前的分词。
// 遇到首个词法错误（非法字符、未闭合字符串、非法数字）时记录错误，之后只返回 TokenEOF
type Lexer struct {
	input string
	pos   int
	ch    byte
	err   error // 首个词法错误（*SyntaxError），由 Err 返回
}

// Err 返回扫描过程中遇到的首个词法错误（*SyntaxError），无错误时返回 nil
func (l *Lexer) Err() error {
	return l.err
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

// atEOF 当前字符是否已越过输入末尾（区分输入中真实的 NUL 字节）
func (l *Lexer) atEOF() bool {
	return l.pos > len(l.input)
}

// peekChar 返回当前字符的下一个字符但不前进位置，用于前瞻
func (l *Lexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// fail 记录首个词法错误并返回位于 pos 的 EOF Token；此后扫描停止
func (l *Lexer) fail(pos int, format string, args ...interface{}) Token {
	if l.err == nil {
		l.err = errorf(pos, format, args...)
	}
	l.pos = len(l.input) + 1
	l.ch = 0
	return Token{Type: TokenEOF, Pos: pos}
}

// skipWhitespace 跳过连续空白字符（空格、制表符、换行、回车）
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier 从当前字符起读取 ASCII 标识符（字母/数字/下划线）
func (l *Lexer) readIdentifier() string {
	start := l.pos - 1
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

// readNumber 按 GraphQL IntValue/FloatValue 规则读取数字：可选负号、无前导零、可选小数与指数部分；
// 数字后紧跟字母、下划线或 '.' 视为非法（如 1_000、0x10、1.5.3）
func (l *Lexer) readNumber() Token {
	start := l.pos - 1
	if l.ch == '-' {
		l.readChar()
	}
	if l.ch == '0' {
		l.readChar()
		if isDigit(l.ch) {
			return l.fail(start, "invalid number: leading zeros are not allowed")
		}
	} else {
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	if l.ch == '.' {
		l.readChar()
		if !isDigit(l.ch) {
			return l.fail(start, "invalid number: expected digit after '.'")
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	if l.ch == 'e' || l.ch == 'E' {
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		if !isDigit(l.ch) {
			return l.fail(start, "invalid number: expected digit in exponent")
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	if isLetter(l.ch) || l.ch == '.' {
		return l.fail(start, "invalid number: unexpected %q after number", rune(l.ch))
	}
	return Token{Type: TokenNumber, Value: l.input[start : l.pos-1], Pos: start}
}

// readString 从当前引号起读取字符串字面量，返回引号内的原文（不还原转义，由 Unescape 处理）。
// 反斜杠总是与其后一个字符成对跳过；字符串未闭合（包括以反斜杠结尾）时返回已读取的内容并记录词法错误，不会越界
func (l *Lexer) readString() string {
	quote := l.ch
	quotePos := l.pos - 1
	l.readChar()
	start := l.pos - 1

	for l.ch != quote && !l.atEOF() {
		if l.ch == '\\' {
			l.readChar()
			if l.atEOF() {
				// 反斜杠位于输入末尾：转义目标不存在，停止扫描，避免继续前进导致切片越界
				break
			}
		}
		l.readChar()
	}

	end := l.pos - 1
	if end > len(l.input) {
		end = len(l.input)
	}
	str := l.input[start:end]
	if !l.atEOF() && l.ch == quote {
		l.readChar()
	} else if l.err == nil {
		l.err = errorf(quotePos, "unterminated string")
	}
	return str
}

// NextToken 跳过空白后返回下一个 Token，到达输入末尾或发生词法错误后返回 TokenEOF
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	pos := l.pos - 1
	if l.atEOF() {
		if pos > len(l.input) {
			pos = len(l.input)
		}
		return Token{Type: TokenEOF, Value: "", Pos: pos}
	}

	var t TokenType
	switch l.ch {
	case '(':
		t = TokenLParen
	case ')':
		t = TokenRParen
	case '[':
		t = TokenLBracket
	case ']':
		t = TokenRBracket
	case '!':
		t = TokenBang
	case ':':
		t = TokenColon
	case ',':
		t = TokenComma
	case '=':
		t = TokenEqual
	case '$':
		t = TokenDollar
	case '"', '\'':
		return Token{Type: TokenString, Value: l.readString(), Pos: pos}
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			if ident == "true" || ident == "false" {
				return Token{Type: TokenBool, Value: ident, Pos: pos}
			}
			return Token{Type: TokenIdent, Value: ident, Pos: pos}
		}
		if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			return l.readNumber()
		}
		if l.ch == '.' && isDigit(l.peekChar()) {
			return l.fail(pos, "invalid number: a number must start with a digit (write 0.5, not .5)")
		}
		if l.ch >= utf8.RuneSelf {
			r, _ := utf8.DecodeRuneInString(l.input[pos:])
			return l.fail(pos, "unexpected character %q (U+%04X); identifiers must be ASCII letters, digits or '_' (non-ASCII text must be quoted)", r, r)
		}
		return l.fail(pos, "unexpected character %q", rune(l.ch))
	}
	l.readChar()
	return Token{Type: t, Value: tokenNames[t], Pos: pos}
}

// isLetter 判断字符是否为 ASCII 字母或下划线（GraphQL NameStart）
func isLetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_'
}

// isDigit 判断字符是否为十进制数字
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
