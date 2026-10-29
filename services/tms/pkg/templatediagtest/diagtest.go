package templatediagtest

import (
	"github.com/emoss08/trenova/pkg/templatediag"
)

func HasCode(diags templatediag.Diagnostics, code string) bool {
	for i := range diags {
		if diags[i].Code == code {
			return true
		}
	}
	return false
}

func Codes(diags templatediag.Diagnostics) []string {
	out := make([]string, 0, len(diags))
	for i := range diags {
		out = append(out, diags[i].Code)
	}
	return out
}
