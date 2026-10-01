package workersafetyresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/worker"
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
	shipmentID, err := base.OptionalID(f.shipmentID)
	if err != nil {
		return errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"Shipment is invalid",
		)
	}
	documentID, err := base.OptionalID(f.documentID)
	if err != nil {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	fine, err := base.ParseNullDecimalField("fineAmount", f.fineAmount)
	if err != nil {
		return err
	}
	cost, err := base.ParseNullDecimalField("costAmount", f.costAmount)
	if err != nil {
		return err
	}

	entity.Kind = f.kind
	entity.Severity = f.severity
	entity.OccurredAt = int64(f.occurredAt)
	entity.Location = strings.TrimSpace(base.StringValue(f.location))
	entity.Description = strings.TrimSpace(f.description)
	entity.Preventable = base.BoolValue(f.preventable)
	entity.ReferenceNumber = strings.TrimSpace(base.StringValue(f.referenceNumber))
	entity.ShipmentID = shipmentID
	entity.OutOfService = base.BoolValue(f.outOfService)
	entity.FineAmount = fine
	entity.CostAmount = cost
	entity.DocumentID = documentID
	entity.PointsExpireAt = base.Int64Ptr(f.pointsExpireAt)
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
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	entity := &worker.WorkerSafetyEvent{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       workerID,
		Status:         worker.SafetyEventStatusOpen,
	}
	if err = applySafetyEventFields(entity, &safetyEventFields{
		kind:             input.Kind,
		severity:         input.Severity,
		occurredAt:       input.OccurredAt,
		location:         input.Location,
		description:      input.Description,
		preventable:      input.Preventable,
		points:           input.Points,
		pointsExpireAt:   input.PointsExpireAt,
		referenceNumber:  input.ReferenceNumber,
		shipmentID:       input.ShipmentID,
		inspectionLevel:  input.InspectionLevel,
		inspectionResult: input.InspectionResult,
		outOfService:     input.OutOfService,
		fineAmount:       input.FineAmount,
		costAmount:       input.CostAmount,
		documentID:       input.DocumentID,
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
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	points := input.Points
	entity := &worker.WorkerSafetyEvent{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Resolution:     strings.TrimSpace(base.StringValue(input.Resolution)),
		Version:        int64(input.Version),
	}
	if err = applySafetyEventFields(entity, &safetyEventFields{
		kind:             input.Kind,
		severity:         input.Severity,
		occurredAt:       input.OccurredAt,
		location:         input.Location,
		description:      input.Description,
		preventable:      input.Preventable,
		points:           &points,
		pointsExpireAt:   input.PointsExpireAt,
		referenceNumber:  input.ReferenceNumber,
		shipmentID:       input.ShipmentID,
		inspectionLevel:  input.InspectionLevel,
		inspectionResult: input.InspectionResult,
		outOfService:     input.OutOfService,
		fineAmount:       input.FineAmount,
		costAmount:       input.CostAmount,
		documentID:       input.DocumentID,
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
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	return &workersafetyservice.EventStatusRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		Resolution: base.StringValue(input.Resolution),
		Version:    int64(base.IntValue(input.Version)),
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
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	eventID, err := base.OptionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
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
	req := &workersafetyservice.IssueActionRequest{
		TenantInfo:            tenantInfo,
		WorkerID:              workerID,
		Level:                 input.Level,
		Reason:                input.Reason,
		Details:               base.StringValue(input.Details),
		OccurredAt:            base.Int64Ptr(input.OccurredAt),
		ExpiresAt:             base.Int64Ptr(input.ExpiresAt),
		SafetyEventID:         eventID,
		DocumentID:            documentID,
		RecordEmploymentEvent: base.BoolValue(input.RecordEmploymentEvent),
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
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
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
		Message:         base.StringValue(input.Message),
		OccurredAt:      int64(base.IntValue(input.OccurredAt)),
		VisibleToWorker: visible,
	}, nil
}

func disciplinaryLevelPtr(level worker.DisciplinaryLevel) *worker.DisciplinaryLevel {
	if level == worker.DisciplinaryLevelNone {
		return nil
	}
	return &level
}

func int64PtrToInt(value *int64) *int {
	if value == nil {
		return nil
	}
	out := int(*value)
	return &out
}
