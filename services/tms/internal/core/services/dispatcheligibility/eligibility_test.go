package dispatcheligibility_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(86400)

func blockingControl() *dispatchcontrol.DispatchControl {
	return &dispatchcontrol.DispatchControl{
		EnforceDriverQualificationCompliance: true,
		EnforceMedicalCertCompliance:         true,
		EnforceDrugAndAlcoholCompliance:      true,
		EnforceHazmatCompliance:              true,
		EnforceHOSCompliance:                 true,
		ComplianceEnforcementLevel:           dispatchcontrol.ComplianceEnforcementLevelBlock,
	}
}

func warningControl() *dispatchcontrol.DispatchControl {
	dc := blockingControl()
	dc.ComplianceEnforcementLevel = dispatchcontrol.ComplianceEnforcementLevelWarning
	return dc
}

func compliantWorker() *worker.Worker {
	now := timeutils.NowUnix()
	mvr := now + 90*day
	physical := now + 90*day
	medical := now + 365*day

	return &worker.Worker{
		ID:                   pulid.MustNew("wrk_"),
		Status:               domaintypes.StatusActive,
		CanBeAssigned:        true,
		AvailableForDispatch: true,
		Profile: &worker.WorkerProfile{
			DOB:               timeutils.YearsAgoUnix(35),
			LicenseExpiry:     now + 365*day,
			HireDate:          now - 365*day,
			LastDrugTest:      now - 300*day,
			LastMVRCheck:      now - 30*day,
			MVRDueDate:        &mvr,
			PhysicalDueDate:   &physical,
			MedicalCardExpiry: &medical,
		},
	}
}

func healthyHOS() *telematics.WorkerHOSState {
	return &telematics.WorkerHOSState{
		WorkerID:         pulid.MustNew("wrk_"),
		DutyStatus:       telematics.DutyStatusOffDuty,
		DriveRemainingMs: 10 * 3600 * 1000,
		ShiftRemainingMs: 12 * 3600 * 1000,
		CycleRemainingMs: 50 * 3600 * 1000,
		BreakRemainingMs: 8 * 3600 * 1000,
		RecordedAt:       timeutils.NowUnix(),
	}
}

func codes(eval *dispatcheligibility.Evaluation) []string {
	out := make([]string, 0, len(eval.Findings))
	for _, f := range eval.Findings {
		out = append(out, f.Code)
	}
	return out
}

func TestEvaluateWorkerCompliance_CleanWorkerProducesNoFindings(t *testing.T) {
	t.Parallel()

	eval := dispatcheligibility.EvaluateWorkerCompliance(dispatcheligibility.WorkerComplianceInput{
		Worker:  compliantWorker(),
		Control: blockingControl(),
	})

	assert.Empty(t, eval.Findings)
	assert.False(t, eval.Blocked())
}

func TestEvaluateWorkerCompliance_ExpiredLicenseBlocksUnderBlockLevel(t *testing.T) {
	t.Parallel()

	w := compliantWorker()
	w.Profile.LicenseExpiry = timeutils.NowUnix() - day

	eval := dispatcheligibility.EvaluateWorkerCompliance(dispatcheligibility.WorkerComplianceInput{
		Worker:  w,
		Control: blockingControl(),
	})

	require.Len(t, eval.Findings, 1)
	assert.Equal(t, dispatcheligibility.CodeLicenseExpired, eval.Findings[0].Code)
	assert.Equal(t, dispatcheligibility.SeverityBlock, eval.Findings[0].Severity)
	assert.Equal(t, "49 CFR 391.11(b)(5)", eval.Findings[0].Regulation)
	assert.True(t, eval.Blocked())
}

func TestEvaluateWorkerCompliance_WarningLevelDowngradesToWarn(t *testing.T) {
	t.Parallel()

	w := compliantWorker()
	w.Profile.LicenseExpiry = timeutils.NowUnix() - day

	eval := dispatcheligibility.EvaluateWorkerCompliance(dispatcheligibility.WorkerComplianceInput{
		Worker:  w,
		Control: warningControl(),
	})

	require.Len(t, eval.Findings, 1)
	assert.Equal(t, dispatcheligibility.SeverityWarn, eval.Findings[0].Severity)
	assert.False(t, eval.Blocked())
}

