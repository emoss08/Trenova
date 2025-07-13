package expression

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rotisserie/eris"
)

// Tokenizer performs lexical analysis on expression strings
type Tokenizer struct {
	input    string
	position int  // current position in input (points to current char)
	readPos  int  // current reading position in input (after current char)
	ch       rune // current char under examination
	line     int
	column   int

	// String interning for efficiency
	interner *StringInterner

	// Debug mode
	debug bool
}

// NewTokenizer creates a new tokenizer for the input string
func NewTokenizer(input string) *Tokenizer {
	t := &Tokenizer{
		input:    input,
		line:     1,
		column:   0,
		interner: NewStringInterner(),
		debug:    false,
	}
	t.readChar()
	return t
}

// EnableDebug turns on debug logging
func (t *Tokenizer) EnableDebug() {
	t.debug = true
}

// debugf prints debug messages if debug mode is enabled
func (t *Tokenizer) debugf(format string, args ...any) {
	if t.debug {
		fmt.Printf("[Tokenizer] "+format+"\n", args...)
	}
}

// * Tokenize converts the input string into a slice of tokens
func (t *Tokenizer) Tokenize() ([]Token, error) {
	// Pre-allocate with reasonable capacity
	tokens := make([]Token, 0, len(t.input)/3)

	// Validate input length
	if len(t.input) > MaxExpressionLength {
		return nil, fmt.Errorf("expression too long: %d characters (max %d)",
			len(t.input), MaxExpressionLength)
	}

	for {
		// Check token count limit
		if len(tokens) > MaxTokenCount {
			return nil, fmt.Errorf("expression too complex: %d tokens (max %d)",
				len(tokens), MaxTokenCount)
		}

		tok := t.nextToken()
		tokens = append(tokens, tok)

		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}

	return tokens, nil
}

// * nextToken returns the next token from the input
func (t *Tokenizer) nextToken() Token {
	t.skipWhitespace()

	// Track position for error reporting
	pos := t.position
	line := t.line
	col := t.column

	t.debugf("nextToken() at pos=%d, ch='%c' (%d)", pos, t.ch, t.ch)

	var tok Token
	tok.Position = pos
	tok.Line = uint16(line)
	tok.Column = uint16(col)

	switch t.ch {
	case 0:
		tok.Type = TokenEOF

	// Operators and delimiters
	case '+':
		tok.Type = TokenPlus
		t.readChar()
	case '-':
		tok.Type = TokenMinus
		t.readChar()
	case '*':
		tok.Type = TokenMultiply
		t.readChar()
	case '/':
		tok.Type = TokenDivide
		t.readChar()
	case '%':
		tok.Type = TokenModulo
		t.readChar()
	case '^':
		tok.Type = TokenPower
		t.readChar()
	case '(':
		tok.Type = TokenLeftParen
		t.readChar()
	case ')':
		tok.Type = TokenRightParen
		t.readChar()
	case ',':
		tok.Type = TokenComma
		t.readChar()
	case '[':
		tok.Type = TokenLeftBracket
		t.readChar()
	case ']':
		tok.Type = TokenRightBracket
		t.readChar()
	case '?':
		tok.Type = TokenQuestion
		t.readChar()
	case ':':
		tok.Type = TokenColon
		t.readChar()

	// Comparison operators
	case '=':
		t.readChar()
		if t.ch == '=' {
			tok.Type = TokenEqual
			t.readChar()
		} else {
			tok.Type = TokenError
			tok.Value = "unexpected '=', use '==' for equality"
		}
	case '!':
		t.readChar()
		if t.ch == '=' {
			tok.Type = TokenNotEqual
			t.readChar()
		} else {
			tok.Type = TokenNot
		}
	case '<':
		t.readChar()
		if t.ch == '=' {
			tok.Type = TokenLessEqual
			t.readChar()
		} else {
			tok.Type = TokenLess
		}
	case '>':
		t.readChar()
		if t.ch == '=' {
			tok.Type = TokenGreaterEqual
			t.readChar()
		} else {
			tok.Type = TokenGreater
		}

	// Logical operators
	case '&':
		t.readChar()
		if t.ch == '&' {
			tok.Type = TokenAnd
			t.readChar()
		} else {
			tok.Type = TokenError
			tok.Value = "unexpected '&', use '&&' for logical AND"
		}
	case '|':
		t.readChar()
		if t.ch == '|' {
			tok.Type = TokenOr
			t.readChar()
		} else {
			tok.Type = TokenError
			tok.Value = "unexpected '|', use '||' for logical OR"
		}

	// String literals
	case '"':
		str, err := t.readString()
		if err != nil {
			tok.Type = TokenError
			tok.Value = err.Error()
		} else {
			tok.Type = TokenString
			tok.Value = t.interner.Intern(str)
		}

	default:
		if isDigit(t.ch) || (t.ch == '.' && isDigit(t.peekChar())) {
			// Number literal
			num, err := t.readNumber()
			if err != nil {
				tok.Type = TokenError
				tok.Value = err.Error()
			} else {
				tok.Type = TokenNumber
				tok.Value = num
			}
		} else if isLetter(t.ch) || t.ch == '_' {
			// Identifier or keyword
			ident := t.readIdentifier()

			// Check for keywords
			switch ident {
			case "true":
				tok.Type = TokenTrue
				tok.Value = "true"
			case "false":
				tok.Type = TokenFalse
				tok.Value = "false"
			default:
				tok.Type = TokenIdentifier
				tok.Value = t.interner.Intern(ident)
			}
		} else {
			tok.Type = TokenError
			tok.Value = fmt.Sprintf("unexpected character: %c", t.ch)
			t.readChar()
		}
	}

	t.debugf("nextToken() returning token: Type=%s, Value=%q", tok.Type, tok.Value)
	return tok
}

