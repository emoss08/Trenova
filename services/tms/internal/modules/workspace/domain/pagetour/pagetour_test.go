package pagetour_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func view() *pagetour.View {
	return &pagetour.View{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Tour:           "freight-outlook",
		TourVersion:    1,
		Outcome:        pagetour.OutcomeFinished,
	}
}

func TestView_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*pagetour.View)
		field  string
	}{
		{name: "a finished tour is valid", change: func(*pagetour.View) {}},
		{name: "a skipped tour is valid", change: func(v *pagetour.View) {
			v.Outcome = pagetour.OutcomeSkipped
		}},
		{name: "one word is a key", change: func(v *pagetour.View) { v.Tour = "console" }},
		{name: "the longest key", change: func(v *pagetour.View) {
			v.Tour = strings.Repeat("a", pagetour.MaxTourLength)
		}},
		{name: "no user", change: func(v *pagetour.View) { v.UserID = "" }, field: "userId"},
		{name: "no tour", change: func(v *pagetour.View) { v.Tour = "" }, field: "tour"},
		{name: "a key too long to store", change: func(v *pagetour.View) {
			v.Tour = strings.Repeat("a", pagetour.MaxTourLength+1)
		}, field: "tour"},
		{name: "a path is not a key", change: func(v *pagetour.View) {
			v.Tour = "/dispatch/outlook"
		}, field: "tour"},
		{name: "capitals are not a key", change: func(v *pagetour.View) {
			v.Tour = "FreightOutlook"
		}, field: "tour"},
		{name: "a trailing hyphen is not a key", change: func(v *pagetour.View) {
			v.Tour = "freight-"
		}, field: "tour"},
		{name: "version zero", change: func(v *pagetour.View) {
			v.TourVersion = 0
		}, field: "tourVersion"},
		{name: "a version past the ceiling", change: func(v *pagetour.View) {
			v.TourVersion = pagetour.MaxTourVersion + 1
		}, field: "tourVersion"},
		{name: "an outcome nobody defined", change: func(v *pagetour.View) {
			v.Outcome = "Paused"
		}, field: "outcome"},
		{name: "no outcome", change: func(v *pagetour.View) { v.Outcome = "" }, field: "outcome"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := view()
			tt.change(v)
			multiErr := errortypes.NewMultiError()
			v.Validate(multiErr)

			if tt.field == "" {
				assert.False(t, multiErr.HasErrors(), "%v", multiErr.Errors)
				return
			}
			fields := make([]string, 0, len(multiErr.Errors))
			for _, fieldErr := range multiErr.Errors {
				fields = append(fields, fieldErr.Field())
			}
			assert.Contains(t, fields, tt.field)
		})
	}
}

func TestOutcome_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, pagetour.OutcomeFinished.IsValid())
	assert.True(t, pagetour.OutcomeSkipped.IsValid())
	assert.False(t, pagetour.Outcome("").IsValid())
	assert.False(t, pagetour.Outcome("finished").IsValid())
}
