//go:build devlog

package prettylog

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	entryTimeLayout = "15:04:05.000"
	stacktraceKey   = "stacktrace"

	ansiReset   = "\x1b[0m"
	ansiBold    = "1"
	ansiFaint   = "2"
	ansiRed     = "31"
	ansiGreen   = "32"
	ansiYellow  = "33"
	ansiBlue    = "34"
	ansiMagenta = "35"
	ansiCyan    = "36"
	ansiBoldRed = "1;31"
)

var (
	pool       = buffer.NewPool()
	nestedJSON = sonic.Config{SortMapKeys: true}.Froze()
)

type Options struct {
	Color bool
}

type block struct {
	key   string
	value string
}

type encoder struct {
	color  bool
	prefix string
	fields *buffer.Buffer
	blocks []block
}

var _ zapcore.Encoder = (*encoder)(nil)

func NewEncoder(opts Options) zapcore.Encoder {
	return &encoder{color: opts.Color, fields: pool.Get()}
}

func (e *encoder) Clone() zapcore.Encoder {
	return e.clone()
}

func (e *encoder) clone() *encoder {
	c := &encoder{
		color:  e.color,
		prefix: e.prefix,
		fields: pool.Get(),
		blocks: slices.Clone(e.blocks),
	}
	c.fields.AppendBytes(e.fields.Bytes())
	return c
}

func (e *encoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := e.clone()
	defer final.fields.Free()

	for _, field := range fields {
		field.AddTo(final)
	}

	line := pool.Get()
	e.paint(line, ansiFaint, ent.Time.Format(entryTimeLayout))

	label, code := levelStyle(ent.Level)
	line.AppendByte(' ')
	e.paint(line, code, label)

	if ent.LoggerName != "" {
		line.AppendByte(' ')
		e.paint(line, ansiBlue, ent.LoggerName)
	}

	if ent.Caller.Defined {
		line.AppendByte(' ')
		e.paint(line, ansiFaint, ent.Caller.TrimmedPath()+" >")
	}

	line.AppendByte(' ')
	e.paint(line, ansiBold, ent.Message)

	if final.fields.Len() > 0 {
		line.AppendByte(' ')
		line.AppendBytes(final.fields.Bytes())
	}
	line.AppendByte('\n')

	for _, b := range final.blocks {
		e.appendBlock(line, b.key, b.value)
	}
	if ent.Stack != "" {
		e.appendBlock(line, stacktraceKey, ent.Stack)
	}

	return line, nil
}

func (e *encoder) AddArray(key string, v zapcore.ArrayMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	if err := m.AddArray(key, v); err != nil {
		return err
	}
	return e.addJSON(key, readable(m.Fields[key]))
}

func (e *encoder) AddObject(key string, v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	if err := v.MarshalLogObject(m); err != nil {
		return err
	}
	return e.addJSON(key, readable(m.Fields))
}

func (e *encoder) AddReflected(key string, v any) error {
	return e.addJSON(key, v)
}

func (e *encoder) AddBinary(key string, v []byte) {
	e.addRaw(key, base64.StdEncoding.EncodeToString(v))
}

func (e *encoder) AddByteString(key string, v []byte) {
	e.AddString(key, string(v))
}

func (e *encoder) AddBool(key string, v bool) {
	code := e.beginField(key)
	e.fields.AppendBool(v)
	e.endField(code)
}

func (e *encoder) AddComplex128(key string, v complex128) {
	e.addRaw(key, strconv.FormatComplex(v, 'g', -1, 128))
}

func (e *encoder) AddComplex64(key string, v complex64) {
	e.addRaw(key, strconv.FormatComplex(complex128(v), 'g', -1, 64))
}

func (e *encoder) AddDuration(key string, v time.Duration) {
	e.addRaw(key, v.String())
}

func (e *encoder) AddFloat64(key string, v float64) {
	code := e.beginField(key)
	e.fields.AppendFloat(v, 64)
	e.endField(code)
}

func (e *encoder) AddFloat32(key string, v float32) {
	code := e.beginField(key)
	e.fields.AppendFloat(float64(v), 32)
	e.endField(code)
}

func (e *encoder) AddInt(key string, v int)     { e.AddInt64(key, int64(v)) }
func (e *encoder) AddInt32(key string, v int32) { e.AddInt64(key, int64(v)) }
func (e *encoder) AddInt16(key string, v int16) { e.AddInt64(key, int64(v)) }
func (e *encoder) AddInt8(key string, v int8)   { e.AddInt64(key, int64(v)) }

func (e *encoder) AddInt64(key string, v int64) {
	code := e.beginField(key)
	e.fields.AppendInt(v)
	e.endField(code)
}

