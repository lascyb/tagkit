package tagkit

import (
	"sort"
	"strconv"
	"strings"

	"github.com/lascyb/tagkit/core"
)

// formatValue 将手动构造（无 Literal）的 Go 值格式化为 tag 写法：string 加引号，nil 为 null，切片逐元素格式化
func formatValue(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return core.Quote(x)
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case float64:
		s := strconv.FormatFloat(x, 'g', -1, 64)
		if !strings.ContainsAny(s, ".eE") {
			s += ".0"
		}
		return s
	case []interface{}:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = formatValue(e)
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return core.Quote("")
	}
}

// sortedKeys 返回 map 的键，按字典序排列
func sortedKeys(m map[string]ArgValue) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
