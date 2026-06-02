package edi

type PartnerKind string

const (
	PartnerKindInternal = PartnerKind("Internal")
	PartnerKindExternal = PartnerKind("External")
)

type PartnerRole string

const (
	PartnerRoleCustomer  = PartnerRole("Customer")
	PartnerRoleCarrier   = PartnerRole("Carrier")
	PartnerRoleBroker    = PartnerRole("Broker")
	PartnerRoleVendor    = PartnerRole("Vendor")
	PartnerRoleShipper   = PartnerRole("Shipper")
	PartnerRoleConsignee = PartnerRole("Consignee")
	PartnerRoleBillTo    = PartnerRole("BillTo")
)

type ConnectionMethod string

const (
	ConnectionMethodInternal = ConnectionMethod("Internal")
	ConnectionMethodAS2      = ConnectionMethod("AS2")
	ConnectionMethodSFTP     = ConnectionMethod("SFTP")
	ConnectionMethodVAN      = ConnectionMethod("VAN")
)

type ConnectionStatus string

const (
	ConnectionStatusPendingAcceptance = ConnectionStatus("PendingAcceptance")
	ConnectionStatusActive            = ConnectionStatus("Active")
	ConnectionStatusSuspended         = ConnectionStatus("Suspended")
	ConnectionStatusRejected          = ConnectionStatus("Rejected")
	ConnectionStatusRevoked           = ConnectionStatus("Revoked")
)

type MappingEntityType string

const (
	MappingEntityTypeCustomer          = MappingEntityType("Customer")
	MappingEntityTypeServiceType       = MappingEntityType("ServiceType")
	MappingEntityTypeShipmentType      = MappingEntityType("ShipmentType")
	MappingEntityTypeFormulaTemplate   = MappingEntityType("FormulaTemplate")
	MappingEntityTypeLocation          = MappingEntityType("Location")
	MappingEntityTypeCommodity         = MappingEntityType("Commodity")
	MappingEntityTypeAccessorialCharge = MappingEntityType("AccessorialCharge")
)

type TransferStatus string

const (
	TransferStatusSubmitted       = TransferStatus("Submitted")
	TransferStatusMappingRequired = TransferStatus("MappingRequired")
	TransferStatusPendingApproval = TransferStatus("PendingApproval")
	TransferStatusProcessing      = TransferStatus("Processing")
	TransferStatusApproved        = TransferStatus("Approved")
	TransferStatusRejected        = TransferStatus("Rejected")
	TransferStatusExpired         = TransferStatus("Expired")
	TransferStatusCanceled        = TransferStatus("Canceled")
	TransferStatusFailed          = TransferStatus("Failed")
)

func (s TransferStatus) IsFinal() bool {
	switch s {
	case TransferStatusSubmitted,
		TransferStatusMappingRequired,
		TransferStatusPendingApproval,
		TransferStatusProcessing:
		return false
	case TransferStatusApproved,
		TransferStatusRejected,
		TransferStatusExpired,
		TransferStatusCanceled,
		TransferStatusFailed:
		return true
	default:
		return false
	}
}

func (s TransferStatus) IsActionable() bool {
	return !s.IsFinal() && s != TransferStatusProcessing
}

type DocumentDirection string

const (
	DocumentDirectionInbound  = DocumentDirection("Inbound")
	DocumentDirectionOutbound = DocumentDirection("Outbound")
)

type EDIStandard string

const (
	EDIStandardX12 = EDIStandard("X12")
)

type TransactionSet string

const (
	TransactionSet204 = TransactionSet("204")
	TransactionSet210 = TransactionSet("210")
	TransactionSet214 = TransactionSet("214")
	TransactionSet990 = TransactionSet("990")
	TransactionSet997 = TransactionSet("997")
	TransactionSet999 = TransactionSet("999")
)

type DocumentStatus string

const (
	DocumentStatusActive   = DocumentStatus("Active")
	DocumentStatusInactive = DocumentStatus("Inactive")
)

type TemplateStatus string

const (
	TemplateStatusDraft      = TemplateStatus("Draft")
	TemplateStatusCertified  = TemplateStatus("Certified")
	TemplateStatusActive     = TemplateStatus("Active")
	TemplateStatusDeprecated = TemplateStatus("Deprecated")
	TemplateStatusArchived   = TemplateStatus("Archived")
	TemplateStatusSuperseded = TemplateStatus("Superseded")
)

type ScriptLanguage string

const (
	ScriptLanguageStarlark = ScriptLanguage("Starlark")
)

type MessageStatus string

const (
	MessageStatusGenerated = MessageStatus("Generated")
	MessageStatusFailed    = MessageStatus("Failed")
)

type MessageDeliveryStatus string

const (
	MessageDeliveryStatusQueued  = MessageDeliveryStatus("Queued")
	MessageDeliveryStatusSending = MessageDeliveryStatus("Sending")
	MessageDeliveryStatusSent    = MessageDeliveryStatus("Sent")
	MessageDeliveryStatusFailed  = MessageDeliveryStatus("Failed")
)

type ValidationMode string

const (
	ValidationModeStrict   = ValidationMode("Strict")
	ValidationModeWarnOnly = ValidationMode("WarnOnly")
	ValidationModeDisabled = ValidationMode("Disabled")
)

type ValidationSeverity string

