package agentread

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestAuthContextFor(t *testing.T) {
	t.Parallel()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	user := pulid.MustNew("usr_")
	definition := pulid.MustNew("agd_")

	t.Run("a person reads as themselves", func(t *testing.T) {
		t.Parallel()

		auth, err := authContextFor(&permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeUser, PrincipalID: user, UserID: user,
			OrganizationID: org, BusinessUnitID: bu,
		})
		require.NoError(t, err)
		assert.Equal(t, authctx.PrincipalTypeUser, auth.PrincipalType)
		assert.Equal(t, user, auth.UserID)
		assert.Equal(t, user, auth.PrincipalID)
	})

	t.Run("an unattended agent reads as itself and names no person", func(t *testing.T) {
		t.Parallel()

		auth, err := authContextFor(&permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeAgent, PrincipalID: definition,
			UserID: pulid.MustNew("usr_"), OrganizationID: org, BusinessUnitID: bu,
		})
		require.NoError(t, err)
		assert.True(t, auth.IsAgent())
		assert.Equal(t, definition, auth.PrincipalID)
		assert.True(t, auth.UserID.IsNil(), "the system user an agent carries is attribution only")
	})

	refused := map[string]*permissionports.RequestActor{
		"no actor": nil,
		"an API key": {
			PrincipalType: permissionports.PrincipalTypeAPIKey, PrincipalID: pulid.MustNew("ak_"),
			OrganizationID: org, BusinessUnitID: bu,
		},
		"the system": {
			PrincipalType: permissionports.PrincipalTypeSystem, OrganizationID: org,
			BusinessUnitID: bu,
		},
		"a person with no tenant": {
			PrincipalType: permissionports.PrincipalTypeUser, PrincipalID: user, UserID: user,
		},
	}
	for name, actor := range refused {
		t.Run("refuses "+name, func(t *testing.T) {
			t.Parallel()

			_, err := authContextFor(actor)
			assert.ErrorIs(t, err, ErrUnsupportedActor)
		})
	}
}

func TestRead_RunsOnlyBoundOperations(t *testing.T) {
	t.Parallel()

	reader := &Reader{operations: map[string]Operation{"CustomerTable": {Name: "CustomerTable"}}}
	ctx := readscope.With(t.Context(), nil)

	_, err := reader.Read(ctx, &graphqlreadports.GraphQLReadRequest{Operation: "AnythingElse"})

	assert.ErrorIs(t, err, graphqlreadports.ErrUnknownReadOperation)
}

func TestRead_RefusesToReadWithoutTheToolCallsReadScope(t *testing.T) {
	t.Parallel()

	reader := &Reader{operations: map[string]Operation{"CustomerTable": {Name: "CustomerTable"}}}

	_, err := reader.Read(
		t.Context(),
		&graphqlreadports.GraphQLReadRequest{Operation: "CustomerTable"},
	)

	assert.ErrorIs(t, err, ErrNoReadScope)
}

func TestStrip_RemovesAWithheldValueAtItsPath(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"workers": map[string]any{
			"edges": []any{
				map[string]any{"node": map[string]any{"id": "wrk_1", "email": "a@x.com"}},
				map[string]any{"node": map[string]any{"id": "wrk_2", "email": "b@x.com"}},
			},
		},
	}

	strip(data, ast.Path{
		ast.PathName("workers"), ast.PathName("edges"), ast.PathIndex(1),
		ast.PathName("node"), ast.PathName("email"),
	})
	strip(data, ast.Path{ast.PathName("missing"), ast.PathName("email")})
	strip(data, ast.Path{ast.PathName("workers"), ast.PathName("edges"), ast.PathIndex(9)})

	edges := data["workers"].(map[string]any)["edges"].([]any)
	assert.Equal(t, "a@x.com", edges[0].(map[string]any)["node"].(map[string]any)["email"])
	assert.NotContains(t, edges[1].(map[string]any)["node"], "email")
}

func TestDecode_ReportsWithheldFieldsAndStripsNonNullOnes(t *testing.T) {
	t.Parallel()

	ceiling := gqlctx.NewFieldCeiling(nil)
	ceiling.Withhold(ast.Path{ast.PathName("worker"), ast.PathName("email")}, true)
	ceiling.Withhold(ast.Path{ast.PathName("worker"), ast.PathName("phoneNumber")}, false)

	result, err := decode(&graphql.Response{
		Data: []byte(`{"worker":{"id":"wrk_1","email":"a@x.com","phoneNumber":null}}`),
	}, ceiling)

	require.NoError(t, err)
	assert.Equal(t, []string{"worker.email", "worker.phoneNumber"}, result.Withheld)
	assert.Equal(t, map[string]any{"id": "wrk_1", "phoneNumber": nil}, result.Data["worker"])
}
