package paletteanswer

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func idSchema(param string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{param: map[string]any{"type": "string"}},
		"required":   []string{param},
	}
}

type fixture struct {
	get         *agenttest.StubQueryTool
	list        *agenttest.StubQueryTool
	permissions *agenttest.StubPermissions
	reader      *reader
}

func newFixture() *fixture {
	get := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Resource: permission.ResourceShipment,
		Schema:   idSchema("shipmentId"),
		Result:   map[string]any{"id": "shp_1", "proNumber": "S-104"},
	}
	list := &agenttest.StubQueryTool{
		ToolName: "shipment_list",
		Resource: permission.ResourceShipment,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"type": "string"},
				"after":  map[string]any{"type": "string"},
			},
		},
		Result: map[string]any{"items": []any{}, "total": 0},
	}
	permissions := &agenttest.StubPermissions{Denied: map[string]bool{}}
	actor := permissionports.UserActor(pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	})

	return &fixture{
		get:         get,
		list:        list,
		permissions: permissions,
		reader: &reader{
			catalog: newCatalog(&agenttest.StubQueryRegistry{
				Tools: []agenttoolports.AgentQueryTool{get, list},
			}),
			validator:   toolschema.NewValidator(),
			permissions: permissions,
			actor:       actor,
		},
	}
}

func TestCatalog_IndexesEachKindsGetAndListByTheRecordTheyRead(t *testing.T) {
	t.Parallel()

	f := newFixture()

	get, ok := f.reader.catalog.get(permission.ResourceShipment)
	require.True(t, ok)
	assert.Equal(t, "shipment_get", get.Name())
	list, ok := f.reader.catalog.list("shipment")
	require.True(t, ok)
	assert.Equal(t, "shipment_list", list.Name())
	assert.Equal(t, []string{"shipment"}, f.reader.catalog.kinds)
	assert.Equal(t, "- shipment: status", f.reader.catalog.listGuide(), "paging is left out")
}

func TestFilterNames_PutsRequiredThenFixedValuesFirstAndWritesFewValuesOut(t *testing.T) {
	t.Parallel()

	got := filterNames(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"q":          map[string]any{"type": "string"},
			"after":      map[string]any{"type": "string"},
			"customerId": map[string]any{"type": "string"},
			"status": map[string]any{
				"type": "string", "enum": []any{"New", "InTransit", "Delivered"},
			},
			"equipment": map[string]any{
				"type": "string", "enum": []string{"a", "b", "c", "d", "e", "f", "g"},
			},
		},
		"required": []string{"customerId"},
	})

	assert.Equal(t, []string{
		"customerId",
		"equipment",
		"status (New|InTransit|Delivered)",
		"q",
	}, got)
}

func TestListRecords_RefusesArgumentsTheListDoesNotTakeAndSaysWhatItWanted(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.list.Schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{"type": "string", "enum": []string{"New", "InTransit"}},
		},
		"additionalProperties": false,
	}

	result := (&listTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"kind":    "shipment",
		"filters": map[string]any{"status": "Lost"},
	})

	assert.True(t, result.Failed)
	assert.Contains(t, result.Content, "status")
	assert.Zero(t, f.list.Calls, "a call that does not fit is not run")
}

func TestTools_OffersSearchOnlyWhereSearchIsWired(t *testing.T) {
	t.Parallel()

	names := func(r *reader) []string {
		out := []string{}
		for _, tool := range r.tools() {
			out = append(out, tool.Spec().Name)
		}
		return out
	}

	assert.Equal(t, []string{"get_record", "list_records"}, names(newFixture().reader))
}

func TestGetRecord_ReadsTheRecordAsThePersonAsking(t *testing.T) {
	t.Parallel()

	f := newFixture()
	id := pulid.MustNew("shp_").String()

	result := (&getTool{reader: f.reader}).Call(t.Context(), map[string]any{"id": id})

	require.False(t, result.Failed, result.Content)
	assert.Contains(t, result.Content, "S-104")
	assert.True(t, strings.HasPrefix(result.Content, "Result from shipment_get:"))
	require.Equal(t, 1, f.get.Calls)
	assert.Equal(t, map[string]any{"shipmentId": id}, f.get.LastParams.Params)
	assert.Equal(t, f.reader.actor, f.get.LastParams.Actor)
	require.Len(t, f.permissions.Requests, 1)
	assert.Equal(t, "shipment", f.permissions.Requests[0].Resource)
	assert.Equal(t, permission.OpRead, f.permissions.Requests[0].Operation)
}

// A document is opened the way every other kind is, through the read named
// document_get: a question about a file the palette found ("what does this
// rate confirmation say", "who uploaded it") reads the file itself.
func TestGetRecord_OpensADocumentThroughItsRead(t *testing.T) {
	t.Parallel()

	read := &agenttest.StubQueryTool{
		ToolName: "document_get",
		Resource: permission.ResourceDocument,
		Schema:   idSchema("documentId"),
		Result: map[string]any{
			"documentId": "doc_1",
			"fileName":   "405515639Rate Confirmation.pdf",
			"uploadedBy": "Dana Reyes",
		},
	}
	f := newFixture()
	f.reader.catalog = newCatalog(&agenttest.StubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{f.get, f.list, read},
	})
	id := pulid.MustNew("doc_").String()

	result := (&getTool{reader: f.reader}).Call(t.Context(), map[string]any{"id": id})

	require.False(t, result.Failed, result.Content)
	assert.True(t, strings.HasPrefix(result.Content, "Result from document_get:"))
	assert.Contains(t, result.Content, "Dana Reyes")
	require.Equal(t, 1, read.Calls)
	assert.Equal(t, map[string]any{"documentId": id}, read.LastParams.Params)
}

