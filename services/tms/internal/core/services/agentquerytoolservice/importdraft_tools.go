package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	draftAppliedNote = "The page applies this change now and shows it in the page draft on your " +
		"next turn. Nothing is saved until the person creates the shipment."
	draftRationale = "Hands a change to the shipment the person is building on their own page; " +
		"nothing is saved and nothing is sent."
	fieldKeyDescription  = "The field's key exactly as the page draft lists it, such as rate or bol."
	stopIndexDescription = "The stop's number as the page draft lists it, counting from 0."
)

func importDraftToolProviders() []any {
	return []any{
		newAcceptFieldTool,
		newAcceptAllConfidentTool,
		newSetFieldValueTool,
		provideSetRequiredFieldTool,
		provideSetStopLocationTool,
		newSetStopScheduleTool,
	}
}

func draftPolicy(name string) serviceports.ToolPolicy {
	return readPolicy(name, readSpec{
		resource:  permission.ResourceDocument,
		scope:     agent.ToolScopeSelf,
		effect:    agent.ToolEffectPresent,
		rationale: draftRationale,
	})
}

func importEdit(action pagedraft.Action) pagedraft.Edit {
	return pagedraft.Edit{Surface: pagedraft.SurfaceShipmentImport, Action: action}
}

func draftResult(edit pagedraft.Edit) pagedraft.EditResult {
	return pagedraft.EditResult{Draft: edit, Note: draftAppliedNote}
}

func requireFieldKey(params map[string]any) (string, error) {
	key, err := requireString(params, "fieldKey")
	if err != nil {
		return "", err
	}
	if utf8.RuneCountInString(key) > pagedraft.MaxFieldKeyLength {
		return "", errors.New("parameter \"fieldKey\" is longer than any field on the page")
	}

	return key, nil
}

func requireStopIndex(params map[string]any) (int, error) {
	raw, ok := params["stopIndex"]
	if !ok || raw == nil {
		return 0, errors.New("missing required parameter \"stopIndex\"")
	}

	index := optionalInt(params, "stopIndex", -1)
	if index < 0 || index >= pagedraft.MaxImportStops {
		return 0, fmt.Errorf(
			"parameter \"stopIndex\" must be a stop's number from the page draft, 0 to %d",
			pagedraft.MaxImportStops-1,
		)
	}

	return index, nil
}

type acceptFieldTool struct{}

func newAcceptFieldTool() serviceports.AgentQueryTool { return &acceptFieldTool{} }

func (t *acceptFieldTool) Name() string { return string(pagedraft.ActionAcceptField) }

func (t *acceptFieldTool) Description() string {
	return "Accept one field read from the document as correct on the import page. Use it " +
		"once the person agrees with the value the page draft shows, or when its confidence " +
		"is high and nothing on the page disagrees."
}

func (t *acceptFieldTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fieldKey": map[string]any{"type": "string", "description": fieldKeyDescription},
		},
		"required":             []string{"fieldKey"},
		"additionalProperties": false,
	}
}

func (t *acceptFieldTool) Policy() serviceports.ToolPolicy { return draftPolicy(t.Name()) }

func (t *acceptFieldTool) SearchTerms() []string {
	return []string{"accept", "confirm", "extracted", "correct", "approve value", "import"}
}

func (t *acceptFieldTool) Query(
	_ context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	key, err := requireFieldKey(params.Params)
	if err != nil {
		return nil, err
	}

	edit := importEdit(pagedraft.ActionAcceptField)
	edit.FieldKey = key

	return draftResult(edit), nil
}

type acceptAllConfidentTool struct{}

func newAcceptAllConfidentTool() serviceports.AgentQueryTool {
	return &acceptAllConfidentTool{}
}

func (t *acceptAllConfidentTool) Name() string {
	return string(pagedraft.ActionAcceptAllConfident)
}

func (t *acceptAllConfidentTool) Description() string {
	return "Accept every field the page read from the document with high confidence, at " +
		"once. Use it when the person asks to take what was read as it is; fields that " +
		"need review, conflict or are missing are left for you to settle one at a time."
}

func (t *acceptAllConfidentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
}

