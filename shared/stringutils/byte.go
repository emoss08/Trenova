package stringutils

import (
	"strconv"

	"github.com/bytedance/sonic"
)

func DecodeByteString(value []byte) string {
	if len(value) == 0 {
		return ""
	}

	raw := string(value)
	if unquoted, err := strconv.Unquote(raw); err == nil {
		return unquoted
	}

	var decoded string
	if err := sonic.Unmarshal(value, &decoded); err == nil {
		return decoded
	}

	return raw
}
