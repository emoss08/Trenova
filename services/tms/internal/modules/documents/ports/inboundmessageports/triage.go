package inboundmessageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Lane string

const (
	LaneWaiting       = Lane("Waiting")
	LaneWaitingOnThem = Lane("WaitingOnThem")
	LaneAssignedToMe  = Lane("AssignedToMe")
	LaneSnoozed       = Lane("Snoozed")
	LaneHandled       = Lane("Handled")
	LaneIgnored       = Lane("Ignored")
	LaneEverything    = Lane("Everything")
)

func (l Lane) IsValid() bool {
	switch l {
	case LaneWaiting, LaneWaitingOnThem, LaneAssignedToMe, LaneSnoozed, LaneHandled,
		LaneIgnored, LaneEverything:
		return true
	default:
		return false
	}
}

type Sort string

const (
	SortDue    = Sort("Due")
	SortNewest = Sort("Newest")
)

func (s Sort) IsValid() bool {
	return s == SortDue || s == SortNewest
}

type ListAttachmentsByMessageIDsRequest struct {
	TenantInfo pagination.TenantInfo
	MessageIDs []pulid.ID
}

type TriageMessageNotifier interface {
	MessagesChanged(ctx context.Context, messages ...*inboundmessage.InboundMessage)
}

type TriageActivityRecorder interface {
	RecordActivity(ctx context.Context, activities ...*triage.Activity) error
}

type TriageBriefingInvalidator interface {
	InvalidateBriefing(ctx context.Context, tenantInfo pagination.TenantInfo)
}

type RuleApplication struct {
	Rule    *triage.Rule
	Settled bool
}

type TriageRuleEvaluator interface {
	ApplyRules(
		ctx context.Context,
		message *inboundmessage.InboundMessage,
	) (*RuleApplication, error)
}

type SenderHintProvider interface {
	HintsFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		senderDomain string,
	) ([]*triage.SenderHint, error)
}

type FileOnMatchedRequest struct {
	Message     *inboundmessage.InboundMessage
	ActorID     pulid.ID
	ByNova      bool
	QueueForPay bool
}

type TriageFiler interface {
	FileOnMatchedShipment(ctx context.Context, req FileOnMatchedRequest) (int, error)
}

type TriageDrafter interface {
	DraftReplyFor(ctx context.Context, message *inboundmessage.InboundMessage) error
	DraftLoadFor(ctx context.Context, message *inboundmessage.InboundMessage) error
}

type InboundReplyReader interface {
	LatestByMessageIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		messageIDs []pulid.ID,
	) (map[pulid.ID]*triage.Reply, error)
	ThreadKeyForMessageKeys(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		messageKeys []string,
	) (string, bool, error)
}

type TriageEventSink interface {
	ShipmentDelivered(ctx context.Context, tenantInfo pagination.TenantInfo, shipmentID pulid.ID)
	DocumentAttached(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
		documentTypeID pulid.ID,
	)
	InvoicePaid(ctx context.Context, tenantInfo pagination.TenantInfo, shipmentID pulid.ID)
}

type BulkResult struct {
	Messages []*inboundmessage.InboundMessage `json:"messages"`
	Affected int                              `json:"affected"`
	RuleID   *pulid.ID                        `json:"ruleId"`
}

type HandOffResult struct {
	ToUserID pulid.ID `json:"toUserId"`
	Affected int      `json:"affected"`
}

type ResponseTargetView struct {
	Classification inboundmessage.Classification `json:"classification"`
	Minutes        int                           `json:"minutes"`
	IsDefault      bool                          `json:"isDefault"`
}

type AssigneeView struct {
	UserID  pulid.ID `json:"userId"`
	Waiting int      `json:"waiting"`
}

type SnoozeView struct {
	Until           *int64                      `json:"until"`
	Event           *inboundmessage.SnoozeEvent `json:"event"`
	WatchShipmentID pulid.ID                    `json:"watchShipmentId"`
	DocumentTypeID  *pulid.ID                   `json:"documentTypeId"`
	SnoozedAt       int64                       `json:"snoozedAt"`
	SnoozedBy       pulid.ID                    `json:"snoozedBy"`
}

type AwaitingReplyView struct {
	Since int64 `json:"since"`
	Until int64 `json:"until"`
}

