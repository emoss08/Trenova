package accountingmappingservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
)

func MappingChoice(
	req *accountingmappingports.SetAccountingMappingRequest,
	ref *accountingmapping.AccountingReferenceObject,
	at int64,
) *accountingmapping.Choice {
	source := req.Source
	if source != accountingmapping.MappingSourceAgent {
		source = accountingmapping.MappingSourceManual
	}

	return &accountingmapping.Choice{
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
	RequestedSource accountingmapping.MappingSource
	IntegrationType integration.Type
	ActorID         pulid.ID
	At              int64
}

func CreatedReferenceChoice(created *CreatedReference) *accountingmapping.Choice {
	source := accountingmapping.MappingSourceCreatedInProvider
	if created.RequestedSource == accountingmapping.MappingSourceAgent {
		source = accountingmapping.MappingSourceAgent
	}

	return &accountingmapping.Choice{
		ExternalID:   created.ExternalID,
		ExternalName: created.ExternalName,
		Source:       source,
		Reason:       "Created in " + accountingprovider.ProviderName(created.IntegrationType),
		ActorID:      created.ActorID,
		At:           created.At,
	}
}
