package core

import "fmt"

// ============ 解析器 ============

// Parser 递归下降解析器，消费 Lexer 产出的 Token 流并构建 AST；用于将 tag 字符串解析为 Node 列表
type Parser struct {
	lexer      *Lexer
	curToken   Token
	peekToken  Token
	parenStack []Token
}

// NewParser 根据输入字符串创建 Parser 并预读两个 Token
func NewParser(input string) *Parser {
	l := NewLexer(input)
	p := &Parser{
		lexer:      l,
		parenStack: make([]Token, 0),
	}
	p.nextToken()
	p.nextToken()
	return p
}

// nextToken 将 peekToken 变为 curToken 并读取下一个 Token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// pushParen 将左括号 Token 压入括号栈，用于检测未闭合括号
func (p *Parser) pushParen(t Token) {
	p.parenStack = append(p.parenStack, t)
}

// popParen 从括号栈弹出栈顶左括号，栈空时返回 false
func (p *Parser) popParen() (Token, bool) {
	if len(p.parenStack) == 0 {
		return Token{}, false
	}
	t := p.parenStack[len(p.parenStack)-1]
	p.parenStack = p.parenStack[:len(p.parenStack)-1]
	return t, true
}

// Parse 解析整个输入，返回 AST 节点列表；遇错或未闭合括号时返回错误。
// 允许前导/连续逗号，视为空位。标记位存在赋值（name=value）时允许省略前导逗号；仅布尔标记（裸 ident）
// 时需本元素前已消费过逗号（afterComma）才解析为标记位，否则解析为无参字段（如 ,inline）。
func (p *Parser) Parse() ([]Node, error) {
	var nodes []Node
	// 前导逗号后为 true：仅在此情况下将「无 ( 无 = 的 ident」解析为布尔标记位，否则解析为无参字段
	afterComma := false

	for p.curToken.Type != TokenEOF {
		if p.curToken.Type == TokenComma {
			p.nextToken()
			afterComma = true
			continue
		}
		node, err := p.parseElement(afterComma)
		if err != nil {
			return nil, err
		}
		afterComma = false
		nodes = append(nodes, node)

		if p.curToken.Type == TokenComma {
			p.nextToken()
			afterComma = true // 下一项在逗号后，按标记位解析
		}
	}

	if len(p.parenStack) > 0 {
		return nil, fmt.Errorf("unmatched '(' at position %d", p.parenStack[0].Pos)
	}

	return nodes, nil
}

// parseElement 解析一个顶层元素。name=value 始终为标记位（可省略前导逗号）；无 ( 无 = 时，仅 afterComma 为 true 时为布尔标记位，否则为无参字段 name()。
func (p *Parser) parseElement(afterComma bool) (Node, error) {
	var name string
	if p.curToken.Type == TokenLParen {
		name = ""
	} else if p.curToken.Type == TokenIdent {
		name = p.curToken.Value
		p.nextToken()
	} else {
		return nil, fmt.Errorf("expected identifier or '(', got %v at pos %d", p.curToken, p.curToken.Pos)
	}

	if p.curToken.Type == TokenLParen {
		return p.parseFieldCall(name)
	}
	if p.curToken.Type == TokenEqual {
		return p.parseFlagWithValue(name) // 存在赋值时允许省略前导逗号
	}
	// 无 ( 无 =：仅前导逗号后解析为布尔标记位，否则为无参字段
	if afterComma {
		return &FlagNode{Name: name, Value: nil}, nil
	}
	return &FieldNode{Name: name, Args: map[string]Node{}}, nil
}

// parseFieldCall 解析 name(arg1:val1, ...) 形式的字段调用，已消费 name 与 (
func (p *Parser) parseFieldCall(name string) (*FieldNode, error) {
	p.pushParen(p.curToken)
	p.nextToken()

	field := &FieldNode{
		Name: name,
		Args: make(map[string]Node),
	}

	if p.curToken.Type == TokenRParen {
		p.popParen()
		p.nextToken()
		return field, nil
	}

	for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
		if p.curToken.Type != TokenIdent {
			return nil, fmt.Errorf("expected parameter name, got %v at pos %d", p.curToken, p.curToken.Pos)
		}

		key := p.curToken.Value
		p.nextToken()

		if p.curToken.Type != TokenColon {
			return nil, fmt.Errorf("expected ':', got %v at pos %d", p.curToken, p.curToken.Pos)
		}
		p.nextToken()

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		field.Args[key] = value

		if p.curToken.Type == TokenComma {
			p.nextToken()
		}
	}

	if p.curToken.Type != TokenRParen {
		return nil, fmt.Errorf("expected ')', got %v at pos %d", p.curToken, p.curToken.Pos)
	}

	p.popParen()
	p.nextToken()

	return field, nil
}

// parseFlagWithValue 解析 name=value 形式的标记，已消费 name 与 =
func (p *Parser) parseFlagWithValue(name string) (*FlagNode, error) {
	p.nextToken()

	value, err := p.parseLiteralOrArray()
	if err != nil {
		return nil, err
	}

	return &FlagNode{Name: name, Value: value}, nil
}

// parseValue 解析参数值：变量 $x:Type=default、字面量或数组字面量
func (p *Parser) parseValue() (Node, error) {
	switch p.curToken.Type {
	case TokenDollar:
		return p.parseVariable()
	case TokenLBracket:
		return p.parseArrayLiteral()
	case TokenNumber, TokenString, TokenBool, TokenIdent:
		return p.parseLiteral()
	default:
		return nil, fmt.Errorf("unexpected token %v at pos %d", p.curToken, p.curToken.Pos)
	}
}

