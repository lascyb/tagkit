package tagkit_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/lascyb/tagkit"
	"github.com/lascyb/tagkit/core"
)

// ============ 期望值构造辅助 ============

func lit(kind tagkit.ValueKind, v interface{}, text string) *tagkit.Literal {
	return &tagkit.Literal{Kind: kind, Value: v, Text: text}
}
func str(s, text string) *tagkit.Literal { return lit(tagkit.KindString, s, text) }
func enum(s string) *tagkit.Literal      { return lit(tagkit.KindEnum, s, s) }
func num(v interface{}, text string) *tagkit.Literal {
	return lit(tagkit.KindNumber, v, text)
}
func boolean(b bool) *tagkit.Literal {
	if b {
		return lit(tagkit.KindBool, true, "true")
	}
	return lit(tagkit.KindBool, false, "false")
}
func null() *tagkit.Literal { return lit(tagkit.KindNull, nil, "null") }
func list(elems ...*tagkit.Literal) *tagkit.Literal {
	values := make([]interface{}, len(elems))
	texts := make([]string, len(elems))
	for i, e := range elems {
		values[i] = e.Value
		texts[i] = e.Text
	}
	if elems == nil {
		elems = []*tagkit.Literal{}
	}
	return &tagkit.Literal{Kind: tagkit.KindList, Value: values, Text: "[" + strings.Join(texts, ",") + "]", Elems: elems}
}

// arg 期望的单个参数；variable 非 nil 时为变量参数
type arg struct {
	key      string
	literal  *tagkit.Literal
	variable *variable
}
type variable struct {
	name, typ, raw string
	def            *tagkit.Literal
	arrayType      string
}

func litArg(key string, l *tagkit.Literal) arg { return arg{key: key, literal: l} }
func varArg(key, name, typ, raw string, def *tagkit.Literal, arrayType string) arg {
	return arg{key: key, variable: &variable{name, typ, raw, def, arrayType}}
}

// typeNode 通过 core 解析类型字符串得到 TypeNode（core 包中对 TypeNode 结构有独立的精确测试）
func typeNode(t *testing.T, typ string) *core.TypeNode {
	if typ == "" {
		return nil
	}
	nodes, err := core.NewParser("f(a:$:" + typ + ")").Parse()
	if err != nil {
		t.Fatalf("bad type %s: %v", typ, err)
	}
	return nodes[0].(*core.FieldNode).ArgList[0].Value.(*core.VariableNode).VarType
}

// field 构造带字段调用的期望 TagValue
func field(t *testing.T, name string, args []arg, flags ...tagkit.FlagInfo) *tagkit.TagValue {
	tv := &tagkit.TagValue{Name: name, Args: map[string]tagkit.ArgValue{}, ArgKeys: []string{}, Variables: []tagkit.VariableDetail{}, Flags: append([]tagkit.FlagInfo{}, flags...)}
	for _, a := range args {
		tv.ArgKeys = append(tv.ArgKeys, a.key)
		if a.variable == nil {
			av := tagkit.ArgValue{Type: tagkit.ArgTypeLiteral, Value: a.literal.Value, Kind: a.literal.Kind, Literal: a.literal}
			av.Dimension = a.literal.Dimension()
			tv.Args[a.key] = av
			continue
		}
		v := a.variable
		av := tagkit.ArgValue{Type: tagkit.ArgTypeVariable, Value: v.raw, VarName: v.name, VarType: v.typ, Kind: tagkit.KindVariable}
		vd := tagkit.VariableDetail{Name: v.name, VarType: v.typ, TypeStruct: typeNode(t, v.typ), Key: a.key}
		if v.def != nil {
			av.HasDefault, vd.HasDefault = true, true
			av.DefaultVal, vd.DefaultValue = v.def.Value, v.def.Value
			av.Default, vd.Default = v.def, v.def
			if v.def.Kind == tagkit.KindList {
				av.IsArrayDefault, vd.IsArrayDefault = true, true
				av.Dimension, vd.Dimension = v.def.Dimension(), v.def.Dimension()
				vd.ArrayType = v.arrayType
			}
		}
		tv.Args[a.key] = av
		tv.Variables = append(tv.Variables, vd)
	}
	return tv
}

