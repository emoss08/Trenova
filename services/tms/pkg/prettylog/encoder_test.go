//go:build devlog

package prettylog_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/prettylog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var at = time.Date(2026, 11, 3, 14, 5, 9, 123_000_000, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time                         { return at }
func (fixedClock) NewTicker(d time.Duration) *time.Ticker { return time.NewTicker(d) }

type detailedError struct{}

func (detailedError) Error() string { return "load failed" }

func (e detailedError) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('+') {
		_, _ = io.WriteString(s, "load failed\n  at loader.go:10")
		return
	}
	_, _ = io.WriteString(s, e.Error())
}

func newLogger(color bool) (*zap.Logger, *bytes.Buffer) {
	var out bytes.Buffer
	core := zapcore.NewCore(
		prettylog.NewEncoder(prettylog.Options{Color: color}),
		zapcore.AddSync(&out),
		zapcore.DebugLevel,
	)
	return zap.New(core, zap.WithClock(fixedClock{})), &out
}

func encode(t *testing.T, ent zapcore.Entry, fields ...zapcore.Field) string {
	t.Helper()

	buf, err := prettylog.NewEncoder(prettylog.Options{}).EncodeEntry(ent, fields)
	require.NoError(t, err)
	defer buf.Free()

	return buf.String()
}

func TestEncoder_WritesTimeLevelMessageThenFieldsInTheOrderAdded(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(false)

	logger.Info("Starting application",
		zap.String("name", "trenova"),
		zap.Int("attempt", 3),
		zap.Duration("took", 1500*time.Millisecond),
		zap.Bool("ok", true),
		zap.Float64("ratio", 0.25),
	)

	assert.Equal(
		t,
		"14:05:09.123 INF Starting application name=trenova attempt=3 took=1.5s ok=true ratio=0.25\n",
		out.String(),
	)
}

func TestEncoder_LoggerNameAndCallerPrecedeTheMessage(t *testing.T) {
	t.Parallel()

	got := encode(t, zapcore.Entry{
		Level:      zapcore.InfoLevel,
		Time:       at,
		LoggerName: "worker",
		Caller: zapcore.EntryCaller{
			Defined: true,
			File:    "/src/trenova/internal/infrastructure/config/provider.go",
			Line:    180,
		},
		Message: "ready",
	})

	assert.Equal(t, "14:05:09.123 INF worker config/provider.go:180 > ready\n", got)
}

func TestEncoder_ContextFieldsLeadAndDoNotLeakBetweenChildLoggers(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(false)
	base := logger.With(zap.String("org_id", "org_1"), zap.String("note", "a\nb"))

	base.With(zap.String("user_id", "usr_1"), zap.String("detail", "c\nd")).
		Info("one", zap.String("step", "a"))
	base.Info("two")

	assert.Equal(t,
		"14:05:09.123 INF one org_id=org_1 user_id=usr_1 step=a\n"+
			"  note:\n    a\n    b\n"+
			"  detail:\n    c\n    d\n"+
			"14:05:09.123 INF two org_id=org_1\n"+
			"  note:\n    a\n    b\n",
		out.String(),
	)
}

func TestEncoder_NamespacePrefixesTheKeysThatFollowIt(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(false)

	logger.Info("request",
		zap.String("route", "/v1/shipments"),
		zap.Namespace("http"),
		zap.Int("status", 200),
		zap.String("method", "GET"),
	)

	assert.Equal(t,
		"14:05:09.123 INF request route=/v1/shipments http.status=200 http.method=GET\n",
		out.String(),
	)
}

func TestEncoder_QuotesValuesThatWouldBreakKeyValueReading(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(false)

	logger.Info("values",
		zap.String("empty", ""),
		zap.String("spaced", "two words"),
		zap.String("pair", "a=b"),
		zap.String("quoted", `say "hi"`),
		zap.String("tab", "a\tb"),
		zap.String("escape", "\x1b[31mred"),
		zap.String("plain", "über-load_1"),
	)

	assert.Equal(t,
		`14:05:09.123 INF values empty="" spaced="two words" pair="a=b" quoted="say \"hi\"" `+
			`tab="a\tb" escape="\x1b[31mred" plain=über-load_1`+"\n",
		out.String(),
	)
}

