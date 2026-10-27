package report

import (
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*ReportView)(nil)
)

const MaxViewNameLength = 120

// ReportView is a named set of parameter values for a report. A parameterized
// report is one question asked many ways — this quarter for the West region,
// last quarter for the East — and without views every asking means retyping
// every parameter. The view stores only the answers, never the report itself,
// so a definition edit reaches every view of it at once.
type ReportView struct {
	keel.Table `keel:"report_views"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:rview_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	DefinitionID   pulid.ID       `json:"definitionId"   keel:"definition_id"`
	OwnerID        pulid.ID       `json:"ownerId"        keel:"owner_id"`
	Name           string         `json:"name"           keel:"name"`
	Description    string         `json:"description"    keel:"description,nullzero"`
	Params         map[string]any `json:"params"         keel:"params,jsonb,nullzero"`
	// Shared publishes the view to everyone who can read the report. A private
	// view stays with its owner, which is what keeps one person's working set
	// out of everybody else's list.
	Shared bool `json:"shared" keel:"shared"`
	// Pinned is per-owner ordering intent: a pinned view sorts to the front of
	// the picker so the one someone opens daily is never buried.
	Pinned    bool   `json:"pinned"    keel:"pinned"`
	Format    Format `json:"format"    keel:"format,nullzero"`
	LastRunAt int64  `json:"lastRunAt" keel:"last_run_at,nullzero"`
	RunCount  int64  `json:"runCount"  keel:"run_count"`
	Version   int64  `json:"version"   keel:"version,version"`
	CreatedAt int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization     *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Owner            *tenant.User         `json:"owner,omitempty"            keel:"belongs:owner_id"`
	ReportDefinition *ReportDefinition    `json:"reportDefinition,omitempty" keel:"belongs:definition_id"`
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

func (rv *ReportView) GetID() pulid.ID { return rv.ID }

func (rv *ReportView) GetOrganizationID() pulid.ID { return rv.OrganizationID }

func (rv *ReportView) GetBusinessUnitID() pulid.ID { return rv.BusinessUnitID }

func (rv *ReportView) GetTableName() string { return "report_views" }