// What a call's record keeps to explain it: the read it went to and what that
// read was asked.
func TestGetRecord_NamesTheReadAndWhatItWasAsked(t *testing.T) {
	t.Parallel()

	f := newFixture()
	id := pulid.MustNew("shp_").String()

	result := (&getTool{reader: f.reader}).Call(t.Context(), map[string]any{"id": id})

	assert.Equal(t, "shipment_get", result.Read)
	assert.Equal(t, map[string]any{"shipmentId": id}, result.Params)
	assert.False(t, result.Refused)
}

func TestGetRecord_MarksAReadThePersonMayNotMakeAsRefused(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.Denied["shipment:read"] = true

	result := (&getTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"id": pulid.MustNew("shp_").String(),
	})

	assert.True(t, result.Failed)
	assert.True(t, result.Refused)
	assert.Equal(t, "shipment_get", result.Read)
}

// A read that ran and went wrong is a failure, not a refusal: the person may
// read it, and something underneath did not work.
func TestListRecords_MarksAReadThatFailedAsFailedNotRefused(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.list.Err = errors.New("an agent read needs the tool call's read scope")

	result := (&listTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"kind":    "shipment",
		"filters": map[string]any{"status": "InTransit"},
	})

	assert.True(t, result.Failed)
	assert.False(t, result.Refused)
	assert.Equal(t, "shipment_list", result.Read)
	assert.Equal(t, map[string]any{"status": "InTransit"}, result.Params)
	assert.Equal(t,
		"shipment_list failed: an agent read needs the tool call's read scope", result.Content)
}

func TestGetRecord_RefusesWhatThePersonCannotRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.Denied["shipment:read"] = true

	result := (&getTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"id": pulid.MustNew("shp_").String(),
	})

	assert.True(t, result.Failed)
	assert.Equal(t, "The person asking cannot read shipment records.", result.Content)
	assert.Zero(t, f.get.Calls)
}

func TestGetRecord_SendsAnythingButAnIDBackToSearch(t *testing.T) {
	t.Parallel()

	f := newFixture()

	result := (&getTool{reader: f.reader}).Call(t.Context(), map[string]any{"id": "S-104"})

	assert.True(t, result.Failed)
	assert.Contains(t, result.Content, "search_records")
	assert.Zero(t, f.get.Calls)
}

func TestListRecords_ListsAKindWithItsFiltersAndCountsThem(t *testing.T) {
	t.Parallel()

	f := newFixture()

	result := (&listTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"kind":    "shipment",
		"filters": map[string]any{"status": "InTransit"},
	})

	require.False(t, result.Failed, result.Content)
	require.Equal(t, 1, f.list.Calls)
	assert.Equal(t, map[string]any{"status": "InTransit"}, f.list.LastParams.Params)
	assert.True(t, f.list.LastParams.IncludeTotal)
}

// A model wrote {"kind": "shipment", "late": true, "query": "Gordon & Romero"}
// and the list ran with no filters at all, its most recent shipments read
// back as the customer's late ones.
func TestListRecords_ReadsAFilterWrittenBesideKind(t *testing.T) {
	t.Parallel()

	f := newFixture()

	result := (&listTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"kind":    "shipment",
		"status":  "InTransit",
		"filters": map[string]any{"after": "c1"},
	})

	require.False(t, result.Failed, result.Content)
	require.Equal(t, 1, f.list.Calls)
	assert.Equal(t, map[string]any{"status": "InTransit", "after": "c1"}, f.list.LastParams.Params)
}

func TestListRecords_ReadsAFilterNamedTwiceFromFilters(t *testing.T) {
	t.Parallel()

	f := newFixture()

	(&listTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"kind":    "shipment",
		"status":  "New",
		"filters": map[string]any{"status": "InTransit"},
	})

	require.Equal(t, 1, f.list.Calls)
	assert.Equal(t, map[string]any{"status": "InTransit"}, f.list.LastParams.Params)
}

func TestListRecords_RefusesAFilterBesideKindTheListDoesNotTake(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.list.Schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}

	result := (&listTool{reader: f.reader}).Call(t.Context(), map[string]any{
		"kind":  "shipment",
		"color": "blue",
	})

	assert.True(t, result.Failed)
	assert.Contains(t, result.Content, "color")
	assert.Zero(t, f.list.Calls, "a filter the list does not take is refused, never dropped")
}

func TestListRecords_RefusesAKindItCannotList(t *testing.T) {
	t.Parallel()

	f := newFixture()

	result := (&listTool{reader: f.reader}).Call(t.Context(), map[string]any{"kind": "invoice"})

	assert.True(t, result.Failed)
	assert.Equal(t, `"invoice" is not a kind list_records can list.`, result.Content)
}

func TestQuestionContext_FencesThePageAndTheMentionsAsData(t *testing.T) {
	t.Parallel()

	got := questionContext(&paletteports.Question{
		Content: "what about this one?",
		Page:    &agentpagecontext.PageContext{Path: "/shipments", Title: "Shipments"},
		Mentions: []agentpagecontext.EntityRef{
			{Type: "shipment", ID: "shp_1", Label: "S-104"},
		},
	})

	require.Len(t, got.Sections, 2)
	assert.Equal(t, "The page under the palette", got.Sections[0].Title)
	assert.False(t, got.Sections[0].Trusted)
	assert.Contains(t, got.Sections[0].Content, `"path":"/shipments"`)
	assert.Equal(t, "Mentioned records", got.Sections[1].Title)
	assert.False(t, got.Sections[1].Trusted)
	assert.Contains(t, got.Sections[1].Content, `"label":"S-104"`)

	assert.Empty(t, questionContext(&paletteports.Question{Content: "hi"}).Sections)
}