func (t *acceptAllConfidentTool) Policy() serviceports.ToolPolicy {
	return draftPolicy(t.Name())
}

func (t *acceptAllConfidentTool) SearchTerms() []string {
	return []string{"accept all", "high confidence", "bulk accept", "import"}
}

func (t *acceptAllConfidentTool) Query(
	_ context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	return draftResult(importEdit(pagedraft.ActionAcceptAllConfident)), nil
}

type setFieldValueTool struct{}

func newSetFieldValueTool() serviceports.AgentQueryTool { return &setFieldValueTool{} }

func (t *setFieldValueTool) Name() string { return string(pagedraft.ActionSetFieldValue) }

func (t *setFieldValueTool) Description() string {
	return "Set or correct the value of a field on the import page, such as the BOL, the " +
		"weight, the pieces or the freight rate. Use it with the value the person gave you " +
		"or one the document plainly shows; never make a value up."
}

func (t *setFieldValueTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fieldKey": map[string]any{"type": "string", "description": fieldKeyDescription},
			"value": map[string]any{
				"type":        "string",
				"description": "The value as the person would type it.",
			},
		},
		"required":             []string{"fieldKey", "value"},
		"additionalProperties": false,
	}
}

func (t *setFieldValueTool) Policy() serviceports.ToolPolicy { return draftPolicy(t.Name()) }

func (t *setFieldValueTool) SearchTerms() []string {
	return []string{"set", "change", "correct", "bol", "weight", "pieces", "rate", "import"}
}

func (t *setFieldValueTool) Query(
	_ context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	key, err := requireFieldKey(params.Params)
	if err != nil {
		return nil, err
	}
	raw, ok := params.Params["value"].(string)
	if !ok {
		return nil, errors.New("parameter \"value\" must be a string")
	}
	value := strings.TrimSpace(raw)
	if utf8.RuneCountInString(value) > pagedraft.MaxFieldValueLength {
		return nil, fmt.Errorf("parameter \"value\" is longer than %d characters",
			pagedraft.MaxFieldValueLength)
	}

	edit := importEdit(pagedraft.ActionSetFieldValue)
	edit.FieldKey = key
	edit.Value = value

	return draftResult(edit), nil
}

type requiredRecordReader struct {
	customers     repositories.CustomerRepository
	serviceTypes  repositories.ServiceTypeRepository
	shipmentTypes repositories.ShipmentTypeRepository
	formulas      repositories.FormulaTemplateRepository
}

func (r requiredRecordReader) resource(field pagedraft.RequiredField) permission.Resource {
	switch field {
	case pagedraft.RequiredCustomer:
		return permission.ResourceCustomer
	case pagedraft.RequiredServiceType:
		return permission.ResourceServiceType
	case pagedraft.RequiredShipmentType:
		return permission.ResourceShipmentType
	case pagedraft.RequiredFormulaTemplate:
		return permission.ResourceFormulaTemplate
	default:
		return ""
	}
}

