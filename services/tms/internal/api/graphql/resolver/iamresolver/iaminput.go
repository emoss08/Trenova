package iamresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const defaultProvisioningAuditLimit = 100

func stringMapToJSON(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func stringMapFromJSON(field string, values map[string]any) (map[string]string, error) {
	if values == nil {
		return nil, nil
	}

	out := make(map[string]string, len(values))
	for key, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, errortypes.NewValidationError(
				field,
				errortypes.ErrInvalid,
				"Every value must be text",
			)
		}
		out[key] = text
	}

	return out, nil
}

func identityProviderRequestFromInput(
	input *gqlmodel.IdentityProviderInput,
) (*iamports.IdentityProviderRequest, error) {
	attributeMap, err := stringMapFromJSON("attributeMap", input.AttributeMap)
	if err != nil {
		return nil, err
	}

	return &iamports.IdentityProviderRequest{
		Name:              input.Name,
		Slug:              gqlscalar.StringValue(input.Slug),
		Protocol:          iam.IdentityProviderProtocol(input.Protocol),
		Enabled:           input.Enabled,
		EnforceSSO:        input.EnforceSso,
		AutoProvision:     input.AutoProvision,
		AllowFederatedMFA: input.AllowFederatedMfa,
		FederatedMFAACRs:  input.FederatedMfaAcrValues,
		AllowedDomains:    input.AllowedDomains,
		AttributeMap:      attributeMap,
		OIDCIssuerURL:     input.OidcIssuerURL,
		OIDCClientID:      input.OidcClientID,
		OIDCClientSecret:  gqlscalar.StringValue(input.OidcClientSecret),
		OIDCRedirectURL:   input.OidcRedirectURL,
		OIDCScopes:        input.OidcScopes,
		Version:           gqlscalar.Int64Value(input.Version),
	}, nil
}

func accessPolicyFromInput(input *gqlmodel.AccessPolicyInput) (*iam.AccessPolicy, error) {
	conditions, err := stringMapFromJSON("conditions", input.Conditions)
	if err != nil {
		return nil, err
	}

	return &iam.AccessPolicy{
		Name:       input.Name,
		Resource:   input.Resource,
		Operation:  input.Operation,
		Effect:     iam.PolicyEffect(input.Effect),
		Priority:   input.Priority,
		Conditions: conditions,
		Enabled:    input.Enabled,
	}, nil
}

func scimGroupRoleMappingFromInput(
	input *gqlmodel.SCIMGroupRoleMappingInput,
) (*iam.SCIMGroupRoleMapping, error) {
	roleID, err := gqlid.ParseIDField("roleId", input.RoleID)
	if err != nil {
		return nil, err
	}

	return &iam.SCIMGroupRoleMapping{
		ExternalGroupID: input.ExternalGroupID,
		DisplayName:     gqlscalar.StringValue(input.DisplayName),
		RoleID:          roleID,
		Version:         gqlscalar.Int64Value(input.Version),
	}, nil
}
