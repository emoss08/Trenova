package agentquerytoolservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDraftLocations struct {
	repositories.LocationRepository

	found     *location.Location
	requested repositories.GetLocationByIDRequest
}

func (f *fakeDraftLocations) GetByID(
	_ context.Context,
	req repositories.GetLocationByIDRequest,
) (*location.Location, error) {
	f.requested = req
	if f.found == nil || f.found.ID != req.ID {
		return nil, errors.New("location not found")
	}

	return f.found, nil
}

type fakeDraftCustomers struct {
	repositories.CustomerRepository

	found *customer.Customer
}

func (f *fakeDraftCustomers) GetByID(
	_ context.Context,
	req repositories.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	if f.found == nil || f.found.ID != req.ID {
		return nil, errors.New("customer not found")
	}

	return f.found, nil
}

func draftEditOf(t *testing.T, result any) pagedraft.Edit {
	t.Helper()

	edited, ok := result.(pagedraft.EditResult)
	require.True(t, ok, "a draft tool answers with a draft edit")
	assert.NotEmpty(t, edited.Note)

	return edited.Draft
}

func TestImportDraftTools_HandTheChangeToThePersonsOwnPage(t *testing.T) {
	t.Parallel()

	tools := []serviceports.AgentQueryTool{
		newAcceptFieldTool(),
		newAcceptAllConfidentTool(),
		newSetFieldValueTool(),
		provideSetRequiredFieldTool(nil, nil, nil, nil, &fakePermissions{}),
		provideSetStopLocationTool(nil, &fakePermissions{}),
		newSetStopScheduleTool(),
	}

	for _, tool := range tools {
		assert.True(t, pagedraft.IsEditTool(tool.Name()), tool.Name())
		policy := tool.Policy()
		assert.Equal(t, agent.ToolKindQuery, policy.Kind, tool.Name())
		assert.Equal(t, agent.ToolScopeSelf, policy.Scope, tool.Name())
		assert.Equal(t, agent.ToolEffectPresent, policy.Effect, tool.Name())
		assert.Equal(t, permission.ResourceDocument, policy.Resource, tool.Name())
		assert.Equal(t, permission.OpRead, policy.Operation, tool.Name())
	}
}

func TestAcceptField_NamesTheFieldTheDraftListed(t *testing.T) {
	t.Parallel()

	result, err := newAcceptFieldTool().Query(t.Context(),
		chatParams(map[string]any{"fieldKey": "rate"}, ""))
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	assert.Equal(t, pagedraft.SurfaceShipmentImport, edit.Surface)
	assert.Equal(t, pagedraft.ActionAcceptField, edit.Action)
	assert.Equal(t, "rate", edit.FieldKey)
}

func TestAcceptField_RefusesAKeyLongerThanAnyField(t *testing.T) {
	t.Parallel()

	long := make([]byte, pagedraft.MaxFieldKeyLength+1)
	for idx := range long {
		long[idx] = 'k'
	}

	_, err := newAcceptFieldTool().Query(t.Context(),
		chatParams(map[string]any{"fieldKey": string(long)}, ""))
	require.Error(t, err)

	_, err = newAcceptFieldTool().Query(t.Context(), chatParams(map[string]any{}, ""))
	require.Error(t, err)
}

func TestAcceptAllConfident_TakesNoArguments(t *testing.T) {
	t.Parallel()

	result, err := newAcceptAllConfidentTool().Query(t.Context(),
		chatParams(map[string]any{}, ""))
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	assert.Equal(t, pagedraft.ActionAcceptAllConfident, edit.Action)
	assert.Empty(t, edit.FieldKey)
}

func TestSetFieldValue_TrimsAndBoundsTheValue(t *testing.T) {
	t.Parallel()

	tool := newSetFieldValueTool()

	result, err := tool.Query(t.Context(),
		chatParams(map[string]any{"fieldKey": "weight", "value": "  42000 "}, ""))
	require.NoError(t, err)
	edit := draftEditOf(t, result)
	assert.Equal(t, "weight", edit.FieldKey)
	assert.Equal(t, "42000", edit.Value)

	_, err = tool.Query(t.Context(),
		chatParams(map[string]any{"fieldKey": "weight", "value": 42000}, ""))
	require.Error(t, err, "a number is not the value a person types")

	long := make([]byte, pagedraft.MaxFieldValueLength+1)
	for idx := range long {
		long[idx] = 'v'
	}
	_, err = tool.Query(t.Context(),
		chatParams(map[string]any{"fieldKey": "bol", "value": string(long)}, ""))
	require.Error(t, err)
}

func TestSetRequiredField_ResolvesTheRecordsOwnName(t *testing.T) {
	t.Parallel()

	found := &customer.Customer{ID: pulid.MustNew("cus_"), Code: "ACME", Name: "Acme Foods"}
	tool := provideSetRequiredFieldTool(
		&fakeDraftCustomers{found: found}, nil, nil, nil, &fakePermissions{allowed: true},
	)

	result, err := tool.Query(t.Context(), chatParams(map[string]any{
		"field":    string(pagedraft.RequiredCustomer),
		"recordId": found.ID.String(),
	}, ""))
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	assert.Equal(t, pagedraft.ActionSetRequiredField, edit.Action)
	assert.Equal(t, string(pagedraft.RequiredCustomer), edit.FieldKey)
	assert.Equal(t, found.ID.String(), edit.Value)
	assert.Equal(t, "ACME — Acme Foods", edit.Label)
}

