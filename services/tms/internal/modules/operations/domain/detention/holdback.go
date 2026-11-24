package detention

type HoldBack string

const (
	HoldBackNone             = HoldBack("")
	HoldBackNoticeMissed     = HoldBack("NoticeMissed")
	HoldBackNoticeFailed     = HoldBack("NoticeFailed")
	HoldBackNoticeLate       = HoldBack("NoticeLate")
	HoldBackDepartureByHand  = HoldBack("DepartureByHand")
	HoldBackArrivalByHand    = HoldBack("ArrivalByHand")
	HoldBackAwaitingApproval = HoldBack("AwaitingApproval")
)

type HoldBackFacts struct {
	Status             OccurrenceStatus
	NotificationStatus NotificationStatus
	NoticeDeadlineAt   *int64
	RequiresApproval   bool
	ArrivalByHand      bool
	DepartureByHand    bool
}

func (f *HoldBackFacts) HoldBack(now int64) HoldBack {
	if f.Status != OccurrenceStatusPending && f.Status != OccurrenceStatusApproved {
		return HoldBackNone
	}
	lapsed := f.NotificationStatus == NotificationStatusPending && f.NoticeDeadlineAt != nil &&
		*f.NoticeDeadlineAt < now
	switch {
	case f.NotificationStatus == NotificationStatusMissed || lapsed:
		return HoldBackNoticeMissed
	case f.NotificationStatus == NotificationStatusFailed:
		return HoldBackNoticeFailed
	case f.NotificationStatus == NotificationStatusLate:
		return HoldBackNoticeLate
	case f.DepartureByHand:
		return HoldBackDepartureByHand
	case f.ArrivalByHand:
		return HoldBackArrivalByHand
	case f.Status == OccurrenceStatusPending && f.RequiresApproval:
		return HoldBackAwaitingApproval
	}

	return HoldBackNone
}
