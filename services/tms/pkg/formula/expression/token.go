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
	TokenNumber       // 123, 45.67, 1.23e10
	TokenIdentifier   // variable names
	TokenString       // "quoted strings"
	TokenTrue         // true
	TokenFalse        // false
	TokenPlus         // +
	TokenMinus        // -
	TokenMultiply     // *
	TokenDivide       // /
	TokenModulo       // %
	TokenPower        // ^
	TokenEqual        // ==
	TokenNotEqual     // !=
	TokenGreater      // >
	TokenLess         // <
	TokenGreaterEqual // >=
	TokenLessEqual    // <=
	TokenAnd          // &&
	TokenOr           // ||
	TokenNot          // !
	TokenLeftParen    // (
	TokenRightParen   // )
	TokenComma        // ,
	TokenLeftBracket  // [
	TokenRightBracket // ]
	TokenQuestion     // ?
	TokenColon        // :
)

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

func (t TokenType) String() string {
	if int(t) < len(tokenTypeStrings) {
		return tokenTypeStrings[t]
	}
	return fmt.Sprintf("UNKNOWN(%d)", t)
}

type Token struct {
	Value    string    // 16 bytes (string header)
	Position int       // 8 bytes
	Type     TokenType // 1 byte
	Line     uint16    // 2 bytes
	Column   uint16    // 2 bytes
}

func (t Token) String() string {
	if t.Value != "" && t.Value != t.Type.String() {
		return fmt.Sprintf("%s:%s@%d:%d", t.Type, t.Value, t.Line, t.Column)
	}
	return fmt.Sprintf("%s@%d:%d", t.Type, t.Line, t.Column)
}

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

func (t Token) IsOperator() bool {
	switch t.Type { //nolint:exhaustive // all operators are covered
	case TokenPlus, TokenMinus, TokenMultiply, TokenDivide, TokenModulo, TokenPower:
		return true
	default:
		return false
	}
}

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

func (t Token) IsUnaryOperator() bool {
	switch t.Type { //nolint:exhaustive // all operators are covered
	case TokenNot, TokenMinus:
		return true
	default:
		return false
	}
}

func (t Token) IsRightAssociative() bool {
	return t.Type == TokenPower || t.Type == TokenQuestion
}

var tokenPool = sync.Pool{
	New: func() any {
		return &Token{}
	},
}

func GetToken() *Token {
	token, ok := tokenPool.Get().(*Token)
	if !ok {
		return &Token{}
	}

	return token
}

func PutToken(t *Token) {
	t.Value = ""
	t.Position = 0
	t.Type = TokenEOF
	t.Line = 0
	t.Column = 0
	tokenPool.Put(t)
}

type StringInterner struct {
	strings map[string]string
	mu      sync.RWMutex
}

func NewStringInterner() *StringInterner {
	return &StringInterner{
		strings: make(map[string]string),
	}
}

func (si *StringInterner) Intern(s string) string {
	si.mu.RLock()
	if interned, ok := si.strings[s]; ok {
		si.mu.RUnlock()
		return interned
	}
	si.mu.RUnlock()

	si.mu.Lock()
	defer si.mu.Unlock()

	if interned, ok := si.strings[s]; ok {
		return interned
	}

	si.strings[s] = s
	return s
}

func (si *StringInterner) Clear() {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.strings = make(map[string]string)
}
