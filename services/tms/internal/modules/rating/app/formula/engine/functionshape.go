package engine

import (
	"context"
	"reflect"
	"time"
)

const (
	ValueNumber   = "Number"
	ValueInteger  = "Integer"
	ValueString   = "String"
	ValueBoolean  = "Boolean"
	ValueDate     = "Date"
	ValueDuration = "Duration"
	ValueArray    = "Array"
	ValueObject   = "Object"
	ValueAny      = "Any"

	VariadicArgs = -1
)

type FunctionShape struct {
	MinArgs     int    `json:"minArgs"`
	MaxArgs     int    `json:"maxArgs"`
	Returns     string `json:"returns"`
	NumericArgs bool   `json:"numericArgs"`
	Predicate   bool   `json:"predicate"`
}

func shape(minArgs, maxArgs int, returns string) FunctionShape {
	return FunctionShape{MinArgs: minArgs, MaxArgs: maxArgs, Returns: returns}
}

func predicateShape(minArgs, maxArgs int, returns string) FunctionShape {
	return FunctionShape{MinArgs: minArgs, MaxArgs: maxArgs, Returns: returns, Predicate: true}
}

var (
	contextType  = reflect.TypeFor[context.Context]()
	timeType     = reflect.TypeFor[time.Time]()
	durationType = reflect.TypeFor[time.Duration]()
)

func deriveShape(signatures []any) FunctionShape {
	derived := FunctionShape{MinArgs: -1, Returns: ""}
	numeric := true
	for _, signature := range signatures {
		fnType := reflect.TypeOf(signature).Elem()
		params := make([]reflect.Type, 0, fnType.NumIn())
		for i := range fnType.NumIn() {
			if fnType.In(i) == contextType {
				continue
			}
			params = append(params, fnType.In(i))
		}

		count := len(params)
		if fnType.IsVariadic() {
			count--
			derived.MaxArgs = VariadicArgs
			if count == 0 {
				count = 1
			}
		} else if derived.MaxArgs != VariadicArgs && count > derived.MaxArgs {
			derived.MaxArgs = count
		}
		if derived.MinArgs == -1 || count < derived.MinArgs {
			derived.MinArgs = count
		}

		for i, param := range params {
			if fnType.IsVariadic() && i == len(params)-1 {
				param = param.Elem()
			}
			if !isNumericKind(param.Kind()) {
				numeric = false
			}
		}

		if fnType.NumOut() > 0 {
			derived.Returns = joinValueTypes(derived.Returns, valueTypeOf(fnType.Out(0)))
		}
	}
	if derived.MinArgs < 0 {
		derived.MinArgs = 0
	}
	if derived.Returns == "" {
		derived.Returns = ValueAny
	}
	derived.NumericArgs = numeric && len(signatures) > 0
	return derived
}

func joinValueTypes(current, next string) string {
	if current == "" || current == next {
		return next
	}
	return ValueAny
}

//nolint:exhaustive // the other kinds have no value type in the formula language
var kindValueTypes = map[reflect.Kind]string{
	reflect.Float32: ValueNumber,
	reflect.Float64: ValueNumber,
	reflect.Int:     ValueInteger,
	reflect.Int8:    ValueInteger,
	reflect.Int16:   ValueInteger,
	reflect.Int32:   ValueInteger,
	reflect.Int64:   ValueInteger,
	reflect.Uint:    ValueInteger,
	reflect.Uint8:   ValueInteger,
	reflect.Uint16:  ValueInteger,
	reflect.Uint32:  ValueInteger,
	reflect.Uint64:  ValueInteger,
	reflect.String:  ValueString,
	reflect.Bool:    ValueBoolean,
	reflect.Slice:   ValueArray,
	reflect.Array:   ValueArray,
	reflect.Map:     ValueObject,
	reflect.Struct:  ValueObject,
}

func isNumericKind(kind reflect.Kind) bool {
	valueType := kindValueTypes[kind]
	return valueType == ValueNumber || valueType == ValueInteger
}

func valueTypeOf(typ reflect.Type) string {
	switch typ {
	case timeType:
		return ValueDate
	case durationType:
		return ValueDuration
	}
	if valueType, ok := kindValueTypes[typ.Kind()]; ok {
		return valueType
	}
	return ValueAny
}

type BuiltinSpec struct {
	Name      string `json:"name"`
	Signature string `json:"signature"`
	FunctionShape
}