func (e *encoder) AddUint(key string, v uint)     { e.AddUint64(key, uint64(v)) }
func (e *encoder) AddUint32(key string, v uint32) { e.AddUint64(key, uint64(v)) }
func (e *encoder) AddUint16(key string, v uint16) { e.AddUint64(key, uint64(v)) }
func (e *encoder) AddUint8(key string, v uint8)   { e.AddUint64(key, uint64(v)) }

func (e *encoder) AddUint64(key string, v uint64) {
	code := e.beginField(key)
	e.fields.AppendUint(v)
	e.endField(code)
}

func (e *encoder) AddUintptr(key string, v uintptr) {
	e.addRaw(key, "0x"+strconv.FormatUint(uint64(v), 16))
}

func (e *encoder) AddString(key, v string) {
	if strings.ContainsRune(v, '\n') {
		e.blocks = append(e.blocks, block{key: e.prefix + key, value: v})
		return
	}

	code := e.beginField(key)
	if needsQuotes(v) {
		e.fields.AppendString(strconv.Quote(v))
	} else {
		e.fields.AppendString(v)
	}
	e.endField(code)
}

func (e *encoder) AddTime(key string, v time.Time) {
	e.addRaw(key, v.Format(time.RFC3339Nano))
}

func (e *encoder) OpenNamespace(key string) {
	e.prefix += key + "."
}

func (e *encoder) addRaw(key, v string) {
	code := e.beginField(key)
	e.fields.AppendString(v)
	e.endField(code)
}

func (e *encoder) addJSON(key string, v any) error {
	data, err := nestedJSON.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode %s: %w", key, err)
	}

	code := e.beginField(key)
	e.fields.AppendBytes(data)
	e.endField(code)
	return nil
}

func (e *encoder) beginField(key string) string {
	if e.fields.Len() > 0 {
		e.fields.AppendByte(' ')
	}
	e.paint(e.fields, ansiCyan, e.prefix+key)
	e.paint(e.fields, ansiFaint, "=")

	code := valueStyle(key)
	e.style(e.fields, code)
	return code
}

func (e *encoder) endField(code string) {
	e.reset(e.fields, code)
}

func (e *encoder) appendBlock(b *buffer.Buffer, key, value string) {
	b.AppendString("  ")
	e.paint(b, ansiCyan, key+":")
	b.AppendByte('\n')

	for line := range strings.SplitSeq(strings.TrimRight(value, "\n"), "\n") {
		b.AppendString("    ")
		e.paint(b, ansiFaint, line)
		b.AppendByte('\n')
	}
}

func (e *encoder) paint(b *buffer.Buffer, code, s string) {
	e.style(b, code)
	b.AppendString(s)
	e.reset(b, code)
}

func (e *encoder) style(b *buffer.Buffer, code string) {
	if !e.color || code == "" {
		return
	}
	b.AppendString("\x1b[")
	b.AppendString(code)
	b.AppendByte('m')
}

func (e *encoder) reset(b *buffer.Buffer, code string) {
	if !e.color || code == "" {
		return
	}
	b.AppendString(ansiReset)
}

func readable(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for k, item := range v {
			v[k] = readable(item)
		}
		return v
	case []any:
		for i, item := range v {
			v[i] = readable(item)
		}
		return v
	case time.Duration:
		return v.String()
	case complex128:
		return strconv.FormatComplex(v, 'g', -1, 128)
	case complex64:
		return strconv.FormatComplex(complex128(v), 'g', -1, 64)
	default:
		return v
	}
}

func levelStyle(level zapcore.Level) (label, code string) {
	switch level {
	case zapcore.DebugLevel:
		return "DBG", ansiMagenta
	case zapcore.InfoLevel:
		return "INF", ansiGreen
	case zapcore.WarnLevel:
		return "WRN", ansiYellow
	case zapcore.ErrorLevel:
		return "ERR", ansiRed
	case zapcore.DPanicLevel:
		return "DPN", ansiBoldRed
	case zapcore.PanicLevel:
		return "PNC", ansiBoldRed
	case zapcore.FatalLevel:
		return "FTL", ansiBoldRed
	default:
		return level.CapitalString(), ""
	}
}

func valueStyle(key string) string {
	if key == "error" || key == "err" || strings.HasSuffix(key, "Error") ||
		strings.HasSuffix(key, "_error") {
		return ansiRed
	}
	return ""
}

func needsQuotes(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r == utf8.RuneError || r <= ' ' || r == '"' || r == '=' || r == '\\' ||
			!unicode.IsPrint(r) {
			return true
		}
	}
	return false
}
