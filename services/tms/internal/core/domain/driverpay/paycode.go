package driverpay

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
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
	_ bun.BeforeAppendModelHook          = (*PayCode)(nil)
	_ pagination.CursorEntity            = (*PayCode)(nil)
	_ validationframework.TenantedEntity = (*PayCode)(nil)

	payCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]*$`)
)

type PayCodeDirection string

const (
	PayCodeDirectionEarning   = PayCodeDirection("Earning")
	PayCodeDirectionDeduction = PayCodeDirection("Deduction")
)

func (d PayCodeDirection) String() string { return string(d) }

func (d PayCodeDirection) IsValid() bool {
	switch d {
	case PayCodeDirectionEarning, PayCodeDirectionDeduction:
		return true
	default:
		return false
	}
}

type PayCode struct {
	bun.BaseModel             `bun:"table:pay_codes,alias:payc" json:"-"`
	pagination.CursorValueSet `bun:",embed"                     json:"-"`

	ID                    pulid.ID           `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID           `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID           `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Status                domaintypes.Status `json:"status"                bun:"status,type:status_enum,notnull,default:'Active'"`
	Direction             PayCodeDirection   `json:"direction"             bun:"direction,type:VARCHAR(20),notnull"`
	Code                  string             `json:"code"                  bun:"code,type:VARCHAR(20),notnull"`
	Name                  string             `json:"name"                  bun:"name,type:VARCHAR(100),notnull"`
	Description           string             `json:"description"           bun:"description,type:TEXT,nullzero"`
	Taxable               bool               `json:"taxable"               bun:"taxable,type:BOOLEAN,notnull,default:true"`
	CountsTowardGuarantee bool               `json:"countsTowardGuarantee" bun:"counts_toward_guarantee,type:BOOLEAN,notnull,default:true"`
	GLAccountID           *pulid.ID          `json:"glAccountId"           bun:"gl_account_id,type:VARCHAR(100),nullzero"`
	DefaultAmountMinor    *int64             `json:"defaultAmountMinor"    bun:"default_amount_minor,type:BIGINT,nullzero"`
	IsSystem              bool               `json:"isSystem"              bun:"is_system,type:BOOLEAN,notnull,default:false"`
	Version               int64              `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt             int64              `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64              `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	GLAccount *glaccount.GLAccount `json:"glAccount,omitempty" bun:"rel:belongs-to,join:gl_account_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (p *PayCode) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(p,
		validation.Field(&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 20).Error("Code must be between 1 and 20 characters"),
		),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if p.Code != "" && !payCodePattern.MatchString(p.Code) {
		multiErr.Add(
			"code",
			errortypes.ErrInvalid,
			"Code must use uppercase letters, digits, dashes, or underscores and start with a letter or digit",
		)
	}
	if !p.Direction.IsValid() {
		multiErr.Add("direction", errortypes.ErrInvalid, "Direction must be Earning or Deduction")
	}
	if p.Status != domaintypes.StatusActive && p.Status != domaintypes.StatusInactive {
		multiErr.Add("status", errortypes.ErrInvalid, "Status must be either Active or Inactive")
	}
	if p.DefaultAmountMinor != nil && *p.DefaultAmountMinor <= 0 {
		multiErr.Add(
			"defaultAmountMinor",
			errortypes.ErrInvalid,
			"Default amount must be greater than zero when provided",
		)
	}
}

func (p *PayCode) LineIsReimbursement() bool {
	return p.Direction == PayCodeDirectionEarning && !p.Taxable
}

func (p *PayCode) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "payc",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   searchFieldDescription,
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (p *PayCode) GetID() pulid.ID { return p.ID }

func (p *PayCode) GetCreatedAt() int64 { return p.CreatedAt }

func (p *PayCode) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *PayCode) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *PayCode) GetTableName() string { return "pay_codes" }

func (p *PayCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("payc_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}

type SystemPayCode struct {
	Direction PayCodeDirection
	Code      string
	Name      string
	Taxable   bool
}

func SystemPayCodes() []SystemPayCode {
	return []SystemPayCode{
		{PayCodeDirectionEarning, "PERDIEM", "Per Diem", false},
		{PayCodeDirectionEarning, "SAFETY", "Safety Bonus", true},
		{PayCodeDirectionEarning, "PERFORM", "Performance Bonus", true},
		{PayCodeDirectionEarning, "LONGEVITY", "Longevity Bonus", true},
		{PayCodeDirectionEarning, "STIPEND", "Stipend", false},
		{PayCodeDirectionEarning, "EQUIPRENT", "Equipment Rental", true},
		{PayCodeDirectionEarning, "OTHER", "Other Earning", true},
		{PayCodeDirectionDeduction, "INSUR", "Insurance", true},
		{PayCodeDirectionDeduction, "TRKLEASE", "Truck Lease", true},
		{PayCodeDirectionDeduction, "TRLLEASE", "Trailer Lease", true},
		{PayCodeDirectionDeduction, "ELD", "ELD Service", true},
		{PayCodeDirectionDeduction, "FUELCARD", "Fuel Card", true},
		{PayCodeDirectionDeduction, "ESCROW", "Escrow Contribution", true},
		{PayCodeDirectionDeduction, "LOAN", "Loan Repayment", true},
		{PayCodeDirectionDeduction, "OTHER", "Other Deduction", true},
	}
}
