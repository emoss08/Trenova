/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package expression

import (
	"fmt"
	"strconv"

	"github.com/rotisserie/eris"
)

// Parser builds an AST from tokens
type Parser struct {
	tokens   []Token
	position int
	current  Token
	errors   []error
}

// NewParser creates a new parser for the given tokens
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

// Parse builds the AST from tokens
func (p *Parser) Parse() (Node, error) {
	if len(p.tokens) == 0 {
		return nil, eris.New("no tokens to parse")
	}

	// Parse the expression
	node := p.parseExpression()

	// Ensure we consumed all tokens (except EOF)
	if p.current.Type != TokenEOF {
		p.addError(fmt.Errorf("unexpected token after expression: %s", p.current.Type))
	}

	// Return any accumulated errors
	if len(p.errors) > 0 {
		return nil, p.errors[0] // Return first error for simplicity
	}

	return node, nil
}

// * parseExpression parses a full expression (handles ternary operator)
func (p *Parser) parseExpression() Node {
	// Start with OR precedence
	node := p.parseOr()

	// Handle ternary conditional
	if p.current.Type == TokenQuestion {
		p.advance() // consume '?'

		trueExpr := p.parseExpression()

		if p.current.Type != TokenColon {
			p.addError(fmt.Errorf("expected ':' in ternary expression, got %s", p.current.Type))
			return node
		}
		p.advance() // consume ':'

		falseExpr := p.parseExpression()

		node = &ConditionalNode{
			Condition: node,
			TrueExpr:  trueExpr,
			FalseExpr: falseExpr,
		}
	}

	return node
}

// * parseOr handles logical OR operations
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

// * parseAnd handles logical AND operations
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

// * parseEquality handles equality operations
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

// * parseComparison handles comparison operations
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

// * parseAddition handles addition and subtraction
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

// * parseMultiplication handles multiplication, division, and modulo
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

// * parsePower handles exponentiation (right-associative)
func (p *Parser) parsePower() Node {
	node := p.parseUnary()

	if p.current.Type == TokenPower {
		op := p.current
		p.advance()

		// Right-associative: parse the right side recursively
		right := p.parsePower()
		node = &BinaryOpNode{
			Left:     node,
			Right:    right,
			Operator: op.Type,
		}
	}

	return node
}

// * parseUnary handles unary operators
func (p *Parser) parseUnary() Node {
	if p.current.Type == TokenNot || p.current.Type == TokenMinus {
		op := p.current
		p.advance()

		operand := p.parseUnary() // Allow chaining of unary operators
		return &UnaryOpNode{
			Operator: op.Type,
			Operand:  operand,
		}
	}

	return p.parsePostfix()
}

// * parsePostfix handles postfix operations like array indexing
func (p *Parser) parsePostfix() Node {
	node := p.parsePrimary()

	for {
		if p.current.Type == TokenLeftBracket {
			p.advance() // consume '['

			index := p.parseExpression()

			if p.current.Type != TokenRightBracket {
				p.addError(fmt.Errorf("expected ']' after array index, got %s", p.current.Type))
			} else {
				p.advance() // consume ']'
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

// * parsePrimary handles primary expressions
func (p *Parser) parsePrimary() Node {
	switch p.current.Type {
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
		p.advance() // consume '('

		node := p.parseExpression()

		if p.current.Type != TokenRightParen {
			p.addError(fmt.Errorf("expected ')', got %s", p.current.Type))
		} else {
			p.advance() // consume ')'
		}

		return node

	case TokenLeftBracket:
		return p.parseArrayLiteral()

	default:
		p.addError(fmt.Errorf("unexpected token: %s", p.current.Type))
		// Create error node to continue parsing
		return &NumberNode{Value: 0}
	}
}

// * parseNumber parses a numeric literal
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

// * parseIdentifierOrFunction parses an identifier or function call
func (p *Parser) parseIdentifierOrFunction() Node {
	name := p.current.Value
	p.advance()

	// Check if it's a function call
	if p.current.Type == TokenLeftParen {
		p.advance() // consume '('

		// Parse arguments
		args := []Node{}

		// Handle empty argument list
		if p.current.Type != TokenRightParen {
			// Parse first argument
			args = append(args, p.parseExpression())

			// Parse remaining arguments
			for p.current.Type == TokenComma {
				p.advance() // consume ','
				args = append(args, p.parseExpression())
			}
		}

		if p.current.Type != TokenRightParen {
			p.addError(fmt.Errorf("expected ')' after function arguments, got %s", p.current.Type))
		} else {
			p.advance() // consume ')'
		}

		return &FunctionCallNode{
			Name:      name,
			Arguments: args,
		}
	}

	// It's a variable reference
	return &IdentifierNode{Name: name}
}

// * Helper methods

func (p *Parser) advance() {
	if p.position < len(p.tokens)-1 {
		p.position++
		p.current = p.tokens[p.position]
	}
}

func (p *Parser) isComparisonOperator(t TokenType) bool {
	switch t {
	case TokenGreater, TokenLess, TokenGreaterEqual, TokenLessEqual:
		return true
	default:
		return false
	}
}

func (p *Parser) addError(err error) {
	// Add position information to error
	err = fmt.Errorf("%w at position %d (line %d, column %d)",
		err, p.current.Position, p.current.Line, p.current.Column)
	p.errors = append(p.errors, err)
}

// * parseArrayLiteral parses array literal syntax: [expr1, expr2, ...]
func (p *Parser) parseArrayLiteral() Node {
	p.advance() // consume '['

	elements := []Node{}

	// Handle empty array
	if p.current.Type == TokenRightBracket {
		p.advance() // consume ']'
		return &ArrayNode{Elements: elements}
	}

	for {
		elem := p.parseExpression()
		elements = append(elements, elem)

		if p.current.Type == TokenComma {
			p.advance() // consume ','

			// Check for trailing comma
			if p.current.Type == TokenRightBracket {
				break
			}
		} else if p.current.Type == TokenRightBracket {
			break
		} else {
			p.addError(fmt.Errorf("expected ',' or ']' in array literal, got %s", p.current.Type))
			break
		}
	}

	if p.current.Type == TokenRightBracket {
		p.advance() // consume ']'
	} else {
		p.addError(fmt.Errorf("expected ']' to close array literal, got %s", p.current.Type))
	}

	return &ArrayNode{Elements: elements}
}
