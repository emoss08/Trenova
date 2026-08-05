package report

import (
	"context"
	"fmt"

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
	_ bun.BeforeAppendModelHook          = (*ReportView)(nil)
	_ validationframework.TenantedEntity = (*ReportView)(nil)
)

const MaxViewNameLength = 120

// ReportView is a named set of parameter values for a report. A parameterized
// report is one question asked many ways — this quarter for the West region,
// last quarter for the East — and without views every asking means retyping
// every parameter. The view stores only the answers, never the report itself,
// so a definition edit reaches every view of it at once.
type ReportView struct {
	bun.BaseModel `bun:"table:report_views,alias:rview" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DefinitionID   pulid.ID       `json:"definitionId"   bun:"definition_id,type:VARCHAR(100),notnull"`
	OwnerID        pulid.ID       `json:"ownerId"        bun:"owner_id,type:VARCHAR(100),notnull"`
	Name           string         `json:"name"           bun:"name,type:VARCHAR(120),notnull"`
	Description    string         `json:"description"    bun:"description,type:TEXT,nullzero"`
	Params         map[string]any `json:"params"         bun:"params,type:JSONB,nullzero"`
	// Shared publishes the view to everyone who can read the report. A private
	// view stays with its owner, which is what keeps one person's working set
	// out of everybody else's list.
	Shared bool `json:"shared" bun:"shared,type:BOOLEAN,notnull,default:false"`
	// Pinned is per-owner ordering intent: a pinned view sorts to the front of
	// the picker so the one someone opens daily is never buried.
	Pinned    bool   `json:"pinned"    bun:"pinned,type:BOOLEAN,notnull,default:false"`
	Format    Format `json:"format"    bun:"format,type:VARCHAR(10),nullzero"`
	LastRunAt int64  `json:"lastRunAt" bun:"last_run_at,type:BIGINT,nullzero"`
	RunCount  int64  `json:"runCount"  bun:"run_count,type:BIGINT,notnull,default:0"`
	Version   int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64  `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64  `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization     *tenant.Organization `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	Owner            *tenant.User         `json:"owner,omitempty"            bun:"rel:belongs-to,join:owner_id=id"`
	ReportDefinition *ReportDefinition    `json:"reportDefinition,omitempty" bun:"rel:belongs-to,join:definition_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (rv *ReportView) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(rv,
		validation.Field(&rv.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&rv.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&rv.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, MaxViewNameLength).Error(fmt.Sprintf(
				"Name must be %d characters or fewer", MaxViewNameLength,
			)),
		),
		validation.Field(&rv.DefinitionID, validation.Required.Error("Report is required")),
		validation.Field(&rv.OwnerID, validation.Required.Error("Owner is required")),
		// An empty format means "whatever the report defaults to", which is why
		// it is nullable; a populated one still has to name a real renderer.
		validation.Field(&rv.Format, domainvalidation.ValidEnum[Format]("Format is invalid")),
		validation.Field(&rv.RunCount,
			validation.Min(int64(0)).Error("Run count cannot be negative"),
		),
	))
}

func (rv *ReportView) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rv.ID.IsNil() {
			rv.ID = pulid.MustNew("rview_")
		}
		rv.CreatedAt = now
	case *bun.UpdateQuery:
		rv.UpdatedAt = now
	}

	return nil
}

func (rv *ReportView) GetID() pulid.ID { return rv.ID }

func (rv *ReportView) GetOrganizationID() pulid.ID { return rv.OrganizationID }

func (rv *ReportView) GetBusinessUnitID() pulid.ID { return rv.BusinessUnitID }

func (rv *ReportView) GetTableName() string { return "report_views" }
