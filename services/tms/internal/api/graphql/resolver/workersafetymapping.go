package resolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type safetyEventFields struct {
	kind             worker.SafetyEventKind
	severity         worker.SafetySeverity
	occurredAt       int
	location         *string
	description      string
	preventable      *bool
	points           *int
	pointsExpireAt   *int
	referenceNumber  *string
	shipmentID       *string
	inspectionLevel  *int
	inspectionResult *worker.InspectionResult
	outOfService     *bool
	fineAmount       *string
	costAmount       *string
	documentID       *string
}

func applySafetyEventFields(entity *worker.WorkerSafetyEvent, f *safetyEventFields) error {
	shipmentID, err := optionalID(f.shipmentID)
	if err != nil {
		return errortypes.NewValidationError("shipmentId", errortypes.ErrInvalid, "Shipment is invalid")
	}
	documentID, err := optionalID(f.documentID)
	if err != nil {
		return errortypes.NewValidationError("documentId", errortypes.ErrInvalid, "Document is invalid")
	}
	fine, err := parseNullDecimalField("fineAmount", f.fineAmount)
	if err != nil {
		return err
	}
	cost, err := parseNullDecimalField("costAmount", f.costAmount)
	if err != nil {
		return err
	}

	entity.Kind = f.kind
	entity.Severity = f.severity
	entity.OccurredAt = int64(f.occurredAt)
	entity.Location = strings.TrimSpace(stringValue(f.location))
	entity.Description = strings.TrimSpace(f.description)
	entity.Preventable = boolValue(f.preventable)
	entity.ReferenceNumber = strings.TrimSpace(stringValue(f.referenceNumber))
	entity.ShipmentID = shipmentID
	entity.OutOfService = boolValue(f.outOfService)
	entity.FineAmount = fine
	entity.CostAmount = cost
	entity.DocumentID = documentID
	entity.PointsExpireAt = int64Ptr(f.pointsExpireAt)
	if f.inspectionResult != nil {
		entity.InspectionResult = *f.inspectionResult
	} else {
		entity.InspectionResult = worker.InspectionResultNone
	}
	if f.inspectionLevel != nil {
		level := int16(*f.inspectionLevel) //nolint:gosec // bounded by validation
		entity.InspectionLevel = &level
	} else {
		entity.InspectionLevel = nil
	}
	if f.points != nil {
		entity.Points = int32(*f.points) //nolint:gosec // bounded by validation
	} else {
		entity.Points = worker.DefaultSafetyPoints(
			entity.Kind,
			entity.Severity,
			entity.Preventable,
			entity.InspectionResult,
		)
	}
	return nil
}

func safetyEventFromInput(
	input *gqlmodel.WorkerSafetyEventInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerSafetyEvent, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError("workerId", errortypes.ErrInvalid, "Worker is invalid")
	}
	entity := &worker.WorkerSafetyEvent{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       workerID,
		Status:         worker.SafetyEventStatusOpen,
	}
	if err = applySafetyEventFields(entity, &safetyEventFields{
		kind: input.Kind, severity: input.Severity, occurredAt: input.OccurredAt,
		location: input.Location, description: input.Description, preventable: input.Preventable,
		points: input.Points, pointsExpireAt: input.PointsExpireAt, referenceNumber: input.ReferenceNumber,
		shipmentID: input.ShipmentID, inspectionLevel: input.InspectionLevel,
		inspectionResult: input.InspectionResult, outOfService: input.OutOfService,
		fineAmount: input.FineAmount, costAmount: input.CostAmount, documentID: input.DocumentID,
	}); err != nil {
		return nil, err
	}
	return entity, nil
}

func safetyEventFromUpdateInput(
	input *gqlmodel.UpdateWorkerSafetyEventInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerSafetyEvent, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, "Safety event is invalid")
	}
	points := input.Points
	entity := &worker.WorkerSafetyEvent{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Resolution:     strings.TrimSpace(stringValue(input.Resolution)),
		Version:        int64(input.Version),
	}
	if err = applySafetyEventFields(entity, &safetyEventFields{
		kind: input.Kind, severity: input.Severity, occurredAt: input.OccurredAt,
		location: input.Location, description: input.Description, preventable: input.Preventable,
		points: &points, pointsExpireAt: input.PointsExpireAt, referenceNumber: input.ReferenceNumber,
		shipmentID: input.ShipmentID, inspectionLevel: input.InspectionLevel,
		inspectionResult: input.InspectionResult, outOfService: input.OutOfService,
		fineAmount: input.FineAmount, costAmount: input.CostAmount, documentID: input.DocumentID,
	}); err != nil {
		return nil, err
	}
	return entity, nil
}

func safetyEventStatusRequest(
	input *gqlmodel.SafetyEventStatusInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workersafetyservice.EventStatusRequest, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, "Safety event is invalid")
	}
	return &workersafetyservice.EventStatusRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		Resolution: stringValue(input.Resolution),
		Version:    int64(intValue(input.Version)),
		UserID:     userID,
	}, nil
}

