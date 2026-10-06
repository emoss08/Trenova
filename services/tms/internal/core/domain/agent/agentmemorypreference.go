package agent

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
	_ bun.BeforeAppendModelHook          = (*MemoryPreference)(nil)
	_ validationframework.TenantedEntity = (*MemoryPreference)(nil)
)

// MemoryPreference is one person's choice about the memories agents pick up
// in their conversations. A row exists only once the person has chosen;
// until then they save automatically, which is how agents have always saved.
type MemoryPreference struct {
	bun.BaseModel `bun:"table:agent_memory_preferences,alias:amp" json:"-"`

	ID             pulid.ID         `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	UserID         pulid.ID         `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	SavingMode     MemorySavingMode `json:"savingMode"     bun:"saving_mode,type:VARCHAR(20),notnull,default:'Automatic'"`
	Version        int64            `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64            `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64            `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (p *MemoryPreference) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("ampf_")
		}
		if p.SavingMode == "" {
			p.SavingMode = MemorySavingAutomatic
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
