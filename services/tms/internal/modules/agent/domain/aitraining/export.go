package aitraining

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	ExampleFormat             = "trenova.extraction-training/v1"
	ManifestFormat            = "trenova.extraction-training-manifest/v1"
	ObjectKeyPrefix           = "ai-training-exports/"
	DefaultMaxPerOrganization = 2000
	MaxMaxPerOrganization     = 20000
	DefaultValidationPercent  = 10
	MaxValidationPercent      = 50
	MaxRequestedByRunes       = 255
	MaxNoteRunes              = 2000
	MaxFailureRunes           = 2000
	MaxParts                  = 20000
)

func ExportWorkflowID(exportID pulid.ID) string {
	return "ai-training-export:" + exportID.String()
}

type Part struct {
	Ordinal  int    `json:"ordinal"`
	Split    Split  `json:"split"`
	Key      string `json:"key"`
	Examples int    `json:"examples"`
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
}

type OrganizationProgress struct {
	Ordinal          int                `json:"ordinal"`
	Examples         int                `json:"examples"`
	ConsentWithdrawn bool               `json:"consentWithdrawn"`
	Dropped          map[DropReason]int `json:"dropped"`
	Parts            []Part             `json:"parts"`
}

type TrainingExport struct {
	keel.Table `keel:"ai_training_exports"`

	ID                 pulid.ID          `json:"id" keel:"id,pk,autoid:aitx_"`
	Task               aicorrection.Task `json:"task" keel:"task"`
	Status             ExportStatus      `json:"status" keel:"status"`
	Format             string            `json:"format" keel:"format"`
	CapturedFrom       int64             `json:"capturedFrom" keel:"captured_from"`
	CapturedTo         int64             `json:"capturedTo" keel:"captured_to"`
	MaxPerOrganization int               `json:"maxPerOrganization" keel:"max_per_organization"`
	ValidationPercent  int               `json:"validationPercent" keel:"validation_percent"`
	RequestedBy        string            `json:"requestedBy" keel:"requested_by"`
	Note               string            `json:"note" keel:"note,nullzero"`
	WorkflowID         string            `json:"workflowId" keel:"workflow_id,nullzero"`

	OrganizationsConsidered int                    `json:"organizationsConsidered" keel:"organizations_considered"`
	OrganizationsIncluded   int                    `json:"organizationsIncluded" keel:"organizations_included"`
	ExamplesTotal           int                    `json:"examplesTotal" keel:"examples_total"`
	TrainExamples           int                    `json:"trainExamples" keel:"train_examples"`
	ValidationExamples      int                    `json:"validationExamples" keel:"validation_examples"`
	Dropped                 map[DropReason]int     `json:"dropped" keel:"dropped,jsonb"`
	Parts                   []Part                 `json:"parts" keel:"parts,jsonb"`
	Progress                []OrganizationProgress `json:"progress" keel:"progress,jsonb"`

	ManifestKey    string `json:"manifestKey" keel:"manifest_key,nullzero"`
	ManifestSHA256 string `json:"manifestSha256" keel:"manifest_sha256,nullzero"`
	FailureMessage string `json:"failureMessage" keel:"failure_message,nullzero"`

	StartedAt  *int64 `json:"startedAt" keel:"started_at"`
	FinishedAt *int64 `json:"finishedAt" keel:"finished_at"`
	Version    int64  `json:"version" keel:"version,version"`
	CreatedAt  int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt  int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (e *TrainingExport) BeforeCreate(_ context.Context) error {
	if e.Dropped == nil {
		e.Dropped = map[DropReason]int{}
	}
	if e.Parts == nil {
		e.Parts = []Part{}
	}
	if e.Progress == nil {
		e.Progress = []OrganizationProgress{}
	}
	if e.Status == "" {
		e.Status = ExportStatusQueued
	}

	return nil
}

func (e *TrainingExport) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		e,
		validation.Field(&e.Task,
			validation.Required.Error("Task is required"),
			domainvalidation.ValidEnum[aicorrection.Task]("Task is invalid"),
		),
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ExportStatus]("Status is invalid"),
		),
		validation.Field(&e.Format, validation.Required.Error("Format is required")),
		validation.Field(
			&e.CapturedFrom,
			validation.Min(int64(0)).Error("The window cannot start before 1970"),
		),
		validation.Field(&e.CapturedTo, validation.Required.Error("The window must have an end")),
		validation.Field(&e.MaxPerOrganization,
			validation.Min(1).Error("Export at least one example per organization"),
			validation.Max(MaxMaxPerOrganization).Error(
				fmt.Sprintf("Export at most %d examples per organization", MaxMaxPerOrganization),
			),
		),
		validation.Field(&e.ValidationPercent,
			validation.Min(0).Error("The validation share cannot be negative"),
			validation.Max(MaxValidationPercent).Error(
				fmt.Sprintf("The validation share must be at most %d percent", MaxValidationPercent),
			),
		),
		validation.Field(&e.RequestedBy,
			validation.Required.Error("Say who is requesting the export"),
			validation.RuneLength(1, MaxRequestedByRunes).Error(
				fmt.Sprintf("Requested by must be at most %d characters", MaxRequestedByRunes),
			),
		),
		validation.Field(&e.Note,
			validation.RuneLength(0, MaxNoteRunes).Error(
				fmt.Sprintf("The note must be at most %d characters", MaxNoteRunes),
			),
		),
	))

	if e.CapturedTo != 0 && e.CapturedFrom >= e.CapturedTo {
		multiErr.Add("capturedTo", errortypes.ErrInvalid, "The window must end after it starts")
	}
}

