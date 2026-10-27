package ediconnectiondraft

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
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
		req *ediservice.UpsertEDICommunicationProfileRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDICommunicationProfile, error)
	UpdateCommunicationProfile(
		ctx context.Context,
		req *ediservice.UpsertEDICommunicationProfileRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDICommunicationProfile, error)
	SaveMappingProfile(
		ctx context.Context,
		req *ediports.SaveMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	StartControlNumber(
		ctx context.Context,
		req *ediports.ResetEDIControlNumberRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDIControlNumberSequence, error)
}

type projectStore interface {
	Import(
		ctx context.Context,
		req *edilading.ImportRequest,
		actor *permissionports.RequestActor,
	) (*edi.LadingVersion, error)
	Samples(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		version *edi.LadingVersion,
	) ([]lading.Sample, error)
}

type Params struct {
	fx.In

	Repo          ediports.EDIConnectionDraftRepository
	DocumentTypes ediports.EDIDocumentTypeRepository
	Setup         *ediservice.Service
	Validator     *ediservice.Validator
	Lading        *edilading.Runtime
	Projects      *edilading.Projects
	DB            kernelports.Transactor
	Logger        *zap.Logger
}

type Service struct {
	repo          ediports.EDIConnectionDraftRepository
	documentTypes ediports.EDIDocumentTypeRepository
	setup         setupService
	validator     *ediservice.Validator
	lading        *edilading.Runtime
	projects      projectStore
	db            kernelports.Transactor
	l             *zap.Logger
}

var _ ediports.EDIConnectionDraftService = (*Service)(nil)

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
