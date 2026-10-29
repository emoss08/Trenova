package locationtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramLocationID = "locationId"

const paramLocationCategoryID = "locationCategoryId"

type LocationKeeper interface {
	Get(ctx context.Context, req locationports.GetLocationByIDRequest) (*location.Location, error)
	PlanUpdate(
		ctx context.Context,
		entity *location.Location,
	) (*kernelports.RecordChange[location.Location], error)
	Update(
		ctx context.Context,
		entity *location.Location,
		actor *permissionports.RequestActor,
	) (*location.Location, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *locationports.BulkUpdateLocationStatusRequest,
	) ([]kernelports.RecordChange[location.Location], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *locationports.BulkUpdateLocationStatusRequest,
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
		State:              masterdatakit.StateName(states, entity.StateID),
		PostalCode:         entity.PostalCode,
		LocationCategoryID: entity.LocationCategoryID.String(),
	}
}

const Lister = "location_list"

var locationRecord = &masterdatakit.Record[location.Location]{
	Kind:     locationRecordEntity,
	Resource: permission.ResourceLocation,
	Entity:   locationRecordEntity,
	IDParam:  paramLocationID,
	IDsParam: "locationIds",
	Lister:   Lister,
	Label:    func(entity *location.Location) string { return entity.Name },
	ID:       func(entity *location.Location) pulid.ID { return entity.ID },
	Version:  func(entity *location.Location) int64 { return entity.Version },
	Detach: func(entity *location.Location) {
		entity.State = nil
		entity.LocationCategory = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	StateIDs: func(entity *location.Location) []pulid.ID { return []pulid.ID{entity.StateID} },
	View:     locationChangeViewOf,
	Options: []toolpreview.Option{toolpreview.WithRefs(map[string]permission.Resource{
		paramLocationCategoryID: permission.ResourceLocationCategory,
	})},
}

func locationFields() []masterdatakit.Field[location.Location] {
	return []masterdatakit.Field[location.Location]{
		masterdatakit.Text(masterdatakit.MdName, "The facility or company name at the address.",
			maxLocationNameRunes, true, func(l *location.Location) *string { return &l.Name }),
		masterdatakit.Text(
			masterdatakit.MdDescription,
			"Anything drivers and dispatchers should know about it.",
			masterdatakit.MdMaxNotes,
			false,
			func(l *location.Location) *string { return &l.Description },
		),
		masterdatakit.Text(
			masterdatakit.MdAddressLine1,
			"The street address.",
			maxLocationAddressRunes,
			true,
			func(l *location.Location) *string { return &l.AddressLine1 },
		),
		masterdatakit.Text(
			masterdatakit.MdAddressLine2,
			"Suite, building or dock.",
			maxLocationAddressRunes,
			false,
			func(l *location.Location) *string { return &l.AddressLine2 },
		),
		masterdatakit.Text(masterdatakit.MdCity, "The city.", maxLocationCityRunes, true,
			func(l *location.Location) *string { return &l.City }),
		masterdatakit.State(masterdatakit.MdState, "The state.", true,
			func(l *location.Location) *pulid.ID { return &l.StateID }),
		masterdatakit.Text(masterdatakit.MdPostalCode, "The ZIP code.", maxPostalCodeRunes, true,
			func(l *location.Location) *string { return &l.PostalCode }),
		masterdatakit.ID(paramLocationCategoryID, permission.ResourceLocationCategory,
			"What kind of place it is, from location_list_categories.",
			func(l *location.Location) *pulid.ID { return &l.LocationCategoryID }),
	}
}

func NewUpdateLocationTool(
	locations LocationKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewUpdateTool(&masterdatakit.UpdateSpec[location.Location]{
		Record: locationRecord,
		Name:   "location_update",
		Description: "Change a location's name, description, address or category. Fields " +
			"left out keep their value, and its code never changes. Use " +
			"location_set_status to activate or retire it.",
		Rationale: "Changes a location inside Trenova; nothing is sent. Its address is often " +
			"read from a document someone outside sent, so such a change is proposed.",
		Fields:      locationFields(),
		SearchTerms: []string{"edit location", "facility address", "fix address", "dock"},
		Policy:      masterdatakit.MdInternalAsk,
		States:      states,
		Get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*location.Location, error) {
			return locations.Get(ctx, locationports.GetLocationByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		Plan:   locations.PlanUpdate,
		Update: locations.Update,
	})
}

func NewUpdateLocationStatusTool(
	locations LocationKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewStatusTool(
		&masterdatakit.StatusSpec[location.Location, domaintypes.Status]{
			Record: locationRecord,
			Name:   "location_set_status",
			Description: "Set one or more locations Active or Inactive, which decides whether " +
				"stops can be booked at them. Get the ids from location_list.",
			Rationale:   "Changes whether locations are offered for booking inside Trenova; nothing is sent.",
			SearchTerms: []string{"deactivate location", "close facility", "reactivate location"},
			Statuses:    masterdatakit.ActiveStatuses,
			StatusNote:  masterdatakit.ActiveStatusNote,
			Policy:      masterdatakit.MdInternalStatus,
			States:      states,
			Plan: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]kernelports.RecordChange[location.Location], error) {
				return locations.PlanBulkUpdateStatus(ctx,
					&locationports.BulkUpdateLocationStatusRequest{
						TenantInfo:  tenant,
						LocationIDs: ids,
						Status:      status,
					})
			},
			Run: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]*location.Location, error) {
				return locations.BulkUpdateStatus(
					ctx,
					&locationports.BulkUpdateLocationStatusRequest{
						TenantInfo:  tenant,
						LocationIDs: ids,
						Status:      status,
					},
				)
			},
		},
	)
}
