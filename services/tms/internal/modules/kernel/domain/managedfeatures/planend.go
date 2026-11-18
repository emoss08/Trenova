package managedfeatures

const (
	day                 = int64(24 * 60 * 60)
	PlanEndHorizon      = 400 * day
	PlanEndReminderLead = 3 * day
)

type PlanEndNoticeKind string

const (
	PlanEndSet      PlanEndNoticeKind = "set"
	PlanEndKept     PlanEndNoticeKind = "kept"
	PlanEndReminder PlanEndNoticeKind = "reminder"
)

type PlanEndNotice struct {
	Kind            PlanEndNoticeKind
	EndsAt          int64
	FinalBackupDays int
}

type planEndMarks struct {
	endsAt    int64
	announced int64
	reminded  int64
}

func marksOf(s *State) planEndMarks {
	if s == nil {
		return planEndMarks{}
	}
	return planEndMarks{
		endsAt:    s.PlanEndsAt,
		announced: s.PlanEndAnnounced,
		reminded:  s.PlanEndReminded,
	}
}

func (m planEndMarks) apply(s *State) {
	s.PlanEndsAt, s.PlanEndAnnounced, s.PlanEndReminded = m.endsAt, m.announced, m.reminded
}

func announcePlanEnd(held planEndMarks, next, now int64) (planEndMarks, *PlanEndNotice) {
	out := planEndMarks{endsAt: next, announced: held.announced, reminded: held.reminded}
	switch {
	case next != 0 && next != held.announced:
		out.announced = next
		if next-now < PlanEndReminderLead {
			out.reminded = next
		}
		if next <= now {
			return out, nil
		}
		return out, &PlanEndNotice{Kind: PlanEndSet, EndsAt: next}
	case next == 0 && held.announced != 0:
		out.announced, out.reminded = 0, 0
		return out, &PlanEndNotice{Kind: PlanEndKept, EndsAt: held.announced}
	default:
		return out, nil
	}
}

func (s *State) PlanEndReminderDue(now int64) bool {
	if s == nil || s.PlanEndsAt == 0 || s.PlanEndReminded == s.PlanEndsAt {
		return false
	}
	return now >= s.PlanEndsAt-PlanEndReminderLead && now < s.PlanEndsAt
}

func RemindPlanEnd(held *State, now int64) *Replacement {
	if !held.PlanEndReminderDue(now) {
		return &Replacement{Previous: held, Current: held}
	}
	current := *held
	current.PlanEndReminded = held.PlanEndsAt
	return &Replacement{
		Previous: held,
		Current:  &current,
		PlanEnd: &PlanEndNotice{
			Kind:            PlanEndReminder,
			EndsAt:          held.PlanEndsAt,
			FinalBackupDays: held.BackupDays(),
		},
	}
}
