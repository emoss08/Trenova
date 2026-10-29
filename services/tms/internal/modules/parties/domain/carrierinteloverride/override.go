package carrierinteloverride

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelrule"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CarrierIntelOverride struct {
	keel.Table `keel:"carrier_intel_overrides"`

	ID             pulid.ID                  `json:"id"             keel:"id,pk,autoid:ciovr_"`
	BusinessUnitID pulid.ID                  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID                  `json:"organizationId" keel:"organization_id,pk,tenant"`
	CarrierID      pulid.ID                  `json:"carrierId"      keel:"carrier_id"`
	RuleCode       carrierintelrule.RuleCode `json:"ruleCode"       keel:"rule_code"`
	Reason         string                    `json:"reason"         keel:"reason"`
	GrantedByID    pulid.ID                  `json:"grantedById"    keel:"granted_by_id"`
	GrantedAt      int64                     `json:"grantedAt"      keel:"granted_at"`
	ExpiresAt      int64                     `json:"expiresAt"      keel:"expires_at"`
	RevokedByID    pulid.ID                  `json:"revokedById"    keel:"revoked_by_id,nullzero"`
	RevokedAt      *int64                    `json:"revokedAt"      keel:"revoked_at"`
	RevokeReason   string                    `json:"revokeReason"   keel:"revoke_reason,nullzero"`
	Version        int64                     `json:"version"        keel:"version,version"`
	CreatedAt      int64                     `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64                     `json:"updatedAt"      keel:"updated_at,autoupdate"`
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

	def, ok := carrierintelrule.RuleByCode(o.RuleCode)
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
	if o.ExpiresAt-o.GrantedAt > carrierintelcontrol.MaxOverrideDurationSeconds {
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

func (o *CarrierIntelOverride) Ref() carrierintelrule.OverrideRef {
	return carrierintelrule.OverrideRef{ID: o.ID, RuleCode: o.RuleCode, ExpiresAt: o.ExpiresAt}
}

func OverrideRefs(overrides []*CarrierIntelOverride, now int64) []carrierintelrule.OverrideRef {
	refs := make([]carrierintelrule.OverrideRef, 0, len(overrides))
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
