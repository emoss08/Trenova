package modeprofile

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	SpecificityWeightCustomer      int32 = 16
	SpecificityWeightEquipmentType int32 = 8
	SpecificityWeightServiceType   int32 = 4
	SpecificityWeightShipmentType  int32 = 2
)

var (
	_ validationframework.TenantedEntity = (*Profile)(nil)
	_ domaintypes.PostgresSearchable     = (*Profile)(nil)
)

type Profile struct {
	keel.Table     `keel:"mode_profiles"`
	ID             pulid.ID      `keel:"id,pk,autoid:mpf_"          json:"id"`
	BusinessUnitID pulid.ID      `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID pulid.ID      `keel:"organization_id,pk,tenant"  json:"organizationId"`
	Name           string        `keel:"name"                       json:"name"`
	Code           string        `keel:"code"                       json:"code"`
	Description    string        `keel:"description,nullzero"       json:"description"`
	Status         ProfileStatus `keel:"status"                     json:"status"`

	ServiceModel   ServiceModel   `json:"serviceModel"   keel:"service_model"`
	EquipmentClass EquipmentClass `json:"equipmentClass" keel:"equipment_class"`
	ExecutionParty ExecutionParty `json:"executionParty" keel:"execution_party"`
	Capabilities   []Capability   `json:"capabilities"   keel:"capabilities,jsonb,nullzero"`

	IsOrgDefault     bool       `json:"isOrgDefault"     keel:"is_org_default"`
	Priority         int16      `json:"priority"         keel:"priority"`
	SpecificityScore int32      `json:"specificityScore" keel:"specificity_score"`
	CustomerID       *pulid.ID  `json:"customerId"       keel:"customer_id"`
	ShipmentTypeIDs  []pulid.ID `json:"shipmentTypeIds"  keel:"shipment_type_ids,jsonb,nullzero"`
	ServiceTypeIDs   []pulid.ID `json:"serviceTypeIds"   keel:"service_type_ids,jsonb,nullzero"`
	EquipmentTypeIDs []pulid.ID `json:"equipmentTypeIds" keel:"equipment_type_ids,jsonb,nullzero"`

	EffectiveStartDate *int64 `json:"effectiveStartDate" keel:"effective_start_date"`
	EffectiveEndDate   *int64 `json:"effectiveEndDate"   keel:"effective_end_date"`

	Version      int64  `json:"version"   keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-"         keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	Rules        []*CapabilityRule    `json:"rules,omitempty" keel:"many:mode_profile_id"`
}

func (p *Profile) BeforeUpdate(_ context.Context) error {
	p.applyDefaults()

	return nil
}

func (p *Profile) BeforeCreate(_ context.Context) error {
	p.applyDefaults()
	if p.Status == "" {
		p.Status = ProfileStatusDraft
	}
	if p.ExecutionParty == "" {
		p.ExecutionParty = ExecutionPartyCompanyAsset
	}

	return nil
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
			ruleErr.Add(
				"ruleKey",
				errortypes.ErrInvalid,
				"This rule requires the {0} capability, which the profile does not declare",
				def.Capability.Label(),
			)
		}
	}
}

func (p *Profile) applyDefaults() {
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
}

func (p *Profile) BeforeSave(_ context.Context) error {
	p.applyDefaults()
	return nil
}
