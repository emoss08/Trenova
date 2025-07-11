package expression

import (
	"fmt"
	"strings"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantAST   string // String representation of AST
		wantError bool
	}{
		// Literals
		{
			name:    "number literal",
			input:   "42",
			wantAST: "42",
		},
		{
			name:    "decimal number",
			input:   "3.14",
			wantAST: "3.14",
		},
		{
			name:    "string literal",
			input:   `"hello"`,
			wantAST: `"hello"`,
		},
		{
			name:    "identifier",
			input:   "variable",
			wantAST: "variable",
		},
		
		// Array literals
		{
			name:    "empty array",
			input:   "[]",
			wantAST: "[]",
		},
		{
			name:    "array with one element",
			input:   "[42]",
			wantAST: "[42]",
		},
		{
			name:    "array with multiple elements",
			input:   "[1, 2, 3]",
			wantAST: "[1, 2, 3]",
		},
		{
			name:    "array with mixed types",
			input:   `[1, "hello", true]`,
			wantAST: `[1, "hello", true]`,
		},
		{
			name:    "nested arrays",
			input:   "[[1, 2], [3, 4]]",
			wantAST: "[[1, 2], [3, 4]]",
		},
		
		// Array indexing
		{
			name:    "simple array indexing",
			input:   "arr[0]",
			wantAST: "arr[0]",
		},
		{
			name:    "array literal indexing",
			input:   "[1, 2, 3][1]",
			wantAST: "[1, 2, 3][1]",
		},
		{
			name:    "nested indexing",
			input:   "matrix[0][1]",
			wantAST: "matrix[0][1]",
		},
		{
			name:    "computed index",
			input:   "arr[i + 1]",
			wantAST: "arr[(i + 1)]",
		},
		
		// Binary operations
		{
			name:    "addition",
			input:   "1 + 2",
			wantAST: "(1 + 2)",
		},
		{
			name:    "subtraction",
			input:   "5 - 3",
			wantAST: "(5 - 3)",
		},
		{
			name:    "multiplication",
			input:   "4 * 5",
			wantAST: "(4 * 5)",
		},
		{
			name:    "division",
			input:   "10 / 2",
			wantAST: "(10 / 2)",
		},
		{
			name:    "modulo",
			input:   "10 % 3",
			wantAST: "(10 % 3)",
		},
		{
			name:    "power",
			input:   "2 ^ 3",
			wantAST: "(2 ^ 3)",
		},
		
		// Operator precedence
		{
			name:    "precedence: multiplication before addition",
			input:   "1 + 2 * 3",
			wantAST: "(1 + (2 * 3))",
		},
		{
			name:    "precedence: power before multiplication",
			input:   "2 * 3 ^ 4",
			wantAST: "(2 * (3 ^ 4))",
		},
		{
			name:    "precedence: parentheses override",
			input:   "(1 + 2) * 3",
			wantAST: "((1 + 2) * 3)",
		},
		{
			name:    "precedence: complex",
			input:   "1 + 2 * 3 ^ 4 - 5",
			wantAST: "((1 + (2 * (3 ^ 4))) - 5)",
		},
		
		// Comparison operators
		{
			name:    "greater than",
			input:   "a > b",
			wantAST: "(a > b)",
		},
		{
			name:    "less than",
			input:   "x < 10",
			wantAST: "(x < 10)",
		},
		{
			name:    "greater or equal",
			input:   "price >= 100",
			wantAST: "(price >= 100)",
		},
		{
			name:    "less or equal",
			input:   "count <= 5",
			wantAST: "(count <= 5)",
		},
		{
			name:    "equal",
			input:   "status == 1",
			wantAST: "(status == 1)",
		},
		{
			name:    "not equal",
			input:   "type != 0",
			wantAST: "(type != 0)",
		},
		
		// Logical operators
		{
			name:    "logical and",
			input:   "a && b",
			wantAST: "(a && b)",
		},
		{
			name:    "logical or",
			input:   "x || y",
			wantAST: "(x || y)",
		},
		{
			name:    "logical not",
			input:   "!active",
			wantAST: "(!active)",
		},
		{
			name:    "logical precedence",
			input:   "a || b && c",
			wantAST: "(a || (b && c))",
		},
		{
			name:    "complex logical",
			input:   "a > 5 && b < 10 || c == 0",
			wantAST: "(((a > 5) && (b < 10)) || (c == 0))",
		},
		
		// Unary operators
		{
			name:    "negative number",
			input:   "-42",
			wantAST: "(-42)",
		},
		{
			name:    "negative expression",
			input:   "-(a + b)",
			wantAST: "(-(a + b))",
		},
		{
			name:    "double negative",
			input:   "--x",
			wantAST: "(-(-x))",
		},
		{
			name:    "not not",
			input:   "!!flag",
			wantAST: "(!(!flag))",
		},
		
		// Ternary conditional
		{
			name:    "simple ternary",
			input:   "a ? b : c",
			wantAST: "(a ? b : c)",
		},
		{
			name:    "nested ternary",
			input:   "a ? b ? c : d : e",
			wantAST: "(a ? (b ? c : d) : e)",
		},
		{
			name:    "ternary with expressions",
			input:   "x > 0 ? x * 2 : -x",
			wantAST: "((x > 0) ? (x * 2) : (-x))",
		},
		
		// Function calls
		{
			name:    "function no args",
			input:   "random()",
			wantAST: "random()",
		},
		{
			name:    "function one arg",
			input:   "abs(-5)",
			wantAST: "abs((-5))",
		},
		{
			name:    "function multiple args",
			input:   "max(a, b, c)",
			wantAST: "max(a, b, c)",
		},
		{
			name:    "nested function calls",
			input:   "min(abs(x), max(y, z))",
			wantAST: "min(abs(x), max(y, z))",
		},
		{
			name:    "function in expression",
			input:   "2 * max(a, b) + 1",
			wantAST: "((2 * max(a, b)) + 1)",
		},
		
		// Complex expressions
		{
			name:    "formula example 1",
			input:   "base_rate * distance",
			wantAST: "(base_rate * distance)",
		},
		{
			name:    "formula example 2",
			input:   "has_hazmat ? base_rate * 1.25 : base_rate",
			wantAST: "(has_hazmat ? (base_rate * 1.25) : base_rate)",
		},
		{
			name:    "formula example 3",
			input:   "(weight * weight_rate + distance * distance_rate) * (1 + fuel_surcharge_pct)",
			wantAST: "(((weight * weight_rate) + (distance * distance_rate)) * (1 + fuel_surcharge_pct))",
		},
		{
			name:    "formula example 4",
			input:   "max(min_charge, weight * rate_per_pound)",
			wantAST: "max(min_charge, (weight * rate_per_pound))",
		},
		
		// Error cases
		{
			name:      "empty expression",
			input:     "",
			wantError: true,
		},
		{
			name:      "missing operand",
			input:     "1 +",
			wantError: true,
		},
		{
			name:      "missing closing paren",
			input:     "(1 + 2",
			wantError: true,
		},
		{
			name:      "missing ternary colon",
			input:     "a ? b",
			wantError: true,
		},
		{
			name:      "unexpected token", 
			input:     "1 + + 2",
			wantError: true, // Unary plus not supported
		},
		{
			name:      "trailing comma",
			input:     "max(1, 2,)",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()
			if err != nil {
				if !tt.wantError {
					t.Fatalf("Tokenize() error = %v", err)
				}
				return
			}
			
			// Parse
			parser := NewParser(tokens)
			ast, err := parser.Parse()
			
			if (err != nil) != tt.wantError {
				t.Errorf("Parse() error = %v, wantError %v", err, tt.wantError)
				return
			}
			
			if tt.wantError {
				return
			}
			
			// Check AST string representation
			gotAST := ast.String()
			if gotAST != tt.wantAST {
				t.Errorf("AST = %s, want %s", gotAST, tt.wantAST)
			}
		})
	}
}

