package carriertools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCarrierIntel struct {
	guard   *agenttoolstest.WriteGuard
	carrier *carrier.Carrier

	vetted    *carrierintelservice.VetCarrierRequest
	broker    pulid.ID
	enrolled  *carrierintelservice.EnrollSubjectsRequest
	reviewed  *carrierintelservice.MarkReviewedRequest
	applied   *carrierintelservice.ApplySuggestionsRequest
	imported  *carrierintelservice.ImportProspectRequest
	verified  *carrierintelservice.VerifyEquipmentRequest
	enrollOld carrierintel.DesiredState
}

func newFakeCarrierIntel() *fakeCarrierIntel {
	return &fakeCarrierIntel{
		guard: &agenttoolstest.WriteGuard{},
		carrier: &carrier.Carrier{
			ID:        pulid.MustNew("car_"),
			Name:      "Swift Haul",
			DOTNumber: "1234567",
			MCNumber:  "MC-1",
			City:      "Dallas",
			Version:   3,
		},
		enrollOld: carrierintel.DesiredStateNotEnrolled,
	}
}

func (f *fakeCarrierIntel) vetPlan() *carrierintelservice.VetPlan {
	return &carrierintelservice.VetPlan{
		Subject: carrierintelports.CarrierIntelSubject{
			Name:      f.carrier.Name,
			DOTNumber: f.carrier.DOTNumber,
		},
		Provider: integration.TypeCarrierOK,
		Depth:    carrierintel.LookupDepthFull,
		Current:  &carrierintel.CarrierIntelSnapshot{RiskLevel: carrierintel.RiskLevelLow},
	}
}

func (f *fakeCarrierIntel) VetCarrier(
	_ context.Context,
	req *carrierintelservice.VetCarrierRequest,
) (*carrierintelservice.FetchResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.vetted = req

	return &carrierintelservice.FetchResult{Snapshot: &carrierintel.CarrierIntelSnapshot{
		RiskLevel: carrierintel.RiskLevelLow,
	}}, nil
}

func (f *fakeCarrierIntel) PlanVetCarrier(
	context.Context,
	*carrierintelservice.VetCarrierRequest,
) (*carrierintelservice.VetPlan, error) {
	return f.vetPlan(), nil
}

func (f *fakeCarrierIntel) VetCustomerBroker(
	_ context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
	_ bool,
) (*carrierintelservice.FetchResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.broker = customerID

	return &carrierintelservice.FetchResult{}, nil
}

func (f *fakeCarrierIntel) PlanVetCustomerBroker(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*carrierintelservice.VetPlan, error) {
	plan := f.vetPlan()
	plan.Depth = carrierintel.LookupDepthFMCSA

	return plan, nil
}

func (f *fakeCarrierIntel) SetManualEnrollment(
	_ context.Context,
	req *carrierintelservice.EnrollSubjectsRequest,
) (int, error) {
	if err := f.guard.Write(); err != nil {
		return 0, err
	}
	f.enrolled = req

	return len(req.CarrierIDs), nil
}

func (f *fakeCarrierIntel) PlanManualEnrollment(
	_ context.Context,
	req *carrierintelservice.EnrollSubjectsRequest,
) ([]carrierintelservice.EnrollmentChange, error) {
	after := carrierintel.DesiredStateNotEnrolled
	if req.Enroll {
		after = carrierintel.DesiredStateEnrolled
	}

	return []carrierintelservice.EnrollmentChange{{
		CarrierID: req.CarrierIDs[0],
		Name:      f.carrier.Name,
		Before:    f.enrollOld,
		After:     after,
	}}, nil
}

func (f *fakeCarrierIntel) MarkReviewed(
	_ context.Context,
	req *carrierintelservice.MarkReviewedRequest,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.reviewed = req

	return nil
}

func (f *fakeCarrierIntel) PlanMarkReviewed(
	_ context.Context,
	req *carrierintelservice.MarkReviewedRequest,
) (*carrierintelservice.SnapshotChange, error) {
	before := &carrierintel.CarrierIntelSnapshot{RiskLevel: carrierintel.RiskLevelElevated}
	after := *before
	now := int64(1_900_000_000)
	after.ReviewedAt = &now
	after.ReviewedByID = req.TenantInfo.UserID
	after.ReviewNote = req.Note

	return &carrierintelservice.SnapshotChange{Before: before, After: &after}, nil
}

