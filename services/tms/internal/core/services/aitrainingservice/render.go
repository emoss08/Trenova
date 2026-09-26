package aitrainingservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.AITrainingDatasetRenderer = (*Renderer)(nil)

const (
	maxManifestBytes  = 32 << 20
	withdrawnPageSize = 1000
)

type RendererParams struct {
	fx.In

	Logger   *zap.Logger
	Exports  repositories.AITrainingExportRepository
	Records  repositories.AITrainingRecordRepository
	Storage  storage.Client
	Contract services.ExtractionContract
	Prompts  services.StructuredPromptRenderer
}

type Renderer struct {
	l        *zap.Logger
	exports  repositories.AITrainingExportRepository
	records  repositories.AITrainingRecordRepository
	storage  storage.Client
	contract services.ExtractionContract
	prompts  services.StructuredPromptRenderer
	now      func() int64
}

func NewRenderer(p RendererParams) *Renderer {
	return &Renderer{
		l:        p.Logger.Named("service.aitraining-renderer"),
		exports:  p.Exports,
		records:  p.Records,
		storage:  p.Storage,
		contract: p.Contract,
		prompts:  p.Prompts,
		now:      timeutils.NowUnix,
	}
}

func AsRenderer(r *Renderer) services.AITrainingDatasetRenderer { return r }

type renderJob struct {
	mode      aiprovider.StructuredOutputMode
	pageLimit int
	withdrawn map[string]struct{}
	files     *datasetFiles
	counts    aitraining.DatasetCounts
}

func (r *Renderer) Render(
	ctx context.Context,
	req *services.RenderTrainingDatasetRequest,
) (*aitraining.DatasetManifest, error) {
	if req == nil || req.Sink == nil {
		return nil, errortypes.NewValidationError("sink", errortypes.ErrRequired, "An output is required")
	}
	mode := req.StructuredOutputMode
	if mode == "" {
		mode = aiprovider.StructuredOutputJSONSchema
	}
	if !mode.IsValid() {
		return nil, errortypes.NewValidationError(
			"structuredOutputMode", errortypes.ErrInvalid, "Structured output mode is invalid",
		)
	}

	export, err := r.exports.GetByID(ctx, req.ExportID)
	if err != nil {
		return nil, err
	}
	if export.Status.IsActive() || export.ManifestKey == "" {
		return nil, errortypes.NewBusinessError(
			"This export has not finished; render it once its manifest is written",
		)
	}

	manifest, err := r.readManifest(ctx, export)
	if err != nil {
		return nil, err
	}
	withdrawn, err := r.withdrawnExamples(ctx, export.ID)
	if err != nil {
		return nil, err
	}

	files, err := openDatasetFiles(req.Sink,
		aitraining.DatasetTrainFile,
		aitraining.DatasetValidationFile,
		aitraining.DatasetEvaluationFile,
	)
	if err != nil {
		return nil, err
	}
	job := &renderJob{
		mode:      mode,
		pageLimit: r.contract.PageLimit(),
		withdrawn: withdrawn,
		files:     files,
	}

	for i := range manifest.Parts {
		if err = r.renderPart(ctx, job, &manifest.Parts[i]); err != nil {
			files.abandon()
			return nil, err
		}
	}

	written, err := files.close()
	if err != nil {
		return nil, err
	}

	template := r.contract.CompletionRequest("", nil)
	prompt := r.prompts.RenderStructuredPrompt(template, mode)
	schemaFile, err := r.writeDocument(req.Sink, aitraining.DatasetSchemaFile, template.OutputSchema)
	if err != nil {
		return nil, err
	}
	schemaJSON, err := sonic.Marshal(template.OutputSchema)
	if err != nil {
		return nil, fmt.Errorf("encode output schema: %w", err)
	}

	dataset := &aitraining.DatasetManifest{
		Format:               aitraining.DatasetFormat,
		ExportID:             export.ID,
		ExportManifestSHA256: export.ManifestSHA256,
		ExampleFormat:        manifest.ExampleFormat,
		Task:                 export.Task,
		StructuredOutputMode: mode,
		SchemaName:           template.SchemaName,
		PromptSHA256:         hashutils.SHA256Hex(prompt.System + "\n" + string(schemaJSON)),
		Temperature:          prompt.Temperature,
		TopP:                 prompt.TopP,
		PageLimit:            job.pageLimit,
		FieldKeys:            r.contract.FieldKeys(),
		Counts:               job.counts,
		Files:                append(written, schemaFile),
		RenderedAt:           r.now(),
	}
	if _, err = r.writeDocument(req.Sink, aitraining.DatasetManifestFile, dataset); err != nil {
		return nil, err
	}

	r.l.Info("training dataset rendered",
		zap.String("exportId", export.ID.String()),
		zap.Int("examples", dataset.Counts.Examples),
		zap.Int("withdrawnExcluded", dataset.Counts.WithdrawnExcluded),
	)

	return dataset, nil
}

