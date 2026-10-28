package masterdatatools

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const locationRecordEntity = "location"

const maxLocationNameRunes = 100

const maxLocationAddressRunes = 150

const maxLocationCityRunes = 100

const maxPostalCodeRunes = 10

type locationCreator interface {
	Create(
		ctx context.Context,
		entity *location.Location,
		actor *permissionports.RequestActor,
	) (*location.Location, error)
}

type stateReader interface {
	GetByAbbreviation(ctx context.Context, abbreviation string) (*usstate.UsState, error)
}

type locationCategoryReader interface {
	GetByID(
		ctx context.Context,
		req locationcategoryports.GetLocationCategoryByIDRequest,
	) (*locationcategory.LocationCategory, error)
}

type CreateLocationTool struct {
	locations  locationCreator
	states     stateReader
	categories locationCategoryReader
	places     placeFinder
}

type CreateLocationToolParams struct {
	Locations  locationCreator
	States     stateReader
	Categories locationCategoryReader
	Places     placeFinder
}

func NewCreateLocationTool(p CreateLocationToolParams) agenttoolports.AgentTool {
	return &CreateLocationTool{
		locations:  p.Locations,
		states:     p.States,
		categories: p.Categories,
		places:     p.Places,
	}
}

func (t *CreateLocationTool) Name() string { return agentpreset.ToolCreateLocation }

func (t *CreateLocationTool) Description() string {
	return "Create a location record for a pickup or delivery address that has none, such " +
		"as a facility read from a document. Look for it with location_list first; a " +
		"person approves the new location before it is saved, and the code is assigned."
}

func (t *CreateLocationTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The facility or company name at the address.",
			},
			"addressLine1": map[string]any{"type": "string", "description": "The street address."},
			"addressLine2": map[string]any{
				"type":        "string",
				"description": "Optional: suite, building or dock.",
			},
			"city": map[string]any{"type": "string", "description": "The city."},
			"state": map[string]any{
				"type":        "string",
				"description": "The two-letter US state abbreviation, such as TX.",
			},
			"postalCode": map[string]any{"type": "string", "description": "The ZIP code."},
			"locationCategoryId": agenttoolschema.RecordIDText(permission.ResourceLocationCategory,
				"What kind of place it is, from location_list_categories: a warehouse, a "+
					"customer site, a terminal."),
		},
		"required": []string{
			"name", "addressLine1", "city", "state", "postalCode", "locationCategoryId",
		},
		"additionalProperties": false,
	}
}

func (t *CreateLocationTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceLocation,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Artifact:      locationRecordEntity,
		Rationale: "A new location is where colleagues will book freight, and its address is " +
			"usually read from a document someone outside sent, so a person approves it first.",
	}
}

func (t *CreateLocationTool) SearchTerms() []string {
	return []string{
		"new location", "add location", "facility", "address", "warehouse", "site", "consignee",
	}
}

func (t *CreateLocationTool) Prerequisites() []string {
	return []string{"location_list", "location_list_categories"}
}

type locationDraft struct {
	name         string
	addressLine1 string
	addressLine2 string
	city         string
	state        string
	postalCode   string
	categoryID   pulid.ID
}

func readLocationDraft(params map[string]any) (*locationDraft, error) {
	multiErr := errortypes.NewMultiError()
	text := func(key string, limit int, required bool) string {
		value := strings.TrimSpace(toolparams.OptionalString(params, key))
		switch {
		case value == "" && required:
			multiErr.Add(key, errortypes.ErrRequired, "This is required")
		case utf8.RuneCountInString(value) > limit:
			multiErr.Add(key, errortypes.ErrInvalid, "This is too long")
		}

		return value
	}

	draft := &locationDraft{
		name:         text("name", maxLocationNameRunes, true),
		addressLine1: text("addressLine1", maxLocationAddressRunes, true),
		addressLine2: text("addressLine2", maxLocationAddressRunes, false),
		city:         text("city", maxLocationCityRunes, true),
		state:        strings.ToUpper(text("state", 2, true)),
		postalCode:   text("postalCode", maxPostalCodeRunes, true),
	}
	if draft.state != "" && utf8.RuneCountInString(draft.state) != 2 {
		multiErr.Add("state", errortypes.ErrInvalid, "Use the two-letter state abbreviation")
	}

	categoryID, err := toolparams.RequirePulid(params, "locationCategoryId")
	if err != nil {
		multiErr.Add("locationCategoryId", errortypes.ErrInvalid,
			"Name a location category from location_list_categories")
	}
	draft.categoryID = categoryID

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return draft, nil
}

func (t *CreateLocationTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *CreateLocationTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}

	plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	created, err := t.locations.Create(ctx, plan.entity, params.Actor)
	if err != nil {
		return nil, err
	}

	id := created.ID.String()

	return &agent.ToolExecutionResult{
		Action: "created",
		Kind:   "location",
		Name:   created.Name,
		IDs:    map[string]string{"locationId": id},
		Record: &agent.RecordRef{EntityType: locationRecordEntity, ID: id},
	}, nil
}

type locationPlan struct {
	entity   *location.Location
	state    *usstate.UsState
	category *locationcategory.LocationCategory
	unplaced string
}

func (t *CreateLocationTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*locationPlan, error) {
	draft, err := readLocationDraft(params.Params)
	if err != nil {
		return nil, err
	}

	return t.Build(ctx, draft, toolparams.TenantFrom(*params))
}

func (t *CreateLocationTool) Build(
	ctx context.Context,
	draft *locationDraft,
	tenant pagination.TenantInfo,
) (*locationPlan, error) {
	state, err := t.states.GetByAbbreviation(ctx, draft.state)
	if err != nil || state == nil {
		return nil, errortypes.NewValidationError(
			"state", errortypes.ErrInvalid,
			fmt.Sprintf("%s is not a US state abbreviation", draft.state),
		)
	}

	category, err := t.categories.GetByID(ctx, locationcategoryports.GetLocationCategoryByIDRequest{
		ID:         draft.categoryID,
		TenantInfo: tenant,
	})
	if err != nil || category == nil {
		return nil, errortypes.NewValidationError(
			"locationCategoryId", errortypes.ErrInvalid,
			"That is not one of this organization's location categories; use "+
				"location_list_categories",
		)
	}

	entity := &location.Location{
		OrganizationID:     tenant.OrgID,
		BusinessUnitID:     tenant.BuID,
		LocationCategoryID: category.ID,
		StateID:            state.ID,
		Status:             domaintypes.StatusActive,
		Name:               draft.name,
		AddressLine1:       draft.addressLine1,
		AddressLine2:       draft.addressLine2,
		City:               draft.city,
		PostalCode:         draft.postalCode,
	}
	place, unplaced := t.place(ctx, draft, tenant)
	if place != nil {
		latitude, longitude := place.Latitude, place.Longitude
		entity.PlaceID = place.PlaceID
		entity.IsGeocoded = true
		entity.Latitude = &latitude
		entity.Longitude = &longitude
	}

	return &locationPlan{
		entity:   entity,
		state:    state,
		category: category,
		unplaced: unplaced,
	}, nil
}

func (t *CreateLocationTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if params.Actor == nil {
		return toolguard.ErrMissingActor
	}

	_, err := t.plan(ctx, &params)

	return err
}