func (f *fakeCarrierIntel) SyncPlan(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*carrierintel.SyncPlan, error) {
	return &carrierintel.SyncPlan{Suggestions: []carrierintel.FieldUpdate{
		{Field: carrierintel.SyncFieldMCNumber, Current: "MC-1", Proposed: "MC-2"},
		{Field: carrierintel.SyncFieldCity, Current: "Dallas", Proposed: "Fort Worth"},
	}}, nil
}

func (f *fakeCarrierIntel) ApplySuggestions(
	_ context.Context,
	req *carrierintelservice.ApplySuggestionsRequest,
) (int, error) {
	if err := f.guard.Write(); err != nil {
		return 0, err
	}
	f.applied = req

	return len(req.Fields), nil
}

func (f *fakeCarrierIntel) PlanApplySuggestions(
	_ context.Context,
	req *carrierintelservice.ApplySuggestionsRequest,
) (*carrierintelservice.SuggestionPlan, error) {
	after := *f.carrier
	updates := make([]carrierintel.FieldUpdate, 0, len(req.Fields))
	for _, field := range req.Fields {
		switch field {
		case carrierintel.SyncFieldMCNumber:
			after.MCNumber = "MC-2"
		case carrierintel.SyncFieldCity:
			after.City = "Fort Worth"
		}
		updates = append(updates, carrierintel.FieldUpdate{Field: field})
	}

	return &carrierintelservice.SuggestionPlan{
		Before:  f.carrier,
		After:   &after,
		Fields:  updates,
		Applied: len(updates),
	}, nil
}

func (f *fakeCarrierIntel) ImportProspect(
	_ context.Context,
	req *carrierintelservice.ImportProspectRequest,
) (*carrier.Carrier, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.imported = req

	return f.carrier, nil
}

func (f *fakeCarrierIntel) PlanImportProspect(
	_ context.Context,
	req *carrierintelservice.ImportProspectRequest,
) (*carrierintelservice.ImportPlan, error) {
	return &carrierintelservice.ImportPlan{
		DOTNumber:        req.DOTNumber,
		Code:             req.Code,
		Provider:         integration.TypeCarrierOK,
		EnrollMonitoring: req.EnrollMonitoring,
	}, nil
}

func (f *fakeCarrierIntel) VerifyEquipment(
	_ context.Context,
	req *carrierintelservice.VerifyEquipmentRequest,
) (*carrierintel.CarrierEquipmentVerification, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.verified = req

	return &carrierintel.CarrierEquipmentVerification{
		Result: carrierintel.VerificationResultMatch,
	}, nil
}

func (f *fakeCarrierIntel) PlanVerifyEquipment(
	_ context.Context,
	req *carrierintelservice.VerifyEquipmentRequest,
) (*carrierintel.CarrierEquipmentVerification, *carrier.Carrier, error) {
	return &carrierintel.CarrierEquipmentVerification{
		CarrierAssignmentID: req.CarrierAssignmentID,
		CarrierID:           f.carrier.ID,
		ExpectedDOTNumber:   f.carrier.DOTNumber,
		UnitType:            req.UnitType,
		VIN:                 req.VIN,
	}, f.carrier, nil
}

func TestVetCarrier_PreviewsTheLookupAndIsAMoneyWrite(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := NewVetCarrierTool(intel)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamCarrierID:   intel.carrier.ID.String(),
		paramDepth:                  "Lite",
		toolparams.ParamForceResend: true,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		intel.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.True(t, preview.Partial)
	assert.Equal(
		t,
		agentpreview.PreviewOperationRun,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	assert.Contains(t, preview.Summary, "is charged")

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, carrierintel.LookupDepthLite, intel.vetted.Depth)
	assert.True(t, intel.vetted.Force)
	assert.Equal(t, "Low", result.Name)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceCarrierIntelligence, policy.Resource)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceCarrier, target.Resource)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.ExecuteParams(
			map[string]any{
				toolparams.ParamCarrierID: intel.carrier.ID.String(),
				paramDepth:                "Deep",
			},
		),
	))
}

