package resolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func trainingCourseFromInput(
	input *gqlmodel.TrainingCourseInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*worker.TrainingCourse, error) {
	passingScore, err := parseNullDecimalField("passingScore", input.PassingScore)
	if err != nil {
		return nil, err
	}

	entity := &worker.TrainingCourse{
		ID:                      id,
		OrganizationID:          tenantInfo.OrgID,
		BusinessUnitID:          tenantInfo.BuID,
		Code:                    strings.TrimSpace(input.Code),
		Name:                    strings.TrimSpace(input.Name),
		Description:             strings.TrimSpace(stringValue(input.Description)),
		Category:                input.Category,
		Status:                  input.Status,
		Delivery:                input.Delivery,
		ContentURL:              strings.TrimSpace(stringValue(input.ContentURL)),
		DurationMinutes:         int32(input.DurationMinutes), //nolint:gosec // bounded by validation
		PassingScore:            passingScore,
		RenewalWindowDays:       int32(input.RenewalWindowDays), //nolint:gosec // bounded by validation
		IsRequired:              input.IsRequired,
		RequiredForDriverTypes:  input.RequiredForDriverTypes,
		DueDaysAfterAssignment:  int32(input.DueDaysAfterAssignment), //nolint:gosec // bounded by validation
		RequiresAcknowledgement: input.RequiresAcknowledgement,
		SortOrder:               int32(intValue(input.SortOrder)), //nolint:gosec // small ordinal
		Version:                 int64(intValue(input.Version)),
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
	recordID, err := optionalID(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, "Training record is invalid")
	}
	workerID, err := optionalID(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError("workerId", errortypes.ErrInvalid, "Worker is invalid")
	}
	courseID, err := optionalID(input.CourseID)
	if err != nil {
		return nil, errortypes.NewValidationError("courseId", errortypes.ErrInvalid, "Course is invalid")
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError("documentId", errortypes.ErrInvalid, "Document is invalid")
	}
	score, err := parseNullDecimalField("score", input.Score)
	if err != nil {
		return nil, err
	}

	return &workertrainingservice.CompleteRequest{
		TenantInfo:  tenantInfo,
		ID:          recordID,
		WorkerID:    workerID,
		CourseID:    courseID,
		CompletedAt: int64(intValue(input.CompletedAt)),
		Score:       score,
		DocumentID:  documentID,
		Notes:       stringValue(input.Notes),
		Version:     int64(intValue(input.Version)),
		UserID:      userID,
	}, nil
}

func trainingCourseCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.TrainingCourse],
) (*gqlmodel.TrainingCourseConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.TrainingCourse, cursor string) *gqlmodel.TrainingCourseEdge {
			return &gqlmodel.TrainingCourseEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.TrainingCourseConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.TrainingCourseEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func trainingCategoryString(category *worker.TrainingCategory) string {
	if category == nil {
		return ""
	}
	return string(*category)
}

func nullDecimalPtr(value decimal.NullDecimal) *string {
	if !value.Valid {
		return nil
	}
	return stringPtr(value.Decimal.StringFixed(2))
}