// readChar advances to the next character
func (t *Tokenizer) readChar() {
	if t.readPos >= len(t.input) {
		t.ch = 0
		// Important: update position to readPos when we hit EOF
		t.position = t.readPos
	} else {
		r, w := utf8.DecodeRuneInString(t.input[t.readPos:])
		t.ch = r
		t.position = t.readPos
		t.readPos += w

		// Track line and column
		if r == '\n' {
			t.line++
			t.column = 0
		} else {
			t.column++
		}
	}
}

// * peekChar returns the next character without advancing
func (t *Tokenizer) peekChar() rune {
	if t.readPos >= len(t.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(t.input[t.readPos:])
	return r
}

// * skipWhitespace skips whitespace characters
func (t *Tokenizer) skipWhitespace() {
	for unicode.IsSpace(t.ch) {
		t.readChar()
	}
}

// * readNumber reads a numeric literal
func (t *Tokenizer) readNumber() (string, error) {
	startPos := t.position
	t.debugf("readNumber() starting at pos=%d, ch='%c'", startPos, t.ch)

	// Read integer part
	for isDigit(t.ch) {
		t.debugf("  reading digit '%c' at pos=%d", t.ch, t.position)
		t.readChar()
	}

	// Read decimal part
	if t.ch == '.' && isDigit(t.peekChar()) {
		t.debugf("  found decimal point at pos=%d, next='%c'", t.position, t.peekChar())
		t.readChar() // consume '.'
		for isDigit(t.ch) {
			t.debugf("  reading decimal digit '%c' at pos=%d", t.ch, t.position)
			t.readChar()
		}
	}

	// Read exponent part
	if t.ch == 'e' || t.ch == 'E' {
		t.debugf("  found exponent '%c' at pos=%d", t.ch, t.position)
		t.readChar() // consume 'e' or 'E'

		// Optional sign
		if t.ch == '+' || t.ch == '-' {
			t.debugf("  found exponent sign '%c' at pos=%d", t.ch, t.position)
			t.readChar()
		}

		// Exponent digits
		if !isDigit(t.ch) {
			return "", eris.New("invalid number: expected digits after exponent")
		}
		for isDigit(t.ch) {
			t.debugf("  reading exponent digit '%c' at pos=%d", t.ch, t.position)
			t.readChar()
		}
	}

	// Validate that we don't have trailing invalid characters
	if isLetter(t.ch) || t.ch == '_' {
		return "", fmt.Errorf("invalid number: unexpected character '%c'", t.ch)
	}

	result := t.input[startPos:t.position]
	t.debugf("readNumber() returning %q (startPos=%d, position=%d)", result, startPos, t.position)
	return result, nil
}

// * readString reads a string literal
func (t *Tokenizer) readString() (string, error) {
	t.readChar() // consume opening quote

	var result strings.Builder
	startLine := t.line

	for t.ch != '"' && t.ch != 0 {
		if t.ch == '\\' {
			t.readChar()

			// Handle escape sequences
			switch t.ch {
			case '"', '\\', '/':
				result.WriteRune(t.ch)
			case 'n':
				result.WriteRune('\n')
			case 'r':
				result.WriteRune('\r')
			case 't':
				result.WriteRune('\t')
			case 'b':
				result.WriteRune('\b')
			case 'f':
				result.WriteRune('\f')
			default:
				return "", fmt.Errorf("invalid escape sequence: \\%c", t.ch)
			}
			t.readChar()
		} else {
			result.WriteRune(t.ch)
			t.readChar()
		}
	}

	if t.ch == 0 {
		return "", fmt.Errorf("unterminated string literal starting at line %d", startLine)
	}

	t.readChar() // consume closing quote
	return result.String(), nil
}

// * readIdentifier reads an identifier
func (t *Tokenizer) readIdentifier() string {
	startPos := t.position

	// First character must be letter or underscore
	if !isLetter(t.ch) && t.ch != '_' {
		return ""
	}

	for isLetter(t.ch) || isDigit(t.ch) || t.ch == '_' {
		t.readChar()
	}

	return t.input[startPos:t.position]
}

// * Helper functions

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
