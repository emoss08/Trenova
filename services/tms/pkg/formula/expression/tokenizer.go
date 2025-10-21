package expression

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/utils"
)

type Tokenizer struct {
	input    string
	position int  // current position in input (points to current char)
	readPos  int  // current reading position in input (after current char)
	ch       rune // current char under examination
	line     int
	column   int
	interner *StringInterner
	debug    bool
}

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

func (t *Tokenizer) EnableDebug() {
	t.debug = true
}

func (t *Tokenizer) debugf(format string, args ...any) {
	if t.debug {
		fmt.Printf("[Tokenizer] "+format+"\n", args...)
	}
}

func (t *Tokenizer) Tokenize() ([]Token, error) {
	tokens := make([]Token, 0, len(t.input)/3)

	if len(t.input) > MaxExpressionLength {
		return nil, fmt.Errorf("expression too long: %d characters (max %d)",
			len(t.input), MaxExpressionLength)
	}

	for {
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

func (t *Tokenizer) nextToken() Token { //nolint:gocognit,gocyclo,cyclop,funlen // this is fine
	t.skipWhitespace()

	pos := t.position
	line := t.line
	col := t.column

	t.debugf("nextToken() at pos=%d, ch='%c' (%d)", pos, t.ch, t.ch)

	var tok Token
	tok.Position = pos

	tok.Line = utils.SafeIntToUint16(line)
	tok.Column = utils.SafeIntToUint16(col)

	switch t.ch {
	case 0:
		tok.Type = TokenEOF
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
		switch {
		case isDigit(t.ch) || (t.ch == '.' && isDigit(t.peekChar())):
			num, err := t.readNumber()
			if err != nil {
				tok.Type = TokenError
				tok.Value = err.Error()
			} else {
				tok.Type = TokenNumber
				tok.Value = num
			}
		case isLetter(t.ch) || t.ch == '_':
			ident := t.readIdentifier()
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
		default:
			tok.Type = TokenError
			tok.Value = fmt.Sprintf("unexpected character: %c", t.ch)
			t.readChar()
		}
	}

	t.debugf("nextToken() returning token: Type=%s, Value=%q", tok.Type, tok.Value)
	return tok
}

func (t *Tokenizer) readChar() {
	if t.readPos >= len(t.input) {
		t.ch = 0
		// ! update position to readPos when we hit EOF
		t.position = t.readPos
	} else {
		r, w := utf8.DecodeRuneInString(t.input[t.readPos:])
		t.ch = r
		t.position = t.readPos
		t.readPos += w

		if r == '\n' {
			t.line++
			t.column = 0
		} else {
			t.column++
		}
	}
}

func (t *Tokenizer) peekChar() rune {
	if t.readPos >= len(t.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(t.input[t.readPos:])
	return r
}

func (t *Tokenizer) skipWhitespace() {
	for unicode.IsSpace(t.ch) {
		t.readChar()
	}
}

func (t *Tokenizer) readNumber() (string, error) {
	startPos := t.position
	t.debugf("readNumber() starting at pos=%d, ch='%c'", startPos, t.ch)

	for isDigit(t.ch) {
		t.debugf("  reading digit '%c' at pos=%d", t.ch, t.position)
		t.readChar()
	}

	if t.ch == '.' && isDigit(t.peekChar()) {
		t.debugf("  found decimal point at pos=%d, next='%c'", t.position, t.peekChar())
		t.readChar() // consume '.'
		for isDigit(t.ch) {
			t.debugf("  reading decimal digit '%c' at pos=%d", t.ch, t.position)
			t.readChar()
		}
	}

	if t.ch == 'e' || t.ch == 'E' {
		t.debugf("  found exponent '%c' at pos=%d", t.ch, t.position)
		t.readChar() // consume 'e' or 'E'

		if t.ch == '+' || t.ch == '-' {
			t.debugf("  found exponent sign '%c' at pos=%d", t.ch, t.position)
			t.readChar()
		}

		if !isDigit(t.ch) {
			return "", ErrInvalidNumberExpectedDigitsAfterExponent
		}
		for isDigit(t.ch) {
			t.debugf("  reading exponent digit '%c' at pos=%d", t.ch, t.position)
			t.readChar()
		}
	}

	if isLetter(t.ch) || t.ch == '_' {
		return "", fmt.Errorf("invalid number: unexpected character '%c'", t.ch)
	}

	result := t.input[startPos:t.position]
	t.debugf("readNumber() returning %q (startPos=%d, position=%d)", result, startPos, t.position)
	return result, nil
}

func (t *Tokenizer) readString() (string, error) {
	t.readChar()

	var result strings.Builder
	startLine := t.line

	for t.ch != '"' && t.ch != 0 {
		if t.ch == '\\' {
			t.readChar()

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

	t.readChar()
	return result.String(), nil
}

func (t *Tokenizer) readIdentifier() string {
	startPos := t.position

	if !isLetter(t.ch) && t.ch != '_' {
		return ""
	}

	for isLetter(t.ch) || isDigit(t.ch) || t.ch == '_' {
		t.readChar()
	}

	return t.input[startPos:t.position]
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
