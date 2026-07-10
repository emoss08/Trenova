package email

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Profile)(nil)
	_ validationframework.TenantedEntity = (*Profile)(nil)
	_ domaintypes.PostgresSearchable     = (*Profile)(nil)
)

type Profile struct {
	bun.BaseModel `bun:"table:email_profiles,alias:ep" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID      `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Name           string        `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	Description    string        `json:"description"    bun:"description,type:TEXT,nullzero"`
	SenderName     string        `json:"senderName"     bun:"from_name,type:VARCHAR(255),notnull"`
	SenderEmail    string        `json:"senderEmail"    bun:"from_address,type:VARCHAR(255),notnull"`
	ReplyToEmail   string        `json:"replyToEmail"   bun:"reply_to,type:VARCHAR(255),nullzero"`
	Provider       Provider      `json:"provider"       bun:"provider_type,type:email_provider_type_enum,notnull"`
	AuthType       AuthType      `json:"-"              bun:"auth_type,type:email_auth_type_enum,notnull"`
	EncryptionType Encryption    `json:"-"              bun:"encryption_type,type:email_encryption_type_enum,notnull"`
	Status         ProfileStatus `json:"status"         bun:"status,type:status_enum,notnull"`
	Version        int64         `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string        `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
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
	err := validation.ValidateStruct(p,
		validation.Field(&p.Name, validation.Required.Error("Name is required"), validation.Length(1, 100)),
		validation.Field(&p.SenderName, validation.Required.Error("Sender name is required"), validation.Length(1, 100)),
		validation.Field(&p.SenderEmail, validation.Required.Error("Sender email is required"), validation.Length(1, 320)),
		validation.Field(
			&p.Provider,
			validation.Required,
			validation.In(ProviderResend, ProviderPostmark).Error("Invalid email provider"),
		),
		validation.Field(&p.Status, validation.Required, validation.In(ProfileStatusActive, ProfileStatusInactive).Error("Invalid status")),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
	if !strings.Contains(p.SenderEmail, "@") {
		multiErr.Add("senderEmail", errortypes.ErrInvalid, "Sender email must be a valid email address")
	}
	if p.ReplyToEmail != "" && !strings.Contains(p.ReplyToEmail, "@") {
		multiErr.Add("replyToEmail", errortypes.ErrInvalid, "Reply-to email must be a valid email address")
	}
}

func (p *Profile) GetID() pulid.ID             { return p.ID }
func (p *Profile) GetCreatedAt() int64         { return p.CreatedAt }
func (p *Profile) GetTableName() string        { return "email_profiles" }
func (p *Profile) GetOrganizationID() pulid.ID { return p.OrganizationID }
func (p *Profile) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Profile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ep",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "from_address", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "from_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (p *Profile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("emlprof_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}

type ProfileAssignment struct {
	bun.BaseModel `bun:"table:email_profile_assignments,alias:epa" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Purpose        Purpose  `json:"purpose"        bun:"purpose,type:email_purpose_enum,notnull"`
	ProfileID      pulid.ID `json:"profileId"      bun:"profile_id,type:VARCHAR(100),notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Profile *Profile `json:"profile,omitempty" bun:"rel:belongs-to,join:profile_id=id"`
}

func (a *ProfileAssignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("emlassn_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}
