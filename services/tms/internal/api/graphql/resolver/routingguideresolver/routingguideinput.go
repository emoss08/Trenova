package routingguideresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/authctx"
)

func routingGuideFromInput(
	input *gqlmodel.RoutingGuideInput,
	authCtx *authctx.AuthContext,
) (*tender.RoutingGuide, error) {
	originLocationID, err := gqlid.PulidPtrFromOptionalString(input.OriginLocationID)
	if err != nil {
		return nil, gqlid.InvalidIDError("originLocationId", "Invalid identifier")
	}
	destinationLocationID, err := gqlid.PulidPtrFromOptionalString(input.DestinationLocationID)
	if err != nil {
		return nil, gqlid.InvalidIDError("destinationLocationId", "Invalid identifier")
	}

	entity := &tender.RoutingGuide{
		OrganizationID:        authCtx.OrganizationID,
		BusinessUnitID:        authCtx.BusinessUnitID,
		Name:                  input.Name,
		Description:           gqlscalar.StringValue(input.Description),
		Status:                input.Status,
		OriginLocationID:      originLocationID,
		DestinationLocationID: destinationLocationID,
		OriginCity:            gqlscalar.StringValue(input.OriginCity),
		OriginState:           gqlscalar.StringValue(input.OriginState),
		DestinationCity:       gqlscalar.StringValue(input.DestinationCity),
		DestinationState:      gqlscalar.StringValue(input.DestinationState),
		Entries:               make([]*tender.RoutingGuideEntry, 0, len(input.Entries)),
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	for _, entryInput := range input.Entries {
		if entryInput == nil {
			continue
		}
		entry, entryErr := routingGuideEntryFromInput(entryInput, authCtx)
		if entryErr != nil {
			return nil, entryErr
		}
		entity.Entries = append(entity.Entries, entry)
	}

	return entity, nil
}

func routingGuideEntryFromInput(
	input *gqlmodel.RoutingGuideEntryInput,
	authCtx *authctx.AuthContext,
) (*tender.RoutingGuideEntry, error) {
	entryID, err := gqlid.ParseOptionalIDField("id", input.ID)
	if err != nil {
		return nil, err
	}
	carrierID, err := gqlid.RequiredID("carrierId", input.CarrierID)
	if err != nil {
		return nil, err
	}
	rate, err := gqldecimal.ParseDecimalField("rate", input.Rate, true)
	if err != nil {
		return nil, err
	}

	entry := &tender.RoutingGuideEntry{
		ID:              entryID,
		OrganizationID:  authCtx.OrganizationID,
		BusinessUnitID:  authCtx.BusinessUnitID,
		CarrierID:       carrierID,
		Rank:            int16(input.Rank),
		RateMethod:      input.RateMethod,
		Rate:            rate,
		UseContractRate: gqlscalar.BoolValue(input.UseContractRate),
		OfferTTLSeconds: int32(input.OfferTTLSeconds),
		Channel:         input.Channel,
	}
	if input.Version != nil {
		entry.Version = int64(*input.Version)
	}

	return entry, nil
}
