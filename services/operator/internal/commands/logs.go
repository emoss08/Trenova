package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/logs"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	noGrantMessage = "This staff address holds no live support access grant on the instance, " +
		"so no logs were collected. An administrator of the instance can grant support access in Trenova."
	noRequesterMessage = "The command does not say who asked for it, so no logs were collected."
	noPeriodMessage    = "The command does not say which period to collect."
	noBundleMessage    = "The command names no log bundle."
)

type requester struct {
	staff bool
	email string
}

func whoAsked(cmd *instanceapi.Command) (requester, bool) {
	by := cmd.RequestedBy
	if by == nil || string(by.Email) == "" {
		return requester{}, false
	}
	switch by.Kind {
	case instanceapi.RequestedByCustomer:
		return requester{email: string(by.Email)}, true
	case instanceapi.RequestedByStaff:
		return requester{staff: true, email: string(by.Email)}, true
	default:
		return requester{}, false
	}
}

func (r *Runner) grant(
	ctx context.Context,
	who requester,
	kind tms.LogPullKind,
) (instanceapi.CommandResult, bool) {
	if !who.staff {
		return instanceapi.CommandResult{}, true
	}
	err := r.grants.AuthorizeLogPull(ctx, who.email, kind)
	switch {
	case err == nil:
		return instanceapi.CommandResult{}, true
	case errors.Is(err, tms.ErrNoGrant):
		return refused(noGrantMessage), false
	default:
		return failedMessage("Trenova could not be asked whether support access is granted, " +
			"so no logs were collected: " + err.Error()), false
	}
}

func (r *Runner) collectLines(
	ctx context.Context,
	cmd *instanceapi.Command,
) instanceapi.CommandResult {
	component := deref(cmd.Component)
	if component == "" {
		return failed(errors.New("name the component whose logs to collect"))
	}
	who, ok := whoAsked(cmd)
	if !ok {
		return failedMessage(noRequesterMessage)
	}
	if res, granted := r.grant(ctx, who, tms.LogPullLines); !granted {
		return res
	}
	lines := defaultLines
	if cmd.Lines != nil {
		lines = min(max(*cmd.Lines, 1), maxKeptLines)
	}
	redact, err := r.redactor(ctx)
	if err != nil {
		return failed(err)
	}
	var out string
	if r.kept.Kept() {
		out, err = r.kept.Last(ctx, component, lines, redact)
	} else {
		out, err = r.kubeletLines(ctx, component, lines, redact)
	}
	if err != nil {
		return failed(err)
	}
	return succeeded("", &out)
}

func (r *Runner) collectBundle(
	ctx context.Context,
	cmd *instanceapi.Command,
) (instanceapi.CommandResult, state) {
	if cmd.BundleId == nil {
		return failedMessage(noBundleMessage), answered
	}
	id := *cmd.BundleId
	end := func(outcome logs.Outcome) (instanceapi.CommandResult, state) {
		r.kept.Finish(ctx, id, &outcome)
		return fromOutcome(&outcome), answered
	}
	if cmd.Since == nil || cmd.Until == nil {
		return end(logs.Failed(noPeriodMessage))
	}
	if msg := logs.PeriodProblem(*cmd.Since, *cmd.Until, r.kept.Longest()); msg != "" {
		return end(logs.Failed(msg))
	}
	who, ok := whoAsked(cmd)
	if !ok {
		return end(logs.Failed(noRequesterMessage))
	}
	if res, granted := r.grant(ctx, who, tms.LogPullBundle); !granted {
		if res.Status == instanceapi.CommandResultStatusRefused {
			return end(logs.Refused(deref(res.Message)))
		}
		return end(logs.Failed(deref(res.Message)))
	}
	if !r.kept.Kept() {
		return end(logs.Failed(logs.NotKeptMessage))
	}
	commandID := cmd.Id
	r.mu.Lock()
	r.pending[commandID] = struct{}{}
	r.mu.Unlock()
	submitted := r.kept.Submit(&logs.Job{
		BundleID: id,
		Since:    cmd.Since.UTC(),
		Until:    cmd.Until.UTC(),
		Done: func(o logs.Outcome) {
			r.settle(commandID, fromOutcome(&o))
			r.logger.Info("a log bundle command is ready to answer",
				"command", commandID, "bundle", id, "status", string(o.Status))
		},
	})
	if submitted == logs.QueueFull {
		r.mu.Lock()
		delete(r.pending, commandID)
		r.mu.Unlock()
		r.logger.InfoContext(
			ctx,
			"the log bundle queue is full; the command waits for the next tick",
			"command",
			commandID,
			"bundle",
			id,
		)
		return instanceapi.CommandResult{}, retry
	}
	return instanceapi.CommandResult{}, later
}

func fromOutcome(o *logs.Outcome) instanceapi.CommandResult {
	switch o.Status {
	case instanceapi.LogBundleStored:
		return succeeded(fmt.Sprintf("stored the log bundle at %s (%d bytes)", o.Key, o.Size), nil)
	case instanceapi.LogBundleRefused:
		return refused(o.Message)
	case instanceapi.LogBundleFailed, instanceapi.LogBundleCollecting:
		return failedMessage(o.Message)
	}
	return failedMessage(o.Message)
}

func refused(message string) instanceapi.CommandResult {
	msg := health.Clean(message, maxMessage)
	return instanceapi.CommandResult{Status: instanceapi.CommandResultStatusRefused, Message: &msg}
}

func failedMessage(message string) instanceapi.CommandResult {
	msg := health.Clean(message, maxMessage)
	return instanceapi.CommandResult{Status: instanceapi.CommandResultStatusFailed, Message: &msg}
}
