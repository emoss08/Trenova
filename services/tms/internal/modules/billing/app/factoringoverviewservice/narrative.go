package factoringoverviewservice

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/shared/i18n"
)

type Target string

const (
	TargetDelivery  = Target("Delivery")
	TargetCustomers = Target("Customers")
	TargetLedger    = Target("Ledger")
	TargetAgreement = Target("Agreement")
)

const (
	tagStrong     = "b"
	tagDelivery   = "d"
	tagChargeback = "c"
	tagOverdue    = "o"
	tagNotices    = "n"
	tagAgreement  = "a"

	paceWorthSaying = 3
)

var narrativeTags = []string{
	tagStrong,
	tagDelivery,
	tagChargeback,
	tagOverdue,
	tagNotices,
	tagAgreement,
}

type Segment struct {
	Text   string
	Strong bool
	Tone   Tone
	Target Target
}

type narration struct {
	locale i18n.Locale
	loc    *time.Location
	read   *factoringports.OverviewRead
	result *Overview
}

type parts map[string]Segment

var narrativeParts = parts{
	tagStrong:     {Strong: true},
	tagDelivery:   {Tone: ToneDanger, Target: TargetDelivery},
	tagChargeback: {Tone: ToneDanger, Target: TargetLedger},
	tagOverdue:    {Tone: ToneWarning, Target: TargetDelivery},
	tagNotices:    {Target: TargetCustomers},
	tagAgreement:  {Target: TargetAgreement},
}

var setupParts = parts{
	tagStrong:    {Strong: true},
	tagDelivery:  {Target: TargetDelivery},
	tagAgreement: {Target: TargetAgreement},
}

func compose(pieces []i18n.RichPiece, p parts) []Segment {
	out := make([]Segment, 0, len(pieces))
	for _, piece := range pieces {
		segment, ok := p[piece.Tag]
		if piece.Tag == "" || !ok {
			segment = Segment{}
		}
		segment.Text = piece.Text
		if segment.Text != "" {
			out = append(out, segment)
		}
	}

	return out
}

func join(sentences ...[]Segment) []Segment {
	out := make([]Segment, 0, 24)
	for _, sentence := range sentences {
		if len(sentence) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, Segment{Text: " "})
		}
		out = append(out, sentence...)
	}

	return out
}

func (n *narration) money(minor int64) string {
	whole := minor / 100
	if whole < 0 {
		whole = -whole
	}

	return i18n.Translate(n.locale, "${0}", i18n.FormatNumber(n.locale, float64(whole)))
}

func (n *narration) date(at int64) string {
	return i18n.LongDate(n.locale, time.Unix(at, 0).In(n.loc))
}

func summarize(n *narration) []Segment {
	return join(
		openingSentence(n),
		deliverySentence(n),
		chargebackSentence(n),
		overdueSentence(n),
		paceSentence(n),
		noticeSentence(n),
	)
}

func openingSentence(n *narration) []Segment {
	o := n.result
	switch {
	case o.Funded.Count > 0 && o.AmountsIncluded:
		return compose(
			i18n.TranslateRich(
				n.locale,
				"{0} holds <b>{1}</b> of reserve on {2, plural, one {# funded invoice} other {# funded invoices}}.",
				narrativeTags,
				o.FactorName,
				n.money(n.read.Summary.ReserveHeld.AmountMinor),
				o.Funded.Count,
			),
			narrativeParts,
		)
	case o.Funded.Count > 0:
		return compose(
			i18n.TranslateRich(
				n.locale,
				"{0} has <b>{1, plural, one {# funded invoice} other {# funded invoices}}</b> outstanding.",
				narrativeTags,
				o.FactorName,
				o.Funded.Count,
			),
			narrativeParts,
		)
	case o.Submitted.Count > 0:
		return compose(
			i18n.TranslateRich(
				n.locale,
				"{0} has <b>{1, plural, one {# invoice} other {# invoices}}</b> on schedules waiting to be funded.",
				narrativeTags,
				o.FactorName,
				o.Submitted.Count,
			),
			narrativeParts,
		)
	case !o.Setup.Agreement && !o.Delivery.Configured:
		return compose(i18n.TranslateRich(
			n.locale,
			"<b>{0}</b> can't take invoices yet: it needs an <a>agreement</a> and a <d>way to send schedules</d>.",
			narrativeTags,
			o.FactorName,
		), setupParts)
	case !o.Setup.Agreement:
		return compose(i18n.TranslateRich(
			n.locale,
			"<b>{0}</b> can't take invoices yet: it needs an <a>agreement</a>.",
			narrativeTags,
			o.FactorName,
		), setupParts)
	case !o.Delivery.Configured:
		return compose(i18n.TranslateRich(
			n.locale,
			"<b>{0}</b> can't take invoices yet: it needs a <d>way to send schedules</d>.",
			narrativeTags,
			o.FactorName,
		), setupParts)
	default:
		return compose(
			i18n.TranslateRich(
				n.locale,
				"Nothing has been sent to {0} yet.",
				narrativeTags,
				o.FactorName,
			),
			narrativeParts,
		)
	}
}

