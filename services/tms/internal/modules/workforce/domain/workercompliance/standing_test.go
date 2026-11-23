package workercompliance_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/safetyscorecard"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpage"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	standingNow = int64(1_800_000_000)
	standingDay = int64(timeutils.SecondsPerDay)
)

func activeWorker() *worker.Worker {
	return &worker.Worker{
		ID:                   pulid.MustNew("wrk_"),
		Status:               domaintypes.StatusActive,
		CanBeAssigned:        true,
		AvailableForDispatch: true,
		Profile:              &worker.WorkerProfile{},
	}
}

type credentialSlot struct {
	name     string
	category workercredential.CredentialCategory
	field    workercredential.CredentialProfileField
	required bool
	health   workercredential.CredentialHealth
	expires  *int64
}

func slot(s credentialSlot) *workercredential.CredentialSummaryItem {
	item := &workercredential.CredentialSummaryItem{
		CredentialType: &workercredential.WorkerCredentialType{
			ID:           pulid.MustNew("wct_"),
			Name:         s.name,
			Category:     s.category,
			ProfileField: s.field,
		},
		Health:   s.health,
		Required: s.required,
	}
	if s.health != workercredential.CredentialHealthMissing {
		item.Credential = &workercredential.WorkerCredential{
			ID:        pulid.MustNew("wcr_"),
			ExpiresAt: s.expires,
		}
	}
	return item
}

func summaryOf(
	items ...*workercredential.CredentialSummaryItem,
) *workercredential.WorkerCredentialSummary {
	summary := &workercredential.WorkerCredentialSummary{
		ComplianceStatus: worker.ComplianceStatusCompliant,
		Items:            items,
	}
	for _, item := range items {
		if item.BlocksWhenLapsed() && item.Health.Blocks() {
			summary.ComplianceStatus = worker.ComplianceStatusNonCompliant
		}
	}
	return summary
}

func at(offsetDays int64) *int64 {
	value := standingNow + offsetDays*standingDay
	return &value
}

func concernsWith(
	concerns []workercompliance.Concern,
	code workercompliance.ConcernCode,
) []workercompliance.Concern {
	out := make([]workercompliance.Concern, 0, len(concerns))
	for _, concern := range concerns {
		if concern.Code == code {
			out = append(out, concern)
		}
	}
	return out
}

func onlyConcern(
	t *testing.T,
	concerns []workercompliance.Concern,
	code workercompliance.ConcernCode,
) workercompliance.Concern {
	t.Helper()
	found := concernsWith(concerns, code)
	require.Len(t, found, 1, "concerns: %+v", concerns)
	return found[0]
}

func TestBuildStanding_CleanRecordIsGood(t *testing.T) {
	t.Parallel()

	standing, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:      activeWorker(),
		Credentials: summaryOf(),
		Training:    &workertraining.WorkerTrainingSummary{Compliant: true},
		Safety:      &safetyscorecard.SafetyScorecard{Rating: worker.SafetyRatingGood},
		Now:         standingNow,
	})

	assert.Equal(t, workercompliance.StandingGood, standing)
	assert.Empty(t, concerns)
}

func TestBuildStanding_NotEmployedShortCircuits(t *testing.T) {
	t.Parallel()
	wrk := activeWorker()
	wrk.Status = domaintypes.StatusInactive

	standing, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: wrk,
		Credentials: summaryOf(slot(credentialSlot{
			name:     "CDL",
			required: true,
			health:   workercredential.CredentialHealthMissing,
		})),
		Now: standingNow,
	})

	assert.Equal(t, workercompliance.StandingBlocked, standing)
	require.Len(t, concerns, 1)
	assert.Equal(t, workercompliance.ConcernNotEmployed, concerns[0].Code)
	assert.Equal(t, workerpage.TabEmployment, concerns[0].Destination.Tab)
}

