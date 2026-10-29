package modeladaptertest

import (
	"fmt"
	"strings"
)

type LiveBindingArm struct {
	Name    string
	Err     error
	Status  int
	Dropped int
}

func (a LiveBindingArm) String() string {
	switch {
	case a.Err == nil:
		return fmt.Sprintf("%s: 200, %d thinking block(s) dropped", a.Name, a.Dropped)
	case a.Status != 0:
		return fmt.Sprintf("%s: refused with %d: %s", a.Name, a.Status, firstLine(a.Err.Error()))
	default:
		return fmt.Sprintf("%s: error %s", a.Name, firstLine(a.Err.Error()))
	}
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")

	return line
}
