package stringutils

import (
	"fmt"
	"strings"
)

func JoinMethods[T fmt.Stringer](methods []T) string {
	strs := make([]string, len(methods))
	for i, m := range methods {
		strs[i] = m.String()
	}
	return strings.Join(strs, ", ")
}
