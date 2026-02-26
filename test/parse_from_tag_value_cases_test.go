package tagkit_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/lascyb/tagkit"
)

// TagValueCases 按「字段 → 参数 → 标记」分模块递进：每类从简到繁，覆盖语法规则。
// expect 格式：fields:N,flags:M；expect:"error" 表示期望报错。tag 为空时输入为 ""。
type TagValueCases struct {
	// ========== 一、字段（递进：空 → name → name() → (args) → name(args)）==========
	EmptyInput     string `tag:"" expect:"fields:0,flags:0"`                                   // 1. 空输入
	FieldNameOnly  string `tag:"single2" expect:"fields:1,flags:0"`                            // 2. 裸标识符 → 无参字段
	FieldNoArgs    string `tag:"single()" expect:"fields:1,flags:0"`                           // 3. name() 无参
	AnonymousField string `tag:"(age:18,sex:$:String,sum:$),inline" expect:"fields:1,flags:1"` // 4. 匿名字段 (args)

	// ========== 二、参数（递进：字面量 → $ → $:Type → $:Type=default → 匿名变量 → 数组）==========
	ArgLiteralOnly           string `tag:"foo(x:1,y:2,z:true,label:hello)" expect:"fields:1,flags:0"`                                                   // 1. 仅字面量
	ArgVariableNoType        string `tag:"bare(id:$id,name:$name)" expect:"fields:1,flags:0"`                                                           // 2. 变量无类型 $
	ArgVariableType          string `tag:"bar(id:$id:Int,name:$name:String)" expect:"fields:1,flags:0"`                                                 // 3. 变量+类型 $:Type
	ArgVariableDefault       string `tag:"name(age:$age:Int=18, first:10),inline=3,union=false" expect:"fields:1,flags:2"`                              // 4. 变量+类型+默认 $:Type=default
	ArgAnonymousVar          string `tag:"(sex:$:String!,sum:$,age:18),inline" expect:"fields:1,flags:1"`                                               // 5. 匿名变量 $:Type、key:$
	ArgArrayQuoted           string `tag:"tags(items:$items:[String]=['a','b','c'], count:$count:Int=5),flatten" expect:"fields:1,flags:1"`             // 6. 数组默认 ['a','b','c']
	ArgArrayUnquoted         string `tag:"tags(items:$items:[String]=[a,b,c], count:$count:Int=5),flatten=123,union=false" expect:"fields:1,flags:2"`   // 7. 数组 [a,b,c]
	ArgArray1Elem            string `tag:"one(x:$x:[Int]=[1], s:$s:[String]=['only'])" expect:"fields:1,flags:0"`                                       // 8. 单元素数组
	ArgArray2D               string `tag:"matrix(data:$data:[[Int]]=[[1,2],[3,4]], scale:$scale:Float=1.0),normalize" expect:"fields:1,flags:1"`        // 9. 二维数组
	ArgArray3D               string `tag:"deep(nested:$nested:[[[Int]]]=[[[1,2]],[[3,4]]]),test" expect:"fields:1,flags:1"`                             // 10. 三维数组
	ArgEmptyArrayStr         string `tag:"empty(arr:$arr:[Int]=[], str:$str:String=''),check" expect:"fields:1,flags:1"`                                // 11. 空数组、空串
	ArgStringWithSpecialChar string `tag:"quote(str:$str:String='$1.2')" expect:"fields:1,flags:0"`                                                     // 12. 字符串默认值含特殊字符（单引号包裹）
	ArgLiteralFloatBool      string `tag:"op(scale:1.5, enabled:true, off:false)" expect:"fields:1,flags:0"`                                            // 13. 字面量 float/bool
	ArgNonNullList           string `tag:"query(id_s:$id_s:[Int!]!=[1,2,3], names:$names:[String]=['foo','bar']),cache=true" expect:"fields:1,flags:1"` // 14. [Int!]! 等
	ArgMixedComplex          string `tag:"mixed(data:$data:[[[Int]]]=[[[1,2]],[[3,4]]], label:$label:String=\"test\"),flag" expect:"fields:1,flags:1"`  // 15. 混合复杂参数

	// ========== 三、标记（递进：仅布尔 → 仅键值 → 混合；前导逗号规则）==========
	FlagOnlyBoolean           string `tag:",verbose" expect:"fields:0,flags:1"`                                   // 1. 仅布尔（需前导逗号）
	FlagOnlyBooleanMulti      string `tag:",verbose,debug,dryRun" expect:"fields:0,flags:3"`                      // 2. 多个布尔
	FlagOnlyKeyValueWithComma string `tag:",limit=10" expect:"fields:0,flags:1"`                                  // 3. 键值有前导逗号
	FlagOnlyKeyValueNoComma   string `tag:"limit=10" expect:"fields:0,flags:1"`                                   // 4. 键值可省略前导逗号
	FlagOnlyKeyValueMulti     string `tag:"a=1,b=2" expect:"fields:0,flags:2"`                                    // 5. 多个键值无前导逗号
	FlagKeyValueThenBoolean   string `tag:"limit=10,verbose" expect:"fields:0,flags:2"`                           // 6. 键值+布尔
	FlagStringValues          string `tag:",name=\"hello world\",title='single quote'" expect:"fields:0,flags:2"` // 7. 字符串值
	FlagArrayValues           string `tag:",items=[1,2,3],names=['a','b']" expect:"fields:0,flags:2"`             // 8. 数组值
	FlagFloatAndBool          string `tag:",ratio=1.5,enabled=true,disabled=false" expect:"fields:0,flags:3"`     // 9. 数值与布尔
	FlagLeadingCommaOnly      string `tag:",flag=false" expect:"fields:0,flags:1"`                                // 10. 仅标记时前导逗号
	FlagConsecutiveCommas     string `tag:",,flag" expect:"fields:0,flags:1"`                                     // 11. 连续逗号
	FlagOnlyKeyValueFour      string `tag:",a=1,b=2,c=hello,d=true" expect:"fields:0,flags:4"`                    // 12. 四个键值

	// ========== 四、字段+标记 组合 ==========
	FieldThenFlag          string `tag:"name(),flag" expect:"fields:1,flags:1"`                       // 字段后跟标记
	FieldWithArgsThenFlags string `tag:"name(age:18,sex:man),inline,union" expect:"fields:1,flags:2"` // 有参字段+标记
	FieldTrailingComma     string `tag:"name()," expect:"fields:1,flags:0"`                           // 尾逗号
	FieldNoArgsThenFlags   string `tag:"noop(),a,b=2,c=true" expect:"fields:1,flags:3" `              // 无参+多标记
}

