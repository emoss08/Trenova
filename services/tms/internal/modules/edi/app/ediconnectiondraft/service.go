package ediconnectiondraft

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	UpsertPartnerDocumentProfile(
		ctx context.Context,
		req *ediservice.UpsertEDIPartnerDocumentProfileRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDIPartnerDocumentProfile, error)
	SaveMappingProfile(
		ctx context.Context,
		req *ediports.SaveMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	StartControlNumber(
		ctx context.Context,
		req *ediports.ResetEDIControlNumberRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDIControlNumberSequence, error)
	ValidatePartnerSettings(
		ctx context.Context,
		req *ediservice.ValidatePartnerSettingsRequest,
	) ([]edix12.Diagnostic, error)
}

type Params struct {
	fx.In

	Repo          ediports.EDIConnectionDraftRepository
	Templates     ediports.EDITemplateRepository
	DocumentTypes ediports.EDIDocumentTypeRepository
	Setup         *ediservice.Service
	Validator     *ediservice.Validator
	DB            kernelports.DBConnection
	Logger        *zap.Logger
}

type Service struct {
	repo          ediports.EDIConnectionDraftRepository
	templates     ediports.EDITemplateRepository
	documentTypes ediports.EDIDocumentTypeRepository
	setup         setupService
	validator     *ediservice.Validator
	db            kernelports.DBConnection
	l             *zap.Logger
}

var _ ediports.EDIConnectionDraftService = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		repo:          p.Repo,
		templates:     p.Templates,
		documentTypes: p.DocumentTypes,
		setup:         p.Setup,
		validator:     p.Validator,
		db:            p.DB,
		l:             p.Logger.Named("service.edi-connection-draft"),
	}
}
