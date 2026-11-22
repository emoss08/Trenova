package resp

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func read(t *testing.T, wire string) (Value, error) {
	t.Helper()
	return NewReader(strings.NewReader(wire), 0).Read()
}

func TestReaderReadsEveryReplyKind(t *testing.T) {
	cases := []struct {
		name string
		wire string
		want Value
	}{
		{"status", "+OK\r\n", Value{Kind: KindStatus, Str: "OK"}},
		{"error", "-ERR wrong\r\n", Value{Kind: KindError, Str: "ERR wrong"}},
		{"integer", ":42\r\n", Value{Kind: KindInt, Int: 42}},
		{"negative integer", ":-7\r\n", Value{Kind: KindInt, Int: -7}},
		{"min integer", ":-9223372036854775808\r\n", Value{Kind: KindInt, Int: -1 << 63}},
		{"bulk", "$5\r\nhello\r\n", Value{Kind: KindBulk, Str: "hello"}},
		{"binary bulk", "$4\r\na\r\nb\r\n", Value{Kind: KindBulk, Str: "a\r\nb"}},
		{"empty bulk", "$0\r\n\r\n", Value{Kind: KindBulk, Str: ""}},
		{"nil bulk", "$-1\r\n", Value{Kind: KindNil}},
		{"nil array", "*-1\r\n", Value{Kind: KindNil}},
		{"empty array", "*0\r\n", Value{Kind: KindArray, Elems: []Value{}}},
		{
			"nested array",
			"*3\r\n:1\r\n*2\r\n$1\r\na\r\n$-1\r\n-ERR inner\r\n",
			Value{Kind: KindArray, Elems: []Value{
				{Kind: KindInt, Int: 1},
				{Kind: KindArray, Elems: []Value{{Kind: KindBulk, Str: "a"}, {Kind: KindNil}}},
				{Kind: KindError, Str: "ERR inner"},
			}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := read(t, tc.wire)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestReaderReadsConsecutiveReplies(t *testing.T) {
	rd := NewReader(strings.NewReader("+OK\r\n:1\r\n$1\r\nx\r\n"), 0)
	for _, want := range []Value{
		{Kind: KindStatus, Str: "OK"},
		{Kind: KindInt, Int: 1},
		{Kind: KindBulk, Str: "x"},
	} {
		got, err := rd.Read()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := rd.Read()
	assert.ErrorIs(t, err, io.EOF)
}

func TestReaderRejectsMalformedReplies(t *testing.T) {
	cases := map[string]string{
		"unknown type":          "!oops\r\n",
		"missing carriage":      "+OK\n",
		"empty line":            "\r\n",
		"malformed integer":     ":12a\r\n",
		"empty integer":         ":\r\n",
		"bare minus":            ":-\r\n",
		"integer overflow":      ":9223372036854775808\r\n",
		"huge integer":          ":99999999999999999999999\r\n",
		"negative bulk length":  "$-2\r\n",
		"oversized bulk":        "$1073741824\r\n",
		"unterminated bulk":     "$3\r\nabcde\r\n",
		"negative array length": "*-5\r\n",
		"oversized array":       "*99999999\r\n",
	}
	for name, wire := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := read(t, wire)
			assert.ErrorIs(t, err, ErrProtocol)
		})
	}
}

func TestReaderReportsTruncatedRepliesAsUnexpectedEOF(t *testing.T) {
	for _, wire := range []string{"$5\r\nhel", "*2\r\n:1\r\n"} {
		_, err := read(t, wire)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF, wire)
	}
}

func TestReaderRejectsRepliesNestedTooDeeply(t *testing.T) {
	_, err := read(t, strings.Repeat("*1\r\n", maxDepth+1)+":1\r\n")
	assert.ErrorIs(t, err, ErrProtocol)
}

func TestReaderRejectsLinesLongerThanItsBuffer(t *testing.T) {
	_, err := read(t, "+"+strings.Repeat("a", 2*minBufferSize)+"\r\n")
	assert.ErrorIs(t, err, ErrProtocol)
}

type marshaler struct {
	data []byte
	err  error
}

func (m marshaler) MarshalBinary() ([]byte, error) { return m.data, m.err }

func TestAppendCommandEncodesEveryArgumentType(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC)
	got, err := AppendCommand(nil, []any{
		"set", []byte("k"), nil, 1, int8(-2), int16(3), int32(4), int64(5),
		uint(6), uint8(7), uint16(8), uint32(9), uint64(10),
		float32(1.5), 2.25, true, false, at, 3 * time.Millisecond, marshaler{data: []byte("m")},
	})
	require.NoError(t, err)

	want := "*20\r\n$3\r\nset\r\n$1\r\nk\r\n$0\r\n\r\n$1\r\n1\r\n$2\r\n-2\r\n$1\r\n3\r\n" +
		"$1\r\n4\r\n$1\r\n5\r\n$1\r\n6\r\n$1\r\n7\r\n$1\r\n8\r\n$1\r\n9\r\n$2\r\n10\r\n" +
		"$3\r\n1.5\r\n$4\r\n2.25\r\n$1\r\n1\r\n$1\r\n0\r\n" +
		"$30\r\n2026-01-02T03:04:05.000000006Z\r\n$7\r\n3000000\r\n$1\r\nm\r\n"
	assert.Equal(t, want, string(got))
}

