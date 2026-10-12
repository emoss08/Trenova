package formulatemplate

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*TestCase)(nil)

// TestCase is a saved scenario for a formula template: named input values with
// the charge the author expects them to produce. Scenarios re-run on demand in
// the studio and gate approval, so a template's behaviour is pinned by examples
// a billing clerk can read, not just by an expression a reviewer must parse.
type TestCase struct {
	bun.BaseModel `bun:"table:formula_template_test_cases,alias:ftc" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	TemplateID     pulid.ID        `json:"templateId"     bun:"template_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Name           string          `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string          `json:"description"    bun:"description,type:TEXT"`
	Variables      map[string]any  `json:"variables"      bun:"variables,type:JSONB,notnull,default:'{}'"`
	ExpectedAmount decimal.Decimal `json:"expectedAmount" bun:"expected_amount,type:NUMERIC(19,4),notnull"`
	Tolerance      decimal.Decimal `json:"tolerance"      bun:"tolerance,type:NUMERIC(19,4),notnull,default:0.01"`
	Version        int64           `json:"version"        bun:"version,type:BIGINT"`
	CreatedByID    pulid.ID        `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

const maxTestCaseNameLength = 100

func (tc *TestCase) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(tc,
		validation.Field(&tc.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&tc.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxTestCaseNameLength).
				Error("Name cannot be longer than 100 characters"),
		),
	))

	if tc.ExpectedAmount.IsNegative() {
		multiErr.Add(
			"expectedAmount",
			errortypes.ErrInvalid,
			"Expected amount cannot be negative",
		)
	}

	if tc.Tolerance.IsNegative() {
		multiErr.Add("tolerance", errortypes.ErrInvalid, "Tolerance cannot be negative")
	}
}

func (tc *TestCase) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if tc.ID.IsNil() {
			tc.ID = pulid.MustNew("ftc_")
		}
		tc.CreatedAt = now
		tc.UpdatedAt = now
	case *bun.UpdateQuery:
		tc.UpdatedAt = now
	}

	return nil
}

func (tc *TestCase) GetID() pulid.ID {
	return tc.ID
}

func (tc *TestCase) GetOrganizationID() pulid.ID {
	return tc.OrganizationID
}

func (tc *TestCase) GetBusinessUnitID() pulid.ID {
	return tc.BusinessUnitID
}

func (tc *TestCase) GetTableName() string {
	return "formula_template_test_cases"
}
