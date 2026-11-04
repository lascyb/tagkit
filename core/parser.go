package core

import (
	"strconv"
	"strings"
)

// ============ 解析器 ============

// Parser 递归下降解析器，消费 Lexer 产出的 Token 流并构建 AST；用于将 tag 字符串解析为 Node 列表
type Parser struct {
	lexer     Lexer // 按值内嵌，减少一次分配
	curToken  Token
	peekToken Token
}

// NewParser 根据输入字符串创建 Parser 并预读两个 Token
func NewParser(input string) *Parser {
	p := &Parser{lexer: Lexer{input: input}}
	p.lexer.readChar()
	p.nextToken()
	p.nextToken()
	return p
}

// nextToken 将 peekToken 变为 curToken 并读取下一个 Token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// unexpected 构造「期望 X，实际得到当前 Token」的语法错误
func (p *Parser) unexpected(want string) *SyntaxError {
	return errorf(p.curToken.Pos, "expected %s, got %s", want, p.curToken.describe())
}

// Parse 解析整个输入，返回 AST 节点列表；出错时返回 *SyntaxError。
//
// 顶层规则：元素之间必须用逗号分隔；允许前导、连续与末尾逗号（视为空位）。
// 标记位存在赋值（name=value）时允许省略前导逗号；仅布尔标记（裸 ident）时需本元素前有逗号才解析为标记位，
// 否则解析为无参字段（如 single2 为字段、,inline 为标记）。
func (p *Parser) Parse() ([]Node, error) {
	nodes, err := p.parse()
	// 词法错误优先：非法字符/未闭合字符串会截断后续输入，由此产生的语法错误不如词法错误准确
	if lexErr := p.lexer.Err(); lexErr != nil {
		return nil, lexErr
	}
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// parse 为 Parse 的实际实现，词法错误由 Parse 统一检查
func (p *Parser) parse() ([]Node, error) {
	var nodes []Node
	afterComma := false // 前导逗号后为 true：仅在此情况下将「无 ( 无 = 的 ident」解析为布尔标记位
	needSep := false    // 上一个元素之后尚未出现逗号

	for p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenComma {
			p.nextToken()
			afterComma = true
			needSep = false
			continue
		}
		if p.curToken.Type == TokenRParen {
			return nil, errorf(p.curToken.Pos, "unmatched ')'")
		}
		if needSep {
			return nil, p.unexpected("',' between elements")
		}
		node, err := p.parseElement(afterComma)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
		afterComma = false
		needSep = true
	}
	return nodes, nil
}

// parseElement 解析一个顶层元素。name=value 始终为标记位（可省略前导逗号）；无 ( 无 = 时，仅 afterComma 为 true 时为布尔标记位，否则为无参字段 name()。
func (p *Parser) parseElement(afterComma bool) (Node, error) {
	var name string
	pos := p.curToken.Pos
	switch p.curToken.Type {
	case TokenLParen:
	case TokenIdent:
		name = p.curToken.Value
		p.nextToken()
	default:
		return nil, p.unexpected("identifier or '('")
	}

	switch p.curToken.Type {
	case TokenLParen:
		return p.parseFieldCall(name, pos)
	case TokenEqual:
		return p.parseFlagWithValue(name, pos) // 存在赋值时允许省略前导逗号
	}
	if afterComma {
		return &FlagNode{Name: name, Pos: pos}, nil
	}
	return &FieldNode{Name: name, Args: map[string]Node{}, Pos: pos}, nil
}

// parseFieldCall 解析 name(arg1:val1, ...) 形式的字段调用，当前 Token 为 (。
// 参数之间必须有逗号，允许末尾逗号 f(a:1,)；重复的参数名返回错误
func (p *Parser) parseFieldCall(name string, pos int) (*FieldNode, error) {
	p.nextToken() // (
	field := &FieldNode{Name: name, Args: make(map[string]Node), ArgList: make([]*ArgNode, 0, 4), Pos: pos}

	for {
		if p.curToken.Type == TokenRParen {
			p.nextToken()
			return field, nil
		}
		if p.curToken.Type != TokenIdent {
			return nil, p.unexpected("parameter name or ')'")
		}
		key, keyPos := p.curToken.Value, p.curToken.Pos
		p.nextToken()

		if p.curToken.Type != TokenColon {
			return nil, p.unexpected("':' after parameter name")
		}
		p.nextToken()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if _, dup := field.Args[key]; dup {
			return nil, errorf(keyPos, "duplicate argument %q", key)
		}
		field.Args[key] = value
		field.ArgList = append(field.ArgList, &ArgNode{Name: key, Value: value, Pos: keyPos})

		switch p.curToken.Type {
		case TokenComma:
			p.nextToken()
		case TokenRParen:
		default:
			return nil, p.unexpected("',' or ')' after argument")
		}
	}
}

// parseFlagWithValue 解析 name=value 形式的标记，当前 Token 为 =
func (p *Parser) parseFlagWithValue(name string, pos int) (*FlagNode, error) {
	p.nextToken()
	value, err := p.parseLiteralOrArray()
	if err != nil {
		return nil, err
	}
	return &FlagNode{Name: name, Value: value, Pos: pos}, nil
}

// parseValue 解析参数值：变量 $x:Type=default、字面量或数组字面量
func (p *Parser) parseValue() (Node, error) {
	if p.curToken.Type == TokenDollar {
		return p.parseVariable()
	}
	return p.parseLiteralOrArray()
}