func TestBuildStanding_OneConcernPerCredentialNamingIt(t *testing.T) {
	t.Parallel()
	cdl := slot(credentialSlot{
		name:     "CDL",
		category: workercredential.CredentialCategoryLicense,
		required: true,
		health:   workercredential.CredentialHealthExpired,
		expires:  at(-3),
	})
	medical := slot(credentialSlot{
		name:     "Medical card",
		category: workercredential.CredentialCategoryMedical,
		required: true,
		health:   workercredential.CredentialHealthExpiringSoon,
		expires:  at(20),
	})
	mvr := slot(credentialSlot{
		name:     "MVR review",
		required: true,
		health:   workercredential.CredentialHealthMissing,
	})

	standing, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:      activeWorker(),
		Credentials: summaryOf(cdl, medical, mvr),
		Now:         standingNow,
	})

	assert.Equal(t, workercompliance.StandingBlocked, standing)

	expired := onlyConcern(t, concerns, workercompliance.ConcernCredentialExpired)
	assert.Equal(t, workercompliance.ConcernCritical, expired.Severity)
	require.NotNil(t, expired.Subject)
	assert.Equal(t, "CDL", *expired.Subject)
	assert.Equal(t, 1, expired.Count)
	assert.True(t, expired.StopsAssignment)
	require.NotNil(t, expired.DaysUntil)
	assert.Equal(t, int64(-3), *expired.DaysUntil)
	require.NotNil(t, expired.Destination.RecordID)
	assert.Equal(t, cdl.Credential.ID, *expired.Destination.RecordID)
	assert.Equal(t, workerpage.RecordCredential, *expired.Destination.Record)

	expiring := onlyConcern(t, concerns, workercompliance.ConcernCredentialExpiring)
	assert.Equal(t, workercompliance.ConcernWarning, expiring.Severity)
	assert.True(t, expiring.StopsAssignment, "a required credential stops assignment at its date")
	require.NotNil(t, expiring.DaysUntil)
	assert.Equal(t, int64(20), *expiring.DaysUntil)

	missing := onlyConcern(t, concerns, workercompliance.ConcernCredentialMissing)
	assert.Nil(t, missing.Destination.RecordID)
	assert.Nil(t, missing.DueAt)
	assert.True(t, missing.StopsAssignment)
}

func TestBuildStanding_OptionalCredentialStopsOnlyWhenDispatchRefuses(t *testing.T) {
	t.Parallel()
	twic := slot(credentialSlot{
		name:    "TWIC",
		field:   workercredential.CredentialProfileFieldTWICExpiry,
		health:  workercredential.CredentialHealthExpiringSoon,
		expires: at(10),
	})
	hazmat := slot(credentialSlot{
		name:    "Hazmat",
		field:   workercredential.CredentialProfileFieldHazmatExpiry,
		health:  workercredential.CredentialHealthExpiringSoon,
		expires: at(12),
	})
	physical := slot(credentialSlot{
		name:    "Physical",
		field:   workercredential.CredentialProfileFieldPhysicalDueDate,
		health:  workercredential.CredentialHealthExpiringSoon,
		expires: at(14),
	})
	dispatch := &workercompliance.DispatchVerdicts{Stops: []workercompliance.DispatchStop{
		{Requirement: workercompliance.DispatchRequirementHazmat, At: *at(12), HazmatOnly: true},
		{Requirement: workercompliance.DispatchRequirementPhysical, At: *at(14)},
	}}

	_, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:      activeWorker(),
		Credentials: summaryOf(twic, hazmat, physical),
		Dispatch:    dispatch,
		Now:         standingNow,
	})

	byName := map[string]bool{}
	for _, concern := range concernsWith(concerns, workercompliance.ConcernCredentialExpiring) {
		byName[*concern.Subject] = concern.StopsAssignment
	}
	assert.Equal(t, map[string]bool{"TWIC": false, "Hazmat": false, "Physical": true}, byName)
}

