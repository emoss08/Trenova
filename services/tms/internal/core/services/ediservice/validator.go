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

func (v *Validator) ValidateConnection(entity *edi.EDIConnection) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if entity == nil {
		multiErr.Add("", errortypes.ErrRequired, "EDI connection is required")
		return multiErr
	}

	err := validation.ValidateStruct(
		entity,
		validation.Field(
			&entity.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(
			&entity.SourceOrganizationID,
			validation.Required.Error("Source organization is required"),
		),
		validation.Field(
			&entity.TargetOrganizationID,
			validation.Required.Error("Target organization is required"),
		),
		validation.Field(
			&entity.Method,
			validation.Required.Error("Method is required"),
			validation.In(
				edi.ConnectionMethodInternal,
				edi.ConnectionMethodAS2,
				edi.ConnectionMethodSFTP,
				edi.ConnectionMethodVAN,
			),
		),
		validation.Field(
			&entity.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				edi.ConnectionStatusPendingAcceptance,
				edi.ConnectionStatusActive,
				edi.ConnectionStatusSuspended,
				edi.ConnectionStatusRejected,
				edi.ConnectionStatusRevoked,
			),
		),
		validation.Field(
			&entity.SourcePartnerConfig.Code,
			validation.Required.Error("Source partner code is required"),
			validation.Length(1, 100),
		),
		validation.Field(
			&entity.SourcePartnerConfig.Name,
			validation.Required.Error("Source partner name is required"),
			validation.Length(1, 200),
		),
		validation.Field(
			&entity.TargetPartnerConfig.Code,
			validation.Required.Error("Target partner code is required"),
			validation.Length(1, 100),
		),
		validation.Field(
			&entity.TargetPartnerConfig.Name,
			validation.Required.Error("Target partner name is required"),
			validation.Length(1, 200),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
	if entity.SourceOrganizationID == entity.TargetOrganizationID {
		multiErr.Add(
			"targetOrganizationId",
			errortypes.ErrInvalid,
			"Target organization must be different from the current organization",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (v *Validator) ValidateCommunicationProfile(
	entity *edi.EDICommunicationProfile,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if entity == nil {
		multiErr.Add("", errortypes.ErrRequired, "EDI communication profile is required")
		return multiErr
	}

	err := validation.ValidateStruct(
		entity,
		validation.Field(
			&entity.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(
			&entity.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(
			&entity.Method,
			validation.Required.Error("Method is required"),
			validation.In(
				edi.ConnectionMethodInternal,
				edi.ConnectionMethodAS2,
				edi.ConnectionMethodSFTP,
				edi.ConnectionMethodVAN,
			),
		),
		validation.Field(
			&entity.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 200),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
	v.validateProfileConfig(entity, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (v *Validator) validateProfileConfig(
	entity *edi.EDICommunicationProfile,
	multiErr *errortypes.MultiError,
) {
	switch entity.Method {
	case edi.ConnectionMethodInternal:
		return
	case edi.ConnectionMethodAS2:
		requireConfigString(multiErr, entity.Config, "localAS2Id", "Local AS2 ID is required")
		requireConfigString(multiErr, entity.Config, "partnerAS2Id", "Partner AS2 ID is required")
		requireConfigString(multiErr, entity.Config, "endpointUrl", "Endpoint URL is required")
		requireConfigString(multiErr, entity.Config, "mdnMode", "MDN mode is required")
	case edi.ConnectionMethodSFTP:
		requireConfigString(multiErr, entity.Config, "host", "Host is required")
		requireConfigString(multiErr, entity.Config, "port", "Port is required")
		requireConfigString(multiErr, entity.Config, "username", "Username is required")
		requireConfigString(multiErr, entity.Config, "authMode", "Authentication mode is required")
		requireConfigString(multiErr, entity.Config, "knownHostKey", "Known host key is required")
		requireAnySecret(
			multiErr,
			entity.EncryptedSecrets,
			[]string{"password", "privateKey"},
			"SFTP password or private key secret is required",
		)
	case edi.ConnectionMethodVAN:
		requireConfigString(multiErr, entity.Config, "providerName", "Provider name is required")
		requireConfigString(multiErr, entity.Config, "mailboxId", "Mailbox ID is required")
		requireAnySecret(
			multiErr,
			entity.EncryptedSecrets,
			[]string{"credential", "token"},
			"VAN credential or token secret is required",
		)
	}
	if entity.Method != edi.ConnectionMethodInternal {
		requireConfigString(multiErr, entity.Config, "isaSenderQualifier", "ISA sender qualifier is required")
		requireConfigString(multiErr, entity.Config, "isaSenderId", "ISA sender ID is required")
		requireConfigString(multiErr, entity.Config, "isaReceiverQualifier", "ISA receiver qualifier is required")
		requireConfigString(multiErr, entity.Config, "isaReceiverId", "ISA receiver ID is required")
		requireConfigString(multiErr, entity.Config, "gsSenderId", "GS sender ID is required")
		requireConfigString(multiErr, entity.Config, "gsReceiverId", "GS receiver ID is required")
		requireConfigString(multiErr, entity.Config, "x12Version", "X12 version is required")
		requireConfigString(multiErr, entity.Config, "environment", "Environment is required")
	}
}

func requireConfigString(
	multiErr *errortypes.MultiError,
	config map[string]any,
	key string,
	message string,
) {
	value, ok := config[key].(string)
	if ok && value != "" {
		return
	}
	if !ok && config[key] != nil {
		return
	}

	multiErr.Add("config."+key, errortypes.ErrRequired, message)
}

func requireAnySecret(
	multiErr *errortypes.MultiError,
	secrets map[string]string,
	keys []string,
	message string,
) {
	for _, key := range keys {
		if secrets[key] != "" {
			return
		}
	}

	multiErr.Add("secrets", errortypes.ErrRequired, message)
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
