package carrier

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CarrierInsurancePolicy)(nil)

type CarrierInsurancePolicy struct {
	bun.BaseModel `bun:"table:carrier_insurance_policies,alias:carins" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull"`

	PolicyType     InsurancePolicyType `json:"policyType"     bun:"policy_type,type:VARCHAR(50),notnull"`
	PolicyNumber   string              `json:"policyNumber"   bun:"policy_number,type:VARCHAR(100),notnull"`
	ProviderName   string              `json:"providerName"   bun:"provider_name,type:VARCHAR(255),notnull"`
	CoverageAmount decimal.Decimal     `json:"coverageAmount" bun:"coverage_amount,type:NUMERIC(19,4),notnull"`
	EffectiveDate  int64               `json:"effectiveDate"  bun:"effective_date,type:BIGINT,notnull"`
	ExpirationDate int64               `json:"expirationDate" bun:"expiration_date,type:BIGINT,notnull"`
	IsVerified     bool                `json:"isVerified"     bun:"is_verified,type:BOOLEAN,notnull,default:false"`
	Version        int64               `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Carrier *Carrier `json:"carrier,omitempty" bun:"rel:belongs-to,join:carrier_id=id"`
}

func (cip *CarrierInsurancePolicy) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		cip,
		validation.Field(&cip.PolicyType,
			validation.Required.Error("Policy type is required"),
			domainvalidation.ValidEnum[InsurancePolicyType]("Policy type is invalid"),
		),
		validation.Field(&cip.PolicyNumber,
			validation.Required.Error("Policy number is required"),
			validation.Length(1, 100).Error("Policy number must be between 1 and 100 characters"),
		),
		validation.Field(&cip.ProviderName,
			validation.Required.Error("Provider name is required"),
			validation.Length(1, 255).Error("Provider name must be between 1 and 255 characters"),
		),
		validation.Field(&cip.EffectiveDate,
			validation.Required.Error("Effective date is required"),
			validation.Min(int64(1)).Error("Effective date must be a positive value"),
		),
		validation.Field(&cip.ExpirationDate,
			validation.Required.Error("Expiration date is required"),
			validation.Min(int64(1)).Error("Expiration date must be a positive value"),
		),
	))

	if cip.CoverageAmount.IsNegative() {
		multiErr.Add("coverageAmount", errortypes.ErrInvalid, "Coverage amount cannot be negative")
	}

	if cip.ExpirationDate <= cip.EffectiveDate {
		multiErr.Add(
			"expirationDate",
			errortypes.ErrInvalid,
			"Expiration date must be after the effective date",
		)
	}
}

func (cip *CarrierInsurancePolicy) IsExpired(now int64) bool {
	return cip.ExpirationDate <= now
}

func (cip *CarrierInsurancePolicy) ExpiresWithin(now, seconds int64) bool {
	return !cip.IsExpired(now) && cip.ExpirationDate <= now+seconds
}

func (cip *CarrierInsurancePolicy) GetID() pulid.ID {
	return cip.ID
}

func (cip *CarrierInsurancePolicy) GetCreatedAt() int64 {
	return cip.CreatedAt
}

func (cip *CarrierInsurancePolicy) GetTableName() string {
	return "carrier_insurance_policies"
}

func (cip *CarrierInsurancePolicy) GetOrganizationID() pulid.ID {
	return cip.OrganizationID
}

func (cip *CarrierInsurancePolicy) GetBusinessUnitID() pulid.ID {
	return cip.BusinessUnitID
}

func (cip *CarrierInsurancePolicy) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if cip.ID.IsNil() {
			cip.ID = pulid.MustNew("carins_")
		}

		cip.CreatedAt = now
	case *bun.UpdateQuery:
		cip.UpdatedAt = now
	}

	return nil
}
