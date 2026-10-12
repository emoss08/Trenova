package tenant

import (
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type AvatarStyle string

const (
	AvatarStyleAuto   AvatarStyle = "auto"
	AvatarStyleDusk   AvatarStyle = "dusk"
	AvatarStyleLagoon AvatarStyle = "lagoon"
	AvatarStyleCitrus AvatarStyle = "citrus"
	AvatarStyleOrchid AvatarStyle = "orchid"
	AvatarStyleEmber  AvatarStyle = "ember"
	AvatarStyleMint   AvatarStyle = "mint"
	AvatarStyleSlate  AvatarStyle = "slate"
)

func AvatarStyles() []AvatarStyle {
	return []AvatarStyle{
		AvatarStyleAuto,
		AvatarStyleDusk,
		AvatarStyleLagoon,
		AvatarStyleCitrus,
		AvatarStyleOrchid,
		AvatarStyleEmber,
		AvatarStyleMint,
		AvatarStyleSlate,
	}
}

func (s AvatarStyle) IsValid() bool {
	return slices.Contains(AvatarStyles(), s)
}

type AvailabilityStatus string

const (
	AvailabilityAvailable AvailabilityStatus = "available"
	AvailabilityFocus     AvailabilityStatus = "focus"
	AvailabilityAway      AvailabilityStatus = "away"
)

func AvailabilityStatuses() []AvailabilityStatus {
	return []AvailabilityStatus{AvailabilityAvailable, AvailabilityFocus, AvailabilityAway}
}

func (s AvailabilityStatus) IsValid() bool {
	return slices.Contains(AvailabilityStatuses(), s)
}

const (
	MaxJobTitleLength  = 100
	MaxPhoneLength     = 30
	MaxAwayNoteLength  = 280
	maxFocusDuration   = 24 * time.Hour
	maxAwayHorizonDays = 366
)

func (u *User) EffectiveAvailability(now int64) AvailabilityStatus {
	switch u.AvailabilityStatus {
	case AvailabilityFocus:
		if u.FocusUntil != nil && *u.FocusUntil > now {
			return AvailabilityFocus
		}
		return AvailabilityAvailable
	case AvailabilityAway:
		if u.AwayBackOn == nil || *u.AwayBackOn > now {
			return AvailabilityAway
		}
		return AvailabilityAvailable
	case AvailabilityAvailable:
		return AvailabilityAvailable
	default:
		return AvailabilityAvailable
	}
}

func (u *User) ActiveDelegate(now int64) *pulid.ID {
	if u.EffectiveAvailability(now) != AvailabilityAway {
		return nil
	}
	if u.AwayDelegateUserID == nil || u.AwayDelegateUserID.IsNil() {
		return nil
	}
	return u.AwayDelegateUserID
}

type ProfileUpdate struct {
	JobTitle    *string
	PhoneNumber *string
	AvatarStyle *AvatarStyle
}

func (p *ProfileUpdate) Normalize() {
	if p.JobTitle != nil {
		trimmed := strings.TrimSpace(*p.JobTitle)
		p.JobTitle = &trimmed
	}
	if p.PhoneNumber != nil {
		trimmed := strings.TrimSpace(*p.PhoneNumber)
		p.PhoneNumber = &trimmed
	}
}

func (p *ProfileUpdate) IsEmpty() bool {
	return p.JobTitle == nil && p.PhoneNumber == nil && p.AvatarStyle == nil
}

func (p *ProfileUpdate) Validate(multiErr *errortypes.MultiError) {
	if p.JobTitle != nil {
		if err := validation.Validate(*p.JobTitle,
			validation.RuneLength(0, MaxJobTitleLength).
				Error("Job title must be 100 characters or fewer"),
		); err != nil {
			multiErr.Add("jobTitle", errortypes.ErrInvalidLength, err.Error())
		}
	}
	if p.PhoneNumber != nil && *p.PhoneNumber != "" {
		if !validPhone(*p.PhoneNumber) {
			multiErr.Add(
				"phoneNumber",
				errortypes.ErrInvalidFormat,
				"Phone number must contain only digits, spaces and + ( ) - . characters",
			)
		}
	}
	if p.AvatarStyle != nil && !p.AvatarStyle.IsValid() {
		multiErr.Add("avatarStyle", errortypes.ErrInvalid, "Avatar style is not one Trenova offers")
	}
}

func validPhone(value string) bool {
	if len(value) > MaxPhoneLength {
		return false
	}
	digits := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == ' ' || r == '+' || r == '(' || r == ')' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return digits >= 7 && digits <= 15
}

type AvailabilityUpdate struct {
	Status         AvailabilityStatus
	FocusUntil     *int64
	BackOn         *int64
	DelegateUserID *pulid.ID
	AwayNote       string
}

func (a *AvailabilityUpdate) Normalize() {
	a.AwayNote = strings.TrimSpace(a.AwayNote)
	if a.DelegateUserID != nil && a.DelegateUserID.IsNil() {
		a.DelegateUserID = nil
	}
	switch a.Status {
	case AvailabilityAvailable:
		a.FocusUntil = nil
		a.BackOn = nil
		a.DelegateUserID = nil
		a.AwayNote = ""
	case AvailabilityFocus:
		a.BackOn = nil
		a.DelegateUserID = nil
		a.AwayNote = ""
	case AvailabilityAway:
		a.FocusUntil = nil
	}
}

func (a *AvailabilityUpdate) Validate(self pulid.ID, now int64, multiErr *errortypes.MultiError) {
	if !a.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Status must be available, focus or away")
		return
	}

	if a.Status == AvailabilityFocus {
		switch {
		case a.FocusUntil == nil:
			multiErr.Add("focusUntil", errortypes.ErrRequired, "Say how long you are focusing for")
		case *a.FocusUntil <= now:
			multiErr.Add("focusUntil", errortypes.ErrInvalid, "Focus must end in the future")
		case *a.FocusUntil > now+int64(maxFocusDuration/time.Second):
			multiErr.Add("focusUntil", errortypes.ErrInvalid, "Focus can last at most a day")
		}
	}

	if a.Status == AvailabilityAway {
		if a.BackOn != nil {
			horizon := now + int64(maxAwayHorizonDays*24*time.Hour/time.Second)
			if *a.BackOn > horizon {
				multiErr.Add("backOn", errortypes.ErrInvalid, "Back on must be within a year")
			}
		}
		if a.DelegateUserID != nil && *a.DelegateUserID == self {
			multiErr.Add("delegateUserId", errortypes.ErrInvalid, "Hand off to someone other than yourself")
		}
		if len([]rune(a.AwayNote)) > MaxAwayNoteLength {
			multiErr.Add(
				"awayNote",
				errortypes.ErrInvalidLength,
				"Away note must be 280 characters or fewer",
			)
		}
	}
}

func (u *User) ApplyAvailability(a *AvailabilityUpdate) {
	u.AvailabilityStatus = a.Status
	u.FocusUntil = a.FocusUntil
	u.AwayBackOn = a.BackOn
	u.AwayDelegateUserID = a.DelegateUserID
	u.AwayNote = a.AwayNote
}