func TestVetCustomerBroker_ChecksTheCustomer(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := NewVetCustomerBrokerTool(intel)
	customerID := pulid.MustNew("cus_")
	params := agenttoolstest.ExecuteParams(
		map[string]any{changepreview.FieldCustomerID: customerID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		intel.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		permission.ResourceCustomer,
		agenttoolstest.PreviewChange(t, preview, 0).Resource,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, customerID, intel.broker)
}

func TestSetCarrierMonitoring_ShowsOnlyTheCarriersThatChange(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := NewSetCarrierMonitoringTool(intel)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramCarrierIDs: []any{intel.carrier.ID.String(), pulid.MustNew("car_").String()},
		ParamEnabled:    true,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		intel.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 1)
	assert.Contains(t, preview.Summary, "1 have no DOT number")
	assert.Equal(
		t,
		"Enrolled",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			"monitoring",
		).After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, intel.enrolled.Enroll)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.ExecuteParams(
			map[string]any{paramCarrierIDs: []any{intel.carrier.ID.String()}},
		),
	))
}

func TestMarkCarrierIntelReviewed_IsAPersonsSignOff(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := NewMarkCarrierIntelReviewedTool(intel)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamCarrierID: intel.carrier.ID.String(),
		toolparams.ParamNote:      "Insurance lapse was a filing delay; certificate on file",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		intel.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"Insurance lapse was a filing delay; certificate on file",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			"reviewNote",
		).After,
	)

	require.ErrorIs(t, tool.Execute(t.Context(), params), toolspec.ErrNeedsAPersonsApproval)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(params.Params)))
	require.NotNil(t, intel.reviewed)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
}

func TestApplyCarrierIntelSuggestions_AppliesEverySuggestionWhenNoneAreNamed(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	permissions := &agenttoolstest.FakePermissionCheck{Allowed: true}
	tool := NewApplyCarrierIntelSuggestionsTool(intel, permissions)
	params := agenttoolstest.ExecuteParams(
		map[string]any{toolparams.ParamCarrierID: intel.carrier.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		intel.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "MC-2", agenttoolstest.FieldByPath(t, change, "mcNumber").After)
	assert.Equal(t, "Fort Worth", agenttoolstest.FieldByPath(t, change, "city").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.ElementsMatch(t, []carrierintel.SyncField{
		carrierintel.SyncFieldMCNumber, carrierintel.SyncFieldCity,
	}, intel.applied.Fields)
	require.NotEmpty(t, permissions.Requests)
	assert.Equal(t, permission.ResourceCarrier.String(), permissions.Requests[0].Resource)
	assert.Equal(t, permission.OpUpdate, permissions.Requests[0].Operation)

	named := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamCarrierID: intel.carrier.ID.String(),
		paramSuggestionFields:     []any{"city"},
	})
	require.NoError(t, tool.Execute(t.Context(), named))
	assert.Equal(t, []carrierintel.SyncField{carrierintel.SyncFieldCity}, intel.applied.Fields)

	refused := NewApplyCarrierIntelSuggestionsTool(
		intel,
		&agenttoolstest.FakePermissionCheck{Allowed: false},
	)
	require.Error(t, refused.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			toolparams.ParamCarrierID: intel.carrier.ID.String(),
			paramSuggestionFields:     []any{"fax"},
		})))
}

func TestImportSourcedCarrier_NeedsTheCarrierCreateGrantToo(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	permissions := &agenttoolstest.FakePermissionCheck{Allowed: true}
	tool := NewImportSourcedCarrierTool(intel, permissions)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramDOTNumber:        "7654321",
		paramEnrollMonitoring: true,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		intel.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationCreate,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "7654321", intel.imported.DOTNumber)
	assert.True(t, intel.imported.EnrollMonitoring)
	assert.Equal(t, permission.OpCreate, permissions.Requests[0].Operation)
	assert.Equal(t, permission.ResourceCarrierSourcing, tool.Policy().Resource)
	assert.Equal(t, permission.OpImport, tool.Policy().Operation)

	refused := NewImportSourcedCarrierTool(
		intel,
		&agenttoolstest.FakePermissionCheck{Allowed: false},
	)
	require.Error(t, refused.Execute(t.Context(), params))
}

func TestVerifyCarrierEquipment_RecordsTheCheck(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := NewVerifyCarrierEquipmentTool(intel)
	assignmentID := pulid.MustNew("ca_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramCarrierAssignmentID: assignmentID.String(),
		paramUnitType:            "Tractor",
		paramVIN:                 "1xkwd49x0kj123456",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		intel.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(t, "1XKWD49X0KJ123456",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), paramVIN).After)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, assignmentID, intel.verified.CarrierAssignmentID)
	assert.Equal(t, "Match", result.Name)
	assert.Equal(t, permission.ResourceEquipmentVerification, tool.Policy().Resource)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			paramCarrierAssignmentID: assignmentID.String(),
			paramUnitType:            "Van",
		})))
}