func TestSetRequiredField_RefusesARecordThePersonMayNotRead(t *testing.T) {
	t.Parallel()

	found := &customer.Customer{ID: pulid.MustNew("cus_"), Code: "ACME", Name: "Acme Foods"}
	tool := provideSetRequiredFieldTool(
		&fakeDraftCustomers{found: found}, nil, nil, nil, &fakePermissions{allowed: false},
	)

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"field":    string(pagedraft.RequiredCustomer),
		"recordId": found.ID.String(),
	}, ""))
	require.ErrorContains(t, err, "may not read")
}

func TestSetRequiredField_RefusesAnUnknownFieldOrRecord(t *testing.T) {
	t.Parallel()

	tool := provideSetRequiredFieldTool(
		&fakeDraftCustomers{}, nil, nil, nil, &fakePermissions{allowed: true},
	)

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"field":    "carrierId",
		"recordId": pulid.MustNew("cus_").String(),
	}, ""))
	require.Error(t, err)

	_, err = tool.Query(t.Context(), chatParams(map[string]any{
		"field":    string(pagedraft.RequiredCustomer),
		"recordId": pulid.MustNew("cus_").String(),
	}, ""))
	require.ErrorContains(t, err, "list_customers")
}

func TestSetStopLocation_MatchesARecordInTheCallersTenant(t *testing.T) {
	t.Parallel()

	found := &location.Location{
		ID:   pulid.MustNew("loc_"),
		Code: "RENO01",
		Name: "Reno Warehouse",
		City: "Reno",
	}
	locations := &fakeDraftLocations{found: found}
	tool := provideSetStopLocationTool(locations, &fakePermissions{allowed: true})
	params := chatParams(map[string]any{"stopIndex": 1, "locationId": found.ID.String()}, "")

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	require.NotNil(t, edit.StopIndex)
	assert.Equal(t, 1, *edit.StopIndex)
	assert.Equal(t, found.ID.String(), edit.Value)
	assert.Equal(t, "RENO01 — Reno Warehouse (Reno)", edit.Label)
	assert.Equal(t, params.OrganizationID, locations.requested.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, locations.requested.TenantInfo.BuID)
}

func TestSetStopLocation_RefusesAnOutOfRangeStopOrAnUnreadableLocation(t *testing.T) {
	t.Parallel()

	found := &location.Location{ID: pulid.MustNew("loc_"), Name: "Reno Warehouse"}

	allowed := provideSetStopLocationTool(
		&fakeDraftLocations{found: found}, &fakePermissions{allowed: true},
	)
	_, err := allowed.Query(t.Context(), chatParams(map[string]any{
		"stopIndex": pagedraft.MaxImportStops, "locationId": found.ID.String(),
	}, ""))
	require.Error(t, err)

	_, err = allowed.Query(t.Context(), chatParams(map[string]any{
		"locationId": found.ID.String(),
	}, ""))
	require.Error(t, err)

	_, err = allowed.Query(t.Context(), chatParams(map[string]any{
		"stopIndex": 0, "locationId": pulid.MustNew("loc_").String(),
	}, ""))
	require.ErrorContains(t, err, "list_locations")

	denied := provideSetStopLocationTool(
		&fakeDraftLocations{found: found}, &fakePermissions{allowed: false},
	)
	_, err = denied.Query(t.Context(), chatParams(map[string]any{
		"stopIndex": 0, "locationId": found.ID.String(),
	}, ""))
	require.ErrorContains(t, err, "may not read")
}

func TestSetStopSchedule_ReadsLocalTimesInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	params := chatParams(map[string]any{
		"stopIndex":   0,
		"windowStart": "2026-10-06T08:00",
		"windowEnd":   "2026-10-06T10:30",
	}, "")
	params.Timezone = "America/Chicago"

	result, err := newSetStopScheduleTool().Query(t.Context(), params)
	require.NoError(t, err)

	zone, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	edit := draftEditOf(t, result)
	assert.Equal(t, time.Date(2026, 10, 6, 8, 0, 0, 0, zone).Unix(), edit.WindowStart)
	assert.Equal(t, time.Date(2026, 10, 6, 10, 30, 0, 0, zone).Unix(), edit.WindowEnd)
}

func TestSetStopSchedule_HonoursAnExplicitOffset(t *testing.T) {
	t.Parallel()

	params := chatParams(map[string]any{
		"stopIndex":   2,
		"windowStart": "2026-10-06T08:00:00-04:00",
	}, "")
	params.Timezone = "America/Los_Angeles"

	result, err := newSetStopScheduleTool().Query(t.Context(), params)
	require.NoError(t, err)

	edit := draftEditOf(t, result)
	assert.Equal(t, time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC).Unix(), edit.WindowStart)
	assert.Zero(t, edit.WindowEnd)
}

func TestSetStopSchedule_RefusesATimeRangeWithoutADate(t *testing.T) {
	t.Parallel()

	tool := newSetStopScheduleTool()

	_, err := tool.Query(t.Context(), chatParams(map[string]any{
		"stopIndex": 0, "windowStart": "06:00-22:00",
	}, ""))
	require.ErrorContains(t, err, "windowStart")

	_, err = tool.Query(t.Context(), chatParams(map[string]any{
		"stopIndex":   0,
		"windowStart": "2026-10-06T10:00",
		"windowEnd":   "2026-10-06T08:00",
	}, ""))
	require.ErrorContains(t, err, "close before it opens")

	_, err = tool.Query(t.Context(), chatParams(map[string]any{"stopIndex": 0}, ""))
	require.Error(t, err)
}
