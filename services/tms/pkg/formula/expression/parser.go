package expression

import (
	"fmt"
	"strconv"
)

type Parser struct {
	tokens   []Token
	position int
	current  Token
	errors   []error
}

func NewParser(tokens []Token) *Parser {
	p := &Parser{
		tokens: tokens,
		errors: make([]error, 0),
	}

	if len(tokens) > 0 {
		p.current = tokens[0]
	}
	return p
}

func (p *Parser) Parse() (Node, error) {
	if len(p.tokens) == 0 {
		return nil, ErrNoTokensToParse
	}

	node := p.parseExpression()

	if p.current.Type != TokenEOF {
		p.addError(fmt.Errorf("unexpected token after expression: %s", p.current.Type))
	}

	if len(p.errors) > 0 {
		return nil, p.errors[0]
	}

	return node, nil
}

func (p *Parser) parseExpression() Node {
	node := p.parseOr()

	if p.current.Type == TokenQuestion {
		p.advance()

		trueExpr := p.parseExpression()

		if p.current.Type != TokenColon {
			p.addError(fmt.Errorf("expected ':' in ternary expression, got %s", p.current.Type))
			return node
		}
		p.advance()

		falseExpr := p.parseExpression()

		node = &ConditionalNode{
			Condition: node,
			TrueExpr:  trueExpr,
			FalseExpr: falseExpr,
		}
	}

	return node
}

func (p *Parser) parseOr() Node {
	node := p.parseAnd()

	for p.current.Type == TokenOr {
		op := p.current
		p.advance()

		right := p.parseAnd()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

func (p *Parser) parseAnd() Node {
	node := p.parseEquality()

	for p.current.Type == TokenAnd {
		op := p.current
		p.advance()

		right := p.parseEquality()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

func (p *Parser) parseEquality() Node {
	node := p.parseComparison()

	for p.current.Type == TokenEqual || p.current.Type == TokenNotEqual {
		op := p.current
		p.advance()

		right := p.parseComparison()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

func (p *Parser) parseComparison() Node {
	node := p.parseAddition()

	for p.isComparisonOperator(p.current.Type) {
		op := p.current
		p.advance()

		right := p.parseAddition()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

func (p *Parser) parseAddition() Node {
	node := p.parseMultiplication()

	for p.current.Type == TokenPlus || p.current.Type == TokenMinus {
		op := p.current
		p.advance()

		right := p.parseMultiplication()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

func (p *Parser) parseMultiplication() Node {
	node := p.parsePower()

	for p.current.Type == TokenMultiply || p.current.Type == TokenDivide || p.current.Type == TokenModulo {
		op := p.current
		p.advance()

		right := p.parsePower()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

func (p *Parser) parsePower() Node {
	node := p.parseUnary()

	if p.current.Type == TokenPower {
		op := p.current
		p.advance()

		right := p.parsePower()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

func (p *Parser) parseUnary() Node {
	if p.current.Type == TokenNot || p.current.Type == TokenMinus {
		op := p.current
		p.advance()

		operand := p.parseUnary()
		return &UnaryOpNode{
			Operator: op.Type,
			Operand:  operand,
		}
	}

	return p.parsePostfix()
}

func (p *Parser) parsePostfix() Node {
	node := p.parsePrimary()

	for {
		if p.current.Type == TokenLeftBracket {
			p.advance()

			index := p.parseExpression()

			if p.current.Type != TokenRightBracket {
				p.addError(fmt.Errorf("expected ']' after array index, got %s", p.current.Type))
			} else {
				p.advance()
			}

			node = &IndexNode{
				Array: node,
				Index: index,
			}
		} else {
			break
		}
	}

	return node
}

func (p *Parser) parsePrimary() Node {
	switch p.current.Type { //nolint:exhaustive // all tokens are covered
	case TokenNumber:
		return p.parseNumber()

	case TokenString:
		node := &StringNode{Value: p.current.Value}
		p.advance()
		return node

	case TokenTrue:
		node := &BooleanNode{Value: true}
		p.advance()
		return node

	case TokenFalse:
		node := &BooleanNode{Value: false}
		p.advance()
		return node

	case TokenIdentifier:
		return p.parseIdentifierOrFunction()

	case TokenLeftParen:
		p.advance()

		node := p.parseExpression()

		if p.current.Type != TokenRightParen {
			p.addError(fmt.Errorf("expected ')', got %s", p.current.Type))
		} else {
			p.advance()
		}

		return node

	case TokenLeftBracket:
		return p.parseArrayLiteral()

	default:
		p.addError(fmt.Errorf("unexpected token: %s", p.current.Type))
		return &NumberNode{Value: 0}
	}
}

func (p *Parser) parseNumber() Node {
	val, err := strconv.ParseFloat(p.current.Value, 64)
	if err != nil {
		p.addError(fmt.Errorf("invalid number: %s", p.current.Value))
		val = 0
	}

	node := &NumberNode{Value: val}
	p.advance()
	return node
}

func (p *Parser) parseIdentifierOrFunction() Node {
	name := p.current.Value
	p.advance()

	if p.current.Type == TokenLeftParen {
		p.advance()

		args := []Node{}

		if p.current.Type != TokenRightParen {
			args = append(args, p.parseExpression())

			for p.current.Type == TokenComma {
				p.advance()
				args = append(args, p.parseExpression())
			}
		}

		if p.current.Type != TokenRightParen {
			p.addError(fmt.Errorf("expected ')' after function arguments, got %s", p.current.Type))
		} else {
			p.advance()
		}

		return &FunctionCallNode{
			Name:      name,
			Arguments: args,
		}
	}

	return &IdentifierNode{Name: name}
}

func (p *Parser) advance() {
	if p.position < len(p.tokens)-1 {
		p.position++
		p.current = p.tokens[p.position]
	}
}

func (p *Parser) isComparisonOperator(t TokenType) bool {
	switch t { //nolint:exhaustive // all operators are covered
	case TokenGreater, TokenLess, TokenGreaterEqual, TokenLessEqual:
		return true
	default:
		return false
	}
}

func (p *Parser) addError(err error) {
	err = fmt.Errorf("%w at position %d (line %d, column %d)",
		err, p.current.Position, p.current.Line, p.current.Column)
	p.errors = append(p.errors, err)
}

func (p *Parser) parseArrayLiteral() Node {
	p.advance()

	elements := []Node{}

	if p.current.Type == TokenRightBracket {
		p.advance()
		return &ArrayNode{Elements: elements}
	}

loop:
	for {
		elem := p.parseExpression()
		elements = append(elements, elem)

		switch p.current.Type { //nolint:exhaustive // all tokens are covered
		case TokenComma:
			p.advance()
		case TokenRightBracket:
			break loop
		default:
			p.addError(fmt.Errorf("expected ',' or ']' in array literal, got %s", p.current.Type))
			break loop
		}
	}

	p.advance()

	return &ArrayNode{Elements: elements}
}
