package email

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Profile)(nil)
	_ domain.Validatable        = (*Profile)(nil)
	_ infra.PostgresSearchable  = (*Profile)(nil)
)

// Profile represents an email configuration profile
type Profile struct {
	bun.BaseModel `bun:"table:email_profiles,alias:ep" json:"-"`

	// Primary identifiers
	ID                 pulid.ID       `json:"id"                       bun:"id,type:varchar(100),pk,notnull"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"           bun:"business_unit_id,type:varchar(100),pk,notnull"`
	OrganizationID     pulid.ID       `json:"organizationId"           bun:"organization_id,type:varchar(100),pk,notnull"`
	Name               string         `json:"name"                     bun:"name,type:varchar(255),notnull"`
	Description        string         `json:"description"              bun:"description,type:text"`
	Status             domain.Status  `json:"status"                   bun:"status,type:status_enum,default:'active'"`
	ProviderType       ProviderType   `json:"providerType"             bun:"provider_type,type:email_provider_type_enum,notnull"`
	AuthType           AuthType       `json:"authType"                 bun:"auth_type,type:email_auth_type_enum,notnull"`
	EncryptionType     EncryptionType `json:"encryptionType"           bun:"encryption_type,type:email_encryption_type_enum,notnull"`
	Host               string         `json:"host"                     bun:"host,type:varchar(255)"`
	Username           string         `json:"username"                 bun:"username,type:varchar(255)"`
	EncryptedPassword  string         `json:"-"                        bun:"encrypted_password,type:text"`
	EncryptedAPIKey    string         `json:"-"                        bun:"encrypted_api_key,type:text"`
	OAuth2ClientID     string         `json:"oauth2ClientId,omitempty" bun:"oauth2_client_id,type:varchar(255)"`
	OAuth2ClientSecret string         `json:"-"                        bun:"oauth2_client_secret,type:text"`
	OAuth2TenantID     string         `json:"oauth2TenantId,omitempty" bun:"oauth2_tenant_id,type:varchar(255)"`
	FromAddress        string         `json:"fromAddress"              bun:"from_address,type:varchar(255),notnull"`
	FromName           string         `json:"fromName"                 bun:"from_name,type:varchar(255)"`
	ReplyTo            string         `json:"replyTo"                  bun:"reply_to,type:varchar(255)"`
	Port               int            `json:"port"                     bun:"port,type:integer"`
	MaxConnections     int            `json:"maxConnections"           bun:"max_connections,type:integer,default:5"`
	TimeoutSeconds     int            `json:"timeoutSeconds"           bun:"timeout_seconds,type:integer,default:30"`
	RetryCount         int            `json:"retryCount"               bun:"retry_count,type:integer,default:3"`
	RetryDelaySeconds  int            `json:"retryDelaySeconds"        bun:"retry_delay_seconds,type:integer,default:5"`
	RateLimitPerMinute int            `json:"rateLimitPerMinute"       bun:"rate_limit_per_minute,type:integer,default:60"`
	RateLimitPerHour   int            `json:"rateLimitPerHour"         bun:"rate_limit_per_hour,type:integer,default:1000"`
	RateLimitPerDay    int            `json:"rateLimitPerDay"          bun:"rate_limit_per_day,type:integer,default:10000"`
	Version            int64          `json:"version"                  bun:"version,type:BIGINT"`
	CreatedAt          int64          `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64          `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Metadata           map[string]any `json:"metadata"                 bun:"metadata,type:jsonb"`
	IsDefault          bool           `json:"isDefault"                bun:"is_default,type:boolean,default:false"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`

	// Transient fields for handling plain text values (not persisted)
	Password string `json:"password,omitempty" bun:"-"`
	APIKey   string `json:"apiKey,omitempty"   bun:"-"`
}

