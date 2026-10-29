package assistantartifacts

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type pagingListTool struct {
	agenttoolports.AgentQueryTool

	properties map[string]any
	result     map[string]any
	err        error
	calls      []agenttoolports.QueryToolParams
}

func (t *pagingListTool) Name() string { return "shipment_list" }

func (t *pagingListTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{Resource: permission.ResourceShipment}
}

func (t *pagingListTool) ParamSchema() map[string]any {
	return map[string]any{toolschema.KeyProperties: t.properties}
}

func (t *pagingListTool) Query(
	_ context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	t.calls = append(t.calls, *params)
	if t.err != nil {
		return nil, t.err
	}

	return t.result, nil
}

type pagingTools struct {
	agenttoolports.AgentQueryToolRegistry

	tool *pagingListTool
}

func (r pagingTools) Get(name string) (agenttoolports.AgentQueryTool, bool) {
	if name != r.tool.Name() {
		return nil, false
	}

	return r.tool, true
}

type rowsFixture struct {
	svc      *Service
	tool     *pagingListTool
	thread   *conversation.Thread
	artifact *assistantartifact.Artifact
}

func newRowsFixture(t *testing.T, engine permissionports.PermissionEngine) *rowsFixture {
	t.Helper()

	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	artifact := &assistantartifact.Artifact{
		ID:               pulid.MustNew("art_"),
		ThreadID:         thread.ID,
		Kind:             assistantartifact.KindTableView,
		SourceToolCallID: "call_1",
		Payload: map[string]any{
			artifactpayload.PayloadTool:   "shipment_list",
			artifactpayload.PayloadEntity: "shipments",
			"recordEntity":                "shipment",
			artifactpayload.PayloadColumns: []any{
				map[string]any{"key": "proNumber", "label": "Pro number", "type": "text"},
				map[string]any{"key": "customer", "label": "Customer", "type": "text"},
				map[string]any{"key": "freightCharge", "label": "Freight charge", "type": "money"},
			},
			artifactpayload.PayloadReadWith: map[string]any{
				"status": []any{"InTransit"}, "limit": float64(10), "after": "c-old",
			},
		},
	}
	tool := &pagingListTool{
		properties: map[string]any{
			argQuery: map[string]any{toolschema.KeyType: toolschema.TypeString},
			argSortBy: map[string]any{
				toolschema.KeyEnum: []string{"createdAt", "customer.name", "proNumber"},
			},
		},
		result: map[string]any{
			"items": []any{
				map[string]any{
					"id": "shp_1", "proNumber": "S-1001", "customer": "Acme Foods",
					"freightCharge": "1180.00", "internalNote": "not a column",
				},
			},
			"count":            float64(1),
			"hasMore":          true,
			"nextCursor":       "c-2",
			"total":            float64(128),
			"withheldByAccess": []any{"freightCharge"},
		},
	}
	repo := &assistanttest.StubArtifactRepo{
		Stored: map[pulid.ID]*assistantartifact.Artifact{artifact.ID: artifact},
	}

	return &rowsFixture{
		svc: &Service{
			logger:        zap.NewNop(),
			artifacts:     repo,
			conversations: &assistanttest.StubConversationRepo{Thread: thread},
			permissions:   engine,
			queries:       pagingTools{tool: tool},
		},
		tool:     tool,
		thread:   thread,
		artifact: artifact,
	}
}

func (f *rowsFixture) request() conversationports.GetThreadRequest {
	user := pulid.MustNew("usr_")

	return conversationports.GetThreadRequest{
		ID:     f.thread.ID,
		UserID: user,
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_"), UserID: user,
		},
	}
}

func TestArtifactRows_ReadsTheNextPageWithThePersonsAccess(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.AllowAllPermissionEngine{})
	req := f.request()

	rows, err := f.svc.ArtifactRows(
		t.Context(),
		req,
		f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{
			After:         "c-1",
			Limit:         25,
			Query:         "dallas",
			SortBy:        "customer",
			SortDirection: "DESC",
			IncludeTotal:  true,
		},
	)
	require.NoError(t, err)

	require.Len(t, f.tool.calls, 1)
	call := f.tool.calls[0]
	assert.Equal(t, map[string]any{
		"status":        []any{"InTransit"},
		"limit":         25,
		"after":         "c-1",
		"query":         "dallas",
		"sortBy":        "customer.name",
		"sortDirection": "desc",
	}, call.Params)
	assert.True(t, call.IncludeTotal)
	assert.Equal(t, req.UserID, call.Actor.UserID)
	assert.Equal(t, permissionports.PrincipalTypeUser, call.Actor.PrincipalType)
	assert.Equal(t, permission.SensitivityPublic, call.DataAccessCeiling)

	assert.Equal(t, []any{map[string]any{
		"id": "shp_1", "proNumber": "S-1001", "customer": "Acme Foods", "freightCharge": "1180.00",
	}}, rows.Rows)
	assert.True(t, rows.HasMore)
	assert.Equal(t, "c-2", rows.NextCursor)
	require.NotNil(t, rows.Total)
	assert.Equal(t, 128, *rows.Total)
	assert.False(t, rows.TotalEstimated)
	assert.Nil(t, rows.NextOffset)
	assert.Equal(t, []string{"freightCharge"}, rows.Withheld)
	assert.True(t, rows.Searchable)
	assert.Equal(t, []string{"customer", "proNumber"}, rows.Sortable)
}