// TestParseTagValue_FromTagValueCases 遍历 TagValueCases 各字段的 tag value 作为输入，按 expect 断言解析结果，贴合真实使用场景
func TestParseTagValue_FromTagValueCases(t *testing.T) {
	cases := TagValueCases{}
	typ := reflect.TypeOf(cases)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagVal := field.Tag.Get("tag")
		expect := field.Tag.Get("expect")
		if tagVal == "" && expect == "" {
			continue
		}
		name := field.Name
		input := tagVal
		if tagVal == "" {
			input = ""
		}
		t.Run(name, func(t *testing.T) {
			result, err := tagkit.ParseTagValue(input)
			if expect == "error" {
				if err == nil {
					t.Fatalf("期望解析报错，却成功: %s", input)
				}
				return
			}
			if err != nil {
				t.Fatalf("解析失败: %v\n输入: %s", err, input)
			}
			if result == nil {
				t.Fatal("解析结果不应为 nil")
			}
			wantFields, wantFlags, ok := parseExpect(expect)
			if !ok {
				return
			}
			hasField := result.Args != nil
			if (wantFields == 0 && hasField) || (wantFields == 1 && !hasField) {
				t.Errorf("字段: 期望 wantFields=%d，得到 hasField=%v", wantFields, hasField)
			}
			if len(result.Flags) != wantFlags {
				t.Errorf("标记位数: 期望 %d，得到 %d", wantFlags, len(result.Flags))
			}

			printTagValueResult(input, result)
		})
	}
}

