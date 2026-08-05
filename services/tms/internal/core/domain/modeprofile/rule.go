package modeprofile

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*CapabilityRule)(nil)
	_ validationframework.TenantedEntity = (*CapabilityRule)(nil)
)

type CapabilityRule struct {
	bun.BaseModel `bun:"table:mode_profile_capability_rules,alias:mpr" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ModeProfileID  pulid.ID `json:"modeProfileId"  bun:"mode_profile_id,type:VARCHAR(100),notnull"`

	RuleKey     RuleKey                 `json:"ruleKey"     bun:"rule_key,type:VARCHAR(100),notnull"`
	Capability  Capability              `json:"capability"  bun:"capability,type:VARCHAR(100),notnull"`
	Enforcement tenant.EnforcementLevel `json:"enforcement" bun:"enforcement,type:enforcement_level_enum,notnull,default:'Block'"`
	Enabled     bool                    `json:"enabled"     bun:"enabled,type:BOOLEAN,notnull,default:true"`
	Parameters  map[string]any          `json:"parameters"  bun:"parameters,type:JSONB,nullzero"`

	OverrideReason string `json:"overrideReason" bun:"override_reason,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Profile *Profile `json:"-" bun:"rel:belongs-to,join:mode_profile_id=id"`
}

func (r *CapabilityRule) GetID() pulid.ID { return r.ID }

func (r *CapabilityRule) GetCreatedAt() int64 { return r.CreatedAt }

func (r *CapabilityRule) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *CapabilityRule) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *CapabilityRule) GetTableName() string { return "mode_profile_capability_rules" }

func (r *CapabilityRule) Blocks() bool {
	return r.Enabled && r.Enforcement == tenant.EnforcementLevelBlock
}

func (r *CapabilityRule) Records() bool {
	if !r.Enabled {
		return false
	}
	return r.Enforcement == tenant.EnforcementLevelWarn ||
		r.Enforcement == tenant.EnforcementLevelRequireReview
}

func (r *CapabilityRule) Applies() bool {
	return r.Enabled && r.Enforcement != tenant.EnforcementLevelIgnore
}

func (r *CapabilityRule) IntParam(name string) (int64, bool) {
	value, ok := r.resolveParam(name)
	if !ok {
		return 0, false
	}

	switch typed := value.(type) {
	case int64:
		return typed, true
	case int32:
		return int64(typed), true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case float32:
		return int64(typed), true
	default:
		return 0, false
	}
}

func (r *CapabilityRule) BoolParam(name string) (bool, bool) {
	value, ok := r.resolveParam(name)
	if !ok {
		return false, false
	}

	typed, isBool := value.(bool)
	return typed, isBool
}

func (r *CapabilityRule) resolveParam(name string) (any, bool) {
	if value, ok := r.Parameters[name]; ok && value != nil {
		return value, true
	}

	def, err := RuleDefinitionFor(r.RuleKey)
	if err != nil {
		return nil, false
	}

	for _, param := range def.Parameters {
		if param.Name == name && param.Default != nil {
			return param.Default, true
		}
	}

	return nil, false
}

func (r *CapabilityRule) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.RuleKey,
			validation.Required.Error("Rule key is required"),
		),
		validation.Field(&r.Enforcement,
			validation.Required.Error("Enforcement level is required"),
			domainvalidation.ValidEnum[tenant.EnforcementLevel]("Enforcement level is invalid"),
		),
	))

	if r.RuleKey != "" && !IsKnownRuleKey(r.RuleKey) {
		multiErr.Add("ruleKey", errortypes.ErrInvalid,
			"Rule key is not registered in the capability catalog")
		return
	}

	r.validateParameters(multiErr)
	r.validateOverrideReason(multiErr)
}

func (r *CapabilityRule) validateParameters(multiErr *errortypes.MultiError) {
	def, err := RuleDefinitionFor(r.RuleKey)
	if err != nil {
		return
	}

	known := make(map[string]RuleParameter, len(def.Parameters))
	for _, param := range def.Parameters {
		known[param.Name] = param
	}

	for name := range r.Parameters {
		if _, ok := known[name]; !ok {
			multiErr.Add("parameters", errortypes.ErrInvalid,
				"Parameter "+name+" is not accepted by "+def.Label)
		}
	}

	for _, param := range def.Parameters {
		if !param.Required {
			continue
		}
		if _, ok := r.resolveParam(param.Name); !ok {
			multiErr.Add("parameters", errortypes.ErrRequired,
				param.Label+" is required for "+def.Label)
		}
	}

	if r.RuleKey == RuleKeyMaxShipmentWeight {
		if weight, ok := r.IntParam(ParamMaxWeight); ok && weight <= 0 {
			multiErr.Add("parameters", errortypes.ErrInvalid,
				"Maximum weight must be greater than zero")
		}
	}
}

func (r *CapabilityRule) validateOverrideReason(multiErr *errortypes.MultiError) {
	def, err := RuleDefinitionFor(r.RuleKey)
	if err != nil {
		return
	}

	if r.Enforcement == def.DefaultEnforcement {
		return
	}

	if r.OverrideReason == "" {
		multiErr.Add("overrideReason", errortypes.ErrRequired,
			"Explain why "+def.Label+" differs from the recommended "+
				string(def.DefaultEnforcement)+" enforcement")
	}
}

func (r *CapabilityRule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if r.Enforcement == "" {
		r.Enforcement = tenant.EnforcementLevelBlock
	}
	if def, err := RuleDefinitionFor(r.RuleKey); err == nil {
		r.Capability = def.Capability
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("mpr_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
