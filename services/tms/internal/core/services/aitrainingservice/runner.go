package aitrainingservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.AITrainingExportRunner = (*Runner)(nil)

const (
	correctionPageSize   = 100
	exampleIDBytes       = 16
	manifestContentType  = "application/json"
	errExampleWriteLabel = "write training example"
)

type RunnerParams struct {
	fx.In

	Logger        *zap.Logger
	Exports       repositories.AITrainingExportRepository
	Records       repositories.AITrainingRecordRepository
	Corrections   repositories.AICorrectionRepository
	Contents      repositories.DocumentContentRepository
	Documents     repositories.DocumentRepository
	Organizations repositories.OrganizationRepository
	Storage       storage.Client
}

type Runner struct {
	l             *zap.Logger
	exports       repositories.AITrainingExportRepository
	records       repositories.AITrainingRecordRepository
	corrections   repositories.AICorrectionRepository
	contents      repositories.DocumentContentRepository
	documents     repositories.DocumentRepository
	organizations repositories.OrganizationRepository
	storage       storage.Client
	now           func() int64
	seed          func() ([32]byte, error)
	exampleID     func() (string, error)
}

func NewRunner(p RunnerParams) *Runner {
	return &Runner{
		l:             p.Logger.Named("service.aitraining-runner"),
		exports:       p.Exports,
		records:       p.Records,
		corrections:   p.Corrections,
		contents:      p.Contents,
		documents:     p.Documents,
		organizations: p.Organizations,
		storage:       p.Storage,
		now:           timeutils.NowUnix,
		seed:          tokenutils.RandomSeed,
		exampleID:     func() (string, error) { return tokenutils.RandomHex(exampleIDBytes) },
	}
}

func AsRunner(r *Runner) services.AITrainingExportRunner { return r }

func (r *Runner) Begin(ctx context.Context, exportID pulid.ID) (*aitraining.TrainingExport, error) {
	entity, err := r.exports.GetByID(ctx, exportID)
	if err != nil {
		return nil, err
	}
	switch entity.Status {
	case aitraining.ExportStatusRunning:
		return entity, nil
	case aitraining.ExportStatusQueued:
		started := r.now()
		entity.Status = aitraining.ExportStatusRunning
		entity.StartedAt = &started
		return r.exports.Update(ctx, entity)
	default:
		return nil, aitraining.ErrExportInactive
	}
}

func (r *Runner) IsActive(ctx context.Context, exportID pulid.ID) (bool, error) {
	entity, err := r.exports.GetByID(ctx, exportID)
	if err != nil {
		return false, err
	}

	return entity.Status.IsActive(), nil
}

func (r *Runner) ListOrganizations(
	ctx context.Context,
	req repositories.ListConsentingOrganizationsRequest,
) ([]repositories.TrainingConsent, error) {
	return r.exports.ListConsentingOrganizations(ctx, req)
}