func (r requiredRecordReader) label(
	ctx context.Context,
	field pagedraft.RequiredField,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (string, error) {
	switch field {
	case pagedraft.RequiredCustomer:
		found, err := r.customers.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID: id, TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return codedLabel(found.Code, found.Name), nil
	case pagedraft.RequiredServiceType:
		found, err := r.serviceTypes.GetByID(ctx, repositories.GetServiceTypeByIDRequest{
			ID: id, TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return codedLabel(found.Code, found.Description), nil
	case pagedraft.RequiredShipmentType:
		found, err := r.shipmentTypes.GetByID(ctx, repositories.GetShipmentTypeByIDRequest{
			ID: id, TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return codedLabel(found.Code, found.Description), nil
	case pagedraft.RequiredFormulaTemplate:
		found, err := r.formulas.GetByID(ctx, repositories.GetFormulaTemplateByIDRequest{
			TemplateID: id, TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return found.Name, nil
	default:
		return "", fmt.Errorf("%q is not a required field", field)
	}
}

func codedLabel(code, name string) string {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	switch {
	case code == "":
		return name
	case name == "":
		return code
	default:
		return code + " — " + name
	}
}

func requiredFieldSource(field pagedraft.RequiredField) string {
	switch field {
	case pagedraft.RequiredCustomer:
		return "list_customers"
	case pagedraft.RequiredServiceType:
		return "list_service_types"
	case pagedraft.RequiredShipmentType:
		return "list_shipment_types"
	case pagedraft.RequiredFormulaTemplate:
		return "list_formula_templates"
	default:
		return "the matching list tool"
	}
}

type setRequiredFieldTool struct {
	records requiredRecordReader
	access  fieldAccess
}

func provideSetRequiredFieldTool(
	customers repositories.CustomerRepository,
	serviceTypes repositories.ServiceTypeRepository,
	shipmentTypes repositories.ShipmentTypeRepository,
	formulas repositories.FormulaTemplateRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &setRequiredFieldTool{
		records: requiredRecordReader{
			customers:     customers,
			serviceTypes:  serviceTypes,
			shipmentTypes: shipmentTypes,
			formulas:      formulas,
		},
		access: newFieldAccess(permissions),
	}
}

func (t *setRequiredFieldTool) Name() string { return string(pagedraft.ActionSetRequiredField) }

func (t *setRequiredFieldTool) Description() string {
	return "Set the customer, service type, shipment type or rating method on the import " +
		"page, by record id. Find the record first with list_customers, list_service_types, " +
		"list_shipment_types or list_formula_templates, and set it once the person agrees; " +
		"the page shows the record's own name."
}

func (t *setRequiredFieldTool) ParamSchema() map[string]any {
	fields := pagedraft.AllRequiredFields()
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, string(field))
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"field": map[string]any{
				"type":        "string",
				"enum":        names,
				"description": "Which of the four the record is for.",
			},
			"recordId": map[string]any{
				"type": "string",
				"description": "The record's id, from list_customers, list_service_types, " +
					"list_shipment_types or list_formula_templates to match the field.",
			},
		},
		"required":             []string{"field", "recordId"},
		"additionalProperties": false,
	}
}

func (t *setRequiredFieldTool) Policy() serviceports.ToolPolicy { return draftPolicy(t.Name()) }

func (t *setRequiredFieldTool) SearchTerms() []string {
	return []string{
		"customer", "service type", "shipment type", "rating method", "formula template",
		"required", "import",
	}
}

func (t *setRequiredFieldTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	fieldName, err := requireString(params.Params, "field")
	if err != nil {
		return nil, err
	}
	field := pagedraft.RequiredField(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf(
			"parameter \"field\" must be customerId, serviceTypeId, shipmentTypeId or " +
				"formulaTemplateId",
		)
	}

	id, err := requirePulid(params.Params, "recordId")
	if err != nil {
		return nil, err
	}

	resource := t.records.resource(field)
	if !t.access.mayRead(ctx, params, resource) {
		return nil, fmt.Errorf(
			"the person you are working for may not read %s records, so it was not set; "+
				"tell them it needs %s access", resource, resource,
		)
	}

	label, err := t.records.label(ctx, field, id, pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"%s is not a record this field can take; look it up with %s: %w",
			id, requiredFieldSource(field), err,
		)
	}

	edit := importEdit(pagedraft.ActionSetRequiredField)
	edit.FieldKey = string(field)
	edit.Value = id.String()
	edit.Label = label

	return draftResult(edit), nil
}

type setStopLocationTool struct {
	locations repositories.LocationRepository
	access    fieldAccess
}

func provideSetStopLocationTool(
	locations repositories.LocationRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &setStopLocationTool{locations: locations, access: newFieldAccess(permissions)}
}

func (t *setStopLocationTool) Name() string { return string(pagedraft.ActionSetStopLocation) }

func (t *setStopLocationTool) Description() string {
	return "Match a stop on the import page to a location record. Find the location with " +
		"list_locations by name, city or address; when none matches, create_location " +
		"proposes a new one for the person to approve, and you set it once it exists."
}

func (t *setStopLocationTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"stopIndex": map[string]any{"type": "integer", "description": stopIndexDescription},
			"locationId": map[string]any{
				"type":        "string",
				"description": "The location's id, from list_locations or create_location.",
			},
		},
		"required":             []string{"stopIndex", "locationId"},
		"additionalProperties": false,
	}
}

func (t *setStopLocationTool) Policy() serviceports.ToolPolicy { return draftPolicy(t.Name()) }

func (t *setStopLocationTool) SearchTerms() []string {
	return []string{"stop", "location", "match", "pickup", "delivery", "facility", "import"}
}

func (t *setStopLocationTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	index, err := requireStopIndex(params.Params)
	if err != nil {
		return nil, err
	}
	locationID, err := requirePulid(params.Params, "locationId")
	if err != nil {
		return nil, err
	}
	if !t.access.mayRead(ctx, params, permission.ResourceLocation) {
		return nil, errors.New(
			"the person you are working for may not read locations, so the stop was not " +
				"matched; tell them it needs location access",
		)
	}

	found, err := t.locations.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID: locationID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf(
			"%s is not a location; find one with list_locations: %w", locationID, err,
		)
	}

	edit := importEdit(pagedraft.ActionSetStopLocation)
	edit.StopIndex = &index
	edit.Value = found.ID.String()
	edit.Label = locationLabel(found.Code, found.Name, found.City)

	return draftResult(edit), nil
}

