package report_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveable() *report.ReportDefinition {
	return &report.ReportDefinition{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Name:           "In-Transit Shipments",
		Category:       "Operations",
		Kind:           report.DefinitionKindCustom,
		OwnerID:        pulid.MustNew("usr_"),
		Visibility:     report.VisibilityPrivate,
		Status:         report.DefinitionStatusActive,
		DefaultFormat:  report.FormatCSV,
		Definition:     &report.Definition{Entity: "shipment"},
	}
}

func messages(entity *report.ReportDefinition) []string {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if !multiErr.HasErrors() {
		return nil
	}

	out := make([]string, 0)
	for _, detail := range multiErr.Errors {
		out = append(out, detail.Message)
	}

	return out
}

/*
The revision is the repository's to assign: it sets 1 on insert, increments on
update, and writes the matching revision row. A caller has nothing to put
there, so a validator demanding one refused every create — which is what
create_report hit on approval, failing with "Revision must be at least one"
after the proposal had already been accepted.
*/
func TestValidate_UnsavedDefinitionOwesNoRevision(t *testing.T) {
	t.Parallel()

	assert.NotContains(t, messages(saveable()), "Revision must be at least one")
}

// A stored definition has been through the repository, so a revision below one
// is corruption rather than an unset field.
func TestValidate_StoredDefinitionKeepsTheRevisionFloor(t *testing.T) {
	t.Parallel()

	stored := saveable()
	stored.ID = pulid.MustNew("rd_")

	assert.Contains(t, messages(stored), "Revision must be at least one")

	stored.CurrentRevision = 1
	assert.NotContains(t, messages(stored), "Revision must be at least one")
}

func TestValidate_StillRefusesWhatTheCallerOwes(t *testing.T) {
	t.Parallel()

	blank := saveable()
	blank.Name = ""

	require.NotEmpty(t, messages(blank))
}
