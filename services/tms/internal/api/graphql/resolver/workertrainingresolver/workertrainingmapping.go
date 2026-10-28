package workertrainingresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func trainingCourseFromInput(
	input *gqlmodel.TrainingCourseInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*workertraining.TrainingCourse, error) {
	passingScore, err := gqldecimal.ParseNullDecimalField("passingScore", input.PassingScore)
	if err != nil {
		return nil, err
	}

	entity := &workertraining.TrainingCourse{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           strings.TrimSpace(input.Code),
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(gqlscalar.StringValue(input.Description)),
		Category:       input.Category,
		Status:         input.Status,
		Delivery:       input.Delivery,
		ContentURL:     strings.TrimSpace(gqlscalar.StringValue(input.ContentURL)),
		DurationMinutes: int32(
			input.DurationMinutes,
		), //nolint:gosec // bounded by validation
		PassingScore: passingScore,
		RenewalWindowDays: int32(
			input.RenewalWindowDays,
		), //nolint:gosec // bounded by validation
		IsRequired:             input.IsRequired,
		RequiredForDriverTypes: input.RequiredForDriverTypes,
		DueDaysAfterAssignment: int32(
			input.DueDaysAfterAssignment,
		), //nolint:gosec // bounded by validation
		RequiresAcknowledgement: input.RequiresAcknowledgement,
		SortOrder:               intutils.SafeToInt32(gqlscalar.IntValue(input.SortOrder)),
		Version:                 int64(gqlscalar.IntValue(input.Version)),
	}
	if entity.RequiredForDriverTypes == nil {
		entity.RequiredForDriverTypes = []worker.DriverType{}
	}
	if !entity.IsRequired {
		entity.RequiredForDriverTypes = []worker.DriverType{}
	}
	if input.ValidityMonths != nil {
		months := int32(*input.ValidityMonths) //nolint:gosec // bounded by validation
		entity.ValidityMonths = &months
	}
	return entity, nil
}

func completeTrainingRequestFromInput(
	input *gqlmodel.CompleteWorkerTrainingInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workertrainingservice.CompleteRequest, error) {
	recordID, err := gqlid.OptionalID(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"Training record is invalid",
		)
	}
	workerID, err := gqlid.OptionalID(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	courseID, err := gqlid.OptionalID(input.CourseID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"courseId",
			errortypes.ErrInvalid,
			"Course is invalid",
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
	score, err := gqldecimal.ParseNullDecimalField("score", input.Score)
	if err != nil {
		return nil, err
	}

	return &workertrainingservice.CompleteRequest{
		TenantInfo:  tenantInfo,
		ID:          recordID,
		WorkerID:    workerID,
		CourseID:    courseID,
		CompletedAt: int64(gqlscalar.IntValue(input.CompletedAt)),
		Score:       score,
		DocumentID:  documentID,
		Notes:       gqlscalar.StringValue(input.Notes),
		Version:     int64(gqlscalar.IntValue(input.Version)),
		UserID:      userID,
	}, nil
}

func trainingCourseCursorConnectionToModel(
	result *pagination.CursorListResult[*workertraining.TrainingCourse],
) (*gqlmodel.TrainingCourseConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
		func(node *workertraining.TrainingCourse, cursor string) *gqlmodel.TrainingCourseEdge {
			return &gqlmodel.TrainingCourseEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.TrainingCourseConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.TrainingCourseEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func trainingCategoryString(category *workertraining.TrainingCategory) string {
	if category == nil {
		return ""
	}
	return string(*category)
}