func TestAppendCommandLeavesTheBufferUnchangedOnAnUnsupportedArgument(t *testing.T) {
	prefix := []byte("*1\r\n$4\r\nping\r\n")

	got, err := AppendCommand(bytes.Clone(prefix), []any{"set", struct{}{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't marshal struct {}")
	assert.Equal(t, prefix, got)

	failing := errors.New("boom")
	got, err = AppendCommand(bytes.Clone(prefix), []any{"set", marshaler{err: failing}})
	require.ErrorIs(t, err, failing)
	assert.Equal(t, prefix, got)
}

func TestAppendCommandRoundTripsThroughTheReader(t *testing.T) {
	args := []any{"hset", "key", "field with spaces", "line\r\nbreak", ""}
	encoded, err := AppendCommand(nil, args)
	require.NoError(t, err)

	got, err := read(t, string(encoded))
	require.NoError(t, err)
	strs, err := got.Strings()
	require.NoError(t, err)
	assert.Equal(t, []string{"hset", "key", "field with spaces", "line\r\nbreak", ""}, strs)
}

func TestValueConversions(t *testing.T) {
	_, err := Value{Kind: KindNil}.Text()
	require.ErrorIs(t, err, Nil)

	n, err := Value{Kind: KindBulk, Str: "12"}.Integer()
	require.NoError(t, err)
	assert.EqualValues(t, 12, n)

	f, err := Value{Kind: KindInt, Int: 3}.Float()
	require.NoError(t, err)
	assert.InDelta(t, 3.0, f, 0)

	floats, err := Value{Kind: KindArray, Elems: []Value{
		{Kind: KindBulk, Str: "1.5"}, {Kind: KindNil},
	}}.Floats()
	require.NoError(t, err)
	assert.Equal(t, []float64{1.5, 0}, floats)

	_, err = Value{Kind: KindInt}.Array()
	require.Error(t, err)

	assert.Equal(
		t,
		[]any{"a", int64(1), Error("ERR x"), nil},
		Value{Kind: KindArray, Elems: []Value{
			{Kind: KindBulk, Str: "a"}, {Kind: KindInt, Int: 1}, {Kind: KindError, Str: "ERR x"},
			{Kind: KindNil},
		}}.Any(),
	)

	pairs, err := Value{Kind: KindArray, Elems: []Value{
		{Kind: KindBulk, Str: "name"}, {Kind: KindBulk, Str: "g"},
	}}.Pairs()
	require.NoError(t, err)
	assert.Equal(t, "g", pairs["name"].Str)
}
