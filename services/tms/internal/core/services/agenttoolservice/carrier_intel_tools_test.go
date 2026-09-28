package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCarrierIntel struct {
	guard   *writeGuard
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
		guard: &writeGuard{},
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
		Subject:  repositories.CarrierIntelSubject{Name: f.carrier.Name, DOTNumber: f.carrier.DOTNumber},
		Provider: integration.TypeCarrierOK,
		Depth:    carrierintel.LookupDepthFull,
		Current:  &carrierintel.CarrierIntelSnapshot{RiskLevel: carrierintel.RiskLevelLow},
	}
}

func (f *fakeCarrierIntel) VetCarrier(
	_ context.Context,
	req *carrierintelservice.VetCarrierRequest,
) (*carrierintelservice.FetchResult, error) {
	if err := f.guard.write(); err != nil {
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
	if err := f.guard.write(); err != nil {
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
	if err := f.guard.write(); err != nil {
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
	if err := f.guard.write(); err != nil {
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
	if err := f.guard.write(); err != nil {
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
	if err := f.guard.write(); err != nil {
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
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.verified = req

	return &carrierintel.CarrierEquipmentVerification{Result: carrierintel.VerificationResultMatch}, nil
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
	tool := newVetCarrierTool(intel)
	params := executeParams(map[string]any{
		paramCarrierID:   intel.carrier.ID.String(),
		paramDepth:       "Lite",
		paramForceResend: true,
	})

	preview := previewWithoutWrites(t, intel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.True(t, preview.Partial)
	assert.Equal(t, agent.PreviewOperationRun, previewChange(t, preview, 0).Operation)
	assert.Contains(t, preview.Summary, "is charged")

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, carrierintel.LookupDepthLite, intel.vetted.Depth)
	assert.True(t, intel.vetted.Force)
	assert.Equal(t, "Low", result.Name)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceCarrierIntelligence, policy.Resource)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceCarrier, target.Resource)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramCarrierID: intel.carrier.ID.String(), paramDepth: "Deep"})))
}

func TestVetCustomerBroker_ChecksTheCustomer(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := newVetCustomerBrokerTool(intel)
	customerID := pulid.MustNew("cus_")
	params := executeParams(map[string]any{fieldCustomerID: customerID.String()})

	preview := previewWithoutWrites(t, intel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, permission.ResourceCustomer, previewChange(t, preview, 0).Resource)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, customerID, intel.broker)
}

func TestSetCarrierMonitoring_ShowsOnlyTheCarriersThatChange(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := newSetCarrierMonitoringTool(intel)
	params := executeParams(map[string]any{
		paramCarrierIDs: []any{intel.carrier.ID.String(), pulid.MustNew("car_").String()},
		paramEnabled:    true,
	})

	preview := previewWithoutWrites(t, intel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 1)
	assert.Contains(t, preview.Summary, "1 have no DOT number")
	assert.Equal(t, "Enrolled", fieldByPath(t, previewChange(t, preview, 0), "monitoring").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, intel.enrolled.Enroll)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramCarrierIDs: []any{intel.carrier.ID.String()}})))
}

func TestMarkCarrierIntelReviewed_IsAPersonsSignOff(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := newMarkCarrierIntelReviewedTool(intel)
	params := executeParams(map[string]any{
		paramCarrierID: intel.carrier.ID.String(),
		paramNote:      "Insurance lapse was a filing delay; certificate on file",
	})

	preview := previewWithoutWrites(t, intel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Insurance lapse was a filing delay; certificate on file",
		fieldByPath(t, previewChange(t, preview, 0), "reviewNote").After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(params.Params)))
	require.NotNil(t, intel.reviewed)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
}

func TestApplyCarrierIntelSuggestions_AppliesEverySuggestionWhenNoneAreNamed(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	permissions := &fakePermissionCheck{allowed: true}
	tool := newApplyCarrierIntelSuggestionsTool(intel, permissions)
	params := executeParams(map[string]any{paramCarrierID: intel.carrier.ID.String()})

	preview := previewWithoutWrites(t, intel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "MC-2", fieldByPath(t, change, "mcNumber").After)
	assert.Equal(t, "Fort Worth", fieldByPath(t, change, "city").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.ElementsMatch(t, []carrierintel.SyncField{
		carrierintel.SyncFieldMCNumber, carrierintel.SyncFieldCity,
	}, intel.applied.Fields)
	require.NotEmpty(t, permissions.requests)
	assert.Equal(t, permission.ResourceCarrier.String(), permissions.requests[0].Resource)
	assert.Equal(t, permission.OpUpdate, permissions.requests[0].Operation)

	named := executeParams(map[string]any{
		paramCarrierID:        intel.carrier.ID.String(),
		paramSuggestionFields: []any{"city"},
	})
	require.NoError(t, tool.Execute(t.Context(), named))
	assert.Equal(t, []carrierintel.SyncField{carrierintel.SyncFieldCity}, intel.applied.Fields)

	refused := newApplyCarrierIntelSuggestionsTool(intel, &fakePermissionCheck{allowed: false})
	require.Error(t, refused.(serviceports.ToolValidator).Validate(t.Context(), params))
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramCarrierID:        intel.carrier.ID.String(),
			paramSuggestionFields: []any{"fax"},
		})))
}

func TestImportSourcedCarrier_NeedsTheCarrierCreateGrantToo(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	permissions := &fakePermissionCheck{allowed: true}
	tool := newImportSourcedCarrierTool(intel, permissions)
	params := executeParams(map[string]any{
		paramDOTNumber:        "7654321",
		paramEnrollMonitoring: true,
	})

	preview := previewWithoutWrites(t, intel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "7654321", intel.imported.DOTNumber)
	assert.True(t, intel.imported.EnrollMonitoring)
	assert.Equal(t, permission.OpCreate, permissions.requests[0].Operation)
	assert.Equal(t, permission.ResourceCarrierSourcing, tool.Policy().Resource)
	assert.Equal(t, permission.OpImport, tool.Policy().Operation)

	refused := newImportSourcedCarrierTool(intel, &fakePermissionCheck{allowed: false})
	require.Error(t, refused.Execute(t.Context(), params))
}

func TestVerifyCarrierEquipment_RecordsTheCheck(t *testing.T) {
	t.Parallel()

	intel := newFakeCarrierIntel()
	tool := newVerifyCarrierEquipmentTool(intel)
	assignmentID := pulid.MustNew("ca_")
	params := executeParams(map[string]any{
		paramCarrierAssignmentID: assignmentID.String(),
		paramUnitType:            "Tractor",
		paramVIN:                 "1xkwd49x0kj123456",
	})

	preview := previewWithoutWrites(t, intel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "1XKWD49X0KJ123456",
		fieldByPath(t, previewChange(t, preview, 0), paramVIN).After)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, assignmentID, intel.verified.CarrierAssignmentID)
	assert.Equal(t, "Match", result.Name)
	assert.Equal(t, permission.ResourceEquipmentVerification, tool.Policy().Resource)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramCarrierAssignmentID: assignmentID.String(),
			paramUnitType:            "Van",
		})))
}
