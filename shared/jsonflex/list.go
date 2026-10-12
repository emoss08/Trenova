package jsonflex

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
)

type StringList []string

func (l *StringList) UnmarshalJSON(data []byte) error {
	switch KindOf(data) {
	case KindNull:
		*l = nil
		return nil
	case KindArray:
		items, err := DecodeArray(data)
		if err != nil {
			return err
		}
		out := make([]string, 0, len(items))
		for _, item := range items {
			if KindOf(item) == KindNull {
				continue
			}
			text, _, ok := scalarText(item)
			if !ok {
				return fmt.Errorf("%w: expected a list of strings", ErrUnsupported)
			}
			if text = strings.TrimSpace(text); text != "" {
				out = append(out, text)
			}
		}
		*l = out
		return nil
	default:
		text, _, ok := scalarText(data)
		if !ok {
			return fmt.Errorf("%w: expected a string or a list of strings", ErrUnsupported)
		}
		if text = strings.TrimSpace(text); text == "" {
			*l = nil
			return nil
		}
		*l = StringList{text}
		return nil
	}
}

func (l StringList) MarshalJSON() ([]byte, error) {
	return sonic.Marshal([]string(l))
}

func (l StringList) First() string {
	if len(l) == 0 {
		return ""
	}
	return l[0]
}
