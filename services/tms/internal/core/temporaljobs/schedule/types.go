package schedule

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

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
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("%s-%d", workflowIDPrefix, time.Now().Unix()),
			Workflow:  s.Workflow,
			TaskQueue: s.TaskQueue,
			Args:      s.Args,
			Memo:      memo,
		},
		Overlap: overlapPolicy,
	}
}