func TestParser_ComplexityCheck(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantComplexity int
	}{
		{
			name:           "simple literal",
			input:          "42",
			wantComplexity: 1,
		},
		{
			name:           "simple operation",
			input:          "a + b",
			wantComplexity: 5, // 2 vars (2 each) + 1 op = 5
		},
		{
			name:           "function call",
			input:          "max(a, b)",
			wantComplexity: 7, // 3 base + 2*2 args = 7
		},
		{
			name:           "nested expression",
			input:          "a + b * c",
			wantComplexity: 8, // 3 vars (2 each) + 2 ops = 8
		},
		{
			name:           "ternary",
			input:          "a ? b : c",
			wantComplexity: 5, // condition(2) + max(true(2), false(2)) + 1 = 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize and parse
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}
			
			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			
			// Check complexity
			got := ast.Complexity()
			if got != tt.wantComplexity {
				t.Errorf("Complexity() = %d, want %d", got, tt.wantComplexity)
			}
		})
	}
}

func TestParser_ErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrMsg  string
	}{
		{
			name:       "missing closing paren",
			input:      "(1 + 2",
			wantErrMsg: "expected ')'",
		},
		{
			name:       "missing ternary colon",
			input:      "a ? b",
			wantErrMsg: "expected ':'",
		},
		{
			name:       "unexpected EOF",
			input:      "1 +",
			wantErrMsg: "unexpected token: EOF",
		},
		{
			name:       "invalid number",
			input:      "1.2.3", // This tokenizes as two numbers
			wantErrMsg: "unexpected token after expression",
		},
		{
			name:       "missing function close",
			input:      "max(1, 2",
			wantErrMsg: "expected ')'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}
			
			// Parse
			parser := NewParser(tokens)
			_, err = parser.Parse()
			
			if err == nil {
				t.Fatalf("Parse() expected error, got nil")
			}
			
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Error message = %q, want to contain %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

// Test that parser creates correct AST node types
func TestParser_NodeTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkFn  func(Node) error
	}{
		{
			name:  "number node",
			input: "42",
			checkFn: func(n Node) error {
				if _, ok := n.(*NumberNode); !ok {
					return fmt.Errorf("expected NumberNode, got %T", n)
				}
				return nil
			},
		},
		{
			name:  "string node",
			input: `"test"`,
			checkFn: func(n Node) error {
				if _, ok := n.(*StringNode); !ok {
					return fmt.Errorf("expected StringNode, got %T", n)
				}
				return nil
			},
		},
		{
			name:  "identifier node",
			input: "variable",
			checkFn: func(n Node) error {
				if _, ok := n.(*IdentifierNode); !ok {
					return fmt.Errorf("expected IdentifierNode, got %T", n)
				}
				return nil
			},
		},
		{
			name:  "binary op node",
			input: "a + b",
			checkFn: func(n Node) error {
				bn, ok := n.(*BinaryOpNode)
				if !ok {
					return fmt.Errorf("expected BinaryOpNode, got %T", n)
				}
				if bn.Operator != TokenPlus {
					return fmt.Errorf("expected TokenPlus, got %v", bn.Operator)
				}
				return nil
			},
		},
		{
			name:  "unary op node",
			input: "-x",
			checkFn: func(n Node) error {
				un, ok := n.(*UnaryOpNode)
				if !ok {
					return fmt.Errorf("expected UnaryOpNode, got %T", n)
				}
				if un.Operator != TokenMinus {
					return fmt.Errorf("expected TokenMinus, got %v", un.Operator)
				}
				return nil
			},
		},
		{
			name:  "conditional node",
			input: "a ? b : c",
			checkFn: func(n Node) error {
				if _, ok := n.(*ConditionalNode); !ok {
					return fmt.Errorf("expected ConditionalNode, got %T", n)
				}
				return nil
			},
		},
		{
			name:  "function call node",
			input: "max(1, 2)",
			checkFn: func(n Node) error {
				fn, ok := n.(*FunctionCallNode)
				if !ok {
					return fmt.Errorf("expected FunctionCallNode, got %T", n)
				}
				if fn.Name != "max" {
					return fmt.Errorf("expected function name 'max', got %q", fn.Name)
				}
				if len(fn.Arguments) != 2 {
					return fmt.Errorf("expected 2 arguments, got %d", len(fn.Arguments))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize and parse
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}
			
			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			
			// Check node type
			if err := tt.checkFn(ast); err != nil {
				t.Error(err)
			}
		})
	}
}