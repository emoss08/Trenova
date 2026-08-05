package modeprofile

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	SpecificityWeightCustomer      int32 = 16
	SpecificityWeightEquipmentType int32 = 8
	SpecificityWeightServiceType   int32 = 4
	SpecificityWeightShipmentType  int32 = 2
)

var (
	_ bun.BeforeAppendModelHook          = (*Profile)(nil)
	_ validationframework.TenantedEntity = (*Profile)(nil)
	_ domaintypes.PostgresSearchable     = (*Profile)(nil)
)

type Profile struct {
	bun.BaseModel             `bun:"table:mode_profiles,alias:mpf" json:"-"`
	pagination.CursorValueSet `bun:",embed"                       json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name           string        `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Code           string        `json:"code"           bun:"code,type:VARCHAR(50),notnull"`
	Description    string        `json:"description"    bun:"description,type:TEXT,nullzero"`
	Status         ProfileStatus `json:"status"         bun:"status,type:mode_profile_status_enum,notnull,default:'Draft'"`

	ServiceModel   ServiceModel   `json:"serviceModel"   bun:"service_model,type:mode_profile_service_model_enum,notnull"`
	EquipmentClass EquipmentClass `json:"equipmentClass" bun:"equipment_class,type:mode_profile_equipment_class_enum,notnull"`
	ExecutionParty ExecutionParty `json:"executionParty" bun:"execution_party,type:mode_profile_execution_party_enum,notnull,default:'CompanyAsset'"`
	Capabilities   []Capability   `json:"capabilities"   bun:"capabilities,type:JSONB,nullzero"`

	IsOrgDefault     bool       `json:"isOrgDefault"     bun:"is_org_default,type:BOOLEAN,notnull,default:false"`
	Priority         int16      `json:"priority"         bun:"priority,type:SMALLINT,notnull,default:0"`
	SpecificityScore int32      `json:"specificityScore" bun:"specificity_score,type:INTEGER,notnull,default:0"`
	CustomerID       *pulid.ID  `json:"customerId"       bun:"customer_id,type:VARCHAR(100),nullzero"`
	ShipmentTypeIDs  []pulid.ID `json:"shipmentTypeIds"  bun:"shipment_type_ids,type:JSONB,nullzero"`
	ServiceTypeIDs   []pulid.ID `json:"serviceTypeIds"   bun:"service_type_ids,type:JSONB,nullzero"`
	EquipmentTypeIDs []pulid.ID `json:"equipmentTypeIds" bun:"equipment_type_ids,type:JSONB,nullzero"`

	EffectiveStartDate *int64 `json:"effectiveStartDate" bun:"effective_start_date,type:BIGINT,nullzero"`
	EffectiveEndDate   *int64 `json:"effectiveEndDate"   bun:"effective_end_date,type:BIGINT,nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-"               bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"               bun:"rel:belongs-to,join:organization_id=id"`
	Rules        []*CapabilityRule    `json:"rules,omitempty" bun:"rel:has-many,join:id=mode_profile_id"`
}

func (p *Profile) GetID() pulid.ID { return p.ID }

func (p *Profile) GetCreatedAt() int64 { return p.CreatedAt }

func (p *Profile) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Profile) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Profile) GetTableName() string { return "mode_profiles" }

func (p *Profile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "mpf",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (p *Profile) ComputeSpecificity() int32 {
	var score int32

	if p.CustomerID != nil && !p.CustomerID.IsNil() {
		score += SpecificityWeightCustomer
	}
	if len(p.EquipmentTypeIDs) > 0 {
		score += SpecificityWeightEquipmentType
	}
	if len(p.ServiceTypeIDs) > 0 {
		score += SpecificityWeightServiceType
	}
	if len(p.ShipmentTypeIDs) > 0 {
		score += SpecificityWeightShipmentType
	}

	return score
}

func (p *Profile) IsEffectiveAt(timestamp int64) bool {
	if p.EffectiveStartDate != nil && timestamp < *p.EffectiveStartDate {
		return false
	}
	if p.EffectiveEndDate != nil && timestamp > *p.EffectiveEndDate {
		return false
	}
	return true
}

func (p *Profile) HasCapability(capability Capability) bool {
	return slices.Contains(p.Capabilities, capability)
}

func (p *Profile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ProfileStatus]("Status is invalid"),
		),
		validation.Field(&p.ServiceModel,
			validation.Required.Error("Service model is required"),
			domainvalidation.ValidEnum[ServiceModel]("Service model is invalid"),
		),
		validation.Field(&p.EquipmentClass,
			validation.Required.Error("Equipment class is required"),
			domainvalidation.ValidEnum[EquipmentClass]("Equipment class is invalid"),
		),
		validation.Field(&p.ExecutionParty,
			validation.Required.Error("Execution party is required"),
			domainvalidation.ValidEnum[ExecutionParty]("Execution party is invalid"),
		),
	))

	p.validateCapabilities(multiErr)
	p.validateEffectiveDates(multiErr)
	p.validateScope(multiErr)
	p.validateRules(multiErr)
}

