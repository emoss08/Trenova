package accountingmappingservice

import (
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

func MappingChoice(
	req *services.SetAccountingMappingRequest,
	ref *accountingsync.AccountingReferenceObject,
	at int64,
) *accountingsync.Choice {
	source := req.Source
	if source != accountingsync.MappingSourceAgent {
		source = accountingsync.MappingSourceManual
	}

	return &accountingsync.Choice{
		ExternalID:   ref.ExternalID,
		ExternalName: ref.Label(),
		Source:       source,
		Reason:       req.Reason,
		ActorID:      req.UserID,
		At:           at,
	}
}

type CreatedReference struct {
	ExternalID      string
	ExternalName    string
	RequestedSource accountingsync.MappingSource
	IntegrationType integration.Type
	ActorID         pulid.ID
	At              int64
}

func CreatedReferenceChoice(created *CreatedReference) *accountingsync.Choice {
	source := accountingsync.MappingSourceCreatedInProvider
	if created.RequestedSource == accountingsync.MappingSourceAgent {
		source = accountingsync.MappingSourceAgent
	}

	return &accountingsync.Choice{
		ExternalID:   created.ExternalID,
		ExternalName: created.ExternalName,
		Source:       source,
		Reason:       "Created in " + accountingsync.ProviderName(created.IntegrationType),
		ActorID:      created.ActorID,
		At:           created.At,
	}
}
