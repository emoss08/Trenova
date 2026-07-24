package telematicsservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const shipmentCustomFieldResource = "shipment"

type ingestFormInput struct {
	Provider     string
	SubmissionID string
	TemplateID   string
	TemplateName string
	DriverID     string
	RouteStopID  string
	SubmittedAt  int64
	Fields       []services.ProviderFormField
}

func (s *Service) ingestFormSubmission(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workersByExternalID map[string]pulid.ID,
	mappingsByTemplate map[string]*telematics.FormMapping,
	input *ingestFormInput,
) error {
	workerID := workersByExternalID[input.DriverID]

	record := &telematics.FormSubmission{
		ID:                   telematics.NewFormSubmissionID(),
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		Provider:             input.Provider,
		ProviderSubmissionID: input.SubmissionID,
		TemplateID:           input.TemplateID,
		TemplateName:         input.TemplateName,
		WorkerID:             workerID,
		SubmittedAt:          input.SubmittedAt,
		Fields:               toFieldValues(input.Fields),
		CreatedAt:            timeutils.NowUnix(),
	}

	assignment, err := s.resolveActiveAssignment(ctx, tenantInfo, pulid.Nil, workerID)
	if err == nil && assignment != nil {
		record.ShipmentMoveID = assignment.ShipmentMoveID
		if move, moveErr := s.shipmentMoveRepo.GetByID(ctx, &repositories.GetMoveByIDRequest{
			MoveID:     assignment.ShipmentMoveID,
			TenantInfo: tenantInfo,
		}); moveErr == nil && move != nil {
			record.ShipmentID = move.ShipmentID
		}
	}

	if mapping := mappingsByTemplate[input.TemplateID]; mapping != nil &&
		mapping.Enabled && !record.ShipmentID.IsNil() {
		applied, applyErr := s.applyFormMapping(ctx, tenantInfo, record, mapping)
		if applyErr != nil {
			s.l.Warn("failed to apply form mapping",
				zap.String("submissionId", input.SubmissionID),
				zap.Error(applyErr))
		} else if applied > 0 {
			now := timeutils.NowUnix()
			record.Applied = true
			record.AppliedFields = applied
			record.AppliedAt = &now
		}
	}

	inserted, err := s.repo.UpsertFormSubmission(ctx, record)
	if err != nil {
		return err
	}
	if inserted && !record.ShipmentID.IsNil() {
		s.publishInvalidation(ctx, tenantInfo, "telematicsFormSubmission")
	}
	return nil
}

func (s *Service) applyFormMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *telematics.FormSubmission,
	mapping *telematics.FormMapping,
) (int, error) {
	valuesByLabel := make(map[string]string, len(record.Fields))
	for _, field := range record.Fields {
		valuesByLabel[normalizeLabel(field.Label)] = field.Value
	}

	shipmentUpdates := make(map[string]string)
	customFieldValues := make(map[string]any)
	for _, item := range mapping.Items {
		value, ok := valuesByLabel[normalizeLabel(item.SourceFieldLabel)]
		if !ok || value == "" {
			continue
		}
		switch item.TargetKind {
		case telematics.FormMappingTargetShipmentField:
			if _, allowed := telematics.ShipmentFieldTargets[item.TargetField]; allowed {
				shipmentUpdates[item.TargetField] = value
			}
		case telematics.FormMappingTargetShipmentCustomField:
			if item.TargetCustomFieldKey != "" {
				customFieldValues[item.TargetCustomFieldKey] = value
			}
		case telematics.FormMappingTargetStopField:
		}
	}

	applied := 0
	if len(shipmentUpdates) > 0 {
		n, err := s.applyShipmentFields(ctx, tenantInfo, record.ShipmentID, shipmentUpdates)
		if err != nil {
			return applied, err
		}
		applied += n
	}
	if len(customFieldValues) > 0 && s.customFieldValues != nil {
		if multiErr := s.customFieldValues.ValidateAndSave(
			ctx,
			tenantInfo,
			shipmentCustomFieldResource,
			record.ShipmentID.String(),
			customFieldValues,
		); multiErr != nil {
			return applied, multiErr
		}
		applied += len(customFieldValues)
	}
	return applied, nil
}

func (s *Service) applyShipmentFields(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	updates map[string]string,
) (int, error) {
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return 0, err
	}

	applied := 0
	for field, value := range updates {
		if setShipmentField(entity, field, value) {
			applied++
		}
	}
	if applied == 0 {
		return 0, nil
	}

	if _, err = s.shipmentRepo.Update(ctx, entity); err != nil {
		return 0, err
	}
	return applied, nil
}

func setShipmentField(entity *shipment.Shipment, field, value string) bool {
	switch field {
	case "bol":
		entity.BOL = value
		return true
	case "temperatureMin":
		if parsed, ok := parseInt16(value); ok {
			entity.TemperatureMin = &parsed
			return true
		}
	case "temperatureMax":
		if parsed, ok := parseInt16(value); ok {
			entity.TemperatureMax = &parsed
			return true
		}
	case "pieces":
		if parsed, ok := parseInt64(value); ok {
			entity.Pieces = &parsed
			return true
		}
	case "weight":
		if parsed, ok := parseInt64(value); ok {
			entity.Weight = &parsed
			return true
		}
	}
	return false
}

func toFieldValues(fields []services.ProviderFormField) []telematics.FormFieldValue {
	out := make([]telematics.FormFieldValue, 0, len(fields))
	for _, field := range fields {
		out = append(out, telematics.FormFieldValue{
			Label: field.Label,
			Type:  field.Type,
			Value: field.Value,
		})
	}
	return out
}

func normalizeLabel(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func parseInt16(value string) (int16, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, false
	}
	return int16(parsed), true
}

func parseInt64(value string) (int64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, false
	}
	return int64(parsed), true
}
