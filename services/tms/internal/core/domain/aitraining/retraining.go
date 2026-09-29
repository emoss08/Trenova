package aitraining

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RetrainingCycle)(nil)

const (
	RetrainingRequestedByScheduler = "scheduler"
	MaxTrainerRunes                = 255
	MaxRetrainingPathRunes         = 1024
	MaxRetrainingConfigRunes       = 1024
	MinRetrainingLookbackDays      = 30
	MaxRetrainingLookbackDays      = 1095
	MaxRetrainingMinNewExamples    = 1_000_000
	MaxRetrainingIntervalDays      = 365
	percentScale                   = 100
)

var (
	ErrRetrainingLeaseLost = errors.New("the retraining cycle is no longer held by this trainer")
	ErrRetrainingNotReady  = errors.New("the retraining cycle is not ready to train")
	ErrRetrainingClosed    = errors.New("the retraining cycle is already finished")
)

type RetrainingGate struct {
	MinAccuracyPercent  int `json:"minAccuracyPercent"`
	MaxRegressionPoints int `json:"maxRegressionPoints"`
}

func (g RetrainingGate) Check(model, baseline *ScoreSide) (passed bool, reason string) {
	if model.Scored == 0 {
		return false, "No field of the validation set was scored"
	}
	if model.Correct*percentScale < g.MinAccuracyPercent*model.Scored {
		return false, fmt.Sprintf(
			"Accuracy %s is below the minimum of %d%%",
			formatShare(model.Correct, model.Scored),
			g.MinAccuracyPercent,
		)
	}
	if intutils.RatioLeadExceedsPoints(
		baseline.Correct,
		baseline.Scored,
		model.Correct,
		model.Scored,
		g.MaxRegressionPoints,
	) {
		return false, fmt.Sprintf(
			"Accuracy %s is more than %d points below production's %s",
			formatShare(model.Correct, model.Scored),
			g.MaxRegressionPoints,
			formatShare(baseline.Correct, baseline.Scored),
		)
	}

	return true, ""
}

func formatShare(correct, scored int) string {
	return fmt.Sprintf("%.2f%%", aicorrection.Accuracy(correct, scored)*percentScale)
}

type RetrainingPolicy struct {
	LookbackDays         int
	MinNewExamples       int
	MinIntervalDays      int
	RetrainOnDrift       bool
	MaxPerOrganization   int
	ValidationPercent    int
	StructuredOutputMode aiprovider.StructuredOutputMode
	Gate                 RetrainingGate
}

type RetrainingWindow struct {
	CapturedFrom int64
	CapturedTo   int64
	NewSince     int64
}

func NewRetrainingWindow(
	now int64,
	policy *RetrainingPolicy,
	last *RetrainingCycle,
) RetrainingWindow {
	from := max(now-int64(policy.LookbackDays)*timeutils.SecondsPerDay, 0)
	since := from
	if last != nil && last.CapturedTo > since {
		since = min(last.CapturedTo, now)
	}

	return RetrainingWindow{CapturedFrom: from, CapturedTo: now, NewSince: since}
}

type RetrainingPlanInput struct {
	Now            int64
	Policy         RetrainingPolicy
	Manual         bool
	RequestedBy    string
	Note           string
	Last           *RetrainingCycle
	OpenCycle      bool
	ExportActive   bool
	NewExamples    int
	DriftingSeries int
}

