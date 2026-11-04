package core

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// canonical 将 Parse 结果重新序列化为 tag 写法：标记位前总带逗号，字段带括号
func canonical(nodes []Node) string {
	var b strings.Builder
	for i, n := range nodes {
		if _, isFlag := n.(*FlagNode); isFlag || i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(n.String())
	}
	return b.String()
}

// TestParserAST 表驱动：解析结果的规范写法与节点结构
func TestParserAST(t *testing.T) {
	cases := []struct {
		input string
		want  string // canonical(nodes)
	}{
		{``, ``},
		{`,,`, ``},
		{`single`, `single()`},
		{`single()`, `single()`},
		{`()`, `()`},
		{`a,b`, `a(),b`},
		{`,a`, `,a`},
		{`limit=10,verbose`, `,limit=10,verbose`},
		{`f(z:1,a:2,m:3)`, `f(z:1,a:2,m:3)`},
		{`f(a:1,)`, `f(a:1)`},
		{`f( a : 1 , b : 'x' )`, `f(a:1,b:'x')`},
		{`f(s:"it's",t:'a\'b',u:'a\\b',v:'a\nb')`, `f(s:'it\'s',t:'a\'b',u:'a\\b',v:'a\\nb')`},
		{`f(e:CREATED_AT,n:null,b:true,x:-1.5e3)`, `f(e:CREATED_AT,n:null,b:true,x:-1.5e3)`},
		{`f(ids:[1,2,],m:[[1],[],[2,3]],s:[a,'b'],b:[true,null])`, `f(ids:[1,2],m:[[1],[],[2,3]],s:[a,'b'],b:[true,null])`},
		{`f(a:$,b:$:Int!,c:$c,d:$d:[[ID!]]!=[[1,'x']])`, `f(a:$,b:$:Int!,c:$c,d:$d:[[ID!]]!=[[1,'x']])`},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			nodes, err := NewParser(c.input).Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := canonical(nodes)
			if got != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
			again, err := NewParser(got).Parse()
			if err != nil || canonical(again) != got {
				t.Fatalf("round trip failed: %s → %v %v", got, canonical(again), err)
			}
		})
	}
}

// TestParserNodes 精确比较节点结构（含位置、种类、有序参数）
func TestParserNodes(t *testing.T) {
	nodes, err := NewParser(`q(b:$v:[Int!]=[1], a:E, s:'x'),k=null`).Parse()
	if err != nil {
		t.Fatal(err)
	}
	def := &ArrayNode{Pos: 14, Elements: []Node{&LiteralNode{Value: "1", Type: "number", Kind: LiteralNumber, Pos: 15}}}
	v := &VariableNode{Name: "v", Pos: 4, VarType: &TypeNode{IsList: true, InnerType: &TypeNode{Name: "Int", IsNonNull: true}}, DefaultValue: def, Raw: "$v:[Int!]=[1]"}
	e := &LiteralNode{Value: "E", Type: "string", IsString: true, Kind: LiteralEnum, Pos: 21}
	s := &LiteralNode{Value: "x", Type: "string", IsString: true, Kind: LiteralString, Pos: 26}
	want := []Node{
		&FieldNode{Name: "q", Pos: 0,
			Args:    map[string]Node{"b": v, "a": e, "s": s},
			ArgList: []*ArgNode{{Name: "b", Value: v, Pos: 2}, {Name: "a", Value: e, Pos: 19}, {Name: "s", Value: s, Pos: 24}}},
		&FlagNode{Name: "k", Pos: 31, Value: &LiteralNode{Value: "null", Type: "null", Kind: LiteralNull, Pos: 33}},
	}
	if !reflect.DeepEqual(nodes, want) {
		t.Fatalf("got  %#v\nwant %#v", nodes, want)
	}
}

