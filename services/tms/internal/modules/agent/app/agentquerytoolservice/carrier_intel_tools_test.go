package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCarrierIntelEventReader struct {
	events  []*carrierintel.CarrierIntelEvent
	hasMore bool
	listed  *carrierintelports.ListCarrierIntelEventsRequest
}

func (f *fakeCarrierIntelEventReader) ListEvents(
	_ context.Context,
	req *carrierintelports.ListCarrierIntelEventsRequest,
) (*pagination.CursorListResult[*carrierintel.CarrierIntelEvent], error) {
	f.listed = req

	return &pagination.CursorListResult[*carrierintel.CarrierIntelEvent]{
		Items:       f.events,
		HasNextPage: f.hasMore,
	}, nil
}

func sampleCarrierIntelEvent() *carrierintel.CarrierIntelEvent {
	acknowledged := int64(1_791_000_000)

	return &carrierintel.CarrierIntelEvent{
		ID:               pulid.MustNew("cievt_"),
		SubjectType:      carrierintel.SubjectTypeCarrier,
		SubjectID:        "car_1",
		CarrierID:        pulid.MustNew("car_"),
		DOTNumber:        "1234567",
		SubjectName:      "Northwind Haulers",
		Category:         carrierintel.SectionInsurance,
		Severity:         carrierintel.SeverityCritical,
		Action:           carrierintel.RuleActionBlock,
		Summary:          "Auto liability insurance lapsed",
		PriorValue:       map[string]any{"insurer": "Acme Mutual"},
		CurrentValue:     map[string]any{"insurer": nil},
		DetectedAt:       1_790_900_000,
		Fingerprint:      "fingerprint-not-for-the-model",
		Status:           carrierintel.EventStatusAcknowledged,
		AcknowledgedByID: pulid.MustNew("usr_"),
		AcknowledgedAt:   &acknowledged,
		CreatedAt:        1_790_900_000,
	}
}

func carrierIntelEventProperties(t *testing.T) map[string]any {
	t.Helper()

	schema := newListCarrierIntelEventsTool(nil).ParamSchema()
	properties, ok := schema[toolschema.KeyProperties].(map[string]any)
	require.True(t, ok)

	return properties
}

func TestListCarrierIntelEvents_SchemaOffersTheFiltersTheEventListSupports(t *testing.T) {
	t.Parallel()

	tool := newListCarrierIntelEventsTool(nil)
	schema := tool.ParamSchema()
	assert.Equal(t, false, schema[toolschema.KeyAdditionalProperties])
	assert.NotContains(t, schema, "required", "every filter is optional")

	properties := carrierIntelEventProperties(t)
	for _, name := range []string{
		"status", "openOnly", "severity", "category", "carrierId", "query",
		"fromDate", "toDate", "limit", "after",
	} {
		assert.Containsf(t, properties, name, "the schema offers %s", name)
	}

	enumOf := func(name string) []string {
		property, _ := properties[name].(map[string]any)
		items, _ := property[toolschema.KeyItems].(map[string]any)
		values, _ := items[toolschema.KeyEnum].([]string)

		return values
	}
	assert.Equal(t, []string{"Open", "Acknowledged", "Resolved", "Dismissed"}, enumOf("status"))
	assert.Equal(t, []string{"Critical", "High", "Medium", "Low", "Info"}, enumOf("severity"))
	assert.Len(t, enumOf("category"), len(carrierintel.AllSections()))
	assert.Contains(t, enumOf("category"), "Authority")
	assert.Contains(t, enumOf("category"), "Insurance")

	carrier, _ := properties["carrierId"].(map[string]any)
	assert.Equal(t, permission.ResourceCarrier.String(), carrier[toolschema.KeyRecordOf])
	assert.Contains(t, carrier[toolschema.KeyDescription], "carrier_list")

	description := tool.Description()
	for _, next := range []string{
		"carrier_get_intelligence_event", "carrier_acknowledge_intelligence_event", "carrier_resolve_intelligence_event",
	} {
		assert.Containsf(t, description, next, "a row's id is handed on to %s", next)
	}
}

