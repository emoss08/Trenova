package ediservice

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/maputils"
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
			).Error("Method must be Internal, AS2, SFTP, or VAN"),
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
			).Error("Method must be Internal, AS2, SFTP, or VAN"),
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

func (v *Validator) ValidatePartnerDocumentProfileRequest(
	req *UpsertEDIPartnerDocumentProfileRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("profile", errortypes.ErrRequired, "Profile is required")
		return multiErr
	}
	if req.EDIPartnerID.IsNil() {
		multiErr.Add("ediPartnerId", errortypes.ErrRequired, "EDI partner is required")
	}
	validateDocumentStatus(multiErr, req.Status)
	validateValidationMode(multiErr, req.ValidationMode)
	validateAcknowledgmentType(multiErr, req.Acknowledgment.Type)
	validateEnvelope(multiErr, &req.Envelope)

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (v *Validator) ValidatePreviewDocumentRequest(
	req *PreviewEDIDocumentRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("document", errortypes.ErrRequired, "Document request is required")
		return multiErr
	}
	if req.PartnerDocumentProfileID.IsNil() && req.EDIPartnerID.IsNil() {
		multiErr.Add(
			"ediPartnerId",
			errortypes.ErrRequired,
			"EDI partner or document profile is required",
		)
	}
	hasPayloadSource := req.Payload != nil ||
		req.TransferID.IsNotNil() ||
		req.ShipmentID.IsNotNil() ||
		req.InvoiceID.IsNotNil() ||
		req.ShipmentEventID.IsNotNil() ||
		req.ServiceFailureID.IsNotNil() ||
		req.SourceMessageID.IsNotNil()
	if !hasPayloadSource {
		multiErr.Add(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment, transfer, invoice, source message, or payload is required",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func validateDocumentStatus(multiErr *errortypes.MultiError, status edi.DocumentStatus) {
	if status == "" {
		return
	}
	switch status {
	case edi.DocumentStatusActive, edi.DocumentStatusInactive:
	default:
		multiErr.Add("status", errortypes.ErrInvalid, "Document profile status is invalid")
	}
}

func validateValidationMode(multiErr *errortypes.MultiError, mode edi.ValidationMode) {
	if mode == "" {
		return
	}
	switch mode {
	case edi.ValidationModeStrict, edi.ValidationModeWarnOnly, edi.ValidationModeDisabled:
	default:
		multiErr.Add("validationMode", errortypes.ErrInvalid, "Validation mode is invalid")
	}
}

func validateAcknowledgmentType(multiErr *errortypes.MultiError, ackType edi.AcknowledgmentType) {
	if ackType == "" {
		return
	}
	switch ackType {
	case edi.AcknowledgmentTypeNone, edi.AcknowledgmentType997, edi.AcknowledgmentType999:
	default:
		multiErr.Add("acknowledgment.type", errortypes.ErrInvalid, "Acknowledgment type is invalid")
	}
}

func validateEnvelope(multiErr *errortypes.MultiError, envelope *edi.X12EnvelopeSettings) {
	requireX12ID(
		multiErr,
		"envelope.interchangeSenderId",
		envelope.InterchangeSenderID,
		"ISA sender ID is required",
	)
	requireX12ID(
		multiErr,
		"envelope.interchangeReceiverId",
		envelope.InterchangeReceiverID,
		"ISA receiver ID is required",
	)
	validateISAQualifier(
		multiErr,
		"envelope.interchangeSenderQualifier",
		envelope.InterchangeSenderQualifier,
	)
	validateISAQualifier(
		multiErr,
		"envelope.interchangeReceiverQualifier",
		envelope.InterchangeReceiverQualifier,
	)
	requireSeparator(multiErr, "envelope.elementSeparator", envelope.ElementSeparator)
	requireSeparator(multiErr, "envelope.segmentTerminator", envelope.SegmentTerminator)
	requireSeparator(multiErr, "envelope.componentSeparator", envelope.ComponentSeparator)
	requireSeparator(multiErr, "envelope.repetitionSeparator", envelope.RepetitionSeparator)

	if envelope.InterchangeUsageIndicator == "" {
		return
	}
	switch envelope.InterchangeUsageIndicator {
	case "P", "T":
	default:
		multiErr.Add(
			"envelope.interchangeUsageIndicator",
			errortypes.ErrInvalid,
			"Usage indicator must be P or T",
		)
	}
}

func requireX12ID(multiErr *errortypes.MultiError, field, value, message string) {
	value = strings.TrimSpace(value)
	if value == "" {
		multiErr.Add(field, errortypes.ErrRequired, message)
		return
	}
	if len(value) > 15 {
		multiErr.Add(field, errortypes.ErrInvalid, "X12 envelope ID must be 15 characters or fewer")
	}
}

func requireSeparator(multiErr *errortypes.MultiError, field, value string) {
	if value == "" {
		multiErr.Add(field, errortypes.ErrRequired, "Separator is required")
		return
	}
	if len(value) != 1 {
		multiErr.Add(field, errortypes.ErrInvalid, "Separator must be exactly one character")
	}
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
		v.validateAS2ProfileConfig(entity, multiErr)
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
		requireConfigString(multiErr, entity.Config, "host", "Host is required")
		requireConfigString(multiErr, entity.Config, "port", "Port is required")
		requireConfigString(multiErr, entity.Config, "username", "Username is required")
		requireConfigString(multiErr, entity.Config, "authMode", "Authentication mode is required")
		requireConfigString(multiErr, entity.Config, "knownHostKey", "Known host key is required")
		requireAnySecret(
			multiErr,
			entity.EncryptedSecrets,
			[]string{"password", "privateKey"},
			"VAN password or private key secret is required",
		)
	}
	if entity.Method != edi.ConnectionMethodInternal {
		requireConfigString(
			multiErr,
			entity.Config,
			"isaSenderQualifier",
			"ISA sender qualifier is required",
		)
		requireConfigString(multiErr, entity.Config, "isaSenderId", "ISA sender ID is required")
		requireConfigString(
			multiErr,
			entity.Config,
			"isaReceiverQualifier",
			"ISA receiver qualifier is required",
		)
		requireConfigString(multiErr, entity.Config, "isaReceiverId", "ISA receiver ID is required")
		requireConfigString(multiErr, entity.Config, "gsSenderId", "GS sender ID is required")
		requireConfigString(multiErr, entity.Config, "gsReceiverId", "GS receiver ID is required")
		requireConfigString(multiErr, entity.Config, "x12Version", "X12 version is required")
		requireConfigString(multiErr, entity.Config, "environment", "Environment is required")
		validateDeliveryRetryConfig(multiErr, entity.Config)
	}
}

