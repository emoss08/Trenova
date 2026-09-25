package aicorrectionservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.AICorrectionService = (*Service)(nil)

type correctionStore interface {
	Upsert(ctx context.Context, entity *aicorrection.Correction) (*aicorrection.Correction, error)
	PurgeBefore(ctx context.Context, req repositories.PurgeAICorrectionsRequest) (int64, error)
}

type shipmentReader interface {
	GetByID(ctx context.Context, req *repositories.GetShipmentByIDRequest) (*shipment.Shipment, error)
}

type contentReader interface {
	GetByDocumentID(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentcontent.Content, error)
}

type extractionReader interface {
	GetByDocumentExtractedAt(
		ctx context.Context,
		req repositories.GetDocumentAIExtractionRequest,
	) (*documentaiextraction.Extraction, error)
}

type retentionReader interface {
	Get(ctx context.Context, req repositories.GetDataRetentionRequest) (*tenant.DataRetention, error)
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Repo        repositories.AICorrectionRepository
	Shipments   repositories.ShipmentRepository
	Contents    repositories.DocumentContentRepository
	Extractions repositories.DocumentAIExtractionRepository
	Retention   repositories.DataRetentionRepository
}

type Service struct {
	l           *zap.Logger
	repo        correctionStore
	shipments   shipmentReader
	contents    contentReader
	extractions extractionReader
	retention   retentionReader
	now         func() int64
}

func New(p Params) services.AICorrectionService {
	return &Service{
		l:           p.Logger.Named("service.aicorrection"),
		repo:        p.Repo,
		shipments:   p.Shipments,
		contents:    p.Contents,
		extractions: p.Extractions,
		retention:   p.Retention,
		now:         timeutils.NowUnix,
	}
}

func (s *Service) CaptureShipmentDraft(
	ctx context.Context,
	req *services.CaptureShipmentDraftCorrectionRequest,
) (*aicorrection.Correction, error) {
	if req == nil || req.Draft == nil {
		return nil, errortypes.NewValidationError("draft", errortypes.ErrRequired, "Draft is required")
	}
	if req.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId", errortypes.ErrRequired, "Shipment is required",
		)
	}
	if req.Draft.OrganizationID != req.TenantInfo.OrgID ||
		req.Draft.BusinessUnitID != req.TenantInfo.BuID {
		return nil, errortypes.NewValidationError(
			"draft", errortypes.ErrInvalid, "Draft does not belong to this organization",
		)
	}

	predicted := readPrediction(req.Draft.DraftData)
	if predicted.empty() {
		return nil, aicorrection.ErrNothingPredicted
	}

	shp, err := s.shipments.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              req.ShipmentID,
		TenantInfo:      req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, fmt.Errorf("load confirmed shipment: %w", err)
	}

	confirmed := readConfirmation(shp)
	documentID := req.Draft.DocumentID
	entity := &aicorrection.Correction{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Task:           aicorrection.TaskShipmentDraftExtraction,
		SourceType:     aicorrection.SourceDocumentShipmentDraft,
		SourceID:       req.Draft.ID,
		DocumentID:     &documentID,
		SubjectType:    aicorrection.SubjectShipment,
		SubjectID:      shp.ID,
		CapturedByID:   req.CapturedByID,
		DocumentKind: stringutils.TruncateRunes(
			stringutils.FirstNonEmpty(req.Draft.DocumentKind, predicted.kind),
			aicorrection.MaxDocumentKindLength,
		),
		DocumentFingerprint: stringutils.TruncateRunes(predicted.issuer, aicorrection.MaxFingerprintLength),
		PredictedConfidence: predictedConfidence(req.Draft.Confidence, predicted.confidence),
		Predicted:           predicted.snapshot,
		Confirmed:           confirmed.snapshot,
		FieldResults:        compareSnapshots(predicted, confirmed),
		CapturedAt:          s.now(),
	}
	s.attachExtractionModel(ctx, entity, documentID, req.TenantInfo)
	entity.ApplyTally()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.repo.Upsert(ctx, entity)
}

func (s *Service) attachExtractionModel(
	ctx context.Context,
	entity *aicorrection.Correction,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) {
	content, err := s.contents.GetByDocumentID(ctx, documentID, tenantInfo)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to read document content for ai correction",
				zap.String("documentId", documentID.String()),
				zap.Error(err),
			)
		}
		return
	}
	if content.LastExtractedAt == nil {
		return
	}

	extraction, err := s.extractions.GetByDocumentExtractedAt(
		ctx,
		repositories.GetDocumentAIExtractionRequest{
			DocumentID:  documentID,
			ExtractedAt: *content.LastExtractedAt,
			TenantInfo:  tenantInfo,
		},
	)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to read ai extraction for ai correction",
				zap.String("documentId", documentID.String()),
				zap.Error(err),
			)
		}
		return
	}
	if extraction.Status != documentaiextraction.StatusCompleted &&
		extraction.Status != documentaiextraction.StatusApplied {
		return
	}

	entity.ExtractionModel = stringutils.TruncateRunes(extraction.Model, aicorrection.MaxModelLength)
	if extraction.ProviderID.IsNotNil() {
		providerID := extraction.ProviderID
		entity.ExtractionProviderID = &providerID
	}
}

func predictedConfidence(draft, data float64) float64 {
	if draft > 0 {
		return floatutils.Clamp(draft, 0, 1)
	}

	return floatutils.Clamp(data, 0, 1)
}
