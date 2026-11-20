package edi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediflowclock"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const (
	startTimeout = 30 * time.Second
	stopTimeout  = 15 * time.Second
	timeLayout   = time.RFC3339
)

var advanceFlags struct {
	instance string
	by       time.Duration
	toDue    bool
	json     bool
}

var advanceCmd = &cobra.Command{
	Use:   "advance",
	Short: "Move one flow instance's clock forward and wake it, in development only",
	Long: `Moves one Lading flow instance's clock forward and wakes it through the same path the
minute timer sweep uses, then prints what the flow did.

Moving the clock forward by a duration moves every stored time the flow compares against now
back by that duration: the instance's next due time, when a tender began waiting and when its
response is due, and when each unacknowledged send and each gathered batch is due. The flow
then sees exactly what it would see had that much time passed. The advance is recorded in the
instance's history as an "advanced" event.

It refuses to run unless app.env is development.

Examples:
  trenova edi flows advance --instance edfi_01J... --by 2h
  trenova edi flows advance --instance edfi_01J... --to-due
  trenova edi flows advance --instance edfi_01J... --by 30m --json`,
	Args: cobra.NoArgs,
	RunE: runAdvance,
}

type advanceAction struct {
	Kind    string `json:"kind"`
	Set     string `json:"set,omitempty"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Due     *int64 `json:"due,omitempty"`
}

type advanceSent struct {
	ID             string `json:"id"`
	TransactionSet string `json:"transactionSet"`
	ControlNumber  string `json:"controlNumber"`
}

type advanceReport struct {
	InstanceID  string          `json:"instanceId"`
	Flow        string          `json:"flow"`
	Key         string          `json:"key"`
	AdvancedBy  int64           `json:"advancedBy"`
	PreviousDue *int64          `json:"previousDue"`
	Woke        bool            `json:"woke"`
	Status      string          `json:"status"`
	Reason      string          `json:"reason,omitempty"`
	NextDue     *int64          `json:"nextDue"`
	Actions     []advanceAction `json:"actions"`
	Sent        []advanceSent   `json:"sent"`
	Error       string          `json:"error,omitempty"`
}

func runAdvance(cmd *cobra.Command, _ []string) error {
	if cfg == nil || !cfg.App.IsDevelopment() {
		env := ""
		if cfg != nil {
			env = cfg.App.Env
		}
		return fmt.Errorf(
			"edi flows advance runs only in development; app.env is %q",
			env,
		)
	}

	instanceID, err := pulid.Parse(strings.TrimSpace(advanceFlags.instance))
	if err != nil {
		return fmt.Errorf("invalid --instance %q: %w", advanceFlags.instance, err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	var clock *ediflowclock.Service
	app := fx.New(
		bootstrap.Options(),
		fx.Decorate(quietConfig),
		fx.Populate(&clock),
		fx.NopLogger,
	)
	if err = app.Err(); err != nil {
		return fmt.Errorf("build the application: %w", err)
	}

	startCtx, cancel := context.WithTimeout(ctx, startTimeout)
	defer cancel()
	if err = app.Start(startCtx); err != nil {
		return fmt.Errorf("start the application: %w", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
		defer stopCancel()
		_ = app.Stop(stopCtx)
	}()

	advance, advanceErr := clock.Advance(ctx, &ediflowclock.AdvanceRequest{
		InstanceID: instanceID,
		By:         advanceFlags.by,
		ToDue:      advanceFlags.toDue,
	})
	if advance == nil {
		return advanceErr
	}

	report := buildReport(advance, advanceErr)
	out := cmd.OutOrStdout()
	if advanceFlags.json {
		if err = writeJSON(out, report); err != nil {
			return errors.Join(err, advanceErr)
		}
	} else {
		writeText(out, report)
	}
	return advanceErr
}

func quietConfig(c *config.Config) *config.Config {
	c.App.Debug = false
	c.Logging.Output = "stderr"
	c.Logging.Level = "warn"
	c.Logging.Stacktrace = false
	return c
}

func buildReport(advance *ediflowclock.Advance, advanceErr error) *advanceReport {
	instance := advance.Timer.Instance
	report := &advanceReport{
		InstanceID:  instance.ID.String(),
		Flow:        instance.Flow,
		Key:         instance.Key,
		AdvancedBy:  int64(advance.By / time.Second),
		PreviousDue: advance.PreviousDue,
		Woke:        advance.Timer.Woke,
		Status:      string(instance.Status),
		NextDue:     instance.WakeAt,
		Actions:     make([]advanceAction, 0, len(advance.Timer.Actions)),
		Sent:        make([]advanceSent, 0, len(advance.Timer.Sent)),
	}
	if instance.Reason != nil {
		report.Reason = instance.Reason.String()
	}
	for index := range advance.Timer.Actions {
		report.Actions = append(report.Actions, reportAction(&advance.Timer.Actions[index]))
	}
	for _, message := range advance.Timer.Sent {
		report.Sent = append(report.Sent, advanceSent{
			ID:             message.ID.String(),
			TransactionSet: string(message.TransactionSet),
			ControlNumber:  message.TransactionControlNumber,
		})
	}
	if advanceErr != nil {
		report.Error = advanceErr.Error()
	}
	return report
}

func reportAction(action *lading.FlowAction) advanceAction {
	reported := advanceAction{
		Kind: action.Kind,
		Set:  action.Set,
		ID:   action.ID,
		Due:  action.Due,
	}
	if action.Message != nil {
		reported.Message = action.Message.String()
	}
	return reported
}

func writeJSON(out io.Writer, report *advanceReport) error {
	encoded, err := sonic.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode the report: %w", err)
	}
	if _, err = fmt.Fprintln(out, string(encoded)); err != nil {
		return fmt.Errorf("write the report: %w", err)
	}
	return nil
}

func writeText(out io.Writer, report *advanceReport) {
	lines := []string{
		fmt.Sprintf("Flow %s, key %s (%s)", report.Flow, report.Key, report.InstanceID),
		fmt.Sprintf(
			"Advanced by %s; previously due: %s",
			seconds(report.AdvancedBy),
			instant(report.PreviousDue),
		),
	}
	if report.Woke {
		lines = append(lines, "Woke the flow with a timer event")
	} else {
		lines = append(lines, "Not due yet, so the flow was not woken")
	}
	lines = append(lines, "Status: "+report.Status)
	if report.Reason != "" {
		lines = append(lines, "Why: "+report.Reason)
	}
	lines = append(lines, "Next due: "+instant(report.NextDue))
	if len(report.Actions) == 0 {
		lines = append(lines, "Actions: none")
	} else {
		lines = append(lines, "Actions:")
		for _, action := range report.Actions {
			lines = append(lines, "  - "+describeAction(action))
		}
	}
	for _, sent := range report.Sent {
		lines = append(lines, fmt.Sprintf(
			"Sent %s, control number %s (%s)",
			sent.TransactionSet,
			sent.ControlNumber,
			sent.ID,
		))
	}
	if report.Error != "" {
		lines = append(lines, "Error: "+report.Error)
	}
	_, _ = fmt.Fprintln(out, strings.Join(lines, "\n"))
}

func describeAction(action advanceAction) string {
	parts := []string{action.Kind}
	if action.Set != "" {
		parts = append(parts, action.Set)
	}
	if action.ID != "" {
		parts = append(parts, "("+action.ID+")")
	}
	if action.Due != nil {
		parts = append(parts, "due "+instant(action.Due))
	}
	if action.Message != "" {
		parts = append(parts, "- "+action.Message)
	}
	return strings.Join(parts, " ")
}

func instant(at *int64) string {
	if at == nil {
		return "nothing"
	}
	return time.Unix(*at, 0).UTC().Format(timeLayout)
}

func seconds(value int64) string {
	return (time.Duration(value) * time.Second).String()
}

func init() {
	flags := advanceCmd.Flags()
	flags.StringVar(&advanceFlags.instance, "instance", "", "the flow instance's id")
	flags.DurationVar(&advanceFlags.by, "by", 0, "how far to move the clock, such as 2h or 90m")
	flags.BoolVar(&advanceFlags.toDue, "to-due", false,
		"move the clock to the instance's next due time")
	flags.BoolVar(&advanceFlags.json, "json", false, "print the result as JSON")
	_ = advanceCmd.MarkFlagRequired("instance")
	advanceCmd.MarkFlagsMutuallyExclusive("by", "to-due")
	advanceCmd.MarkFlagsOneRequired("by", "to-due")
	flowsCmd.AddCommand(advanceCmd)
}
