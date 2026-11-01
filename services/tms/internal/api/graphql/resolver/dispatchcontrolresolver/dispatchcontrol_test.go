package dispatchcontrolresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchControlResolvers_CheckTheDispatchControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"dispatchControl", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DispatchControl(ctx)
			return err
		}},
		{"updateDispatchControl", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateDispatchControl(
				ctx,
				gqlmodel.UpdateDispatchControlInput{},
			)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceDispatchControl,
				call.operation,
			)
		})
	}
}

func TestDispatchControlFromInput_MapsEveryFieldOntoTheStoredControl(t *testing.T) {
	t.Parallel()

	existing := &dispatchcontrol.DispatchControl{
		ID:             pulid.MustNew("x_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        3,
		CreatedAt:      100,
		UpdatedAt:      200,
	}
	v14 := 17
	v18 := 21
	v30 := pulid.MustNew("x_")
	v30Raw := v30.String()
	v31 := pulid.MustNew("x_")
	v31Raw := v31.String()
	v35 := "4.5"
	v44 := 0.25
	v45 := 48

	entity, err := dispatchControlFromInput(existing, &gqlmodel.UpdateDispatchControlInput{
		Version:                              3,
		EnableAutoAssignment:                 true,
		AutoAssignmentStrategy:               dispatchcontrol.AutoAssignmentStrategy("Performance"),
		EnforceWorkerAssign:                  true,
		EnforceTrailerContinuity:             true,
		EnforceHosCompliance:                 true,
		EnforceWorkerPtaRestrictions:         true,
		EnforceWorkerTractorFleetContinuity:  true,
		EnforceDriverQualificationCompliance: true,
		EnforceMedicalCertCompliance:         true,
		EnforceHazmatCompliance:              true,
		EnforceDrugAndAlcoholCompliance:      true,
		EnableAutoStopActuals:                true,
		ScoringWeights:                       map[string]any{"deadhead": 2.5},
		AutoAssignConfidenceThreshold:        "12.75",
		AutoAssignMaxDeadheadMiles:           &v14,
		AutoAssignPlanningHorizonHours:       18,
		PlanningMode:                         dispatchcontrol.PlanningMode("Horizon"),
		HorizonMaxMovesPerDriver:             20,
		HorizonSearchIterations:              &v18,
		CoverageRiskWindowHours:              22,
		EnableCheckCallMonitoring:            true,
		CheckCallIntervalHours:               24,
		EnableEtaAlerts:                      true,
		EtaRiskThresholdMinutes:              26,
		EtaDwellMinutes:                      27,
		ExceptionDueInfoMinutes:              28,
		ExceptionDueWarningMinutes:           29,
		ExceptionDueCriticalMinutes:          30,
		ExceptionEscalationMinutes:           31,
		ExceptionMaxEscalations:              32,
		ExceptionDefaultAssigneeID:           &v30Raw,
		ExceptionEscalationContactID:         &v31Raw,
		ExceptionMissedStopGraceMinutes:      35,
		ExceptionPodGraceHours:               36,
		TenderCounterOffersEnabled:           true,
		TenderCounterMarginFloorPercent:      &v35,
		TenderAutoAcceptCounters:             true,
		TenderCounterDecisionMinutes:         40,
		TenderExhaustedAction:                dispatchcontrol.TenderExhaustedAction("Rewaterfall"),
		TenderEscalationRateIncreasePercent:  "12.75",
		TenderEscalationOfferTTLMinutes:      43,
		TenderMaxEscalations:                 44,
		ComplianceEnforcementLevel:           dispatchcontrol.ComplianceEnforcementLevel("Audit"),
		RecordServiceFailures: dispatchcontrol.ServiceIncidentType(
			"AllExceptShipper",
		),
		ServiceFailureTarget:      &v44,
		ServiceFailureGracePeriod: &v45,
	})
	require.NoError(t, err)

	assert.Equal(t, existing.ID, entity.ID)
	assert.Equal(t, existing.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, existing.OrganizationID, entity.OrganizationID)
	assert.Equal(t, int64(100), entity.CreatedAt)
	assert.Equal(t, int64(3), entity.Version)
	assert.True(t, entity.EnableAutoAssignment)
	assert.Equal(
		t,
		dispatchcontrol.AutoAssignmentStrategy("Performance"),
		entity.AutoAssignmentStrategy,
	)
	assert.True(t, entity.EnforceWorkerAssign)
	assert.True(t, entity.EnforceTrailerContinuity)
	assert.True(t, entity.EnforceHOSCompliance)
	assert.True(t, entity.EnforceWorkerPTARestrictions)
	assert.True(t, entity.EnforceWorkerTractorFleetContinuity)
	assert.True(t, entity.EnforceDriverQualificationCompliance)
	assert.True(t, entity.EnforceMedicalCertCompliance)
	assert.True(t, entity.EnforceHazmatCompliance)
	assert.True(t, entity.EnforceDrugAndAlcoholCompliance)
	assert.True(t, entity.EnableAutoStopActuals)
	assert.InDelta(t, 2.5, entity.ScoringWeights["deadhead"], 0.0001)
	assert.Equal(t, "12.75", entity.AutoAssignConfidenceThreshold.String())
	require.NotNil(t, entity.AutoAssignMaxDeadheadMiles)
	assert.Equal(t, int32(17), *entity.AutoAssignMaxDeadheadMiles)
	assert.Equal(t, int16(18), entity.AutoAssignPlanningHorizonHours)
	assert.Equal(t, dispatchcontrol.PlanningMode("Horizon"), entity.PlanningMode)
	assert.Equal(t, int16(20), entity.HorizonMaxMovesPerDriver)
	require.NotNil(t, entity.HorizonSearchIterations)
	assert.Equal(t, int16(21), *entity.HorizonSearchIterations)
	assert.Equal(t, int16(22), entity.CoverageRiskWindowHours)
	assert.True(t, entity.EnableCheckCallMonitoring)
	assert.Equal(t, int16(24), entity.CheckCallIntervalHours)
	assert.True(t, entity.EnableEtaAlerts)
	assert.Equal(t, int16(26), entity.EtaRiskThresholdMinutes)
	assert.Equal(t, int16(27), entity.EtaDwellMinutes)
	assert.Equal(t, int32(28), entity.ExceptionDueInfoMinutes)
	assert.Equal(t, int32(29), entity.ExceptionDueWarningMinutes)
	assert.Equal(t, int32(30), entity.ExceptionDueCriticalMinutes)
	assert.Equal(t, int32(31), entity.ExceptionEscalationMinutes)
	assert.Equal(t, int16(32), entity.ExceptionMaxEscalations)
	assert.Equal(t, v30, entity.ExceptionDefaultAssigneeID)
	assert.Equal(t, v31, entity.ExceptionEscalationContactID)
	assert.Equal(t, int16(35), entity.ExceptionMissedStopGraceMinutes)
	assert.Equal(t, int16(36), entity.ExceptionPODGraceHours)
	assert.True(t, entity.TenderCounterOffersEnabled)
	assert.True(t, entity.TenderCounterMarginFloorPercent.Valid)
	assert.Equal(t, "4.5", entity.TenderCounterMarginFloorPercent.Decimal.String())
	assert.True(t, entity.TenderAutoAcceptCounters)
	assert.Equal(t, int32(40), entity.TenderCounterDecisionMinutes)
	assert.Equal(
		t,
		dispatchcontrol.TenderExhaustedAction("Rewaterfall"),
		entity.TenderExhaustedAction,
	)
	assert.Equal(t, "12.75", entity.TenderEscalationRateIncreasePercent.String())
	assert.Equal(t, int32(43), entity.TenderEscalationOfferTTLMinutes)
	assert.Equal(t, int16(44), entity.TenderMaxEscalations)
	assert.Equal(
		t,
		dispatchcontrol.ComplianceEnforcementLevel("Audit"),
		entity.ComplianceEnforcementLevel,
	)
	assert.Equal(
		t,
		dispatchcontrol.ServiceIncidentType("AllExceptShipper"),
		entity.RecordServiceFailures,
	)
	assert.Equal(t, &v44, entity.ServiceFailureTarget)
	require.NotNil(t, entity.ServiceFailureGracePeriod)
	assert.Equal(t, int(48), *entity.ServiceFailureGracePeriod)
}

func TestDispatchControlFromInput_NamesAMalformedField(t *testing.T) {
	t.Parallel()

	_, err := dispatchControlFromInput(
		&dispatchcontrol.DispatchControl{},
		&gqlmodel.UpdateDispatchControlInput{
			AutoAssignConfidenceThreshold:       "twelve",
			TenderEscalationRateIncreasePercent: "1",
		},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Equal(t, "autoAssignConfidenceThreshold", multiErr.Errors[0].Field)
}

func TestDispatchControlResolver_RendersDecimalsExactly(t *testing.T) {
	t.Parallel()

	value, err := (&DispatchControlResolver{&Deps{}}).AutoAssignConfidenceThreshold(
		t.Context(),
		&dispatchcontrol.DispatchControl{
			AutoAssignConfidenceThreshold: decimal.RequireFromString("0.725"),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "0.725", value)
}
