package workerpage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpage"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDestinationQuery_UsesTheClientSlugs(t *testing.T) {
	t.Parallel()
	id := pulid.MustNew("wcr_")

	cases := map[string]struct {
		dest workerpage.Destination
		want map[string]string
	}{
		"overview is the bare address": {
			dest: workerpage.At(workerpage.TabOverview),
			want: map[string]string{},
		},
		"a tab without a view": {
			dest: workerpage.At(workerpage.TabSafety),
			want: map[string]string{"tab": "safety"},
		},
		"views are kebab case": {
			dest: workerpage.AtView(workerpage.TabTime, workerpage.ViewHoursOfService),
			want: map[string]string{"tab": "time", "view": "hours-of-service"},
		},
		"a record opens by its param": {
			dest: workerpage.AtView(workerpage.TabQualification, workerpage.ViewCredentials).
				WithRecord(workerpage.RecordCredential, id),
			want: map[string]string{
				"tab":        "qualification",
				"view":       "credentials",
				"credential": id.String(),
			},
		},
		"a safety event is safetyEvent": {
			dest: workerpage.At(workerpage.TabSafety).WithRecord(workerpage.RecordEvent, id),
			want: map[string]string{"tab": "safety", "safetyEvent": id.String()},
		},
		"an employment event is event": {
			dest: workerpage.AtView(workerpage.TabEmployment, workerpage.ViewHistory).
				WithRecord(workerpage.RecordEvent, id),
			want: map[string]string{"tab": "employment", "view": "history", "event": id.String()},
		},
		"a time off request is timeOff": {
			dest: workerpage.AtView(workerpage.TabTime, workerpage.ViewTimeOff).
				WithRecord(workerpage.RecordRequest, id),
			want: map[string]string{"tab": "time", "view": "time-off", "timeOff": id.String()},
		},
		"a profile change request is request": {
			dest: workerpage.AtView(workerpage.TabFiles, workerpage.ViewDriverPortal).
				WithRecord(workerpage.RecordRequest, id),
			want: map[string]string{
				"tab":     "files",
				"view":    "driver-portal",
				"request": id.String(),
			},
		},
		"a missing record opens the list": {
			dest: workerpage.AtView(workerpage.TabQualification, workerpage.ViewDqFile).
				WithRecord(workerpage.RecordEmployer, pulid.Nil),
			want: map[string]string{"tab": "qualification", "view": "dq-file"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.dest.Query())
		})
	}
}

func TestViewSlugs_CoverEveryView(t *testing.T) {
	t.Parallel()
	views := []workerpage.View{
		workerpage.ViewHistory, workerpage.ViewChecklists, workerpage.ViewCredentials,
		workerpage.ViewDqFile, workerpage.ViewDrugAndAlcohol, workerpage.ViewTraining,
		workerpage.ViewReviews, workerpage.ViewHoursOfService, workerpage.ViewSchedule,
		workerpage.ViewTimeOff, workerpage.ViewLeave, workerpage.ViewPayProfile,
		workerpage.ViewBenefits, workerpage.ViewDocuments, workerpage.ViewDriverPortal,
		workerpage.ViewAgentMemory,
	}
	slugs := make([]string, 0, len(views))
	for _, view := range views {
		slugs = append(slugs, view.Slug())
	}

	assert.Equal(t, []string{
		"history", "checklists", "credentials", "dq-file", "drug-and-alcohol", "training",
		"reviews", "hours-of-service", "schedule", "time-off", "leave", "pay-profile",
		"benefits", "documents", "driver-portal", "agent-memory",
	}, slugs)
}
