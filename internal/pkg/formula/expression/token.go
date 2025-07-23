// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package expression

import (
	"fmt"
	"sync"
)

// TokenType represents the type of a token
type TokenType uint8 // Use uint8 to save memory

const (
	TokenEOF TokenType = iota
	TokenError

	// * Literals
	TokenNumber     // 123, 45.67, 1.23e10
	TokenIdentifier // variable names
	TokenString     // "quoted strings"
	TokenTrue       // true
	TokenFalse      // false

	// * Arithmetic Operators
	TokenPlus     // +
	TokenMinus    // -
	TokenMultiply // *
	TokenDivide   // /
	TokenModulo   // %
	TokenPower    // ^

	// * Comparison Operators
	TokenEqual        // ==
	TokenNotEqual     // !=
	TokenGreater      // >
	TokenLess         // <
	TokenGreaterEqual // >=
	TokenLessEqual    // <=

	// * Logical Operators
	TokenAnd // &&
	TokenOr  // ||
	TokenNot // !

	// * Delimiters
	TokenLeftParen    // (
	TokenRightParen   // )
	TokenComma        // ,
	TokenLeftBracket  // [
	TokenRightBracket // ]

	// * Conditional
	TokenQuestion // ?
	TokenColon    // :
)

// * tokenTypeStrings for efficient string conversion
var tokenTypeStrings = [...]string{
	TokenEOF:          "EOF",
	TokenError:        "ERROR",
	TokenNumber:       "NUMBER",
	TokenIdentifier:   "IDENTIFIER",
	TokenString:       "STRING",
	TokenTrue:         "TRUE",
	TokenFalse:        "FALSE",
	TokenPlus:         "+",
	TokenMinus:        "-",
	TokenMultiply:     "*",
	TokenDivide:       "/",
	TokenModulo:       "%",
	TokenPower:        "^",
	TokenEqual:        "==",
	TokenNotEqual:     "!=",
	TokenGreater:      ">",
	TokenLess:         "<",
	TokenGreaterEqual: ">=",
	TokenLessEqual:    "<=",
	TokenAnd:          "&&",
	TokenOr:           "||",
	TokenNot:          "!",
	TokenLeftParen:    "(",
	TokenRightParen:   ")",
	TokenComma:        ",",
	TokenLeftBracket:  "[",
	TokenRightBracket: "]",
	TokenQuestion:     "?",
	TokenColon:        ":",
}

// String returns the string representation of a token type
func (t TokenType) String() string {
	if int(t) < len(tokenTypeStrings) {
		return tokenTypeStrings[t]
	}
	return fmt.Sprintf("UNKNOWN(%d)", t)
}

// Token represents a lexical token
// Fields are ordered for optimal memory alignment
type Token struct {
	Value    string    // 16 bytes (string header)
	Position int       // 8 bytes
	Type     TokenType // 1 byte
	Line     uint16    // 2 bytes
	Column   uint16    // 2 bytes
	// Total: 29 bytes, padded to 32 bytes
}

// String returns a string representation of the token
func (t Token) String() string {
	if t.Value != "" && t.Value != t.Type.String() {
		return fmt.Sprintf("%s:%s@%d:%d", t.Type, t.Value, t.Line, t.Column)
	}
	return fmt.Sprintf("%s@%d:%d", t.Type, t.Line, t.Column)
}

// Operator precedence levels
const (
	PrecedenceLowest      = iota
	PrecedenceConditional // ?:
	PrecedenceOr          // ||
	PrecedenceAnd         // &&
	PrecedenceEquality    // ==, !=
	PrecedenceComparison  // >, <, >=, <=
	PrecedenceAddition    // +, -
	PrecedenceMultiply    // *, /, %
	PrecedencePower       // ^
	PrecedenceUnary       // !, -
	PrecedenceHighest
)

// Precedence returns the operator precedence
func (t Token) Precedence() int {
	switch t.Type { //nolint:exhaustive // all operators are covered
	case TokenQuestion:
		return PrecedenceConditional
	case TokenOr:
		return PrecedenceOr
	case TokenAnd:
		return PrecedenceAnd
	case TokenEqual, TokenNotEqual:
		return PrecedenceEquality
	case TokenGreater, TokenLess, TokenGreaterEqual, TokenLessEqual:
		return PrecedenceComparison
	case TokenPlus, TokenMinus:
		return PrecedenceAddition
	case TokenMultiply, TokenDivide, TokenModulo:
		return PrecedenceMultiply
	case TokenPower:
		return PrecedencePower
	default:
		return PrecedenceLowest
	}
}

// IsOperator returns true if the token is an arithmetic operator
func (t Token) IsOperator() bool {
	switch t.Type { //nolint:exhaustive // all operators are covered
	case TokenPlus, TokenMinus, TokenMultiply, TokenDivide, TokenModulo, TokenPower:
		return true
	default:
		return false
	}
}

// IsBinaryOperator returns true if the token is a binary operator
func (t Token) IsBinaryOperator() bool {
	switch t.Type { //nolint:exhaustive // all operators are covered
	case TokenPlus, TokenMinus, TokenMultiply, TokenDivide, TokenModulo, TokenPower,
		TokenEqual, TokenNotEqual, TokenGreater, TokenLess, TokenGreaterEqual, TokenLessEqual,
		TokenAnd, TokenOr:
		return true
	default:
		return false
	}
}

// IsUnaryOperator returns true if the token is a unary operator
func (t Token) IsUnaryOperator() bool {
	switch t.Type { //nolint:exhaustive // all operators are covered
	case TokenNot, TokenMinus:
		return true
	default:
		return false
	}
}

// IsRightAssociative returns true if the operator is right associative
func (t Token) IsRightAssociative() bool {
	return t.Type == TokenPower || t.Type == TokenQuestion
}

// * TokenPool for reusing token objects
var tokenPool = sync.Pool{
	New: func() any {
		return &Token{}
	},
}

// GetToken returns a token from the pool
func GetToken() *Token {
	return tokenPool.Get().(*Token)
}

// PutToken returns a token to the pool
func PutToken(t *Token) {
	t.Value = ""
	t.Position = 0
	t.Type = TokenEOF
	t.Line = 0
	t.Column = 0
	tokenPool.Put(t)
}

// StringInterner for deduplicating string values
type StringInterner struct {
	strings map[string]string
	mu      sync.RWMutex
}

// NewStringInterner creates a new string interner
func NewStringInterner() *StringInterner {
	return &StringInterner{
		strings: make(map[string]string),
	}
}

// Intern returns an interned version of the string
func (si *StringInterner) Intern(s string) string {
	si.mu.RLock()
	if interned, ok := si.strings[s]; ok {
		si.mu.RUnlock()
		return interned
	}
	si.mu.RUnlock()

	si.mu.Lock()
	defer si.mu.Unlock()

	// Double-check after acquiring write lock
	if interned, ok := si.strings[s]; ok {
		return interned
	}

	si.strings[s] = s
	return s
}

// Clear removes all interned strings
func (si *StringInterner) Clear() {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.strings = make(map[string]string)
}