func TestEvaluateWorkerCompliance_HazmatOnlyWhenShipmentCarriesHazmat(t *testing.T) {
	t.Parallel()

	w := compliantWorker()

	without := dispatcheligibility.EvaluateWorkerCompliance(
		dispatcheligibility.WorkerComplianceInput{
			Worker:               w,
			Control:              blockingControl(),
			HasHazmatCommodities: false,
		},
	)
	assert.Empty(t, without.Findings)

	with := dispatcheligibility.EvaluateWorkerCompliance(
		dispatcheligibility.WorkerComplianceInput{
			Worker:               w,
			Control:              blockingControl(),
			HasHazmatCommodities: true,
		},
	)
	assert.Contains(t, codes(with), dispatcheligibility.CodeHazmatEndorsementMissing)
}

func TestEvaluateWorkerCompliance_DisabledEnforcementSuppressesFindings(t *testing.T) {
	t.Parallel()

	w := compliantWorker()
	w.Profile.LicenseExpiry = timeutils.NowUnix() - day

	dc := blockingControl()
	dc.EnforceDriverQualificationCompliance = false

	eval := dispatcheligibility.EvaluateWorkerCompliance(dispatcheligibility.WorkerComplianceInput{
		Worker:  w,
		Control: dc,
	})

	assert.Empty(t, eval.Findings)
}

func TestEvaluateHOSClocks_MissingAndStaleStayInformational(t *testing.T) {
	t.Parallel()

	missing := dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		Control: blockingControl(),
	})
	require.Len(t, missing.Findings, 1)
	assert.Equal(t, dispatcheligibility.CodeHOSMissing, missing.Findings[0].Code)
	assert.Equal(t, dispatcheligibility.SeverityInfo, missing.Findings[0].Severity)
	assert.False(t, missing.Blocked())

	stale := healthyHOS()
	stale.RecordedAt = timeutils.NowUnix() - dispatcheligibility.HOSStateMaxAgeSeconds - 1
	staleEval := dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		State:   stale,
		Control: blockingControl(),
	})
	require.Len(t, staleEval.Findings, 1)
	assert.Equal(t, dispatcheligibility.CodeHOSStale, staleEval.Findings[0].Code)
	assert.Equal(t, dispatcheligibility.SeverityInfo, staleEval.Findings[0].Severity)
}

// Info findings describe telematics gaps. They must never surface as validation errors
// on the assignment write path, which has always treated absent HOS as "cannot judge".
func TestAppendToMultiError_SkipsInfoFindings(t *testing.T) {
	t.Parallel()

	eval := dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		Control: blockingControl(),
	})

	multiErr := errortypes.NewMultiError()
	eval.AppendToMultiError(multiErr, "primaryWorker")

	assert.False(t, multiErr.HasErrors())
}

func TestEvaluateHOSClocks_ExhaustedClocksReportEveryViolation(t *testing.T) {
	t.Parallel()

	state := healthyHOS()
	state.DriveRemainingMs = 0
	state.ShiftRemainingMs = 0
	state.CycleRemainingMs = 0

	eval := dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		State:   state,
		Control: blockingControl(),
	})

	assert.ElementsMatch(t, []string{
		dispatcheligibility.CodeHOSNoDriveTime,
		dispatcheligibility.CodeHOSNoShiftTime,
		dispatcheligibility.CodeHOSNoCycleTime,
	}, codes(eval))
	assert.True(t, eval.Blocked())
}

func TestEvaluateHOSClocks_BreakOnlyRequiredWhileDriving(t *testing.T) {
	t.Parallel()

	state := healthyHOS()
	state.BreakRemainingMs = 0

	offDuty := dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		State:   state,
		Control: blockingControl(),
	})
	assert.Empty(t, offDuty.Findings)

	state.DutyStatus = telematics.DutyStatusDriving
	driving := dispatcheligibility.EvaluateHOSClocks(dispatcheligibility.HOSInput{
		State:   state,
		Control: blockingControl(),
	})
	assert.Contains(t, codes(driving), dispatcheligibility.CodeHOSBreakRequired)
}

