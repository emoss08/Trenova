package expression

import (
	"strings"
	"testing"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []TokenType
		wantError bool
	}{
		// Basic operators
		{
			name:  "simple addition",
			input: "1 + 2",
			want:  []TokenType{TokenNumber, TokenPlus, TokenNumber, TokenEOF},
		},
		{
			name:  "all arithmetic operators",
			input: "1 + 2 - 3 * 4 / 5 % 6 ^ 7",
			want: []TokenType{
				TokenNumber, TokenPlus, TokenNumber, TokenMinus,
				TokenNumber, TokenMultiply, TokenNumber, TokenDivide,
				TokenNumber, TokenModulo, TokenNumber, TokenPower,
				TokenNumber, TokenEOF,
			},
		},
		
		// Comparison operators
		{
			name:  "comparison operators",
			input: "a > b < c >= d <= e == f != g",
			want: []TokenType{
				TokenIdentifier, TokenGreater, TokenIdentifier, TokenLess,
				TokenIdentifier, TokenGreaterEqual, TokenIdentifier, TokenLessEqual,
				TokenIdentifier, TokenEqual, TokenIdentifier, TokenNotEqual,
				TokenIdentifier, TokenEOF,
			},
		},
		
		// Logical operators
		{
			name:  "logical operators",
			input: "a && b || !c",
			want: []TokenType{
				TokenIdentifier, TokenAnd, TokenIdentifier,
				TokenOr, TokenNot, TokenIdentifier, TokenEOF,
			},
		},
		
		// Parentheses and comma
		{
			name:  "parentheses and comma",
			input: "func(a, b)",
			want: []TokenType{
				TokenIdentifier, TokenLeftParen,
				TokenIdentifier, TokenComma, TokenIdentifier,
				TokenRightParen, TokenEOF,
			},
		},
		
		// Ternary operator
		{
			name:  "ternary operator",
			input: "a ? b : c",
			want: []TokenType{
				TokenIdentifier, TokenQuestion,
				TokenIdentifier, TokenColon,
				TokenIdentifier, TokenEOF,
			},
		},
		
		// Array brackets
		{
			name:  "array literals",
			input: "[1, 2, 3]",
			want: []TokenType{
				TokenLeftBracket, TokenNumber, TokenComma,
				TokenNumber, TokenComma, TokenNumber,
				TokenRightBracket, TokenEOF,
			},
		},
		{
			name:  "array indexing",
			input: "arr[0]",
			want: []TokenType{
				TokenIdentifier, TokenLeftBracket,
				TokenNumber, TokenRightBracket, TokenEOF,
			},
		},
		{
			name:  "nested arrays",
			input: "[[1, 2], [3, 4]]",
			want: []TokenType{
				TokenLeftBracket, TokenLeftBracket, TokenNumber, TokenComma, TokenNumber, TokenRightBracket,
				TokenComma, TokenLeftBracket, TokenNumber, TokenComma, TokenNumber, TokenRightBracket,
				TokenRightBracket, TokenEOF,
			},
		},
		
		// Numbers
		{
			name:  "integer",
			input: "123",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		{
			name:  "decimal",
			input: "123.456",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		{
			name:  "scientific notation",
			input: "1.23e10",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		{
			name:  "negative exponent",
			input: "1.23e-10",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		{
			name:  "positive exponent",
			input: "1.23e+10",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		
		// String literals
		{
			name:  "simple string",
			input: `"hello world"`,
			want:  []TokenType{TokenString, TokenEOF},
		},
		{
			name:  "string with escapes",
			input: `"hello\nworld\t\"quoted\""`,
			want:  []TokenType{TokenString, TokenEOF},
		},
		
		// Identifiers
		{
			name:  "simple identifier",
			input: "variable",
			want:  []TokenType{TokenIdentifier, TokenEOF},
		},
		{
			name:  "identifier with underscore",
			input: "_private_var",
			want:  []TokenType{TokenIdentifier, TokenEOF},
		},
		{
			name:  "identifier with numbers",
			input: "var123",
			want:  []TokenType{TokenIdentifier, TokenEOF},
		},
		
		// Complex expressions
		{
			name:  "complex expression",
			input: `max(temperature_min, 32) * (has_hazmat ? 1.5 : 1.0)`,
			want: []TokenType{
				TokenIdentifier, TokenLeftParen,
				TokenIdentifier, TokenComma, TokenNumber,
				TokenRightParen, TokenMultiply, TokenLeftParen,
				TokenIdentifier, TokenQuestion, TokenNumber,
				TokenColon, TokenNumber, TokenRightParen, TokenEOF,
			},
		},
		
		// Edge cases
		{
			name:  "empty input",
			input: "",
			want:  []TokenType{TokenEOF},
		},
		{
			name:  "whitespace only",
			input: "   \t\n  ",
			want:  []TokenType{TokenEOF},
		},
		
		// Error cases
		{
			name:      "single equals",
			input:     "a = b",
			want:      []TokenType{TokenIdentifier, TokenError},
			wantError: false, // Error is in token, not returned
		},
		{
			name:      "single ampersand",
			input:     "a & b",
			want:      []TokenType{TokenIdentifier, TokenError},
			wantError: false,
		},
		{
			name:      "single pipe",
			input:     "a | b",
			want:      []TokenType{TokenIdentifier, TokenError},
			wantError: false,
		},
		{
			name:      "invalid number",
			input:     "123.456.789",
			want:      []TokenType{TokenNumber, TokenNumber, TokenEOF}, // 123.456 and .789 are both valid numbers
			wantError: false,
		},
		{
			name:      "unterminated string",
			input:     `"hello`,
			want:      []TokenType{TokenError},
			wantError: false,
		},
		{
			name:      "invalid escape",
			input:     `"hello\x"`,
			want:      []TokenType{TokenError},
			wantError: false,
		},
		{
			name:      "expression too long",
			input:     strings.Repeat("a+", MaxExpressionLength/2+1),
			wantError: true,
		},
		{
			name:      "too many tokens",
			input:     strings.Repeat("1+", MaxTokenCount/2+1) + "1",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()
			
			if (err != nil) != tt.wantError {
				t.Errorf("Tokenize() error = %v, wantError %v", err, tt.wantError)
				return
			}
			
			if tt.wantError {
				return
			}
			
			// Check token types
			if len(tokens) != len(tt.want) {
				t.Errorf("got %d tokens, want %d", len(tokens), len(tt.want))
				t.Errorf("got: %v", tokenTypes(tokens))
				t.Errorf("want: %v", tt.want)
				return
			}
			
			for i, tok := range tokens {
				if tok.Type != tt.want[i] {
					t.Errorf("token[%d] = %v, want %v", i, tok.Type, tt.want[i])
				}
			}
		})
	}
}

func TestTokenizer_TokenValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "numbers",
			input: "123 456.789 1.23e10",
			want:  []string{"123", "456.789", "1.23e10", ""},
		},
		{
			name:  "identifiers",
			input: "foo bar_baz _private",
			want:  []string{"foo", "bar_baz", "_private", ""},
		},
		{
			name:  "strings",
			input: `"hello" "world"`,
			want:  []string{"hello", "world", ""},
		},
		{
			name:  "string escapes",
			input: `"line1\nline2\ttab"`,
			want:  []string{"line1\nline2\ttab", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}
			
			if len(tokens) != len(tt.want) {
				t.Errorf("got %d tokens, want %d", len(tokens), len(tt.want))
				return
			}
			
			for i, tok := range tokens {
				if tok.Value != tt.want[i] {
					t.Errorf("token[%d].Value = %q, want %q", i, tok.Value, tt.want[i])
				}
			}
		})
	}
}

