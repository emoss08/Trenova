// Package settingversion keeps each save of an organization's AI settings
// that people edit together (the organization-wide agent controls, a model
// provider), so a save that lost a race can say who saved in between, when
// and what they changed.
package settingversion

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:ai_setting_versions,alias:aisv" json:"-"`
	keel.Table    `keel:"ai_setting_versions"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	Kind           Kind           `json:"kind"           bun:"kind,type:VARCHAR(30),notnull" keel:"kind"`
	SubjectID      pulid.ID       `json:"subjectId"      bun:"subject_id,type:VARCHAR(100),notnull" keel:"subject_id"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT,notnull" keel:"version,version"`
	Snapshot       map[string]any `json:"snapshot"       bun:"snapshot,type:JSONB,notnull" keel:"snapshot,jsonb"`
	AuthorID       *pulid.ID      `json:"authorId"       bun:"author_id,type:VARCHAR(100),nullzero" keel:"author_id"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	Author *tenant.User `json:"author,omitempty" bun:"rel:belongs-to,join:author_id=id" keel:"belongs:author_id"`
}

func (v *SettingVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if v.ID.IsNil() {
			v.ID = pulid.MustNew(IDPrefix)
		}
		v.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
