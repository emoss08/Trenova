package email

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Profile)(nil)
	_ domaintypes.PostgresSearchable     = (*Profile)(nil)
)

type Profile struct {
	keel.Table `                                             keel:"email_profiles"`

	ID             pulid.ID      `json:"id" keel:"id,pk,autoid:emlprof_"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID      `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name           string        `json:"name" keel:"name"`
	Description    string        `json:"description" keel:"description,nullzero"`
	SenderName     string        `json:"senderName" keel:"from_name,nullzero"`
	SenderEmail    string        `json:"senderEmail" keel:"from_address"`
	ReplyToEmail   string        `json:"replyToEmail" keel:"reply_to,nullzero"`
	Provider       Provider      `json:"provider" keel:"provider_type"`
	AuthType       AuthType      `json:"-" keel:"auth_type"`
	EncryptionType Encryption    `json:"-" keel:"encryption_type"`
	Status         ProfileStatus `json:"status" keel:"status,nullzero"`
	Version        int64         `json:"version" keel:"version,version"`
	CreatedAt      int64         `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64         `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector   string        `json:"-" keel:"search_vector,tsvector,readonly,noselect"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (p *Profile) Validate(multiErr *errortypes.MultiError) {
	if p.Provider == "" {
		p.Provider = ProviderResend
	}
	if p.AuthType == "" {
		p.AuthType = AuthTypeAPIKey
	}
	if p.EncryptionType == "" {
		p.EncryptionType = EncryptionNone
	}
	if p.Status == "" {
		p.Status = ProfileStatusActive
	}
	multiErr.AddOzzoError(validation.ValidateStruct(
		p,
		validation.Field(
			&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100),
		),
		validation.Field(
			&p.SenderName,
			validation.Required.Error("Sender name is required"),
			validation.Length(1, 100),
		),
		validation.Field(
			&p.SenderEmail,
			validation.Required.Error("Sender email is required"),
			validation.Length(1, 320),
		),
		validation.Field(
			&p.Provider,
			validation.Required,
			validation.In(ProviderResend, ProviderPostmark).Error("Invalid email provider"),
		),
		validation.Field(
			&p.Status,
			validation.Required,
			validation.In(ProfileStatusActive, ProfileStatusInactive).Error("Invalid status"),
		),
	))
	if !strings.Contains(p.SenderEmail, "@") {
		multiErr.Add(
			"senderEmail",
			errortypes.ErrInvalid,
			"Sender email must be a valid email address",
		)
	}
	if p.ReplyToEmail != "" && !strings.Contains(p.ReplyToEmail, "@") {
		multiErr.Add(
			"replyToEmail",
			errortypes.ErrInvalid,
			"Reply-to email must be a valid email address",
		)
	}
}

func (p *Profile) GetID() pulid.ID             { return p.ID }
func (p *Profile) GetCreatedAt() int64         { return p.CreatedAt }
func (p *Profile) GetTableName() string        { return "email_profiles" }
func (p *Profile) GetOrganizationID() pulid.ID { return p.OrganizationID }
func (p *Profile) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

// profileSearchDescription is the description column the profile's search
// vector matches.
const profileSearchDescription = "description"

func (p *Profile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ep",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "from_address",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "host", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   profileSearchDescription,
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

type ProfileAssignment struct {
	keel.Table `                                                         keel:"email_profile_assignments"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:emlassn_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	Purpose        Purpose  `json:"purpose" keel:"purpose"`
	ProfileID      pulid.ID `json:"profileId" keel:"profile_id"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`

	Profile *Profile `json:"profile,omitempty" keel:"belongs:profile_id"`
}

func (a *ProfileAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&a.Purpose,
			validation.Required.Error("Purpose is required"),
			domainvalidation.ValidEnum[Purpose]("Purpose is invalid"),
		),
		validation.Field(&a.ProfileID, validation.Required.Error("Email profile is required")),
	))
}
