package aitraining

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validExport() *TrainingExport {
	return &TrainingExport{
		Task:               aicorrection.TaskShipmentDraftExtraction,
		Status:             ExportStatusQueued,
		Format:             ExampleFormat,
		CapturedFrom:       1_700_000_000,
		CapturedTo:         1_710_000_000,
		MaxPerOrganization: DefaultMaxPerOrganization,
		ValidationPercent:  DefaultValidationPercent,
		RequestedBy:        "ops",
	}
}

func TestExportValidate(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	validExport().Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	reversed := validExport()
	reversed.CapturedFrom = reversed.CapturedTo
	multiErr = errortypes.NewMultiError()
	reversed.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	tooMuch := validExport()
	tooMuch.ValidationPercent = MaxValidationPercent + 1
	tooMuch.MaxPerOrganization = 0
	tooMuch.RequestedBy = ""
	multiErr = errortypes.NewMultiError()
	tooMuch.Validate(multiErr)
	assert.Len(t, multiErr.Errors, 3)
}

func TestExportRecordProgressAndKeys(t *testing.T) {
	t.Parallel()

	export := validExport()
	export.ID = pulid.MustNew("aitx_")
	export.RecordProgress(OrganizationProgress{
		Ordinal:  2,
		Examples: 10,
		Dropped:  map[DropReason]int{DropResidualIdentifier: 2},
		Parts: []Part{
			{Ordinal: 2, Split: SplitValidation, Examples: 3},
			{Ordinal: 2, Split: SplitTrain, Examples: 7},
		},
	})
	export.RecordProgress(OrganizationProgress{
		Ordinal:          1,
		ConsentWithdrawn: true,
		Dropped:          map[DropReason]int{DropConsentWithdrawn: 4},
	})
	export.RecordProgress(OrganizationProgress{
		Ordinal:  3,
		Examples: 1,
		Dropped:  map[DropReason]int{DropResidualIdentifier: 1, DropNoDocumentText: 0},
		Parts:    []Part{{Ordinal: 3, Split: SplitTrain, Examples: 1}},
	})
	export.RecordProgress(OrganizationProgress{
		Ordinal:  3,
		Examples: 2,
		Dropped:  map[DropReason]int{DropResidualIdentifier: 1},
		Parts:    []Part{{Ordinal: 3, Split: SplitTrain, Examples: 2}},
	})

	assert.Equal(t, 3, export.OrganizationsConsidered)
	assert.Equal(t, 2, export.OrganizationsIncluded)
	assert.Equal(t, 12, export.ExamplesTotal)
	assert.Equal(t, 9, export.TrainExamples)
	assert.Equal(t, 3, export.ValidationExamples)
	assert.Equal(t, map[DropReason]int{DropResidualIdentifier: 3, DropConsentWithdrawn: 4}, export.Dropped)
	assert.Equal(t, SplitTrain, export.Parts[0].Split)
	assert.Equal(t, 3, export.Parts[2].Ordinal)
	assert.Equal(t, 1, export.Progress[0].Ordinal)
	assert.Equal(t,
		"ai-training-exports/"+export.ID.String()+"/parts/00002-validation.jsonl",
		export.PartKey(2, SplitValidation),
	)
	assert.Equal(t, "ai-training-exports/"+export.ID.String()+"/manifest.json", export.ManifestObjectKey())
}

func TestExportStatus(t *testing.T) {
	t.Parallel()

	assert.True(t, ExportStatusQueued.IsActive())
	assert.True(t, ExportStatusRunning.IsActive())
	assert.False(t, ExportStatusCompleted.IsActive())
	assert.False(t, ExportStatus("Paused").IsValid())
	assert.True(t, SplitValidation.IsValid())
}

func TestSplitForIsStableAndProportional(t *testing.T) {
	t.Parallel()

	exportID := pulid.MustNew("aitx_")
	validation := 0
	const total = 4000
	for range total {
		correctionID := pulid.MustNew("aicr_")
		split := SplitFor(exportID, correctionID, 10)
		assert.Equal(t, split, SplitFor(exportID, correctionID, 10))
		if split == SplitValidation {
			validation++
		}
	}
	assert.InDelta(t, 0.10, float64(validation)/total, 0.03)
	assert.Equal(t, SplitTrain, SplitFor(exportID, pulid.MustNew("aicr_"), 0))
}

func TestNewManifestCarriesNoTenant(t *testing.T) {
	t.Parallel()

	export := validExport()
	export.ID = pulid.MustNew("aitx_")
	export.RecordProgress(OrganizationProgress{
		Ordinal:  1,
		Examples: 4,
		Dropped:  map[DropReason]int{DropNoDocumentText: 2},
		Parts:    []Part{{Ordinal: 1, Split: SplitTrain, Key: export.PartKey(1, SplitTrain), Examples: 4}},
	})

	manifest := NewManifest(export, 1_710_000_100)
	assert.Equal(t, ManifestFormat, manifest.Format)
	assert.Equal(t, 4, manifest.Examples.Total)
	assert.Equal(t, 2, manifest.Dropped[DropNoDocumentText])
	assert.Equal(t, export.PartKey(1, SplitTrain), manifest.Parts[0].Key)
}
