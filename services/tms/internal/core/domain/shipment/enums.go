package shipment

type Status string

const (
	StatusNew                = Status("New")
	StatusPartiallyAssigned  = Status("PartiallyAssigned")
	StatusAssigned           = Status("Assigned")
	StatusInTransit          = Status("InTransit")
	StatusDelayed            = Status("Delayed")
	StatusPartiallyCompleted = Status("PartiallyCompleted")
	StatusReadyToInvoice     = Status("ReadyToInvoice")
	StatusCompleted          = Status("Completed")
	StatusInvoiced           = Status("Invoiced")
	StatusCanceled           = Status("Canceled")
)

type TenderStatus string

const (
	TenderStatusTendered = TenderStatus("Tendered")
	TenderStatusAccepted = TenderStatus("Accepted")
	TenderStatusRejected = TenderStatus("Rejected")
	TenderStatusExpired  = TenderStatus("Expired")
	TenderStatusCanceled = TenderStatus("Canceled")
)

type EntryMethod string

const (
	EntryMethodManual = EntryMethod("Manual")
	EntryMethodEDI    = EntryMethod("EDI")
)

type BillingTransferStatus string

const (
	BillingTransferNone           = BillingTransferStatus("")
	BillingTransferReadyForReview = BillingTransferStatus("ReadyForReview")
	BillingTransferInReview       = BillingTransferStatus("InReview")
	BillingTransferOnHold         = BillingTransferStatus("OnHold")
	BillingTransferException      = BillingTransferStatus("Exception")
	BillingTransferSentBackToOps  = BillingTransferStatus("SentBackToOps")
	BillingTransferApproved       = BillingTransferStatus("Approved")
	BillingTransferCanceled       = BillingTransferStatus("Canceled")
)

type RatingMethod string

const (
	RatingMethodFlatRate        = RatingMethod("FlatRate")
	RatingMethodPerMile         = RatingMethod("PerMile")
	RatingMethodPerStop         = RatingMethod("PerStop")
	RatingMethodPerPallet       = RatingMethod("PerPallet")
	RatingMethodPerLinearFoot   = RatingMethod("PerLinearFeet")
	RatingMethodOther           = RatingMethod("Other")
	RatingMethodFormulaTemplate = RatingMethod("FormulaTemplate")
)

type MoveStatus string

const (
	MoveStatusNew       = MoveStatus("New")
	MoveStatusAssigned  = MoveStatus("Assigned")
	MoveStatusInTransit = MoveStatus("InTransit")
	MoveStatusCompleted = MoveStatus("Completed")
	MoveStatusCanceled  = MoveStatus("Canceled")
)

type AssignmentAck string

const (
	AssignmentAckPending  = AssignmentAck("Pending")
	AssignmentAckAccepted = AssignmentAck("Accepted")
	AssignmentAckDeclined = AssignmentAck("Declined")
)

func (a AssignmentAck) String() string { return string(a) }

func (a AssignmentAck) IsValid() bool {
	switch a {
	case AssignmentAckPending, AssignmentAckAccepted, AssignmentAckDeclined:
		return true
	default:
		return false
	}
}

type AssignmentStatus string

const (
	AssignmentStatusNew        = AssignmentStatus("New")
	AssignmentStatusInProgress = AssignmentStatus("InProgress")
	AssignmentStatusCompleted  = AssignmentStatus("Completed")
	AssignmentStatusCanceled   = AssignmentStatus("Canceled")
)

type StopStatus string

const (
	StopStatusNew       = StopStatus("New")
	StopStatusInTransit = StopStatus("InTransit")
	StopStatusCompleted = StopStatus("Completed")
	StopStatusCanceled  = StopStatus("Canceled")
)

type StopType string

const (
	StopTypePickup        = StopType("Pickup")
	StopTypeDelivery      = StopType("Delivery")
	StopTypeSplitDelivery = StopType("SplitDelivery")
	StopTypeSplitPickup   = StopType("SplitPickup")
)

type StopScheduleType string

const (
	StopScheduleTypeOpen        = StopScheduleType("Open")
	StopScheduleTypeAppointment = StopScheduleType("Appointment")
)

type CommentType string

const (
	CommentTypeInternal            = CommentType("Internal")
	CommentTypeDispatch            = CommentType("Dispatch")
	CommentTypeDriverUpdate        = CommentType("DriverUpdate")
	CommentTypePickupInstruction   = CommentType("PickupInstruction")
	CommentTypeDeliveryInstruction = CommentType("DeliveryInstruction")
	CommentTypeStatusUpdate        = CommentType("StatusUpdate")
	CommentTypeException           = CommentType("Exception")
	CommentTypeCustomerUpdate      = CommentType("CustomerUpdate")
	CommentTypeAppointment         = CommentType("Appointment")
	CommentTypeDocument            = CommentType("Document")
	CommentTypeBilling             = CommentType("Billing")
	CommentTypeCompliance          = CommentType("Compliance")
)