// flagsOnly 构造仅含标记位的期望 TagValue
func flagsOnly(flags ...tagkit.FlagInfo) *tagkit.TagValue {
	return &tagkit.TagValue{Flags: append([]tagkit.FlagInfo{}, flags...)}
}

func bare(name string) tagkit.FlagInfo {
	return tagkit.FlagInfo{Name: name, IsBoolean: true, Value: true, ValueType: tagkit.ValueTypeBool, Kind: tagkit.KindBool}
}
func kv(name string, l *tagkit.Literal) tagkit.FlagInfo {
	vt := string(l.Kind)
	switch l.Kind {
	case tagkit.KindList:
		vt = tagkit.ValueTypeArray
	case tagkit.KindEnum:
		vt = tagkit.ValueTypeString
	}
	return tagkit.FlagInfo{Name: name, Value: l.Value, ValueType: vt, Kind: l.Kind, Literal: l}
}

// ============ 表驱动：精确比较 ============

// TestParseTagValue 按「字段 → 参数 → 标记 → 组合」递进覆盖语法规则，逐字段精确比较 TagValue
func TestParseTagValue(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  *tagkit.TagValue
	}{
		// ========== 一、字段 ==========
		{"空输入", ``, flagsOnly()},
		{"仅逗号", `,,`, flagsOnly()},
		{"裸标识符为无参字段", `single2`, field(t, "single2", nil)},
		{"name() 无参", `single()`, field(t, "single", nil)},
		{"匿名字段", `(age:18,sex:$:String,sum:$),inline`, field(t, "", []arg{
			litArg("age", num(18, "18")),
			varArg("sex", "", "String", "$:String", nil, ""),
			varArg("sum", "", "", "$", nil, ""),
		}, bare("inline"))},

		// ========== 二、参数 ==========
		{"字面量：数字/布尔/枚举，保持书写顺序", `foo(x:1,y:2,z:true,label:hello)`, field(t, "foo", []arg{
			litArg("x", num(1, "1")), litArg("y", num(2, "2")), litArg("z", boolean(true)), litArg("label", enum("hello")),
		})},
		{"字面量：枚举、字符串与 null 可区分", `f(e:CREATED_AT,s:'CREATED_AT',n:null,q:"x")`, field(t, "f", []arg{
			litArg("e", enum("CREATED_AT")), litArg("s", str("CREATED_AT", "'CREATED_AT'")), litArg("n", null()), litArg("q", str("x", "'x'")),
		})},
		{"字面量：负数、小数、指数", `op(a:-1,b:1.5,c:-0.25,d:1e3,e:2E-2,f:0)`, field(t, "op", []arg{
			litArg("a", num(-1, "-1")), litArg("b", num(1.5, "1.5")), litArg("c", num(-0.25, "-0.25")),
			litArg("d", num(1000.0, "1e3")), litArg("e", num(0.02, "2E-2")), litArg("f", num(0, "0")),
		})},
		{"字面量：转义还原", `f(a:'it\'s',b:"say \"hi\"",c:'a\\b',d:'a\nb',e:"it's")`, field(t, "f", []arg{
			litArg("a", str("it's", `'it\'s'`)), litArg("b", str(`say "hi"`, `'say "hi"'`)), litArg("c", str(`a\b`, `'a\\b'`)),
			litArg("d", str(`a\nb`, `'a\\nb'`)), litArg("e", str("it's", `'it\'s'`)),
		})},
		{"字面量：列表参数不再丢失", `f(ids:[1,2],tags:[A,'b'],m:[[true],[]],n:[null,1],e:[])`, field(t, "f", []arg{
			litArg("ids", list(num(1, "1"), num(2, "2"))),
			litArg("tags", list(enum("A"), str("b", "'b'"))),
			litArg("m", list(list(boolean(true)), list())),
			litArg("n", list(null(), num(1, "1"))),
			litArg("e", list()),
		})},
		{"字面量：非 ASCII 字符串", `f(s:'中文',t:"ä")`, field(t, "f", []arg{litArg("s", str("中文", "'中文'")), litArg("t", str("ä", "'ä'"))})},
		{"末尾逗号", `f(a:1,)`, field(t, "f", []arg{litArg("a", num(1, "1"))})},
		{"空白", " f ( a : 1 ,\tb : $b : [ Int ! ] ! ) ", field(t, "f", []arg{litArg("a", num(1, "1")), varArg("b", "b", "[Int!]!", "$b:[Int!]!", nil, "")})},
		{"变量无类型", `bare(id:$id,name:$name)`, field(t, "bare", []arg{varArg("id", "id", "", "$id", nil, ""), varArg("name", "name", "", "$name", nil, "")})},
		{"变量+类型", `bar(id:$id:Int,name:$name:String)`, field(t, "bar", []arg{varArg("id", "id", "Int", "$id:Int", nil, ""), varArg("name", "name", "String", "$name:String", nil, "")})},
		{"变量+类型+默认", `name(age:$age:Int=18, first:10),inline=3,union=false`, field(t, "name", []arg{
			varArg("age", "age", "Int", "$age:Int=18", num(18, "18"), ""), litArg("first", num(10, "10")),
		}, kv("inline", num(3, "3")), kv("union", boolean(false)))},
		{"匿名变量", `(sex:$:String!,sum:$,age:18),inline`, field(t, "", []arg{
			varArg("sex", "", "String!", "$:String!", nil, ""), varArg("sum", "", "", "$", nil, ""), litArg("age", num(18, "18")),
		}, bare("inline"))},
		{"数组默认值（引号）", `tags(items:$items:[String]=['a','b','c'], count:$count:Int=5),flatten`, field(t, "tags", []arg{
			varArg("items", "items", "[String]", "$items:[String]=['a','b','c']", list(str("a", "'a'"), str("b", "'b'"), str("c", "'c'")), "string"),
			varArg("count", "count", "Int", "$count:Int=5", num(5, "5"), ""),
		}, bare("flatten"))},
		{"数组默认值（裸标识符）", `tags(items:$items:[String]=[a,b,c]),flatten=123,union=false`, field(t, "tags", []arg{
			varArg("items", "items", "[String]", "$items:[String]=[a,b,c]", list(enum("a"), enum("b"), enum("c")), "string"),
		}, kv("flatten", num(123, "123")), kv("union", boolean(false)))},
		{"二维数组", `matrix(data:$data:[[Int]]=[[1,2],[3,4]], scale:$scale:Float=1.0),normalize`, field(t, "matrix", []arg{
			varArg("data", "data", "[[Int]]", "$data:[[Int]]=[[1,2],[3,4]]", list(list(num(1, "1"), num(2, "2")), list(num(3, "3"), num(4, "4"))), "number"),
			varArg("scale", "scale", "Float", "$scale:Float=1.0", num(1.0, "1.0"), ""),
		}, bare("normalize"))},
		{"三维数组", `deep(nested:$nested:[[[Int]]]=[[[1,2]],[[3,4]]])`, field(t, "deep", []arg{
			varArg("nested", "nested", "[[[Int]]]", "$nested:[[[Int]]]=[[[1,2]],[[3,4]]]",
				list(list(list(num(1, "1"), num(2, "2"))), list(list(num(3, "3"), num(4, "4")))), "number"),
		})},
		{"空数组、空串", `empty(arr:$arr:[Int]=[], str:$str:String=''),check`, field(t, "empty", []arg{
			varArg("arr", "arr", "[Int]", "$arr:[Int]=[]", list(), ""), varArg("str", "str", "String", "$str:String=''", str("", "''"), ""),
		}, bare("check"))},
		{"字符串默认值含特殊字符", `quote(str:$str:String='$1.2')`, field(t, "quote", []arg{varArg("str", "str", "String", "$str:String='$1.2'", str("$1.2", "'$1.2'"), "")})},
		{"ID 列表允许混用整数与字符串", `f(ids:$ids:[ID!]=[1,'x'])`, field(t, "f", []arg{varArg("ids", "ids", "[ID!]", "$ids:[ID!]=[1,'x']", list(num(1, "1"), str("x", "'x'")), "mixed")})},
		{"无类型变量：null 默认与 bool 数组", `f(a:$a=null,b:$b=[true,false])`, field(t, "f", []arg{
			varArg("a", "a", "", "$a=null", null(), ""), varArg("b", "b", "", "$b=[true,false]", list(boolean(true), boolean(false)), "bool"),
		})},
		{"[Int!]! 等", `query(id_s:$id_s:[Int!]!=[1,2,3], names:$names:[String]=['foo','bar']),cache=true`, field(t, "query", []arg{
			varArg("id_s", "id_s", "[Int!]!", "$id_s:[Int!]!=[1,2,3]", list(num(1, "1"), num(2, "2"), num(3, "3")), "number"),
			varArg("names", "names", "[String]", "$names:[String]=['foo','bar']", list(str("foo", "'foo'"), str("bar", "'bar'")), "string"),
		}, kv("cache", boolean(true)))},
		{"单元素数组", `one(x:$x:[Int]=[1], s:$s:[String]=['only'])`, field(t, "one", []arg{
			varArg("x", "x", "[Int]", "$x:[Int]=[1]", list(num(1, "1")), "number"),
			varArg("s", "s", "[String]", "$s:[String]=['only']", list(str("only", "'only'")), "string"),
		})},
		{"混合复杂参数", `mixed(data:$data:[[[Int]]]=[[[1,2]],[[3,4]]], label:$label:String="test"),flag`, field(t, "mixed", []arg{
			varArg("data", "data", "[[[Int]]]", "$data:[[[Int]]]=[[[1,2]],[[3,4]]]",
				list(list(list(num(1, "1"), num(2, "2"))), list(list(num(3, "3"), num(4, "4")))), "number"),
			varArg("label", "label", "String", "$label:String='test'", str("test", "'test'"), ""),
		}, bare("flag"))},
		{"字面量 float/bool", `op(scale:1.5, enabled:true, off:false)`, field(t, "op", []arg{
			litArg("scale", num(1.5, "1.5")), litArg("enabled", boolean(true)), litArg("off", boolean(false)),
		})},
		{"默认值中的 null 元素", `f(a:$a:[[Int]]=[[1],null])`, field(t, "f", []arg{
			varArg("a", "a", "[[Int]]", "$a:[[Int]]=[[1],null]", list(list(num(1, "1")), null()), "number"),
		})},

		// ========== 三、标记 ==========
		{"仅布尔（需前导逗号）", `,verbose`, flagsOnly(bare("verbose"))},
		{"多个布尔", `,verbose,debug,dryRun`, flagsOnly(bare("verbose"), bare("debug"), bare("dryRun"))},
		{"键值有前导逗号", `,limit=10`, flagsOnly(kv("limit", num(10, "10")))},
		{"键值可省略前导逗号", `limit=10`, flagsOnly(kv("limit", num(10, "10")))},
		{"多个键值", `a=1,b=2`, flagsOnly(kv("a", num(1, "1")), kv("b", num(2, "2")))},
		{"键值+布尔", `limit=10,verbose`, flagsOnly(kv("limit", num(10, "10")), bare("verbose"))},
		{"字符串值", `,name="hello world",title='single quote'`, flagsOnly(kv("name", str("hello world", "'hello world'")), kv("title", str("single quote", "'single quote'")))},
		{"数组值", `,items=[1,2,3],names=['a','b'],bs=[true,null]`, flagsOnly(
			kv("items", list(num(1, "1"), num(2, "2"), num(3, "3"))), kv("names", list(str("a", "'a'"), str("b", "'b'"))), kv("bs", list(boolean(true), null())))},
		{"数值、布尔、枚举、null", `,ratio=1.5,enabled=true,mode=FAST,x=null,n=-3`, flagsOnly(
			kv("ratio", num(1.5, "1.5")), kv("enabled", boolean(true)), kv("mode", enum("FAST")), kv("x", null()), kv("n", num(-3, "-3")))},
		{"连续逗号", `,,flag,,`, flagsOnly(bare("flag"))},
		{"同名标记保留顺序", `directive='a',directive='b',inline`, flagsOnly(kv("directive", str("a", "'a'")), kv("directive", str("b", "'b'")), bare("inline"))},

		// ========== 四、字段+标记 ==========
		{"字段后跟标记", `name(),flag`, field(t, "name", nil, bare("flag"))},
		{"裸字段后跟标记", `a,b`, field(t, "a", nil, bare("b"))},
		{"有参字段+标记", `name(age:18,sex:man),inline,union`, field(t, "name", []arg{litArg("age", num(18, "18")), litArg("sex", enum("man"))}, bare("inline"), bare("union"))},
		{"尾逗号", `name(),`, field(t, "name", nil)},
		{"标记在字段之前", `limit=10,name(a:1)`, field(t, "name", []arg{litArg("a", num(1, "1"))}, kv("limit", num(10, "10")))},
		{"无参+多标记", `noop(),a,b=2,c=true`, field(t, "noop", nil, bare("a"), kv("b", num(2, "2")), kv("c", boolean(true)))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := tagkit.ParseTagValue(c.input)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", c.input, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("%s:\n got  %s\n want %s", c.input, dump(got), dump(c.want))
			}
			assertRoundTrip(t, c.input, got)
		})
	}
}

