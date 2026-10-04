package cloudsignup

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

const (
	MaxEmailLength           = 255
	MaxNameLength            = 255
	MaxCompanyNameLength     = 255
	MaxClientIPLength        = 64
	MaxUserAgentLength       = 512
	MaxRejectionReasonLength = 100
	TokenHashLength          = 64
)

var (
	_ bun.BeforeAppendModelHook = (*CloudSignup)(nil)

	tokenHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type CloudSignup struct {
	bun.BaseModel `bun:"table:cloud_signups,alias:csu" json:"-"`

	ID                        pulid.ID `json:"id"                                  bun:"id,pk,type:VARCHAR(100),notnull"`
	EmailAddress              string   `json:"emailAddress"                        bun:"email_address,type:VARCHAR(255),notnull"`
	EmailNormalized           string   `json:"emailNormalized"                     bun:"email_normalized,type:VARCHAR(255),notnull"`
	Name                      string   `json:"name"                                bun:"name,type:VARCHAR(255),notnull"`
	CompanyName               string   `json:"companyName"                         bun:"company_name,type:VARCHAR(255),notnull"`
	PasswordHash              string   `json:"-"                                   bun:"password_hash,type:TEXT,notnull"`
	TokenHash                 string   `json:"-"                                   bun:"token_hash,type:VARCHAR(64),notnull"`
	Status                    Status   `json:"status"                              bun:"status,type:VARCHAR(20),notnull"`
	ClientIP                  string   `json:"clientIp,omitempty"                  bun:"client_ip,type:VARCHAR(64),nullzero"`
	UserAgent                 string   `json:"userAgent,omitempty"                 bun:"user_agent,type:VARCHAR(512),nullzero"`
	Attempts                  int      `json:"attempts"                            bun:"attempts,type:INTEGER,notnull"`
	RejectionReason           string   `json:"rejectionReason,omitempty"           bun:"rejection_reason,type:VARCHAR(100),nullzero"`
	ExpiresAt                 int64    `json:"expiresAt"                           bun:"expires_at,type:BIGINT,notnull"`
	VerifiedAt                *int64   `json:"verifiedAt"                          bun:"verified_at,type:BIGINT,nullzero"`
	ProvisionedOrganizationID pulid.ID `json:"provisionedOrganizationId,omitempty" bun:"provisioned_organization_id,type:VARCHAR(100),nullzero"`
	ProvisionedBusinessUnitID pulid.ID `json:"provisionedBusinessUnitId,omitempty" bun:"provisioned_business_unit_id,type:VARCHAR(100),nullzero"`
	ProvisionedUserID         pulid.ID `json:"provisionedUserId,omitempty"         bun:"provisioned_user_id,type:VARCHAR(100),nullzero"`
	CreatedAt                 int64    `json:"createdAt"                           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                 int64    `json:"updatedAt"                           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (c *CloudSignup) IsExpired(now int64) bool {
	return now >= c.ExpiresAt
}

func (c *CloudSignup) IsPending() bool {
	return c.Status == StatusPending
}

func (c *CloudSignup) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		c,
		validation.Field(
			&c.EmailAddress,
			validation.Required.Error("Email address is required"),
			validation.RuneLength(1, MaxEmailLength).
				Error("Email address must be at most 255 characters"),
			is.EmailFormat.Error("Email address must be a valid email address"),
		),
		validation.Field(
			&c.EmailNormalized,
			validation.Required.Error("Normalized email address is required"),
			validation.RuneLength(1, MaxEmailLength).
				Error("Normalized email address must be at most 255 characters"),
		),
		validation.Field(
			&c.Name,
			validation.Required.Error("Name is required"),
			validation.RuneLength(1, MaxNameLength).Error("Name must be at most 255 characters"),
		),
		validation.Field(
			&c.CompanyName,
			validation.Required.Error("Company name is required"),
			validation.RuneLength(1, MaxCompanyNameLength).
				Error("Company name must be at most 255 characters"),
		),
		validation.Field(&c.PasswordHash, validation.Required.Error("Password is required")),
		validation.Field(
			&c.TokenHash,
			validation.Required.Error("Verification token is required"),
			validation.Match(tokenHashPattern).
				Error("Verification token hash must be 64 lowercase hexadecimal characters"),
		),
		validation.Field(
			&c.Status,
			validation.Required.Error("Status is required"),
			validation.By(func(any) error {
				if !c.Status.IsValid() {
					return validation.NewError("validation_invalid", "Status is invalid")
				}
				return nil
			}),
		),
		validation.Field(
			&c.ClientIP,
			validation.RuneLength(0, MaxClientIPLength).
				Error("Client IP must be at most 64 characters"),
		),
		validation.Field(
			&c.UserAgent,
			validation.RuneLength(0, MaxUserAgentLength).
				Error("User agent must be at most 512 characters"),
		),
		validation.Field(
			&c.Attempts,
			validation.Min(0).Error("Attempts must not be negative"),
		),
		validation.Field(
			&c.RejectionReason,
			validation.RuneLength(0, MaxRejectionReasonLength).
				Error("Rejection reason must be at most 100 characters"),
		),
		validation.Field(&c.ExpiresAt, validation.Required.Error("Expiry is required")),
	))
}

func (c *CloudSignup) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("csu_")
		}
		if c.CreatedAt == 0 {
			c.CreatedAt = now
		}
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
