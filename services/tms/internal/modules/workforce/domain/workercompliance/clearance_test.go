package workercompliance_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpage"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func itemNamed(
	t *testing.T,
	clearance *workercompliance.Clearance,
	name string,
) *workercompliance.ClearanceItem {
	t.Helper()
	for _, item := range clearance.Items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("no clearance item named %q", name)
	return nil
}

func TestBuildClearance_NilWithoutCredentials(t *testing.T) {
	t.Parallel()

	assert.Nil(t, workercompliance.BuildClearance(workercompliance.ClearanceInput{
		Worker: activeWorker(),
		Now:    standingNow,
	}))
}

func TestBuildClearance_ClearedThroughTheFirstStop(t *testing.T) {
	t.Parallel()
	in := workercompliance.ClearanceInput{
		Worker: activeWorker(),
		Credentials: summaryOf(
			slot(credentialSlot{
				name:     "CDL",
				category: workercredential.CredentialCategoryLicense,
				required: true,
				health:   workercredential.CredentialHealthValid,
				expires:  at(200),
			}),
			slot(credentialSlot{
				name:     "Medical card",
				category: workercredential.CredentialCategoryMedical,
				required: true,
				health:   workercredential.CredentialHealthExpiringSoon,
				expires:  at(84),
			}),
			slot(credentialSlot{
				name:     "TWIC",
				category: workercredential.CredentialCategorySecurity,
				health:   workercredential.CredentialHealthValid,
				expires:  at(30),
			}),
		),
		Now: standingNow,
	}

	clearance := workercompliance.BuildClearance(in)

	require.NotNil(t, clearance)
	assert.True(t, clearance.Cleared)
	assert.Empty(t, clearance.Blocks)
	require.NotNil(t, clearance.ClearedThrough)
	assert.Equal(t, *at(84), *clearance.ClearedThrough)
	require.NotNil(t, clearance.FirstStop)
	assert.Equal(t, "Medical card", clearance.FirstStop.Name)
	assert.Nil(t, clearance.NotClearedSince)

	names := make([]string, 0, len(clearance.Items))
	for _, item := range clearance.Items {
		names = append(names, item.Name)
	}
	assert.Equal(t, []string{"TWIC", "Medical card", "CDL"}, names, "ascending by date")

	medical := itemNamed(t, clearance, "Medical card")
	assert.Equal(t, workercompliance.ClearanceLaneMedical, medical.Lane)
	assert.Equal(t, workercompliance.ClearanceEffectStopsAssignment, medical.Effect)
	require.NotNil(t, medical.Rule)
	assert.Equal(t, workercompliance.ClearanceRuleRequiredCredential, *medical.Rule)
	assert.Equal(t, int64(84), medical.DaysUntil)
	assert.Equal(t, workerpage.ViewCredentials, *medical.Destination.View)

	twic := itemNamed(t, clearance, "TWIC")
	assert.Equal(t, workercompliance.ClearanceEffectDue, twic.Effect)
	assert.Nil(t, twic.Rule)
	assert.Equal(t, workercompliance.ClearanceLaneLicense, twic.Lane)
}

