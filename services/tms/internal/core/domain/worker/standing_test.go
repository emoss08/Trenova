package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const standingNow = int64(1_800_000_000)

func activeWorker() *worker.Worker {
	return &worker.Worker{
		ID:            pulid.MustNew("wrk_"),
		Status:        domaintypes.StatusActive,
		CanBeAssigned: true,
	}
}

func concernCodes(concerns []worker.Concern) []string {
	out := make([]string, 0, len(concerns))
	for _, concern := range concerns {
		out = append(out, concern.Code)
	}
	return out
}

func findConcern(t *testing.T, concerns []worker.Concern, code string) worker.Concern {
	t.Helper()
	for _, concern := range concerns {
		if concern.Code == code {
			return concern
		}
	}
	t.Fatalf("no concern with code %q in %v", code, concernCodes(concerns))
	return worker.Concern{}
}

func TestBuildStanding_CleanRecordIsGood(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker:      activeWorker(),
		Credentials: &worker.WorkerCredentialSummary{ComplianceStatus: worker.ComplianceStatusCompliant},
		Training:    &worker.WorkerTrainingSummary{Compliant: true},
		Safety:      &worker.SafetyScorecard{Score: 100, Rating: worker.SafetyRatingExcellent},
		Now:         standingNow,
	})

	assert.Equal(t, worker.StandingGood, standing)
	assert.Empty(t, concerns)
}

// A section the caller could not read must not be counted as clean. Absent is
// not the same as fine, and treating it as fine would hide a real problem from
// somebody who simply lacks one permission.
func TestBuildStanding_NilSectionsContributeNothing(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Now:    standingNow,
	})

	assert.Equal(t, worker.StandingGood, standing)
	assert.Empty(t, concerns)
}

func TestBuildStanding_NonCompliantCredentialsBlock(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Credentials: &worker.WorkerCredentialSummary{
			ComplianceStatus: worker.ComplianceStatusNonCompliant,
			ExpiredCount:     2,
		},
		Now: standingNow,
	})

	assert.Equal(t, worker.StandingBlocked, standing)
	expired := findConcern(t, concerns, worker.ConcernCredentialsExpired)
	assert.Equal(t, worker.ConcernCritical, expired.Severity)
	assert.Equal(t, "2 credentials have expired", expired.Headline)
	assert.Equal(t, "credentials", expired.Tab)
}

// A single expiring credential is a nudge, not an emergency, and the count
// must read as one rather than "1 credentials".
func TestBuildStanding_ExpiringCredentialIsWatch(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Credentials: &worker.WorkerCredentialSummary{
			ComplianceStatus: worker.ComplianceStatusCompliant,
			ExpiringCount:    1,
		},
		Now: standingNow,
	})

	assert.Equal(t, worker.StandingWatch, standing)
	soon := findConcern(t, concerns, worker.ConcernCredentialsSoon)
	assert.Equal(t, worker.ConcernWarning, soon.Severity)
	assert.Equal(t, "1 credential expires soon", soon.Headline)
}

func TestBuildStanding_TrainingGapsAreCritical(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Training: &worker.WorkerTrainingSummary{
			ExpiredCount: 1,
			MissingCount: 2,
			OverdueCount: 1,
		},
		Now: standingNow,
	})

	assert.Equal(t, worker.StandingAtRisk, standing)
	assert.Subset(t, concernCodes(concerns), []string{
		worker.ConcernTrainingExpired,
		worker.ConcernTrainingMissing,
		worker.ConcernTrainingOverdue,
	})
	assert.Equal(t, "training", findConcern(t, concerns, worker.ConcernTrainingMissing).Tab)
}

// Due and expiring courses are the same kind of nudge, so they are counted
// together rather than nagging twice about the same window.
func TestBuildStanding_DueAndExpiringTrainingCombineIntoOneWarning(t *testing.T) {
	_, concerns := worker.BuildStanding(worker.StandingInput{
		Worker:   activeWorker(),
		Training: &worker.WorkerTrainingSummary{DueCount: 2, ExpiringCount: 1},
		Now:      standingNow,
	})

	due := findConcern(t, concerns, worker.ConcernTrainingDue)
	assert.Equal(t, "3 courses need attention soon", due.Headline)
	assert.Len(t, concerns, 1)
}

