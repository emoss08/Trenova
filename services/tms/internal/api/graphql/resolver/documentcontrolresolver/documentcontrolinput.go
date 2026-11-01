package documentcontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
)

func documentControlFromInput(
	existing *documentcontrol.DocumentControl,
	input *gqlmodel.UpdateDocumentControlInput,
) (*documentcontrol.DocumentControl, error) {
	parse := gqlinput.NewParser()
	entity := &documentcontrol.DocumentControl{
		ID:                              existing.ID,
		BusinessUnitID:                  existing.BusinessUnitID,
		OrganizationID:                  existing.OrganizationID,
		Version:                         int64(input.Version),
		CreatedAt:                       existing.CreatedAt,
		UpdatedAt:                       existing.UpdatedAt,
		EnableDocumentIntelligence:      input.EnableDocumentIntelligence,
		EnableOCR:                       input.EnableOcr,
		EnableAutoClassification:        input.EnableAutoClassification,
		EnableAutoDocumentTypeAssociate: input.EnableAutoDocumentTypeAssociate,
		EnableAutoCreateDocumentTypes:   input.EnableAutoCreateDocumentTypes,
		EnableShipmentDraftExtraction:   input.EnableShipmentDraftExtraction,
		EnableAIAssistedClassification:  input.EnableAiAssistedClassification,
		EnableAIAssistedExtraction:      input.EnableAiAssistedExtraction,
		ShipmentDraftAllowedResources:   input.ShipmentDraftAllowedResources,
		EnableFullTextIndexing:          input.EnableFullTextIndexing,
		EnableCapture:                   input.EnableCapture,
		CaptureAutoFileCoverSheets:      input.CaptureAutoFileCoverSheets,
		CaptureRetentionDays:            input.CaptureRetentionDays,
		CaptureMinAgentVersion:          input.CaptureMinAgentVersion,
		CaptureAllowAutoUpdate:          input.CaptureAllowAutoUpdate,
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return entity, nil
}