// parseVariable 解析变量：$name、$name:Type=default 等；类型与默认值均可选，不写 :Type 时 VarType 为 nil；已消费 $
func (p *Parser) parseVariable() (*VariableNode, error) {
	if p.curToken.Type == TokenDollar {
		p.nextToken()
	}

	var varName string
	if p.curToken.Type == TokenIdent {
		varName = p.curToken.Value
		p.nextToken()
	}
	// 匿名变量：$ 后无标识符则变量名为空。类型可选：无 : 则不设置类型（VarType 为 nil）
	var varType *TypeNode
	if p.curToken.Type == TokenColon {
		p.nextToken()
		varType = p.parseType()
	}

	var defaultVal Node
	if p.curToken.Type == TokenEqual {
		p.nextToken()
		var err error
		defaultVal, err = p.parseLiteralOrArray()
		if err != nil {
			return nil, fmt.Errorf("invalid default value: %v", err)
		}
	}

	raw := "$" + varName
	if varType != nil {
		raw += ":" + varType.String()
	}
	if defaultVal != nil {
		raw += "=" + defaultVal.String()
	}

	return &VariableNode{
		Name:         varName,
		VarType:      varType,
		DefaultValue: defaultVal,
		Raw:          raw,
	}, nil
}

// parseLiteralOrArray 根据当前 Token 解析字面量或数组字面量
func (p *Parser) parseLiteralOrArray() (Node, error) {
	if p.curToken.Type == TokenLBracket {
		return p.parseArrayLiteral()
	}
	return p.parseLiteral()
}

// parseArrayLiteral 解析 [ ... ] 数组字面量，支持嵌套；已消费 [
func (p *Parser) parseArrayLiteral() (*ArrayNode, error) {
	if p.curToken.Type != TokenLBracket {
		return nil, fmt.Errorf("expected '[', got %v at pos %d", p.curToken, p.curToken.Pos)
	}
	p.nextToken()

	array := &ArrayNode{
		Elements: make([]Node, 0),
	}

	if p.curToken.Type == TokenRBracket {
		p.nextToken()
		return array, nil
	}

	for p.curToken.Type != TokenRBracket && p.curToken.Type != TokenEOF {
		var elem Node
		var err error

		// 递归支持嵌套数组；无单引号时以逗号分割，标识符视为字符串元素；有单引号时以单引号为准
		if p.curToken.Type == TokenLBracket {
			elem, err = p.parseArrayLiteral()
		} else if p.curToken.Type == TokenNumber || p.curToken.Type == TokenString || p.curToken.Type == TokenIdent {
			elem, err = p.parseLiteral()
		} else {
			return nil, fmt.Errorf("array element must be literal or array, got %v at pos %d", p.curToken, p.curToken.Pos)
		}

		if err != nil {
			return nil, err
		}

		array.Elements = append(array.Elements, elem)

		if p.curToken.Type == TokenComma {
			p.nextToken()
		}
	}

	if p.curToken.Type != TokenRBracket {
		return nil, fmt.Errorf("expected ']', got %v at pos %d", p.curToken, p.curToken.Pos)
	}
	p.nextToken()

	return array, nil
}

// parseType 解析类型：标量 Name、Name! 或列表 [Inner]、[Inner]! 等
func (p *Parser) parseType() *TypeNode {
	if p.curToken.Type == TokenLBracket {
		p.nextToken()
		inner := p.parseType()
		isNonNull := false

		if p.curToken.Type == TokenBang {
			inner.IsNonNull = true
			p.nextToken()
		}

		if p.curToken.Type != TokenRBracket {
			return &TypeNode{Name: "Unknown"}
		}
		p.nextToken()

		if p.curToken.Type == TokenBang {
			isNonNull = true
			p.nextToken()
		}

		return &TypeNode{
			IsList:    true,
			InnerType: inner,
			IsNonNull: isNonNull,
		}
	}

	if p.curToken.Type != TokenIdent {
		return &TypeNode{Name: "Unknown"}
	}

	typeName := p.curToken.Value
	p.nextToken()

	isNonNull := false
	if p.curToken.Type == TokenBang {
		isNonNull = true
		p.nextToken()
	}

	return &TypeNode{
		Name:      typeName,
		IsNonNull: isNonNull,
	}
}

// parseLiteral 解析数字、字符串、布尔或标识符形式的字面量
func (p *Parser) parseLiteral() (Node, error) {
	switch p.curToken.Type {
	case TokenNumber:
		val := p.curToken.Value
		p.nextToken()
		return &LiteralNode{Value: val, Type: "number"}, nil
	case TokenString:
		val := p.curToken.Value
		p.nextToken()
		return &LiteralNode{Value: val, Type: "string", IsString: true}, nil
	case TokenBool:
		val := p.curToken.Value
		p.nextToken()
		return &LiteralNode{Value: val, Type: "bool"}, nil
	case TokenIdent:
		val := p.curToken.Value
		p.nextToken()
		return &LiteralNode{Value: val, Type: "string", IsString: true}, nil
	default:
		return nil, fmt.Errorf("expected literal, got %v at pos %d", p.curToken, p.curToken.Pos)
	}
}
