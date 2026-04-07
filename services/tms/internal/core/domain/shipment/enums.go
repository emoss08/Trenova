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

type BillingTransferStatus string

const (
	BillingTransferNone         = BillingTransferStatus("")
	BillingTransferReadyForReview = BillingTransferStatus("ReadyForReview")
	BillingTransferInReview     = BillingTransferStatus("InReview")
	BillingTransferOnHold       = BillingTransferStatus("OnHold")
	BillingTransferException    = BillingTransferStatus("Exception")
	BillingTransferSentBackToOps = BillingTransferStatus("SentBackToOps")
	BillingTransferApproved     = BillingTransferStatus("Approved")
	BillingTransferCanceled     = BillingTransferStatus("Canceled")
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