type Sample struct {
	MessageID      pulid.ID                       `json:"messageId"`
	FromName       string                         `json:"fromName"`
	FromAddress    string                         `json:"fromAddress"`
	Subject        string                         `json:"subject"`
	Classification *inboundmessage.Classification `json:"classification"`
	ReceivedAt     int64                          `json:"receivedAt"`
	Reference      string                         `json:"reference"`
}

type RuleMismatch string

const (
	MismatchKindCorrected   = RuleMismatch("KindCorrected")
	MismatchNoMatchedRecord = RuleMismatch("NoMatchedRecord")
	MismatchPersonDisagreed = RuleMismatch("PersonDisagreed")
)

type RuleMatch struct {
	MessageID   pulid.ID      `json:"messageId"`
	FromName    string        `json:"fromName"`
	FromAddress string        `json:"fromAddress"`
	Subject     string        `json:"subject"`
	ReceivedAt  int64         `json:"receivedAt"`
	Reference   string        `json:"reference"`
	Mismatch    *RuleMismatch `json:"mismatch"`
}

type RulePreview struct {
	Matched int          `json:"matched"`
	Off     int          `json:"off"`
	Matches []*RuleMatch `json:"matches"`
	Since   int64        `json:"since"`
}

type RuleSuggestion struct {
	Classification inboundmessage.Classification `json:"classification"`
	SenderScope    triage.SenderScope            `json:"senderScope"`
	Sender         string                        `json:"sender"`
	MailboxID      *pulid.ID                     `json:"mailboxId"`
	Action         inboundmessage.NovaAction     `json:"action"`
	HandledByHand  int                           `json:"handledByHand"`
	Total          int                           `json:"total"`
}

type DeckItem struct {
	MessageID      pulid.ID                       `json:"messageId"`
	FromName       string                         `json:"fromName"`
	FromAddress    string                         `json:"fromAddress"`
	Subject        string                         `json:"subject"`
	Classification *inboundmessage.Classification `json:"classification"`
	ReceivedAt     int64                          `json:"receivedAt"`
	DueAt          *int64                         `json:"dueAt"`
	Action         *triage.Action                 `json:"action"`
	OwnerID        pulid.ID                       `json:"ownerId"`
}

type DeckCard struct {
	Group           triage.DeckGroup `json:"group"`
	Count           int              `json:"count"`
	EstimateMinutes int              `json:"estimateMinutes"`
	Items           []*DeckItem      `json:"items"`
	MessageIDs      []pulid.ID       `json:"messageIds"`
}

type HandledGroup struct {
	Action         inboundmessage.NovaAction      `json:"action"`
	Classification *inboundmessage.Classification `json:"classification"`
	Count          int                            `json:"count"`
	Samples        []*Sample                      `json:"samples"`
}

type ArrivalBucket struct {
	HourStart     int64 `json:"hourStart"`
	HandledByNova int   `json:"handledByNova"`
	LeftForPerson int   `json:"leftForPerson"`
}

type RiverPoint struct {
	MessageID      pulid.ID                       `json:"messageId"`
	FromName       string                         `json:"fromName"`
	FromAddress    string                         `json:"fromAddress"`
	Subject        string                         `json:"subject"`
	Classification *inboundmessage.Classification `json:"classification"`
	ReceivedAt     int64                          `json:"receivedAt"`
	DueAt          *int64                         `json:"dueAt"`
	Done           bool                           `json:"done"`
}

type CorrectionsSummary struct {
	ThisWeek       int     `json:"thisWeek"`
	AccuracyChange float64 `json:"accuracyChange"`
	Accuracy       float64 `json:"accuracy"`
}

type BriefingPhase string

const (
	BriefingDay      = BriefingPhase("Day")
	BriefingEndOfDay = BriefingPhase("EndOfDay")
)

type Briefing struct {
	GeneratedAt         int64              `json:"generatedAt"`
	DayStart            int64              `json:"dayStart"`
	Phase               BriefingPhase      `json:"phase"`
	FirstDay            bool               `json:"firstDay"`
	NovaEnabled         bool               `json:"novaEnabled"`
	ActiveSources       int                `json:"activeSources"`
	Waiting             int                `json:"waiting"`
	Overdue             int                `json:"overdue"`
	ClosedByPeopleToday int                `json:"closedByPeopleToday"`
	HandledByNovaToday  int                `json:"handledByNovaToday"`
	HandledByNova       []*HandledGroup    `json:"handledByNova"`
	Arrivals            []*ArrivalBucket   `json:"arrivals"`
	NovaPoints          []*RiverPoint      `json:"novaPoints"`
	Yours               []*RiverPoint      `json:"yours"`
	Deck                []*DeckCard        `json:"deck"`
	LongestWaiting      *DeckItem          `json:"longestWaiting"`
	EstimateMinutes     int                `json:"estimateMinutes"`
	Corrections         CorrectionsSummary `json:"corrections"`
	ComingBack          int                `json:"comingBack"`
	UnhealthySourceIDs  []pulid.ID         `json:"unhealthySourceIds"`
	Suggestion          *RuleSuggestion    `json:"suggestion"`
	OnCallUserID        pulid.ID           `json:"onCallUserId"`
	CarryOver           []*DeckItem        `json:"carryOver"`
}

