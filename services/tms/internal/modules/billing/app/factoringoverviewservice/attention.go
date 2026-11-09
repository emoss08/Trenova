package factoringoverviewservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/shared/pulid"
)

type AttentionKind string

const (
	AttentionScheduleFailed    = AttentionKind("ScheduleFailed")
	AttentionDeliveryFailing   = AttentionKind("DeliveryFailing")
	AttentionNoDelivery        = AttentionKind("NoDelivery")
	AttentionNoAgreement       = AttentionKind("NoAgreement")
	AttentionChargebacksDue    = AttentionKind("ChargebacksDue")
	AttentionFundingOverdue    = AttentionKind("FundingOverdue")
	AttentionRemittanceWaiting = AttentionKind("RemittanceWaiting")
	AttentionNoticesMissing    = AttentionKind("NoticesMissing")
	AttentionNoticesAwaiting   = AttentionKind("NoticesAwaiting")
	AttentionReadyWaiting      = AttentionKind("ReadyWaiting")
)

type Tone string

const (
	ToneDanger  = Tone("danger")
	ToneWarning = Tone("warning")
	ToneNeutral = Tone("neutral")
)

type Attention struct {
	Kind           AttentionKind
	Tone           Tone
	Count          int
	AmountMinor    int64
	At             *int64
	SubmissionID   pulid.ID
	Number         string
	InvoiceID      pulid.ID
	InvoiceNumber  string
	CustomerName   string
	RemittanceID   pulid.ID
	FileName       string
	LineCount      int
	ExceptionCount int
	Detail         string
}

func attention(read *factoringports.OverviewRead, o *Overview) []Attention {
	out := make([]Attention, 0, 8)

	if len(read.Failed) > 0 {
		latest := read.Failed[len(read.Failed)-1]
		item := Attention{
			Kind:         AttentionScheduleFailed,
			Tone:         ToneDanger,
			Count:        len(read.Failed),
			SubmissionID: latest.SubmissionID,
			Number:       latest.Number,
			Detail:       latest.LastError,
			At:           &latest.FailedAt,
		}
		for _, sub := range read.Failed {
			item.AmountMinor += sub.FaceMinor
			item.LineCount += sub.InvoiceCount
		}
		out = append(out, item)
	} else if o.Delivery.Failing() {
		out = append(out, Attention{
			Kind:   AttentionDeliveryFailing,
			Tone:   ToneDanger,
			Detail: o.Delivery.LastTestError,
			At:     o.Delivery.LastTestedAt,
		})
	}

	if !o.Delivery.Configured {
		out = append(out, Attention{Kind: AttentionNoDelivery, Tone: ToneWarning})
	}
	if read.Agreement == nil {
		out = append(out, Attention{Kind: AttentionNoAgreement, Tone: ToneWarning})
	}

	if due := read.Summary.ChargebacksDue; due.Count > 0 {
		item := Attention{
			Kind:        AttentionChargebacksDue,
			Tone:        ToneDanger,
			Count:       due.Count,
			AmountMinor: due.AmountMinor,
			At:          o.ChargebackDeadline,
		}
		if soonest := read.SoonestChargeback; soonest != nil {
			item.InvoiceID = soonest.InvoiceID
			item.InvoiceNumber = soonest.InvoiceNumber
			item.CustomerName = soonest.CustomerName
		}
		out = append(out, item)
	}

	if len(read.Overdue) > 0 {
		oldest := read.Overdue[0]
		item := Attention{
			Kind:         AttentionFundingOverdue,
			Tone:         ToneWarning,
			Count:        len(read.Overdue),
			SubmissionID: oldest.SubmissionID,
			Number:       oldest.Number,
			At:           &oldest.DueAt,
		}
		for _, sub := range read.Overdue {
			item.AmountMinor += sub.ExpectedAdvanceMinor
		}
		out = append(out, item)
	}

	if read.RemittanceCount > 0 && len(read.Remittances) > 0 {
		latest := read.Remittances[0]
		at := latest.CreatedAt
		if latest.RemittedOn != nil {
			at = *latest.RemittedOn
		}
		out = append(out, Attention{
			Kind:           AttentionRemittanceWaiting,
			Tone:           ToneNeutral,
			Count:          read.RemittanceCount,
			AmountMinor:    latest.TotalCashMinor,
			At:             &at,
			RemittanceID:   latest.RemittanceID,
			FileName:       latest.FileName,
			LineCount:      latest.LineCount,
			ExceptionCount: latest.ExceptionCount,
		})
	}

	if read.Notices.NeedsNotice > 0 {
		out = append(out, Attention{
			Kind:  AttentionNoticesMissing,
			Tone:  ToneWarning,
			Count: read.Notices.NeedsNotice,
		})
	}
	if read.Notices.Awaiting > 0 {
		out = append(out, Attention{
			Kind:  AttentionNoticesAwaiting,
			Tone:  ToneNeutral,
			Count: read.Notices.Awaiting,
		})
	}

	if late := read.Summary.ReadyUnsubmitted; late.Count > 0 {
		out = append(out, Attention{
			Kind:        AttentionReadyWaiting,
			Tone:        ToneNeutral,
			Count:       late.Count,
			AmountMinor: late.AmountMinor,
		})
	}

	return out
}
