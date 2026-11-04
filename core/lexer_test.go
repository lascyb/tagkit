package core

import (
	"errors"
	"strings"
	"testing"
)

// lexAll 扫描输入直到 EOF，返回全部 Token；设置上限防止死循环
func lexAll(t *testing.T, input string) ([]Token, *Lexer) {
	t.Helper()
	l := NewLexer(input)
	var toks []Token
	for i := 0; i <= len(input)+1; i++ {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == TokenEOF {
			return toks, l
		}
	}
	t.Fatalf("lexer did not reach EOF for %q", input)
	return nil, nil
}

// TestLexerUnterminatedString 回归测试：未闭合字符串（含以反斜杠结尾）不得 panic，且需记录词法错误
func TestLexerUnterminatedString(t *testing.T) {
	cases := []struct {
		input string
		value string
	}{
		{`'\`, `\`},
		{`"\`, `\`},
		{`'abc\`, `abc\`},
		{`'abc`, `abc`},
		{`"`, ``},
		{`f(a:'x\`, `x\`},
		{`f(a:$a:String='\`, `\`},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			toks, l := lexAll(t, c.input)
			if l.Err() == nil || !strings.Contains(l.Err().Error(), "unterminated string") {
				t.Fatalf("expected unterminated string error, got %v", l.Err())
			}
			str := toks[len(toks)-2]
			if str.Type != TokenString || str.Value != c.value {
				t.Fatalf("expected STR(%q), got %v", c.value, str)
			}
		})
	}
}

// TestLexerTerminatedString 正常字符串（含转义）保持原有行为且无错误
func TestLexerTerminatedString(t *testing.T) {
	cases := map[string]string{
		`'abc'`:  `abc`,
		`"a\"b"`: `a\"b`,
		`'a\\'`:  `a\\`,
		`'\''`:   `\'`,
		`''`:     ``,
		`"$1.2"`: `$1.2`,
		`'中文'`:   `中文`,
	}
	for input, want := range cases {
		toks, l := lexAll(t, input)
		if l.Err() != nil {
			t.Fatalf("%q: unexpected error %v", input, l.Err())
		}
		if toks[0].Type != TokenString || toks[0].Value != want {
			t.Fatalf("%q: expected STR(%q), got %v", input, want, toks[0])
		}
	}
}

// TestParserUnterminatedString 解析器需把词法错误作为错误返回
func TestParserUnterminatedString(t *testing.T) {
	for _, input := range []string{`'\`, `"\`, `name(a:'x`, `,flag='abc\`, `f(a:$a:[String]=['a\`} {
		nodes, err := NewParser(input).Parse()
		if err == nil || !strings.Contains(err.Error(), "unterminated string") {
			t.Fatalf("%q: expected unterminated string error, got nodes=%v err=%v", input, nodes, err)
		}
	}
}

// FuzzLexer 任意输入下词法分析器不得 panic，且必须在有限步内到达 EOF
func FuzzLexer(f *testing.F) {
	for _, s := range []string{``, `'\`, `"\`, `a(b:$c:[Int!]!=[1,2]),d=e`, `'a\'b'`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		l := NewLexer(input)
		for i := 0; i <= len(input)+1; i++ {
			if l.NextToken().Type == TokenEOF {
				return
			}
		}
		t.Fatalf("lexer did not reach EOF for %q", input)
	})
}

// TestLexerTokens 表驱动：合法输入的完整 Token 序列（类型、值、位置）
func TestLexerTokens(t *testing.T) {
	cases := []struct {
		input string
		want  []Token
	}{
		{``, []Token{{TokenEOF, "", 0}}},
		{`  `, []Token{{TokenEOF, "", 2}}},
		{`a_1(B:$c)`, []Token{{TokenIdent, "a_1", 0}, {TokenLParen, "(", 3}, {TokenIdent, "B", 4}, {TokenColon, ":", 5}, {TokenDollar, "$", 6}, {TokenIdent, "c", 7}, {TokenRParen, ")", 8}, {TokenEOF, "", 9}}},
		{`[Int!]!=,`, []Token{{TokenLBracket, "[", 0}, {TokenIdent, "Int", 1}, {TokenBang, "!", 4}, {TokenRBracket, "]", 5}, {TokenBang, "!", 6}, {TokenEqual, "=", 7}, {TokenComma, ",", 8}, {TokenEOF, "", 9}}},
		{"true false null\t\n_x", []Token{{TokenBool, "true", 0}, {TokenBool, "false", 5}, {TokenIdent, "null", 11}, {TokenIdent, "_x", 17}, {TokenEOF, "", 19}}},
		{`0 -1 1.5 -0.25 1e3 2E-2 3.5e+10`, []Token{{TokenNumber, "0", 0}, {TokenNumber, "-1", 2}, {TokenNumber, "1.5", 5}, {TokenNumber, "-0.25", 9}, {TokenNumber, "1e3", 15}, {TokenNumber, "2E-2", 19}, {TokenNumber, "3.5e+10", 24}, {TokenEOF, "", 31}}},
		{`'中文'"x"`, []Token{{TokenString, "中文", 0}, {TokenString, "x", 8}, {TokenEOF, "", 11}}},
		{`1]`, []Token{{TokenNumber, "1", 0}, {TokenRBracket, "]", 1}, {TokenEOF, "", 2}}},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, l := lexAll(t, c.input)
			if l.Err() != nil {
				t.Fatalf("unexpected error %v", l.Err())
			}
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("token %d: got %v@%d, want %v@%d", i, got[i], got[i].Pos, c.want[i], c.want[i].Pos)
				}
			}
		})
	}
}