func PlanRetraining(in *RetrainingPlanInput) *RetrainingCycle {
	window := NewRetrainingWindow(in.Now, &in.Policy, in.Last)
	cycle := &RetrainingCycle{
		Task:                 aicorrection.TaskShipmentDraftExtraction,
		Trigger:              RetrainingTriggerScheduled,
		RequestedBy:          in.RequestedBy,
		Note:                 in.Note,
		CapturedFrom:         window.CapturedFrom,
		CapturedTo:           window.CapturedTo,
		NewSince:             window.NewSince,
		NewExamples:          in.NewExamples,
		MinNewExamples:       in.Policy.MinNewExamples,
		DriftingProviders:    in.DriftingSeries,
		MaxPerOrganization:   in.Policy.MaxPerOrganization,
		ValidationPercent:    in.Policy.ValidationPercent,
		StructuredOutputMode: in.Policy.StructuredOutputMode,
		MinAccuracyPercent:   in.Policy.Gate.MinAccuracyPercent,
		MaxRegressionPoints:  in.Policy.Gate.MaxRegressionPoints,
	}
	if cycle.RequestedBy == "" {
		cycle.RequestedBy = RetrainingRequestedByScheduler
	}

	drift := in.Policy.RetrainOnDrift && in.DriftingSeries > 0
	switch {
	case in.Manual:
		cycle.Trigger = RetrainingTriggerManual
	case drift:
		cycle.Trigger = RetrainingTriggerDrift
	}

	switch {
	case in.OpenCycle:
		cycle.skip(RetrainingSkipCycleOpen, in.Now)
	case in.ExportActive:
		cycle.skip(RetrainingSkipExportActive, in.Now)
	case in.Manual:
		cycle.Status = RetrainingStatusExporting
	case !drift && in.Last != nil &&
		in.Now-in.Last.CreatedAt < int64(in.Policy.MinIntervalDays)*timeutils.SecondsPerDay:
		cycle.skip(RetrainingSkipTooSoon, in.Now)
	case in.NewExamples < in.Policy.MinNewExamples:
		cycle.skip(RetrainingSkipNotEnoughExamples, in.Now)
	default:
		cycle.Status = RetrainingStatusExporting
	}

	return cycle
}

type RetrainingResult struct {
	TrainingConfig string
	RunDirectory   string
	ModelDirectory string
	PromptSHA256   string
	Report         *ScoreReport
}

