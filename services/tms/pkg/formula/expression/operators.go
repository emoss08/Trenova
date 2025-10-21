package expression

import (
	"fmt"
	"math"

	"github.com/emoss08/trenova/pkg/formula/conversion"
	"github.com/emoss08/trenova/pkg/formula/errors"
)

// * evaluateBinaryOp evaluates a binary operation
func evaluateBinaryOp(op TokenType, left, right any) (any, error) {
	switch op { //nolint:exhaustive // all operators are covered
	case TokenPlus:
		return addValues(left, right)
	case TokenMinus:
		return subtractValues(left, right)
	case TokenMultiply:
		return multiplyValues(left, right)
	case TokenDivide:
		return divideValues(left, right)
	case TokenModulo:
		return moduloValues(left, right)
	case TokenPower:
		return powerValues(left, right)
	case TokenEqual:
		return equalValues(left, right), nil
	case TokenNotEqual:
		return !equalValues(left, right), nil
	case TokenGreater:
		return compareValues(left, right) > 0, nil
	case TokenLess:
		return compareValues(left, right) < 0, nil
	case TokenGreaterEqual:
		return compareValues(left, right) >= 0, nil
	case TokenLessEqual:
		return compareValues(left, right) <= 0, nil
	case TokenAnd:
		return toBool(left) && toBool(right), nil
	case TokenOr:
		return toBool(left) || toBool(right), nil

	default:
		return nil, fmt.Errorf("unknown binary operator: %s", op)
	}
}

func evaluateUnaryOp(op TokenType, operand any) (any, error) {
	switch op { //nolint:exhaustive // all operators are covered
	case TokenNot:
		return !toBool(operand), nil
	case TokenMinus:
		return negateValue(operand)
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

func addValues(left, right any) (any, error) {
	if l, ok := left.(string); ok {
		if r, valOk := right.(string); valOk {
			return l + r, nil
		}
		return l + fmt.Sprint(right), nil
	}
	if r, ok := right.(string); ok {
		return fmt.Sprint(left) + r, nil
	}

	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if !ok1 || !ok2 {
		return nil, errors.NewTransformError(
			"addition",
			"number",
			left,
			fmt.Errorf("cannot add %T and %T", left, right),
		)
	}

	if (r > 0 && l > math.MaxFloat64-r) || (r < 0 && l < -math.MaxFloat64-r) {
		return nil, errors.NewComputeError("addition", "overflow", ErrNumericOverflow)
	}

	return l + r, nil
}

func subtractValues(left, right any) (any, error) {
	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if !ok1 || !ok2 {
		return nil, errors.NewTransformError(
			"subtraction",
			"number",
			left,
			fmt.Errorf("cannot subtract %T from %T", right, left),
		)
	}

	if (r < 0 && l > math.MaxFloat64+r) || (r > 0 && l < -math.MaxFloat64+r) {
		return nil, errors.NewComputeError(
			"subtraction",
			"overflow",
			ErrNumericOverflow,
		)
	}

	return l - r, nil
}

func multiplyValues(left, right any) (any, error) {
	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if !ok1 || !ok2 {
		return nil, errors.NewTransformError(
			"multiplication",
			"number",
			left,
			fmt.Errorf("cannot multiply %T and %T", left, right),
		)
	}

	if l != 0 && math.Abs(r) > math.MaxFloat64/math.Abs(l) {
		return nil, errors.NewComputeError(
			"multiplication",
			"overflow",
			ErrNumericOverflow,
		)
	}

	return l * r, nil
}

func divideValues(left, right any) (any, error) {
	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if !ok1 || !ok2 {
		return nil, errors.NewTransformError(
			"division",
			"number",
			left,
			fmt.Errorf("cannot divide %T by %T", left, right),
		)
	}

	if r == 0 {
		return nil, errors.NewComputeError("division", "zero", ErrDivisionByZero)
	}

	return l / r, nil
}

func moduloValues(left, right any) (any, error) {
	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if !ok1 || !ok2 {
		return nil, errors.NewTransformError(
			"modulo",
			"number",
			left,
			fmt.Errorf("cannot modulo %T by %T", left, right),
		)
	}

	if r == 0 {
		return nil, errors.NewComputeError("modulo", "zero", ErrModuloByZero)
	}

	return math.Mod(l, r), nil
}

func powerValues(left, right any) (any, error) {
	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if !ok1 || !ok2 {
		return nil, errors.NewTransformError(
			"power",
			"number",
			left,
			fmt.Errorf("cannot raise %T to power %T", left, right),
		)
	}

	result := math.Pow(l, r)

	if math.IsInf(result, 0) || math.IsNaN(result) {
		return nil, errors.NewComputeError(
			"power",
			"overflow",
			ErrPowerResultOutOfRange,
		)
	}

	return result, nil
}

func negateValue(operand any) (any, error) {
	val, ok := conversion.ToFloat64(operand)
	if !ok {
		return nil, errors.NewTransformError(
			"negation",
			"number",
			operand,
			fmt.Errorf("cannot negate %T", operand),
		)
	}

	return -val, nil
}

func equalValues(left, right any) bool {
	if left == nil || right == nil {
		return left == right
	}

	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if ok1 && ok2 {
		return math.Abs(l-r) < 1e-9
	}

	if ls, ok := left.(string); ok {
		if rs, valOk := right.(string); valOk {
			return ls == rs
		}
	}

	if lb, ok := left.(bool); ok {
		if rb, valOk := right.(bool); valOk {
			return lb == rb
		}
	}

	if la, ok := left.([]any); ok { //nolint:nestif // this is fine
		if ra, valOk := right.([]any); valOk {
			if len(la) != len(ra) {
				return false
			}
			for i := range la {
				if !equalValues(la[i], ra[i]) {
					return false
				}
			}
			return true
		}
	}

	return left == right
}

func compareValues(left, right any) int {
	l, ok1 := conversion.ToFloat64(left)
	r, ok2 := conversion.ToFloat64(right)
	if ok1 && ok2 {
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
		return 0
	}

	if ls, ok := left.(string); ok { //nolint:nestif // this is fine
		if rs, valOk := right.(string); valOk {
			if ls < rs {
				return -1
			}
			if ls > rs {
				return 1
			}
			return 0
		}
	}

	return 0
}
