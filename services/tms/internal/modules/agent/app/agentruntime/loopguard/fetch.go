package loopguard

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

const getToolPrefix = "get_"

func IsFetch(name string) bool {
	return strings.HasPrefix(agenttoolports.ReadingOrder(name), getToolPrefix)
}
