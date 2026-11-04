package tagkit_test

import (
	"errors"
	"fmt"

	"github.com/lascyb/tagkit"
)

// 解析字段调用、参数与标记位；ArgKeys 保留参数书写顺序，Kind 区分字符串与枚举
func ExampleParseTagValue() {
	tv, err := tagkit.ParseTagValue(`products(first:$first:Int=10, sortKey:CREATED_AT, query:'tag:new'),alias=items,inline`)
	if err != nil {
		panic(err)
	}
	fmt.Println("字段:", tv.Name)
	for _, a := range tv.OrderedArgs() {
		if a.Type == tagkit.ArgTypeVariable {
			fmt.Printf("  %s = $%s (%s) 默认 %v\n", a.Name, a.VarName, a.VarType, a.DefaultVal)
		} else {
			fmt.Printf("  %s = %v (%s)\n", a.Name, a.Value, a.Kind)
		}
	}
	alias, _ := tv.Flag("alias")
	fmt.Println("alias:", alias.Value, "inline:", tv.HasFlag("inline"))
	// Output:
	// 字段: products
	//   first = $first (Int) 默认 10
	//   sortKey = CREATED_AT (enum)
	//   query = tag:new (string)
	// alias: items inline: true
}

// String 输出规范写法，可被重新解析为相同结果
func ExampleTagValue_String() {
	tv, _ := tagkit.ParseTagValue(`f( b : "it's" , a : [1, 2,] ), limit=10 ,verbose`)
	fmt.Println(tv.String())
	// Output:
	// f(b:'it\'s',a:[1,2]),limit=10,verbose
}

// 同名标记允许重复，FlagsNamed 按书写顺序返回
func ExampleTagValue_FlagsNamed() {
	tv, _ := tagkit.ParseTagValue(`email,directive='include(if:$a)',directive='cached(ttl:60)'`)
	for _, f := range tv.FlagsNamed("directive") {
		fmt.Println(f.Value)
	}
	// Output:
	// include(if:$a)
	// cached(ttl:60)
}

// 解析错误为 *tagkit.SyntaxError，Pos 为字节偏移
func ExampleSyntaxError() {
	_, err := tagkit.ParseTagValue(`f(a:1 b:2)`)
	var se *tagkit.SyntaxError
	if errors.As(err, &se) {
		fmt.Println(se.Pos, se.Msg)
	}
	// Output:
	// 6 expected ',' or ')' after argument, got identifier "b"
}
