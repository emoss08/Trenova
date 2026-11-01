package apikeyresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/apikeyports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func apiKeyTenant(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
}

func apiKeyPermissionsFromInput(
	inputs []*gqlmodel.APIKeyPermissionInput,
) []apikeyports.APIKeyPermissionInput {
	out := make([]apikeyports.APIKeyPermissionInput, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		ops := make([]permission.Operation, len(input.Operations))
		for i, op := range input.Operations {
			ops[i] = permission.Operation(op)
		}
		out = append(out, apikeyports.APIKeyPermissionInput{
			Resource:   input.Resource,
			Operations: ops,
			DataScope:  permission.DataScope(input.DataScope),
		})
	}
	return out
}

func createAPIKeyRequestFromInput(input *gqlmodel.APIKeyInput) *apikeyports.CreateAPIKeyRequest {
	return &apikeyports.CreateAPIKeyRequest{
		Name:        input.Name,
		Description: gqlscalar.StringValue(input.Description),
		ExpiresAt:   gqlscalar.Int64Value(input.ExpiresAt),
		Permissions: apiKeyPermissionsFromInput(input.Permissions),
	}
}

func updateAPIKeyRequestFromInput(input *gqlmodel.APIKeyInput) *apikeyports.UpdateAPIKeyRequest {
	return &apikeyports.UpdateAPIKeyRequest{
		Name:        input.Name,
		Description: gqlscalar.StringValue(input.Description),
		ExpiresAt:   gqlscalar.Int64Value(input.ExpiresAt),
		Permissions: apiKeyPermissionsFromInput(input.Permissions),
	}
}

func issuedAPIKeyToModel(issued *apikeyports.APIKeySecretResponse) *gqlmodel.IssuedAPIKey {
	detail := issued.APIKeyResponse
	return &gqlmodel.IssuedAPIKey{
		APIKey: &detail,
		Token:  issued.Token,
	}
}