// TestLexerErrors 非法字符、非 ASCII 标识符与非法数字必须报 *SyntaxError（v1.0.x 会静默跳过）
func TestLexerErrors(t *testing.T) {
	cases := []struct {
		input string
		pos   int
		msg   string
	}{
		{`@`, 0, "unexpected character '@'"},
		{`a #`, 2, "unexpected character '#'"},
		{`f(a:-x)`, 4, "unexpected character '-'"},
		{`ä`, 0, "unexpected character 'ä' (U+00E4)"},
		{`a中`, 1, "unexpected character '中' (U+4E2D)"},
		{"a\x00", 1, "unexpected character '\\x00'"},
		{`.5`, 0, "must start with a digit"},
		{`-`, 0, "unexpected character '-'"},
		{`.`, 0, "unexpected character '.'"},
		{`1.`, 0, "expected digit after '.'"},
		{`1.x`, 0, "expected digit after '.'"},
		{`1e`, 0, "expected digit in exponent"},
		{`1e+`, 0, "expected digit in exponent"},
		{`007`, 0, "leading zeros"},
		{`-01`, 0, "leading zeros"},
		{`1_000`, 0, "unexpected '_' after number"},
		{`0x10`, 0, "unexpected 'x' after number"},
		{`1.5.3`, 0, "unexpected '.' after number"},
		{`12abc`, 0, "unexpected 'a' after number"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			toks, l := lexAll(t, c.input)
			var se *SyntaxError
			if !errors.As(l.Err(), &se) {
				t.Fatalf("expected *SyntaxError, got %v (tokens %v)", l.Err(), toks)
			}
			if se.Pos != c.pos || !strings.Contains(se.Msg, c.msg) {
				t.Fatalf("got %q at %d, want %q at %d", se.Msg, se.Pos, c.msg, c.pos)
			}
			if last := toks[len(toks)-1]; last.Type != TokenEOF {
				t.Fatalf("lexer must stop with EOF after error, got %v", last)
			}
			if l.NextToken().Type != TokenEOF {
				t.Fatal("lexer must keep returning EOF after an error")
			}
		})
	}
}

// TestUnescapeQuote 转义还原与引号编码：\\ \' \" 还原，其余反斜杠序列保留；Quote 与 Unescape 互逆
func TestUnescapeQuote(t *testing.T) {
	cases := map[string]string{
		``:          ``,
		`abc`:       `abc`,
		`it\'s`:     `it's`,
		`a\"b`:      `a"b`,
		`a\\b`:      `a\b`,
		`a\\\'b`:    `a\'b`,
		`a\nb`:      `a\nb`,
		`a\\n`:      `a\n`,
		`\`:         `\`,
		`x\`:        `x\`,
		`中\'文`:      `中'文`,
		`\t\\\"end`: `\t\"end`,
	}
	for in, want := range cases {
		if got := Unescape(in); got != want {
			t.Errorf("Unescape(%q) = %q, want %q", in, got, want)
		}
	}
	for _, s := range []string{``, `abc`, `it's`, `a\b`, `a\nb`, `\`, `'`, `\'`, `"`, `中文`} {
		q := Quote(s)
		if q[0] != '\'' || q[len(q)-1] != '\'' {
			t.Fatalf("Quote(%q) = %q: not single-quoted", s, q)
		}
		if got := Unescape(q[1 : len(q)-1]); got != s {
			t.Errorf("Unescape(Quote(%q)) = %q", s, got)
		}
	}
	if Quote(`it's`) != `'it\'s'` || Quote(`a\b`) != `'a\\b'` {
		t.Errorf("unexpected Quote output %s %s", Quote(`it's`), Quote(`a\b`))
	}
}

// TestTokenString Token.String 与未知类型名称
func TestTokenString(t *testing.T) {
	if s := (Token{Type: TokenIdent, Value: "a"}).String(); s != `IDENT("a")` {
		t.Fatalf("got %s", s)
	}
	if s := (Token{Type: TokenType(99)}).String(); s != `UNKNOWN("")` {
		t.Fatalf("got %s", s)
	}
	if s := (Token{Type: TokenType(-1)}).String(); s != `UNKNOWN("")` {
		t.Fatalf("got %s", s)
	}
}

// BenchmarkLexer 典型 tag 的词法扫描
func BenchmarkLexer(b *testing.B) {
	const input = `products(first:$first:Int=10, query:$q:String, sortKey:CREATED_AT),alias=items,directive='include(if:$w)'`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l := NewLexer(input)
		for l.NextToken().Type != TokenEOF {
		}
	}
}
