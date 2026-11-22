package resp

import (
	"fmt"
	"strconv"
)

//nolint:errname // go-redis names these sentinels; the client exports them as redis.Nil and redis.TxFailedErr
const (
	Nil         = Error("redis: nil")
	TxFailedErr = Error("redis: transaction failed")
)

func (v Value) Unexpected() error {
	return fmt.Errorf("redis: unexpected reply kind %d", v.Kind)
}

func (v Value) Text() (string, error) {
	switch v.Kind {
	case KindStatus, KindBulk:
		return v.Str, nil
	case KindInt:
		return strconv.FormatInt(v.Int, 10), nil
	case KindNil:
		return "", Nil
	case KindError, KindArray:
		return "", v.Unexpected()
	default:
		return "", v.Unexpected()
	}
}

func (v Value) Integer() (int64, error) {
	switch v.Kind {
	case KindInt:
		return v.Int, nil
	case KindStatus, KindBulk:
		return strconv.ParseInt(v.Str, 10, 64)
	case KindNil:
		return 0, Nil
	case KindError, KindArray:
		return 0, v.Unexpected()
	default:
		return 0, v.Unexpected()
	}
}

func (v Value) Float() (float64, error) {
	switch v.Kind {
	case KindStatus, KindBulk:
		return strconv.ParseFloat(v.Str, 64)
	case KindInt:
		return float64(v.Int), nil
	case KindNil:
		return 0, Nil
	case KindError, KindArray:
		return 0, v.Unexpected()
	default:
		return 0, v.Unexpected()
	}
}

func (v Value) Array() ([]Value, error) {
	switch v.Kind {
	case KindArray:
		return v.Elems, nil
	case KindNil:
		return nil, Nil
	case KindStatus, KindError, KindInt, KindBulk:
		return nil, v.Unexpected()
	default:
		return nil, v.Unexpected()
	}
}

func (v Value) Strings() ([]string, error) {
	elems, err := v.Array()
	if err != nil {
		return nil, err
	}
	out := make([]string, len(elems))
	for idx, elem := range elems {
		if elem.Kind == KindNil {
			continue
		}
		if out[idx], err = elem.Text(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (v Value) Floats() ([]float64, error) {
	elems, err := v.Array()
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(elems))
	for idx, elem := range elems {
		if elem.Kind == KindNil {
			continue
		}
		if out[idx], err = elem.Float(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (v Value) Any() any {
	switch v.Kind {
	case KindStatus, KindBulk:
		return v.Str
	case KindInt:
		return v.Int
	case KindError:
		return Error(v.Str)
	case KindArray:
		out := make([]any, len(v.Elems))
		for idx, elem := range v.Elems {
			out[idx] = elem.Any()
		}
		return out
	case KindNil:
		return nil
	default:
		return nil
	}
}

func (v Value) Pairs() (map[string]Value, error) {
	elems, err := v.Array()
	if err != nil {
		return nil, err
	}
	out := make(map[string]Value, len(elems)/2)
	for idx := 0; idx+1 < len(elems); idx += 2 {
		out[elems[idx].Str] = elems[idx+1]
	}
	return out, nil
}