func TestTimeWindowOverlaps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		a      dispatcheligibility.TimeWindow
		b      dispatcheligibility.TimeWindow
		expect bool
	}{
		{"disjoint before", dispatcheligibility.TimeWindow{Start: 100, End: 200}, dispatcheligibility.TimeWindow{Start: 200, End: 300}, false},
		{"disjoint after", dispatcheligibility.TimeWindow{Start: 300, End: 400}, dispatcheligibility.TimeWindow{Start: 100, End: 300}, false},
		{"partial overlap", dispatcheligibility.TimeWindow{Start: 100, End: 250}, dispatcheligibility.TimeWindow{Start: 200, End: 300}, true},
		{"contained", dispatcheligibility.TimeWindow{Start: 150, End: 175}, dispatcheligibility.TimeWindow{Start: 100, End: 300}, true},
		{"open ended overlaps", dispatcheligibility.TimeWindow{Start: 100}, dispatcheligibility.TimeWindow{Start: 500, End: 600}, true},
		{"zero window never overlaps", dispatcheligibility.TimeWindow{}, dispatcheligibility.TimeWindow{Start: 100, End: 200}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expect, tc.a.Overlaps(tc.b))
		})
	}
}

func TestEvaluateAvailability_BlocksAreIndependentOfEnforcementLevel(t *testing.T) {
	t.Parallel()

	w := compliantWorker()
	w.AvailableForDispatch = false
	w.AssignmentBlocked = "Pending drug screen"

	eval := dispatcheligibility.EvaluateAvailability(dispatcheligibility.AvailabilityInput{
		Worker:     w,
		MoveWindow: dispatcheligibility.TimeWindow{Start: 100, End: 200},
		Control:    warningControl(),
	})

	assert.ElementsMatch(t, []string{
		dispatcheligibility.CodeWorkerNotForDispatch,
		dispatcheligibility.CodeWorkerBlocked,
	}, codes(eval))
	assert.True(t, eval.Blocked())
}

func TestEvaluateAvailability_ApprovedPTOOverlappingTheMoveBlocks(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()

	eval := dispatcheligibility.EvaluateAvailability(dispatcheligibility.AvailabilityInput{
		Worker: compliantWorker(),
		ApprovedPTO: []dispatcheligibility.PTOWindow{{
			Window: dispatcheligibility.TimeWindow{Start: now, End: now + 3*day},
			Type:   worker.PTOTypeVacation,
		}},
		MoveWindow: dispatcheligibility.TimeWindow{Start: now + day, End: now + 2*day},
		Control:    blockingControl(),
	})

	require.Len(t, eval.Findings, 1)
	assert.Equal(t, dispatcheligibility.CodeWorkerOnPTO, eval.Findings[0].Code)
}

func TestEvaluateAvailability_PTOOutsideTheMoveIsIgnored(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()

	eval := dispatcheligibility.EvaluateAvailability(dispatcheligibility.AvailabilityInput{
		Worker: compliantWorker(),
		ApprovedPTO: []dispatcheligibility.PTOWindow{{
			Window: dispatcheligibility.TimeWindow{Start: now + 10*day, End: now + 12*day},
			Type:   worker.PTOTypeVacation,
		}},
		MoveWindow: dispatcheligibility.TimeWindow{Start: now, End: now + day},
		Control:    blockingControl(),
	})

	assert.Empty(t, eval.Findings)
}

func TestEvaluateAvailability_OverlappingCommitmentBlocks(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()

	eval := dispatcheligibility.EvaluateAvailability(dispatcheligibility.AvailabilityInput{
		Worker: compliantWorker(),
		CommittedWindows: []dispatcheligibility.TimeWindow{
			{Start: now, End: now + 6*3600},
		},
		MoveWindow: dispatcheligibility.TimeWindow{Start: now + 3600, End: now + 8*3600},
		Control:    blockingControl(),
	})

	require.Len(t, eval.Findings, 1)
	assert.Equal(t, dispatcheligibility.CodeWorkerOverlappingMove, eval.Findings[0].Code)
}

func TestEvaluateEquipment_OutOfServiceTractorBlocksTypeMismatchWarns(t *testing.T) {
	t.Parallel()

	requiredType := pulid.MustNew("etp_")

	eval := dispatcheligibility.EvaluateEquipment(dispatcheligibility.EquipmentInput{
		Tractor: &tractor.Tractor{
			Code:            "T-100",
			Status:          domaintypes.EquipmentStatusOOS,
			EquipmentTypeID: pulid.MustNew("etp_"),
		},
		RequiredTractorTypeID: requiredType,
		Control:               blockingControl(),
	})

	assert.ElementsMatch(t, []string{
		dispatcheligibility.CodeTractorUnavailable,
		dispatcheligibility.CodeTractorTypeMismatch,
	}, codes(eval))
	assert.True(t, eval.Blocked())
}