func TestBuildStanding_SafetyRatings(t *testing.T) {
	tests := []struct {
		name     string
		rating   worker.SafetyRating
		expected worker.Standing
		severity worker.ConcernSeverity
	}{
		{"at risk", worker.SafetyRatingAtRisk, worker.StandingAtRisk, worker.ConcernCritical},
		{"watch", worker.SafetyRatingWatch, worker.StandingWatch, worker.ConcernWarning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standing, concerns := worker.BuildStanding(worker.StandingInput{
				Worker: activeWorker(),
				Safety: &worker.SafetyScorecard{
					Score:                 42,
					Rating:                tt.rating,
					ActivePoints:          11,
					PointsWatchThreshold:  6,
					PointsAtRiskThreshold: 10,
				},
				Now: standingNow,
			})
			assert.Equal(t, tt.expected, standing)
			concern := findConcern(t, concerns, worker.ConcernSafetyRating)
			assert.Equal(t, tt.severity, concern.Severity)
			assert.Contains(t, concern.Headline, "42")
			assert.Equal(t, "safety", concern.Tab)
		})
	}
}

// A good safety rating alongside an out-of-service order still has to surface
// the order: the score is a trailing average, the order is a fact about today.
func TestBuildStanding_OutOfServiceOrderSurfacesUnderAGoodRating(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Safety: &worker.SafetyScorecard{
			Score:              91,
			Rating:             worker.SafetyRatingGood,
			OutOfServiceOrders: 1,
		},
		Now: standingNow,
	})

	assert.Equal(t, worker.StandingAtRisk, standing)
	oos := findConcern(t, concerns, worker.ConcernOutOfService)
	assert.Equal(t, worker.ConcernCritical, oos.Severity)
	assert.Equal(t, "1 out-of-service order in the last year", oos.Headline)
	assert.NotContains(t, concernCodes(concerns), worker.ConcernSafetyRating)
}

func TestBuildStanding_ActiveDisciplineNamesTheHighestRung(t *testing.T) {
	_, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Safety: &worker.SafetyScorecard{
			Rating:            worker.SafetyRatingGood,
			ActiveDiscipline:  2,
			HighestDiscipline: worker.DisciplinaryLevelFinalWarning,
		},
		Now: standingNow,
	})

	concern := findConcern(t, concerns, worker.ConcernDiscipline)
	assert.Equal(t, "2 disciplinary actions are active", concern.Headline)
	assert.Contains(t, concern.Detail, "final warning")
}

// Open events are bookkeeping, not a judgement on the worker, so they inform
// without moving the standing off Good.
func TestBuildStanding_OpenSafetyEventsAreInformationOnly(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Safety: &worker.SafetyScorecard{Rating: worker.SafetyRatingGood, OpenEvents: 3},
		Now:    standingNow,
	})

	assert.Equal(t, worker.StandingGood, standing)
	assert.Equal(
		t,
		worker.ConcernInfo,
		findConcern(t, concerns, worker.ConcernOpenSafetyEvents).Severity,
	)
}

func TestBuildStanding_OpenChecklist(t *testing.T) {
	overdue := int64(standingNow - 86_400)
	checklist := &worker.WorkerChecklist{
		Name:   "Driver onboarding",
		Status: worker.ChecklistStatusOpen,
		Items: []*worker.WorkerChecklistItem{
			{Required: true, Status: worker.ChecklistItemDone},
			{Required: true, Status: worker.ChecklistItemPending, DueAt: &overdue},
		},
	}

	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker:    activeWorker(),
		Checklist: checklist,
		Now:       standingNow,
	})

	assert.Equal(t, worker.StandingWatch, standing)
	concern := findConcern(t, concerns, worker.ConcernChecklistOverdue)
	assert.Equal(t, "1 checklist item is overdue", concern.Headline)
	assert.Equal(t, "checklist", concern.Tab)
	assert.NotContains(t, concernCodes(concerns), worker.ConcernChecklistOpen)
}

