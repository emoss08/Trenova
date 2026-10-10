package carrierintel

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CarrierIntelOverride)(nil)

type CarrierIntelOverride struct {
	bun.BaseModel `bun:"table:carrier_intel_overrides,alias:ciovr" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull"`
	RuleCode       RuleCode `json:"ruleCode"       bun:"rule_code,type:VARCHAR(100),notnull"`
	Reason         string   `json:"reason"         bun:"reason,type:TEXT,notnull"`
	GrantedByID    pulid.ID `json:"grantedById"    bun:"granted_by_id,type:VARCHAR(100),notnull"`
	GrantedAt      int64    `json:"grantedAt"      bun:"granted_at,type:BIGINT,notnull"`
	ExpiresAt      int64    `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	RevokedByID    pulid.ID `json:"revokedById"    bun:"revoked_by_id,type:VARCHAR(100),nullzero"`
	RevokedAt      *int64   `json:"revokedAt"      bun:"revoked_at,type:BIGINT,nullzero"`
	RevokeReason   string   `json:"revokeReason"   bun:"revoke_reason,type:TEXT,nullzero"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (o *CarrierIntelOverride) Validate(multiErr *errortypes.MultiError) {
	o.Reason = strings.TrimSpace(o.Reason)
	multiErr.AddOzzoError(validation.ValidateStruct(o,
		validation.Field(&o.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(&o.RuleCode, validation.Required.Error("Rule is required")),
		validation.Field(&o.Reason,
			validation.Required.Error("A reason is required to override a finding"),
			validation.Length(1, 2000).Error("Reason cannot exceed 2000 characters"),
		),
	))

	def, ok := RuleByCode(o.RuleCode)
	switch {
	case !ok:
		multiErr.Add("ruleCode", errortypes.ErrInvalid, "Rule is invalid")
	case !def.GateRelevant:
		multiErr.Add("ruleCode", errortypes.ErrInvalid,
			"Only findings that affect carrier eligibility can be overridden")
	}

	if o.ExpiresAt <= o.GrantedAt {
		multiErr.Add(
			"expiresAt",
			errortypes.ErrInvalid,
			"An override must expire after it is granted",
		)
	}
	if o.ExpiresAt-o.GrantedAt > MaxOverrideDurationSeconds {
		multiErr.Add(
			"expiresAt",
			errortypes.ErrInvalid,
			"An override cannot last longer than 90 days",
		)
	}
}

func (o *CarrierIntelOverride) IsActive(now int64) bool {
	return o.RevokedAt == nil && o.ExpiresAt > now
}

func (o *CarrierIntelOverride) Revoke(userID pulid.ID, reason string, now int64) error {
	if o.RevokedAt != nil {
		return errortypes.NewBusinessError("This override has already been revoked")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"A reason is required to revoke an override")
	}
	o.RevokedByID = userID
	o.RevokedAt = &now
	o.RevokeReason = reason
	return nil
}

func (o *CarrierIntelOverride) Ref() OverrideRef {
	return OverrideRef{ID: o.ID, RuleCode: o.RuleCode, ExpiresAt: o.ExpiresAt}
}

func OverrideRefs(overrides []*CarrierIntelOverride, now int64) []OverrideRef {
	refs := make([]OverrideRef, 0, len(overrides))
	for _, o := range overrides {
		if o != nil && o.IsActive(now) {
			refs = append(refs, o.Ref())
		}
	}
	return refs
}

func (o *CarrierIntelOverride) GetID() pulid.ID { return o.ID }

func (o *CarrierIntelOverride) GetOrganizationID() pulid.ID { return o.OrganizationID }

func (o *CarrierIntelOverride) GetBusinessUnitID() pulid.ID { return o.BusinessUnitID }

func (o *CarrierIntelOverride) GetTableName() string { return "carrier_intel_overrides" }

func (o *CarrierIntelOverride) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew("ciovr_")
		}
		o.CreatedAt = now
		o.UpdatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}
