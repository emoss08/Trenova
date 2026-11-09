package ozzoerrors

import (
	"errors"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/fieldpath"
	val "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Kind int

const (
	KindInvalid Kind = iota
	KindRequired
	KindLength
	KindFormat
)

type Leaf struct {
	Path    fieldpath.Path
	Kind    Kind
	Message string
	Args    []any
}

type rule struct {
	Err     val.Error
	Kind    Kind
	Message string
	Params  []string
}

var rules = []rule{
	{Err: val.ErrRequired, Kind: KindRequired, Message: "This field is required"},
	{Err: val.ErrNilOrNotEmpty, Kind: KindRequired, Message: "This field is required"},
	{Err: val.ErrNotNilRequired, Kind: KindRequired, Message: "This field is required"},
	{Err: val.ErrNil, Kind: KindInvalid, Message: "This field must be empty"},
	{Err: val.ErrEmpty, Kind: KindInvalid, Message: "This field must be empty"},
	{Err: val.ErrLengthEmptyRequired, Kind: KindLength, Message: "This field must be empty"},
	{
		Err:     val.ErrLengthTooLong,
		Kind:    KindLength,
		Message: "Must be no more than {0} characters",
		Params:  []string{"max"},
	},
	{
		Err:     val.ErrLengthTooShort,
		Kind:    KindLength,
		Message: "Must be at least {0} characters",
		Params:  []string{"min"},
	},
	{
		Err:     val.ErrLengthInvalid,
		Kind:    KindLength,
		Message: "Must be exactly {0} characters",
		Params:  []string{"min"},
	},
	{
		Err:     val.ErrLengthOutOfRange,
		Kind:    KindLength,
		Message: "Must be between {0} and {1} characters",
		Params:  []string{"min", "max"},
	},
	{
		Err:     val.ErrMinGreaterEqualThanRequired,
		Kind:    KindInvalid,
		Message: "Must be no less than {0}",
		Params:  []string{"threshold"},
	},
	{
		Err:     val.ErrMinGreaterThanRequired,
		Kind:    KindInvalid,
		Message: "Must be greater than {0}",
		Params:  []string{"threshold"},
	},
	{
		Err:     val.ErrMaxLessEqualThanRequired,
		Kind:    KindInvalid,
		Message: "Must be no greater than {0}",
		Params:  []string{"threshold"},
	},
	{
		Err:     val.ErrMaxLessThanRequired,
		Kind:    KindInvalid,
		Message: "Must be less than {0}",
		Params:  []string{"threshold"},
	},
	{
		Err:     val.ErrMultipleOfInvalid,
		Kind:    KindInvalid,
		Message: "Must be a multiple of {0}",
		Params:  []string{"base"},
	},
	{Err: val.ErrInInvalid, Kind: KindInvalid, Message: "Must be one of the allowed values"},
	{Err: val.ErrNotInInvalid, Kind: KindInvalid, Message: "This value is not allowed"},
	{Err: val.ErrMatchInvalid, Kind: KindFormat, Message: "Must be in a valid format"},
	{Err: val.ErrDateInvalid, Kind: KindFormat, Message: "Must be a valid date"},
	{Err: val.ErrDateOutOfRange, Kind: KindInvalid, Message: "The date is out of range"},
	{Err: is.ErrEmail, Kind: KindFormat, Message: "Must be a valid email address"},
	{Err: is.ErrURL, Kind: KindFormat, Message: "Must be a valid URL"},
	{Err: is.ErrHexColor, Kind: KindFormat, Message: "Must be a valid hexadecimal color code"},
	{Err: is.ErrE164, Kind: KindFormat, Message: "Must be a valid phone number in E.164 format"},
	{
		Err:     is.ErrPrintableASCII,
		Kind:    KindFormat,
		Message: "Must contain printable ASCII characters only",
	},
	{Err: is.ErrIP, Kind: KindFormat, Message: "Must be a valid IP address"},
	{Err: is.ErrHost, Kind: KindFormat, Message: "Must be a valid IP address or DNS name"},
	{Err: is.ErrDigit, Kind: KindFormat, Message: "Must contain digits only"},
	{Err: is.ErrUUID, Kind: KindFormat, Message: "Must be a valid UUID"},
	{Err: is.ErrInt, Kind: KindFormat, Message: "Must be a whole number"},
	{Err: is.ErrFloat, Kind: KindFormat, Message: "Must be a number"},
}

var byCode = func() map[string]rule {
	index := make(map[string]rule, len(rules))
	for _, r := range rules {
		index[r.Err.Code()] = r
	}
	return index
}()

const formatCodePrefix = "validation_is_"

var templateParam = regexp.MustCompile(`\{\{\s*\.(\w+)\s*\}\}`)

func Flatten(err error) ([]Leaf, error) {
	if err == nil {
		return nil, nil
	}

	var leaves []Leaf
	internal := walk(nil, err, &leaves)

	return leaves, internal
}

func walk(prefix fieldpath.Path, err error, leaves *[]Leaf) error {
	if internal, ok := errors.AsType[val.InternalError](err); ok && internal.InternalError() != nil {
		return internal.InternalError()
	}

	nested, ok := err.(val.Errors)
	if !ok {
		*leaves = append(*leaves, leafOf(prefix, err))
		return nil
	}

	var internal error
	for _, key := range slices.Sorted(maps.Keys(nested)) {
		child := nested[key]
		if child == nil {
			continue
		}
		if failure := walk(prefix.Append(segmentOf(key)), child, leaves); failure != nil {
			internal = errors.Join(internal, failure)
		}
	}

	return internal
}

func segmentOf(key string) fieldpath.Segment {
	if path := fieldpath.Parse(key); len(path) == 1 {
		return path[0]
	}

	return fieldpath.Key(key)
}

func leafOf(path fieldpath.Path, err error) Leaf {
	described, ok := errors.AsType[val.Error](err)
	if !ok {
		return Leaf{Path: path, Kind: KindInvalid, Message: err.Error()}
	}

	code := described.Code()
	template := described.Message()
	known, isKnown := byCode[code]
	kind := kindOf(code, known, isKnown)
	if isKnown && template == known.Err.Message() {
		return Leaf{
			Path:    path,
			Kind:    kind,
			Message: known.Message,
			Args:    argsOf(described.Params(), known.Params),
		}
	}

	message, args := positional(template, described.Params())

	return Leaf{Path: path, Kind: kind, Message: message, Args: args}
}

func kindOf(code string, known rule, isKnown bool) Kind {
	switch {
	case isKnown:
		return known.Kind
	case strings.HasPrefix(code, formatCodePrefix):
		return KindFormat
	default:
		return KindInvalid
	}
}

func argsOf(params map[string]any, names []string) []any {
	if len(names) == 0 {
		return nil
	}

	args := make([]any, len(names))
	for i, name := range names {
		args[i] = params[name]
	}

	return args
}

func positional(template string, params map[string]any) (string, []any) {
	if !strings.Contains(template, "{{") {
		return template, nil
	}

	var args []any
	order := make(map[string]int)
	message := templateParam.ReplaceAllStringFunc(template, func(match string) string {
		name := templateParam.FindStringSubmatch(match)[1]
		position, seen := order[name]
		if !seen {
			position = len(args)
			order[name] = position
			args = append(args, params[name])
		}
		return "{" + strconv.Itoa(position) + "}"
	})

	return message, args
}
