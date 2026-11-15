package schedule

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

var workflowContextType = reflect.TypeFor[workflow.Context]()

type Schedule struct {
	ID            string
	Feature       platformcatalog.FeatureKey
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
	fmt.Fprintf(h, "%d", s.Spec.Jitter)
	h.Write([]byte(s.GetWorkflowName()))
	h.Write([]byte(s.TaskQueue))
	fmt.Fprintf(h, "%d", s.OverlapPolicy)
	fmt.Fprintf(h, "%v", s.Paused)
	if len(s.Args) > 0 {
		if args, err := sonic.ConfigStd.Marshal(s.Args); err == nil {
			h.Write(args)
		} else {
			for _, arg := range s.Args {
				fmt.Fprintf(h, "%T", arg)
			}
		}
	}
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
	if s.Feature == "" {
		return ErrFeatureRequired
	}
	if _, known := platformcatalog.Lookup(s.Feature); !known {
		return fmt.Errorf("%w: %q", ErrUnknownFeature, s.Feature)
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

// ManagedByMemoKey marks, on the schedule itself, what created it. The memo is
// written once at creation and returned by List, so ownership is known without
// describing every schedule. The reconciler only ever deletes schedules marked
// as its own, which is what lets other code keep schedules of its own in the
// same namespace.
const (
	ManagedByMemoKey  = "managedBy"
	ManagedByRegistry = "schedule-registry"
)