type SenderRole string

const (
	SenderRoleCustomer = SenderRole("Customer")
	SenderRoleBroker   = SenderRole("Broker")
	SenderRoleCarrier  = SenderRole("Carrier")
	SenderRoleDriver   = SenderRole("Driver")
	SenderRoleUnknown  = SenderRole("Unknown")
)

type SenderKindCount struct {
	Classification inboundmessage.Classification `json:"classification"`
	Count          int                           `json:"count"`
}

type SenderProfile struct {
	Address            string             `json:"address"`
	Name               string             `json:"name"`
	Domain             string             `json:"domain"`
	Role               SenderRole         `json:"role"`
	Since              *int64             `json:"since"`
	MessagesLast90Days int                `json:"messagesLast90Days"`
	UsualReplyMinutes  *int               `json:"usualReplyMinutes"`
	Kinds              []*SenderKindCount `json:"kinds"`
	Rules              []*triage.Rule     `json:"rules"`
	Recent             []*Sample          `json:"recent"`
}

type DocumentField struct {
	Label string                  `json:"label"`
	Value string                  `json:"value"`
	Tone  inboundmessage.FactTone `json:"tone"`
}

type DocumentReading struct {
	DocumentType string           `json:"documentType"`
	Confidence   float64          `json:"confidence"`
	PageCount    int              `json:"pageCount"`
	Fields       []*DocumentField `json:"fields"`
}

type StopTimes struct {
	ShipmentID             pulid.ID `json:"shipmentId"`
	StopName               string   `json:"stopName"`
	AppointmentStart       *int64   `json:"appointmentStart"`
	AppointmentEnd         *int64   `json:"appointmentEnd"`
	ArrivedAt              *int64   `json:"arrivedAt"`
	CheckedInAt            *int64   `json:"checkedInAt"`
	DepartedAt             *int64   `json:"departedAt"`
	LateMinutes            int      `json:"lateMinutes"`
	BilledDetentionMinutes int      `json:"billedDetentionMinutes"`
	BilledDetentionAmount  *string  `json:"billedDetentionAmount"`
	DetentionPaysWhenLate  bool     `json:"detentionPaysWhenLate"`
	ArrivalSource          string   `json:"arrivalSource"`
}

type PeekStop struct {
	Kind        string `json:"kind"`
	City        string `json:"city"`
	State       string `json:"state"`
	ScheduledAt *int64 `json:"scheduledAt"`
	ArrivedAt   *int64 `json:"arrivedAt"`
	DepartedAt  *int64 `json:"departedAt"`
	Done        bool   `json:"done"`
}

type ShipmentPeek struct {
	ID              pulid.ID    `json:"id"`
	ProNumber       string      `json:"proNumber"`
	BOL             string      `json:"bol"`
	PONumber        string      `json:"poNumber"`
	Status          string      `json:"status"`
	CustomerName    string      `json:"customerName"`
	Equipment       string      `json:"equipment"`
	DriverName      string      `json:"driverName"`
	Progress        float64     `json:"progress"`
	Late            bool        `json:"late"`
	LateMinutes     int         `json:"lateMinutes"`
	ETA             *int64      `json:"eta"`
	DeliveredAt     *int64      `json:"deliveredAt"`
	Stops           []*PeekStop `json:"stops"`
	DocumentsOnFile []string    `json:"documentsOnFile"`
}

type CustomerPeek struct {
	ID                pulid.ID        `json:"id"`
	Code              string          `json:"code"`
	Name              string          `json:"name"`
	Since             int64           `json:"since"`
	ShipmentsThisWeek int             `json:"shipmentsThisWeek"`
	WaitingInTriage   int             `json:"waitingInTriage"`
	Shipments         []*ShipmentPeek `json:"shipments"`
	Messages          []*Sample       `json:"messages"`
}