func deliverySentence(n *narration) []Segment {
	for _, item := range n.result.Attention {
		switch item.Kind {
		case AttentionScheduleFailed:
			if item.Count == 1 {
				return compose(
					i18n.TranslateRich(
						n.locale,
						"<d>Schedule {0} didn't send</d>, so its {1, plural, one {# invoice waits} other {# invoices wait}} to go again.",
						narrativeTags,
						item.Number,
						item.LineCount,
					),
					narrativeParts,
				)
			}
			return compose(
				i18n.TranslateRich(
					n.locale,
					"<d>{0, plural, other {# schedules}} didn't send</d>, with {1, plural, one {# invoice} other {# invoices}} waiting to go again.",
					narrativeTags,
					item.Count,
					item.LineCount,
				),
				narrativeParts,
			)
		case AttentionDeliveryFailing:
			return compose(
				i18n.TranslateRich(
					n.locale,
					"<d>The last delivery test failed</d>, so the next schedule may not reach {0}.",
					narrativeTags,
					n.result.FactorName,
				),
				narrativeParts,
			)
		case AttentionNoDelivery, AttentionNoAgreement, AttentionChargebacksDue,
			AttentionFundingOverdue, AttentionRemittanceWaiting, AttentionNoticesMissing,
			AttentionNoticesAwaiting, AttentionReadyWaiting:
		}
	}

	return nil
}

func chargebackSentence(n *narration) []Segment {
	o := n.result
	if o.ChargebacksDue.Count == 0 || o.ChargebackDeadline == nil {
		return nil
	}
	if o.AmountsIncluded {
		return compose(
			i18n.TranslateRich(
				n.locale,
				"<c>{0, plural, one {# recourse invoice} other {# recourse invoices}} ({1})</c> reach the chargeback window by {2}.",
				narrativeTags,
				o.ChargebacksDue.Count,
				n.money(o.ChargebacksDue.AmountMinor),
				n.date(*o.ChargebackDeadline),
			),
			narrativeParts,
		)
	}

	return compose(
		i18n.TranslateRich(
			n.locale,
			"<c>{0, plural, one {# recourse invoice} other {# recourse invoices}}</c> reach the chargeback window by {1}.",
			narrativeTags,
			o.ChargebacksDue.Count,
			n.date(*o.ChargebackDeadline),
		),
		narrativeParts,
	)
}

func overdueSentence(n *narration) []Segment {
	for _, item := range n.result.Attention {
		if item.Kind != AttentionFundingOverdue {
			continue
		}
		if item.Count == 1 {
			return compose(
				i18n.TranslateRich(
					n.locale,
					"<o>Schedule {0}</o> is past its funding date.",
					narrativeTags,
					item.Number,
				),
				narrativeParts,
			)
		}
		return compose(
			i18n.TranslateRich(
				n.locale,
				"<o>{0, plural, other {# schedules}}</o> are past their funding date.",
				narrativeTags,
				item.Count,
			),
			narrativeParts,
		)
	}

	return nil
}

func paceSentence(n *narration) []Segment {
	o := n.result
	if o.Agreement == nil || o.PaceMedianDays == nil || len(o.Pace) < paceWorthSaying {
		return nil
	}
	if o.PaceLate == 0 {
		return compose(
			i18n.TranslateRich(
				n.locale,
				"Funding has kept to <b>{0, plural, one {# business day} other {# business days}}</b>.",
				narrativeTags,
				*o.PaceMedianDays,
			),
			narrativeParts,
		)
	}

	return compose(
		i18n.TranslateRich(
			n.locale,
			"<b>{0} of the last {1, plural, other {# schedules}}</b> funded late.",
			narrativeTags,
			o.PaceLate,
			len(o.Pace),
		),
		narrativeParts,
	)
}

func noticeSentence(n *narration) []Segment {
	notices := n.result.Notices
	switch {
	case notices.NeedsNotice > 0:
		return compose(
			i18n.TranslateRich(
				n.locale,
				"<n>{0, plural, one {# customer has} other {# customers have}} no notice</n> in force.",
				narrativeTags,
				notices.NeedsNotice,
			),
			narrativeParts,
		)
	case notices.Awaiting > 0:
		return compose(
			i18n.TranslateRich(
				n.locale,
				"<n>{0, plural, one {# notice is} other {# notices are}}</n> waiting on the customer.",
				narrativeTags,
				notices.Awaiting,
			),
			narrativeParts,
		)
	default:
		return nil
	}
}
