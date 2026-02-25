package tagkit_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/lascyb/tagkit"
)

// TagValueCases 定义全场景测试用例：每个字段的 tag 值为真实使用场景下的 tag value（与业务中 struct field 的 tag 一致），作为 ParseTagValue 的输入。
// 语义：字段 = 字段名 + 参数，如 name(age:18,sex:man)；字段名可为空即 (age:18,sex:man)；存在标记时不可省略逗号，例如 ",inline"。
// expect 格式：fields:N,flags:M 表示期望解析得到 N 个字段、M 个标记位；expect:"error" 表示期望解析报错。
type TagValueCases struct {
	// 字段名 + 参数，后跟标记：name 为字段名，age/sex 为参数；逗号分隔字段与标记
	FieldNameAndParamsWithFlags string `tag:"name(age:18,sex:man),inline,union" expect:"fields:1,flags:2"`
	// 唯一字段：变量参数 + 字面量参数 + 标记位（字段唯一，仅保留一个 name(...)）
	QueryListWithInlineAndUnion string `tag:"name(age:$age:Int = 18, sex:$sex_value:[String!]!, first:10),inline=3,union=false" expect:"fields:1,flags:2"`
	TagsWithFlatten             string `tag:"tags(items:$items:[String]=['a','b','c'], count:$count:Int=5),flatten" expect:"fields:1,flags:1"`
	TagsWithFlattenAndUnion     string `tag:"tags(items:$items:[String]=[a,b,c], count:$count:Int=5),flatten=123,union=false" expect:"fields:1,flags:2"`
	MatrixWithNormalize         string `tag:"matrix(data:$data:[[Int]]=[[1,2],[3,4]], scale:$scale:Float=1.0),normalize" expect:"fields:1,flags:1"`
	ConfigPortsAndHosts         string `tag:"config(ports:$ports:[Int]=[8080,8081,8082], hosts:$hosts:[String]=['localhost','0.0.0.0']),strict" expect:"fields:1,flags:1"`
	QueryIdsAndNamesWithCache   string `tag:"query(id_s:$id_s:[Int!]!=[1,2,3], names:$names:[String]=['foo','bar']),cache=true" expect:"fields:1,flags:1"`
	DeepNested3DArray           string `tag:"deep(nested:$nested:[[[Int]]]=[[[1,2]],[[3,4]]]),test" expect:"fields:1,flags:1"`
	EmptyArrayAndString         string `tag:"empty(arr:$arr:[Int]=[], str:$str:String=''),check" expect:"fields:1,flags:1"`
	Mixed3DArrayAndLabel        string `tag:"mixed(data:$data:[[[Int]]]=[[[1,2]],[[3,4]]], label:$label:String=\"test\"),flag" expect:"fields:1,flags:1"`
	// 存在标记时不可省略逗号：仅标记时写 ",inline" 等
	LeadingCommaOnlyFlag string `tag:",flag=false" expect:"fields:0,flags:1"`
	// 字段名可为空：匿名字段 (age:18,sex:man)
	AnonymousField             string `tag:"(age:18,sex:$:String,sum:$),inline" expect:"fields:1,flags:1"`
	AnonymousArgs              string `tag:"(sex:$:String!,sum:$,age:18),inline" expect:"fields:1,flags:1"`
	FieldNoArgs                string `tag:"single()" expect:"fields:1,flags:0"`
	OnlyBooleanFlags           string `tag:",verbose,debug,dryRun" expect:"fields:0,flags:3"`
	OnlyKeyValueFlags          string `tag:",a=1,b=2,c=hello,d=true" expect:"fields:0,flags:4"`
	FieldLiteralArgsOnly       string `tag:"foo(x:1,y:2,z:true,label:hello)" expect:"fields:1,flags:0"`
	VariableNoDefault          string `tag:"bar(id:$id:Int,name:$name:String)" expect:"fields:1,flags:0"`
	FlagStringValues           string `tag:",name=\"hello world\",title='single quote'" expect:"fields:0,flags:2"`
	FlagArrayValues            string `tag:",items=[1,2,3],names=['a','b']" expect:"fields:0,flags:2"`
	SingleElementArrayDefaults string `tag:"one(x:$x:[Int]=[1], s:$s:[String]=['only'])" expect:"fields:1,flags:0"`
	TrailingComma              string `tag:"name()," expect:"fields:1,flags:0"`
	TrailingCommaThenFlag      string `tag:"name(),flag" expect:"fields:1,flags:1"`
	ConsecutiveCommas          string `tag:",,flag" expect:"fields:0,flags:1"`
	FieldFloatAndBoolLiterals  string `tag:"op(scale:1.5, enabled:true, off:false)" expect:"fields:1,flags:0"`
	FlagFloatAndBool           string `tag:",ratio=1.5,enabled=true,disabled=false" expect:"fields:0,flags:3"`
	SingleFieldThenFlags       string `tag:"noop(),a,b=2,c=true" expect:"fields:1,flags:3"`
	SingleBooleanFlag          string `tag:",verbose" expect:"fields:0,flags:1"`
	SingleKeyValueFlag         string `tag:",limit=10" expect:"fields:0,flags:1"`
}

