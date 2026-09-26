package aitrainingservice

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/aidocumentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memorySink struct {
	mu    sync.Mutex
	files map[string]*bytes.Buffer
}

type memoryFile struct{ *bytes.Buffer }

func (memoryFile) Close() error { return nil }

func (s *memorySink) Create(name string) (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.files == nil {
		s.files = map[string]*bytes.Buffer{}
	}
	buffer := &bytes.Buffer{}
	s.files[name] = buffer
	return memoryFile{buffer}, nil
}

func (s *memorySink) lines(name string) [][]byte {
	content := bytes.TrimSpace(s.files[name].Bytes())
	if len(content) == 0 {
		return nil
	}
	return bytes.Split(content, []byte("\n"))
}

type echoPrompts struct{}

func (echoPrompts) RenderStructuredPrompt(
	req *services.StructuredCompletionRequest,
	mode aiprovider.StructuredOutputMode,
) services.RenderedPrompt {
	var user strings.Builder
	for _, section := range req.Context.Sections {
		user.WriteString(section.Title + "\n" + section.Content + "\n")
	}
	temperature := 0.1
	return services.RenderedPrompt{System: req.System + "|" + string(mode), User: user.String(), Temperature: &temperature}
}

func correctedSnapshot() *aicorrection.Snapshot {
	snapshot := confirmedSnapshot()
	snapshot.Fields[aicorrection.FieldRate] = "2760.50"
	return snapshot
}

func exportedFixture(t *testing.T, corrections int, validationPercent int) *runnerFixture {
	t.Helper()

	f := newRunnerFixture()
	f.export.ValidationPercent = validationPercent
	for range corrections {
		correction := f.addCorrection(documentText, confirmedSnapshot())
		prediction := confirmedSnapshot()
		prediction.Fields[aicorrection.FieldShipper] = "Wrong Shipper"
		prediction.Fields["loadNumber"] = "AFB-7731902"
		correction.Predicted = prediction
		correction.FieldResults = append(correction.FieldResults, aicorrection.FieldResult{
			Key: aicorrection.FieldShipper, Outcome: aicorrection.OutcomeCorrected,
		})
	}
	_, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)
	_, err = f.runner.Finish(t.Context(), &services.FinishAITrainingExportRequest{ExportID: f.export.ID})
	require.NoError(t, err)

	return f
}

func newTestRenderer(f *runnerFixture) *Renderer {
	return &Renderer{
		l:        zap.NewNop(),
		exports:  f.exports,
		records:  f.records,
		storage:  f.objects,
		contract: aidocumentservice.Contract{},
		prompts:  echoPrompts{},
		now:      func() int64 { return testNow },
	}
}

func TestRenderWritesRawExamples(t *testing.T) {
	t.Parallel()

	f := exportedFixture(t, 6, 50)
	sink := &memorySink{}
	dataset, err := newTestRenderer(f).Render(t.Context(), &services.RenderTrainingDatasetRequest{
		ExportID: f.export.ID,
		Sink:     sink,
	})
	require.NoError(t, err)

	contract := aidocumentservice.Contract{}
	assert.Equal(t, aitraining.DatasetFormat, dataset.Format)
	assert.Equal(t, aiprovider.StructuredOutputJSONSchema, dataset.StructuredOutputMode)
	assert.Equal(t, contract.PageLimit(), dataset.PageLimit)
	assert.Equal(t, contract.FieldKeys(), dataset.FieldKeys)
	assert.Equal(t, 6, dataset.Counts.Examples)
	assert.Equal(t, 6, dataset.Counts.Train+dataset.Counts.Validation)
	assert.Len(t, sink.lines(aitraining.DatasetTrainFile), dataset.Counts.Train)
	assert.Len(t, sink.lines(aitraining.DatasetValidationFile), dataset.Counts.Validation)
	assert.Len(t, sink.lines(aitraining.DatasetEvaluationFile), dataset.Counts.Validation)
	assert.Len(t, dataset.PromptSHA256, 64)
	require.NotNil(t, dataset.Temperature)

	for _, file := range dataset.Files {
		content := sink.files[file.Name].Bytes()
		assert.Equal(t, hashutils.SHA256BytesHex(content), file.SHA256, file.Name)
		assert.Equal(t, int64(len(content)), file.Bytes, file.Name)
	}
	var manifest aitraining.DatasetManifest
	require.NoError(t, sonic.Unmarshal(sink.files[aitraining.DatasetManifestFile].Bytes(), &manifest))
	assert.Equal(t, f.export.ID, manifest.ExportID)
	assert.Contains(t, sink.files[aitraining.DatasetSchemaFile].String(), `"fields"`)

	lines := append(sink.lines(aitraining.DatasetTrainFile), sink.lines(aitraining.DatasetValidationFile)...)
	for _, line := range lines {
		var record aitraining.TrainingRecord
		require.NoError(t, sonic.Unmarshal(line, &record))
		require.Len(t, record.Prompt, 2)
		assert.Equal(t, aitraining.RoleSystem, record.Prompt[0].Role)
		assert.True(t, strings.HasSuffix(record.Prompt[0].Content, "|JSONSchema"))
		require.Len(t, record.VisiblePages, 1)
		assert.Contains(t, record.Prompt[1].Content, record.VisiblePages[0].Text)

		reference := record.Target.Fields[aicorrection.FieldReference]
		assert.NotEmpty(t, reference)
		assert.Contains(t, record.VisiblePages[0].Text, reference)
		assert.NotEqual(t,
			record.Target.Fields[aicorrection.FieldShipper],
			record.Prediction.Fields[aicorrection.FieldShipper],
		)
		assert.Equal(t, aicorrection.OutcomeCorrected, record.Outcomes[aicorrection.FieldShipper])
		assert.Equal(t, "RateConfirmation", record.DocumentKind)
	}

	for _, line := range sink.lines(aitraining.DatasetEvaluationFile) {
		var record aitraining.EvaluationRecord
		require.NoError(t, sonic.Unmarshal(line, &record))
		assert.NotEmpty(t, record.Expected.Fields[aicorrection.FieldReference])
		assert.NotEqual(t,
			record.Expected.Fields[aicorrection.FieldShipper],
			record.Baseline.Fields[aicorrection.FieldShipper],
		)
	}
}

