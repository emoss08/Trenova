package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func FunctionalGroupDefault(transactionSet TransactionSet) string {
	switch transactionSet {
	case TransactionSet210:
		return "IM"
	case TransactionSet204:
		return "SM"
	case TransactionSet214:
		return "QM"
	case TransactionSet990:
		return "GF"
	case TransactionSet997, TransactionSet999:
		return "FA"
	default:
		return "SM"
	}
}

type EDIControlNumberSequence struct {
	keel.Table `keel:"edi_control_number_sequences"`

	ID             pulid.ID          `json:"id"             keel:"id,pk,autoid:edicn_"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	EDIPartnerID   pulid.ID          `json:"ediPartnerId"   keel:"edi_partner_id"`
	DocumentTypeID pulid.ID          `json:"documentTypeId" keel:"document_type_id"`
	Kind           ControlNumberKind `json:"kind"           keel:"kind"`
	NextValue      int64             `json:"nextValue"      keel:"next_value"`
	MinValue       int64             `json:"minValue"       keel:"min_value"`
	MaxValue       int64             `json:"maxValue"       keel:"max_value"`
	Version        int64             `json:"version"        keel:"version,version"`
	CreatedAt      int64             `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64             `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (s *EDIControlNumberSequence) BeforeUpdate(ctx context.Context) error {
	if err := s.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (s *EDIControlNumberSequence) BeforeCreate(ctx context.Context) error {
	if err := s.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (s *EDIControlNumberSequence) BeforeSave(_ context.Context) error {
	if s.MinValue == 0 {
		s.MinValue = 1
	}
	if s.NextValue == 0 {
		s.NextValue = 1
	}
	if s.MaxValue == 0 {
		s.MaxValue = 999999999
	}
	return nil
}

func (c *EDIControlNumberSequence) GetID() pulid.ID {
	return c.ID
}

func (c *EDIControlNumberSequence) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *EDIControlNumberSequence) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

// Validate checks the acknowledgment expectations, which is where an unmet
// service level turns into an alert rather than a silence.
func (s *EDIControlNumberSequence) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.EDIPartnerID, validation.Required.Error("Partner is required")),
		validation.Field(&s.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
		validation.Field(&s.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[ControlNumberKind]("Kind is invalid"),
		),
		// Control numbers roll over between the bounds, so an inverted or empty
		// range would either never issue a number or reissue one a partner has
		// already seen.
		validation.Field(&s.MinValue,
			validation.Min(int64(1)).Error("Minimum control number must be at least one"),
		),
		validation.Field(&s.MaxValue,
			validation.Min(s.MinValue+1).Error("Maximum control number must exceed the minimum"),
		),
		validation.Field(&s.NextValue,
			validation.Min(s.MinValue).Error("Next control number cannot precede the minimum"),
			validation.Max(s.MaxValue).Error("Next control number cannot exceed the maximum"),
		),
	))
}
