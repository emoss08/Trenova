package jsonflex

import (
	"github.com/bytedance/sonic"
)

type Kind uint8

const (
	KindInvalid Kind = iota
	KindNull
	KindString
	KindNumber
	KindBool
	KindArray
	KindObject
)

func KindOf(raw []byte) Kind {
	for _, c := range raw {
		switch c {
		case ' ', '\t', '\r', '\n':
			continue
		case '"':
			return KindString
		case 'n':
			return KindNull
		case 't', 'f':
			return KindBool
		case '[':
			return KindArray
		case '{':
			return KindObject
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return KindNumber
		default:
			return KindInvalid
		}
	}
	return KindInvalid
}

func IsAbsent(raw []byte) bool {
	kind := KindOf(raw)
	return kind == KindNull || kind == KindInvalid
}

func scalarText(raw []byte) (string, Kind, bool) {
	kind := KindOf(raw)
	switch kind {
	case KindString:
		var value string
		if err := sonic.Unmarshal(raw, &value); err != nil {
			return "", kind, false
		}
		return value, kind, true
	case KindNumber, KindBool:
		return string(trimSpace(raw)), kind, true
	default:
		return "", kind, false
	}
}

func trimSpace(raw []byte) []byte {
	start, end := 0, len(raw)
	for start < end && isSpace(raw[start]) {
		start++
	}
	for end > start && isSpace(raw[end-1]) {
		end--
	}
	return raw[start:end]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}