func TestListCarrierIntelEvents_IsAReadUnderCarrierIntelligence(t *testing.T) {
	t.Parallel()

	tool := newListCarrierIntelEventsTool(nil)
	policy := tool.Policy()

	assert.Equal(t, "carrier_list_intelligence_events", tool.Name())
	assert.Equal(t, "carrier_list_intelligence_events", policy.Name)
	assert.Equal(t, agent.ToolKindQuery, policy.Kind)
	assert.Equal(t, permission.ResourceCarrierIntelligence, policy.Resource,
		"the same resource the carrierIntelEvents query requires")
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressNone}, policy.Egress)
}

func TestListCarrierIntelEvents_PassesEveryFilterToTheServiceWithinTheTenant(t *testing.T) {
	t.Parallel()

	carrierID := pulid.MustNew("car_")
	reader := &fakeCarrierIntelEventReader{}
	params := testParamsIn("America/Chicago", map[string]any{
		"status":    []any{"Open", "Acknowledged"},
		"openOnly":  true,
		"severity":  []any{"Critical", "High"},
		"category":  []any{"Authority", "Insurance"},
		"carrierId": carrierID.String(),
		"query":     " Northwind ",
		"fromDate":  "2026-10-12",
		"toDate":    "2026-10-16",
		"limit":     500.0,
	})

	out, err := newListCarrierIntelEventsTool(reader).Query(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, reader.listed)

	req := reader.listed
	assert.Equal(t, []carrierintel.EventStatus{
		carrierintel.EventStatusOpen,
		carrierintel.EventStatusAcknowledged,
	}, req.Statuses)
	assert.True(t, req.OpenOnly)
	assert.Equal(t, []carrierintel.Severity{
		carrierintel.SeverityCritical,
		carrierintel.SeverityHigh,
	}, req.Severities)
	assert.Equal(t, []carrierintel.Section{
		carrierintel.SectionAuthority,
		carrierintel.SectionInsurance,
	}, req.Categories)
	assert.Equal(t, carrierID, req.CarrierID)

	require.NotNil(t, req.Filter)
	assert.Equal(t, "Northwind", req.Filter.Query)
	assert.Equal(t, params.OrganizationID, req.Filter.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, req.Filter.TenantInfo.BuID)
	assert.Equal(t, params.Actor.UserID, req.Filter.TenantInfo.UserID)
	assert.Equal(t, carrierIntelEventMaxLimit, req.Cursor.Limit, "a page is bounded")
	assert.False(t, req.Cursor.IncludeTotalCount, "a page never pays for a count")

	clk := filtercatalog.NewClock("America/Chicago")
	from, ok := clk.Day("2026-10-12")
	require.True(t, ok)
	to, ok := clk.Day("2026-10-16")
	require.True(t, ok)
	assert.Equal(t, from, req.DetectedFrom, "the window opens on the organization's midnight")
	assert.Equal(t, clk.DayEnd(to), req.DetectedTo, "the last day is included in full")

	result, ok := out.(*carrierIntelEventsResult)
	require.True(t, ok)
	assert.Equal(t, 0, result.Count)
	assert.NotEmpty(t, result.Note, "an empty page says what was searched for")
	assert.Contains(t, result.SearchedFor, `text matching "Northwind"`)
}

