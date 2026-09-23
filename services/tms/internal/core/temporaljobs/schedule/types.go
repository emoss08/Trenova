package schedule

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

var workflowContextType = reflect.TypeFor[workflow.Context]()

type Schedule struct {
	ID            string
	Description   string
	Spec          Spec
	Workflow      any
	TaskQueue     string
	Args          []any
	OverlapPolicy enums.ScheduleOverlapPolicy
	Paused        bool
	Memo          map[string]any
}

type Spec struct {
	Cron     string
	Interval time.Duration
	Timezone string
	Jitter   time.Duration
	StartAt  *time.Time
	EndAt    *time.Time
}

func Cron(expression string) Spec {
	return Spec{Cron: expression, Timezone: "UTC"}
}

func Every(interval time.Duration) Spec {
	return Spec{Interval: interval, Timezone: "UTC"}
}

func (s Spec) WithTimezone(tz string) Spec {
	s.Timezone = tz
	return s
}

func (s Spec) WithJitter(jitter time.Duration) Spec {
	s.Jitter = jitter
	return s
}

func (s Spec) WithStartAt(t time.Time) Spec {
	s.StartAt = &t
	return s
}

func (s Spec) WithEndAt(t time.Time) Spec {
	s.EndAt = &t
	return s
}

func (s Spec) IsInterval() bool {
	return s.Interval > 0
}

func (s Spec) IsCron() bool {
	return s.Cron != ""
}

func (s *Schedule) GetWorkflowName() string {
	if s.Workflow == nil {
		return ""
	}
	fn := runtime.FuncForPC(reflect.ValueOf(s.Workflow).Pointer())
	if fn == nil {
		return ""
	}
	return fn.Name()
}

func (s *Schedule) Hash() string {
	h := sha256.New()
	h.Write([]byte(s.ID))
	h.Write([]byte(s.Spec.Cron))
	fmt.Fprintf(h, "%d", s.Spec.Interval)
	h.Write([]byte(s.Spec.Timezone))
	h.Write([]byte(s.GetWorkflowName()))
	h.Write([]byte(s.TaskQueue))
	fmt.Fprintf(h, "%d", s.OverlapPolicy)
	fmt.Fprintf(h, "%v", s.Paused)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (s *Schedule) Validate() error {
	if s.ID == "" {
		return ErrScheduleIDRequired
	}
	if s.Workflow == nil {
		return ErrWorkflowRequired
	}
	if s.TaskQueue == "" {
		return ErrTaskQueueRequired
	}
	if !s.Spec.IsInterval() && !s.Spec.IsCron() {
		return ErrInvalidScheduleSpec
	}
	return s.validateArgs()
}

func (s *Schedule) validateArgs() error {
	fnType := reflect.TypeOf(s.Workflow)
	if fnType.Kind() != reflect.Func {
		return nil
	}

	params := fnType.NumIn()
	if params > 0 && fnType.In(0) == workflowContextType {
		params--
	}

	if fnType.IsVariadic() {
		if len(s.Args) < params-1 {
			return fmt.Errorf(
				"%w: %s expects at least %d args, got %d",
				ErrWorkflowArgsInvalid,
				s.GetWorkflowName(),
				params-1,
				len(s.Args),
			)
		}
		return nil
	}

	if len(s.Args) != params {
		return fmt.Errorf(
			"%w: %s expects %d args, got %d",
			ErrWorkflowArgsInvalid,
			s.GetWorkflowName(),
			params,
			len(s.Args),
		)
	}

	return nil
}

func (s *Schedule) ToScheduleOptions() client.ScheduleOptions {
	timezone := s.Spec.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	spec := client.ScheduleSpec{
		TimeZoneName: timezone,
		Jitter:       s.Spec.Jitter,
	}

	if s.Spec.StartAt != nil {
		spec.StartAt = *s.Spec.StartAt
	}
	if s.Spec.EndAt != nil {
		spec.EndAt = *s.Spec.EndAt
	}

	if s.Spec.IsInterval() {
		spec.Intervals = []client.ScheduleIntervalSpec{{Every: s.Spec.Interval}}
	} else if s.Spec.IsCron() {
		spec.CronExpressions = []string{s.Spec.Cron}
	}

	workflowIDPrefix := fmt.Sprintf("%s-run", s.ID)
	memo := s.Memo
	if memo == nil {
		memo = make(map[string]any)
	}
	memo["scheduleHash"] = s.Hash()
	memo["description"] = s.Description

	overlapPolicy := s.OverlapPolicy
	if overlapPolicy == enums.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED {
		overlapPolicy = enums.SCHEDULE_OVERLAP_POLICY_SKIP
	}

	return client.ScheduleOptions{
		ID:     s.ID,
		Spec:   spec,
		Paused: s.Paused,
		Note:   registryNote(s.Hash()),
		Memo:   map[string]any{ManagedByMemoKey: ManagedByRegistry},
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-%d", workflowIDPrefix, timeutils.NowUnix()),
			Workflow:  s.Workflow,
			TaskQueue: s.TaskQueue,
			Args:      s.Args,
			Memo:      memo,
		},
		Overlap: overlapPolicy,
	}
}

// ManagedByMemoKey marks, on the schedule itself, what created it. The memo is
// written once at creation and returned by List, so ownership is known without
// describing every schedule. The reconciler only ever deletes schedules marked
// as its own, which is what lets other code keep schedules of its own in the
// same namespace.
const (
	ManagedByMemoKey  = "managedBy"
	ManagedByRegistry = "schedule-registry"
)

// registryNotePrefix carries the registry's change hash in the schedule's note.
// The note, unlike the schedule memo, can be changed by an update and is
// returned by List, so a schedule's current definition can be compared with the
// desired one without a describe call per schedule, and the comparison stays
// right after the schedule has been updated.
const registryNotePrefix = "Managed by the schedule registry. hash="

func registryNote(hash string) string {
	return registryNotePrefix + hash
}

func hashFromNote(note string) (string, bool) {
	hash, ok := strings.CutPrefix(note, registryNotePrefix)
	return hash, ok && hash != ""
}