func (p *Profile) validateCapabilities(multiErr *errortypes.MultiError) {
	seen := make(map[Capability]struct{}, len(p.Capabilities))

	for idx, capability := range p.Capabilities {
		if !capability.IsValid() {
			multiErr.WithIndex("capabilities", idx).Add(
				"", errortypes.ErrInvalid, "Capability is not recognized")
			continue
		}
		if _, duplicate := seen[capability]; duplicate {
			multiErr.WithIndex("capabilities", idx).Add(
				"", errortypes.ErrInvalid, "Capability is listed more than once")
			continue
		}
		seen[capability] = struct{}{}
	}

	if _, ok := seen[CapabilityCore]; !ok {
		multiErr.Add("capabilities", errortypes.ErrInvalid,
			"Every profile must include the Core capability")
	}
}

func (p *Profile) validateEffectiveDates(multiErr *errortypes.MultiError) {
	if p.EffectiveStartDate == nil || p.EffectiveEndDate == nil {
		return
	}

	if *p.EffectiveEndDate <= *p.EffectiveStartDate {
		multiErr.Add("effectiveEndDate", errortypes.ErrInvalid,
			"Effective end date must be after the effective start date")
	}
}

func (p *Profile) validateScope(multiErr *errortypes.MultiError) {
	if !p.IsOrgDefault {
		return
	}

	hasScope := (p.CustomerID != nil && !p.CustomerID.IsNil()) ||
		len(p.ShipmentTypeIDs) > 0 ||
		len(p.ServiceTypeIDs) > 0 ||
		len(p.EquipmentTypeIDs) > 0

	if hasScope {
		multiErr.Add("isOrgDefault", errortypes.ErrInvalid,
			"The organization default profile cannot target a customer or type")
	}
}

func (p *Profile) validateRules(multiErr *errortypes.MultiError) {
	seen := make(map[RuleKey]struct{}, len(p.Rules))

	for idx, rule := range p.Rules {
		if rule == nil {
			continue
		}

		ruleErr := multiErr.WithIndex("rules", idx)
		rule.Validate(ruleErr)

		if _, duplicate := seen[rule.RuleKey]; duplicate {
			ruleErr.Add("ruleKey", errortypes.ErrInvalid,
				"This rule is configured more than once on the profile")
			continue
		}
		seen[rule.RuleKey] = struct{}{}

		if !IsKnownRuleKey(rule.RuleKey) {
			continue
		}

		def, defErr := RuleDefinitionFor(rule.RuleKey)
		if defErr != nil {
			continue
		}

		if !slices.Contains(p.Capabilities, def.Capability) {
			ruleErr.Add("ruleKey", errortypes.ErrInvalid,
				"This rule requires the "+def.Capability.Label()+
					" capability, which the profile does not declare")
		}
	}
}

func (p *Profile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if p.ExecutionParty == "" {
		p.ExecutionParty = ExecutionPartyCompanyAsset
	}
	if p.Status == "" {
		p.Status = ProfileStatusDraft
	}
	if !slices.Contains(p.Capabilities, CapabilityCore) {
		p.Capabilities = append([]Capability{CapabilityCore}, p.Capabilities...)
	}
	if p.IsOrgDefault {
		p.CustomerID = nil
		p.ShipmentTypeIDs = nil
		p.ServiceTypeIDs = nil
		p.EquipmentTypeIDs = nil
	}

	p.SpecificityScore = p.ComputeSpecificity()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("mpf_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
