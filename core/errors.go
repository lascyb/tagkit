package core

import "fmt"

// SyntaxError 结构化的解析错误；Pos 为输入中的字节偏移（从 0 开始），Msg 为不含位置的错误描述。
// 词法错误（非法字符、未闭合字符串、非法数字）与语法/语义错误（缺少逗号、重复参数、默认值与类型不符等）均使用该类型，
// 可通过 errors.As 取得
type SyntaxError struct {
	Pos int    // 出错位置（字节偏移）
	Msg string // 错误描述
}

// Error 实现 error 接口，格式为 "<msg> at position <pos>"
func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s at position %d", e.Msg, e.Pos)
}

// errorf 构造 *SyntaxError
func errorf(pos int, format string, args ...interface{}) *SyntaxError {
	return &SyntaxError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}
