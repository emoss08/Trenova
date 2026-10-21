package extractioneval

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CasePages)(nil)

type CasePages struct {
	bun.BaseModel `bun:"table:extraction_eval_case_pages,alias:eecp" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	CaseID           pulid.ID   `json:"caseId"           bun:"case_id,type:VARCHAR(100),notnull"`
	Pages            []Page     `json:"pages"            bun:"pages,type:JSONB,notnull"`
	PageCount        int        `json:"pageCount"        bun:"page_count,type:INTEGER,notnull,default:0"`
	InputHash        string     `json:"inputHash"        bun:"input_hash,type:VARCHAR(64),notnull"`
	ConverterVersion string     `json:"converterVersion" bun:"converter_version,type:VARCHAR(40),nullzero"`
	FrozenByID       pulid.ID   `json:"frozenById"       bun:"frozen_by_id,type:VARCHAR(100),notnull"`
	FrozenAt         int64      `json:"frozenAt"         bun:"frozen_at,type:BIGINT,notnull"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (p *CasePages) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("eecp_")
		}
		if p.CreatedAt == 0 {
			p.CreatedAt = now
		}
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
