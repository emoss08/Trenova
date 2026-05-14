package ediservice

import (
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidatePartner(entity *edi.EDIPartner) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if entity == nil {
		multiErr.Add("", errortypes.ErrRequired, "EDI partner is required")
		return multiErr
	}

	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateMappingItems(
	items []*edi.EDIMappingProfileItem,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	for i, item := range items {
		itemErr := multiErr.WithIndex("items", i)
		if item == nil {
			itemErr.Add("", errortypes.ErrRequired, "Mapping item is required")
			continue
		}
		err := validation.ValidateStruct(
			item,
			validation.Field(
				&item.EntityType,
				validation.Required.Error("Entity type is required"),
			),
			validation.Field(&item.SourceID, validation.Required.Error("Source ID is required")),
			validation.Field(&item.TargetID, validation.Required.Error("Target ID is required")),
		)
		if err != nil {
			if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
				errortypes.FromOzzoErrors(validationErrs, itemErr)
			}
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
