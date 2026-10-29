package billingqueuereview

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// eldToleranceMinutes is how far the driver's dock time may drift from the
// ELD's before the difference is worth a person's look. Below it the two are
// the same wait read off two clocks.
const eldToleranceMinutes = 15

// DetentionDwell is one detention charge set against what the truck's own
// clock says about the same wait.
type DetentionDwell struct {
	OccurrenceID pulid.ID `json:"occurrenceId"`
	ChargeID     pulid.ID `json:"chargeId"`
	LocationName string   `json:"locationName"`
	// LoggedMinutes is the dwell the charge was billed on.
	LoggedMinutes int32 `json:"loggedMinutes"`
	// ELDMinutes is the dwell the telematics or geofence evidence shows.
	ELDMinutes   int32           `json:"eldMinutes"`
	FreeMinutes  int32           `json:"freeMinutes"`
	HourlyRate   decimal.Decimal `json:"hourlyRate"`
	BilledAmount decimal.Decimal `json:"billedAmount"`
	ELDAmount    decimal.Decimal `json:"eldAmount"`
}

// ELDBillableMinutes is the wait the ELD supports past the free time.
func (d *DetentionDwell) ELDBillableMinutes() int32 {
	return max(d.ELDMinutes-d.FreeMinutes, 0)
}

// LoggedBillableMinutes is the wait the charge bills past the free time.
func (d *DetentionDwell) LoggedBillableMinutes() int32 {
	return max(d.LoggedMinutes-d.FreeMinutes, 0)
}

// Disagrees reports whether the ELD tells a different enough story.
func (d *DetentionDwell) Disagrees() bool {
	delta := d.ELDBillableMinutes() - d.LoggedBillableMinutes()
	if delta < 0 {
		delta = -delta
	}

	return delta >= eldToleranceMinutes && !d.ELDAmount.Equal(d.BilledAmount)
}

const optionKeep = "keep"

// FindingInput is what the deterministic checks look at.
type FindingInput struct {
	CustomerName     string
	ShipmentRef      string
	Charges          *billingqueue.ChargeReview
	POD              *PODState
	Detention        []*DetentionDwell
	DetentionHolds   []*billingqueue.DetentionHold
	RatingReview     *billingqueue.RatingReviewHold
	Duplicates       []*billingqueue.DuplicateRef
	CreditHold       bool
	CreditHoldReason string
	PayerShareError  string
}

// Findings is every issue the item has right now, unsaved. The same input
// gives the same findings with the same keys, which is what lets the sync
// that stores them run on every read without raising anything twice.
func Findings(in *FindingInput) []*billingqueue.Issue {
	out := make([]*billingqueue.Issue, 0, 4)
	customer := in.CustomerName
	if customer == "" {
		customer = "The customer"
	}

	if in.PayerShareError != "" {
		out = append(out, &billingqueue.Issue{
			CheckKey:  billingqueue.CheckCharges,
			Code:      billingqueue.IssueChargesUnsplittable,
			Summary:   "The charges can't be divided between payers",
			Reasoning: in.PayerShareError,
		})
	}

	for _, hold := range in.DetentionHolds {
		where := ""
		if hold.LocationName != "" {
			where = " at " + hold.LocationName
		}
		out = append(out, &billingqueue.Issue{
			CheckKey:   billingqueue.CheckCharges,
			Code:       billingqueue.IssueDetentionAwaitingApproval,
			SubjectKey: hold.OccurrenceID.String(),
			Summary: FormatMoney(
				hold.BillableAmount,
			) + " detention" + where + " waits on an approver",
			Reasoning: "The item can't be approved until this detention charge is approved or dropped on the shipment.",
		})
	}

	if hold := in.RatingReview; hold != nil {
		out = append(out, &billingqueue.Issue{
			CheckKey:   billingqueue.CheckCharges,
			Code:       billingqueue.IssueRatingAwaitingReview,
			SubjectKey: hold.ShipmentID.String(),
			Summary:    "The tariff held this rating for review",
			Reasoning: "The item can't be approved until a person approves the rating: " +
				hold.Summary() + ".",
		})
	}

	out = append(out, detentionFindings(in.Detention)...)
	out = append(out, chargeFindings(in.Charges, customer, in.Detention)...)
	out = append(out, podFindings(in.POD, customer)...)

	if in.CreditHold {
		reason := in.CreditHoldReason
		if reason == "" {
			reason = customer + " is past its credit terms. Accounts receivable has to release the hold before anything more is billed to them."
		}
		out = append(out, &billingqueue.Issue{
			CheckKey:  billingqueue.CheckTerms,
			Code:      billingqueue.IssueBillToCreditHold,
			Summary:   customer + " is on credit hold",
			Reasoning: reason,
		})
	}

	for _, dup := range in.Duplicates {
		what := "Queue item " + dup.Number
		if dup.Kind == "invoice" {
			what = "Invoice " + dup.Number
		}
		out = append(out, &billingqueue.Issue{
			CheckKey:   billingqueue.CheckDuplicate,
			Code:       billingqueue.IssuePossibleDuplicate,
			SubjectKey: dup.ID.String(),
			Summary:    what + " already bills this shipment",
			Reasoning: "Both bill " + customer + " for " + in.ShipmentRef +
				". If they're for different work, bill it; otherwise put this one on hold.",
			Options: []billingqueue.IssueOption{{
				Key:    "bill",
				Label:  "Bill it anyway",
				Done:   "Billed alongside " + dup.Number,
				Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectAccept},
			}},
		})
	}

	for _, issue := range out {
		issue.Source = billingqueue.IssueSourceDeterministic
		if issue.Options == nil {
			issue.Options = []billingqueue.IssueOption{}
		}
	}

	return out
}

