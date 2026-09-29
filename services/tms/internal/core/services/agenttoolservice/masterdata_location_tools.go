package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramLocationID         = "locationId"
	paramLocationCategoryID = "locationCategoryId"
)

type locationKeeper interface {
	Get(ctx context.Context, req repositories.GetLocationByIDRequest) (*location.Location, error)
	PlanUpdate(
		ctx context.Context,
		entity *location.Location,
	) (*serviceports.RecordChange[location.Location], error)
	Update(
		ctx context.Context,
		entity *location.Location,
		actor *serviceports.RequestActor,
	) (*location.Location, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateLocationStatusRequest,
	) ([]serviceports.RecordChange[location.Location], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateLocationStatusRequest,
	) ([]*location.Location, error)
}

type locationChangeView struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	Status             string `json:"status"`
	AddressLine1       string `json:"addressLine1"`
	AddressLine2       string `json:"addressLine2,omitempty"`
	City               string `json:"city"`
	State              string `json:"state"`
	PostalCode         string `json:"postalCode"`
	LocationCategoryID string `json:"locationCategoryId"`
}

func locationChangeViewOf(entity *location.Location, states map[pulid.ID]string) any {
	return &locationChangeView{
		Code:               entity.Code,
		Name:               entity.Name,
		Description:        entity.Description,
		Status:             string(entity.Status),
		AddressLine1:       entity.AddressLine1,
		AddressLine2:       entity.AddressLine2,
		City:               entity.City,
		State:              stateName(states, entity.StateID),
		PostalCode:         entity.PostalCode,
		LocationCategoryID: entity.LocationCategoryID.String(),
	}
}

var locationRecord = &masterRecord[location.Location]{
	kind:     locationRecordEntity,
	resource: permission.ResourceLocation,
	entity:   locationRecordEntity,
	idParam:  paramLocationID,
	idsParam: "locationIds",
	supplier: "from list_locations",
	label:    func(entity *location.Location) string { return entity.Name },
	id:       func(entity *location.Location) pulid.ID { return entity.ID },
	version:  func(entity *location.Location) int64 { return entity.Version },
	detach: func(entity *location.Location) {
		entity.State = nil
		entity.LocationCategory = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	stateIDs: func(entity *location.Location) []pulid.ID { return []pulid.ID{entity.StateID} },
	view:     locationChangeViewOf,
	options: []toolpreview.Option{toolpreview.WithRefs(map[string]permission.Resource{
		paramLocationCategoryID: permission.ResourceLocationCategory,
	})},
}

func locationFields() []masterField[location.Location] {
	return []masterField[location.Location]{
		masterText(mdName, "The facility or company name at the address.",
			maxLocationNameRunes, true, func(l *location.Location) *string { return &l.Name }),
		masterText(mdDescription, "Anything drivers and dispatchers should know about it.",
			mdMaxNotes, false, func(l *location.Location) *string { return &l.Description }),
		masterText(mdAddressLine1, "The street address.", maxLocationAddressRunes, true,
			func(l *location.Location) *string { return &l.AddressLine1 }),
		masterText(mdAddressLine2, "Suite, building or dock.", maxLocationAddressRunes, false,
			func(l *location.Location) *string { return &l.AddressLine2 }),
		masterText(mdCity, "The city.", maxLocationCityRunes, true,
			func(l *location.Location) *string { return &l.City }),
		masterState(mdState, "The state.", true,
			func(l *location.Location) *pulid.ID { return &l.StateID }),
		masterText(mdPostalCode, "The ZIP code.", maxPostalCodeRunes, true,
			func(l *location.Location) *string { return &l.PostalCode }),
		masterID(paramLocationCategoryID, "What kind of place it is, from "+
			"list_location_categories.",
			func(l *location.Location) *pulid.ID { return &l.LocationCategoryID }),
	}
}

func newUpdateLocationTool(locations locationKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[location.Location]{
		record: locationRecord,
		name:   "update_location",
		description: "Change a location's name, description, address or category. Fields " +
			"left out keep their value, and its code never changes. Use " +
			"update_location_status to activate or retire it.",
		rationale: "Changes a location inside Trenova; nothing is sent. Its address is often " +
			"read from a document someone outside sent, so such a change is proposed.",
		fields:      locationFields(),
		searchTerms: []string{"edit location", "facility address", "fix address", "dock"},
		policy:      mdInternalAsk,
		states:      states,
		get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*location.Location, error) {
			return locations.Get(ctx, repositories.GetLocationByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		plan:   locations.PlanUpdate,
		update: locations.Update,
	})
}

func newUpdateLocationStatusTool(
	locations locationKeeper,
	states stateLookup,
) serviceports.AgentTool {
	return newMasterStatusTool(&masterStatusSpec[location.Location, domaintypes.Status]{
		record: locationRecord,
		name:   "update_location_status",
		description: "Set one or more locations Active or Inactive, which decides whether " +
			"stops can be booked at them. Get the ids from list_locations.",
		rationale:   "Changes whether locations are offered for booking inside Trenova; nothing is sent.",
		searchTerms: []string{"deactivate location", "close facility", "reactivate location"},
		statuses:    activeStatuses,
		statusNote:  activeStatusNote,
		policy:      mdInternalStatus,
		states:      states,
		plan: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) ([]serviceports.RecordChange[location.Location], error) {
			return locations.PlanBulkUpdateStatus(ctx,
				&repositories.BulkUpdateLocationStatusRequest{
					TenantInfo:  tenant,
					LocationIDs: ids,
					Status:      status,
				})
		},
		run: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) ([]*location.Location, error) {
			return locations.BulkUpdateStatus(ctx, &repositories.BulkUpdateLocationStatusRequest{
				TenantInfo:  tenant,
				LocationIDs: ids,
				Status:      status,
			})
		},
	})
}
