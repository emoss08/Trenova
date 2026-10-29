package edisetup

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	AuditService       auditports.AuditService
	ConnectionRepo     edipartnerports.EDIConnectionRepository
	ControlNumberRepo  edipartnerports.EDIControlNumberRepository
	DocumentTypeRepo   edipartnerports.EDIDocumentTypeRepository
	Encryption         *encryptionservice.Service
	Logger             *zap.Logger
	MappingProfileRepo edipartnerports.EDIMappingProfileRepository
	PartnerRepo        edipartnerports.EDIPartnerRepository
	Plans              platformplanports.PlanService `optional:"true"`
	ProfileRepo        edipartnerports.EDICommunicationProfileRepository
	Secret             *ediprofilesecret.Service
	TransactionSetRepo edipartnerports.EDITransactionSetRepository
	Transport          editransportports.EDITransportDispatcher
	Validator          *edivalidation.Validator
}

type Service struct {
	auditService       auditports.AuditService
	connectionRepo     edipartnerports.EDIConnectionRepository
	controlNumberRepo  edipartnerports.EDIControlNumberRepository
	documentTypeRepo   edipartnerports.EDIDocumentTypeRepository
	encryption         *encryptionservice.Service
	l                  *zap.Logger
	mappingProfileRepo edipartnerports.EDIMappingProfileRepository
	partnerRepo        edipartnerports.EDIPartnerRepository
	plans              platformplanports.PlanService
	profileRepo        edipartnerports.EDICommunicationProfileRepository
	transactionSetRepo edipartnerports.EDITransactionSetRepository
	transport          editransportports.EDITransportDispatcher
	validator          *edivalidation.Validator
	secret             *ediprofilesecret.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		auditService:       p.AuditService,
		connectionRepo:     p.ConnectionRepo,
		controlNumberRepo:  p.ControlNumberRepo,
		documentTypeRepo:   p.DocumentTypeRepo,
		encryption:         p.Encryption,
		l:                  p.Logger.Named("service.edi"),
		mappingProfileRepo: p.MappingProfileRepo,
		partnerRepo:        p.PartnerRepo,
		plans:              p.Plans,
		profileRepo:        p.ProfileRepo,
		transactionSetRepo: p.TransactionSetRepo,
		transport:          p.Transport,
		validator:          p.Validator,
		secret:             p.Secret,
	}
}

func (s *Service) logAction(
	entity edilading.Audited,
	actor *permissionports.RequestActor,
	operation permission.Operation,
	previous any,
	current any,
	comment string,
) {
	edilading.LogAction(s.auditService, s.l, &edilading.AuditEntry{
		Entity:    entity,
		Actor:     actor,
		Operation: operation,
		Previous:  previous,
		Current:   current,
		Comment:   comment,
	})
}
