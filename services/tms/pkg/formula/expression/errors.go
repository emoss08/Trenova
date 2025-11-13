package expression

import "errors"

var (
	ErrAbsArgumentMustBeNumber        = errors.New("abs argument must be a number")
	ErrMinRequiresAtLeastOneArgument  = errors.New("min: requires at least one argument")
	ErrMinAllArgumentsMustBeNumbers   = errors.New("min: all arguments must be numbers")
	ErrMaxRequiresAtLeastOneArgument  = errors.New("max: requires at least one argument")
	ErrMaxAllArgumentsMustBeNumbers   = errors.New("max: all arguments must be numbers")
	ErrRoundFirstArgumentMustBeNumber = errors.New(
		"round: first argument must be a number",
	)
	ErrRoundPrecisionMustBeNumber               = errors.New("round: precision must be a number")
	ErrFloorArgumentMustBeNumber                = errors.New("floor: argument must be a number")
	ErrCeilArgumentMustBeNumber                 = errors.New("ceil: argument must be a number")
	ErrSqrtArgumentMustBeNumber                 = errors.New("sqrt: argument must be a number")
	ErrSqrtcannotTakeSquareRootOfNegativeNumber = errors.New(
		"sqrt: cannot take square root of negative number",
	)
	ErrPowBaseMustBeNumber          = errors.New("pow: base must be a number")
	ErrPowExponentMustBeNumber      = errors.New("pow: exponent must be a number")
	ErrPowResultOutOfRange          = errors.New("pow: result out of range")
	ErrLogArgumentMustBeNumber      = errors.New("log: argument must be a number")
	ErrLogArgumentMustBePositive    = errors.New("log: argument must be positive")
	ErrLogBaseMustBeNumber          = errors.New("log: base must be a number")
	ErrLogBaseMustBePositive        = errors.New("log: base must be positive")
	ErrLogBaseMustNotBeOne          = errors.New("log: base must not be one")
	ErrExpArgumentMustBeNumber      = errors.New("exp: argument must be a number")
	ErrExpResultOutOfRange          = errors.New("exp: result out of range")
	ErrSinArgumentMustBeNumber      = errors.New("sin: argument must be a number")
	ErrCosArgumentMustBeNumber      = errors.New("cos: argument must be a number")
	ErrTanArgumentMustBeNumber      = errors.New("tan: argument must be a number")
	ErrLenArgumentMustBeArray       = errors.New("len: argument must be an array")
	ErrSumAllArgumentsMustBeNumbers = errors.New(
		"sum: all arguments must be numbers or arrays",
	)
	ErrSumAllElementsMustBeNumbers  = errors.New("sum: all elements must be numbers")
	ErrAvgAllElementsMustBeNumbers  = errors.New("avg: all elements must be numbers")
	ErrAvgAllArgumentsMustBeNumbers = errors.New(
		"avg: all arguments must be numbers or arrays",
	)
	ErrAvgCannotComputeAverageOfEmptyArray = errors.New(
		"avg: cannot compute average of empty array",
	)
	ErrSliceFirstArgumentMustBeArrayOrString = errors.New(
		"slice: first argument must be array or string",
	)
	ErrSliceStartIndexMustBeNumber              = errors.New("slice: start index must be a number")
	ErrSliceEndIndexMustBeNumber                = errors.New("slice: end index must be a number")
	ErrContainsFirstArgumentMustBeStringOrArray = errors.New(
		"contains: first argument must be string or array",
	)
	ErrIndexOfFirstArgumentMustBeStringOrArray = errors.New(
		"indexOf: first argument must be string or array",
	)
	ErrNumericOverflow = errors.New("numeric overflow")
	ErrDivisionByZero                           = errors.New("division by zero")
	ErrModuloByZero                             = errors.New("modulo by zero")
	ErrPowerResultOutOfRange                    = errors.New("power: result out of range")
	ErrNoTokensToParse                          = errors.New("no tokens to parse")
	ErrInvalidNumberExpectedDigitsAfterExponent = errors.New(
		"invalid number: expected digits after exponent",
	)
)