type RetrainingCycle struct {
	bun.BaseModel `bun:"table:ai_retraining_cycles,alias:airc" json:"-"`

	ID                   pulid.ID                        `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	Task                 aicorrection.Task               `json:"task"                 bun:"task,type:VARCHAR(50),notnull"`
	Trigger              RetrainingTrigger               `json:"trigger"              bun:"trigger,type:VARCHAR(20),notnull"`
	Status               RetrainingStatus                `json:"status"               bun:"status,type:VARCHAR(20),notnull"`
	SkipReason           RetrainingSkipReason            `json:"skipReason"           bun:"skip_reason,type:VARCHAR(30),nullzero"`
	RequestedBy          string                          `json:"requestedBy"          bun:"requested_by,type:VARCHAR(255),notnull"`
	Note                 string                          `json:"note"                 bun:"note,type:TEXT,nullzero"`
	ExportID             *pulid.ID                       `json:"exportId"             bun:"export_id,type:VARCHAR(100),nullzero"`
	CapturedFrom         int64                           `json:"capturedFrom"         bun:"captured_from,type:BIGINT,notnull"`
	CapturedTo           int64                           `json:"capturedTo"           bun:"captured_to,type:BIGINT,notnull"`
	NewSince             int64                           `json:"newSince"             bun:"new_since,type:BIGINT,notnull"`
	NewExamples          int                             `json:"newExamples"          bun:"new_examples,type:INTEGER,notnull,default:0"`
	MinNewExamples       int                             `json:"minNewExamples"       bun:"min_new_examples,type:INTEGER,notnull,default:0"`
	DriftingProviders    int                             `json:"driftingProviders"    bun:"drifting_providers,type:INTEGER,notnull,default:0"`
	MaxPerOrganization   int                             `json:"maxPerOrganization"   bun:"max_per_organization,type:INTEGER,notnull"`
	ValidationPercent    int                             `json:"validationPercent"    bun:"validation_percent,type:INTEGER,notnull"`
	StructuredOutputMode aiprovider.StructuredOutputMode `json:"structuredOutputMode" bun:"structured_output_mode,type:VARCHAR(20),notnull"`
	MinAccuracyPercent   int                             `json:"minAccuracyPercent"   bun:"min_accuracy_percent,type:INTEGER,notnull"`
	MaxRegressionPoints  int                             `json:"maxRegressionPoints"  bun:"max_regression_points,type:INTEGER,notnull"`

	Trainer        string `json:"trainer"        bun:"trainer,type:VARCHAR(255),nullzero"`
	Attempts       int    `json:"attempts"       bun:"attempts,type:INTEGER,notnull,default:0"`
	ClaimedAt      *int64 `json:"claimedAt"      bun:"claimed_at,type:BIGINT,nullzero"`
	LeaseExpiresAt *int64 `json:"leaseExpiresAt" bun:"lease_expires_at,type:BIGINT,nullzero"`

	TrainingConfig  string `json:"trainingConfig"  bun:"training_config,type:VARCHAR(1024),nullzero"`
	RunDirectory    string `json:"runDirectory"    bun:"run_directory,type:VARCHAR(1024),nullzero"`
	ModelDirectory  string `json:"modelDirectory"  bun:"model_directory,type:VARCHAR(1024),nullzero"`
	PromptSHA256    string `json:"promptSha256"    bun:"prompt_sha256,type:VARCHAR(64),nullzero"`
	Examples        int    `json:"examples"        bun:"examples,type:INTEGER,notnull,default:0"`
	ModelCorrect    int    `json:"modelCorrect"    bun:"model_correct,type:INTEGER,notnull,default:0"`
	ModelScored     int    `json:"modelScored"     bun:"model_scored,type:INTEGER,notnull,default:0"`
	BaselineCorrect int    `json:"baselineCorrect" bun:"baseline_correct,type:INTEGER,notnull,default:0"`
	BaselineScored  int    `json:"baselineScored"  bun:"baseline_scored,type:INTEGER,notnull,default:0"`
	GateMessage     string `json:"gateMessage"     bun:"gate_message,type:TEXT,nullzero"`
	FailureMessage  string `json:"failureMessage"  bun:"failure_message,type:TEXT,nullzero"`

	FinishedAt *int64 `json:"finishedAt" bun:"finished_at,type:BIGINT,nullzero"`
	Version    int64  `json:"version"    bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt  int64  `json:"createdAt"  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt  int64  `json:"updatedAt"  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (c *RetrainingCycle) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.Task,
			validation.Required.Error("Task is required"),
			domainvalidation.ValidEnum[aicorrection.Task]("Task is invalid"),
		),
		validation.Field(&c.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[RetrainingTrigger]("Trigger is invalid"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RetrainingStatus]("Status is invalid"),
		),
		validation.Field(&c.SkipReason,
			domainvalidation.ValidEnum[RetrainingSkipReason]("Skip reason is invalid"),
		),
		validation.Field(
			&c.StructuredOutputMode,
			validation.Required.Error("Structured output mode is required"),
			domainvalidation.ValidEnum[aiprovider.StructuredOutputMode](
				"Structured output mode is invalid",
			),
		),
		validation.Field(&c.RequestedBy,
			validation.Required.Error("Say who is requesting the retraining"),
			validation.RuneLength(1, MaxRequestedByRunes).Error(
				fmt.Sprintf("Requested by must be at most %d characters", MaxRequestedByRunes),
			),
		),
		validation.Field(&c.Note,
			validation.RuneLength(0, MaxNoteRunes).Error(
				fmt.Sprintf("The note must be at most %d characters", MaxNoteRunes),
			),
		),
		validation.Field(&c.MaxPerOrganization,
			validation.Min(1).Error("Export at least one example per organization"),
			validation.Max(MaxMaxPerOrganization).Error(
				fmt.Sprintf("Export at most %d examples per organization", MaxMaxPerOrganization),
			),
		),
		validation.Field(&c.ValidationPercent,
			validation.Min(1).Error("Hold out at least one percent for validation"),
			validation.Max(MaxValidationPercent).Error(
				fmt.Sprintf("The validation share must be at most %d percent", MaxValidationPercent),
			),
		),
		validation.Field(&c.MinAccuracyPercent,
			validation.Min(0).Error("The minimum accuracy cannot be negative"),
			validation.Max(percentScale).Error("The minimum accuracy must be at most 100 percent"),
		),
		validation.Field(&c.MaxRegressionPoints,
			validation.Min(0).Error("The allowed regression cannot be negative"),
			validation.Max(percentScale).Error("The allowed regression must be at most 100 points"),
		),
		validation.Field(&c.Trainer,
			validation.RuneLength(0, MaxTrainerRunes).Error(
				fmt.Sprintf("The trainer name must be at most %d characters", MaxTrainerRunes),
			),
		),
		validation.Field(&c.TrainingConfig,
			validation.RuneLength(0, MaxRetrainingConfigRunes).Error(
				fmt.Sprintf("The training config must be at most %d characters", MaxRetrainingConfigRunes),
			),
		),
		validation.Field(&c.RunDirectory,
			validation.RuneLength(0, MaxRetrainingPathRunes).Error(
				fmt.Sprintf("The run directory must be at most %d characters", MaxRetrainingPathRunes),
			),
		),
		validation.Field(&c.ModelDirectory,
			validation.RuneLength(0, MaxRetrainingPathRunes).Error(
				fmt.Sprintf("The model directory must be at most %d characters", MaxRetrainingPathRunes),
			),
		),
	))

	if c.CapturedFrom < 0 || c.CapturedFrom >= c.CapturedTo {
		multiErr.Add("capturedTo", errortypes.ErrInvalid, "The window must end after it starts")
	}
	if (c.Status == RetrainingStatusSkipped) != (c.SkipReason != "") {
		multiErr.Add("skipReason", errortypes.ErrInvalid, "Only a skipped cycle has a skip reason")
	}
}

