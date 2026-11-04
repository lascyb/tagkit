package tagkit_test

import (
	"strings"
	"testing"

	"github.com/lascyb/tagkit"
)

// TestParseTagValueUnterminatedString 回归测试：v1.0.6 对以反斜杠结尾的未闭合字符串会 panic，现应返回错误
func TestParseTagValueUnterminatedString(t *testing.T) {
	for _, input := range []string{`'\`, `"\`, `name(a:'\`, `name(a:"abc`, `,flag='x\`, `q(a:$a:String='\`} {
		tv, err := tagkit.ParseTagValue(input)
		if err == nil || !strings.Contains(err.Error(), "unterminated string") {
			t.Fatalf("%q: expected unterminated string error, got tv=%+v err=%v", input, tv, err)
		}
	}
}

// FuzzParseTagValue 任意输入下 ParseTagValue 不得 panic；成功时结果不得为 nil
func FuzzParseTagValue(f *testing.F) {
	seeds := []string{
		``, `'\`, `"\`, `single()`, `(age:18,sex:$:String,sum:$),inline`,
		`name(age:$age:Int=18, first:10),inline=3,union=false`,
		`matrix(data:$data:[[Int]]=[[1,2],[3,4]], scale:$scale:Float=1.0),normalize`,
		`quote(str:$str:String='$1.2')`, `,verbose`, `,tags=['a','b']`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		tv, err := tagkit.ParseTagValue(input)
		if err == nil && tv == nil {
			t.Fatalf("%q: nil result without error", input)
		}
	})
}
