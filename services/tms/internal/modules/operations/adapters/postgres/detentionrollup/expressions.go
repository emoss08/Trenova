package detentionrollup

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
)

func PastFreeMinutes() keel.Int64Expr {
	o := detentiondb.DetentionOccurrences
	return keel.CastInt64(o.RawDwellMinutes.Sub(o.FreeMinutesGranted).Greatest(keel.Int32(0)))
}

func Detained() keel.Pred {
	return PastFreeMinutes().Gt(0)
}

func Causes() (late, open, early, dock keel.Pred) {
	o := detentiondb.DetentionOccurrences
	isLate := o.ArrivedLate.IsTrue()
	isOpen := o.ScheduleType.Eq(shipmentvocab.StopScheduleTypeOpen)
	isEarly := keel.And(
		o.AppointmentStart.IsNotNull(),
		o.ArrivedAt.IsNotNull(),
		keel.Int64Of(o.ArrivedAt).LtExpr(o.AppointmentStart),
	)

	late = isLate
	open = keel.And(keel.Not(isLate), isOpen)
	early = keel.And(keel.Not(isLate), keel.Not(isOpen), isEarly)
	dock = keel.And(keel.Not(isLate), keel.Not(isOpen), keel.Not(isEarly))

	return late, open, early, dock
}

func NoticeDue() keel.Pred {
	return detentiondb.DetentionOccurrences.NoticeDeadlineAt.IsNotNull()
}

func NoticeOnTime() keel.Pred {
	o := detentiondb.DetentionOccurrences
	return keel.And(
		o.NoticeDeadlineAt.IsNotNull(),
		o.NoticeSentAt.IsNotNull(),
		keel.Int64Of(o.NoticeSentAt).LteExpr(o.NoticeDeadlineAt),
	)
}

func RecoveryBucket() keel.StringExpr {
	o := detentiondb.DetentionOccurrences
	return keel.CaseString().
		When(
			keel.And(o.ArrivedLate.IsTrue(), o.Status.Eq(detention.OccurrenceStatusNotBillable)),
			keel.Text(detentionports.RecoveryLate),
		).
		When(o.SuppressedByGate.IsTrue(), keel.Text(detentionports.RecoveryNotice)).
		When(o.Status.Eq(detention.OccurrenceStatusWaived), keel.Text(detentionports.RecoveryWaived)).
		When(
			o.Status.Eq(detention.OccurrenceStatusDisputed),
			keel.Text(detentionports.RecoveryDisputed),
		).
		When(
			o.Status.In(
				detention.OccurrenceStatusAccruing,
				detention.OccurrenceStatusPending,
				detention.OccurrenceStatusApproved,
			),
			keel.Text(detentionports.RecoveryPending),
		).
		When(o.Status.Eq(detention.OccurrenceStatusBilled), keel.Text(detentionports.RecoveryBilled)).
		Else(keel.Text(detentionports.RecoveryOther))
}