func TestListCarrierIntelEvents_RowsCarryWhatAPersonActsOnAndNothingElse(t *testing.T) {
	t.Parallel()

	event := sampleCarrierIntelEvent()
	reader := &fakeCarrierIntelEventReader{
		events:  []*carrierintel.CarrierIntelEvent{nil, event},
		hasMore: true,
	}

	out, err := newListCarrierIntelEventsTool(reader).Query(
		t.Context(),
		testParams(map[string]any{"limit": 1.0}),
	)
	require.NoError(t, err)

	result, ok := out.(*carrierIntelEventsResult)
	require.True(t, ok)
	rows, ok := result.Items.([]carrierIntelEventRow)
	require.True(t, ok)
	require.Len(t, rows, 1, "a nil event is skipped rather than drawn as an empty row")
	assert.Equal(t, 1, result.Count)
	assert.True(t, result.HasMore)
	require.NotEmpty(t, result.NextCursor)
	_, err = pagination.NewCursorInfo(1, result.NextCursor)
	require.NoError(t, err, "the cursor handed out is one the tool accepts back")

	row := rows[0]
	assert.Equal(t, event.ID.String(), row.ID)
	assert.Equal(t, event.CarrierID.String(), row.CarrierID)
	assert.Equal(t, "Northwind Haulers", row.Carrier)
	assert.Equal(t, "1234567", row.DOTNumber)
	assert.Equal(t, "Insurance", row.Category)
	assert.Equal(t, "Critical", row.Severity)
	assert.Equal(t, "Block", row.Action)
	assert.Equal(t, "Acknowledged", row.Status)
	assert.Equal(t, "Auto liability insurance lapsed", row.Summary)

	document := encodedDocument(t, out)
	assert.Contains(t, document, "nextCursor")
	assert.Equal(t, true, document["hasMore"])
	items, ok := document["items"].([]any)
	require.True(t, ok)
	encodedRow, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "not resolved", encodedRow["resolvedAt"],
		"an unresolved event says so rather than leaving the date out")
	assert.InDelta(t, float64(1_791_000_000), encodedRow["acknowledgedAt"], 0)
	for _, withheld := range []string{
		"fingerprint", "priorValue", "currentValue", "organizationId", "businessUnitId",
		"acknowledgedById", "snapshotId", "provider",
	} {
		assert.NotContainsf(t, encodedRow, withheld,
			"%s is not part of the row; carrier_get_intelligence_event reads the whole event", withheld)
	}
	assert.Equal(t, []string{
		"id", "carrierId", "carrier", "dotNumber", "subjectType", "category", "severity",
		"action", "status", "summary", "detectedAt", "acknowledgedAt", "resolvedAt",
		"resolution",
	}, result.Columns)
}

func TestListCarrierIntelEvents_RefusesWhatItCannotFilterOn(t *testing.T) {
	t.Parallel()

	for name, bad := range map[string]map[string]any{
		"an unknown status":      {"status": []any{"Pending"}},
		"an unknown severity":    {"severity": []any{"Severe"}},
		"an unknown category":    {"category": []any{"Weather"}},
		"a carrier id typo":      {"carrierId": "not-an-id"},
		"a date it cannot read":  {"fromDate": "last week"},
		"an inverted window":     {"fromDate": "2026-10-16", "toDate": "2026-10-12"},
		"a cursor it never gave": {"after": "not-a-cursor"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &fakeCarrierIntelEventReader{}
			_, err := newListCarrierIntelEventsTool(reader).Query(t.Context(), testParams(bad))
			require.Error(t, err)
			assert.Nil(t, reader.listed, "nothing is listed under a filter that was refused")
		})
	}
}

func TestListCarrierIntelEvents_RefusesACallerOutsideTheTenant(t *testing.T) {
	t.Parallel()

	reader := &fakeCarrierIntelEventReader{}
	params := testParams(map[string]any{})
	params.OrganizationID = pulid.MustNew("org_")

	_, err := newListCarrierIntelEventsTool(reader).Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, reader.listed)
}

func TestGetCarrierIntelEvent_NamesTheListThatHandsOutItsID(t *testing.T) {
	t.Parallel()

	tool := newGetCarrierIntelEventTool(nil, nil)
	assert.Contains(t, tool.Description(), "carrier_list_intelligence_events")
	assert.NotContains(t, tool.Description(), "No tool lists")

	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	eventID, _ := properties["eventId"].(map[string]any)
	assert.Contains(t, eventID[toolschema.KeyDescription], "carrier_list_intelligence_events")
}
