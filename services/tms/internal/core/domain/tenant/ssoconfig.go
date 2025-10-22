package tenant

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*SSOConfig)(nil)
	_ domain.Validatable        = (*SSOConfig)(nil)
	_ framework.TenantedEntity  = (*SSOConfig)(nil)
)

// SSOConfig represents the SSO (Single Sign-On) configuration for an organization.
// Currently supports OIDC (OpenID Connect) protocol for integration with various
// identity providers (Okta, Azure AD, Auth0, Google Workspace, etc.)
//
// Implementation uses coreos/go-oidc library for OIDC support.
//
// SECURITY CONSIDERATIONS:
// - OIDCClientSecret MUST be encrypted at rest using KMS or Vault
// - Consider using envelope encryption (KMS + application-level encryption)
// - Audit all access to SSO configurations (create, read, update, delete)
// - Rotate client secrets regularly (90-day cycle recommended)
// - Never log sensitive fields (client secret)
// - Implement rate limiting on SSO endpoints to prevent abuse
// - Use state parameter to prevent CSRF attacks (handled by OAuth2 library)
// - Validate redirect URIs strictly - only allow exact matches
// - Store encryption metadata (algorithm, key version) for rotation
type SSOConfig struct {
	bun.BaseModel `json:"-" bun:"table:sso_configs,alias:sso"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Name           string      `json:"name"           bun:"name,type:VARCHAR(100),notnull"`                   // Friendly name for this SSO config (e.g., "Okta Production")
	Provider       SSOProvider `json:"provider"       bun:"provider,type:sso_provider_enum,notnull"`          // Provider type (Okta, AzureAD, Auth0, Google, GenericOIDC)
	Protocol       SSOProtocol `json:"protocol"       bun:"protocol,type:sso_protocol_enum,notnull"`          // Currently only OIDC supported
	Enabled        bool        `json:"enabled"        bun:"enabled,type:BOOLEAN,notnull,default:false"`       // Whether this SSO config is active
	EnforceSSO     bool        `json:"enforceSSO"     bun:"enforce_sso,type:BOOLEAN,notnull,default:false"`   // Require SSO for all users in this org (disable password login)
	AutoProvision  bool        `json:"autoProvision"  bun:"auto_provision,type:BOOLEAN,notnull,default:true"` // Automatically create user accounts on first SSO login
	DefaultRole    string      `json:"defaultRole"    bun:"default_role,type:VARCHAR(50)"`                    // Default role for auto-provisioned users
	AllowedDomains []string    `json:"allowedDomains" bun:"allowed_domains,type:TEXT[],array"`                // Email domains allowed to SSO (e.g., ["company.com", "contractor.com"])
	AttributeMap   *Attributes `json:"attributeMap"   bun:"attribute_map,type:JSONB"`                         // Maps SSO attributes to user fields

	// OIDC configuration (using coreos/go-oidc library)
	// See: https://github.com/coreos/go-oidc
	OIDCIssuerURL    string   `json:"oidcIssuerUrl"   bun:"oidc_issuer_url,type:VARCHAR(500),notnull"`   // OIDC Issuer URL (e.g., https://accounts.google.com)
	OIDCClientID     string   `json:"oidcClientId"    bun:"oidc_client_id,type:VARCHAR(255),notnull"`    // OAuth2 Client ID from your IdP
	OIDCClientSecret string   `json:"-"               bun:"oidc_client_secret,type:TEXT,notnull"`        // ⚠️ ENCRYPT: OAuth2 Client Secret - MUST BE ENCRYPTED BEFORE STORAGE
	OIDCRedirectURL  string   `json:"oidcRedirectUrl" bun:"oidc_redirect_url,type:VARCHAR(500),notnull"` // Callback URL for OIDC flow (e.g., https://yourapp.com/auth/callback)
	OIDCScopes       []string `json:"oidcScopes"      bun:"oidc_scopes,type:TEXT[],array,notnull"`       // OIDC scopes (minimum: ["openid", "profile", "email"])

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

// Attributes maps SSO provider attributes to application user fields
// Example: {"email": "email", "firstName": "given_name", "lastName": "family_name", "name": "name"}
//
// Common OIDC standard claims:
// - email (string): Email address
// - given_name (string): Given name(s) or first name(s)
// - family_name (string): Surname(s) or last name(s)
// - name (string): Full name
// - preferred_username (string): Shorthand name
// - picture (string): Profile picture URL
// - locale (string): End-User's locale (e.g., "en-US")
type Attributes map[string]string

func (sc *SSOConfig) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(sc,
		validation.Field(&sc.Name,
			validation.Required.Error("SSO configuration name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters")),
		validation.Field(&sc.Provider,
			validation.Required.Error("SSO provider is required"),
			validation.In(
				SSOProviderOkta,
				SSOProviderAzureAD,
				SSOProviderAuth0,
				SSOProviderGoogle,
				SSOProviderGenericOIDC,
			).Error("Invalid SSO provider")),
		validation.Field(&sc.Protocol,
			validation.Required.Error("SSO protocol is required"),
			validation.In(
				SSOProtocolOIDC,
			).Error("Invalid SSO protocol (only OIDC currently supported)")),
		// OIDC validations
		validation.Field(&sc.OIDCIssuerURL,
			validation.Required.Error("OIDC Issuer URL is required"),
			is.URL.Error("OIDC Issuer URL must be a valid URL")),
		validation.Field(&sc.OIDCClientID,
			validation.Required.Error("OIDC Client ID is required"),
			validation.Length(1, 255).Error("OIDC Client ID must be between 1 and 255 characters")),
		validation.Field(&sc.OIDCClientSecret,
			validation.Required.Error("OIDC Client Secret is required")),
		validation.Field(&sc.OIDCRedirectURL,
			validation.Required.Error("OIDC Redirect URL is required"),
			is.URL.Error("OIDC Redirect URL must be a valid URL")),
		validation.Field(
			&sc.OIDCScopes,
			validation.Required.Error("OIDC Scopes are required"),
			validation.Length(1, 0).
				Error("At least one OIDC scope is required (must include 'openid')"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	// Additional validation: ensure 'openid' scope is present
	if len(sc.OIDCScopes) > 0 {
		hasOpenID := false
		for _, scope := range sc.OIDCScopes {
			if scope == "openid" {
				hasOpenID = true
				break
			}
		}
		if !hasOpenID {
			multiErr.Add(
				"oidcScopes",
				errortypes.ErrInvalid,
				"OIDC scopes must include 'openid' scope. Please try again",
			)
		}
	}
}

func (sc *SSOConfig) GetID() string {
	return sc.ID.String()
}

func (sc *SSOConfig) GetTableName() string {
	return "sso_configs"
}

func (sc *SSOConfig) GetOrganizationID() pulid.ID {
	return sc.OrganizationID
}

func (sc *SSOConfig) GetBusinessUnitID() pulid.ID {
	return sc.BusinessUnitID
}

func (sc *SSOConfig) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sc.ID.IsNil() {
			sc.ID = pulid.MustNew("sso_")
		}

		// NOTE: Encryption should be handled in the service layer, not domain
		// Service will use encryption.Service to encrypt OIDCClientSecret before insert
		// Example: encryptedSecret, err := encryptionService.Encrypt(config.OIDCClientSecret)

		sc.CreatedAt = now
	case *bun.UpdateQuery:
		// NOTE: Encryption should be handled in the service layer, not domain
		// Service will use encryption.Service to encrypt OIDCClientSecret before update

		sc.UpdatedAt = now
	}

	return nil
}

// EncryptSensitiveFields encrypts sensitive fields before storing in database
//
// TODO: CRITICAL - Implement encryption for OIDCClientSecret
//
// Recommended implementation approach:
//  1. Choose encryption strategy:
//     a. AWS KMS (recommended for AWS deployments)
//     b. Google Cloud KMS (recommended for GCP deployments)
//     c. HashiCorp Vault (recommended for multi-cloud/on-prem)
//     d. Application-level encryption with secure key storage
//
// 2. Implement envelope encryption:
//   - Generate a Data Encryption Key (DEK) for each secret
//   - Encrypt the DEK with a Key Encryption Key (KEK) from KMS/Vault
//   - Store encrypted DEK alongside encrypted data
//
// 3. Store encryption metadata:
//
//   - Algorithm used (e.g., AES-256-GCM)
//
//   - Key version (for rotation)
//
//   - Initialization vector (IV)
//
//     4. Example implementation structure:
//     type EncryptedValue struct {
//     Ciphertext string
//     Algorithm  string
//     KeyVersion string
//     IV         string
//     }
//
// 5. Add audit logging:
//   - Log encryption operations (but NEVER log the plaintext values)
//   - Track key usage for compliance
//
// Example pseudocode:
//
//	func (sc *SSOConfig) EncryptSensitiveFields(kms KMSClient) error {
//	    if sc.OIDCClientSecret != "" {
//	        encrypted, metadata, err := kms.Encrypt([]byte(sc.OIDCClientSecret))
//	        if err != nil {
//	            return fmt.Errorf("failed to encrypt client secret: %w", err)
//	        }
//	        // Store as JSON with metadata
//	        sc.OIDCClientSecret = encodeEncryptedValue(encrypted, metadata)
//	    }
//	    return nil
//	}
func (sc *SSOConfig) EncryptSensitiveFields() error {
	// TODO: Implementation required
	return nil
}

// DecryptSensitiveFields decrypts sensitive fields after loading from database
//
// TODO: CRITICAL - Implement decryption for OIDCClientSecret
//
// This should be called after loading from database and before using the config.
// Consider:
// 1. Caching decrypted values in memory (with TTL)
// 2. Auditing all decryption operations
// 3. Rate limiting decryption requests
// 4. Handling key rotation gracefully
//
// Example pseudocode:
//
//	func (sc *SSOConfig) DecryptSensitiveFields(kms KMSClient) error {
//	    if sc.OIDCClientSecret != "" {
//	        encrypted, metadata, err := decodeEncryptedValue(sc.OIDCClientSecret)
//	        if err != nil {
//	            return fmt.Errorf("failed to decode encrypted value: %w", err)
//	        }
//	        plaintext, err := kms.Decrypt(encrypted, metadata)
//	        if err != nil {
//	            return fmt.Errorf("failed to decrypt client secret: %w", err)
//	        }
//	        sc.OIDCClientSecret = string(plaintext)
//	    }
//	    return nil
//	}
func (sc *SSOConfig) DecryptSensitiveFields() error {
	// TODO: Implementation required
	return nil
}
