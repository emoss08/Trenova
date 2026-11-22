//go:build fuzz

package resp

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func FuzzReader(f *testing.F) {
	for _, seed := range []string{
		"+OK\r\n",
		"-ERR wrong\r\n",
		":-12\r\n",
		"$5\r\nhello\r\n",
		"$-1\r\n",
		"*2\r\n$1\r\na\r\n*1\r\n:1\r\n",
		"*-1\r\n",
		"*1\r\n*1\r\n*1\r\n+deep\r\n",
		"$3\r\nab",
		":99999999999999999999\r\n",
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		v, err := NewReader(bytes.NewReader(data), 0).Read()
		if err != nil {
			return
		}
		again, err := NewReader(bytes.NewReader(encodeValue(nil, v)), 0).Read()
		if err != nil {
			t.Fatalf("re-reading an encoded reply failed: %v", err)
		}
		if !equalValues(v, again) {
			t.Fatalf("reply changed across a round trip: %#v != %#v", v, again)
		}
	})
}

func FuzzAppendCommand(f *testing.F) {
	f.Add("set\x00key\x00value")
	f.Add("hset\x00k\x00a\r\nb\x00")
	f.Add("")

	f.Fuzz(func(t *testing.T, joined string) {
		parts := strings.Split(joined, "\x00")
		args := make([]any, len(parts))
		for idx, part := range parts {
			args[idx] = part
		}
		encoded, err := AppendCommand(nil, args)
		if err != nil {
			t.Fatalf("encoding strings failed: %v", err)
		}
		v, err := NewReader(bytes.NewReader(encoded), 0).Read()
		if err != nil {
			t.Fatalf("reading an encoded command failed: %v", err)
		}
		got, err := v.Strings()
		if err != nil {
			t.Fatalf("encoded command is not an array of strings: %v", err)
		}
		if strings.Join(got, "\x00") != joined || len(got) != len(parts) {
			t.Fatalf("command changed across a round trip: %q != %q", got, parts)
		}
	})
}

func encodeValue(dst []byte, v Value) []byte {
	switch v.Kind {
	case KindStatus:
		return append(append(append(dst, '+'), v.Str...), '\r', '\n')
	case KindError:
		return append(append(append(dst, '-'), v.Str...), '\r', '\n')
	case KindInt:
		return append(strconv.AppendInt(append(dst, ':'), v.Int, 10), '\r', '\n')
	case KindBulk:
		return appendBulk(dst, v.Str)
	case KindArray:
		dst = append(strconv.AppendInt(append(dst, '*'), int64(len(v.Elems)), 10), '\r', '\n')
		for _, elem := range v.Elems {
			dst = encodeValue(dst, elem)
		}
		return dst
	default:
		return append(dst, "$-1\r\n"...)
	}
}

func equalValues(a, b Value) bool {
	if a.Kind != b.Kind || a.Str != b.Str || a.Int != b.Int || len(a.Elems) != len(b.Elems) {
		return false
	}
	for idx := range a.Elems {
		if !equalValues(a.Elems[idx], b.Elems[idx]) {
			return false
		}
	}
	return true
}