const (
	ValidationSeverityInfo    = ValidationSeverity("Info")
	ValidationSeverityWarning = ValidationSeverity("Warning")
	ValidationSeverityError   = ValidationSeverity("Error")
)

type ControlNumberKind string

const (
	ControlNumberKindInterchange = ControlNumberKind("Interchange")
	ControlNumberKindGroup       = ControlNumberKind("Group")
	ControlNumberKindTransaction = ControlNumberKind("Transaction")
)

type AcknowledgmentType string

const (
	AcknowledgmentTypeNone = AcknowledgmentType("None")
	AcknowledgmentType997  = AcknowledgmentType("997")
	AcknowledgmentType999  = AcknowledgmentType("999")
)

type LoadTenderPurposeCode string

const (
	LoadTenderPurposeOriginal = LoadTenderPurposeCode("00")
	LoadTenderPurposeChange   = LoadTenderPurposeCode("04")
)

type TenderRecipientKind string

const (
	TenderRecipientKindInternal = TenderRecipientKind("Internal")
	TenderRecipientKindExternal = TenderRecipientKind("External")
)

type TenderRecipientStatus string

const (
	TenderRecipientStatusActive = TenderRecipientStatus("Active")
	TenderRecipientStatusClosed = TenderRecipientStatus("Closed")
)

type TenderRecipientBaselineStatus string

const (
	TenderRecipientBaselineStatusSent     = TenderRecipientBaselineStatus("Sent")
	TenderRecipientBaselineStatusAccepted = TenderRecipientBaselineStatus("Accepted")
)

type TenderChangeStatus string

const (
	TenderChangeStatusPendingReview = TenderChangeStatus("PendingReview")
	TenderChangeStatusApplied       = TenderChangeStatus("Applied")
	TenderChangeStatusRejected      = TenderChangeStatus("Rejected")
	TenderChangeStatusQueued        = TenderChangeStatus("Queued")
	TenderChangeStatusSent          = TenderChangeStatus("Sent")
	TenderChangeStatusFailed        = TenderChangeStatus("Failed")
	TenderChangeStatusIgnored       = TenderChangeStatus("Ignored")
	TenderChangeStatusSuperseded    = TenderChangeStatus("Superseded")
)

const TenderChangeTypeLoadTender = "LoadTenderChange"

type SourceContextDataType string

const (
	SourceContextDataTypeString    = SourceContextDataType("string")
	SourceContextDataTypeNumber    = SourceContextDataType("number")
	SourceContextDataTypeInteger   = SourceContextDataType("integer")
	SourceContextDataTypeBoolean   = SourceContextDataType("boolean")
	SourceContextDataTypeTimestamp = SourceContextDataType("timestamp")
	SourceContextDataTypeDate      = SourceContextDataType("date")
	SourceContextDataTypeDecimal   = SourceContextDataType("decimal")
	SourceContextDataTypeObject    = SourceContextDataType("object")
	SourceContextDataTypeArray     = SourceContextDataType("array")
	SourceContextDataTypeUnknown   = SourceContextDataType("unknown")
)

type SourceContextKind string

const (
	SourceContextKindShipment     = SourceContextKind("shipment")
	SourceContextKindRepeat       = SourceContextKind("repeat")
	SourceContextKindPartner      = SourceContextKind("partner")
	SourceContextKindRuntime      = SourceContextKind("runtime")
	SourceContextKindMapping      = SourceContextKind("mapping")
	SourceContextKindOrganization = SourceContextKind("organization")
	SourceContextKindCustomer     = SourceContextKind("customer")
	SourceContextKindLocation     = SourceContextKind("location")
	SourceContextKindCommodity    = SourceContextKind("commodity")
	SourceContextKindCharge       = SourceContextKind("charge")
	SourceContextKindEnvelope     = SourceContextKind("envelope")
)

type SourceContextFieldStatus string

const (
	SourceContextFieldStatusActive     = SourceContextFieldStatus("Active")
	SourceContextFieldStatusDeprecated = SourceContextFieldStatus("Deprecated")
	SourceContextFieldStatusFuture     = SourceContextFieldStatus("Future")
)

type PartnerSettingDataType string

const (
	PartnerSettingDataTypeString  = PartnerSettingDataType("string")
	PartnerSettingDataTypeNumber  = PartnerSettingDataType("number")
	PartnerSettingDataTypeInteger = PartnerSettingDataType("integer")
	PartnerSettingDataTypeBoolean = PartnerSettingDataType("boolean")
	PartnerSettingDataTypeDecimal = PartnerSettingDataType("decimal")
	PartnerSettingDataTypeEnum    = PartnerSettingDataType("enum")
	PartnerSettingDataTypeObject  = PartnerSettingDataType("object")
	PartnerSettingDataTypeArray   = PartnerSettingDataType("array")
	PartnerSettingDataTypeMap     = PartnerSettingDataType("map")
	PartnerSettingDataTypeSecret  = PartnerSettingDataType("secret")
	PartnerSettingDataTypeUnknown = PartnerSettingDataType("unknown")
)

type PartnerSettingStatus string

const (
	PartnerSettingStatusActive     = PartnerSettingStatus("Active")
	PartnerSettingStatusDeprecated = PartnerSettingStatus("Deprecated")
	PartnerSettingStatusFuture     = PartnerSettingStatus("Future")
)