var builtinSpecs = []BuiltinSpec{
	{"all", "all(list, predicate)", predicateShape(2, 2, ValueBoolean)},
	{"none", "none(list, predicate)", predicateShape(2, 2, ValueBoolean)},
	{"any", "any(list, predicate)", predicateShape(2, 2, ValueBoolean)},
	{"one", "one(list, predicate)", predicateShape(2, 2, ValueBoolean)},
	{"filter", "filter(list, predicate)", predicateShape(2, 2, ValueArray)},
	{"map", "map(list, predicate)", predicateShape(2, 2, ValueArray)},
	{"find", "find(list, predicate)", predicateShape(2, 2, ValueAny)},
	{"findIndex", "findIndex(list, predicate)", predicateShape(2, 2, ValueInteger)},
	{"findLast", "findLast(list, predicate)", predicateShape(2, 2, ValueAny)},
	{"findLastIndex", "findLastIndex(list, predicate)", predicateShape(2, 2, ValueInteger)},
	{"count", "count(list, predicate?)", predicateShape(1, 2, ValueInteger)},
	{"groupBy", "groupBy(list, predicate)", predicateShape(2, 2, ValueObject)},
	{"sortBy", "sortBy(list, predicate, order?)", predicateShape(2, 3, ValueArray)},
	{"reduce", "reduce(list, predicate, initial?)", predicateShape(2, 3, ValueAny)},
	{"type", "type(value)", shape(1, 1, ValueString)},
	{"splitAfter", `splitAfter(text, "separator", n?)`, shape(2, 3, ValueArray)},
	{"join", `join(list, "separator"?)`, shape(1, 2, ValueString)},
	{"lastIndexOf", `lastIndexOf(text, "part")`, shape(2, 2, ValueInteger)},
	{"mean", "mean(...values)", shape(1, VariadicArgs, ValueNumber)},
	{"median", "median(...values)", shape(1, VariadicArgs, ValueNumber)},
	{"toBase64", "toBase64(text)", shape(1, 1, ValueString)},
	{"fromBase64", "fromBase64(text)", shape(1, 1, ValueString)},
	{"duration", `duration("1h30m")`, shape(1, 1, ValueDuration)},
	{"date", `date(text, layout?, timezone?)`, shape(1, 3, ValueDate)},
	{"timezone", `timezone("America/Chicago")`, shape(1, 1, ValueAny)},
	{"first", "first(list)", shape(1, 1, ValueAny)},
	{"last", "last(list)", shape(1, 1, ValueAny)},
	{"get", "get(value, key)", shape(2, 2, ValueAny)},
	{"take", "take(list, n)", shape(2, 2, ValueArray)},
	{"keys", "keys(record)", shape(1, 1, ValueArray)},
	{"values", "values(record)", shape(1, 1, ValueArray)},
	{"toPairs", "toPairs(record)", shape(1, 1, ValueArray)},
	{"fromPairs", "fromPairs(list)", shape(1, 1, ValueObject)},
	{"reverse", "reverse(list)", shape(1, 1, ValueArray)},
	{"uniq", "uniq(list)", shape(1, 1, ValueArray)},
	{"concat", "concat(...lists)", shape(1, VariadicArgs, ValueArray)},
	{"flatten", "flatten(list)", shape(1, 1, ValueArray)},
	{"sort", `sort(list, "asc"?)`, shape(1, 2, ValueArray)},
	{"bitand", "bitand(a, b)", shape(2, 2, ValueInteger)},
	{"bitor", "bitor(a, b)", shape(2, 2, ValueInteger)},
	{"bitxor", "bitxor(a, b)", shape(2, 2, ValueInteger)},
	{"bitnand", "bitnand(a, b)", shape(2, 2, ValueInteger)},
	{"bitshl", "bitshl(a, b)", shape(2, 2, ValueInteger)},
	{"bitshr", "bitshr(a, b)", shape(2, 2, ValueInteger)},
	{"bitushr", "bitushr(a, b)", shape(2, 2, ValueInteger)},
	{"bitnot", "bitnot(a)", shape(1, 1, ValueInteger)},
}

func BuiltinSpecs() []BuiltinSpec {
	specs := make([]BuiltinSpec, len(builtinSpecs))
	copy(specs, builtinSpecs)
	return specs
}
