package email

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*EmailProfile)(nil)
	_ domain.Validatable             = (*EmailProfile)(nil)
	_ framework.TenantedEntity       = (*EmailProfile)(nil)
	_ domaintypes.PostgresSearchable = (*EmailProfile)(nil)
)

//nolint:revive // it's a valid struct name
type EmailProfile struct {
	bun.BaseModel `bun:"table:email_profiles,alias:ep" json:"-"`

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
	SearchVector       string         `json:"-"                        bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank               string         `json:"-"                        bun:"rank,type:VARCHAR(100),scanonly"`
	Password           string         `json:"password,omitempty"       bun:"-"`
	APIKey             string         `json:"apiKey,omitempty"         bun:"-"`
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
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (p *EmailProfile) Validate( //nolint:funlen // this is a validation function
	multiErr *errortypes.MultiError,
) {
	err := validation.ValidateStruct(p,
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
				ProviderTypeResend,
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
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (p *EmailProfile) GetConnectionInfo() map[string]any {
	info := map[string]any{
		"provider": p.ProviderType,
		"from":     p.FromAddress,
	}

	switch p.ProviderType { //nolint:exhaustive // mailhog is only for development
	case ProviderTypeSMTP:
		info["host"] = p.Host
		info["port"] = p.Port
		info["encryption"] = p.EncryptionType
	case ProviderTypeResend:
		info["hasAPIKey"] = p.EncryptedAPIKey != ""
	}

	return info
}

func (p *EmailProfile) GetTableName() string {
	return "email_profiles"
}

func (p *EmailProfile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ep",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Weight: domaintypes.SearchWeightA, Type: domaintypes.FieldTypeText},
			{
				Name:   "from_address",
				Weight: domaintypes.SearchWeightB,
				Type:   domaintypes.FieldTypeText,
			},
			{Name: "host", Weight: domaintypes.SearchWeightC, Type: domaintypes.FieldTypeText},
			{
				Name:   "description",
				Weight: domaintypes.SearchWeightD,
				Type:   domaintypes.FieldTypeText,
			},
		},
	}
}

func (p *EmailProfile) GetID() string {
	return p.ID.String()
}

func (p *EmailProfile) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

func (p *EmailProfile) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *EmailProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("emp_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