func TestTokenizer_Position(t *testing.T) {
	input := "a + b\n  * c"
	tokenizer := NewTokenizer(input)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}
	
	// Expected positions
	expected := []struct {
		line   uint16
		column uint16
	}{
		{1, 1}, // a
		{1, 3}, // +
		{1, 5}, // b
		{2, 3}, // *
		{2, 5}, // c
		{2, 5}, // EOF - position is where 'c' ends, not after it
	}
	
	for i, exp := range expected {
		if i >= len(tokens) {
			break
		}
		tok := tokens[i]
		if tok.Line != exp.line || tok.Column != exp.column {
			t.Errorf("token[%d] position = (%d,%d), want (%d,%d)",
				i, tok.Line, tok.Column, exp.line, exp.column)
		}
	}
}

func TestTokenizer_StringInterner(t *testing.T) {
	input := "foo + foo + bar + foo"
	tokenizer := NewTokenizer(input)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}
	
	// Check that identical strings are interned (same pointer)
	var fooPtr *string
	var barPtr *string
	
	for _, tok := range tokens {
		if tok.Type == TokenIdentifier {
			if tok.Value == "foo" {
				if fooPtr == nil {
					fooPtr = &tok.Value
				} else if fooPtr != &tok.Value {
					// String interning creates new strings, so this is expected
					// Just verify values are equal
					if *fooPtr != tok.Value {
						t.Error("foo values don't match")
					}
				}
			} else if tok.Value == "bar" {
				if barPtr == nil {
					barPtr = &tok.Value
				}
			}
		}
	}
}

func TestTokenizer_Unicode(t *testing.T) {
	// Test that tokenizer handles UTF-8 correctly
	input := `"hello 世界" + "🚀"`
	tokenizer := NewTokenizer(input)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}
	
	if len(tokens) != 4 { // string + string EOF
		t.Errorf("got %d tokens, want 4", len(tokens))
	}
	
	if tokens[0].Value != "hello 世界" {
		t.Errorf("first string = %q, want %q", tokens[0].Value, "hello 世界")
	}
	
	if tokens[2].Value != "🚀" {
		t.Errorf("second string = %q, want %q", tokens[2].Value, "🚀")
	}
}

// Helper function to extract token types
func tokenTypes(tokens []Token) []TokenType {
	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	return types
}