func issueActionRequestFromInput(
	input *gqlmodel.IssueDisciplinaryActionInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workersafetyservice.IssueActionRequest, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError("workerId", errortypes.ErrInvalid, "Worker is invalid")
	}
	eventID, err := optionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError("safetyEventId", errortypes.ErrInvalid, "Safety event is invalid")
	}
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError("documentId", errortypes.ErrInvalid, "Document is invalid")
	}
	req := &workersafetyservice.IssueActionRequest{
		TenantInfo:            tenantInfo,
		WorkerID:              workerID,
		Level:                 input.Level,
		Reason:                input.Reason,
		Details:               stringValue(input.Details),
		OccurredAt:            int64Ptr(input.OccurredAt),
		ExpiresAt:             int64Ptr(input.ExpiresAt),
		SafetyEventID:         eventID,
		DocumentID:            documentID,
		RecordEmploymentEvent: boolValue(input.RecordEmploymentEvent),
		UserID:                userID,
	}
	if input.SuspensionDays != nil {
		days := int32(*input.SuspensionDays) //nolint:gosec // bounded by validation
		req.SuspensionDays = &days
	}
	return req, nil
}

func recognitionFromInput(
	input *gqlmodel.WorkerRecognitionInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerRecognition, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError("workerId", errortypes.ErrInvalid, "Worker is invalid")
	}
	visible := true
	if input.VisibleToWorker != nil {
		visible = *input.VisibleToWorker
	}
	return &worker.WorkerRecognition{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		WorkerID:        workerID,
		Kind:            input.Kind,
		Title:           input.Title,
		Message:         stringValue(input.Message),
		OccurredAt:      int64(intValue(input.OccurredAt)),
		VisibleToWorker: visible,
	}, nil
}

func reviewTemplateFromInput(
	input *gqlmodel.PerformanceReviewTemplateInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *worker.PerformanceReviewTemplate {
	items := make([]worker.ReviewItem, 0, len(input.Items))
	for _, item := range input.Items {
		if item == nil {
			continue
		}
		items = append(items, worker.ReviewItem{
			Key:         strings.TrimSpace(item.Key),
			Label:       strings.TrimSpace(item.Label),
			Description: strings.TrimSpace(stringValue(item.Description)),
			Weight:      int32(item.Weight), //nolint:gosec // bounded by validation
		})
	}
	entity := &worker.PerformanceReviewTemplate{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           strings.TrimSpace(input.Code),
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(stringValue(input.Description)),
		Status:         input.Status,
		IsDefault:      input.IsDefault,
		Items:          items,
		Version:        int64(intValue(input.Version)),
	}
	if input.CadenceMonths != nil {
		months := int32(*input.CadenceMonths) //nolint:gosec // bounded by validation
		entity.CadenceMonths = &months
	}
	return entity
}

func updateReviewRequestFromInput(
	input *gqlmodel.UpdatePerformanceReviewInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*performancereviewservice.UpdateReviewRequest, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, "Review is invalid")
	}
	ratings := make([]worker.ReviewRating, 0, len(input.Ratings))
	for _, rating := range input.Ratings {
		if rating == nil {
			continue
		}
		entry := worker.ReviewRating{Key: rating.Key, Comment: stringValue(rating.Comment)}
		if rating.Score != nil {
			score := int32(*rating.Score) //nolint:gosec // bounded by validation
			entry.Score = &score
		}
		ratings = append(ratings, entry)
	}
	goals := make([]worker.ReviewGoal, 0, len(input.Goals))
	for _, goal := range input.Goals {
		if goal == nil {
			continue
		}
		entry := worker.ReviewGoal{
			ID:    stringValue(goal.ID),
			Title: goal.Title,
			DueAt: int64Ptr(goal.DueAt),
		}
		if goal.Status != nil {
			entry.Status = *goal.Status
		}
		goals = append(goals, entry)
	}
	return &performancereviewservice.UpdateReviewRequest{
		TenantInfo:   tenantInfo,
		ID:           id,
		Title:        stringValue(input.Title),
		PeriodStart:  int64(intValue(input.PeriodStart)),
		PeriodEnd:    int64(intValue(input.PeriodEnd)),
		Ratings:      ratings,
		Summary:      stringValue(input.Summary),
		Strengths:    stringValue(input.Strengths),
		Improvements: stringValue(input.Improvements),
		Goals:        goals,
		Version:      int64(input.Version),
		UserID:       userID,
	}, nil
}

func reviewStatusRequestFromInput(
	input *gqlmodel.PerformanceReviewStatusInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*performancereviewservice.ReviewStatusRequest, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, "Review is invalid")
	}
	return &performancereviewservice.ReviewStatusRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		Version:    int64(intValue(input.Version)),
		UserID:     userID,
	}, nil
}

func reviewTemplateCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.PerformanceReviewTemplate],
) (*gqlmodel.PerformanceReviewTemplateConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.PerformanceReviewTemplate, cursor string) *gqlmodel.PerformanceReviewTemplateEdge {
			return &gqlmodel.PerformanceReviewTemplateEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.PerformanceReviewTemplateConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.PerformanceReviewTemplateEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func disciplinaryLevelPtr(level worker.DisciplinaryLevel) *worker.DisciplinaryLevel {
	if level == worker.DisciplinaryLevelNone {
		return nil
	}
	return &level
}

func int32Ptr(value *int32) *int {
	if value == nil {
		return nil
	}
	out := int(*value)
	return &out
}

func int64PtrToInt(value *int64) *int {
	if value == nil {
		return nil
	}
	out := int(*value)
	return &out
}
