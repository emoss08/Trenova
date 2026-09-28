package extractionshadowservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const scoreAttempts = 3

var _ services.ExtractionShadowScorer = (*Scorer)(nil)

type ScorerParams struct {
	fx.In

	Results     repositories.ExtractionShadowResultRepository
	Corrections repositories.AICorrectionRepository
	Contents    repositories.DocumentContentRepository
}

type Scorer struct {
	results     repositories.ExtractionShadowResultRepository
	corrections repositories.AICorrectionRepository
	contents    contentReader
	now         func() int64
}

func NewScorer(p ScorerParams) *Scorer {
	return &Scorer{
		results:     p.Results,
		corrections: p.Corrections,
		contents:    p.Contents,
		now:         timeutils.NowUnix,
	}
}

func AsScorer(s *Scorer) services.ExtractionShadowScorer { return s }

func (s *Scorer) ScoreCorrection(ctx context.Context, correction *aicorrection.Correction) error {
	if correction == nil ||
		correction.Task != aicorrection.TaskShipmentDraftExtraction ||
		correction.DocumentID == nil ||
		correction.DocumentID.IsNil() {
		return nil
	}

	tenant := pagination.TenantInfo{OrgID: correction.OrganizationID, BuID: correction.BusinessUnitID}
	extractedAt, ok, err := s.currentExtraction(ctx, tenant, *correction.DocumentID)
	if err != nil || !ok {
		return err
	}

	for range scoreAttempts {
		result, getErr := s.results.GetByExtraction(ctx, repositories.GetExtractionShadowResultByExtractionRequest{
			TenantInfo:  tenant,
			DocumentID:  *correction.DocumentID,
			ExtractedAt: extractedAt,
		})
		if getErr != nil {
			if errortypes.IsNotFoundError(getErr) {
				return nil
			}
			return getErr
		}
		if result.Status != extractionshadow.ResultStatusCompleted {
			return nil
		}

		err = s.applyScore(ctx, result, correction)
		if !errortypes.IsVersionMismatchError(err) {
			return err
		}
	}

	return err
}

func (s *Scorer) scoreAgainstCorrection(
	ctx context.Context,
	result *extractionshadow.ShadowResult,
) error {
	correction, err := s.corrections.GetLatestByDocument(ctx, repositories.GetLatestAICorrectionByDocumentRequest{
		TenantInfo: tenantOf(result),
		Task:       aicorrection.TaskShipmentDraftExtraction,
		DocumentID: result.DocumentID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if correction.CapturedAt < result.ExtractedAt {
		return nil
	}

	extractedAt, ok, err := s.currentExtraction(ctx, tenantOf(result), result.DocumentID)
	if err != nil || !ok || extractedAt != result.ExtractedAt {
		return err
	}

	return s.applyScore(ctx, result, correction)
}

func (s *Scorer) applyScore(
	ctx context.Context,
	result *extractionshadow.ShadowResult,
	correction *aicorrection.Correction,
) error {
	predicted := aicorrection.ReadPrediction(result.DraftData)
	result.ApplyScore(
		correction.ID,
		aicorrection.Score(predicted, correction.Confirmed),
		correction.FieldResults,
		s.now(),
	)
	_, err := s.results.Save(ctx, result)

	return err
}

func (s *Scorer) currentExtraction(
	ctx context.Context,
	tenant pagination.TenantInfo,
	documentID pulid.ID,
) (int64, bool, error) {
	content, err := s.contents.GetByDocumentID(ctx, documentID, tenant)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if content.LastExtractedAt == nil {
		return 0, false, nil
	}

	return *content.LastExtractedAt, true, nil
}
