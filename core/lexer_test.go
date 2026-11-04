package core

import (
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
