package tenderescalation

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
)

// escalationMode is the kind of tender an escalation policy follows an
// exhausted one with.
func escalationMode(action dispatchcontrol.TenderExhaustedAction) (tender.Mode, bool) {
	switch action {
	case dispatchcontrol.TenderExhaustedActionRebroadcast:
		return tender.ModeSpotBroadcast, true
	case dispatchcontrol.TenderExhaustedActionRewaterfall:
		return tender.ModeSpotSequential, true
	case dispatchcontrol.TenderExhaustedActionNotify:
		return "", false
	default:
		return "", false
	}
}