// dump 以稳定格式输出 TagValue 便于比对失败信息
func dump(tv *tagkit.TagValue) string {
	var b strings.Builder
	b.WriteString(tv.String())
	b.WriteString(" | ")
	for _, k := range tv.ArgKeys {
		a := tv.Args[k]
		b.WriteString(k + "=" + string(a.Kind) + " ")
	}
	for _, v := range tv.Variables {
		b.WriteString("var " + v.Name + ":" + v.VarType + " at=" + v.ArrayType + " ")
	}
	return b.String()
}

// assertRoundTrip 校验 ParseTagValue(tv.String()) 与 tv 完全相等
func assertRoundTrip(t *testing.T, input string, tv *tagkit.TagValue) {
	t.Helper()
	s := tv.String()
	again, err := tagkit.ParseTagValue(s)
	if err != nil {
		t.Fatalf("%q: String() = %q does not parse: %v", input, s, err)
	}
	if !reflect.DeepEqual(tv, again) {
		t.Fatalf("%q: round trip mismatch via %q:\n%s\n%s", input, s, dump(tv), dump(again))
	}
}

// TestParseTagValueErrors 非法输入返回 *tagkit.SyntaxError（含位置），覆盖 v1.0.x 静默接受的写法
func TestParseTagValueErrors(t *testing.T) {
	cases := []struct {
		input string
		pos   int
		msg   string
	}{
		{`'\`, 0, "unterminated string"},
		{`"\`, 0, "unterminated string"},
		{`name(a:'\`, 7, "unterminated string"},
		{`name(a:"abc`, 7, "unterminated string"},
		{`,flag='x\`, 6, "unterminated string"},
		{`q(a:$a:String='\`, 14, "unterminated string"},
		{`f(n:--1)`, 4, "unexpected character '-'"},
		{`f(a:.5)`, 4, "must start with a digit"},
		{`f(a:1.)`, 4, "expected digit after '.'"},
		{`f(a:007)`, 4, "leading zeros"},
		{`f(a:@#1)`, 4, "unexpected character '@'"},
		{`f(a:1)%%,inline`, 6, "unexpected character '%'"},
		{`f(ä:1)`, 2, "unexpected character 'ä'"},
		{`中文(a:1)`, 0, "unexpected character '中'"},
		{`,omit-empty`, 5, "unexpected character '-'"},
		{`f(a:99999999999999999999)`, 4, "out of range"},
		{`f(a:1,a:2)`, 6, `duplicate argument "a"`},
		{`f(a:1 b:2)`, 6, "expected ',' or ')'"},
		{`x=1 y=2`, 4, "expected ',' between elements"},
		{`a(x:1),b(y:2)`, 7, "field must be unique"},
		{`a,b()`, 2, "field must be unique"},
		{`f(a:$v:)`, 7, "expected type name"},
		{`f(a:$v:[Int!!])`, 12, "duplicate '!'"},
		{`f(a:[1,[2]])`, 7, "mixes scalars and lists"},
		{`f(a:$v:[Int]=[[1]])`, 14, "non-list type Int"},
		{`f(a:$v:Int='x')`, 11, "does not match type Int"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			tv, err := tagkit.ParseTagValue(c.input)
			var se *tagkit.SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("expected *SyntaxError, got tv=%v err=%v", tv, err)
			}
			if tv != nil {
				t.Fatal("result must be nil on error")
			}
			if se.Pos != c.pos || !strings.Contains(se.Msg, c.msg) {
				t.Fatalf("got %q at %d, want %q at %d", se.Msg, se.Pos, c.msg, c.pos)
			}
		})
	}
}

// TestHelpers Flag / HasFlag / FlagsNamed / Arg / OrderedArgs / HasField / HasValue / IsAnonymous
func TestHelpers(t *testing.T) {
	tv, err := tagkit.ParseTagValue(`q(b:2,a:$:Int),alias=x,directive='d1',directive='d2',inline`)
	if err != nil {
		t.Fatal(err)
	}
	if !tv.HasField() || !tv.HasFlag("inline") || tv.HasFlag("nope") {
		t.Fatal("HasField/HasFlag")
	}
	if f, ok := tv.Flag("alias"); !ok || f.Value != "x" || f.Kind != tagkit.KindEnum || !f.HasValue() {
		t.Fatalf("Flag(alias) = %+v %v", f, ok)
	}
	if f, ok := tv.Flag("inline"); !ok || f.HasValue() {
		t.Fatalf("Flag(inline) = %+v", f)
	}
	if _, ok := tv.Flag("nope"); ok {
		t.Fatal("Flag(nope)")
	}
	ds := tv.FlagsNamed("directive")
	if len(ds) != 2 || ds[0].Value != "d1" || ds[1].Value != "d2" || tv.FlagsNamed("nope") != nil {
		t.Fatalf("FlagsNamed = %+v", ds)
	}
	if a, ok := tv.Arg("b"); !ok || a.Value != 2 {
		t.Fatalf("Arg(b) = %+v", a)
	}
	if _, ok := tv.Arg("zzz"); ok {
		t.Fatal("Arg(zzz)")
	}
	oa := tv.OrderedArgs()
	if len(oa) != 2 || oa[0].Name != "b" || oa[1].Name != "a" || oa[1].VarType != "Int" {
		t.Fatalf("OrderedArgs = %+v", oa)
	}
	if !tv.Variables[0].IsAnonymous() {
		t.Fatal("IsAnonymous")
	}
	flags, _ := tagkit.ParseTagValue(`,a`)
	if flags.HasField() || flags.Args != nil || flags.ArgKeys != nil || flags.Variables != nil || flags.Flags == nil {
		t.Fatalf("flags-only zero values: %+v", flags)
	}
	if (*tagkit.Literal)(nil).Dimension() != 0 {
		t.Fatal("nil Literal Dimension")
	}
}

// TestStringManual 手动构造（无 Literal / 无 ArgKeys）的 TagValue 也能输出可解析的写法
func TestStringManual(t *testing.T) {
	tv := &tagkit.TagValue{
		Name: "f",
		Args: map[string]tagkit.ArgValue{
			"s": {Type: tagkit.ArgTypeLiteral, Value: "it's"},
			"i": {Type: tagkit.ArgTypeLiteral, Value: 3},
			"f": {Type: tagkit.ArgTypeLiteral, Value: 2.0},
			"g": {Type: tagkit.ArgTypeLiteral, Value: 1.5},
			"b": {Type: tagkit.ArgTypeLiteral, Value: true},
			"n": {Type: tagkit.ArgTypeLiteral, Value: nil},
			"l": {Type: tagkit.ArgTypeLiteral, Value: []interface{}{1, 2}},
			"v": {Type: tagkit.ArgTypeVariable, Value: "$v:Int"},
			"x": {Type: tagkit.ArgTypeLiteral, Value: struct{}{}},
		},
		Flags: []tagkit.FlagInfo{{Name: "k", Value: "v"}, {Name: "on", IsBoolean: true}},
	}
	want := `f(b:true,f:2.0,g:1.5,i:3,l:[1,2],n:null,s:'it\'s',v:$v:Int,x:''),k='v',on`
	if got := tv.String(); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if _, err := tagkit.ParseTagValue(want); err != nil {
		t.Fatal(err)
	}
	if s := (&tagkit.TagValue{}).String(); s != "" {
		t.Fatalf("empty: %q", s)
	}
}

// TestDeterministic 同一输入多次解析结果完全一致（v1.0.x 的参数与变量顺序随 map 遍历变化）
func TestDeterministic(t *testing.T) {
	const in = `f(z:1,y:$y:Int,x:'a',w:$w,v:[1],u:E,t:null,s:true),k=1`
	first, err := tagkit.ParseTagValue(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"z", "y", "x", "w", "v", "u", "t", "s"}
	if !reflect.DeepEqual(first.ArgKeys, want) || first.Variables[0].Key != "y" || first.Variables[1].Key != "w" {
		t.Fatalf("order: %v", first.ArgKeys)
	}
	for i := 0; i < 50; i++ {
		again, _ := tagkit.ParseTagValue(in)
		if again.String() != first.String() || !reflect.DeepEqual(again, first) {
			t.Fatal("non-deterministic result")
		}
	}
}

// FuzzParseTagValue 不变式：不 panic；失败时返回 *SyntaxError 且结果为 nil；成功时所有名称为合法 ASCII 标识符，
// 且 String() 可再次解析并得到完全相同的结果
func FuzzParseTagValue(f *testing.F) {
	seeds := []string{
		``, `'\`, `"\`, `single()`, `(age:18,sex:$:String,sum:$),inline`,
		`name(age:$age:Int=18, first:10),inline=3,union=false`,
		`matrix(data:$data:[[Int]]=[[1,2],[3,4]], scale:$scale:Float=1.0),normalize`,
		`quote(str:$str:String='$1.2')`, `,verbose`, `,tags=['a','b']`,
		`f(e:CREATED_AT,s:'it\'s',n:null,x:-1.5e3,l:[[true],[]])`, `f(ids:$ids:[ID!]=[1,'x'])`, `f(ä:1)`, `f(a:1 b:2)`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		tv, err := tagkit.ParseTagValue(input)
		if err != nil {
			var se *tagkit.SyntaxError
			if !errors.As(err, &se) || tv != nil {
				t.Fatalf("%q: want *SyntaxError and nil result, got %T %v", input, err, tv)
			}
			if se.Pos < 0 || se.Pos > len(input) {
				t.Fatalf("%q: position %d out of range", input, se.Pos)
			}
			return
		}
		if tv == nil {
			t.Fatalf("%q: nil result without error", input)
		}
		checkName := func(what, s string, allowEmpty bool) {
			if s == "" && allowEmpty {
				return
			}
			if !isIdent(s) {
				t.Fatalf("%q: invalid %s name %q", input, what, s)
			}
		}
		checkName("field", tv.Name, true)
		if len(tv.ArgKeys) != len(tv.Args) {
			t.Fatalf("%q: ArgKeys %v vs Args %d", input, tv.ArgKeys, len(tv.Args))
		}
		for _, k := range tv.ArgKeys {
			checkName("argument", k, false)
			if _, ok := tv.Args[k]; !ok {
				t.Fatalf("%q: ArgKeys has %q not in Args", input, k)
			}
		}
		for _, v := range tv.Variables {
			checkName("variable", v.Name, true)
		}
		for _, fl := range tv.Flags {
			checkName("flag", fl.Name, false)
		}
		assertRoundTrip(t, input, tv)
	})
}

// isIdent 是否为 ASCII 标识符 [A-Za-z_][A-Za-z0-9_]*
func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

// BenchmarkParseTagValue 典型 struct tag 的完整解析
func BenchmarkParseTagValue(b *testing.B) {
	inputs := map[string]string{
		"small":   `id`,
		"typical": `products(first:$first:Int=10, query:$q:String, sortKey:CREATED_AT),alias=items,directive='include(if:$w)'`,
		"complex": `matrix(data:$data:[[Int!]!]!=[[1,2],[3,4]], scale:$scale:Float=1.5, tags:[A,B,'c'], n:null),normalize,limit=10`,
	}
	for name, in := range inputs {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := tagkit.ParseTagValue(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