type CommentVisibility string

const (
	CommentVisibilityInternal   = CommentVisibility("Internal")
	CommentVisibilityOperations = CommentVisibility("Operations")
	CommentVisibilityCustomer   = CommentVisibility("Customer")
	CommentVisibilityDriver     = CommentVisibility("Driver")
	CommentVisibilityAccounting = CommentVisibility("Accounting")
)

type CommentPriority string

const (
	CommentPriorityLow    = CommentPriority("Low")
	CommentPriorityNormal = CommentPriority("Normal")
	CommentPriorityHigh   = CommentPriority("High")
	CommentPriorityUrgent = CommentPriority("Urgent")
)

type CommentSource string

const (
	CommentSourceUser        = CommentSource("User")
	CommentSourceSystem      = CommentSource("System")
	CommentSourceIntegration = CommentSource("Integration")
	CommentSourceAI          = CommentSource("AI")
)

func (v Status) IsValid() bool {
	switch v {
	case StatusNew,
		StatusPartiallyAssigned,
		StatusAssigned,
		StatusInTransit,
		StatusDelayed,
		StatusPartiallyCompleted,
		StatusReadyToInvoice,
		StatusCompleted,
		StatusInvoiced,
		StatusCanceled:
		return true
	default:
		return false
	}
}

func (v TenderStatus) IsValid() bool {
	switch v {
	case TenderStatusTendered,
		TenderStatusAccepted,
		TenderStatusRejected,
		TenderStatusExpired,
		TenderStatusCanceled:
		return true
	default:
		return false
	}
}

func (v EntryMethod) IsValid() bool {
	switch v {
	case EntryMethodManual,
		EntryMethodEDI:
		return true
	default:
		return false
	}
}

func (v BillingTransferStatus) IsValid() bool {
	switch v {
	case BillingTransferNone,
		BillingTransferReadyForReview,
		BillingTransferInReview,
		BillingTransferOnHold,
		BillingTransferException,
		BillingTransferSentBackToOps,
		BillingTransferApproved,
		BillingTransferCanceled:
		return true
	default:
		return false
	}
}

func (v RatingMethod) IsValid() bool {
	switch v {
	case RatingMethodFlatRate,
		RatingMethodPerMile,
		RatingMethodPerStop,
		RatingMethodPerPallet,
		RatingMethodPerLinearFoot,
		RatingMethodOther,
		RatingMethodFormulaTemplate:
		return true
	default:
		return false
	}
}

func (v MoveStatus) IsValid() bool {
	switch v {
	case MoveStatusNew,
		MoveStatusAssigned,
		MoveStatusInTransit,
		MoveStatusCompleted,
		MoveStatusCanceled:
		return true
	default:
		return false
	}
}

func (v AssignmentStatus) IsValid() bool {
	switch v {
	case AssignmentStatusNew,
		AssignmentStatusInProgress,
		AssignmentStatusCompleted,
		AssignmentStatusCanceled:
		return true
	default:
		return false
	}
}

func (v StopStatus) IsValid() bool {
	switch v {
	case StopStatusNew,
		StopStatusInTransit,
		StopStatusCompleted,
		StopStatusCanceled:
		return true
	default:
		return false
	}
}

func (v StopType) IsValid() bool {
	switch v {
	case StopTypePickup,
		StopTypeDelivery,
		StopTypeSplitDelivery,
		StopTypeSplitPickup:
		return true
	default:
		return false
	}
}

func (v StopScheduleType) IsValid() bool {
	switch v {
	case StopScheduleTypeOpen,
		StopScheduleTypeAppointment:
		return true
	default:
		return false
	}
}

func (v CommentType) IsValid() bool {
	switch v {
	case CommentTypeInternal,
		CommentTypeDispatch,
		CommentTypeDriverUpdate,
		CommentTypePickupInstruction,
		CommentTypeDeliveryInstruction,
		CommentTypeStatusUpdate,
		CommentTypeException,
		CommentTypeCustomerUpdate,
		CommentTypeAppointment,
		CommentTypeDocument,
		CommentTypeBilling,
		CommentTypeCompliance:
		return true
	default:
		return false
	}
}

func (v CommentVisibility) IsValid() bool {
	switch v {
	case CommentVisibilityInternal,
		CommentVisibilityOperations,
		CommentVisibilityCustomer,
		CommentVisibilityDriver,
		CommentVisibilityAccounting:
		return true
	default:
		return false
	}
}

func (v CommentPriority) IsValid() bool {
	switch v {
	case CommentPriorityLow,
		CommentPriorityNormal,
		CommentPriorityHigh,
		CommentPriorityUrgent:
		return true
	default:
		return false
	}
}

func (v CommentSource) IsValid() bool {
	switch v {
	case CommentSourceUser,
		CommentSourceSystem,
		CommentSourceIntegration,
		CommentSourceAI:
		return true
	default:
		return false
	}
}
