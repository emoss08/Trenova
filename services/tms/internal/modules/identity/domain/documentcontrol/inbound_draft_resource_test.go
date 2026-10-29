package documentcontrol_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
A forwarded tender has to produce a shipment draft on a default-configured
organization, or the whole point of the inbox is lost.

canGenerateShipmentDraft checks the document's resource type against
ShipmentDraftAllowedResources, which defaults to ["shipment"] alone. Without the
normalizer arm an inbound message would upload, extract, and silently produce no
draft — the same trap "shipment_import" already had to be dug out of.
*/
func TestDocumentControl_AllowsDraftsFromInboundMessages(t *testing.T) {
	t.Parallel()

	control := documentcontrol.NewDefaultDocumentControl(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
	)

	assert.True(t, control.AllowsShipmentDraftResource("inbound_message"),
		"a tender forwarded by email must draft like one imported from a file")
	assert.True(t, control.AllowsShipmentDraftResource("shipment_import"))
	assert.True(t, control.AllowsShipmentDraftResource("shipment"))
	assert.False(t, control.AllowsShipmentDraftResource("worker"))
}

/*
A rate confirmation dropped into a Desk conversation is read so the agent can
draft the shipment from it. Before this arm the draft was refused for the
conversation's resource type: the model's reading was started, but the draft
was stored at once from the rule-based reading and marked Unavailable, so the
agent and the extraction card read stop names like "7,000" while the model's
reading was still on its way.
*/
func TestDocumentControl_AllowsDraftsFromConversationAttachments(t *testing.T) {
	t.Parallel()

	control := documentcontrol.NewDefaultDocumentControl(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
	)

	assert.True(t, control.AllowsShipmentDraftResource("assistant_thread"))
	assert.True(t, control.AllowsShipmentDraftResource("ASSISTANT_THREAD"))
}
