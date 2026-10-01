package workerdrugalcoholresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
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
	safetyEventID, err := base.OptionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	drawEntryID, err := base.OptionalID(input.DrawEntryID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"drawEntryId",
			errortypes.ErrInvalid,
			"Random selection is invalid",
		)
	}
	documentID, err := base.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	concentration, err := base.ParseNullDecimalField(
		"alcoholConcentration",
		input.AlcoholConcentration,
	)
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
		Reason:               base.StringValue(input.Reason),
		ScheduledAt:          base.Int64Ptr(input.ScheduledAt),
		CollectedAt:          base.Int64Ptr(input.CollectedAt),
		ResultAt:             base.Int64Ptr(input.ResultAt),
		CollectionSite:       base.StringValue(input.CollectionSite),
		CollectorName:        base.StringValue(input.CollectorName),
		SpecimenID:           base.StringValue(input.SpecimenID),
		LabName:              base.StringValue(input.LabName),
		MROName:              base.StringValue(input.MroName),
		MROVerifiedAt:        base.Int64Ptr(input.MroVerifiedAt),
		AlcoholConcentration: base.NullDecimalValue(concentration),
		SafetyEventID:        safetyEventID,
		DrawEntryID:          drawEntryID,
		DocumentID:           documentID,
		Notes:                base.StringValue(input.Notes),
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
	documentID, err := base.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	concentration, err := base.ParseNullDecimalField(
		"alcoholConcentration",
		input.AlcoholConcentration,
	)
	if err != nil {
		return nil, err
	}

	return &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo:           tenantInfo,
		TestID:               testID,
		Result:               input.Result,
		ResultAt:             base.Int64Value(input.ResultAt),
		LabName:              base.StringValue(input.LabName),
		MROName:              base.StringValue(input.MroName),
		MROVerifiedAt:        base.Int64Ptr(input.MroVerifiedAt),
		AlcoholConcentration: base.NullDecimalValue(concentration),
		Notes:                base.StringValue(input.Notes),
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
	sourceTestID, err := base.OptionalID(input.SourceTestID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"sourceTestId",
			errortypes.ErrInvalid,
			"Test is invalid",
		)
	}
	documentID, err := base.OptionalID(input.DocumentID)
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
		ReportedToClearinghouseAt: base.Int64Ptr(input.ReportedToClearinghouseAt),
		SAPName:                   base.StringValue(input.SapName),
		SAPReferredAt:             base.Int64Ptr(input.SapReferredAt),
		DocumentID:                documentID,
		Notes:                     base.StringValue(input.Notes),
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
	documentID, err := base.OptionalID(input.DocumentID)
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
		SAPReferredAt:             base.Int64Ptr(input.SapReferredAt),
		SAPEvaluationCompletedAt:  base.Int64Ptr(input.SapEvaluationCompletedAt),
		FollowUpEndsAt:            base.Int64Ptr(input.FollowUpEndsAt),
		ReportedToClearinghouseAt: base.Int64Ptr(input.ReportedToClearinghouseAt),
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
	documentID, err := base.OptionalID(input.DocumentID)
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
		ConsentObtainedAt: base.Int64Ptr(input.ConsentObtainedAt),
		ConsentExpiresAt:  base.Int64Ptr(input.ConsentExpiresAt),
		RequestedAt:       base.Int64Value(input.RequestedAt),
		CompletedAt:       base.Int64Ptr(input.CompletedAt),
		Reference:         base.StringValue(input.Reference),
		DocumentID:        documentID,
		Notes:             base.StringValue(input.Notes),
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
	documentID, err := base.OptionalID(input.DocumentID)
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
		CompletedAt:    base.Int64Value(input.CompletedAt),
		ViolationCount: intutils.SafeToInt32(base.IntValue(input.ViolationCount)),
		Reference:      base.StringValue(input.Reference),
		DocumentID:     documentID,
		Notes:          base.StringValue(input.Notes),
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
		Description:         base.StringValue(input.Description),
		Status:              domaintypes.StatusActive,
		Period:              input.Period,
		DrugRatePercent:     int16(input.DrugRatePercent),
		AlcoholRatePercent:  int16(input.AlcoholRatePercent),
		IncludedDriverTypes: input.IncludedDriverTypes,
		IsDefault:           base.BoolValue(input.IsDefault),
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
	edges, err := base.EntityCursorEdges(
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
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.DOTRandomPoolEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}