func TestEncoder_MultiLineValuesAndTheStacktracePrintAsIndentedBlocks(t *testing.T) {
	t.Parallel()

	got := encode(t,
		zapcore.Entry{
			Level:   zapcore.ErrorLevel,
			Time:    at,
			Message: "failed",
			Stack:   "goroutine 1 [running]:\nmain.main()\n\t/src/main.go:12\n",
		},
		zap.String("query", "SELECT 1\nFROM shipments"),
		zap.String("table", "shipments"),
	)

	assert.Equal(t,
		"14:05:09.123 ERR failed table=shipments\n"+
			"  query:\n    SELECT 1\n    FROM shipments\n"+
			"  stacktrace:\n    goroutine 1 [running]:\n    main.main()\n    \t/src/main.go:12\n",
		got,
	)
}

func TestEncoder_VerboseErrorKeepsTheMessageInlineAndTheDetailBelow(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(false)

	logger.Error("failed", zap.Error(detailedError{}))

	assert.Equal(t,
		"14:05:09.123 ERR failed error=\"load failed\"\n"+
			"  errorVerbose:\n    load failed\n      at loader.go:10\n",
		out.String(),
	)
}

func TestEncoder_NestedValuesRenderAsCompactJSONWithSortedKeys(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(false)

	logger.Info("nested",
		zap.Strings("ids", []string{"a", "b"}),
		zap.Any("meta", map[string]any{"b": 1, "a": "x"}),
		zap.Object("load", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("pro", "P-1")
			enc.AddInt("stops", 2)
			return nil
		})),
	)

	assert.Equal(
		t,
		`14:05:09.123 INF nested ids=["a","b"] meta={"a":"x","b":1} load={"pro":"P-1","stops":2}`+"\n",
		out.String(),
	)
}

func TestEncoder_NestedDurationsAndComplexNumbersRenderAsReadableStrings(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(false)

	logger.Info("retry",
		zap.Durations("waits", []time.Duration{250 * time.Millisecond, 2 * time.Second}),
		zap.Object("policy", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddDuration("backoff", 1500*time.Millisecond)
			enc.AddComplex128("z", 1+2i)
			return enc.AddArray(
				"steps",
				zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
					arr.AppendDuration(time.Minute)
					return nil
				}),
			)
		})),
	)

	assert.Equal(
		t,
		`14:05:09.123 INF retry waits=["250ms","2s"] `+
			`policy={"backoff":"1.5s","steps":["1m0s"],"z":"(1+2i)"}`+"\n",
		out.String(),
	)
}

func TestEncoder_LevelsUseThreeLetterLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level zapcore.Level
		label string
	}{
		{zapcore.DebugLevel, "DBG"},
		{zapcore.InfoLevel, "INF"},
		{zapcore.WarnLevel, "WRN"},
		{zapcore.ErrorLevel, "ERR"},
		{zapcore.DPanicLevel, "DPN"},
		{zapcore.PanicLevel, "PNC"},
		{zapcore.FatalLevel, "FTL"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()

			got := encode(t, zapcore.Entry{Level: tt.level, Time: at, Message: "m"})

			assert.Equal(t, "14:05:09.123 "+tt.label+" m\n", got)
		})
	}
}

func TestEncoder_ColorPaintsTimeLevelMessageKeysAndErrorValues(t *testing.T) {
	t.Parallel()

	logger, out := newLogger(true)

	logger.Error("failed", zap.Error(errors.New("boom")), zap.Int("n", 1))

	assert.Equal(t,
		"\x1b[2m14:05:09.123\x1b[0m \x1b[31mERR\x1b[0m \x1b[1mfailed\x1b[0m "+
			"\x1b[36merror\x1b[0m\x1b[2m=\x1b[0m\x1b[31mboom\x1b[0m "+
			"\x1b[36mn\x1b[0m\x1b[2m=\x1b[0m1\n",
		out.String(),
	)
}
