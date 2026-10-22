package rules

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

func (m Message) MarshalJSON() ([]byte, error) {
	return appendMessage(make([]byte, 0, len(m.Template)+32), m)
}

func appendMessage(dst []byte, message Message) ([]byte, error) {
	dst = append(dst, `{"message":`...)
	dst = appendString(dst, message.Template)
	if len(message.Args) == 0 {
		return append(dst, '}'), nil
	}
	dst = append(dst, `,"args":`...)
	dst, err := appendMessageArgs(dst, message.Args)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func appendMessageArgs(dst []byte, args []any) ([]byte, error) {
	dst = append(dst, '[')
	for index, arg := range args {
		if index > 0 {
			dst = append(dst, ',')
		}
		switch typed := arg.(type) {
		case string:
			dst = appendString(dst, typed)
		case int64:
			dst = strconv.AppendInt(dst, typed, 10)
		case int:
			dst = strconv.AppendInt(dst, int64(typed), 10)
		case Message:
			var err error
			if dst, err = appendMessage(dst, typed); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%w: %T", errMessageArg, arg)
		}
	}
	return append(dst, ']'), nil
}

const hexDigits = "0123456789abcdef"

func appendString(dst []byte, text string) []byte {
	dst = append(dst, '"')
	start := 0
	for index := 0; index < len(text); {
		c := text[index]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' {
				index++
				continue
			}
			dst = append(dst, text[start:index]...)
			switch c {
			case '"', '\\':
				dst = append(dst, '\\', c)
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			}
			index++
			start = index
			continue
		}
		r, size := utf8.DecodeRuneInString(text[index:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, text[start:index]...)
			dst = append(dst, `�`...)
			index += size
			start = index
			continue
		}
		index += size
	}
	dst = append(dst, text[start:]...)
	return append(dst, '"')
}
