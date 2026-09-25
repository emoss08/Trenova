package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

var awaitingPreviewer = map[string]struct{}{
	"add_shipment_comment":         {},
	"approve_detention":            {},
	"cancel_shipment":              {},
	"cancel_worker_pto":            {},
	"correct_charge_code":          {},
	"email_customer":               {},
	"match_bank_receipt":           {},
	"notify_driver":                {},
	"post_customer_payment":        {},
	"reject_worker_pto":            {},
	"reply_to_inbound_message":     {},
	"request_credential_renewal":   {},
	"request_missing_docs":         {},
	"resolve_service_failure":      {},
	"schedule_report":              {},
	"send_detention_notice":        {},
	"tender_move_to_carriers":      {},
	"tender_move_to_routing_guide": {},
	"update_shipment":              {},
	"waive_detention":              {},
}

func TestEveryActionToolPreviewsWhatItWouldDo(t *testing.T) {
	t.Parallel()

	registeredNames := make(map[string]struct{})
	for _, tool := range buildRegistered(t).Actions {
		name := tool.Name()
		registeredNames[name] = struct{}{}
		if tool.Policy().Kind != agent.ToolKindAction {
			continue
		}

		_, previews := tool.(serviceports.ToolPreviewer)
		_, waiting := awaitingPreviewer[name]
		switch {
		case waiting && previews:
			t.Errorf("%s now previews itself; take it off awaitingPreviewer", name)
		case !waiting && !previews:
			t.Errorf(
				"%s is an action tool without a previewer; implement serviceports.ToolPreviewer",
				name,
			)
		}
	}

	for name := range awaitingPreviewer {
		assert.Contains(t, registeredNames, name, "awaitingPreviewer names a tool nobody registers")
	}
}
