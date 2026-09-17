package jsonflex

import (
	"errors"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/stringutils"
)

var (
	ErrNotObject = errors.New("jsonflex: JSON value is not an object")
	ErrNotArray  = errors.New("jsonflex: JSON value is not an array")
)

type Object map[string]sonic.NoCopyRawMessage

func DecodeObject(data []byte) (Object, error) {
	if KindOf(data) != KindObject {
		return nil, ErrNotObject
	}
	obj := make(Object)
	if err := sonic.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	return obj, nil
}

func DecodeArray(data []byte) ([]sonic.NoCopyRawMessage, error) {
	if KindOf(data) != KindArray {
		return nil, ErrNotArray
	}
	var items []sonic.NoCopyRawMessage
	if err := sonic.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("decode JSON array: %w", err)
	}
	return items, nil
}

func (o Object) Has(key string) bool {
	_, ok := o[key]
	return ok
}

func (o Object) Lookup(keys ...string) ([]byte, bool) {
	for _, key := range keys {
		raw, ok := o[key]
		if ok && !IsAbsent(raw) {
			return raw, true
		}
	}
	return nil, false
}

func (o Object) Raw(keys ...string) []byte {
	raw, ok := o.Lookup(keys...)
	if !ok {
		return nil
	}
	return slices.Clone(trimSpace(raw))
}

func (o Object) String(keys ...string) *String {
	for _, key := range keys {
		if value, ok := ParseString(o[key]); ok {
			return value
		}
	}
	return nil
}

func (o Object) Int(keys ...string) *Int {
	for _, key := range keys {
		if value, ok := ParseInt(o[key]); ok {
			return value
		}
	}
	return nil
}

func (o Object) Float(keys ...string) *Float {
	for _, key := range keys {
		if value, ok := ParseFloat(o[key]); ok {
			return value
		}
	}
	return nil
}

func (o Object) Bool(keys ...string) *Bool {
	for _, key := range keys {
		if value, ok := ParseBool(o[key]); ok {
			return value
		}
	}
	return nil
}

func (o Object) Time(keys ...string) *Time {
	for _, key := range keys {
		if value, ok := ParseTime(o[key]); ok {
			return value
		}
	}
	return nil
}

func (o Object) Text(keys ...string) string {
	return o.String(keys...).Value()
}

func (o Object) Object(keys ...string) (Object, bool) {
	for _, key := range keys {
		raw, ok := o[key]
		if !ok || KindOf(raw) != KindObject {
			continue
		}
		nested, err := DecodeObject(raw)
		if err == nil {
			return nested, true
		}
	}
	return nil, false
}

func (o Object) Array(keys ...string) ([]sonic.NoCopyRawMessage, bool) {
	for _, key := range keys {
		raw, ok := o[key]
		if !ok || KindOf(raw) != KindArray {
			continue
		}
		items, err := DecodeArray(raw)
		if err == nil {
			return items, true
		}
	}
	return nil, false
}

func (o Object) Strings(keys ...string) []string {
	for _, key := range keys {
		raw, ok := o[key]
		if !ok {
			continue
		}
		switch KindOf(raw) {
		case KindArray:
			items, err := DecodeArray(raw)
			if err != nil {
				continue
			}
			values := make([]string, 0, len(items))
			for _, item := range items {
				if value, parsed := ParseString(item); parsed {
					values = append(values, value.Value())
				}
			}
			if len(values) > 0 {
				return values
			}
		case KindString:
			if values := stringutils.SplitCSV(o.Text(key)); len(values) > 0 {
				return values
			}
		default:
			continue
		}
	}
	return nil
}

func (o Object) Keys() []string {
	keys := make([]string, 0, len(o))
	for key := range o {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func Unwrap(raw []byte, wrapperKey string) []byte {
	if KindOf(raw) != KindObject {
		return raw
	}
	obj, err := DecodeObject(raw)
	if err != nil {
		return raw
	}
	nested, ok := obj[wrapperKey]
	if !ok || KindOf(nested) != KindObject {
		return raw
	}
	return trimSpace(nested)
}
