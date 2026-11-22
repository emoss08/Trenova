package instrument

import (
	"encoding/hex"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

const (
	callerPackage = "github.com/emoss08/trenova/shared/redis"
	callerDepth   = 16
	statementArgs = 1
)

func (i *Instrument) callerAttrs(attrs []attribute.KeyValue) []attribute.KeyValue {
	var pcs [callerDepth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var frame runtime.Frame
	for {
		f, more := frames.Next()
		frame = f
		if !inPackage(f.Function) || !more {
			break
		}
	}

	fn := frame.Function
	if idx := strings.LastIndexByte(fn, '/'); idx != -1 {
		fn = fn[idx+1:]
	}

	return append(attrs,
		attribute.String("code.function", fn),
		attribute.String("code.filepath", frame.File),
		attribute.Int("code.lineno", frame.Line),
	)
}

func inPackage(fn string) bool {
	rest, ok := strings.CutPrefix(fn, callerPackage)
	return ok && rest != "" && (rest[0] == '.' || rest[0] == '/')
}

func appendCmd(b []byte, args []any) []byte {
	for idx, arg := range args {
		if idx > 0 {
			b = append(b, ' ')
		}
		if idx > statementArgs {
			b = append(b, '?')
			continue
		}
		b = appendArg(b, arg)
	}
	return b
}

func appendArg(b []byte, v any) []byte {
	switch v := v.(type) {
	case nil:
		return append(b, "<nil>"...)
	case string:
		return appendText(b, v)
	case []byte:
		return appendText(b, string(v))
	case int:
		return strconv.AppendInt(b, int64(v), 10)
	case int32:
		return strconv.AppendInt(b, int64(v), 10)
	case int64:
		return strconv.AppendInt(b, v, 10)
	case uint64:
		return strconv.AppendUint(b, v, 10)
	case float64:
		return strconv.AppendFloat(b, v, 'f', -1, 64)
	case bool:
		return strconv.AppendBool(b, v)
	case time.Time:
		return v.AppendFormat(b, time.RFC3339Nano)
	default:
		return append(b, fmt.Sprint(v)...)
	}
}

func appendText(b []byte, s string) []byte {
	for idx := range len(s) {
		if s[idx] < 0x20 || s[idx] > 0x7e {
			return append(b, hex.EncodeToString([]byte(s))...)
		}
	}
	return append(b, s...)
}