func TestArtifactRows_ATableAlreadySearchedCannotBeSearchedAgain(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.AllowAllPermissionEngine{})
	f.artifact.Payload[artifactpayload.PayloadReadWith].(map[string]any)["query"] = "acme"

	rows, err := f.svc.ArtifactRows(
		t.Context(),
		f.request(),
		f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{Limit: 25},
	)
	require.NoError(t, err)
	assert.False(t, rows.Searchable)
	assert.Equal(t, "acme", f.tool.calls[0].Params[argQuery])

	_, err = f.svc.ArtifactRows(
		t.Context(), f.request(), f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{Limit: 25, Query: "dallas"},
	)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Len(t, f.tool.calls, 1)
}

func TestArtifactRows_RefusesAPersonWhoCannotReadTheList(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceShipment.String()},
	})

	_, err := f.svc.ArtifactRows(
		t.Context(),
		f.request(),
		f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{Limit: 25},
	)
	var refused *errortypes.AuthorizationError
	require.ErrorAs(t, err, &refused)
	assert.Empty(t, f.tool.calls)
}

func TestArtifactRows_OnlyAListTableIsPaged(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.AllowAllPermissionEngine{})
	f.artifact.Kind = assistantartifact.KindReportPreview

	_, err := f.svc.ArtifactRows(
		t.Context(),
		f.request(),
		f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{Limit: 25},
	)
	require.ErrorIs(t, err, errNotPageable)
	assert.Empty(t, f.tool.calls)
}

func TestArtifactRows_AnotherConversationsTableIsNotFound(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.AllowAllPermissionEngine{})
	f.artifact.ThreadID = pulid.MustNew("athr_")

	_, err := f.svc.ArtifactRows(
		t.Context(),
		f.request(),
		f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{Limit: 25},
	)
	var missing *errortypes.NotFoundError
	require.ErrorAs(t, err, &missing)
	assert.Empty(t, f.tool.calls)
}

func TestRowsArguments(t *testing.T) {
	t.Parallel()

	sortable := map[string]string{"customer": "customer.name"}
	cases := map[string]struct {
		query      assistantartifactports.ArtifactRowsQuery
		searchable bool
		want       map[string]any
		refused    string
	}{
		"a first page of the default size": {
			query: assistantartifactports.ArtifactRowsQuery{},
			want:  map[string]any{argLimit: 25},
		},
		"a page larger than any list hands back": {
			query: assistantartifactports.ArtifactRowsQuery{Limit: 500},
			want:  map[string]any{argLimit: 100},
		},
		"an offset": {
			query: assistantartifactports.ArtifactRowsQuery{Limit: 50, Offset: 50},
			want:  map[string]any{argLimit: 50, artifactpayload.ArgOffset: 50},
		},
		"a sort with no direction is ascending": {
			query: assistantartifactports.ArtifactRowsQuery{Limit: 25, SortBy: "customer"},
			want: map[string]any{
				argLimit: 25, argSortBy: "customer.name", argSortDirection: "asc",
			},
		},
		"a cursor and an offset": {
			query:   assistantartifactports.ArtifactRowsQuery{After: "c-1", Offset: 25},
			refused: artifactpayload.ArgOffset,
		},
		"a negative offset": {
			query:   assistantartifactports.ArtifactRowsQuery{Offset: -1},
			refused: artifactpayload.ArgOffset,
		},
		"a column the list cannot sort by": {
			query:   assistantartifactports.ArtifactRowsQuery{SortBy: "freightCharge"},
			refused: argSortBy,
		},
		"a direction that is neither": {
			query: assistantartifactports.ArtifactRowsQuery{
				SortBy:        "customer",
				SortDirection: "sideways",
			},
			refused: argSortDirection,
		},
		"a search of a list that takes none": {
			query:   assistantartifactports.ArtifactRowsQuery{Query: "dallas"},
			refused: argQuery,
		},
		"a search of a list that takes one": {
			query:      assistantartifactports.ArtifactRowsQuery{Query: "  dallas "},
			searchable: true,
			want:       map[string]any{argLimit: 25, argQuery: "dallas"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := rowsArguments(&tc.query, sortable, tc.searchable)
			if tc.refused != "" {
				var multiErr *errortypes.MultiError
				require.ErrorAs(t, err, &multiErr)
				require.NotEmpty(t, multiErr.Errors)
				assert.Equal(t, tc.refused, multiErr.Errors[0].Field)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestArtifactRows_AStaleCursorIsRefusedAsThePageNotAFault(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.AllowAllPermissionEngine{})
	f.tool.err = fmt.Errorf("read shipments: %w", querytool.ErrStaleCursor)

	_, err := f.svc.ArtifactRows(
		t.Context(),
		f.request(),
		f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{After: "c-1"},
	)
	require.ErrorIs(t, err, errStalePage)
}

func TestArtifactRows_ATableThatDoesNotSayHowItWasReadIsNotReadAgain(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.AllowAllPermissionEngine{})
	delete(f.artifact.Payload, artifactpayload.PayloadReadWith)

	_, err := f.svc.ArtifactRows(
		t.Context(),
		f.request(),
		f.artifact.ID,
		&assistantartifactports.ArtifactRowsQuery{Limit: 25},
	)
	require.ErrorIs(t, err, errNotReread)
	assert.Empty(t, f.tool.calls)
}