func (r *Renderer) readManifest(
	ctx context.Context,
	export *aitraining.TrainingExport,
) (*aitraining.Manifest, error) {
	result, err := r.storage.Download(ctx, export.ManifestKey)
	if err != nil {
		return nil, fmt.Errorf("download training manifest: %w", err)
	}
	defer result.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(result.Body, maxManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read training manifest: %w", err)
	}
	if len(raw) > maxManifestBytes {
		return nil, errortypes.NewBusinessError("The export's manifest is larger than any manifest Trenova writes")
	}
	if hashutils.SHA256BytesHex(raw) != export.ManifestSHA256 {
		return nil, errortypes.NewBusinessError(
			"The export's manifest does not match its recorded checksum; it has been changed since it was written",
		)
	}

	manifest := new(aitraining.Manifest)
	if err = sonic.Unmarshal(raw, manifest); err != nil {
		return nil, fmt.Errorf("decode training manifest: %w", err)
	}
	if manifest.ExportID != export.ID || manifest.Format != aitraining.ManifestFormat ||
		manifest.ExampleFormat != aitraining.ExampleFormat {
		return nil, errortypes.NewBusinessError("The export's manifest is not one this version of Trenova can render")
	}

	return manifest, nil
}

func (r *Renderer) withdrawnExamples(ctx context.Context, exportID pulid.ID) (map[string]struct{}, error) {
	withdrawn := map[string]struct{}{}
	var after pulid.ID
	for {
		page, err := r.records.ListWithdrawn(ctx, repositories.ListWithdrawnTrainingExamplesRequest{
			ExportID: exportID,
			AfterID:  after,
			Limit:    withdrawnPageSize,
		})
		if err != nil {
			return nil, err
		}
		for _, example := range page {
			withdrawn[example.ExampleID] = struct{}{}
			after = example.ID
		}
		if len(page) < withdrawnPageSize {
			return withdrawn, nil
		}
	}
}

func (r *Renderer) renderPart(ctx context.Context, job *renderJob, part *aitraining.ManifestPart) error {
	result, err := r.storage.Download(ctx, part.Key)
	if err != nil {
		return fmt.Errorf("download training part %s: %w", part.Key, err)
	}
	defer result.Body.Close()

	digest := sha256.New()
	counter := &countingWriter{}
	examples := 0
	err = eachJSONLine(io.TeeReader(result.Body, io.MultiWriter(digest, counter)), func(line []byte) error {
		examples++
		return r.renderExample(job, part, line)
	})
	if err != nil {
		return fmt.Errorf("render training part %s: %w", part.Key, err)
	}

	if hex.EncodeToString(digest.Sum(nil)) != part.SHA256 || counter.bytes != part.Bytes ||
		examples != part.Examples {
		return errortypes.NewBusinessError(
			"Training part {0} does not match the manifest; it has been changed since it was written",
			part.Key,
		)
	}

	return nil
}

func (r *Renderer) renderExample(job *renderJob, part *aitraining.ManifestPart, line []byte) error {
	example := new(aitraining.Example)
	if err := sonic.Unmarshal(line, example); err != nil {
		return fmt.Errorf("decode training example in %s: %w", part.Key, err)
	}
	if example.Format != aitraining.ExampleFormat || example.Split != part.Split {
		return errortypes.NewBusinessError(
			"Training part {0} holds an example this version of Trenova cannot render",
			part.Key,
		)
	}
	if _, withdrawn := job.withdrawn[example.ID]; withdrawn {
		job.counts.WithdrawnExcluded++
		return nil
	}

	record := &aitraining.TrainingRecord{
		ID:           example.ID,
		Split:        example.Split,
		DocumentKind: example.DocumentKind,
		Prompt:       r.promptFor(job.mode, example),
		VisiblePages: visiblePages(example.Input.Pages, job.pageLimit),
		Target:       example.Target,
		Prediction:   example.Prediction,
		Outcomes:     example.Outcomes,
	}
	job.counts.Examples++

	if example.Split == aitraining.SplitValidation {
		job.counts.Validation++
		if err := job.files.write(aitraining.DatasetValidationFile, record); err != nil {
			return err
		}
		return job.files.write(aitraining.DatasetEvaluationFile, &aitraining.EvaluationRecord{
			ID:       example.ID,
			Prompt:   record.Prompt,
			Expected: example.Target,
			Baseline: example.Prediction,
		})
	}

	job.counts.Train++
	return job.files.write(aitraining.DatasetTrainFile, record)
}

func (r *Renderer) promptFor(
	mode aiprovider.StructuredOutputMode,
	example *aitraining.Example,
) []aitraining.ChatMessage {
	pages := make([]services.AIDocumentPage, 0, len(example.Input.Pages))
	for _, page := range example.Input.Pages {
		pages = append(pages, services.AIDocumentPage{PageNumber: page.Number, Text: page.Text})
	}
	rendered := r.prompts.RenderStructuredPrompt(
		r.contract.CompletionRequest(example.Input.FileName, pages),
		mode,
	)

	return []aitraining.ChatMessage{
		{Role: aitraining.RoleSystem, Content: rendered.System},
		{Role: aitraining.RoleUser, Content: rendered.User},
	}
}

func (r *Renderer) writeDocument(
	sink services.TrainingDatasetSink,
	name string,
	document any,
) (aitraining.DatasetFile, error) {
	file, err := createDatasetFile(sink, name)
	if err != nil {
		return aitraining.DatasetFile{}, err
	}
	if err = file.writeDocument(document); err != nil {
		_ = file.closer.Close()
		return aitraining.DatasetFile{}, err
	}

	return file.close()
}

func visiblePages(pages []aitraining.ExamplePage, limit int) []aitraining.ExamplePage {
	out := make([]aitraining.ExamplePage, 0, len(pages))
	for _, page := range pages {
		text := page.Text
		if len(text) > limit {
			text = strings.ToValidUTF8(text[:limit], "")
		}
		out = append(out, aitraining.ExamplePage{Number: page.Number, Text: text})
	}

	return out
}