func TestBuildStanding_TrainingConcernsArePerCourseAndNeverStop(t *testing.T) {
	t.Parallel()
	overdueRecord := &workertraining.WorkerTrainingRecord{
		ID:         pulid.MustNew("wtr_"),
		Status:     workertraining.TrainingStatusAssigned,
		AssignedAt: *at(-40),
		DueAt:      at(-5),
	}
	training := &workertraining.WorkerTrainingSummary{Items: []*workertraining.TrainingSummaryItem{
		{
			Course: &workertraining.TrainingCourse{
				ID:   pulid.MustNew("trc_"),
				Name: "Defensive driving",
			},
			Record:   overdueRecord,
			Health:   worker.TrainingHealthOverdue,
			Required: true,
		},
		{
			Course:   &workertraining.TrainingCourse{ID: pulid.MustNew("trc_"), Name: "Hazmat"},
			Health:   worker.TrainingHealthMissing,
			Required: true,
		},
		{
			Course: &workertraining.TrainingCourse{ID: pulid.MustNew("trc_"), Name: "Winter ops"},
			Record: &workertraining.WorkerTrainingRecord{
				ID:     pulid.MustNew("wtr_"),
				Status: workertraining.TrainingStatusAssigned,
				DueAt:  at(5),
			},
			Health: worker.TrainingHealthDueSoon,
		},
	}}

	standing, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:   activeWorker(),
		Training: training,
		Now:      standingNow,
	})

	assert.Equal(t, workercompliance.StandingAtRisk, standing)
	overdue := onlyConcern(t, concerns, workercompliance.ConcernTrainingOverdue)
	assert.Equal(t, "Defensive driving", *overdue.Subject)
	assert.Equal(t, overdueRecord.ID, *overdue.Destination.RecordID)
	require.NotNil(t, overdue.OccurredAt)
	assert.Equal(t, *at(-40), *overdue.OccurredAt)
	assert.Equal(t, int64(-5), *overdue.DaysUntil)
	assert.Equal(
		t,
		"Hazmat",
		*onlyConcern(t, concerns, workercompliance.ConcernTrainingMissing).Subject,
	)
	assert.Equal(
		t,
		"Winter ops",
		*onlyConcern(t, concerns, workercompliance.ConcernTrainingDue).Subject,
	)
	for _, concern := range concerns {
		assert.False(
			t,
			concern.StopsAssignment,
			"training never stops assignment: %s",
			concern.Code,
		)
	}
}

func TestBuildStanding_SafetyEventsAreOnePerRecord(t *testing.T) {
	t.Parallel()
	oos := &workersafety.WorkerSafetyEvent{
		ID:               pulid.MustNew("wse_"),
		Kind:             workersafety.SafetyEventInspection,
		InspectionResult: workersafety.InspectionResultOutOfService,
		Status:           workersafety.SafetyEventStatusClosed,
		OccurredAt:       *at(-30),
	}
	open := &workersafety.WorkerSafetyEvent{
		ID:         pulid.MustNew("wse_"),
		Kind:       workersafety.SafetyEventIncident,
		Status:     workersafety.SafetyEventStatusOpen,
		OccurredAt: *at(-2),
	}
	oldOOS := &workersafety.WorkerSafetyEvent{
		ID:               pulid.MustNew("wse_"),
		Kind:             workersafety.SafetyEventInspection,
		InspectionResult: workersafety.InspectionResultOutOfService,
		Status:           workersafety.SafetyEventStatusClosed,
		OccurredAt:       *at(-500),
	}

	_, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: activeWorker(),
		Safety: &safetyscorecard.SafetyScorecard{
			Rating:           worker.SafetyRatingWatch,
			ActivePoints:     6,
			ActiveDiscipline: 2,
		},
		SafetyEvents: []*workersafety.WorkerSafetyEvent{oos, open, oldOOS},
		Now:          standingNow,
	})

	outOfService := onlyConcern(t, concerns, workercompliance.ConcernOutOfService)
	assert.Equal(t, oos.ID, *outOfService.Destination.RecordID)
	assert.Equal(t, workerpage.TabSafety, outOfService.Destination.Tab)
	assert.Equal(t, workerpage.RecordEvent, *outOfService.Destination.Record)
	assert.Equal(
		t,
		open.ID,
		*onlyConcern(t, concerns, workercompliance.ConcernOpenSafetyEvent).Destination.RecordID,
	)
	assert.Equal(t, 6, onlyConcern(t, concerns, workercompliance.ConcernSafetyRating).Count)
	assert.Equal(t, 2, onlyConcern(t, concerns, workercompliance.ConcernActiveDiscipline).Count)
}

