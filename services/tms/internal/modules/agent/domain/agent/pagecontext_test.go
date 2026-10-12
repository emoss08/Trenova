package agent_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func pageErrors(page agent.PageContext) map[string]bool {
	multiErr := errortypes.NewMultiError()
	page.Validate("context", multiErr)
	fields := make(map[string]bool, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}

	return fields
}

func TestPageContextValidateAcceptsAKnownRecordOnAnAppPath(t *testing.T) {
	t.Parallel()

	errs := pageErrors(agent.PageContext{
		Path:       "/shipments?panelEntityId=shp_1",
		EntityType: "shipment",
		EntityID:   "shp_1",
		Title:      "Shipments",
	})

	assert.Empty(t, errs)
}

func TestPageContextValidateAcceptsEveryRecordTheRegistryOpensOnAPage(t *testing.T) {
	t.Parallel()

	for _, kind := range productguide.Default.RecordEntities() {
		if kind == "assistant_thread" {
			continue
		}
		errs := pageErrors(agent.PageContext{Path: "/x", EntityType: kind, EntityID: "rec_1"})
		assert.False(t, errs["context.entityType"], kind)
	}
	assert.True(
		t,
		pageErrors(agent.PageContext{Path: "/desk", EntityType: "assistant_thread", EntityID: "ath_1"})["context.entityType"],
		"the conversation a question is typed into is never its subject",
	)
}

func TestEntityRefsAcceptEveryKindTheMentionSearchOffers(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	agent.ValidateEntityRefs("mentions", []agent.EntityRef{
		{Type: "invoice_dispute", ID: pulid.MustNew("idsp_").String(), Label: "Dispute"},
		{Type: "billing_queue_item", ID: pulid.MustNew("bqi_").String(), Label: "Item"},
	}, multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestPageContextValidateRejectsWhatAClientCouldNotHaveShown(t *testing.T) {
	t.Parallel()

	assert.True(t, pageErrors(agent.PageContext{Path: ""})["context.path"], "path is required")
	assert.True(
		t,
		pageErrors(agent.PageContext{Path: "https://evil.example/x"})["context.path"],
		"only an application path is a page",
	)
	assert.True(
		t,
		pageErrors(agent.PageContext{Path: "/" + strings.Repeat("a", 500)})["context.path"],
		"an over-long path is rejected",
	)
	assert.True(
		t,
		pageErrors(agent.PageContext{Path: "/x", EntityType: "user_secret", EntityID: "1"})["context.entityType"],
		"an unknown record kind is rejected",
	)
	assert.True(
		t,
		pageErrors(agent.PageContext{Path: "/x", EntityID: "shp_1"})["context.entityType"],
		"a record identifier needs a kind",
	)
	assert.True(
		t,
		pageErrors(agent.PageContext{Path: "/x", Title: strings.Repeat("t", 201)})["context.title"],
		"an over-long title is rejected",
	)
}

func TestPageContextNormalizedTrimsEveryField(t *testing.T) {
	t.Parallel()

	page := (&agent.PageContext{
		Path:       "  /workers ",
		EntityType: " worker ",
		EntityID:   " wrk_1 ",
		Title:      " Drivers ",
	}).Normalized()

	assert.Equal(t, agent.PageContext{
		Path:       "/workers",
		EntityType: "worker",
		EntityID:   "wrk_1",
		Title:      "Drivers",
	}, *page)

	var none *agent.PageContext
	assert.Nil(t, none.Normalized())
}
