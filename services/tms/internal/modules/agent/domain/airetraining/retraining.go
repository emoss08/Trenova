package airetraining

import (
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitraining"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

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

func (g RetrainingGate) Check(model, baseline *aitraining.ScoreSide) (passed bool, reason string) {
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
	Report         *aitraining.ScoreReport
}

type RetrainingCycle struct {
	keel.Table `keel:"ai_retraining_cycles"`

	ID                   pulid.ID                        `json:"id"                   keel:"id,pk,autoid:airc_"`
	Task                 aicorrection.Task               `json:"task"                 keel:"task"`
	Trigger              RetrainingTrigger               `json:"trigger"              keel:"trigger"`
	Status               RetrainingStatus                `json:"status"               keel:"status"`
	SkipReason           RetrainingSkipReason            `json:"skipReason"           keel:"skip_reason,nullzero"`
	RequestedBy          string                          `json:"requestedBy"          keel:"requested_by"`
	Note                 string                          `json:"note"                 keel:"note,nullzero"`
	ExportID             *pulid.ID                       `json:"exportId"             keel:"export_id"`
	CapturedFrom         int64                           `json:"capturedFrom"         keel:"captured_from"`
	CapturedTo           int64                           `json:"capturedTo"           keel:"captured_to"`
	NewSince             int64                           `json:"newSince"             keel:"new_since"`
	NewExamples          int                             `json:"newExamples"          keel:"new_examples"`
	MinNewExamples       int                             `json:"minNewExamples"       keel:"min_new_examples"`
	DriftingProviders    int                             `json:"driftingProviders"    keel:"drifting_providers"`
	MaxPerOrganization   int                             `json:"maxPerOrganization"   keel:"max_per_organization"`
	ValidationPercent    int                             `json:"validationPercent"    keel:"validation_percent"`
	StructuredOutputMode aiprovider.StructuredOutputMode `json:"structuredOutputMode" keel:"structured_output_mode"`
	MinAccuracyPercent   int                             `json:"minAccuracyPercent"   keel:"min_accuracy_percent"`
	MaxRegressionPoints  int                             `json:"maxRegressionPoints"  keel:"max_regression_points"`

	Trainer        string `json:"trainer"        keel:"trainer,nullzero"`
	Attempts       int    `json:"attempts"       keel:"attempts"`
	ClaimedAt      *int64 `json:"claimedAt"      keel:"claimed_at"`
	LeaseExpiresAt *int64 `json:"leaseExpiresAt" keel:"lease_expires_at"`

	TrainingConfig  string `json:"trainingConfig"  keel:"training_config,nullzero"`
	RunDirectory    string `json:"runDirectory"    keel:"run_directory,nullzero"`
	ModelDirectory  string `json:"modelDirectory"  keel:"model_directory,nullzero"`
	PromptSHA256    string `json:"promptSha256"    keel:"prompt_sha256,nullzero"`
	Examples        int    `json:"examples"        keel:"examples"`
	ModelCorrect    int    `json:"modelCorrect"    keel:"model_correct"`
	ModelScored     int    `json:"modelScored"     keel:"model_scored"`
	BaselineCorrect int    `json:"baselineCorrect" keel:"baseline_correct"`
	BaselineScored  int    `json:"baselineScored"  keel:"baseline_scored"`
	GateMessage     string `json:"gateMessage"     keel:"gate_message,nullzero"`
	FailureMessage  string `json:"failureMessage"  keel:"failure_message,nullzero"`

	FinishedAt *int64 `json:"finishedAt" keel:"finished_at"`
	Version    int64  `json:"version"    keel:"version,version"`
	CreatedAt  int64  `json:"createdAt"  keel:"created_at,autocreate"`
	UpdatedAt  int64  `json:"updatedAt"  keel:"updated_at,autoupdate"`
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
			validation.RuneLength(1, aitraining.MaxRequestedByRunes).Error(
				fmt.Sprintf("Requested by must be at most %d characters", aitraining.MaxRequestedByRunes),
			),
		),
		validation.Field(&c.Note,
			validation.RuneLength(0, aitraining.MaxNoteRunes).Error(
				fmt.Sprintf("The note must be at most %d characters", aitraining.MaxNoteRunes),
			),
		),
		validation.Field(&c.MaxPerOrganization,
			validation.Min(1).Error("Export at least one example per organization"),
			validation.Max(aitraining.MaxMaxPerOrganization).Error(
				fmt.Sprintf("Export at most %d examples per organization", aitraining.MaxMaxPerOrganization),
			),
		),
		validation.Field(&c.ValidationPercent,
			validation.Min(1).Error("Hold out at least one percent for validation"),
			validation.Max(aitraining.MaxValidationPercent).Error(
				fmt.Sprintf("The validation share must be at most %d percent", aitraining.MaxValidationPercent),
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

func (c *RetrainingCycle) ExportFinished(export *aitraining.TrainingExport, now int64) bool {
	if c.Status != RetrainingStatusExporting || export == nil {
		return false
	}

	switch export.Status {
	case aitraining.ExportStatusCompleted:
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
	case aitraining.ExportStatusFailed, aitraining.ExportStatusCanceled:
		message := "The training export was " + stringutils.HumanizeCamelCase(
			export.Status.String(),
		)
		if export.FailureMessage != "" {
			message += ": " + export.FailureMessage
		}
		c.fail(message, now)
		return true
	case aitraining.ExportStatusQueued, aitraining.ExportStatusRunning:
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
	c.FailureMessage = stringutils.TruncateRunes(message, aitraining.MaxFailureRunes)
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