func detentionFindings(dwells []*DetentionDwell) []*billingqueue.Issue {
	out := make([]*billingqueue.Issue, 0, len(dwells))
	for _, dwell := range dwells {
		if dwell == nil || !dwell.Disagrees() {
			continue
		}
		chargeID := dwell.ChargeID
		eldBillable := dwell.ELDBillableMinutes()
		reasoning := fmt.Sprintf(
			"The driver logged %s at the dock, but the ELD shows %s — that's %s billable, not %s.",
			FormatHours(dwell.LoggedMinutes),
			FormatDuration(dwell.ELDMinutes),
			FormatHours(eldBillable),
			FormatHours(dwell.LoggedBillableMinutes()),
		)
		options := make([]billingqueue.IssueOption, 0, 2)
		if eldBillable > 0 {
			options = append(options, billingqueue.IssueOption{
				Key:    "eld",
				Label:  "Bill ELD time · " + FormatMoney(dwell.ELDAmount),
				Amount: decimal.NewNullDecimal(dwell.ELDAmount),
				Done:   "Detention set to ELD time",
				Effect: billingqueue.IssueEffect{
					Kind:     billingqueue.EffectSet,
					ChargeID: chargeID,
					Amount:   decimal.NewNullDecimal(dwell.ELDAmount),
					Basis: FormatHours(eldBillable) + " × " + FormatMoney(dwell.HourlyRate) +
						" · from ELD",
				},
			})
		} else {
			options = append(options, billingqueue.IssueOption{
				Key:    optionKeep,
				Label:  "Keep it · bill " + FormatMoney(dwell.BilledAmount),
				Amount: decimal.NewNullDecimal(dwell.BilledAmount),
				Done:   "Detention kept as logged",
				Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectKeep, ChargeID: chargeID},
			})
		}
		options = append(options, billingqueue.IssueOption{
			Key:    "drop",
			Label:  "Remove detention",
			Done:   "Detention removed",
			Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectDrop, ChargeID: chargeID},
		})
		out = append(out, &billingqueue.Issue{
			CheckKey:        billingqueue.CheckCharges,
			Code:            billingqueue.IssueDetentionELDMismatch,
			SubjectKey:      dwell.OccurrenceID.String(),
			Summary:         "Detention doesn't match the ELD",
			Reasoning:       reasoning,
			FlaggedChargeID: &chargeID,
			Options:         options,
		})
	}

	return out
}

