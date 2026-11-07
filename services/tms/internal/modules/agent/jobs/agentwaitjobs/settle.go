package agentwaitjobs

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"
	"go.temporal.io/sdk/workflow"
)

// maxSettledNotices bounds how many records' notices a settled wait passes
// on; a wait watches one record, so more is a burst of unrelated keys.
const maxSettledNotices = 8

// settled is what a settling wait has heard: the latest notice per key, in
// the order the keys first arrived.
type settled struct {
	keys    []string
	details map[string]string
	first   int64
}

func (s *settled) add(met agentwaitcontract.Met) {
	if s.details == nil {
		s.details = make(map[string]string, 1)
	}
	if s.first == 0 || (met.At > 0 && met.At < s.first) {
		s.first = met.At
	}
	key := met.Key
	if _, seen := s.details[key]; !seen {
		if len(s.keys) >= maxSettledNotices {
			return
		}
		s.keys = append(s.keys, key)
	}
	s.details[key] = met.Detail
}

func (s *settled) detail() string {
	lines := make([]string, 0, len(s.keys))
	for _, key := range s.keys {
		if detail := strings.TrimSpace(s.details[key]); detail != "" {
			lines = append(lines, detail)
		}
	}

	return strings.Join(lines, "\n")
}

// settle gathers the notices that follow the first one until none has come
// for the schedule's quiet period, the settle limit has passed since the
// first, or the wait runs out, and says what they came to and when the first
// arrived. A person fixing a record edits it several times; the work picks
// up once, after the last edit.
func settle(
	ctx workflow.Context,
	met workflow.ReceiveChannel,
	first agentwaitcontract.Met,
	schedule *agentwaitcontract.Schedule,
) (string, int64) {
	heard := &settled{}
	heard.add(first)
	if heard.first == 0 {
		heard.first = workflow.Now(ctx).Unix()
	}

	quiet := time.Duration(schedule.SettleSeconds) * time.Second
	deadline := workflow.Now(ctx).Add(time.Duration(schedule.SettleMaxSeconds) * time.Second)
	if expires := time.Unix(schedule.ExpiresAt, 0); expires.Before(deadline) {
		deadline = expires
	}

	for ctx.Err() == nil {
		now := workflow.Now(ctx)
		wake := now.Add(quiet)
		if deadline.Before(wake) {
			wake = deadline
		}
		if !wake.After(now) {
			break
		}

		timerCtx, cancelTimer := workflow.WithCancel(ctx)
		timer := workflow.NewTimer(timerCtx, wake.Sub(now))
		more := false
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(met, func(c workflow.ReceiveChannel, _ bool) {
			var next agentwaitcontract.Met
			c.Receive(ctx, &next)
			heard.add(next)
			more = true
		})
		selector.AddFuture(timer, func(f workflow.Future) { _ = f.Get(ctx, nil) })
		selector.Select(ctx)
		cancelTimer()
		if !more {
			break
		}
	}

	return heard.detail(), heard.first
}