func TestEvaluateEquipment_MissingTractorBlocks(t *testing.T) {
	t.Parallel()

	eval := dispatcheligibility.EvaluateEquipment(dispatcheligibility.EquipmentInput{
		Control: blockingControl(),
	})

	require.Len(t, eval.Findings, 1)
	assert.Equal(t, dispatcheligibility.CodeWorkerNoTractor, eval.Findings[0].Code)
}

func TestEvaluateEquipment_FleetContinuityOnlyWhenEnforced(t *testing.T) {
	t.Parallel()

	workerFleet := pulid.MustNew("fc_")
	tractorFleet := pulid.MustNew("fc_")

	in := dispatcheligibility.EquipmentInput{
		Tractor: &tractor.Tractor{
			Code:            "T-100",
			Status:          domaintypes.EquipmentStatusAvailable,
			EquipmentTypeID: pulid.MustNew("etp_"),
			FleetCodeID:     tractorFleet,
		},
		WorkerFleetCodeID: workerFleet,
		Control:           blockingControl(),
	}

	assert.Empty(t, dispatcheligibility.EvaluateEquipment(in).Findings)

	in.Control.EnforceWorkerTractorFleetContinuity = true
	assert.Equal(
		t,
		[]string{dispatcheligibility.CodeFleetMismatch},
		codes(dispatcheligibility.EvaluateEquipment(in)),
	)
}

func TestEvaluate_ComposesEveryRuleSet(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()

	w := compliantWorker()
	w.AvailableForDispatch = false
	w.Profile.LicenseExpiry = now - day

	state := healthyHOS()
	state.DriveRemainingMs = 0

	eval := dispatcheligibility.Evaluate(dispatcheligibility.EvaluateInput{
		Candidate: dispatcheligibility.Candidate{
			Worker:   w,
			HOSState: state,
			Tractor: &tractor.Tractor{
				Code:            "T-100",
				Status:          domaintypes.EquipmentStatusAtMaintenance,
				EquipmentTypeID: pulid.MustNew("etp_"),
			},
			Trailer: &trailer.Trailer{
				Code:            "TR-200",
				Status:          domaintypes.EquipmentStatusAvailable,
				EquipmentTypeID: pulid.MustNew("etp_"),
			},
		},
		Requirements: dispatcheligibility.MoveRequirements{
			Window: dispatcheligibility.TimeWindow{Start: now, End: now + 8*3600},
		},
		Control: blockingControl(),
		Now:     now,
	})

	assert.ElementsMatch(t, []string{
		dispatcheligibility.CodeWorkerNotForDispatch,
		dispatcheligibility.CodeLicenseExpired,
		dispatcheligibility.CodeHOSNoDriveTime,
		dispatcheligibility.CodeTractorUnavailable,
	}, codes(eval))
	assert.True(t, eval.Blocked())
}

func TestEvaluateMany_SkipsCandidatesWithoutAWorker(t *testing.T) {
	t.Parallel()

	results := dispatcheligibility.EvaluateMany(dispatcheligibility.EvaluateManyInput{
		Candidates: []dispatcheligibility.Candidate{
			{Worker: compliantWorker(), HOSState: healthyHOS(), Tractor: &tractor.Tractor{
				Code:            "T-1",
				Status:          domaintypes.EquipmentStatusAvailable,
				EquipmentTypeID: pulid.MustNew("etp_"),
			}},
			{},
		},
		Requirements: dispatcheligibility.MoveRequirements{},
		Control:      blockingControl(),
	})

	require.Len(t, results, 1)
	assert.False(t, results[0].Evaluation.Blocked())
}

func TestComplianceErrorCode(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		errortypes.ErrComplianceViolation,
		dispatcheligibility.ComplianceErrorCode(dispatchcontrol.ComplianceEnforcementLevelBlock),
	)
	assert.Equal(
		t,
		errortypes.ErrInvalid,
		dispatcheligibility.ComplianceErrorCode(dispatchcontrol.ComplianceEnforcementLevelWarning),
	)
	assert.Equal(
		t,
		errortypes.ErrInvalid,
		dispatcheligibility.ComplianceErrorCode(dispatchcontrol.ComplianceEnforcementLevelAudit),
	)
}
