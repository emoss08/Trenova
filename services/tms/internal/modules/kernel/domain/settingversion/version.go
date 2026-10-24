// Package settingversion keeps each save of an organization's AI settings
// that people edit together (the organization-wide agent controls, a model
// provider), so a save that lost a race can say who saved in between, when
// and what they changed.
package settingversion

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

const IDPrefix = "aisv_"

type Kind string

const (
	KindAgentControl = Kind("AgentControl")
	KindAIProvider   = Kind("AIProvider")
)

func Kinds() []Kind {
	return []Kind{KindAgentControl, KindAIProvider}
}

func (k Kind) IsValid() bool {
	return slices.Contains(Kinds(), k)
}

// SettingVersion is one setting as it stood after a save. The snapshot is the
// setting's own JSON, never a secret: a provider's key is not part of it.
type SettingVersion struct {
	keel.Table `keel:"ai_setting_versions"`

	ID             pulid.ID       `json:"id" keel:"id,pk,autoid:aisv_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	Kind           Kind           `json:"kind" keel:"kind"`
	SubjectID      pulid.ID       `json:"subjectId" keel:"subject_id"`
	Version        int64          `json:"version" keel:"version,version"`
	Snapshot       map[string]any `json:"snapshot" keel:"snapshot,jsonb"`
	AuthorID       *pulid.ID      `json:"authorId" keel:"author_id"`
	CreatedAt      int64          `json:"createdAt" keel:"created_at,autocreate"`

	Author *tenant.User `json:"author,omitempty" keel:"belongs:author_id"`
}
