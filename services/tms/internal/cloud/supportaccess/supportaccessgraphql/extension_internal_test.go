package supportaccessgraphql

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

type recorder struct {
	refusals []*supportaccessservice.RefusalRequest
}

func (r *recorder) RecordRefusal(_ context.Context, req *supportaccessservice.RefusalRequest) {
	r.refusals = append(r.refusals, req)
}

func operation(kind ast.Operation, field, source string) *graphql.OperationContext {
	return &graphql.OperationContext{
		Operation: &ast.OperationDefinition{
			Operation: kind,
			SelectionSet: ast.SelectionSet{
				&ast.Field{
					Name: field,
					Definition: &ast.FieldDefinition{
						Name:     field,
						Position: &ast.Position{Src: &ast.Source{Name: "schema/" + source}},
					},
				},
			},
		},
	}
}

func supportContext(t *testing.T, elevatedUntil int64) context.Context {
	t.Helper()
	return supportctx.WithActive(t.Context(), &supportctx.Active{
		SessionID:       pulid.MustNew("sps_"),
		OrganizationID:  pulid.MustNew("org_"),
		PrincipalUserID: pulid.MustNew("usr_"),
		GrantMode:       supportaccess.AccessModeReadWrite,
		ElevatedUntil:   elevatedUntil,
		Now:             1000,
	})
}

func TestOrdinaryRequestsAreUntouched(t *testing.T) {
	t.Parallel()
	ext := &Extension{recorder: &recorder{}}

	err := ext.MutateOperationContext(
		t.Context(),
		operation(ast.Mutation, "updateShipment", "shipment.graphqls"),
	)
	assert.Nil(t, err)
}

func TestReadOnlySessionsRefuseEveryMutationButNotQueries(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	ext := &Extension{recorder: rec}
	ctx := supportContext(t, 0)

	assert.Nil(t, ext.MutateOperationContext(ctx, operation(ast.Query, "shipments", "shipment.graphqls")))

	err := ext.MutateOperationContext(
		ctx,
		operation(ast.Mutation, "updateShipment", "shipment.graphqls"),
	)
	require.NotNil(t, err)
	assert.Equal(t, ReadOnlyErrorCode, err.Extensions["code"])
	require.Len(t, rec.refusals, 1)
	assert.False(t, rec.refusals[0].Denied)
}

func TestElevatedSessionsMutateExceptDeniedSources(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	ext := &Extension{recorder: rec}
	ctx := supportContext(t, 2000)

	assert.Nil(t, ext.MutateOperationContext(
		ctx,
		operation(ast.Mutation, "updateShipment", "shipment.graphqls"),
	))

	err := ext.MutateOperationContext(ctx, operation(ast.Mutation, "createApiKey", "api_key.graphqls"))
	require.NotNil(t, err)
	assert.Equal(t, DeniedErrorCode, err.Extensions["code"])
	require.Len(t, rec.refusals, 1)
	assert.True(t, rec.refusals[0].Denied)
}