func TestBuildClearance_DispatchRuleDecidesOptionalItems(t *testing.T) {
	t.Parallel()
	in := workercompliance.ClearanceInput{
		Worker: activeWorker(),
		Credentials: summaryOf(
			slot(credentialSlot{
				name:    "Hazmat",
				field:   workercredential.CredentialProfileFieldHazmatExpiry,
				health:  workercredential.CredentialHealthValid,
				expires: at(40),
			}),
			slot(credentialSlot{
				name:    "MVR",
				field:   workercredential.CredentialProfileFieldMVRDueDate,
				health:  workercredential.CredentialHealthValid,
				expires: at(90),
			}),
		),
		Dispatch: &workercompliance.DispatchVerdicts{Stops: []workercompliance.DispatchStop{
			{
				Requirement: workercompliance.DispatchRequirementHazmat,
				At:          *at(40),
				HazmatOnly:  true,
			},
			{
				Requirement: workercompliance.DispatchRequirementMVR,
				At:          *at(90),
				Regulation:  "49 CFR 391.25(c)(2)",
			},
		}},
		Now: standingNow,
	}

	clearance := workercompliance.BuildClearance(in)

	hazmat := itemNamed(t, clearance, "Hazmat")
	assert.Equal(t, workercompliance.ClearanceEffectStopsAssignment, hazmat.Effect)
	assert.True(t, hazmat.HazmatOnly)
	assert.Equal(t, workercompliance.ClearanceRuleDispatchCompliance, *hazmat.Rule)

	mvr := itemNamed(t, clearance, "MVR")
	assert.False(t, mvr.HazmatOnly)
	require.NotNil(t, mvr.Regulation)
	assert.Equal(t, "49 CFR 391.25(c)(2)", *mvr.Regulation)

	assert.True(t, clearance.Cleared)
	require.NotNil(t, clearance.ClearedThrough)
	assert.Equal(t, *at(90), *clearance.ClearedThrough, "a hazmat-only stop does not end the band")
	assert.Equal(t, "MVR", clearance.FirstStop.Name)
}

func TestBuildClearance_EveryBlockToday(t *testing.T) {
	t.Parallel()
	wrk := activeWorker()
	wrk.Status = domaintypes.StatusInactive
	wrk.AvailableForDispatch = false

	clearance := workercompliance.BuildClearance(workercompliance.ClearanceInput{
		Worker: wrk,
		Credentials: summaryOf(
			slot(credentialSlot{
				name:     "CDL",
				required: true,
				health:   workercredential.CredentialHealthExpired,
				expires:  at(-10),
			}),
			slot(credentialSlot{
				name:     "MVR",
				required: true,
				health:   workercredential.CredentialHealthMissing,
			}),
		),
		Dispatch: &workercompliance.DispatchVerdicts{
			RefusedToday: []workercompliance.DispatchRequirement{
				workercompliance.DispatchRequirementLicense,
			},
		},
		Now: standingNow,
	})

	assert.False(t, clearance.Cleared)
	assert.Equal(t, []workercompliance.ClearanceBlock{
		workercompliance.ClearanceBlockNotEmployed,
		workercompliance.ClearanceBlockDispatchHold,
		workercompliance.ClearanceBlockRequiredCredentialLapsed,
		workercompliance.ClearanceBlockRequiredCredentialMissing,
		workercompliance.ClearanceBlockDispatchCompliance,
	}, clearance.Blocks)
	require.NotNil(t, clearance.NotClearedSince)
	assert.Equal(t, *at(-10), *clearance.NotClearedSince)
	assert.Nil(t, clearance.ClearedThrough)
}

func TestBuildClearance_BlockedStandingIsNeverCleared(t *testing.T) {
	t.Parallel()
	notAssignable := activeWorker()
	notAssignable.CanBeAssigned = false
	inactive := activeWorker()
	inactive.Status = domaintypes.StatusInactive

	cases := map[string]workercompliance.ClearanceInput{
		"not assignable": {Worker: notAssignable, Credentials: summaryOf()},
		"not employed":   {Worker: inactive, Credentials: summaryOf()},
		"required credential lapsed": {
			Worker: activeWorker(),
			Credentials: summaryOf(slot(credentialSlot{
				name:     "CDL",
				required: true,
				health:   workercredential.CredentialHealthExpired,
				expires:  at(-1),
			})),
		},
		"required credential missing": {
			Worker: activeWorker(),
			Credentials: summaryOf(slot(credentialSlot{
				name:     "Medical card",
				required: true,
				health:   workercredential.CredentialHealthMissing,
			})),
		},
	}

	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in.Now = standingNow
			standing, _ := workercompliance.BuildStanding(workercompliance.StandingInput{
				Worker:      in.Worker,
				Credentials: in.Credentials,
				Now:         standingNow,
			})
			require.Equal(t, workercompliance.StandingBlocked, standing)

			clearance := workercompliance.BuildClearance(in)

			assert.False(t, clearance.Cleared)
			assert.NotEmpty(t, clearance.Blocks)
		})
	}
}

