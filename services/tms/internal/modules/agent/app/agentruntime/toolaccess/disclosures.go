package toolaccess

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

// ignoredDisclosures is what a running process does with tool disclosures: it
// keeps nothing. The desk bench replaces it to measure what a turn was shown.
type ignoredDisclosures struct{}

// NewIgnoredDisclosures is the process's tool disclosure observer.
func NewIgnoredDisclosures() agenttoolports.ToolDisclosureObserver { return ignoredDisclosures{} }

func (ignoredDisclosures) Preselected(context.Context, *agenttoolports.ToolDisclosure) {}

func (ignoredDisclosures) Found(context.Context, *agenttoolports.ToolDisclosure) {}