// parseVariable 解析变量：$name、$name:Type=default 等；类型与默认值均可选，不写 :Type 时 VarType 为 nil；
// 同时声明类型与默认值时校验默认值与类型（维度、可空性、内置标量）是否相符
func (p *Parser) parseVariable() (*VariableNode, error) {
	v := &VariableNode{Pos: p.curToken.Pos}
	p.nextToken() // $

	switch p.curToken.Type {
	case TokenIdent:
		v.Name = p.curToken.Value
		p.nextToken()
	case TokenNumber, TokenString, TokenBool:
		return nil, errorf(p.curToken.Pos, "invalid variable name %s: must be an identifier", p.curToken.describe())
	}
	// 匿名变量：$ 后无标识符则变量名为空
	if p.curToken.Type == TokenColon {
		p.nextToken()
		t, err := p.parseType()
		if err != nil {
			return nil, err
		}
		v.VarType = t
	}

	if p.curToken.Type == TokenEqual {
		p.nextToken()
		var def Node
		var err error
		if v.VarType == nil {
			def, err = p.parseLiteralOrArray()
		} else if p.curToken.Type == TokenLBracket {
			// 声明了类型时以类型为准逐元素校验（如 [ID] 允许混用整数与字符串），不做通用的类别一致性校验
			def, err = p.parseArrayLiteral()
		} else {
			def, err = p.parseLiteral()
		}
		if err != nil {
			return nil, err
		}
		if v.VarType != nil {
			if err := checkDefault(v.VarType, def); err != nil {
				return nil, err
			}
		}
		v.DefaultValue = def
	}
	v.Raw = v.String()
	return v, nil
}

// parseLiteralOrArray 根据当前 Token 解析字面量或数组字面量；数组会校验元素维度与类别一致（见 checkArray）
func (p *Parser) parseLiteralOrArray() (Node, error) {
	if p.curToken.Type == TokenLBracket {
		arr, err := p.parseArrayLiteral()
		if err != nil {
			return nil, err
		}
		if _, err := checkArray(arr); err != nil {
			return nil, err
		}
		return arr, nil
	}
	return p.parseLiteral()
}

// parseArrayLiteral 解析 [ ... ] 数组字面量，支持嵌套；当前 Token 为 [。
// 元素之间必须有逗号，允许末尾逗号 [1,2,]；元素可为字符串、数字、布尔、枚举、null 或嵌套数组
func (p *Parser) parseArrayLiteral() (*ArrayNode, error) {
	array := &ArrayNode{Elements: make([]Node, 0), Pos: p.curToken.Pos}
	p.nextToken() // [

	for {
		if p.curToken.Type == TokenRBracket {
			p.nextToken()
			return array, nil
		}
		var elem Node
		var err error
		switch p.curToken.Type {
		case TokenLBracket:
			elem, err = p.parseArrayLiteral()
		case TokenNumber, TokenString, TokenBool, TokenIdent:
			elem, err = p.parseLiteral()
		default:
			return nil, p.unexpected("array element (literal or array) or ']'")
		}
		if err != nil {
			return nil, err
		}
		array.Elements = append(array.Elements, elem)

		switch p.curToken.Type {
		case TokenComma:
			p.nextToken()
		case TokenRBracket:
		default:
			return nil, p.unexpected("',' or ']' in array")
		}
	}
}

// parseType 解析类型：标量 Name、Name! 或列表 [Inner]、[Inner]! 等；写法错误返回 *SyntaxError
func (p *Parser) parseType() (*TypeNode, error) {
	var t *TypeNode
	switch p.curToken.Type {
	case TokenLBracket:
		p.nextToken()
		inner, err := p.parseType()
		if err != nil {
			return nil, err
		}
		if p.curToken.Type != TokenRBracket {
			return nil, p.unexpected("']' to close list type")
		}
		p.nextToken()
		t = &TypeNode{IsList: true, InnerType: inner}
	case TokenIdent:
		t = &TypeNode{Name: p.curToken.Value}
		p.nextToken()
	default:
		return nil, p.unexpected("type name or '['")
	}
	if p.curToken.Type == TokenBang {
		t.IsNonNull = true
		p.nextToken()
		if p.curToken.Type == TokenBang {
			return nil, errorf(p.curToken.Pos, "duplicate '!' in type")
		}
	}
	return t, nil
}

// parseLiteral 解析数字、字符串、布尔、null 或标识符（枚举）形式的字面量。
// 整数超出 int 范围、浮点数超出 float64 范围时返回错误
func (p *Parser) parseLiteral() (*LiteralNode, error) {
	tok := p.curToken
	var lit *LiteralNode
	switch tok.Type {
	case TokenNumber:
		if strings.ContainsAny(tok.Value, ".eE") {
			if _, err := strconv.ParseFloat(tok.Value, 64); err != nil {
				return nil, errorf(tok.Pos, "number %s out of range for float64", tok.Value)
			}
		} else if _, err := strconv.ParseInt(tok.Value, 10, strconv.IntSize); err != nil {
			return nil, errorf(tok.Pos, "integer %s out of range for int (quote it as a string if it is an ID)", tok.Value)
		}
		lit = &LiteralNode{Value: tok.Value, Type: "number", Kind: LiteralNumber}
	case TokenString:
		lit = &LiteralNode{Value: Unescape(tok.Value), Type: "string", IsString: true, Kind: LiteralString}
	case TokenBool:
		lit = &LiteralNode{Value: tok.Value, Type: "bool", Kind: LiteralBool}
	case TokenIdent:
		if tok.Value == "null" {
			lit = &LiteralNode{Value: "null", Type: "null", Kind: LiteralNull}
		} else {
			lit = &LiteralNode{Value: tok.Value, Type: "string", IsString: true, Kind: LiteralEnum}
		}
	default:
		return nil, p.unexpected("literal value")
	}
	lit.Pos = tok.Pos
	p.nextToken()
	return lit, nil
}