func TestBuildStanding_ChecklistNamesItselfAndLinksToIt(t *testing.T) {
	t.Parallel()
	checklist := &workerchecklist.WorkerChecklist{
		ID:        pulid.MustNew("wcl_"),
		Name:      "Onboarding",
		Status:    workerchecklist.ChecklistStatusOpen,
		StartedAt: *at(-3),
	}

	_, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:    activeWorker(),
		Checklist: checklist,
		Now:       standingNow,
	})

	concern := onlyConcern(t, concerns, workercompliance.ConcernChecklistOpen)
	assert.Equal(t, "Onboarding", *concern.Subject)
	assert.Equal(t, checklist.ID, *concern.Destination.RecordID)
	assert.Equal(t, workerpage.ViewChecklists, *concern.Destination.View)
}

func TestBuildStanding_ReviewOverdueAndDue(t *testing.T) {
	t.Parallel()

	_, overdue := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:      activeWorker(),
		ReviewDueAt: at(-1),
		Now:         standingNow,
	})
	standing, due := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:      activeWorker(),
		ReviewDueAt: at(12),
		Now:         standingNow,
	})
	_, far := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:      activeWorker(),
		ReviewDueAt: at(90),
		Now:         standingNow,
	})

	assert.Equal(
		t,
		workercompliance.ConcernWarning,
		onlyConcern(t, overdue, workercompliance.ConcernReviewOverdue).Severity,
	)
	reviewDue := onlyConcern(t, due, workercompliance.ConcernReviewDue)
	assert.Equal(t, workercompliance.ConcernInfo, reviewDue.Severity)
	assert.Equal(t, int64(12), *reviewDue.DaysUntil)
	assert.Equal(
		t,
		workercompliance.StandingGood,
		standing,
		"a review coming due does not move the standing",
	)
	assert.Empty(t, far)
}

func TestBuildStanding_ProhibitionIsCriticalAndStopsWhenDispatchRefuses(t *testing.T) {
	t.Parallel()

	standing, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:      activeWorker(),
		DrugAlcohol: &workerdot.DrugAlcoholStanding{Status: worker.DrugAlcoholProhibited},
		Dispatch: &workercompliance.DispatchVerdicts{
			RefusedToday: []workercompliance.DispatchRequirement{
				workercompliance.DispatchRequirementDrugAlcohol,
			},
		},
		Now: standingNow,
	})

	assert.Equal(t, workercompliance.StandingAtRisk, standing)
	concern := onlyConcern(t, concerns, workercompliance.ConcernDrugAlcoholProhibited)
	assert.Equal(t, workercompliance.ConcernCritical, concern.Severity)
	assert.True(t, concern.StopsAssignment)
	assert.Equal(t, workerpage.ViewDrugAndAlcohol, *concern.Destination.View)
}

func TestBuildStanding_ClearinghouseQuery(t *testing.T) {
	t.Parallel()
	lastQuery := at(-370)

	_, overdue := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: activeWorker(),
		DrugAlcohol: &workerdot.DrugAlcoholStanding{
			Status:                    worker.DrugAlcoholClear,
			LastClearinghouseQueryAt:  lastQuery,
			NextClearinghouseQueryDue: at(-5),
		},
		Now: standingNow,
	})
	_, blocking := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: activeWorker(),
		DrugAlcohol: &workerdot.DrugAlcoholStanding{
			Status:                    worker.DrugAlcoholClear,
			NextClearinghouseQueryDue: at(-5),
		},
		Dispatch: &workercompliance.DispatchVerdicts{Stops: []workercompliance.DispatchStop{{
			Requirement: workercompliance.DispatchRequirementClearinghouseQuery,
			At:          *at(-5),
		}}},
		Now: standingNow,
	})
	_, due := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: activeWorker(),
		DrugAlcohol: &workerdot.DrugAlcoholStanding{
			Status:                    worker.DrugAlcoholClear,
			NextClearinghouseQueryDue: at(17),
		},
		Now: standingNow,
	})

	late := onlyConcern(t, overdue, workercompliance.ConcernClearinghouseQueryOverdue)
	assert.Equal(t, workercompliance.ConcernWarning, late.Severity)
	assert.False(t, late.StopsAssignment)
	assert.Equal(t, lastQuery, late.OccurredAt)
	stopped := onlyConcern(t, blocking, workercompliance.ConcernClearinghouseQueryOverdue)
	assert.Equal(t, workercompliance.ConcernCritical, stopped.Severity)
	assert.True(t, stopped.StopsAssignment)
	soon := onlyConcern(t, due, workercompliance.ConcernClearinghouseQueryDue)
	assert.Equal(t, int64(17), *soon.DaysUntil)
	assert.Equal(t, workercompliance.ConcernInfo, soon.Severity)
}

