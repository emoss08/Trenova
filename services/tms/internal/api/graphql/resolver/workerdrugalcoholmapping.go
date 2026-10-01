package resolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func dotTestFromInput(
	input *gqlmodel.RecordDOTTestInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerDOTTest, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	safetyEventID, err := optionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	drawEntryID, err := optionalID(input.DrawEntryID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"drawEntryId",
			errortypes.ErrInvalid,
			"Random selection is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	concentration, err := parseNullDecimalField("alcoholConcentration", input.AlcoholConcentration)
	if err != nil {
		return nil, err
	}

	entity := &worker.WorkerDOTTest{
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		WorkerID:             workerID,
		TestType:             input.TestType,
		Substance:            input.Substance,
		Status:               worker.DOTTestStatusScheduled,
		Result:               worker.DOTResultPending,
		IsDOT:                true,
		Reason:               stringValue(input.Reason),
		ScheduledAt:          int64Ptr(input.ScheduledAt),
		CollectedAt:          int64Ptr(input.CollectedAt),
		ResultAt:             int64Ptr(input.ResultAt),
		CollectionSite:       stringValue(input.CollectionSite),
		CollectorName:        stringValue(input.CollectorName),
		SpecimenID:           stringValue(input.SpecimenID),
		LabName:              stringValue(input.LabName),
		MROName:              stringValue(input.MroName),
		MROVerifiedAt:        int64Ptr(input.MroVerifiedAt),
		AlcoholConcentration: nullDecimalValue(concentration),
		SafetyEventID:        safetyEventID,
		DrawEntryID:          drawEntryID,
		DocumentID:           documentID,
		Notes:                stringValue(input.Notes),
	}
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if input.Result != nil {
		entity.Result = *input.Result
	}
	if input.IsDOT != nil {
		entity.IsDOT = *input.IsDOT
	}

	return entity, nil
}

func dotTestResultRequest(
	input *gqlmodel.RecordDOTTestResultInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerdrugalcoholservice.RecordResultRequest, error) {
	testID, err := pulid.MustParse(input.TestID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"testId",
			errortypes.ErrInvalid,
			"Test is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	concentration, err := parseNullDecimalField("alcoholConcentration", input.AlcoholConcentration)
	if err != nil {
		return nil, err
	}

	return &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo:           tenantInfo,
		TestID:               testID,
		Result:               input.Result,
		ResultAt:             int64Value(input.ResultAt),
		LabName:              stringValue(input.LabName),
		MROName:              stringValue(input.MroName),
		MROVerifiedAt:        int64Ptr(input.MroVerifiedAt),
		AlcoholConcentration: nullDecimalValue(concentration),
		Notes:                stringValue(input.Notes),
		DocumentID:           documentID,
		UserID:               userID,
	}, nil
}

func dotViolationFromInput(
	input *gqlmodel.RecordDOTViolationInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerDOTViolation, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	sourceTestID, err := optionalID(input.SourceTestID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"sourceTestId",
			errortypes.ErrInvalid,
			"Test is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &worker.WorkerDOTViolation{
		OrganizationID:            tenantInfo.OrgID,
		BusinessUnitID:            tenantInfo.BuID,
		WorkerID:                  workerID,
		ViolationType:             input.ViolationType,
		OccurredAt:                int64(input.OccurredAt),
		SourceTestID:              sourceTestID,
		ReportedToClearinghouseAt: int64Ptr(input.ReportedToClearinghouseAt),
		SAPName:                   stringValue(input.SapName),
		SAPReferredAt:             int64Ptr(input.SapReferredAt),
		DocumentID:                documentID,
		Notes:                     stringValue(input.Notes),
	}, nil
}

func dotViolationUpdateRequest(
	input *gqlmodel.UpdateDOTViolationInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerdrugalcoholservice.UpdateViolationRequest, error) {
	violationID, err := pulid.MustParse(input.ViolationID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"violationId",
			errortypes.ErrInvalid,
			"Violation is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	req := &workerdrugalcoholservice.UpdateViolationRequest{
		TenantInfo:                tenantInfo,
		ViolationID:               violationID,
		SAPName:                   input.SapName,
		SAPReferredAt:             int64Ptr(input.SapReferredAt),
		SAPEvaluationCompletedAt:  int64Ptr(input.SapEvaluationCompletedAt),
		FollowUpEndsAt:            int64Ptr(input.FollowUpEndsAt),
		ReportedToClearinghouseAt: int64Ptr(input.ReportedToClearinghouseAt),
		DocumentID:                documentID,
		Notes:                     input.Notes,
		UserID:                    userID,
	}
	if input.FollowUpTestCount != nil {
		count := int32(*input.FollowUpTestCount)
		req.FollowUpTestCount = &count
	}

	return req, nil
}

func clearinghouseQueryFromInput(
	input *gqlmodel.RecordClearinghouseQueryInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerClearinghouseQuery, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	entity := &worker.WorkerClearinghouseQuery{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		WorkerID:          workerID,
		QueryType:         input.QueryType,
		Result:            worker.ClearinghouseResultPending,
		ConsentObtainedAt: int64Ptr(input.ConsentObtainedAt),
		ConsentExpiresAt:  int64Ptr(input.ConsentExpiresAt),
		RequestedAt:       int64Value(input.RequestedAt),
		CompletedAt:       int64Ptr(input.CompletedAt),
		Reference:         stringValue(input.Reference),
		DocumentID:        documentID,
		Notes:             stringValue(input.Notes),
	}
	if input.Result != nil {
		entity.Result = *input.Result
	}
	if input.ViolationCount != nil {
		entity.ViolationCount = int32(*input.ViolationCount)
	}

	return entity, nil
}

func completeQueryRequest(
	input *gqlmodel.CompleteClearinghouseQueryInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerdrugalcoholservice.CompleteQueryRequest, error) {
	queryID, err := pulid.MustParse(input.QueryID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"queryId",
			errortypes.ErrInvalid,
			"Query is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &workerdrugalcoholservice.CompleteQueryRequest{
		TenantInfo:     tenantInfo,
		QueryID:        queryID,
		Result:         input.Result,
		CompletedAt:    int64Value(input.CompletedAt),
		ViolationCount: int32(intValue(input.ViolationCount)),
		Reference:      stringValue(input.Reference),
		DocumentID:     documentID,
		Notes:          stringValue(input.Notes),
		UserID:         userID,
	}, nil
}

func randomPoolFromInput(
	input *gqlmodel.DOTRandomPoolInput,
	tenantInfo pagination.TenantInfo,
) *worker.DOTRandomPool {
	entity := &worker.DOTRandomPool{
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		Code:                strings.TrimSpace(input.Code),
		Name:                strings.TrimSpace(input.Name),
		Description:         stringValue(input.Description),
		Status:              domaintypes.StatusActive,
		Period:              input.Period,
		DrugRatePercent:     int16(input.DrugRatePercent),
		AlcoholRatePercent:  int16(input.AlcoholRatePercent),
		IncludedDriverTypes: input.IncludedDriverTypes,
		IsDefault:           boolValue(input.IsDefault),
	}
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if entity.IncludedDriverTypes == nil {
		entity.IncludedDriverTypes = []string{}
	}

	return entity
}

func randomPoolCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.DOTRandomPool],
) (*gqlmodel.DOTRandomPoolConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.DOTRandomPool, cursor string) *gqlmodel.DOTRandomPoolEdge {
			return &gqlmodel.DOTRandomPoolEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DOTRandomPoolConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.DOTRandomPoolEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func decimalPtrString(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	rendered := value.String()
	return &rendered
}

// nullDecimalValue unwraps an optional decimal into the pointer the domain
// models use for a column that may be absent. Its sibling nullDecimalPtr
// renders one for the wire, which is the opposite direction.
func nullDecimalValue(value decimal.NullDecimal) *decimal.Decimal {
	if !value.Valid {
		return nil
	}
	amount := value.Decimal
	return &amount
}
