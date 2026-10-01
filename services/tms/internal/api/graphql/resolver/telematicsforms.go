package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
)

func mapTelematicsFormSubmission(
	record *telematicsservice.FormSubmissionRecord,
) *gqlmodel.TelematicsFormSubmission {
	submission := record.FormSubmission
	out := &gqlmodel.TelematicsFormSubmission{
		ID:            submission.ID.String(),
		Provider:      submission.Provider,
		TemplateID:    submission.TemplateID,
		TemplateName:  submission.TemplateName,
		WorkerName:    record.WorkerName,
		SubmittedAt:   int(submission.SubmittedAt),
		Applied:       submission.Applied,
		AppliedFields: submission.AppliedFields,
		Fields:        make([]*gqlmodel.TelematicsFormFieldValue, 0, len(submission.Fields)),
	}
	if !submission.WorkerID.IsNil() {
		workerID := submission.WorkerID.String()
		out.WorkerID = &workerID
	}
	if !submission.ShipmentID.IsNil() {
		shipmentID := submission.ShipmentID.String()
		out.ShipmentID = &shipmentID
	}
	if !submission.StopID.IsNil() {
		stopID := submission.StopID.String()
		out.StopID = &stopID
	}
	for _, field := range submission.Fields {
		out.Fields = append(out.Fields, &gqlmodel.TelematicsFormFieldValue{
			Label: field.Label,
			Type:  field.Type,
			Value: field.Value,
		})
	}
	return out
}

func mapTelematicsFormMapping(mapping *telematics.FormMapping) *gqlmodel.TelematicsFormMapping {
	out := &gqlmodel.TelematicsFormMapping{
		ID:           mapping.ID.String(),
		Provider:     mapping.Provider,
		TemplateID:   mapping.TemplateID,
		TemplateName: mapping.TemplateName,
		Name:         mapping.Name,
		Description:  mapping.Description,
		Enabled:      mapping.Enabled,
		Version:      int(mapping.Version),
		Items:        make([]*gqlmodel.TelematicsFormMappingItem, 0, len(mapping.Items)),
	}
	for _, item := range mapping.Items {
		out.Items = append(out.Items, &gqlmodel.TelematicsFormMappingItem{
			ID:                   item.ID.String(),
			SourceFieldLabel:     item.SourceFieldLabel,
			TargetKind:           string(item.TargetKind),
			TargetField:          item.TargetField,
			TargetCustomFieldKey: item.TargetCustomFieldKey,
		})
	}
	return out
}
