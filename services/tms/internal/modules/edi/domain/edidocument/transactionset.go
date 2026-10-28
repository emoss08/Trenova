package edidocument

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type EDITransactionSet struct {
	keel.Table `keel:"edi_transaction_sets"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:edits_"`
	Standard       EDIStandard    `json:"standard"       keel:"standard"`
	Code           TransactionSet `json:"code"           keel:"code"`
	Name           string         `json:"name"           keel:"name"`
	Description    string         `json:"description"    keel:"description,nullzero"`
	DefaultVersion string         `json:"defaultVersion" keel:"default_version"`
	Status         DocumentStatus `json:"status"         keel:"status"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (s *EDITransactionSet) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&s.Code,
			validation.Required.Error("Code is required"),
			domainvalidation.ValidEnum[TransactionSet]("Code is invalid"),
		),
		validation.Field(&s.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&s.DefaultVersion,
			validation.Required.Error("Default version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Default version cannot be longer than 20 characters"),
		),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DocumentStatus]("Status is invalid"),
		),
	))
}