func (r *Runner) ExportOrganization(
	ctx context.Context,
	req *services.ExportTrainingOrganizationRequest,
) (*aitraining.OrganizationProgress, error) {
	export, err := r.exports.GetByID(ctx, req.ExportID)
	if err != nil {
		return nil, err
	}
	if !export.Status.IsActive() {
		return nil, aitraining.ErrExportInactive
	}

	tenantInfo := req.Consent.TenantInfo()
	result := &aitraining.OrganizationProgress{
		Ordinal: req.Ordinal,
		Dropped: map[aitraining.DropReason]int{},
		Parts:   []aitraining.Part{},
	}
	consent, err := r.exports.GetConsent(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !consent.Granted {
		result.ConsentWithdrawn = true
		if err = r.records.DeleteForOrganization(ctx, export.ID, tenantInfo); err != nil {
			return nil, err
		}
		return r.recordProgress(ctx, export.ID, result)
	}

	identity, err := r.identity(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	parts := newPartSet(ctx, r.storage, export, req.Ordinal, r.l)
	written, err := r.writeExamples(ctx, &organizationExport{
		export:    export,
		tenant:    tenantInfo,
		consent:   consent,
		identity:  identity,
		parts:     parts,
		dropped:   result.Dropped,
		heartbeat: req.Heartbeat,
	})
	if err != nil {
		parts.discard(context.WithoutCancel(ctx), err)
		return nil, err
	}

	closed, err := parts.close()
	if err != nil {
		parts.discard(context.WithoutCancel(ctx), err)
		return nil, err
	}

	after, err := r.exports.GetConsent(ctx, tenantInfo)
	if err != nil {
		parts.discard(context.WithoutCancel(ctx), err)
		return nil, err
	}
	if !after.Granted || after.GrantedAt != consent.GrantedAt {
		parts.discard(context.WithoutCancel(ctx), aitraining.ErrExportInactive)
		result.ConsentWithdrawn = true
		result.Dropped[aitraining.DropConsentWithdrawn] += len(written)
		if err = r.records.DeleteForOrganization(ctx, export.ID, tenantInfo); err != nil {
			return nil, err
		}
		return r.recordProgress(ctx, export.ID, result)
	}

	if err = r.records.ReplaceForOrganization(ctx, &repositories.ReplaceAITrainingRecordsRequest{
		ExportID:   export.ID,
		TenantInfo: tenantInfo,
		Records:    written,
	}); err != nil {
		parts.discard(context.WithoutCancel(ctx), err)
		return nil, err
	}

	result.Parts = closed
	result.Examples = len(written)

	return r.recordProgress(ctx, export.ID, result)
}

func (r *Runner) recordProgress(
	ctx context.Context,
	exportID pulid.ID,
	progress *aitraining.OrganizationProgress,
) (*aitraining.OrganizationProgress, error) {
	entity, err := r.exports.GetByID(ctx, exportID)
	if err != nil {
		return nil, err
	}
	entity.RecordProgress(*progress)
	if _, err = r.exports.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("record training export progress: %w", err)
	}

	return progress, nil
}

type organizationExport struct {
	export    *aitraining.TrainingExport
	tenant    pagination.TenantInfo
	consent   repositories.TrainingConsent
	identity  *aitraining.Identity
	parts     *partSet
	dropped   map[aitraining.DropReason]int
	heartbeat func(ctx context.Context, details ...any)
}

func (r *Runner) writeExamples(
	ctx context.Context,
	job *organizationExport,
) ([]*aitraining.TrainingExportRecord, error) {
	limit := job.export.MaxPerOrganization
	written := make([]*aitraining.TrainingExportRecord, 0, min(limit, correctionPageSize))
	var afterCapturedAt int64
	var afterID pulid.ID

	for len(written) < limit {
		active, err := r.IsActive(ctx, job.export.ID)
		if err != nil {
			return nil, err
		}
		if !active {
			return nil, aitraining.ErrExportInactive
		}

		page, err := r.corrections.ListForTraining(ctx, &repositories.ListAICorrectionsForTrainingRequest{
			TenantInfo:      job.tenant,
			Task:            job.export.Task,
			CapturedFrom:    job.export.CapturedFrom,
			CapturedTo:      job.export.CapturedTo,
			AfterCapturedAt: afterCapturedAt,
			AfterID:         afterID,
			Limit:           correctionPageSize,
		})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		for _, correction := range page {
			afterCapturedAt, afterID = correction.CapturedAt, correction.ID
			example, reason, exampleErr := r.example(ctx, job, correction)
			if exampleErr != nil {
				return nil, exampleErr
			}
			if reason != "" {
				job.dropped[reason]++
			} else {
				record, writeErr := r.writeExample(job, correction, example)
				if writeErr != nil {
					return nil, writeErr
				}
				written = append(written, record)
			}
			if job.heartbeat != nil {
				job.heartbeat(ctx, len(written))
			}
			if len(written) >= limit {
				break
			}
		}
		if len(page) < correctionPageSize {
			break
		}
	}

	return written, nil
}

func (r *Runner) writeExample(
	job *organizationExport,
	correction *aicorrection.Correction,
	example *aitraining.Example,
) (*aitraining.TrainingExportRecord, error) {
	line, err := sonic.Marshal(example)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errExampleWriteLabel, err)
	}
	if err = job.parts.write(example.Split, line); err != nil {
		return nil, err
	}

	return &aitraining.TrainingExportRecord{
		ExportID:         job.export.ID,
		OrganizationID:   job.tenant.OrgID,
		BusinessUnitID:   job.tenant.BuID,
		CorrectionID:     correction.ID,
		ExampleID:        example.ID,
		Split:            example.Split,
		ConsentGrantedAt: job.consent.GrantedAt,
		ExportedAt:       r.now(),
	}, nil
}

func (r *Runner) example(
	ctx context.Context,
	job *organizationExport,
	correction *aicorrection.Correction,
) (*aitraining.Example, aitraining.DropReason, error) {
	if correction.DocumentID == nil || correction.DocumentID.IsNil() {
		return nil, aitraining.DropDocumentUnreadable, nil
	}
	if confirmedEmpty(correction.Confirmed) {
		return nil, aitraining.DropNothingConfirmed, nil
	}

	doc, err := r.documents.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         *correction.DocumentID,
		TenantInfo: job.tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, aitraining.DropDocumentUnreadable, nil
		}
		return nil, "", fmt.Errorf("read training document: %w", err)
	}

	stored, err := r.contents.ListPagesByDocumentID(ctx, doc.ID, job.tenant)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, aitraining.DropNoDocumentText, nil
		}
		return nil, "", fmt.Errorf("read training document pages: %w", err)
	}
	pages := examplePages(stored)
	if len(pages) == 0 {
		return nil, aitraining.DropNoDocumentText, nil
	}

	seed, err := r.seed()
	if err != nil {
		return nil, "", err
	}
	anonymized, err := aitraining.Anonymize(&aitraining.Source{
		FileName:   doc.OriginalName,
		Pages:      pages,
		Target:     correction.Confirmed,
		Prediction: correction.Predicted,
		Issuer:     correction.DocumentFingerprint,
	}, job.identity, rand.New(rand.NewChaCha8(seed)))
	if err != nil {
		if errors.Is(err, aitraining.ErrResidualIdentifier) {
			return nil, aitraining.DropResidualIdentifier, nil
		}
		return nil, "", fmt.Errorf("anonymize training example: %w", err)
	}

	id, err := r.exampleID()
	if err != nil {
		return nil, "", err
	}

	return &aitraining.Example{
		ID:           id,
		Format:       job.export.Format,
		Task:         correction.Task,
		Split:        aitraining.SplitFor(job.export.ID, correction.ID, job.export.ValidationPercent),
		DocumentKind: correction.DocumentKind,
		Input: aitraining.ExampleInput{
			FileName: anonymized.FileName,
			Pages:    anonymized.Pages,
		},
		Target:     anonymized.Target,
		Prediction: anonymized.Prediction,
		Outcomes:   aitraining.OutcomesOf(correction.FieldResults),
		Quality:    aitraining.QualityOf(correction),
	}, "", nil
}

