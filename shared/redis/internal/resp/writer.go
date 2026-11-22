package resp

import (
	"encoding"
	"fmt"
	"strconv"
	"time"
)

func AppendCommand(dst []byte, args []any) ([]byte, error) {
	start := len(dst)
	dst = append(dst, '*')
	dst = strconv.AppendInt(dst, int64(len(args)), 10)
	dst = append(dst, '\r', '\n')
	for _, arg := range args {
		var err error
		if dst, err = appendArg(dst, arg); err != nil {
			return dst[:start], err
		}
	}

	return dst, nil
}

func appendArg(dst []byte, arg any) ([]byte, error) {
	var scratch [64]byte
	switch v := arg.(type) {
	case string:
		return appendBulk(dst, v), nil
	case []byte:
		return appendBulkBytes(dst, v), nil
	case nil:
		return appendBulk(dst, ""), nil
	case bool:
		if v {
			return appendBulk(dst, "1"), nil
		}
		return appendBulk(dst, "0"), nil
	case time.Time:
		return appendBulkBytes(dst, v.AppendFormat(scratch[:0], time.RFC3339Nano)), nil
	case encoding.BinaryMarshaler:
		b, err := v.MarshalBinary()
		if err != nil {
			return dst, err
		}
		return appendBulkBytes(dst, b), nil
	}
	if num, ok := appendNumber(scratch[:0], arg); ok {
		return appendBulkBytes(dst, num), nil
	}
	return dst, fmt.Errorf("redis: can't marshal %T (implement encoding.BinaryMarshaler)", arg)
}

func appendNumber(dst []byte, arg any) ([]byte, bool) {
	switch v := arg.(type) {
	case int:
		return strconv.AppendInt(dst, int64(v), 10), true
	case int8:
		return strconv.AppendInt(dst, int64(v), 10), true
	case int16:
		return strconv.AppendInt(dst, int64(v), 10), true
	case int32:
		return strconv.AppendInt(dst, int64(v), 10), true
	case int64:
		return strconv.AppendInt(dst, v, 10), true
	case time.Duration:
		return strconv.AppendInt(dst, int64(v), 10), true
	case uint:
		return strconv.AppendUint(dst, uint64(v), 10), true
	case uint8:
		return strconv.AppendUint(dst, uint64(v), 10), true
	case uint16:
		return strconv.AppendUint(dst, uint64(v), 10), true
	case uint32:
		return strconv.AppendUint(dst, uint64(v), 10), true
	case uint64:
		return strconv.AppendUint(dst, v, 10), true
	case float32:
		return strconv.AppendFloat(dst, float64(v), 'f', -1, 64), true
	case float64:
		return strconv.AppendFloat(dst, v, 'f', -1, 64), true
	default:
		return dst, false
	}
}

func appendBulk(dst []byte, s string) []byte {
	dst = append(dst, '$')
	dst = strconv.AppendInt(dst, int64(len(s)), 10)
	dst = append(dst, '\r', '\n')
	dst = append(dst, s...)
	return append(dst, '\r', '\n')
}

func appendBulkBytes(dst, b []byte) []byte {
	dst = append(dst, '$')
	dst = strconv.AppendInt(dst, int64(len(b)), 10)
	dst = append(dst, '\r', '\n')
	dst = append(dst, b...)
	return append(dst, '\r', '\n')
}
