package deskcase

import (
	"strings"

	"github.com/emoss08/trenova/shared/pulid"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

// ChecklistKind names what a checklist says the record is ready for.
type ChecklistKind string

const (
	ChecklistReadyToBill  = ChecklistKind("ReadyToBill")
	ChecklistReadyToClose = ChecklistKind("ReadyToClose")
)

type ItemKey string

const (
	ItemDelivered        = ItemKey("delivered")
	ItemPOD              = ItemKey("pod")
	ItemPaperwork        = ItemKey("paperwork")
	ItemRateConfirmation = ItemKey("rateConfirmation")
	ItemCarrierRateCon   = ItemKey("carrierRateConfirmed")
	ItemAccessorials     = ItemKey("accessorials")
	ItemCustomerNotified = ItemKey("customerNotified")
	ItemBillingHolds     = ItemKey("billingHolds")
	ItemPosted           = ItemKey("posted")
	ItemSent             = ItemKey("sent")
	ItemDispute          = ItemKey("dispute")
	ItemPaid             = ItemKey("paid")
)

// ItemState is where one item stands. Blocked is something a person or the
// agent can do now; Pending is waiting on the world (the truck has not
// delivered, the payment is not yet due). Both keep the record from being
// ready.
type ItemState string

const (
	ItemDone      = ItemState("Done")
	ItemBlocked   = ItemState("Blocked")
	ItemPending   = ItemState("Pending")
	ItemNotNeeded = ItemState("NotNeeded")
)

// StepKey is the next thing to do about an item. The Desk names it and
// words the request to the case's agent in the person's language.
type StepKey string

const (
	StepTrackDelivery       = StepKey("track_delivery")
	StepRequestPOD          = StepKey("request_pod")
	StepRequestPaperwork    = StepKey("request_paperwork")
	StepReviewRate          = StepKey("review_rate")
	StepConfirmRate         = StepKey("confirm_rate")
	StepApproveAccessorials = StepKey("approve_accessorials")
	StepNotifyCustomer      = StepKey("notify_customer")
	StepClearHolds          = StepKey("clear_holds")
	StepMarkReady           = StepKey("mark_ready")
	StepSendInvoice         = StepKey("send_invoice")
	StepPostInvoice         = StepKey("post_invoice")
	StepWorkDispute         = StepKey("work_dispute")
	StepFollowUpPayment     = StepKey("follow_up_payment")
)

// Item is one line of a checklist. Codes are the checks behind a blocked
// item (missing_bol, credit_hold, AccessorialNotOnRateCon …), which the Desk
// words; Names are the records involved (a document type, a carrier), shown
// as they are. An optional item is shown and ticked but never blocks. An
// item the organization added carries its own Label, StepLabel and Prompt,
// and Manual says a person ticks it on the case.
type Item struct {
	Key       ItemKey   `json:"key"`
	State     ItemState `json:"state"`
	Optional  bool      `json:"optional,omitempty"`
	At        *int64    `json:"at,omitempty"`
	Codes     []string  `json:"codes,omitempty"`
	Names     []string  `json:"names,omitempty"`
	Count     int       `json:"count,omitempty"`
	Step      StepKey   `json:"step,omitempty"`
	Label     string    `json:"label,omitempty"`
	StepLabel string    `json:"stepLabel,omitempty"`
	Prompt    string    `json:"prompt,omitempty"`
	Manual    bool      `json:"manual,omitempty"`
	TickedBy  string    `json:"tickedBy,omitempty"`
}

func (i *Item) open() bool {
	return i.State == ItemBlocked || i.State == ItemPending
}

// blocking is an open item that keeps the record from being ready.
func (i *Item) blocking() bool {
	return i.open() && !i.Optional
}

// Checklist is what stands between the record and what comes next. Next is
// the first step a blocked item offers, or, once nothing blocks, what the
// record is ready for.
type Checklist struct {
	Kind  ChecklistKind `json:"kind"`
	Ready bool          `json:"ready"`
	Items []*Item       `json:"items"`
	Next  StepKey       `json:"next,omitempty"`
}

// finish works out whether the record is ready and what comes next: the
// first required item that blocks and offers a step, else the first required
// one still waiting that does, else what the record is ready for. An
// optional item never decides either.
func finish(kind ChecklistKind, items []*Item, ready StepKey) *Checklist {
	out := &Checklist{Kind: kind, Items: items, Ready: true}
	for _, item := range items {
		if !item.blocking() {
			continue
		}
		out.Ready = false
		if out.Next == "" && item.Step != "" && item.State == ItemBlocked {
			out.Next = item.Step
		}
	}
	if out.Ready {
		out.Next = ready
	}
	if out.Next == "" {
		for _, item := range items {
			if item.blocking() && item.Step != "" {
				out.Next = item.Step
				break
			}
		}
	}

	return out
}

// TickFact is a person's tick on an added step, as the checklist shows it.
type TickFact struct {
	At int64
	By string
}

// Arrangement is how the organization wants the checklist laid out for
// this record, and what ticks its added steps.
type Arrangement struct {
	// Template is the steps in order with how each counts; nil is the
	// default.
	Template TemplateItems
	// Ticks are people's ticks on the added steps, by step.
	Ticks map[ItemKey]TickFact
	// DocumentTypesOnFile are the document types with an accepted copy
	// attached to the record.
	DocumentTypesOnFile map[pulid.ID]struct{}
}

// arrange lays the built-in items and the added ones out in the template's
// order: a step that is off is left out, an optional one marked so.
func arrange(kind ChecklistKind, plan *Arrangement, builtins map[ItemKey]*Item) []*Item {
	template := plan.Template
	if template == nil {
		template = DefaultItems(kind)
	}
	steps := Normalize(kind, template)

	out := make([]*Item, 0, len(steps))
	for _, step := range steps {
		if step.Mode == ModeOff {
			continue
		}
		var item *Item
		if step.Key.IsCustom() {
			item = customItem(step, plan)
		} else {
			item = builtins[step.Key]
		}
		if item == nil {
			continue
		}
		item.Optional = step.Mode == ModeOptional
		out = append(out, item)
	}

	return out
}

// customItem is an added step as it stands for this record: ticked by a
// person on the case, or by a document of its type being on file. Its step
// is its own key, so the Desk asks what the organization wrote.
func customItem(step TemplateItem, plan *Arrangement) *Item {
	custom := step.Custom
	if custom == nil {
		return nil
	}
	item := &Item{
		Key:       step.Key,
		Label:     custom.Label,
		StepLabel: custom.StepLabel,
		Prompt:    custom.Prompt,
	}

	switch custom.Check {
	case CheckManual:
		item.Manual = true
		if tick, ok := plan.Ticks[step.Key]; ok {
			at := tick.At
			item.State, item.At, item.TickedBy = ItemDone, &at, tick.By
			return item
		}
	case CheckDocument:
		if _, ok := plan.DocumentTypesOnFile[custom.DocumentTypeID]; ok {
			item.State = ItemDone
			return item
		}
	}
	item.State = ItemBlocked
	item.Step = StepKey(step.Key)

	return item
}

// Requirement is one document the customer's billing profile asks for, and
// whether an accepted copy is on file.
type Requirement struct {
	Code      string
	Name      string
	Satisfied bool
}

// Validation is one check billing readiness failed, by its code.
type Validation struct {
	Code    string
	Message string
}

// RateCon is one carrier on the shipment and whether its rate confirmation
// came back confirmed.
type RateCon struct {
	CarrierName string
	Confirmed   bool
}

// ShipmentFacts is what the ready-to-bill checklist reads.
type ShipmentFacts struct {
	Status      shipment.Status
	DeliveredAt *int64
	// PODCode is the document type code a proof of delivery is filed
	// under; PODOnFile is whether an accepted one is attached even when
	// the customer does not require it.
	PODCode      string
	PODOnFile    bool
	Requirements []Requirement
	Validations  []Validation
	RateCons     []RateCon
	// OpenChargeIssues are the billing queue's open charge findings
	// against the rate con (AccessorialNotOnRateCon, ChargeOverRateCon).
	OpenChargeIssues []string
	// DetentionAccruing counts clocks still running; DetentionUnapproved
	// counts charges waiting on an approver or in dispute.
	DetentionAccruing   int
	DetentionUnapproved int
	// CustomerNotifiedAt is the latest emailed customer update.
	CustomerNotifiedAt *int64
	// InBillingQueue is whether the shipment is with billing already.
	InBillingQueue bool

	Arrangement
}

const (
	codeRatePrefix = "rate_"
	codeMissingBOL = "missing_bol"
)

func delivered(status shipment.Status) bool {
	switch status {
	case shipment.StatusCompleted, shipment.StatusReadyToInvoice, shipment.StatusInvoiced:
		return true
	case shipment.StatusNew,
		shipment.StatusPartiallyAssigned,
		shipment.StatusAssigned,
		shipment.StatusInTransit,
		shipment.StatusDelayed,
		shipment.StatusPartiallyCompleted,
		shipment.StatusCanceled:
		return false
	default:
		return false
	}
}

// ReadyToBill is the shipment's checklist: delivered, proof of delivery and
// the rest of the paperwork in, the rate matched to the rate confirmation,
// the accessorials approved, the customer told, and nothing holding the
// bill. Once nothing blocks, the next step is to send the invoice.
func ReadyToBill(f *ShipmentFacts) *Checklist {
	isDelivered := delivered(f.Status)
	builtins := map[ItemKey]*Item{
		ItemDelivered:        deliveredItem(f, isDelivered),
		ItemPOD:              podItem(f, isDelivered),
		ItemPaperwork:        paperworkItem(f),
		ItemRateConfirmation: rateItem(f),
		ItemCarrierRateCon:   carrierRateItem(f),
		ItemAccessorials:     accessorialsItem(f),
		ItemCustomerNotified: notifiedItem(f, isDelivered),
		ItemBillingHolds:     holdsItem(f),
	}

	ready := StepMarkReady
	if f.InBillingQueue || f.Status == shipment.StatusReadyToInvoice {
		ready = StepSendInvoice
	}

	return finish(ChecklistReadyToBill, arrange(ChecklistReadyToBill, &f.Arrangement, builtins), ready)
}

func deliveredItem(f *ShipmentFacts, isDelivered bool) *Item {
	if isDelivered {
		return &Item{Key: ItemDelivered, State: ItemDone, At: f.DeliveredAt}
	}

	return &Item{Key: ItemDelivered, State: ItemPending, Step: StepTrackDelivery}
}

func podItem(f *ShipmentFacts, isDelivered bool) *Item {
	item := &Item{Key: ItemPOD}
	required := false
	for _, req := range f.Requirements {
		if !strings.EqualFold(req.Code, f.PODCode) {
			continue
		}
		required = true
		if req.Satisfied {
			item.State = ItemDone
			return item
		}
	}

	switch {
	case f.PODOnFile:
		item.State = ItemDone
	case !required:
		item.State = ItemNotNeeded
	case !isDelivered:
		item.State = ItemPending
	default:
		item.State = ItemBlocked
		item.Step = StepRequestPOD
	}

	return item
}

func paperworkItem(f *ShipmentFacts) *Item {
	item := &Item{Key: ItemPaperwork, State: ItemNotNeeded}
	required := 0
	for _, req := range f.Requirements {
		if strings.EqualFold(req.Code, f.PODCode) {
			continue
		}
		required++
		if !req.Satisfied {
			item.Names = append(item.Names, req.Name)
		}
	}
	for _, failure := range f.Validations {
		if failure.Code == codeMissingBOL {
			item.Codes = append(item.Codes, failure.Code)
		}
	}

	switch {
	case len(item.Names) > 0 || len(item.Codes) > 0:
		item.State = ItemBlocked
		item.Step = StepRequestPaperwork
		item.Count = len(item.Names) + len(item.Codes)
	case required > 0:
		item.State = ItemDone
	}

	return item
}

func rateItem(f *ShipmentFacts) *Item {
	item := &Item{Key: ItemRateConfirmation, State: ItemDone}
	for _, failure := range f.Validations {
		if strings.HasPrefix(failure.Code, codeRatePrefix) {
			item.Codes = append(item.Codes, failure.Code)
		}
	}
	item.Codes = append(item.Codes, f.OpenChargeIssues...)
	if len(item.Codes) > 0 {
		item.State = ItemBlocked
		item.Step = StepReviewRate
		item.Count = len(item.Codes)
	}

	return item
}

// carrierRateItem is each carrier on the shipment having confirmed its rate
// confirmation; a shipment with no carrier needs none.
func carrierRateItem(f *ShipmentFacts) *Item {
	item := &Item{Key: ItemCarrierRateCon, State: ItemNotNeeded}
	if len(f.RateCons) == 0 {
		return item
	}
	for _, rateCon := range f.RateCons {
		if !rateCon.Confirmed {
			item.Names = append(item.Names, rateCon.CarrierName)
		}
	}
	if len(item.Names) > 0 {
		item.State = ItemBlocked
		item.Step = StepConfirmRate
		item.Count = len(item.Names)
		return item
	}
	item.State = ItemDone

	return item
}

func accessorialsItem(f *ShipmentFacts) *Item {
	switch {
	case f.DetentionUnapproved > 0:
		return &Item{
			Key:   ItemAccessorials,
			State: ItemBlocked,
			Count: f.DetentionUnapproved,
			Step:  StepApproveAccessorials,
		}
	case f.DetentionAccruing > 0:
		return &Item{Key: ItemAccessorials, State: ItemPending, Count: f.DetentionAccruing}
	default:
		return &Item{Key: ItemAccessorials, State: ItemDone}
	}
}

// notifiedItem is done once the customer was emailed at or after delivery;
// an update sent while the truck was rolling did not tell them it arrived.
func notifiedItem(f *ShipmentFacts, isDelivered bool) *Item {
	item := &Item{Key: ItemCustomerNotified, At: f.CustomerNotifiedAt}
	switch {
	case !isDelivered:
		item.State = ItemPending
	case f.CustomerNotifiedAt != nil &&
		(f.DeliveredAt == nil || *f.CustomerNotifiedAt >= *f.DeliveredAt):
		item.State = ItemDone
	default:
		item.State = ItemBlocked
		item.Step = StepNotifyCustomer
	}

	return item
}

func holdsItem(f *ShipmentFacts) *Item {
	item := &Item{Key: ItemBillingHolds, State: ItemDone}
	for _, failure := range f.Validations {
		if failure.Code == codeMissingBOL || strings.HasPrefix(failure.Code, codeRatePrefix) {
			continue
		}
		item.Codes = append(item.Codes, failure.Code)
	}
	if len(item.Codes) > 0 {
		item.State = ItemBlocked
		item.Step = StepClearHolds
		item.Count = len(item.Codes)
	}

	return item
}

// InvoiceFacts is what the ready-to-close checklist reads.
type InvoiceFacts struct {
	Status           invoice.Status
	SendStatus       invoice.SendStatus
	DisputeStatus    invoice.DisputeStatus
	SettlementStatus invoice.SettlementStatus
	DueDate          *int64
	Now              int64

	Arrangement
}

// ReadyToClose is the invoice's checklist: posted, sent, clear of dispute
// and paid. Payment not yet due is pending; past due it is the next thing
// to chase.
func ReadyToClose(f *InvoiceFacts) *Checklist {
	builtins := map[ItemKey]*Item{
		ItemPosted:  postedItem(f),
		ItemSent:    sentItem(f),
		ItemDispute: disputeItem(f),
		ItemPaid:    paidItem(f),
	}

	return finish(ChecklistReadyToClose, arrange(ChecklistReadyToClose, &f.Arrangement, builtins), "")
}

func postedItem(f *InvoiceFacts) *Item {
	if f.Status == invoice.StatusPosted {
		return &Item{Key: ItemPosted, State: ItemDone}
	}

	return &Item{Key: ItemPosted, State: ItemBlocked, Step: StepPostInvoice}
}

func sentItem(f *InvoiceFacts) *Item {
	switch f.SendStatus {
	case invoice.SendStatusSent:
		return &Item{Key: ItemSent, State: ItemDone}
	case invoice.SendStatusSending:
		return &Item{Key: ItemSent, State: ItemPending}
	case invoice.SendStatusNotSent, invoice.SendStatusPartiallySent, invoice.SendStatusFailed:
		return &Item{
			Key:   ItemSent,
			State: ItemBlocked,
			Codes: []string{string(f.SendStatus)},
			Step:  StepSendInvoice,
		}
	default:
		return &Item{Key: ItemSent, State: ItemBlocked, Step: StepSendInvoice}
	}
}

func disputeItem(f *InvoiceFacts) *Item {
	if f.DisputeStatus == invoice.DisputeStatusDisputed {
		return &Item{Key: ItemDispute, State: ItemBlocked, Step: StepWorkDispute}
	}

	return &Item{Key: ItemDispute, State: ItemDone}
}

func paidItem(f *InvoiceFacts) *Item {
	item := &Item{Key: ItemPaid, At: f.DueDate}
	switch {
	case f.SettlementStatus == invoice.SettlementStatusPaid:
		item.State = ItemDone
	case f.DueDate != nil && *f.DueDate < f.Now:
		item.State = ItemBlocked
		item.Codes = []string{string(f.SettlementStatus)}
		item.Step = StepFollowUpPayment
	default:
		item.State = ItemPending
		item.Codes = []string{string(f.SettlementStatus)}
	}

	return item
}
