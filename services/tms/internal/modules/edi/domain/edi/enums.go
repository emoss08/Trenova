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
	MappingEntityTypeCustomer                 = MappingEntityType("Customer")
	MappingEntityTypeServiceType              = MappingEntityType("ServiceType")
	MappingEntityTypeShipmentType             = MappingEntityType("ShipmentType")
	MappingEntityTypeFormulaTemplate          = MappingEntityType("FormulaTemplate")
	MappingEntityTypeLocation                 = MappingEntityType("Location")
	MappingEntityTypeCommodity                = MappingEntityType("Commodity")
	MappingEntityTypeAccessorialCharge        = MappingEntityType("AccessorialCharge")
	MappingEntityTypeServiceFailureReasonCode = MappingEntityType("ServiceFailureReasonCode")
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
	MessageDeliveryStatusQueued       = MessageDeliveryStatus("Queued")
	MessageDeliveryStatusSending      = MessageDeliveryStatus("Sending")
	MessageDeliveryStatusSent         = MessageDeliveryStatus("Sent")
	MessageDeliveryStatusFailed       = MessageDeliveryStatus("Failed")
	MessageDeliveryStatusDeadLettered = MessageDeliveryStatus("DeadLettered")
)

func (s MessageDeliveryStatus) IsRetryable() bool {
	return s == MessageDeliveryStatusFailed || s == MessageDeliveryStatusDeadLettered
}

func (s MessageDeliveryStatus) IsDeliverable() bool {
	return s == MessageDeliveryStatusQueued ||
		s == MessageDeliveryStatusSending ||
		s == MessageDeliveryStatusFailed
}

type InboundFileStatus string

const (
	InboundFileStatusReceived           = InboundFileStatus("Received")
	InboundFileStatusParsed             = InboundFileStatus("Parsed")
	InboundFileStatusProcessed          = InboundFileStatus("Processed")
	InboundFileStatusPartiallyProcessed = InboundFileStatus("PartiallyProcessed")
	InboundFileStatusQuarantined        = InboundFileStatus("Quarantined")
	InboundFileStatusDuplicate          = InboundFileStatus("Duplicate")
)

func (s InboundFileStatus) IsReprocessable() bool {
	return s == InboundFileStatusQuarantined || s == InboundFileStatusPartiallyProcessed
}

type MessageAcknowledgmentStatus string

