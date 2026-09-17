package agent_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
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
