package dispatcheligibility_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatcheligibility"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerclearanceports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func hazmatWorker(now int64) *worker.Worker {
	w := compliantWorker()
	hazmat := now + 200*day
	w.Profile.Endorsement = worker.EndorsementTypeHazmat
	w.Profile.HazmatExpiry = &hazmat
	return w
}

func clearanceProbes(w *worker.Worker) []workercompliance.DispatchProbe {
	return []workercompliance.DispatchProbe{
		{
			Requirement: workercompliance.DispatchRequirementMedicalCard,
			At:          *w.Profile.MedicalCardExpiry,
		},
		{Requirement: workercompliance.DispatchRequirementPhysical, At: *w.Profile.PhysicalDueDate},
		{Requirement: workercompliance.DispatchRequirementHazmat, At: *w.Profile.HazmatExpiry},
	}
}

func TestClearanceVerdicts_WarningLevelStopsNothingDated(t *testing.T) {
	t.Parallel()
	now := timeutils.NowUnix()
	w := hazmatWorker(now)

	verdicts := dispatcheligibility.ClearanceVerdicts(w, warningControl(), now, clearanceProbes(w))

	assert.Empty(t, verdicts.Stops)
	assert.Empty(t, verdicts.RefusedToday)
	assert.False(t, verdicts.Refused())
}

func TestClearanceVerdicts_BlockLevelStopsAtEachDate(t *testing.T) {
	t.Parallel()
	now := timeutils.NowUnix()
	w := hazmatWorker(now)

	verdicts := dispatcheligibility.ClearanceVerdicts(w, blockingControl(), now, clearanceProbes(w))

	medical, ok := verdicts.StopAt(clearanceProbes(w)[0])
	require.True(t, ok)
	assert.False(t, medical.HazmatOnly)
	assert.Equal(t, "49 CFR 391.45", medical.Regulation)

	physical, ok := verdicts.StopAt(clearanceProbes(w)[1])
	require.True(t, ok)
	assert.False(t, physical.HazmatOnly)

	hazmat, ok := verdicts.StopAt(clearanceProbes(w)[2])
	require.True(t, ok)
	assert.True(t, hazmat.HazmatOnly, "the hazmat endorsement stops only hazmat loads")

	assert.Empty(t, verdicts.RefusedToday)
}

func TestClearanceVerdicts_DisabledEnforcementStopsNothing(t *testing.T) {
	t.Parallel()
	now := timeutils.NowUnix()
	w := hazmatWorker(now)
	control := blockingControl()
	control.EnforceMedicalCertCompliance = false

	verdicts := dispatcheligibility.ClearanceVerdicts(w, control, now, clearanceProbes(w))

	_, medical := verdicts.StopAt(clearanceProbes(w)[0])
	_, physical := verdicts.StopAt(clearanceProbes(w)[1])
	assert.False(t, medical)
	assert.False(t, physical)
}

func TestClearanceVerdicts_ProhibitionIsRefusedTodayAtEveryLevel(t *testing.T) {
	t.Parallel()
	now := timeutils.NowUnix()
	w := compliantWorker()
	w.Profile.DrugAlcoholStatus = worker.DrugAlcoholProhibited

	verdicts := dispatcheligibility.ClearanceVerdicts(w, warningControl(), now, nil)

	assert.True(t, verdicts.Refused())
	assert.True(t, verdicts.RefusesTodayOn(workercompliance.DispatchRequirementDrugAlcohol))
}

func TestClearanceVerdicts_ALapsedLicenseIsRefusedTodayUnderBlock(t *testing.T) {
	t.Parallel()
	now := timeutils.NowUnix()
	w := compliantWorker()
	w.Profile.LicenseExpiry = now - day

	verdicts := dispatcheligibility.ClearanceVerdicts(w, blockingControl(), now, nil)

	assert.True(t, verdicts.RefusesTodayOn(workercompliance.DispatchRequirementLicense))
}

func TestClearanceVerdicts_ProbesNeverChangeTheWorker(t *testing.T) {
	t.Parallel()
	now := timeutils.NowUnix()
	w := hazmatWorker(now)
	medical := *w.Profile.MedicalCardExpiry

	dispatcheligibility.ClearanceVerdicts(
		w,
		blockingControl(),
		now,
		[]workercompliance.DispatchProbe{
			{Requirement: workercompliance.DispatchRequirementMedicalCard, At: now - day},
		},
	)

	assert.Equal(t, medical, *w.Profile.MedicalCardExpiry)
}

func TestClearanceCheck_FallsBackToTheDefaultControl(t *testing.T) {
	t.Parallel()
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	repo := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
	repo.EXPECT().
		GetByOrgID(mock.Anything, dispatchcontrolports.GetDispatchControlRequest{TenantInfo: tenantInfo}).
		Return(nil, errortypes.NewNotFoundError("no dispatch control")).
		Once()
	check := dispatcheligibility.NewClearanceCheck(dispatcheligibility.ClearanceCheckParams{
		ControlRepo: repo,
	})
	now := timeutils.NowUnix()
	w := hazmatWorker(now)

	verdicts, err := check.Verdicts(t.Context(), &workerclearanceports.DispatchRequest{
		TenantInfo: tenantInfo,
		Worker:     w,
		Now:        now,
		Probes:     clearanceProbes(w),
	})

	require.NoError(t, err)
	assert.Equal(
		t,
		dispatchcontrol.ComplianceEnforcementLevelWarning,
		dispatchcontrol.NewDefaultDispatchControl(tenantInfo.OrgID, tenantInfo.BuID).
			ComplianceEnforcementLevel,
	)
	assert.Empty(t, verdicts.Stops, "the default enforcement level warns")
}

func TestClearanceCheck_FailsWhenTheControlCannotBeRead(t *testing.T) {
	t.Parallel()
	repo := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
	repo.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(nil, assert.AnError).
		Once()
	check := dispatcheligibility.NewClearanceCheck(dispatcheligibility.ClearanceCheckParams{
		ControlRepo: repo,
	})

	_, err := check.Verdicts(t.Context(), &workerclearanceports.DispatchRequest{
		Worker: compliantWorker(),
		Now:    timeutils.NowUnix(),
	})

	require.ErrorIs(t, err, assert.AnError)
}