// printTagValueResult 将输入与解析结果格式化为易读的树形输出
func printTagValueResult(input string, result *tagkit.TagValue) {
	const (
		indent = "  "
		bullet = "•"
		sep    = "───────────────────────────────────────────────────────────────"
		empty  = "(空)"
	)
	fmt.Println()
	fmt.Println(sep)
	if input == "" {
		fmt.Printf("  输入 %s\n", empty)
	} else {
		fmt.Printf("  输入  %s\n", input)
	}
	fmt.Println(sep)

	if result.Args != nil {
		fmt.Printf("\n  📦 字段  %s\n", result.Name)
		fmt.Printf("  %s参数:\n", indent)
		for k, v := range result.Args {
			if v.Type == "variable" {
				var part string
				if v.VarType != "" {
					part = fmt.Sprintf("$%s (%s)", v.VarName, v.VarType)
				} else {
					part = fmt.Sprintf("$%s (未设置类型)", v.VarName)
				}
				if v.HasDefault {
					if v.IsArrayDefault {
						part += fmt.Sprintf(" = %v  [%d 维数组]", v.DefaultVal, v.Dimension)
					} else {
						part += fmt.Sprintf(" = %v", v.DefaultVal)
					}
				}
				fmt.Printf("  %s%s %s  →  %s\n", indent, bullet, k, part)
			} else {
				fmt.Printf("  %s%s %s  →  %v  (%T)\n", indent, bullet, k, v.Value, v.Value)
			}
		}
		if len(result.Variables) > 0 {
			fmt.Printf("  %s变量详情:\n", indent)
			for _, v := range result.Variables {
				var part string
				if v.VarType != "" {
					part = fmt.Sprintf("$%s: %s", v.Name, v.VarType)
				} else {
					part = fmt.Sprintf("$%s (未设置类型)", v.Name)
				}
				if v.HasDefault {
					if v.IsArrayDefault {
						part += fmt.Sprintf(" = %v  [%s, %d 维]", v.DefaultValue, v.ArrayType, v.Dimension)
					} else {
						part += fmt.Sprintf(" = %v", v.DefaultValue)
					}
				}
				fmt.Printf("  %s%s %s  (参数 %s)\n", indent, bullet, part, v.Key)
			}
		}
		fmt.Println()
	}

	if len(result.Flags) > 0 {
		fmt.Printf("  🏷 标记:\n")
		for _, f := range result.Flags {
			if f.IsBoolean {
				fmt.Printf("  %s%s %s  →  true  (布尔)\n", indent, bullet, f.Name)
			} else {
				fmt.Printf("  %s%s %s  →  %v  (%s)\n", indent, bullet, f.Name, f.Value, f.ValueType)
			}
		}
		fmt.Println()
	}

	if result.Args == nil && len(result.Flags) == 0 {
		fmt.Printf("  %s\n\n", empty)
	}
}

// parseExpect 解析 expect 字符串 "fields:N,flags:M"，返回 N、M 与是否解析成功
func parseExpect(expect string) (fields, flags int, ok bool) {
	if expect == "" {
		return 0, 0, false
	}
	for _, part := range strings.Split(expect, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "fields:") {
			n, err := strconv.Atoi(strings.TrimPrefix(part, "fields:"))
			if err != nil {
				return 0, 0, false
			}
			fields = n
		}
		if strings.HasPrefix(part, "flags:") {
			n, err := strconv.Atoi(strings.TrimPrefix(part, "flags:"))
			if err != nil {
				return 0, 0, false
			}
			flags = n
		}
	}
	return fields, flags, true
}
