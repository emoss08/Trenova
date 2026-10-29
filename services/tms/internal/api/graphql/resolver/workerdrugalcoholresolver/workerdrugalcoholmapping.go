package workerdrugalcoholresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotclearinghouse"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotrandom"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func dotTestFromInput(
	input *gqlmodel.RecordDOTTestInput,
	tenantInfo pagination.TenantInfo,
) (*workerdot.WorkerDOTTest, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	safetyEventID, err := gqlid.OptionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	drawEntryID, err := gqlid.OptionalID(input.DrawEntryID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"drawEntryId",
			errortypes.ErrInvalid,
			"Random selection is invalid",
		)
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	concentration, err := gqldecimal.ParseNullDecimalField(
		"alcoholConcentration",
		input.AlcoholConcentration,
	)
	if err != nil {
		return nil, err
	}

	entity := &workerdot.WorkerDOTTest{
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		WorkerID:             workerID,
		TestType:             input.TestType,
		Substance:            input.Substance,
		Status:               workerdot.DOTTestStatusScheduled,
		Result:               workerdot.DOTResultPending,
		IsDOT:                true,
		Reason:               gqlscalar.StringValue(input.Reason),
		ScheduledAt:          gqlscalar.Int64Ptr(input.ScheduledAt),
		CollectedAt:          gqlscalar.Int64Ptr(input.CollectedAt),
		ResultAt:             gqlscalar.Int64Ptr(input.ResultAt),
		CollectionSite:       gqlscalar.StringValue(input.CollectionSite),
		CollectorName:        gqlscalar.StringValue(input.CollectorName),
		SpecimenID:           gqlscalar.StringValue(input.SpecimenID),
		LabName:              gqlscalar.StringValue(input.LabName),
		MROName:              gqlscalar.StringValue(input.MroName),
		MROVerifiedAt:        gqlscalar.Int64Ptr(input.MroVerifiedAt),
		AlcoholConcentration: gqldecimal.NullDecimalValue(concentration),
		SafetyEventID:        safetyEventID,
		DrawEntryID:          drawEntryID,
		DocumentID:           documentID,
		Notes:                gqlscalar.StringValue(input.Notes),
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
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	concentration, err := gqldecimal.ParseNullDecimalField(
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
		ResultAt:             gqlscalar.Int64Value(input.ResultAt),
		LabName:              gqlscalar.StringValue(input.LabName),
		MROName:              gqlscalar.StringValue(input.MroName),
		MROVerifiedAt:        gqlscalar.Int64Ptr(input.MroVerifiedAt),
		AlcoholConcentration: gqldecimal.NullDecimalValue(concentration),
		Notes:                gqlscalar.StringValue(input.Notes),
		DocumentID:           documentID,
		UserID:               userID,
	}, nil
}

func dotViolationFromInput(
	input *gqlmodel.RecordDOTViolationInput,
	tenantInfo pagination.TenantInfo,
) (*workerdot.WorkerDOTViolation, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	sourceTestID, err := gqlid.OptionalID(input.SourceTestID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"sourceTestId",
			errortypes.ErrInvalid,
			"Test is invalid",
		)
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &workerdot.WorkerDOTViolation{
		OrganizationID:            tenantInfo.OrgID,
		BusinessUnitID:            tenantInfo.BuID,
		WorkerID:                  workerID,
		ViolationType:             input.ViolationType,
		OccurredAt:                int64(input.OccurredAt),
		SourceTestID:              sourceTestID,
		ReportedToClearinghouseAt: gqlscalar.Int64Ptr(input.ReportedToClearinghouseAt),
		SAPName:                   gqlscalar.StringValue(input.SapName),
		SAPReferredAt:             gqlscalar.Int64Ptr(input.SapReferredAt),
		DocumentID:                documentID,
		Notes:                     gqlscalar.StringValue(input.Notes),
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
	documentID, err := gqlid.OptionalID(input.DocumentID)
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
		SAPReferredAt:             gqlscalar.Int64Ptr(input.SapReferredAt),
		SAPEvaluationCompletedAt:  gqlscalar.Int64Ptr(input.SapEvaluationCompletedAt),
		FollowUpEndsAt:            gqlscalar.Int64Ptr(input.FollowUpEndsAt),
		ReportedToClearinghouseAt: gqlscalar.Int64Ptr(input.ReportedToClearinghouseAt),
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
) (*dotclearinghouse.WorkerClearinghouseQuery, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	entity := &dotclearinghouse.WorkerClearinghouseQuery{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		WorkerID:          workerID,
		QueryType:         input.QueryType,
		Result:            dotclearinghouse.ClearinghouseResultPending,
		ConsentObtainedAt: gqlscalar.Int64Ptr(input.ConsentObtainedAt),
		ConsentExpiresAt:  gqlscalar.Int64Ptr(input.ConsentExpiresAt),
		RequestedAt:       gqlscalar.Int64Value(input.RequestedAt),
		CompletedAt:       gqlscalar.Int64Ptr(input.CompletedAt),
		Reference:         gqlscalar.StringValue(input.Reference),
		DocumentID:        documentID,
		Notes:             gqlscalar.StringValue(input.Notes),
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
	documentID, err := gqlid.OptionalID(input.DocumentID)
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
		CompletedAt:    gqlscalar.Int64Value(input.CompletedAt),
		ViolationCount: intutils.SafeToInt32(gqlscalar.IntValue(input.ViolationCount)),
		Reference:      gqlscalar.StringValue(input.Reference),
		DocumentID:     documentID,
		Notes:          gqlscalar.StringValue(input.Notes),
		UserID:         userID,
	}, nil
}

func randomPoolFromInput(
	input *gqlmodel.DOTRandomPoolInput,
	tenantInfo pagination.TenantInfo,
) *dotrandom.DOTRandomPool {
	entity := &dotrandom.DOTRandomPool{
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		Code:                strings.TrimSpace(input.Code),
		Name:                strings.TrimSpace(input.Name),
		Description:         gqlscalar.StringValue(input.Description),
		Status:              domaintypes.StatusActive,
		Period:              input.Period,
		DrugRatePercent:     int16(input.DrugRatePercent),
		AlcoholRatePercent:  int16(input.AlcoholRatePercent),
		IncludedDriverTypes: input.IncludedDriverTypes,
		IsDefault:           gqlscalar.BoolValue(input.IsDefault),
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
	result *pagination.CursorListResult[*dotrandom.DOTRandomPool],
) (*gqlmodel.DOTRandomPoolConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
		func(node *dotrandom.DOTRandomPool, cursor string) *gqlmodel.DOTRandomPoolEdge {
			return &gqlmodel.DOTRandomPoolEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DOTRandomPoolConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.DOTRandomPoolEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}
