package ediconnectiondraft

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type setupService interface {
	CreatePartner(
		ctx context.Context,
		entity *edi.EDIPartner,
		actor *permissionports.RequestActor,
	) (*edi.EDIPartner, error)
	UpdatePartner(
		ctx context.Context,
		entity *edi.EDIPartner,
		actor *permissionports.RequestActor,
	) (*edi.EDIPartner, error)
	CreateCommunicationProfile(
		ctx context.Context,
		req *edisetup.UpsertEDICommunicationProfileRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDICommunicationProfile, error)
	UpdateCommunicationProfile(
		ctx context.Context,
		req *edisetup.UpsertEDICommunicationProfileRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDICommunicationProfile, error)
	SaveMappingProfile(
		ctx context.Context,
		req *edipartnerports.SaveMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	StartControlNumber(
		ctx context.Context,
		req *edipartnerports.ResetEDIControlNumberRequest,
		actor *permissionports.RequestActor,
	) (*edidocument.EDIControlNumberSequence, error)
}

type projectStore interface {
	Import(
		ctx context.Context,
		req *edilading.ImportRequest,
		actor *permissionports.RequestActor,
	) (*ladingproject.LadingVersion, error)
	Samples(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		version *ladingproject.LadingVersion,
	) ([]lading.Sample, error)
}

type Params struct {
	fx.In

	Repo          ediconnectiondraftports.EDIConnectionDraftRepository
	DocumentTypes edipartnerports.EDIDocumentTypeRepository
	Setup         *edisetup.Service
	Validator     *edivalidation.Validator
	Lading        *edilading.Runtime
	Projects      *edilading.Projects
	DB            kernelports.Transactor
	Logger        *zap.Logger
}

type Service struct {
	repo          ediconnectiondraftports.EDIConnectionDraftRepository
	documentTypes edipartnerports.EDIDocumentTypeRepository
	setup         setupService
	validator     *edivalidation.Validator
	lading        *edilading.Runtime
	projects      projectStore
	db            kernelports.Transactor
	l             *zap.Logger
}

var _ ediconnectiondraftports.EDIConnectionDraftService = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		repo:          p.Repo,
		documentTypes: p.DocumentTypes,
		setup:         p.Setup,
		validator:     p.Validator,
		lading:        p.Lading,
		projects:      p.Projects,
		db:            p.DB,
		l:             p.Logger.Named("service.edi-connection-draft"),
	}
}
