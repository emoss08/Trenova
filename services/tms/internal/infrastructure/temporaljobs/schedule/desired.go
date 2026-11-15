package schedule

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// desired is a declared schedule as the plan lets it run. A schedule whose
// feature the plan does not include is withheld: it still exists in Temporal,
// paused, so buying the feature only has to unpause it.
type desired struct {
	*Schedule
	withheld bool
}

func (d desired) paused() bool {
	return d.Paused || d.withheld
}

func (d desired) reason() string {
	if !d.withheld {
		return ""
	}
	feature, _ := platformcatalog.Lookup(d.Feature)
	return fmt.Sprintf("Paused because the plan does not include %s.", feature.Name)
}

// hash is the change hash of the schedule's whole desired state. A schedule
// the plan includes hashes exactly as its declaration does, so an instance
// with every feature on (every self-hosted one) sees no change at all; a
// withheld one adds its paused state and the reason, so the plan turning a
// feature on or off is a change the reconciler applies.
func (d desired) hash() string {
	declared := d.Hash()
	if !d.withheld {
		return declared
	}

	h := sha256.New()
	h.Write([]byte(declared))
	h.Write([]byte("withheld"))
	h.Write([]byte(d.reason()))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (d desired) note() string {
	if !d.withheld {
		return registryNote(d.hash())
	}
	return d.reason() + " " + registryNote(d.hash())
}

// state is the paused state and note an update leaves on a schedule whose
// current state is current. The registry owns the paused state while the note
// it wrote is in place. Pausing or unpausing a schedule by hand in Temporal
// always replaces the note, and from then on the paused state is the person's:
// an update keeps it, and marks the note so the next pass sees no change. A
// plan that withholds the feature takes the paused state back, always.
func (d desired) state(current client.ScheduleState) (bool, string) {
	if d.withheld || current.Note == "" || registryOwnsPause(current.Note) {
		return d.paused(), d.note()
	}
	return current.Paused, keptNote(current.Note, d.hash())
}

func (d desired) options() client.ScheduleOptions {
	timezone := d.Spec.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	spec := client.ScheduleSpec{
		TimeZoneName: timezone,
		Jitter:       d.Spec.Jitter,
	}

	if d.Spec.StartAt != nil {
		spec.StartAt = *d.Spec.StartAt
	}
	if d.Spec.EndAt != nil {
		spec.EndAt = *d.Spec.EndAt
	}

	if d.Spec.IsInterval() {
		spec.Intervals = []client.ScheduleIntervalSpec{{Every: d.Spec.Interval}}
	} else if d.Spec.IsCron() {
		spec.CronExpressions = []string{d.Spec.Cron}
	}

	hash := d.hash()
	memo := maps.Clone(d.Memo)
	if memo == nil {
		memo = make(map[string]any, 2)
	}
	memo["scheduleHash"] = hash
	memo["description"] = d.Description

	overlapPolicy := d.OverlapPolicy
	if overlapPolicy == enums.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED {
		overlapPolicy = enums.SCHEDULE_OVERLAP_POLICY_SKIP
	}

	return client.ScheduleOptions{
		ID:     d.ID,
		Spec:   spec,
		Paused: d.paused(),
		Note:   d.note(),
		Memo:   map[string]any{ManagedByMemoKey: ManagedByRegistry},
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-run-%d", d.ID, timeutils.NowUnix()),
			Workflow:  d.Workflow,
			TaskQueue: d.TaskQueue,
			Args:      d.Args,
			Memo:      memo,
		},
		Overlap: overlapPolicy,
	}
}

// registryNotePrefix carries the registry's change hash in the schedule's note.
// The note, unlike the schedule memo, can be changed by an update and is
// returned by List, so a schedule's current definition can be compared with the
// desired one without a describe call per schedule, and the comparison stays
// right after the schedule has been updated. It ends the note, after the reason
// a withheld schedule is paused.
const registryNotePrefix = "Managed by the schedule registry. hash="

// keptNotePrefix ends the note of a schedule whose paused state a person set
// by hand, after the note they left, so the registry knows to keep it.
const keptNotePrefix = "Paused state set by hand; the schedule registry keeps it. hash="

func registryNote(hash string) string {
	return registryNotePrefix + hash
}

func keptNote(note, hash string) string {
	theirs, _, _ := strings.Cut(note, keptNotePrefix)
	theirs = strings.TrimSpace(theirs)
	if theirs == "" {
		return keptNotePrefix + hash
	}
	return theirs + " " + keptNotePrefix + hash
}

func registryOwnsPause(note string) bool {
	return strings.Contains(note, registryNotePrefix)
}

func hashFromNote(note string) (string, bool) {
	for _, prefix := range [...]string{registryNotePrefix, keptNotePrefix} {
		if _, hash, found := strings.Cut(note, prefix); found && hash != "" {
			return hash, true
		}
	}
	return "", false
}