// TestParseTagValue_FromTagValueCases 遍历 TagValueCases 各字段的 tag value 作为输入，按 expect 断言解析结果，贴合真实使用场景
func TestParseTagValue_FromTagValueCases(t *testing.T) {
	cases := TagValueCases{}
	typ := reflect.TypeOf(cases)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagVal := field.Tag.Get("tag")
		expect := field.Tag.Get("expect")
		if tagVal == "" {
			continue
		}
		name := field.Name
		t.Run(name, func(t *testing.T) {
			result, err := tagkit.ParseTagValue(tagVal)
			if expect == "error" {
				if err == nil {
					t.Fatalf("期望解析报错，却成功: %s", tagVal)
				}
				return
			}
			if err != nil {
				t.Fatalf("解析失败: %v\n输入: %s", err, tagVal)
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

			fmt.Printf("═══════════════════════════════════════════════════════════════════\n")
			fmt.Printf("输入: %s\n\n", tagVal)

			fmt.Printf("📦 解析结果:\n\n")
			marshal, err := json.Marshal(result)
			if err != nil {
				return
			}
			fmt.Println(string(marshal))

			if result.Args != nil {
				fmt.Printf("  字段调用: %s\n", result.Name)
				fmt.Printf("      参数:\n")
				for k, v := range result.Args {
					if v.Type == "variable" {
						fmt.Printf("        • %s = $%s (%s)", k, v.VarName, v.VarType)
						if v.HasDefault {
							if v.IsArrayDefault {
								fmt.Printf(" = %v (%d维数组)", v.DefaultVal, v.Dimension)
							} else {
								fmt.Printf(" = %v", v.DefaultVal)
							}
						}
						fmt.Println()
					} else {
						fmt.Printf("        • %s = %v (%T)\n", k, v.Value, v.Value)
					}
				}
				if len(result.Variables) > 0 {
					fmt.Printf("      📌 变量详情:\n")
					for _, v := range result.Variables {
						fmt.Printf("        • $%s: %s", v.Name, v.VarType)
						if v.HasDefault {
							marshal, err := json.Marshal(v.DefaultValue)
							if err != nil {
								return
							}
							if v.IsArrayDefault {
								fmt.Printf(" = %v (%s, %d维)", string(marshal), v.ArrayType, v.Dimension)
							} else {
								fmt.Printf(" = %v", string(marshal))
							}
						}
						fmt.Printf(" (用于参数 %s)\n", v.Key)
					}
				}
				fmt.Println()
			}

			if len(result.Flags) > 0 {
				fmt.Printf("  🏷️  标记位:\n")
				for _, f := range result.Flags {
					if f.IsBoolean {
						fmt.Printf("    • %s = true (布尔标记)\n", f.Name)
					} else {
						fmt.Printf("    • %s = %v (%s)\n", f.Name, f.Value, f.ValueType)
					}
				}
			}
			fmt.Println()
		})
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
