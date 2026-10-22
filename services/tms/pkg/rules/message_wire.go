package rules

import (
	"fmt"
	"strconv"
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