func validateDeliveryRetryConfig(multiErr *errortypes.MultiError, config map[string]any) {
	validateRetryConfigInt(
		multiErr,
		config,
		editransport.ConfigKeyRetryMaxAttempts,
		editransport.MinDeliveryMaxAttempts,
		editransport.MaxDeliveryMaxAttempts,
		fmt.Sprintf(
			"Retry max attempts must be a whole number between %d and %d",
			editransport.MinDeliveryMaxAttempts,
			editransport.MaxDeliveryMaxAttempts,
		),
	)
	initial, initialSet := validateRetryConfigInt(
		multiErr,
		config,
		editransport.ConfigKeyRetryInitialIntervalSeconds,
		editransport.MinDeliveryIntervalSeconds,
		editransport.MaxDeliveryIntervalSeconds,
		fmt.Sprintf(
			"Retry initial interval must be between %d and %d seconds",
			editransport.MinDeliveryIntervalSeconds,
			editransport.MaxDeliveryIntervalSeconds,
		),
	)
	maximum, maximumSet := validateRetryConfigInt(
		multiErr,
		config,
		editransport.ConfigKeyRetryMaxIntervalSeconds,
		editransport.MinDeliveryIntervalSeconds,
		editransport.MaxDeliveryIntervalSeconds,
		fmt.Sprintf(
			"Retry max interval must be between %d and %d seconds",
			editransport.MinDeliveryIntervalSeconds,
			editransport.MaxDeliveryIntervalSeconds,
		),
	)
	if initialSet && maximumSet && maximum < initial {
		multiErr.Add(
			"config."+editransport.ConfigKeyRetryMaxIntervalSeconds,
			errortypes.ErrInvalid,
			"Retry max interval must be greater than or equal to the initial interval",
		)
	}
}

func validateRetryConfigInt(
	multiErr *errortypes.MultiError,
	config map[string]any,
	key string,
	minValue, maxValue int64,
	message string,
) (int64, bool) {
	if maputils.StringValue(config, key) == "" {
		return 0, false
	}
	value, ok := maputils.IntValue(config, key)
	if !ok || value < minValue || value > maxValue {
		multiErr.Add("config."+key, errortypes.ErrInvalid, message)
		return 0, false
	}
	return value, true
}

