package tenant

import (
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const BusinessUnitIDPrefix = "bu_"

type BusinessUnit struct {
	keel.Table `keel:"business_units"`

	ID        pulid.ID       `json:"id" keel:"id,pk,autoid:bu_"`
	Name      string         `json:"name" keel:"name"`
	Code      string         `json:"code" keel:"code"`
	Metadata  map[string]any `json:"-" keel:"metadata,jsonb"`
	Version   int64          `json:"version" keel:"version,version"`
	CreatedAt int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64          `json:"updatedAt" keel:"updated_at,autoupdate"`
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