func (c *RetrainingCycle) skip(reason RetrainingSkipReason, now int64) {
	c.Status = RetrainingStatusSkipped
	c.SkipReason = reason
	c.FinishedAt = &now
}

func (c *RetrainingCycle) StartedExport(exportID pulid.ID) {
	c.ExportID = &exportID
}

func (c *RetrainingCycle) ExportFinished(export *TrainingExport, now int64) bool {
	if c.Status != RetrainingStatusExporting || export == nil {
		return false
	}

	switch export.Status {
	case ExportStatusCompleted:
		if export.TrainExamples == 0 || export.ValidationExamples == 0 {
			c.fail(fmt.Sprintf(
				"The export kept %d training and %d validation examples; both are needed",
				export.TrainExamples,
				export.ValidationExamples,
			), now)
			return true
		}
		c.Status = RetrainingStatusReady
		return true
	case ExportStatusFailed, ExportStatusCanceled:
		message := "The training export was " + stringutils.HumanizeCamelCase(
			export.Status.String(),
		)
		if export.FailureMessage != "" {
			message += ": " + export.FailureMessage
		}
		c.fail(message, now)
		return true
	case ExportStatusQueued, ExportStatusRunning:
		return false
	default:
		return false
	}
}

func (c *RetrainingCycle) Claimable(now int64) bool {
	switch c.Status {
	case RetrainingStatusReady:
		return true
	case RetrainingStatusTraining:
		return c.LeaseExpiresAt != nil && *c.LeaseExpiresAt <= now
	case RetrainingStatusSkipped,
		RetrainingStatusExporting,
		RetrainingStatusPassed,
		RetrainingStatusRejected,
		RetrainingStatusFailed,
		RetrainingStatusCanceled:
		return false
	default:
		return false
	}
}