const (
	MessageAcknowledgmentStatusNotExpected = MessageAcknowledgmentStatus("NotExpected")
	MessageAcknowledgmentStatusPending     = MessageAcknowledgmentStatus("Pending")
	MessageAcknowledgmentStatusAccepted    = MessageAcknowledgmentStatus("Accepted")
	MessageAcknowledgmentStatusRejected    = MessageAcknowledgmentStatus("Rejected")
	MessageAcknowledgmentStatusFailed      = MessageAcknowledgmentStatus("Failed")
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

func (k PartnerKind) IsValid() bool {
	switch k {
	case PartnerKindInternal,
		PartnerKindExternal:
		return true
	default:
		return false
	}
}

func (r PartnerRole) IsValid() bool {
	switch r {
	case PartnerRoleCustomer,
		PartnerRoleCarrier,
		PartnerRoleBroker,
		PartnerRoleVendor,
		PartnerRoleShipper,
		PartnerRoleConsignee,
		PartnerRoleBillTo:
		return true
	default:
		return false
	}
}

func (m ConnectionMethod) IsValid() bool {
	switch m {
	case ConnectionMethodInternal,
		ConnectionMethodAS2,
		ConnectionMethodSFTP,
		ConnectionMethodVAN:
		return true
	default:
		return false
	}
}

func (s ConnectionStatus) IsValid() bool {
	switch s {
	case ConnectionStatusPendingAcceptance,
		ConnectionStatusActive,
		ConnectionStatusSuspended,
		ConnectionStatusRejected,
		ConnectionStatusRevoked:
		return true
	default:
		return false
	}
}

func (t MappingEntityType) IsValid() bool {
	switch t {
	case MappingEntityTypeCustomer,
		MappingEntityTypeServiceType,
		MappingEntityTypeShipmentType,
		MappingEntityTypeFormulaTemplate,
		MappingEntityTypeLocation,
		MappingEntityTypeCommodity,
		MappingEntityTypeAccessorialCharge,
		MappingEntityTypeServiceFailureReasonCode:
		return true
	default:
		return false
	}
}

func (s TransferStatus) IsValid() bool {
	switch s {
	case TransferStatusSubmitted,
		TransferStatusMappingRequired,
		TransferStatusPendingApproval,
		TransferStatusProcessing,
		TransferStatusApproved,
		TransferStatusRejected,
		TransferStatusExpired,
		TransferStatusCanceled,
		TransferStatusFailed:
		return true
	default:
		return false
	}
}

func (d DocumentDirection) IsValid() bool {
	switch d {
	case DocumentDirectionInbound,
		DocumentDirectionOutbound:
		return true
	default:
		return false
	}
}

func (s EDIStandard) IsValid() bool {
	switch s {
	case EDIStandardX12:
		return true
	default:
		return false
	}
}

func (t TransactionSet) IsValid() bool {
	switch t {
	case TransactionSet204,
		TransactionSet210,
		TransactionSet214,
		TransactionSet990,
		TransactionSet997,
		TransactionSet999:
		return true
	default:
		return false
	}
}

func (s DocumentStatus) IsValid() bool {
	switch s {
	case DocumentStatusActive,
		DocumentStatusInactive:
		return true
	default:
		return false
	}
}

func (s TemplateStatus) IsValid() bool {
	switch s {
	case TemplateStatusDraft,
		TemplateStatusCertified,
		TemplateStatusActive,
		TemplateStatusDeprecated,
		TemplateStatusArchived,
		TemplateStatusSuperseded:
		return true
	default:
		return false
	}
}

func (l ScriptLanguage) IsValid() bool {
	switch l {
	case ScriptLanguageStarlark:
		return true
	default:
		return false
	}
}

func (s MessageStatus) IsValid() bool {
	switch s {
	case MessageStatusGenerated,
		MessageStatusFailed:
		return true
	default:
		return false
	}
}

func (s MessageDeliveryStatus) IsValid() bool {
	switch s {
	case MessageDeliveryStatusQueued,
		MessageDeliveryStatusSending,
		MessageDeliveryStatusSent,
		MessageDeliveryStatusFailed,
		MessageDeliveryStatusDeadLettered:
		return true
	default:
		return false
	}
}

func (s InboundFileStatus) IsValid() bool {
	switch s {
	case InboundFileStatusReceived,
		InboundFileStatusParsed,
		InboundFileStatusProcessed,
		InboundFileStatusPartiallyProcessed,
		InboundFileStatusQuarantined,
		InboundFileStatusDuplicate:
		return true
	default:
		return false
	}
}

func (s MessageAcknowledgmentStatus) IsValid() bool {
	switch s {
	case MessageAcknowledgmentStatusNotExpected,
		MessageAcknowledgmentStatusPending,
		MessageAcknowledgmentStatusAccepted,
		MessageAcknowledgmentStatusRejected,
		MessageAcknowledgmentStatusFailed:
		return true
	default:
		return false
	}
}

func (m ValidationMode) IsValid() bool {
	switch m {
	case ValidationModeStrict,
		ValidationModeWarnOnly,
		ValidationModeDisabled:
		return true
	default:
		return false
	}
}

func (s ValidationSeverity) IsValid() bool {
	switch s {
	case ValidationSeverityInfo,
		ValidationSeverityWarning,
		ValidationSeverityError:
		return true
	default:
		return false
	}
}

func (k ControlNumberKind) IsValid() bool {
	switch k {
	case ControlNumberKindInterchange,
		ControlNumberKindGroup,
		ControlNumberKindTransaction:
		return true
	default:
		return false
	}
}

func (t AcknowledgmentType) IsValid() bool {
	switch t {
	case AcknowledgmentTypeNone,
		AcknowledgmentType997,
		AcknowledgmentType999:
		return true
	default:
		return false
	}
}

func (c LoadTenderPurposeCode) IsValid() bool {
	switch c {
	case LoadTenderPurposeOriginal,
		LoadTenderPurposeChange:
		return true
	default:
		return false
	}
}

func (k TenderRecipientKind) IsValid() bool {
	switch k {
	case TenderRecipientKindInternal,
		TenderRecipientKindExternal:
		return true
	default:
		return false
	}
}

func (s TenderRecipientStatus) IsValid() bool {
	switch s {
	case TenderRecipientStatusActive,
		TenderRecipientStatusClosed:
		return true
	default:
		return false
	}
}

func (s TenderRecipientBaselineStatus) IsValid() bool {
	switch s {
	case TenderRecipientBaselineStatusSent,
		TenderRecipientBaselineStatusAccepted:
		return true
	default:
		return false
	}
}

func (s TenderChangeStatus) IsValid() bool {
	switch s {
	case TenderChangeStatusPendingReview,
		TenderChangeStatusApplied,
		TenderChangeStatusRejected,
		TenderChangeStatusQueued,
		TenderChangeStatusSent,
		TenderChangeStatusFailed,
		TenderChangeStatusIgnored,
		TenderChangeStatusSuperseded:
		return true
	default:
		return false
	}
}

func (t SourceContextDataType) IsValid() bool {
	switch t {
	case SourceContextDataTypeString,
		SourceContextDataTypeNumber,
		SourceContextDataTypeInteger,
		SourceContextDataTypeBoolean,
		SourceContextDataTypeTimestamp,
		SourceContextDataTypeDate,
		SourceContextDataTypeDecimal,
		SourceContextDataTypeObject,
		SourceContextDataTypeArray,
		SourceContextDataTypeUnknown:
		return true
	default:
		return false
	}
}

func (k SourceContextKind) IsValid() bool {
	switch k {
	case SourceContextKindShipment,
		SourceContextKindRepeat,
		SourceContextKindPartner,
		SourceContextKindRuntime,
		SourceContextKindMapping,
		SourceContextKindOrganization,
		SourceContextKindCustomer,
		SourceContextKindLocation,
		SourceContextKindCommodity,
		SourceContextKindCharge,
		SourceContextKindEnvelope:
		return true
	default:
		return false
	}
}

func (s SourceContextFieldStatus) IsValid() bool {
	switch s {
	case SourceContextFieldStatusActive,
		SourceContextFieldStatusDeprecated,
		SourceContextFieldStatusFuture:
		return true
	default:
		return false
	}
}

func (t PartnerSettingDataType) IsValid() bool {
	switch t {
	case PartnerSettingDataTypeString,
		PartnerSettingDataTypeNumber,
		PartnerSettingDataTypeInteger,
		PartnerSettingDataTypeBoolean,
		PartnerSettingDataTypeDecimal,
		PartnerSettingDataTypeEnum,
		PartnerSettingDataTypeObject,
		PartnerSettingDataTypeArray,
		PartnerSettingDataTypeMap,
		PartnerSettingDataTypeSecret,
		PartnerSettingDataTypeUnknown:
		return true
	default:
		return false
	}
}

func (s PartnerSettingStatus) IsValid() bool {
	switch s {
	case PartnerSettingStatusActive,
		PartnerSettingStatusDeprecated,
		PartnerSettingStatusFuture:
		return true
	default:
		return false
	}
}