func TestBuildClearance_LanesSourcesAndGates(t *testing.T) {
	t.Parallel()
	review := at(60)
	queryDue := at(45)
	record := &workertraining.WorkerTrainingRecord{
		ID:        pulid.MustNew("wtr_"),
		Status:    workertraining.TrainingStatusCompleted,
		ExpiresAt: at(100),
	}

	clearance := workercompliance.BuildClearance(workercompliance.ClearanceInput{
		Worker: activeWorker(),
		Credentials: summaryOf(slot(credentialSlot{
			name:     "Background check",
			category: workercredential.CredentialCategoryBackground,
			health:   workercredential.CredentialHealthValid,
			expires:  at(300),
		})),
		Training: &workertraining.WorkerTrainingSummary{
			Items: []*workertraining.TrainingSummaryItem{{
				Course: &workertraining.TrainingCourse{
					ID:   pulid.MustNew("trc_"),
					Name: "Hazmat awareness",
				},
				Record:   record,
				Health:   worker.TrainingHealthCurrent,
				Required: true,
			}},
		},
		DrugAlcohol: &workerdot.DrugAlcoholStanding{NextClearinghouseQueryDue: queryDue},
		ReviewDueAt: review,
		Now:         standingNow,
	})

	require.Len(t, clearance.Items, 4)
	sources := map[workercompliance.ClearanceSource]workercompliance.ClearanceLane{}
	for _, item := range clearance.Items {
		sources[item.Source] = item.Lane
		if item.Source != workercompliance.ClearanceSourceCredential {
			assert.Equal(t, workercompliance.ClearanceEffectDue, item.Effect, item.Name)
		}
	}
	assert.Equal(t, map[workercompliance.ClearanceSource]workercompliance.ClearanceLane{
		workercompliance.ClearanceSourceCredential:         workercompliance.ClearanceLaneReviewsAndQueries,
		workercompliance.ClearanceSourceTraining:           workercompliance.ClearanceLaneTraining,
		workercompliance.ClearanceSourceClearinghouseQuery: workercompliance.ClearanceLaneReviewsAndQueries,
		workercompliance.ClearanceSourcePerformanceReview:  workercompliance.ClearanceLaneReviewsAndQueries,
	}, sources)

	training := clearance.Items[2]
	assert.Equal(t, "Hazmat awareness", training.Name)
	assert.Equal(t, record.ID, *training.Destination.RecordID)
	assert.True(t, clearance.Cleared)
	assert.Nil(t, clearance.FirstStop)
	assert.Nil(t, clearance.ClearedThrough, "nothing dated ahead stops assignment")
}

func TestBuildClearance_CapsAtSixtyKeepingTheEarliest(t *testing.T) {
	t.Parallel()
	items := make([]*workercredential.CredentialSummaryItem, 0, 75)
	for day := range int64(75) {
		items = append(items, slot(credentialSlot{
			name:    "Permit",
			health:  workercredential.CredentialHealthValid,
			expires: at(day + 1),
		}))
	}

	clearance := workercompliance.BuildClearance(workercompliance.ClearanceInput{
		Worker:      activeWorker(),
		Credentials: summaryOf(items...),
		Now:         standingNow,
	})

	require.Len(t, clearance.Items, workercompliance.ClearanceItemCap)
	assert.True(t, clearance.ItemsTruncated)
	assert.Equal(t, *at(1), clearance.Items[0].DueAt)
	assert.Equal(t, *at(60), clearance.Items[59].DueAt)
}