func chargeFindings(
	charges *billingqueue.ChargeReview,
	customer string,
	dwells []*DetentionDwell,
) []*billingqueue.Issue {
	if charges == nil || !charges.HasRateCon {
		return nil
	}
	detentionCharges := make(map[pulid.ID]struct{}, len(dwells))
	for _, dwell := range dwells {
		if dwell != nil {
			detentionCharges[dwell.ChargeID] = struct{}{}
		}
	}

	out := make([]*billingqueue.Issue, 0, 2)
	for _, line := range charges.Lines {
		if line.AdditionalChargeID.IsNil() || line.Removed || line.Adjusted {
			continue
		}
		if _, ok := detentionCharges[line.AdditionalChargeID]; ok {
			continue
		}
		chargeID := line.AdditionalChargeID
		label := strings.ToLower(line.Label)
		switch {
		case !line.Expected.Valid && line.Billed.IsPositive():
			out = append(out, &billingqueue.Issue{
				CheckKey:   billingqueue.CheckCharges,
				Code:       billingqueue.IssueAccessorialNotOnRateCon,
				SubjectKey: chargeID.String(),
				Summary:    FormatMoney(line.Billed) + " " + label + " isn't on the rate con",
				Reasoning: customer + "'s rate con doesn't price a " + label +
					". It was added to the shipment by hand, so bill it only if the contract passes it through or there's a receipt for it.",
				FlaggedChargeID: &chargeID,
				Options: []billingqueue.IssueOption{
					{
						Key:    optionKeep,
						Label:  "Keep it · bill " + FormatMoney(line.Billed),
						Amount: decimal.NewNullDecimal(line.Billed),
						Done:   capitalize(label) + " kept",
						Effect: billingqueue.IssueEffect{
							Kind:     billingqueue.EffectKeep,
							ChargeID: chargeID,
						},
					},
					{
						Key:   "drop",
						Label: "Remove it",
						Done:  capitalize(label) + " removed",
						Effect: billingqueue.IssueEffect{
							Kind:     billingqueue.EffectDrop,
							ChargeID: chargeID,
						},
					},
				},
			})
		case line.Expected.Valid && line.Billed.Sub(line.Expected.Decimal).GreaterThan(decimal.NewFromFloat(0.005)):
			over := line.Billed.Sub(line.Expected.Decimal)
			out = append(out, &billingqueue.Issue{
				CheckKey:   billingqueue.CheckCharges,
				Code:       billingqueue.IssueChargeOverRateCon,
				SubjectKey: chargeID.String(),
				Summary:    capitalize(label) + " is " + FormatMoney(over) + " over the rate con",
				Reasoning: customer + "'s rate con prices the " + label + " at " +
					FormatMoney(line.Expected.Decimal) + "; the shipment bills " +
					FormatMoney(line.Billed) + ".",
				FlaggedChargeID: &chargeID,
				Options: []billingqueue.IssueOption{
					{
						Key:    "ratecon",
						Label:  "Bill the rate con · " + FormatMoney(line.Expected.Decimal),
						Amount: line.Expected,
						Done:   capitalize(label) + " set to the rate con",
						Effect: billingqueue.IssueEffect{
							Kind:     billingqueue.EffectSet,
							ChargeID: chargeID,
							Amount:   line.Expected,
							Basis:    "Per rate con",
						},
					},
					{
						Key:    optionKeep,
						Label:  "Keep " + FormatMoney(line.Billed),
						Amount: decimal.NewNullDecimal(line.Billed),
						Done:   capitalize(label) + " kept at " + FormatMoney(line.Billed),
						Effect: billingqueue.IssueEffect{
							Kind:     billingqueue.EffectKeep,
							ChargeID: chargeID,
						},
					},
				},
			})
		}
	}

	return out
}

func podFindings(pod *PODState, customer string) []*billingqueue.Issue {
	if pod == nil || !pod.Required || pod.Signed {
		return nil
	}

	driver := pod.DriverName
	reasoning := customer + "'s terms need a signed POD before invoicing."
	if driver != "" {
		reasoning += " " + driver + " can re-send it from the driver app."
	}
	options := []billingqueue.IssueOption{
		{
			Key:    billingqueue.OptionKeyRequest,
			Label:  "Ask the driver for it",
			Done:   "Asked the driver for a signed POD",
			Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectRequest},
		},
		{
			Key:    "accept",
			Label:  "Bill without it",
			Done:   "Billing without a signed POD",
			Effect: billingqueue.IssueEffect{Kind: billingqueue.EffectAccept},
		},
	}
	if pod.DriverID.IsNil() {
		options = options[1:]
	}

	if !pod.Present {
		return []*billingqueue.Issue{{
			CheckKey:   billingqueue.CheckPOD,
			Code:       billingqueue.IssuePODMissing,
			SubjectKey: pod.DocumentType,
			Summary:    "There's no POD on file",
			Reasoning:  reasoning,
			Options:    options,
		}}
	}
	if !pod.Unsigned {
		return nil
	}

	return []*billingqueue.Issue{{
		CheckKey:   billingqueue.CheckPOD,
		Code:       billingqueue.IssuePODUnsigned,
		SubjectKey: pod.DocumentID.String(),
		Summary:    "POD isn't signed — the receiver's line is blank",
		Reasoning:  reasoning,
		Options:    options,
	}}
}

// FormatMoney writes an amount the way the item shows it: $1,945.65.
func FormatMoney(amount decimal.Decimal) string {
	sign := ""
	if amount.IsNegative() {
		sign = "−"
		amount = amount.Neg()
	}
	fixed := amount.StringFixed(2)
	whole, cents, _ := strings.Cut(fixed, ".")
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}

	return sign + "$" + b.String() + "." + cents
}

// FormatHours writes minutes as hours to the half: 3.5 h.
func FormatHours(minutes int32) string {
	hours := decimal.NewFromInt32(minutes).Div(decimal.NewFromInt(60)).Round(1)

	return hours.String() + " h"
}

// FormatDuration writes minutes as hours and minutes: 2 h 30 m.
func FormatDuration(minutes int32) string {
	h, m := minutes/60, minutes%60
	switch {
	case h == 0:
		return fmt.Sprintf("%d m", m)
	case m == 0:
		return fmt.Sprintf("%d h", h)
	default:
		return fmt.Sprintf("%d h %d m", h, m)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