func (c *RetrainingCycle) Claim(trainer string, now, lease int64) error {
	if !c.Claimable(now) {
		return errortypes.NewBusinessError(
			"Retraining cycle {0} is {1}, not ready to train",
			c.ID.String(),
			stringutils.HumanizeCamelCase(c.Status.String()),
		).WithInternal(ErrRetrainingNotReady)
	}

	expires := now + lease
	c.Status = RetrainingStatusTraining
	c.Trainer = trainer
	c.ClaimedAt = &now
	c.LeaseExpiresAt = &expires
	c.Attempts++

	return nil
}

func (c *RetrainingCycle) HeldBy(trainer string) bool {
	return c.Status == RetrainingStatusTraining && c.Trainer == trainer
}

func (c *RetrainingCycle) Extend(trainer string, now, lease int64) error {
	if !c.HeldBy(trainer) {
		return c.leaseLost()
	}

	expires := now + lease
	c.LeaseExpiresAt = &expires

	return nil
}

func (c *RetrainingCycle) Settle(trainer string, result *RetrainingResult, now int64) error {
	if !c.HeldBy(trainer) {
		return c.leaseLost()
	}
	if result == nil || result.Report == nil {
		return errortypes.NewValidationError(
			"report",
			errortypes.ErrRequired,
			"A score report is required",
		)
	}

	report := result.Report
	c.TrainingConfig = result.TrainingConfig
	c.RunDirectory = result.RunDirectory
	c.ModelDirectory = result.ModelDirectory
	c.PromptSHA256 = result.PromptSHA256
	c.Examples = report.Examples
	c.ModelCorrect = report.Model.Correct
	c.ModelScored = report.Model.Scored
	c.BaselineCorrect = report.Baseline.Correct
	c.BaselineScored = report.Baseline.Scored

	passed, message := c.Gate().Check(&report.Model, &report.Baseline)
	c.GateMessage = message
	if passed {
		c.Status = RetrainingStatusPassed
	} else {
		c.Status = RetrainingStatusRejected
	}
	c.release(now)

	return nil
}

func (c *RetrainingCycle) FailTraining(trainer, message string, now int64) error {
	if !c.HeldBy(trainer) {
		return c.leaseLost()
	}
	c.fail(message, now)

	return nil
}

func (c *RetrainingCycle) Cancel(now int64) error {
	if !c.Status.IsOpen() {
		return errortypes.NewBusinessError(
			"Only an exporting, ready or training cycle can be canceled; this one is {0}",
			stringutils.HumanizeCamelCase(c.Status.String()),
		).WithInternal(ErrRetrainingClosed)
	}
	c.Status = RetrainingStatusCanceled
	c.release(now)

	return nil
}

func (c *RetrainingCycle) Fail(message string, now int64) {
	c.fail(message, now)
}

func (c *RetrainingCycle) Gate() RetrainingGate {
	return RetrainingGate{
		MinAccuracyPercent:  c.MinAccuracyPercent,
		MaxRegressionPoints: c.MaxRegressionPoints,
	}
}

func (c *RetrainingCycle) ModelAccuracy() float64 {
	return aicorrection.Accuracy(c.ModelCorrect, c.ModelScored)
}

func (c *RetrainingCycle) BaselineAccuracy() float64 {
	return aicorrection.Accuracy(c.BaselineCorrect, c.BaselineScored)
}

func (c *RetrainingCycle) fail(message string, now int64) {
	c.Status = RetrainingStatusFailed
	c.FailureMessage = stringutils.TruncateRunes(message, MaxFailureRunes)
	c.release(now)
}

func (c *RetrainingCycle) release(now int64) {
	c.LeaseExpiresAt = nil
	c.FinishedAt = &now
}

func (c *RetrainingCycle) leaseLost() error {
	return errortypes.NewBusinessError(
		"Retraining cycle {0} is no longer held by this trainer",
		c.ID.String(),
	).WithInternal(ErrRetrainingLeaseLost)
}

func (c *RetrainingCycle) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("airc_")
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