func TestDispatchProbes_OnlyDatesTheDispatchCheckReads(t *testing.T) {
	t.Parallel()

	probes := workercompliance.DispatchProbes(workercompliance.ClearanceInput{
		Worker: activeWorker(),
		Credentials: summaryOf(
			slot(credentialSlot{
				name:    "CDL",
				field:   workercredential.CredentialProfileFieldLicenseExpiry,
				health:  workercredential.CredentialHealthValid,
				expires: at(100),
			}),
			slot(credentialSlot{
				name:    "TWIC",
				field:   workercredential.CredentialProfileFieldTWICExpiry,
				health:  workercredential.CredentialHealthValid,
				expires: at(50),
			}),
			slot(credentialSlot{
				name:   "Physical",
				field:  workercredential.CredentialProfileFieldPhysicalDueDate,
				health: workercredential.CredentialHealthMissing,
			}),
		),
		DrugAlcohol: &workerdot.DrugAlcoholStanding{NextClearinghouseQueryDue: at(20)},
		Now:         standingNow,
	})

	assert.Equal(t, []workercompliance.DispatchProbe{
		{Requirement: workercompliance.DispatchRequirementLicense, At: *at(100)},
		{Requirement: workercompliance.DispatchRequirementClearinghouseQuery, At: *at(20)},
	}, probes)
}

func TestDispatchVerdictsWithout_HidesTheRequirementButNotTheRefusal(t *testing.T) {
	t.Parallel()
	verdicts := &workercompliance.DispatchVerdicts{
		RefusedToday: []workercompliance.DispatchRequirement{
			workercompliance.DispatchRequirementDrugAlcohol,
		},
		Stops: []workercompliance.DispatchStop{
			{Requirement: workercompliance.DispatchRequirementClearinghouseQuery, At: *at(-5)},
			{Requirement: workercompliance.DispatchRequirementMVR, At: *at(30)},
		},
	}

	hidden := verdicts.Without(
		workercompliance.DispatchRequirementDrugAlcohol,
		workercompliance.DispatchRequirementClearinghouseQuery,
	)

	assert.False(t, hidden.Refused())
	assert.True(t, hidden.RefusedOutOfSight())
	assert.False(t, hidden.RefusesTodayOn(workercompliance.DispatchRequirementDrugAlcohol))
	assert.Equal(t, []workercompliance.DispatchStop{
		{Requirement: workercompliance.DispatchRequirementMVR, At: *at(30)},
	}, hidden.Stops)
	assert.Len(t, verdicts.RefusedToday, 1, "the original verdicts are untouched")

	clearance := workercompliance.BuildClearance(workercompliance.ClearanceInput{
		Worker:      activeWorker(),
		Credentials: summaryOf(),
		Dispatch:    hidden,
		Now:         standingNow,
	})
	assert.False(t, clearance.Cleared)
	assert.Equal(
		t,
		[]workercompliance.ClearanceBlock{workercompliance.ClearanceBlockDispatchHold},
		clearance.Blocks,
	)
}

func TestDispatchVerdictsWithout_AVisibleRefusalStillNamesDispatchCompliance(t *testing.T) {
	t.Parallel()
	hidden := (&workercompliance.DispatchVerdicts{
		RefusedToday: []workercompliance.DispatchRequirement{
			workercompliance.DispatchRequirementDrugAlcohol,
			workercompliance.DispatchRequirementLicense,
		},
	}).Without(workercompliance.DispatchRequirementDrugAlcohol)

	clearance := workercompliance.BuildClearance(workercompliance.ClearanceInput{
		Worker:      activeWorker(),
		Credentials: summaryOf(),
		Dispatch:    hidden,
		Now:         standingNow,
	})

	assert.False(t, hidden.RefusedOutOfSight())
	assert.Equal(
		t,
		[]workercompliance.ClearanceBlock{workercompliance.ClearanceBlockDispatchCompliance},
		clearance.Blocks,
	)
}
