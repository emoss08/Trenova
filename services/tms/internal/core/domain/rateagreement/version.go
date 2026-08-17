package rateagreement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateAgreementVersion)(nil)

// RateAgreementVersion records the negotiated header terms as they stood.
//
// It deliberately does not copy the rules. Rules carry their own effective
// windows, so "the contract as it stood on the third of March" is the header
// version live on that date plus whichever rules were live on that date — one
// query, and an amendment to forty lanes writes forty rows instead of copying
// every rule in the book.
type RateAgreementVersion struct {
	bun.BaseModel `bun:"table:rate_agreement_versions,alias:ragv" json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateAgreementID pulid.ID `json:"rateAgreementId" bun:"rate_agreement_id,type:VARCHAR(100),notnull"`

	VersionNumber int64  `json:"versionNumber" bun:"version_number,type:BIGINT,notnull"`
	EffectiveFrom int64  `json:"effectiveFrom" bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo   *int64 `json:"effectiveTo"   bun:"effective_to,type:BIGINT,nullzero"`

	Name          string        `json:"name"          bun:"name,type:VARCHAR(150),notnull"`
	Description   string        `json:"description"   bun:"description,type:TEXT,nullzero"`
	AgreementType AgreementType `json:"agreementType" bun:"agreement_type,type:rate_agreement_type_enum,notnull"`
	Status        Status        `json:"status"        bun:"status,type:rate_agreement_status_enum,notnull"`
	ContractRef   string        `json:"contractRef"   bun:"contract_ref,type:VARCHAR(100),nullzero"`

	Currency          string                 `json:"currency"          bun:"currency,type:VARCHAR(3),notnull"`
	DefaultMinCharge  decimal.NullDecimal    `json:"defaultMinCharge"  bun:"default_min_charge,type:NUMERIC(19,4),nullzero"`
	DefaultMaxCharge  decimal.NullDecimal    `json:"defaultMaxCharge"  bun:"default_max_charge,type:NUMERIC(19,4),nullzero"`
	RoundingMode      ratetypes.RoundingMode `json:"roundingMode"      bun:"rounding_mode,type:rate_rounding_mode_enum,notnull"`
	RoundingPrecision int16                  `json:"roundingPrecision" bun:"rounding_precision,type:SMALLINT,notnull"`

	MarginFloorPercent  decimal.NullDecimal `json:"marginFloorPercent"  bun:"margin_floor_percent,type:NUMERIC(9,4),nullzero"`
	MaxPayPercentOfSell decimal.NullDecimal `json:"maxPayPercentOfSell" bun:"max_pay_percent_of_sell,type:NUMERIC(9,4),nullzero"`

	ChangeMessage string                           `json:"changeMessage" bun:"change_message,type:TEXT,nullzero"`
	ChangeSummary map[string]jsonutils.FieldChange `json:"changeSummary" bun:"change_summary,type:JSONB,nullzero"`
	CreatedByID   pulid.ID                         `json:"createdById"   bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt     int64                            `json:"createdAt"     bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Agreement *RateAgreement `json:"-"                   bun:"rel:belongs-to,join:rate_agreement_id=id"`
	CreatedBy *tenant.User   `json:"createdBy,omitempty" bun:"rel:belongs-to,join:created_by_id=id"`
}

// IsEffectiveAt reports whether this version's terms governed at a moment.
func (ragv *RateAgreementVersion) IsEffectiveAt(timestamp int64) bool {
	if timestamp < ragv.EffectiveFrom {
		return false
	}

	if ragv.EffectiveTo != nil && timestamp >= *ragv.EffectiveTo {
		return false
	}

	return true
}

// NewVersionFromAgreement snapshots an agreement's current header terms.
func NewVersionFromAgreement(
	agreement *RateAgreement,
	versionNumber int64,
	effectiveFrom int64,
	createdByID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) *RateAgreementVersion {
	return &RateAgreementVersion{
		OrganizationID:      agreement.OrganizationID,
		BusinessUnitID:      agreement.BusinessUnitID,
		RateAgreementID:     agreement.ID,
		VersionNumber:       versionNumber,
		EffectiveFrom:       effectiveFrom,
		Name:                agreement.Name,
		Description:         agreement.Description,
		AgreementType:       agreement.AgreementType,
		Status:              agreement.Status,
		ContractRef:         agreement.ContractRef,
		Currency:            agreement.Currency,
		DefaultMinCharge:    agreement.DefaultMinCharge,
		DefaultMaxCharge:    agreement.DefaultMaxCharge,
		RoundingMode:        agreement.RoundingMode,
		RoundingPrecision:   agreement.RoundingPrecision,
		MarginFloorPercent:  agreement.MarginFloorPercent,
		MaxPayPercentOfSell: agreement.MaxPayPercentOfSell,
		ChangeMessage:       changeMessage,
		ChangeSummary:       changeSummary,
		CreatedByID:         createdByID,
	}
}

func (ragv *RateAgreementVersion) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(ragv,
		validation.Field(&ragv.RateAgreementID,
			validation.Required.Error("Agreement is required"),
		),
		validation.Field(&ragv.VersionNumber,
			validation.Required.Error("Version number is required"),
			validation.Min(int64(1)).Error("Version number must be at least one"),
		),
		validation.Field(&ragv.EffectiveFrom,
			validation.Required.Error("Effective from is required"),
		),
		validation.Field(&ragv.Name, validation.Required.Error("Name is required")),
		validation.Field(&ragv.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be a three letter code"),
		),
		validation.Field(&ragv.CreatedByID,
			validation.Required.Error("Created by is required"),
		),
	))

	if ragv.EffectiveTo != nil && *ragv.EffectiveTo <= ragv.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Effective to must be after effective from",
		)
	}
}

func (ragv *RateAgreementVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if ragv.ID.IsNil() {
			ragv.ID = pulid.MustNew("ragv_")
		}
		ragv.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (ragv *RateAgreementVersion) GetID() pulid.ID {
	return ragv.ID
}

func (ragv *RateAgreementVersion) GetCreatedAt() int64 {
	return ragv.CreatedAt
}

func (ragv *RateAgreementVersion) GetOrganizationID() pulid.ID {
	return ragv.OrganizationID
}

func (ragv *RateAgreementVersion) GetBusinessUnitID() pulid.ID {
	return ragv.BusinessUnitID
}

func (ragv *RateAgreementVersion) GetTableName() string {
	return "rate_agreement_versions"
}