func locationLabel(code, name, city string) string {
	label := codedLabel(code, name)
	if city = strings.TrimSpace(city); city != "" {
		label += " (" + city + ")"
	}

	return label
}

type setStopScheduleTool struct{}

func newSetStopScheduleTool() serviceports.AgentQueryTool { return &setStopScheduleTool{} }

func (t *setStopScheduleTool) Name() string { return string(pagedraft.ActionSetStopSchedule) }

func (t *setStopScheduleTool) Description() string {
	return "Set when a stop on the import page is scheduled: the start of its window and, " +
		"optionally, the end. A time range the document shows without a date is not a " +
		"schedule; ask the person for the date first."
}

func (t *setStopScheduleTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"stopIndex": map[string]any{"type": "integer", "description": stopIndexDescription},
			"windowStart": map[string]any{
				"type": "string",
				"description": "The date and time the window opens, as YYYY-MM-DDTHH:MM in " +
					"the organization's time zone, or with an offset.",
			},
			"windowEnd": map[string]any{
				"type":        "string",
				"description": "Optional: when the window closes, in the same form.",
			},
		},
		"required":             []string{"stopIndex", "windowStart"},
		"additionalProperties": false,
	}
}

func (t *setStopScheduleTool) Policy() serviceports.ToolPolicy { return draftPolicy(t.Name()) }

func (t *setStopScheduleTool) SearchTerms() []string {
	return []string{"schedule", "window", "appointment", "pickup time", "delivery date", "import"}
}

func (t *setStopScheduleTool) Query(
	_ context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	index, err := requireStopIndex(params.Params)
	if err != nil {
		return nil, err
	}

	location := organizationZone(params.Timezone)
	start, err := parseWindow(params.Params, "windowStart", location)
	if err != nil {
		return nil, err
	}
	if start == 0 {
		return nil, errors.New("missing required parameter \"windowStart\"")
	}
	end, err := parseWindow(params.Params, "windowEnd", location)
	if err != nil {
		return nil, err
	}
	if end != 0 && end < start {
		return nil, errors.New("the window cannot close before it opens")
	}

	edit := importEdit(pagedraft.ActionSetStopSchedule)
	edit.StopIndex = &index
	edit.WindowStart = start
	edit.WindowEnd = end

	return draftResult(edit), nil
}

var windowLayouts = [...]string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
}

func organizationZone(timezone string) *time.Location {
	if timezone = strings.TrimSpace(timezone); timezone != "" {
		if loaded, err := time.LoadLocation(timezone); err == nil {
			return loaded
		}
	}

	return time.UTC
}

func parseWindow(params map[string]any, key string, location *time.Location) (int64, error) {
	raw := optionalString(params, key)
	if raw == "" {
		return 0, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.Unix(), nil
	}
	for _, layout := range windowLayouts {
		if parsed, err := time.ParseInLocation(layout, raw, location); err == nil {
			return parsed.Unix(), nil
		}
	}

	return 0, fmt.Errorf(
		"parameter %q must be a date and time such as 2026-10-01T08:00, not %q", key, raw,
	)
}
