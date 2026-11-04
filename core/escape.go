package core

import "strings"

// Unescape 还原字符串字面量（引号内原文）中的转义：\\ → \、\' → '、\" → "；
// 其余反斜杠序列（如 \n、\t）原样保留（反斜杠与后续字符都保留），与 v1.0.x 的输出兼容
func Unescape(s string) string {
	i := strings.IndexByte(s, '\\')
	if i < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	b.WriteString(s[:i])
	for ; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			switch n := s[i+1]; n {
			case '\\', '\'', '"':
				b.WriteByte(n)
			default:
				b.WriteByte(c)
				b.WriteByte(n)
			}
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// Quote 将字符串编码为单引号字符串字面量，\ 与 ' 被转义；对任意 s，解析 Quote(s) 得到的值等于 s
func Quote(s string) string {
	if strings.IndexByte(s, '\\') < 0 && strings.IndexByte(s, '\'') < 0 {
		return "'" + s + "'"
	}
	var b strings.Builder
	b.Grow(len(s) + 4)
	b.WriteByte('\'')
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '\\' || c == '\'' {
			b.WriteByte('\\')
		}
		b.WriteByte(s[i])
	}
	b.WriteByte('\'')
	return b.String()
}