// Validate implements the Validatable interface
func (p *Profile) Validate( //nolint:funlen // This is a validation function, this is fine.
	ctx context.Context,
	multiErr *errors.MultiError,
) {
	err := validation.ValidateStructWithContext(ctx, p,
		// Basic fields validation
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business Unit is required"),
		),
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&p.Description,
			validation.Length(0, 1000).Error("Description must not exceed 1000 characters"),
		),
		validation.Field(&p.ProviderType,
			validation.Required.Error("Provider Type is required"),
			validation.In(
				ProviderTypeSMTP,
				ProviderTypeSendGrid,
				ProviderTypeAWSSES,
				ProviderTypeMailgun,
				ProviderTypePostmark,
				ProviderTypeExchange,
				ProviderTypeOffice365,
			).Error("Provider Type must be a valid provider"),
		),
		validation.Field(&p.AuthType,
			validation.Required.Error("Auth Type is required"),
			validation.In(
				AuthTypePlain,
				AuthTypeLogin,
				AuthTypeCRAMMD5,
				AuthTypeOAuth2,
				AuthTypeAPIKey,
			).Error("Auth Type must be a valid authentication method"),
		),
		validation.Field(&p.EncryptionType,
			validation.Required.Error("Encryption Type is required"),
			validation.In(
				EncryptionTypeNone,
				EncryptionTypeSSLTLS,
				EncryptionTypeSTARTTLS,
			).Error("Encryption Type must be a valid encryption method"),
		),
		validation.Field(
			&p.Status,
			validation.In(domain.StatusActive, domain.StatusInactive).
				Error("Status must be Active or Inactive"),
		),
		validation.Field(&p.FromAddress,
			validation.Required.Error("From Address is required"),
			is.Email.Error("From Address must be a valid email"),
		),
		validation.Field(&p.ReplyTo,
			validation.When(p.ReplyTo != "", is.Email.Error("Reply To must be a valid email")),
		),

		// Performance settings validation
		validation.Field(&p.MaxConnections,
			validation.Min(1).Error("Max Connections must be at least 1"),
			validation.Max(100).Error("Max Connections must not exceed 100"),
		),
		validation.Field(&p.TimeoutSeconds,
			validation.Min(5).Error("Timeout must be at least 5 seconds"),
			validation.Max(300).Error("Timeout must not exceed 300 seconds"),
		),
		validation.Field(&p.RetryCount,
			validation.Min(0).Error("Retry Count must be at least 0"),
			validation.Max(10).Error("Retry Count must not exceed 10"),
		),
		validation.Field(&p.RetryDelaySeconds,
			validation.Min(1).Error("Retry Delay must be at least 1 second"),
			validation.Max(60).Error("Retry Delay must not exceed 60 seconds"),
		),

		// Rate limiting validation
		validation.Field(&p.RateLimitPerMinute,
			validation.Min(1).Error("Rate Limit Per Minute must be at least 1"),
			validation.Max(1000).Error("Rate Limit Per Minute must not exceed 1000"),
		),
		validation.Field(&p.RateLimitPerHour,
			validation.Min(1).Error("Rate Limit Per Hour must be at least 1"),
			validation.Max(100000).Error("Rate Limit Per Hour must not exceed 100000"),
		),
		validation.Field(&p.RateLimitPerDay,
			validation.Min(1).Error("Rate Limit Per Day must be at least 1"),
			validation.Max(1000000).Error("Rate Limit Per Day must not exceed 1000000"),
		),

		// Provider-specific validation for SMTP
		validation.Field(&p.Host,
			validation.When(
				p.ProviderType == ProviderTypeSMTP,
				validation.Required.Error("Host is required for SMTP provider"),
				validation.Length(1, 255).Error("Host must be between 1 and 255 characters"),
			),
		),
		validation.Field(&p.Port,
			validation.When(
				p.ProviderType == ProviderTypeSMTP,
				validation.Required.Error("Port is required for SMTP provider"),
				validation.Min(1).Error("Port must be at least 1"),
				validation.Max(65535).Error("Port must not exceed 65535"),
			),
		),
		validation.Field(&p.Username,
			validation.When(
				p.ProviderType == ProviderTypeSMTP && p.AuthType != AuthTypeAPIKey,
				validation.Required.Error(
					"Username is required for SMTP provider with this auth type",
				),
			),
		),
		validation.Field(&p.EncryptedPassword,
			validation.When(
				p.ProviderType == ProviderTypeSMTP &&
					(p.AuthType == AuthTypePlain || p.AuthType == AuthTypeLogin),
				validation.Required.Error(
					"Password is required for SMTP provider with Plain or Login auth",
				),
			),
		),

		// API Key validation for API-based providers
		validation.Field(&p.EncryptedAPIKey,
			validation.When(
				p.ProviderType == ProviderTypeSendGrid || p.ProviderType == ProviderTypeMailgun ||
					p.ProviderType == ProviderTypePostmark,
				validation.Required.Error("API Key is required for this provider"),
			),
		),

		// AWS SES validation
		validation.Field(&p.Metadata,
			validation.When(
				p.ProviderType == ProviderTypeAWSSES,
				validation.Required.Error("Metadata configuration is required for AWS SES"),
			),
		),

		// Exchange/Office365 validation
		validation.Field(&p.Username,
			validation.When(
				p.ProviderType == ProviderTypeExchange || p.ProviderType == ProviderTypeOffice365,
				validation.Required.Error("Username is required for Exchange/Office365"),
				is.Email.Error("Username must be a valid email for Exchange/Office365"),
			),
		),
		validation.Field(&p.OAuth2ClientID,
			validation.When(
				(p.ProviderType == ProviderTypeExchange || p.ProviderType == ProviderTypeOffice365) &&
					p.AuthType == AuthTypeOAuth2,
				validation.Required.Error("OAuth2 Client ID is required for OAuth2 authentication"),
			),
		),
		validation.Field(&p.OAuth2ClientSecret,
			validation.When(
				(p.ProviderType == ProviderTypeExchange || p.ProviderType == ProviderTypeOffice365) &&
					p.AuthType == AuthTypeOAuth2,
				validation.Required.Error(
					"OAuth2 Client Secret is required for OAuth2 authentication",
				),
			),
		),
		validation.Field(&p.OAuth2TenantID,
			validation.When(
				(p.ProviderType == ProviderTypeExchange || p.ProviderType == ProviderTypeOffice365) &&
					p.AuthType == AuthTypeOAuth2,
				validation.Required.Error("OAuth2 Tenant ID is required for OAuth2 authentication"),
			),
		),
	)

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetConnectionInfo returns connection information based on provider type
func (p *Profile) GetConnectionInfo() map[string]any {
	info := map[string]any{
		"provider": p.ProviderType,
		"from":     p.FromAddress,
	}

	switch p.ProviderType {
	case ProviderTypeSMTP:
		info["host"] = p.Host
		info["port"] = p.Port
		info["encryption"] = p.EncryptionType
	case ProviderTypeSendGrid, ProviderTypeMailgun, ProviderTypePostmark:
		info["hasAPIKey"] = p.EncryptedAPIKey != ""
	case ProviderTypeExchange, ProviderTypeOffice365:
		info["authType"] = p.AuthType
		if p.AuthType == AuthTypeOAuth2 {
			info["tenantID"] = p.OAuth2TenantID
		}
	}

	return info
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (p *Profile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID == "" {
			p.ID = pulid.MustNew("emp_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

func (p *Profile) GetTableName() string {
	return "email_profiles"
}

// GetPostgresSearchConfig implements the PostgresSearchable interface
func (p *Profile) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "ep",
		Fields: []infra.PostgresSearchableField{
			{Name: "name", Weight: "A", Type: infra.PostgresSearchTypeText},
			{Name: "from_address", Weight: "B", Type: infra.PostgresSearchTypeText},
			{Name: "host", Weight: "C", Type: infra.PostgresSearchTypeText},
			{Name: "description", Weight: "D", Type: infra.PostgresSearchTypeText},
		},
		MinLength:       2,
		MaxTerms:        5,
		UsePartialMatch: true,
	}
}
