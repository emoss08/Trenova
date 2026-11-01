package customfielddefinitionresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customfieldportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func formFieldsResolver(
	repo customfieldports.CustomFieldDefinitionRepository,
	engine *resolvertest.RecordingPermissionEngine,
) *QueryResolver {
	return &QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
		CustomFieldService: customfieldservice.New(customfieldservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}}
}

func formFieldsContext(t *testing.T, orgID, buID pulid.ID) *authctx.AuthContext {
	t.Helper()

	userID := pulid.MustNew("usr_")
	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

func TestCustomFieldFormFields_NeedsReadOnTheFormsResourceNotOnDefinitions(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	definitions := []*customfield.CustomFieldDefinition{
		{ID: pulid.MustNew("cfd_"), ResourceType: "worker", Label: "Shirt size", DisplayOrder: 1},
		{ID: pulid.MustNew("cfd_"), ResourceType: "worker", Label: "Badge", DisplayOrder: 2},
	}
	repo := customfieldportsmock.NewMockCustomFieldDefinitionRepository(t)
	repo.EXPECT().
		GetActiveByResourceType(mock.Anything, mock.MatchedBy(func(req customfieldports.GetActiveByResourceTypeRequest) bool {
			return req.ResourceType == "worker" &&
				req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID
		})).
		Return(definitions, nil).
		Once()
	engine := &resolvertest.RecordingPermissionEngine{}
	ctx := gqlctx.WithAuthContext(t.Context(), formFieldsContext(t, orgID, buID))

	fields, err := formFieldsResolver(repo, engine).CustomFieldFormFields(ctx, "worker")
	require.NoError(t, err)

	assert.Equal(t, definitions, fields)
	require.NotNil(t, engine.Request)
	assert.Equal(t, permission.ResourceWorker.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpRead, engine.Request.Operation)
}

func TestCustomFieldFormFields_RefusesAResourceTypeWithoutCustomFields(t *testing.T) {
	t.Parallel()

	repo := customfieldportsmock.NewMockCustomFieldDefinitionRepository(t)
	engine := &resolvertest.RecordingPermissionEngine{}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		formFieldsContext(t, pulid.MustNew("org_"), pulid.MustNew("bu_")),
	)

	_, err := formFieldsResolver(repo, engine).CustomFieldFormFields(ctx, "invoice")

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "resourceType", validation.Field)
	assert.Nil(t, engine.Request)
	repo.AssertNotCalled(t, "GetActiveByResourceType", mock.Anything, mock.Anything)
}

func TestCustomFieldFormFields_EveryResourceTypeIsAPermissionResource(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	for _, resourceType := range customfield.GetSupportedResourceTypes() {
		assert.Truef(
			t,
			registry.HasResource(resourceType),
			"custom field resource type %q has no permission resource to check read on",
			resourceType,
		)
	}
}

func TestCustomFieldResourceTypes_NeedsCustomFieldDefinitionRead(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.RecordingPermissionEngine{}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		formFieldsContext(t, pulid.MustNew("org_"), pulid.MustNew("bu_")),
	)

	types, err := formFieldsResolver(
		customfieldportsmock.NewMockCustomFieldDefinitionRepository(t),
		engine,
	).CustomFieldResourceTypes(ctx)
	require.NoError(t, err)

	assert.Equal(t, customfield.GetSupportedResourceTypes(), types)
	require.NotNil(t, engine.Request)
	assert.Equal(t, permission.ResourceCustomFieldDefinition.String(), engine.Request.Resource)
}