func (e *TrainingExport) ObjectPrefix() string {
	return ObjectKeyPrefix + e.ID.String() + "/"
}

func (e *TrainingExport) PartKey(ordinal int, split Split) string {
	return fmt.Sprintf("%sparts/%05d-%s.jsonl", e.ObjectPrefix(), ordinal, split)
}

func (e *TrainingExport) ManifestObjectKey() string {
	return e.ObjectPrefix() + "manifest.json"
}

func (e *TrainingExport) RecordProgress(progress OrganizationProgress) {
	replaced := false
	for i := range e.Progress {
		if e.Progress[i].Ordinal == progress.Ordinal {
			e.Progress[i] = progress
			replaced = true
			break
		}
	}
	if !replaced {
		e.Progress = append(e.Progress, progress)
	}
	slices.SortFunc(
		e.Progress,
		func(a, b OrganizationProgress) int { return a.Ordinal - b.Ordinal },
	)
	e.RecomputeProgress()
}

func (e *TrainingExport) RecomputeProgress() {
	parts := make([]Part, 0, len(e.Progress)*2)
	dropped := map[DropReason]int{}
	e.OrganizationsConsidered = len(e.Progress)
	e.OrganizationsIncluded = 0
	for i := range e.Progress {
		parts = append(parts, e.Progress[i].Parts...)
		for reason, count := range e.Progress[i].Dropped {
			if count > 0 {
				dropped[reason] += count
			}
		}
		if e.Progress[i].Examples > 0 {
			e.OrganizationsIncluded++
		}
	}
	e.Dropped = dropped
	e.applyParts(parts)
}

func (e *TrainingExport) applyParts(parts []Part) {
	e.Parts = parts
	slices.SortFunc(e.Parts, func(a, b Part) int {
		if a.Ordinal != b.Ordinal {
			return a.Ordinal - b.Ordinal
		}
		return compareSplit(a.Split, b.Split)
	})

	e.ExamplesTotal, e.TrainExamples, e.ValidationExamples = 0, 0, 0
	for i := range e.Parts {
		e.ExamplesTotal += e.Parts[i].Examples
		switch e.Parts[i].Split {
		case SplitTrain:
			e.TrainExamples += e.Parts[i].Examples
		case SplitValidation:
			e.ValidationExamples += e.Parts[i].Examples
		}
	}
}

func (e *TrainingExport) DroppedReasons() []DropReason {
	return slices.Sorted(maps.Keys(e.Dropped))
}

func compareSplit(a, b Split) int {
	switch {
	case a == b:
		return 0
	case a == SplitTrain:
		return -1
	default:
		return 1
	}
}
