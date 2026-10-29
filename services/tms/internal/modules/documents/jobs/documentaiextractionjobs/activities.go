package documentaiextractionjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionsupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Logger            *zap.Logger
	AIConfig          *aiconfig.Config
	DocumentRepo      documentports.DocumentRepository
	ContentRepo       documentcontentports.DocumentContentRepository
	AIExtractionRepo  documentaiextractionports.DocumentAIExtractionRepository
	AIDocumentService documentaiextractionports.AIDocumentService
	Support           *documentextractionsupport.Service
	ShadowSampler     extractionshadowports.ExtractionShadowSampler  `optional:"true"`
	Rollout           extractionrolloutports.ExtractionRolloutRouter `optional:"true"`
}

type Activities struct {
	logger            *zap.Logger
	cfg               *aiconfig.Config
	documentRepo      documentports.DocumentRepository
	contentRepo       documentcontentports.DocumentContentRepository
	aiExtractionRepo  documentaiextractionports.DocumentAIExtractionRepository
	aiDocumentService documentaiextractionports.AIDocumentService
	support           *documentextractionsupport.Service
	shadowSampler     extractionshadowports.ExtractionShadowSampler
	rollout           extractionrolloutports.ExtractionRolloutRouter
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewActivities(p ActivitiesParams) *Activities {
	aiDocumentService := p.AIDocumentService
	if aiDocumentService == nil {
		aiDocumentService = documentextractionsupport.NoopAIDocumentService{}
	}

	return &Activities{
		logger:            p.Logger.Named("temporal.document-extraction"),
		cfg:               p.AIConfig,
		documentRepo:      p.DocumentRepo,
		contentRepo:       p.ContentRepo,
		aiExtractionRepo:  p.AIExtractionRepo,
		aiDocumentService: aiDocumentService,
		support:           p.Support,
		shadowSampler:     p.ShadowSampler,
		rollout:           p.Rollout,
	}
}