// TestParserErrors 非法输入返回 *SyntaxError，位置指向出错的 Token
func TestParserErrors(t *testing.T) {
	cases := []struct {
		input string
		pos   int
		msg   string
	}{
		// M2 重复参数
		{`f(a:1,a:2)`, 6, `duplicate argument "a"`},
		// M5 逗号
		{`f(a:1 b:2)`, 6, `expected ',' or ')' after argument`},
		{`f(a:1,,b:2)`, 6, `expected parameter name or ')'`},
		{`f(,a:1)`, 2, `expected parameter name or ')'`},
		{`a b`, 2, `expected ',' between elements`},
		{`x=1 y=2`, 4, `expected ',' between elements`},
		{`f(a:1)(b:2)`, 6, `expected ',' between elements`},
		{`f(a:[1 2])`, 7, `expected ',' or ']' in array`},
		{`f(a:[,1])`, 5, `array element`},
		// 括号
		{`)`, 0, `unmatched ')'`},
		{`f(a:1))`, 6, `unmatched ')'`},
		{`f(a:1`, 5, `got end of input`},
		{`f(a:[1`, 6, `got end of input`},
		{`f(a:(1))`, 4, `expected literal value`},
		{`f(a)`, 3, `expected ':' after parameter name`},
		{`=1`, 0, `expected identifier or '('`},
		{`,true`, 1, `expected identifier or '('`},
		{`f(true:1)`, 2, `expected parameter name`},
		{`,flag=`, 6, `expected literal value`},
		{`,flag=$x`, 6, `expected literal value`},
		// M3 类型
		{`f(a:$v:)`, 7, `expected type name or '['`},
		{`f(a:$v:[Int)`, 11, `expected ']' to close list type`},
		{`f(a:$v:Int!!)`, 11, `duplicate '!'`},
		{`f(a:$v:[Int!!])`, 12, `duplicate '!'`},
		{`f(a:$v:[Int]!!)`, 13, `duplicate '!'`},
		{`f(a:$v:true)`, 7, `expected type name`},
		{`f(a:$1)`, 5, `invalid variable name`},
		{`f(a:$'x')`, 5, `invalid variable name`},
		{`f(a:$v:Int=$w)`, 11, `expected literal value`},
		// M6 数组一致性
		{`f(a:[1,[2]])`, 7, `mixes scalars and lists`},
		{`f(a:[[1],2])`, 9, `mixes scalars and lists`},
		{`f(a:['x',1])`, 9, `mixes string and number`},
		{`f(a:[[1],[true]])`, 9, `mixes number and bool`},
		{`f(a:[[1],[[2]]])`, 9, `different dimensions`},
		{`f(a:[[1],null,2])`, 14, `mixes scalars and lists`},
		{`f(a:[[null],[1],['x']])`, 16, `mixes number and string`},
		// M6 默认值与类型
		{`f(a:$v:Int='x')`, 11, `does not match type Int`},
		{`f(a:$v:Int=1.5)`, 11, `does not match type Int`},
		{`f(a:$v:Int=3000000000)`, 11, `out of range for Int`},
		{`f(a:$v:Float=true)`, 13, `does not match type Float`},
		{`f(a:$v:Boolean=1)`, 15, `does not match type Boolean`},
		{`f(a:$v:String=1)`, 14, `does not match type String`},
		{`f(a:$v:ID=1.5)`, 10, `does not match type ID`},
		{`f(a:$v:Int!=null)`, 12, `null is not allowed for non-null type Int!`},
		{`f(a:$v:[Int!]=[1,null])`, 17, `null is not allowed for non-null type Int!`},
		{`f(a:$v:[Int]=1)`, 13, `must be a list`},
		{`f(a:$v:Int=[1])`, 11, `not allowed for non-list type Int`},
		{`f(a:$v:[[Int]]=[1])`, 16, `must be a list`},
		{`f(a:$v:[Int]=[[1]])`, 14, `not allowed for non-list type Int`},
		// M7 数字范围
		{`f(a:99999999999999999999)`, 4, `out of range for int`},
		{`f(a:1e999)`, 4, `out of range for float64`},
		// 词法错误优先
		{`f(a:'x`, 4, `unterminated string`},
		{`f(a:1 @)`, 6, `unexpected character '@'`},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			nodes, err := NewParser(c.input).Parse()
			var se *SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("expected *SyntaxError, got nodes=%v err=%v", nodes, err)
			}
			if se.Pos != c.pos || !strings.Contains(se.Msg, c.msg) {
				t.Fatalf("got %q at %d, want %q at %d", se.Msg, se.Pos, c.msg, c.pos)
			}
			if !strings.HasSuffix(err.Error(), " at position "+strconv.Itoa(c.pos)) {
				t.Fatalf("Error() = %q", err.Error())
			}
		})
	}
}

// TestDefaultsAccepted 合法的默认值与类型组合
func TestDefaultsAccepted(t *testing.T) {
	for _, in := range []string{
		`f(a:$v:Int=-2147483648)`, `f(a:$v:Float=1)`, `f(a:$v:Float=-1.5e3)`, `f(a:$v:String=abc)`, `f(a:$v:String='abc')`,
		`f(a:$v:ID=1)`, `f(a:$v:ID='x')`, `f(a:$v:ID=x)`, `f(a:$v:Boolean=false)`, `f(a:$v:Int=null)`,
		`f(a:$v:[Int]=[])`, `f(a:$v:[[Int]]=[])`, `f(a:$v:[[Int]]=[[],[1]])`, `f(a:$v:[[Int]]=[[1],null])`, `f(a:$v:[Int]!=[1,null])`,
		`f(a:$v:Sort=CREATED_AT)`, `f(a:$v:JSON='{}')`, `f(a:$v:[Custom!]!=[1,2])`, `f(a:$v=[[1,2],[3]])`, `f(a:$v=null)`, `f(a:[[null],[1],null])`, `f(a:[null,[],[[1]]])`,
	} {
		if _, err := NewParser(in).Parse(); err != nil {
			t.Errorf("%s: unexpected error %v", in, err)
		}
	}
}

