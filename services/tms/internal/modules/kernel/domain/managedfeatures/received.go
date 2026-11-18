package managedfeatures

import (
	"cmp"
	"net/url"
	"regexp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	MaxAddOnEndings = 50
	MaxPlanURLLen   = 2048
)

var addOnKeyShape = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)

type AddOnEnding struct {
	AddOn  platformcatalog.AddOnKey `json:"addOn"`
	EndsAt int64                    `json:"endsAt"`
}

type Received struct {
	Features        []platformcatalog.FeatureKey
	PlanURL         *string
	AddOnEndings    *[]AddOnEnding
	PlanEndsAt      int64
	FinalBackupDays int
}

func (r *Received) Validate(now int64) error {
	me := errortypes.NewMultiError()
	if r.PlanURL != nil && !validPlanURL(*r.PlanURL) {
		me.Add(
			"planUrl",
			errortypes.ErrInvalid,
			"The plan URL must be an absolute http or https URL with a host and no user information, at most 2048 characters",
		)
	}
	switch {
	case r.PlanEndsAt == 0:
	case r.PlanEndsAt < 0:
		me.Add("planEndsAt", errortypes.ErrInvalid, "The plan's end is not a valid time")
	case r.PlanEndsAt > now+PlanEndHorizon:
		me.Add("planEndsAt", errortypes.ErrInvalid, "The plan cannot end more than 400 days from now")
	}
	if r.FinalBackupDays != 0 && (r.FinalBackupDays < 1 || r.FinalBackupDays > MaxFinalBackupDays) {
		me.Add(
			"finalBackupDays",
			errortypes.ErrInvalid,
			"The final backup is kept for 1 to {0} days",
			MaxFinalBackupDays,
		)
	}
	if r.AddOnEndings == nil {
		return me.Err()
	}
	endings := *r.AddOnEndings
	if len(endings) > MaxAddOnEndings {
		me.Add("addonEndings", errortypes.ErrInvalid, "At most {0} add-on endings", MaxAddOnEndings)
	}
	for i, ending := range endings {
		field := me.WithIndex("addonEndings", i)
		if !addOnKeyShape.MatchString(string(ending.AddOn)) {
			field.Add(
				"addon",
				errortypes.ErrInvalid,
				"An add-on key is a capital letter followed by 1 to 9 capital letters or digits",
			)
		}
		if ending.EndsAt <= 0 {
			field.Add("endsAt", errortypes.ErrRequired, "The time the add-on ends is required")
		}
	}
	return me.Err()
}

func validPlanURL(raw string) bool {
	if raw == "" || len(raw) > MaxPlanURLLen {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") &&
		u.Opaque == "" &&
		u.User == nil &&
		u.Hostname() != ""
}

type NormalizedEndings struct {
	Endings []AddOnEnding
	Unknown []platformcatalog.AddOnKey
}

func NormalizeEndings(received []AddOnEnding) NormalizedEndings {
	out := NormalizedEndings{
		Endings: make([]AddOnEnding, 0, len(received)),
		Unknown: []platformcatalog.AddOnKey{},
	}
	for _, ending := range received {
		if _, known := platformcatalog.LookupAddOn(ending.AddOn); !known {
			out.Unknown = append(out.Unknown, ending.AddOn)
			continue
		}
		at := slices.IndexFunc(out.Endings, func(e AddOnEnding) bool { return e.AddOn == ending.AddOn })
		switch {
		case at < 0:
			out.Endings = append(out.Endings, ending)
		case ending.EndsAt < out.Endings[at].EndsAt:
			out.Endings[at] = ending
		}
	}
	slices.SortFunc(out.Endings, func(a, b AddOnEnding) int { return cmp.Compare(a.AddOn, b.AddOn) })
	slices.Sort(out.Unknown)
	out.Unknown = slices.Compact(out.Unknown)
	return out
}