func TestBuildStanding_OutstandingEmployerVerificationsArePerEmployer(t *testing.T) {
	t.Parallel()
	requested := at(-20)
	outstanding := &employment.WorkerEmploymentVerification{
		ID:           pulid.MustNew("wemv_"),
		EmployerName: "Ridgeline Freight",
		Status:       employment.VerificationRequested,
		RequestedAt:  requested,
	}
	settled := &employment.WorkerEmploymentVerification{
		ID:           pulid.MustNew("wemv_"),
		EmployerName: "Done Haulers",
		Status:       employment.VerificationReceived,
	}

	_, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker:             activeWorker(),
		Verifications:      []*employment.WorkerEmploymentVerification{outstanding, settled},
		SafetyHistoryDueAt: *at(-2),
		Now:                standingNow,
	})

	concern := onlyConcern(t, concerns, workercompliance.ConcernEmploymentVerificationOutstanding)
	assert.Equal(t, "Ridgeline Freight", *concern.Subject)
	assert.Equal(t, requested, concern.OccurredAt)
	assert.Equal(t, workercompliance.ConcernWarning, concern.Severity, "past the 30-day deadline")
	assert.Equal(t, outstanding.ID, *concern.Destination.RecordID)
	assert.Equal(t, workerpage.RecordEmployer, *concern.Destination.Record)
}

func TestBuildStanding_RanksWorstFirstThenStopsThenSoonest(t *testing.T) {
	t.Parallel()

	_, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: activeWorker(),
		Credentials: summaryOf(
			slot(
				credentialSlot{
					name:    "Late",
					health:  workercredential.CredentialHealthExpiringSoon,
					expires: at(25),
				},
			),
			slot(
				credentialSlot{
					name:    "Soon",
					health:  workercredential.CredentialHealthExpiringSoon,
					expires: at(5),
				},
			),
			slot(credentialSlot{
				name:     "Required",
				required: true,
				health:   workercredential.CredentialHealthExpiringSoon,
				expires:  at(28),
			}),
			slot(credentialSlot{
				name:     "Gone",
				required: true,
				health:   workercredential.CredentialHealthExpired,
				expires:  at(-1),
			}),
		),
		Now: standingNow,
	})

	subjects := make([]string, 0, len(concerns))
	for _, concern := range concerns {
		subjects = append(subjects, *concern.Subject)
	}
	assert.Equal(t, []string{"Gone", "Required", "Soon", "Late"}, subjects)
}

func TestBuildStanding_NilSectionsContributeNothing(t *testing.T) {
	t.Parallel()

	standing, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: activeWorker(),
		Now:    standingNow,
	})

	assert.Equal(t, workercompliance.StandingGood, standing)
	assert.Empty(t, concerns)
}

func TestBuildStanding_NotAssignableBlocks(t *testing.T) {
	t.Parallel()
	wrk := activeWorker()
	wrk.CanBeAssigned = false

	standing, concerns := workercompliance.BuildStanding(workercompliance.StandingInput{
		Worker: wrk,
		Now:    standingNow,
	})

	assert.Equal(t, workercompliance.StandingBlocked, standing)
	concern := onlyConcern(t, concerns, workercompliance.ConcernNotAssignable)
	assert.True(t, concern.StopsAssignment)
	assert.Equal(t, workerpage.TabProfile, concern.Destination.Tab)
}
