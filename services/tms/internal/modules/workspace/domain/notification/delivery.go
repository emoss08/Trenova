package notification

import (
	"slices"
	"time"
)

type Availability string

const (
	AvailabilityAvailable = Availability("available")
	AvailabilityFocus     = Availability("focus")
	AvailabilityAway      = Availability("away")
)

type EmailDelivery string

const (
	EmailDeliveryNone   = EmailDelivery("none")
	EmailDeliveryNow    = EmailDelivery("now")
	EmailDeliveryDigest = EmailDelivery("digest")
)

type HoldReason string

const (
	HoldNone         = HoldReason("")
	HoldModeOff      = HoldReason("mode_off")
	HoldNotUrgent    = HoldReason("not_urgent")
	HoldTypeDisabled = HoldReason("type_disabled")
	HoldPaused       = HoldReason("paused")
	HoldQuietHours   = HoldReason("quiet_hours")
	HoldFocus        = HoldReason("focus")
	HoldAway         = HoldReason("away")
)

type DeliveryInput struct {
	EventType    string
	Priority     Priority
	Availability Availability
	Now          time.Time
	Location     *time.Location
}

type DeliveryDecision struct {
	Urgent      bool          `json:"urgent"`
	Desktop     bool          `json:"desktop"`
	Sound       bool          `json:"sound"`
	Email       EmailDelivery `json:"email"`
	DesktopHold HoldReason    `json:"desktopHold,omitempty"`
	EmailHold   HoldReason    `json:"emailHold,omitempty"`
}

func (p *Preference) Decide(in DeliveryInput) DeliveryDecision {
	urgent := IsUrgent(in.EventType, in.Priority)
	shared := p.sharedHold(in, urgent)

	desktopHold := shared
	if desktopHold == HoldNone {
		desktopHold = modeHold(p.DesktopMode, urgent)
	}

	emailHold := shared
	if emailHold == HoldNone {
		emailHold = modeHold(p.EmailMode, urgent)
	}

	decision := DeliveryDecision{
		Urgent:      urgent,
		Desktop:     desktopHold == HoldNone,
		Email:       EmailDeliveryNone,
		DesktopHold: desktopHold,
		EmailHold:   emailHold,
	}
	decision.Sound = decision.Desktop && p.Sound

	if emailHold == HoldNone {
		decision.Email = EmailDeliveryNow
		if p.EmailMode == DeliveryModeAll && p.EmailDigest && !urgent {
			decision.Email = EmailDeliveryDigest
		}
	}

	return decision
}

func (p *Preference) sharedHold(in DeliveryInput, urgent bool) HoldReason {
	now := in.Now.Unix()
	switch {
	case !p.IsEventTypeEnabled(in.EventType):
		return HoldTypeDisabled
	case p.IsPaused(now):
		return HoldPaused
	case in.Availability == AvailabilityAway && !urgent:
		return HoldAway
	case in.Availability == AvailabilityFocus && !urgent && in.EventType != MentionEventType:
		return HoldFocus
	case p.InQuietHours(in.Now, in.Location) && !(urgent && p.QuietHoursAllowUrgent):
		return HoldQuietHours
	default:
		return HoldNone
	}
}

func modeHold(mode DeliveryMode, urgent bool) HoldReason {
	switch mode {
	case DeliveryModeAll:
		return HoldNone
	case DeliveryModeUrgent:
		if urgent {
			return HoldNone
		}
		return HoldNotUrgent
	case DeliveryModeOff:
		return HoldModeOff
	default:
		return HoldModeOff
	}
}

func (p *Preference) InQuietHours(now time.Time, loc *time.Location) bool {
	if !p.QuietHoursEnabled || len(p.QuietHoursDays) == 0 {
		return false
	}
	start, okStart := parseClock(p.QuietHoursStart)
	end, okEnd := parseClock(p.QuietHoursEnd)
	if !okStart || !okEnd || start == end {
		return false
	}
	if loc == nil {
		loc = time.UTC
	}

	local := now.In(loc)
	minute := local.Hour()*60 + local.Minute()
	today := int(local.Weekday())

	if start < end {
		return minute >= start && minute < end && slices.Contains(p.QuietHoursDays, today)
	}

	if minute >= start {
		return slices.Contains(p.QuietHoursDays, today)
	}
	if minute < end {
		yesterday := int(local.AddDate(0, 0, -1).Weekday())
		return slices.Contains(p.QuietHoursDays, yesterday)
	}
	return false
}