func (r *Runner) identity(ctx context.Context, tenantInfo pagination.TenantInfo) (*aitraining.Identity, error) {
	org, err := r.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
		IncludeBU:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("read organization identity: %w", err)
	}
	people, err := r.exports.ListOrganizationPeople(ctx, tenantInfo, 0)
	if err != nil {
		return nil, err
	}

	parties := []string{org.Name}
	if org.BusinessUnit != nil {
		parties = append(parties, org.BusinessUnit.Name)
	}

	return aitraining.NewIdentity(&aitraining.IdentityValues{
		Parties:     parties,
		People:      people,
		Identifiers: []string{org.ScacCode, org.DOTNumber, org.TaxID, org.LoginSlug},
		Streets:     []string{org.AddressLine1, org.AddressLine2},
		Cities:      []string{org.City},
		PostalCodes: []string{org.PostalCode},
	}), nil
}

func (r *Runner) Finish(
	ctx context.Context,
	req *services.FinishAITrainingExportRequest,
) (*aitraining.TrainingExport, error) {
	entity, err := r.exports.GetByID(ctx, req.ExportID)
	if err != nil {
		return nil, err
	}
	if entity.ManifestKey != "" && !entity.Status.IsActive() {
		return entity, nil
	}

	finished := r.now()
	if entity.Status.IsActive() {
		entity.Status = aitraining.ExportStatusCompleted
	}
	if entity.FinishedAt == nil {
		entity.FinishedAt = &finished
	}
	entity.RecomputeProgress()

	manifest, err := sonic.Marshal(aitraining.NewManifest(entity, *entity.FinishedAt))
	if err != nil {
		return nil, fmt.Errorf("encode training manifest: %w", err)
	}
	key := entity.ManifestObjectKey()
	if _, err = r.storage.Upload(ctx, &storage.UploadParams{
		Key:         key,
		ContentType: manifestContentType,
		Size:        int64(len(manifest)),
		Body:        bytes.NewReader(manifest),
		Metadata:    map[string]string{metadataExport: entity.ID.String()},
	}); err != nil {
		return nil, fmt.Errorf("upload training manifest: %w", err)
	}
	entity.ManifestKey = key
	entity.ManifestSHA256 = hashutils.SHA256BytesHex(manifest)

	updated, err := r.exports.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	r.l.Info("training export finished",
		zap.String("exportId", updated.ID.String()),
		zap.String("status", updated.Status.String()),
		zap.Int("examples", updated.ExamplesTotal),
		zap.Int("organizations", updated.OrganizationsIncluded),
	)

	return updated, nil
}

func (r *Runner) Fail(ctx context.Context, exportID pulid.ID, message string) error {
	entity, err := r.exports.GetByID(ctx, exportID)
	if err != nil {
		return err
	}
	if !entity.Status.IsActive() {
		return nil
	}

	finished := r.now()
	entity.Status = aitraining.ExportStatusFailed
	entity.FailureMessage = stringutils.TruncateRunes(message, aitraining.MaxFailureRunes)
	entity.FinishedAt = &finished
	_, err = r.exports.Update(ctx, entity)

	return err
}

func examplePages(stored []*documentcontent.Page) []aitraining.ExamplePage {
	pages := make([]aitraining.ExamplePage, 0, min(len(stored), aitraining.MaxExamplePages))
	for _, page := range stored {
		if len(pages) >= aitraining.MaxExamplePages {
			break
		}
		if page == nil || strings.TrimSpace(page.ExtractedText) == "" {
			continue
		}
		pages = append(pages, aitraining.ExamplePage{
			Number: page.PageNumber,
			Text:   stringutils.TruncateRunes(page.ExtractedText, aitraining.MaxExamplePageRunes),
		})
	}

	return pages
}

func confirmedEmpty(snapshot *aicorrection.Snapshot) bool {
	return snapshot == nil || (len(snapshot.Fields) == 0 && len(snapshot.Stops) == 0)
}