// TestTypeNodeHelpers ListDepth、BaseName 与 String
func TestTypeNodeHelpers(t *testing.T) {
	cases := []struct {
		in    string
		depth int
		base  string
	}{{`Int`, 0, "Int"}, {`Int!`, 0, "Int"}, {`[Int]`, 1, "Int"}, {`[[ID!]!]!`, 2, "ID"}}
	for _, c := range cases {
		nodes, err := NewParser(`f(a:$:` + c.in + `)`).Parse()
		if err != nil {
			t.Fatal(err)
		}
		typ := nodes[0].(*FieldNode).ArgList[0].Value.(*VariableNode).VarType
		if typ.ListDepth() != c.depth || typ.BaseName() != c.base || typ.String() != c.in {
			t.Errorf("%s: depth=%d base=%s str=%s", c.in, typ.ListDepth(), typ.BaseName(), typ)
		}
	}
	if (&TypeNode{IsList: true}).BaseName() != "" || (&TypeNode{IsList: true}).String() != "[]" {
		t.Error("nil inner type")
	}
}

// TestManualNodes 手动构造的节点（无 Kind / 无 ArgList）序列化结果稳定，兼容 v1.0.x
func TestManualNodes(t *testing.T) {
	f := &FieldNode{Name: "f", Args: map[string]Node{
		"b": &LiteralNode{Value: "x", IsString: true},
		"a": &LiteralNode{Value: "1", Type: "number"},
	}}
	if got := f.String(); got != `f(a:1,b:'x')` {
		t.Fatalf("got %s", got)
	}
	if got := (&FlagNode{Name: "k"}).String(); got != "k" {
		t.Fatalf("got %s", got)
	}
	for k, want := range map[LiteralKind]string{LiteralUnspecified: "unspecified", LiteralString: "string", LiteralEnum: "enum", LiteralNumber: "number", LiteralBool: "bool", LiteralNull: "null"} {
		if k.String() != want {
			t.Errorf("%d: %s", k, k.String())
		}
	}
	if ArrayElemType(&ArrayNode{Elements: []Node{&LiteralNode{Value: "x", IsString: true}}}) != "string" ||
		ArrayElemType(&ArrayNode{Elements: []Node{&LiteralNode{Value: "1", Type: "number"}}}) != "number" ||
		ArrayElemType(&ArrayNode{Elements: []Node{&LiteralNode{Kind: LiteralNull, Value: "null"}}}) != "null" ||
		ArrayElemType(&ArrayNode{Elements: []Node{&ArrayNode{Elements: []Node{&LiteralNode{Kind: LiteralNull}}}}}) != "null" ||
		ArrayElemType(&ArrayNode{}) != "" {
		t.Error("ArrayElemType")
	}
}

// FuzzParser 不变式：不 panic；成功时规范写法可再次解析且结果相同；失败时为 *SyntaxError 且位置在输入范围内
func FuzzParser(f *testing.F) {
	for _, s := range []string{``, `a,b`, `f(a:$v:[Int!]!=[1,2],e:E,s:'x\'y'),k=[[true],null]`, `(x:-1.5e3)`, `'\`, `f(a:1 b:2)`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		nodes, err := NewParser(input).Parse()
		if err != nil {
			var se *SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("%q: error is not *SyntaxError: %v", input, err)
			}
			if se.Pos < 0 || se.Pos > len(input) {
				t.Fatalf("%q: error position %d out of range", input, se.Pos)
			}
			return
		}
		s := canonical(nodes)
		again, err := NewParser(s).Parse()
		if err != nil {
			t.Fatalf("%q: canonical form %q does not parse: %v", input, s, err)
		}
		if s2 := canonical(again); s2 != s {
			t.Fatalf("%q: round trip %q → %q", input, s, s2)
		}
	})
}

// BenchmarkParser 典型 tag 的解析（AST）
func BenchmarkParser(b *testing.B) {
	const input = `products(first:$first:Int=10, query:$q:String, sortKey:CREATED_AT),alias=items,directive='include(if:$w)'`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := NewParser(input).Parse(); err != nil {
			b.Fatal(err)
		}
	}
}