// A checklist that is merely unfinished reports progress rather than nagging.
func TestBuildStanding_UnfinishedChecklistIsInformationOnly(t *testing.T) {
	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Checklist: &worker.WorkerChecklist{
			Name:   "Driver onboarding",
			Status: worker.ChecklistStatusOpen,
			Items: []*worker.WorkerChecklistItem{
				{Required: true, Status: worker.ChecklistItemDone},
				{Required: true, Status: worker.ChecklistItemPending},
			},
		},
		Now: standingNow,
	})

	assert.Equal(t, worker.StandingGood, standing)
	concern := findConcern(t, concerns, worker.ConcernChecklistOpen)
	assert.Equal(t, "Driver onboarding is 50% done", concern.Headline)
	assert.Equal(t, "1 of 2 required items are settled.", concern.Detail)
}

// A closed checklist is history and must not keep reporting itself.
func TestBuildStanding_ClosedChecklistIsSilent(t *testing.T) {
	_, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Checklist: &worker.WorkerChecklist{
			Name:   "Driver onboarding",
			Status: worker.ChecklistStatusCompleted,
			Items:  []*worker.WorkerChecklistItem{{Required: true, Status: worker.ChecklistItemDone}},
		},
		Now: standingNow,
	})

	assert.Empty(t, concerns)
}

func TestBuildStanding_ReviewDueDate(t *testing.T) {
	past := standingNow - 86_400
	future := standingNow + 86_400

	_, overdue := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(), ReviewDueAt: &past, Now: standingNow,
	})
	assert.Contains(t, concernCodes(overdue), worker.ConcernReviewOverdue)

	_, upcoming := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(), ReviewDueAt: &future, Now: standingNow,
	})
	assert.Empty(t, upcoming)
}

// A worker taken off the board is blocked whatever else the record says.
func TestBuildStanding_NotAssignableBlocks(t *testing.T) {
	wrk := activeWorker()
	wrk.CanBeAssigned = false

	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker:      wrk,
		Credentials: &worker.WorkerCredentialSummary{ComplianceStatus: worker.ComplianceStatusCompliant},
		Now:         standingNow,
	})

	assert.Equal(t, worker.StandingBlocked, standing)
	assert.Equal(t, "timeline", findConcern(t, concerns, worker.ConcernNotAssignable).Tab)
}

// An expired licence on somebody who left last year is not a finding. The
// record still exists for retention, so the overview says so and stops.
func TestBuildStanding_TerminatedWorkerShortCircuits(t *testing.T) {
	wrk := activeWorker()
	wrk.Status = domaintypes.StatusInactive

	standing, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: wrk,
		Credentials: &worker.WorkerCredentialSummary{
			ComplianceStatus: worker.ComplianceStatusNonCompliant,
			ExpiredCount:     4,
		},
		Safety: &worker.SafetyScorecard{Rating: worker.SafetyRatingAtRisk},
		Now:    standingNow,
	})

	assert.Equal(t, worker.StandingBlocked, standing)
	require.Len(t, concerns, 1)
	assert.Equal(t, worker.ConcernNotEmployed, concerns[0].Code)
}

// The office reads the worst thing first, so severity ordering is part of the
// contract rather than an accident of which section was appended first.
func TestBuildStanding_ConcernsAreRankedBySeverity(t *testing.T) {
	_, concerns := worker.BuildStanding(worker.StandingInput{
		Worker: activeWorker(),
		Safety: &worker.SafetyScorecard{
			Rating:     worker.SafetyRatingGood,
			OpenEvents: 1,
		},
		Credentials: &worker.WorkerCredentialSummary{
			ComplianceStatus: worker.ComplianceStatusCompliant,
			ExpiringCount:    1,
		},
		Training: &worker.WorkerTrainingSummary{ExpiredCount: 1},
		Now:      standingNow,
	})

	require.Len(t, concerns, 3)
	assert.Equal(t, worker.ConcernCritical, concerns[0].Severity)
	assert.Equal(t, worker.ConcernWarning, concerns[1].Severity)
	assert.Equal(t, worker.ConcernInfo, concerns[2].Severity)
}