func validateISAQualifier(multiErr *errortypes.MultiError, field, value string) {
	if value == "" {
		return
	}
	if len(value) != 2 {
		multiErr.Add(field, errortypes.ErrInvalid, "ISA qualifier must be exactly 2 characters")
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

func (v *Validator) validateAS2ProfileConfig(
	entity *edi.EDICommunicationProfile,
	multiErr *errortypes.MultiError,
) {
	mdnMode := strings.ToLower(maputils.StringValue(entity.Config, editransport.ConfigKeyMDNMode))
	if mdnMode != "" && mdnMode != editransport.MDNModeSync &&
		mdnMode != editransport.MDNModeAsync {
		multiErr.Add("config.mdnMode", errortypes.ErrInvalid, "MDN mode must be sync or async")
	}
	if mdnMode == editransport.MDNModeAsync {
		requireConfigString(
			multiErr,
			entity.Config,
			editransport.ConfigKeyMDNURL,
			"Async MDN return URL is required when MDN mode is async",
		)
	}
	if endpointURL := maputils.StringValue(
		entity.Config,
		editransport.ConfigKeyEndpointURL,
	); endpointURL != "" {
		if parsed, err := url.Parse(endpointURL); err != nil || parsed.Scheme == "" ||
			parsed.Host == "" {
			multiErr.Add(
				"config.endpointUrl",
				errortypes.ErrInvalid,
				"Endpoint URL must be a valid absolute URL",
			)
		}
	}
	validateAS2Algorithm(
		multiErr,
		entity.Config,
		editransport.ConfigKeySigningAlgorithm,
		[]string{
			as2.SigningAlgorithmSHA1,
			as2.SigningAlgorithmSHA256,
			as2.SigningAlgorithmSHA384,
			as2.SigningAlgorithmSHA512,
		},
		"Signing algorithm must be sha1, sha256, sha384, or sha512",
	)
	validateAS2Algorithm(
		multiErr,
		entity.Config,
		editransport.ConfigKeyEncryptionAlgorithm,
		[]string{
			as2.EncryptionAlgorithmTripleDES,
			as2.EncryptionAlgorithmAES128CBC,
			as2.EncryptionAlgorithmAES256CBC,
			as2.EncryptionAlgorithmAES128GCM,
			as2.EncryptionAlgorithmAES256GCM,
		},
		"Encryption algorithm must be 3des, aes128-cbc, aes256-cbc, aes128-gcm, or aes256-gcm",
	)
	validateAS2Algorithm(
		multiErr,
		entity.Config,
		editransport.ConfigKeyCompressionAlgorithm,
		[]string{"none", editransport.CompressionZlib},
		"Compression algorithm must be none or zlib",
	)
	validateAS2Algorithm(
		multiErr,
		entity.Config,
		editransport.ConfigKeyRequireSignedInbound,
		[]string{editransport.AS2InboundRequirementAuto, "true", "false"},
		"Require signed inbound must be auto, true, or false",
	)
	validateAS2Algorithm(
		multiErr,
		entity.Config,
		editransport.ConfigKeyRequireEncryptedInbound,
		[]string{editransport.AS2InboundRequirementAuto, "true", "false"},
		"Require encrypted inbound must be auto, true, or false",
	)
	validateAS2Certificate(
		multiErr,
		entity.Config,
		editransport.ConfigKeyLocalCertificate,
		"config.localCertificate",
	)
	validateAS2Certificate(
		multiErr,
		entity.Config,
		editransport.ConfigKeyPartnerSigningCertificate,
		"config.partnerSigningCertificate",
	)
	validateAS2Certificate(
		multiErr,
		entity.Config,
		editransport.ConfigKeyPartnerEncryptionCertificate,
		"config.partnerEncryptionCertificate",
	)
	if maputils.StringValue(entity.Config, editransport.ConfigKeyLocalCertificate) != "" {
		requireAnySecret(
			multiErr,
			entity.EncryptedSecrets,
			[]string{editransport.SecretKeyAS2PrivateKey},
			"AS2 private key secret is required when a local certificate is configured",
		)
	}
}

func validateAS2Algorithm(
	multiErr *errortypes.MultiError,
	config map[string]any,
	key string,
	allowed []string,
	message string,
) {
	value := strings.ToLower(maputils.StringValue(config, key))
	if value == "" {
		return
	}
	if !slices.Contains(allowed, value) {
		multiErr.Add("config."+key, errortypes.ErrInvalid, message)
	}
}

func validateAS2Certificate(
	multiErr *errortypes.MultiError,
	config map[string]any,
	key, field string,
) {
	pemData := maputils.StringValue(config, key)
	if pemData == "" {
		return
	}
	if _, err := as2.ParseCertificate([]byte(pemData)); err != nil {
		multiErr.Add(field, errortypes.ErrInvalid, "Certificate must be a valid PEM certificate")
	}
}
