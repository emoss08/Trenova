package tenant

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*BusinessUnit)(nil)

type BusinessUnit struct {
	bun.BaseModel `bun:"table:business_units,alias:bu" json:"-"`

	ID        pulid.ID       `json:"id"        bun:"id,pk,type:VARCHAR(100)"`
	Name      string         `json:"name"      bun:"name,type:VARCHAR(100),notnull"`
	Code      string         `json:"code"      bun:"code,type:VARCHAR(10),notnull"`
	Metadata  map[string]any `json:"-"         bun:"metadata,type:JSONB,default:'{}'::jsonb"`
	Version   int64          `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64          `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64          `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (bu *BusinessUnit) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(bu,
		validation.Field(&bu.Name,
			validation.Required.Error("Name is required. Please try again"),
			validation.Length(1, 100).
				Error("Name must be between 1 and 100 characters. Please try again"),
			validation.Match(regexp.MustCompile(`^[a-zA-Z0-9\s\-&.]+$`)).
				Error("Name can only contain letters, numbers, spaces, hyphens, ampersands, and periods"),
		),
		validation.Field(&bu.Code,
			validation.Required.Error("Code is required"),
			validation.Length(2, 10).Error("Code must be between 2 and 10 characters"),
			validation.Match(regexp.MustCompile(`^[A-Z0-9]+$`)).
				Error("Code must contain only uppercase letters and numbers"),
		),
	)

	multiErr.AddOzzoError(err)
}

func (bu *BusinessUnit) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if bu.ID.IsNil() {
			bu.ID = pulid.MustNew("bu_")
		}

		bu.CreatedAt = now
	case *bun.UpdateQuery:
		bu.UpdatedAt = now
	}

	return nil
}
