package querytooltest

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func WithEdges[T any](page *pagination.CursorListResult[T]) *pagination.CursorListResult[T] {
	page.EdgeCursors = make([]string, len(page.Items))
	for idx := range page.Items {
		page.EdgeCursors[idx] = "edge-" + strconv.Itoa(idx)
	}

	return page
}

func ConnectionPage(root string, nodes ...map[string]any) map[string]any {
	edges := make([]any, 0, len(nodes))
	for _, node := range nodes {
		edges = append(edges, map[string]any{"node": node})
	}

	return map[string]any{root: map[string]any{
		"edges":    edges,
		"pageInfo": map[string]any{"hasNextPage": true, "endCursor": NextPage},
	}}
}

// RestrictedRole is a person whose role reaches Restricted fields.
type RestrictedRole struct {
	mocks.DenyResourcesPermissionEngine
}

func (RestrictedRole) GetResourcePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{
		MaxSensitivity: permission.SensitivityRestricted,
	}, nil
}

// KeyedReader answers each operation with what a test gives it, and keeps
// every request it was sent.
type KeyedReader struct {
	mu       sync.Mutex
	Answers  map[string]*graphqlreadports.GraphQLReadResult
	Requests []*graphqlreadports.GraphQLReadRequest
}

func (r *KeyedReader) Read(
	_ context.Context,
	req *graphqlreadports.GraphQLReadRequest,
) (*graphqlreadports.GraphQLReadResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Requests = append(r.Requests, req)
	if answer, ok := r.Answers[req.Operation]; ok {
		return answer, nil
	}

	return &graphqlreadports.GraphQLReadResult{Data: map[string]any{}}, nil
}

func (r *KeyedReader) Sent(operation string) *graphqlreadports.GraphQLReadRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.Requests {
		if req.Operation == operation {
			return req
		}
	}

	return nil
}

func ListSpecNamed(t *testing.T, specs []readspec.List, name string) *readspec.List {
	t.Helper()

	for _, spec := range specs {
		if spec.Name == name {
			return &spec
		}
	}
	require.FailNow(t, "no spec "+name)

	return nil
}

func KeyedCall(params map[string]any) *agenttoolports.QueryToolParams {
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	return &agenttoolports.QueryToolParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Timezone:       "America/Chicago",
		Params:         params,
		RunID:          pulid.MustNew("ar_"),
	}
}

func DispatchCall(params map[string]any) *agenttoolports.QueryToolParams {
	call := KeyedCall(params)
	call.DataAccessCeiling = permission.SensitivityInternal

	return call
}

func SentInput(t *testing.T, reader *KeyedReader) map[string]any {
	t.Helper()

	reader.mu.Lock()
	defer reader.mu.Unlock()
	require.NotEmpty(t, reader.Requests)
	input, ok := reader.Requests[0].Variables["input"].(map[string]any)
	require.True(t, ok)

	return input
}

// NextPage is a real connection cursor, as the root hands one back.
var NextPage, _ = pagination.EncodeKeysetCursor(nil, pulid.MustNew("acct_").String())
