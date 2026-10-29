package agentmemory

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*MemoryPreference)(nil)
)

// MemoryPreference is one person's choice about the memories agents pick up
// in their conversations. A row exists only once the person has chosen;
// until then they save automatically, which is how agents have always saved.
type MemoryPreference struct {
	keel.Table `keel:"agent_memory_preferences"`

	ID             pulid.ID         `json:"id"             keel:"id,pk,autoid:ampf_"`
	OrganizationID pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID         `json:"userId"         keel:"user_id"`
	SavingMode     MemorySavingMode `json:"savingMode"     keel:"saving_mode"`
	Version        int64            `json:"version"        keel:"version,version"`
	CreatedAt      int64            `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64            `json:"updatedAt"      keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (p *MemoryPreference) BeforeCreate(_ context.Context) error {
	if p.SavingMode == "" {
		p.SavingMode = MemorySavingAutomatic
	}

	return nil
}

func (p *MemoryPreference) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&p.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&p.UserID, validation.Required.Error("User is required")),
		validation.Field(&p.SavingMode,
			validation.Required.Error("Saving mode is required"),
			domainvalidation.ValidEnum[MemorySavingMode]("Saving mode is invalid"),
		),
	))
}

// AsksFirst reports whether the person wants to be asked before an agent
// keeps anything. A person with no preference is not asked.
func (p *MemoryPreference) AsksFirst() bool {
	return p != nil && p.SavingMode == MemorySavingAskFirst
}

func (p *MemoryPreference) GetID() pulid.ID { return p.ID }

func (p *MemoryPreference) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *MemoryPreference) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *MemoryPreference) GetTableName() string { return "agent_memory_preferences" }
