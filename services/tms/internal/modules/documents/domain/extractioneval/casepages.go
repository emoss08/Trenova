package extractioneval

import (
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CasePages struct {
	keel.Table `keel:"extraction_eval_case_pages"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:eecp_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	CaseID           pulid.ID `json:"caseId" keel:"case_id"`
	Pages            []Page   `json:"pages" keel:"pages,jsonb"`
	PageCount        int      `json:"pageCount" keel:"page_count"`
	InputHash        string   `json:"inputHash" keel:"input_hash"`
	ConverterVersion string   `json:"converterVersion" keel:"converter_version,nullzero"`
	FrozenByID       pulid.ID `json:"frozenById" keel:"frozen_by_id"`
	FrozenAt         int64    `json:"frozenAt" keel:"frozen_at"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func NewCasePages(
	evalCase *ExtractionCase,
	pages []Page,
	converterVersion string,
	frozenByID pulid.ID,
) *CasePages {
	return &CasePages{
		OrganizationID: evalCase.OrganizationID,
		BusinessUnitID: evalCase.BusinessUnitID,
		CaseID:         evalCase.ID,
		Pages:          pages,
		PageCount:      len(pages),
		InputHash: ComputeInputHash(
			aicorrection.TaskShipmentDraftExtraction,
			evalCase.FileName,
			pages,
		),
		ConverterVersion: converterVersion,
		FrozenByID:       frozenByID,
		FrozenAt:         timeutils.NowUnix(),
	}
}

func (p *CasePages) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		p,
		validation.Field(&p.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&p.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&p.CaseID, validation.Required.Error("Case is required")),
		validation.Field(&p.Pages,
			validation.Required.Error("The document has no readable pages"),
			validation.Length(1, MaxPages).
				Error(fmt.Sprintf("A case holds at most %d pages", MaxPages)),
		),
		validation.Field(&p.InputHash, validation.Length(InputHashLength, InputHashLength)),
		validation.Field(&p.FrozenByID, validation.Required.Error("Frozen by is required")),
	))
}