func TestVisiblePagesCutWhereProductionCuts(t *testing.T) {
	t.Parallel()

	pages := visiblePages([]aitraining.ExamplePage{
		{Number: 1, Text: "short"},
		{Number: 2, Text: "abcdé"},
	}, 5)

	assert.Equal(t, "short", pages[0].Text)
	assert.Equal(t, "abcd", pages[1].Text)
	assert.Equal(t, 2, pages[1].Number)
}

func TestRenderExcludesWithdrawnExamples(t *testing.T) {
	t.Parallel()

	f := exportedFixture(t, 3, 0)
	withdrawn := f.records.records[f.export.ID][1].ExampleID
	f.records.withdrawn[withdrawn] = struct{}{}

	sink := &memorySink{}
	dataset, err := newTestRenderer(f).Render(t.Context(), &services.RenderTrainingDatasetRequest{
		ExportID: f.export.ID,
		Sink:     sink,
	})
	require.NoError(t, err)

	assert.Equal(t, 1, dataset.Counts.WithdrawnExcluded)
	assert.Equal(t, 2, dataset.Counts.Examples)
	assert.NotContains(t, sink.files[aitraining.DatasetTrainFile].String(), `"id":"`+withdrawn+`"`)
}

func TestRenderRejectsATamperedPart(t *testing.T) {
	t.Parallel()

	f := exportedFixture(t, 2, 0)
	key := f.export.PartKey(1, aitraining.SplitTrain)
	f.objects.objects[key] = bytes.Replace(f.objects.objects[key], []byte(`"task"`), []byte(`"task" `), 1)

	_, err := newTestRenderer(f).Render(t.Context(), &services.RenderTrainingDatasetRequest{
		ExportID: f.export.ID,
		Sink:     &memorySink{},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestRenderRejectsATamperedManifest(t *testing.T) {
	t.Parallel()

	f := exportedFixture(t, 1, 0)
	stored := f.exports.items[f.export.ID]
	f.objects.objects[stored.ManifestKey] = append(f.objects.objects[stored.ManifestKey], ' ')

	_, err := newTestRenderer(f).Render(t.Context(), &services.RenderTrainingDatasetRequest{
		ExportID: f.export.ID,
		Sink:     &memorySink{},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestRenderRequiresAFinishedExport(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	_, err := newTestRenderer(f).Render(t.Context(), &services.RenderTrainingDatasetRequest{
		ExportID: f.export.ID,
		Sink:     &memorySink{},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	_, err = newTestRenderer(f).Render(t.Context(), &services.RenderTrainingDatasetRequest{
		ExportID:             f.export.ID,
		StructuredOutputMode: aiprovider.StructuredOutputMode("Telepathy"),
		Sink:                 &memorySink{},
	})
	require.Error(t, err)
}
