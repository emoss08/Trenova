package customertracking

type Field string

const (
	FieldStatus          = Field("Status")
	FieldLocationNames   = Field("LocationNames")
	FieldWindows         = Field("Windows")
	FieldActuals         = Field("Actuals")
	FieldEta             = Field("Eta")
	FieldReferences      = Field("References")
	FieldCarrier         = Field("Carrier")
	FieldEquipment       = Field("Equipment")
	FieldFreight         = Field("Freight")
	FieldLastLocation    = Field("LastLocation")
	FieldHolds           = Field("Holds")
	FieldNotes           = Field("Notes")
	FieldProofOfDelivery = Field("ProofOfDelivery")
)

func FieldValues() []Field {
	return []Field{
		FieldStatus,
		FieldLocationNames,
		FieldWindows,
		FieldActuals,
		FieldEta,
		FieldReferences,
		FieldCarrier,
		FieldEquipment,
		FieldFreight,
		FieldLastLocation,
		FieldHolds,
		FieldNotes,
		FieldProofOfDelivery,
	}
}

func DefaultFields() []Field {
	return []Field{FieldStatus, FieldWindows, FieldActuals, FieldEta, FieldReferences}
}

func (f Field) IsValid() bool {
	switch f {
	case FieldStatus,
		FieldLocationNames,
		FieldWindows,
		FieldActuals,
		FieldEta,
		FieldReferences,
		FieldCarrier,
		FieldEquipment,
		FieldFreight,
		FieldLastLocation,
		FieldHolds,
		FieldNotes,
		FieldProofOfDelivery:
		return true
	default:
		return false
	}
}

type LinkPurpose string

const (
	LinkPurposeShared       = LinkPurpose("Shared")
	LinkPurposeNotification = LinkPurpose("Notification")
)

func (p LinkPurpose) IsValid() bool {
	return p == LinkPurposeShared || p == LinkPurposeNotification
}

type Event string

const (
	EventPickedUp  = Event("PickedUp")
	EventDelayed   = Event("Delayed")
	EventDelivered = Event("Delivered")
	EventPodReady  = Event("PodReady")
)

func EventValues() []Event {
	return []Event{EventPickedUp, EventDelayed, EventDelivered, EventPodReady}
}

func (e Event) IsValid() bool {
	switch e {
	case EventPickedUp, EventDelayed, EventDelivered, EventPodReady:
		return true
	default:
		return false
	}
}

type NotificationStatus string

const (
	NotificationStatusPending = NotificationStatus("Pending")
	NotificationStatusSent    = NotificationStatus("Sent")
	NotificationStatusSkipped = NotificationStatus("Skipped")
	NotificationStatusFailed  = NotificationStatus("Failed")
)

func (s NotificationStatus) IsValid() bool {
	switch s {
	case NotificationStatusPending,
		NotificationStatusSent,
		NotificationStatusSkipped,
		NotificationStatusFailed:
		return true
	default:
		return false
	}
}

func (s NotificationStatus) IsFinal() bool {
	return s != NotificationStatusPending
}

type SkipReason string

const (
	SkipReasonNoRecipients = SkipReason("NoRecipients")
	SkipReasonRuleOff      = SkipReason("RuleOff")
	SkipReasonShipmentGone = SkipReason("ShipmentGone")
)

const (
	DefaultLinkTTLDays        = int16(30)
	MinLinkTTLDays            = int16(1)
	MaxLinkTTLDays            = int16(180)
	MaxRecipients             = 20
	MaxRecipientsLength       = 2000
	MaxRecipientEmailLength   = 320
	MaxDeliveryAttempts       = 8
	MaxLastErrorLength        = 1000
	NotificationLookbackHours = int64(48)
	SecondsPerDay             = int64(24 * 3600)
)
