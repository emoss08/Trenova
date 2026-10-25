package rateconfirmationservice

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const documentReferenceType = "carrier_assignment"

type Params struct {
	fx.In

	Logger                *zap.Logger
	DB                    kernelports.Transactor
	Repo                  rateconfirmationports.RateConfirmationRepository
	CarrierAssignmentRepo shipmentports.CarrierAssignmentRepository
	CarrierRepo           carrierports.CarrierRepository
	ShipmentRepo          shipmentports.ShipmentRepository
	OrgRepo               tenantports.OrganizationRepository
	DocumentTypeRepo      documenttypeports.DocumentTypeRepository
	Templates             documenttemplateports.DocumentTemplateResolver
	EmailService          emailports.EmailService
	UploadService         documentuploadports.DocumentUploadService
	Inliner               documenttemplateports.AssetInliner
	AuditService          auditports.AuditService
	Config                *config.Config
}

type Service struct {
	l                     *zap.Logger
	db                    kernelports.Transactor
	repo                  rateconfirmationports.RateConfirmationRepository
	carrierAssignmentRepo shipmentports.CarrierAssignmentRepository
	carrierRepo           carrierports.CarrierRepository
	shipmentRepo          shipmentports.ShipmentRepository
	orgRepo               tenantports.OrganizationRepository
	documentTypeRepo      documenttypeports.DocumentTypeRepository
	templates             documenttemplateports.DocumentTemplateResolver
	emailService          emailports.EmailService
	uploadService         documentuploadports.DocumentUploadService
	inliner               documenttemplateports.AssetInliner
	auditService          auditports.AuditService
	cfg                   *config.Config
}

func New(p Params) *Service {
	return &Service{
		l:                     p.Logger.Named("service.rate-confirmation"),
		db:                    p.DB,
		repo:                  p.Repo,
		carrierAssignmentRepo: p.CarrierAssignmentRepo,
		carrierRepo:           p.CarrierRepo,
		shipmentRepo:          p.ShipmentRepo,
		orgRepo:               p.OrgRepo,
		documentTypeRepo:      p.DocumentTypeRepo,
		templates:             p.Templates,
		emailService:          p.EmailService,
		uploadService:         p.UploadService,
		inliner:               p.Inliner,
		auditService:          p.AuditService,
		cfg:                   p.Config,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *rateconfirmationports.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ListByMove(
	ctx context.Context,
	req *rateconfirmationports.ListRateConfirmationsByMoveRequest,
) ([]*rateconfirmation.RateConfirmation, error) {
	return s.repo.ListByMoveID(ctx, req)
}

// generateOptions carries provenance for revisions born outside the manual
// dispatcher flow, so the row records how it came to exist from the start.
type generateOptions struct {
	Via                 rateconfirmation.Via
	SourceTenderOfferID *pulid.ID
}

// Generate renders a new rate confirmation revision for the move's active
// carrier assignment, files the PDF as a shipment document, and voids the prior
// active revision so exactly one agreement stands at a time.
func (s *Service) Generate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	actor *permissionports.RequestActor,
) (*rateconfirmation.RateConfirmation, error) {
	created, err := s.generate(ctx, tenantInfo, moveID, actor, nil)
	if err != nil {
		return nil, err
	}

	s.logRateConfirmationAudit(&rateConfirmationAuditParams{
		TenantInfo: tenantInfo,
		Operation:  permission.OpCreate,
		UserID:     actor.UserIDOrNil(),
		Comment:    fmt.Sprintf("Rate confirmation rev %d generated", created.Revision),
		Current:    created,
	})

	return created, nil
}

func (s *Service) generate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	actor *permissionports.RequestActor,
	opts *generateOptions,
) (*rateconfirmation.RateConfirmation, error) {
	plan, err := s.planGenerate(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}
	assignment := plan.assignment
	shipmentEntity := plan.shipment

	// The revision read, render, and create all run in one transaction so two
	// concurrent generates cannot both claim the same revision; the loser hits
	// the unique index and surfaces as a retryable business error.
	var created *rateconfirmation.RateConfirmation
	var rendered *documenttemplateports.RenderedDocument
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		maxRevision, txErr := s.repo.MaxRevisionForAssignment(txCtx, tenantInfo, assignment.ID)
		if txErr != nil {
			return txErr
		}
		revision := maxRevision + 1

		templateContext := buildContext(payloadParams{
			Shipment:    shipmentEntity,
			Move:        plan.move,
			Assignment:  assignment,
			Carrier:     plan.carrier,
			Revision:    revision,
			CompanyName: s.companyName(txCtx, tenantInfo),
		})
		s.applyLogo(txCtx, tenantInfo, templateContext)

		rendered, txErr = s.templates.RenderDocument(
			txCtx,
			&documenttemplateports.RenderDocumentRequest{
				TenantInfo:  tenantInfo,
				Kind:        documenttemplate.KindRateConfirmationPDF,
				Data:        templateContext,
				ReferenceID: assignment.ID,
				UserID:      tenantInfo.UserID,
				Title:       documentTitle(shipmentEntity, revision),
			},
		)
		if txErr != nil {
			return txErr
		}
		if len(rendered.PDF) == 0 {
			return errortypes.NewBusinessError("The rate confirmation PDF rendered no content")
		}

		snapshot, txErr := contextSnapshot(templateContext)
		if txErr != nil {
			return txErr
		}

		if txErr = s.voidActiveRevision(
			txCtx, tenantInfo, assignment.ID, supersededRevisionReason,
		); txErr != nil {
			return txErr
		}

		entity := newRevision(tenantInfo, plan, revision, actor, opts)
		entity.PayloadSnapshot = snapshot

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		created, txErr = s.repo.Create(txCtx, entity)
		return txErr
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewBusinessError(
				"A rate confirmation is already being generated for this assignment. Retry.",
			).WithParam("carrierAssignmentId", assignment.ID.String())
		}
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate confirmation is busy. Retry the request.",
		)
	}

	s.fileDocument(ctx, tenantInfo, created, shipmentEntity, rendered, actor)

	return created, nil
}

// Send re-renders the message and document from the frozen snapshot and emails
// them to the carrier's rate confirmation contacts.
func (s *Service) Send(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
) (*rateconfirmation.RateConfirmation, error) {
	plan, err := s.planSend(ctx, tenantInfo, rateConfirmationID)
	if err != nil {
		return nil, err
	}
	entity := plan.entity
	templateContext := plan.templateContext
	recipients := plan.recipients

	// A not-yet-executed revision travels with a single-use sign link so the
	// carrier can confirm without a login. An already-Confirmed copy is the
	// record copy and gets no link.
	if entity.Status != rateconfirmation.StatusConfirmed {
		if err = s.applySignLink(
			ctx,
			tenantInfo,
			entity,
			recipients[0],
			templateContext,
		); err != nil {
			return nil, err
		}
	}

	message, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindRateConfirmationEmail,
		Data:        templateContext,
		ReferenceID: entity.CarrierAssignmentID,
	})
	if err != nil {
		return nil, err
	}

	rendered, err := s.templates.RenderDocument(ctx, &documenttemplateports.RenderDocumentRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindRateConfirmationPDF,
		Data:        templateContext,
		ReferenceID: entity.CarrierAssignmentID,
		Title:       fmt.Sprintf("Rate Confirmation Revision %d", entity.Revision),
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo: tenantInfo,
		Purpose:    email.PurposeNotifications,
		To:         recipients,
		Subject:    message.Subject,
		Text:       message.Text,
		HTML:       message.HTML,
		Attachments: []emailports.EmailAttachment{
			{
				FileName:    fileName(entity),
				ContentType: "application/pdf",
				Content:     rendered.PDF,
				SizeBytes:   int64(len(rendered.PDF)),
			},
		},
		IdempotencyKey: fmt.Sprintf("rate-confirmation-%s-%d", entity.ID, entity.Revision),
	}); err != nil {
		return nil, err
	}

	// The email is out; the status write re-reads the row so a concurrent
	// transition (e.g. a void) is kept rather than clobbered — and never turned
	// into an error that would invite a duplicate send on retry.
	var updated *rateconfirmation.RateConfirmation
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		fresh, txErr := s.repo.GetByID(txCtx, &rateconfirmationports.GetRateConfirmationByIDRequest{
			TenantInfo:         tenantInfo,
			RateConfirmationID: rateConfirmationID,
		})
		if txErr != nil {
			return txErr
		}
		if !fresh.CanSend() {
			s.l.Warn(
				"rate confirmation changed status while the email was in flight; keeping the newer status",
				zap.String("rateConfirmationId", rateConfirmationID.String()),
				zap.String("status", fresh.Status.String()),
			)
			updated = fresh
			return nil
		}

		markSent(fresh, recipients, timeutils.NowUnix())
		updated, txErr = s.repo.Update(txCtx, fresh)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logRateConfirmationAudit(&rateConfirmationAuditParams{
		TenantInfo: tenantInfo,
		Operation:  permission.OpUpdate,
		UserID:     tenantInfo.UserID,
		Comment:    "Rate confirmation sent to " + strings.Join(recipients, ", "),
		Previous:   entity,
		Current:    updated,
	})

	return updated, nil
}

func (s *Service) MarkConfirmed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	confirmedByName string,
) (*rateconfirmation.RateConfirmation, error) {
	entity, err := s.planConfirm(ctx, tenantInfo, rateConfirmationID, confirmedByName)
	if err != nil {
		return nil, err
	}

	previous := *entity
	confirmed, err := s.markConfirmed(
		ctx,
		tenantInfo,
		entity,
		dispatcherConfirmation(confirmedByName, timeutils.NowUnix()),
	)
	if err != nil {
		return nil, err
	}

	s.logRateConfirmationAudit(&rateConfirmationAuditParams{
		TenantInfo: tenantInfo,
		Operation:  permission.OpApprove,
		UserID:     tenantInfo.UserID,
		Comment:    "Rate confirmation marked confirmed by dispatcher",
		Previous:   &previous,
		Current:    confirmed,
	})

	return confirmed, nil
}

// markConfirmed is the single execution path shared by the dispatcher's
// MarkConfirmed, the public ConfirmByToken, and tender-acceptance issuance:
// the confirmation and the assignment flip in one transaction so the
// agreement and the assignment can never disagree.
func (s *Service) markConfirmed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *rateconfirmation.RateConfirmation,
	p rateconfirmation.Confirmation,
) (*rateconfirmation.RateConfirmation, error) {
	at := p.At
	entity.Confirm(p)

	var updated *rateconfirmation.RateConfirmation
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		updated, txErr = s.repo.Update(txCtx, entity)
		if txErr != nil {
			return txErr
		}
		return s.confirmAssignment(txCtx, tenantInfo, entity.CarrierAssignmentID, at)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate confirmation is busy. Retry the request.",
		)
	}

	return updated, nil
}

func (s *Service) Void(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	reason string,
) (*rateconfirmation.RateConfirmation, error) {
	entity, err := s.planVoid(ctx, tenantInfo, rateConfirmationID, reason)
	if err != nil {
		return nil, err
	}
	if entity.Status == rateconfirmation.StatusVoided {
		return entity, nil
	}
	wasConfirmed := entity.Status == rateconfirmation.StatusConfirmed
	previous := *entity

	entity.Void(timeutils.NowUnix(), reason)

	var updated *rateconfirmation.RateConfirmation
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		updated, txErr = s.repo.Update(txCtx, entity)
		if txErr != nil {
			return txErr
		}
		if !wasConfirmed {
			return nil
		}
		// Voiding a confirmed agreement withdraws the confirmation, so the
		// assignment reverts to Pending — the counterpart of confirmAssignment.
		return s.revertAssignmentConfirmation(txCtx, tenantInfo, entity.CarrierAssignmentID)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate confirmation is busy. Retry the request.",
		)
	}

	s.revokeTokens(ctx, tenantInfo, updated.ID)

	s.logRateConfirmationAudit(&rateConfirmationAuditParams{
		TenantInfo: tenantInfo,
		Operation:  permission.OpUpdate,
		UserID:     tenantInfo.UserID,
		Comment:    "Rate confirmation voided: " + reason,
		Previous:   &previous,
		Current:    updated,
	})

	return updated, nil
}

func (s *Service) voidActiveRevision(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
	reason string,
) error {
	active, err := s.repo.GetActiveByAssignmentID(ctx, tenantInfo, assignmentID)
	if err != nil {
		return err
	}
	if active == nil {
		return nil
	}

	active.Void(timeutils.NowUnix(), reason)
	if _, err = s.repo.Update(ctx, active); err != nil {
		return err
	}

	s.revokeTokens(ctx, tenantInfo, active.ID)
	return nil
}

// revokeTokens retires every outstanding sign link when its agreement dies.
// Best-effort: resolveToken re-checks the rate confirmation's state, so a
// missed revocation cannot resurrect a voided agreement.
func (s *Service) revokeTokens(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
) {
	if err := s.repo.RevokeTokensForRateConfirmation(
		ctx, tenantInfo, rateConfirmationID, timeutils.NowUnix(),
	); err != nil {
		s.l.Warn("failed to revoke rate confirmation tokens",
			zap.Error(err),
			zap.String("rateConfirmationId", rateConfirmationID.String()))
	}
}

// confirmAssignment flips the carrier assignment to Confirmed alongside its
// confirmed rate confirmation. It runs in the caller's transaction so the
// agreement and the assignment can never disagree.
func (s *Service) confirmAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
	now int64,
) error {
	assignment, err := s.carrierAssignmentRepo.GetByID(
		ctx,
		&shipmentports.GetCarrierAssignmentByIDRequest{
			TenantInfo:          tenantInfo,
			CarrierAssignmentID: assignmentID,
		},
	)
	if err != nil {
		return err
	}
	if !assignment.Confirm(now) {
		return nil
	}

	_, err = s.carrierAssignmentRepo.Update(ctx, assignment)
	return err
}

// revertAssignmentConfirmation is confirmAssignment's counterpart for voids: a
// Confirmed assignment whose agreement was withdrawn goes back to Pending.
func (s *Service) revertAssignmentConfirmation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
) error {
	assignment, err := s.carrierAssignmentRepo.GetByID(
		ctx,
		&shipmentports.GetCarrierAssignmentByIDRequest{
			TenantInfo:          tenantInfo,
			CarrierAssignmentID: assignmentID,
		},
	)
	if err != nil {
		return err
	}
	if !assignment.RevertConfirmation() {
		return nil
	}

	_, err = s.carrierAssignmentRepo.Update(ctx, assignment)
	return err
}

func (s *Service) recipients(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *rateconfirmation.RateConfirmation,
) ([]string, error) {
	carrierEntity, err := s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
		ID:         entity.CarrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: carrierports.CarrierFilterOptions{
			IncludeContacts: true,
		},
	})
	if err != nil {
		return nil, err
	}

	recipients := make([]string, 0, len(carrierEntity.Contacts))
	for _, contact := range carrierEntity.Contacts {
		if contact != nil && contact.ReceivesRateConfirmations && contact.Email != "" {
			recipients = append(recipients, contact.Email)
		}
	}
	if len(recipients) == 0 && carrierEntity.Email != "" {
		recipients = append(recipients, carrierEntity.Email)
	}
	if len(recipients) == 0 {
		return nil, errortypes.NewBusinessError(
			"Carrier has no rate confirmation recipients. Add a contact that receives rate confirmations or a carrier email",
		).WithParam("carrierId", entity.CarrierID.String())
	}

	return recipients, nil
}

func (s *Service) loadShipmentAndMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.Shipment, *shipment.ShipmentMove, error) {
	shipmentID, err := s.shipmentIDForMove(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, nil, err
	}

	shipmentEntity, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	for _, move := range shipmentEntity.Moves {
		if move != nil && move.ID == moveID {
			return shipmentEntity, move, nil
		}
	}

	return nil, nil, errortypes.NewBusinessError("Shipment does not contain the target move").
		WithParam("shipmentMoveId", moveID.String())
}

func (s *Service) shipmentIDForMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (pulid.ID, error) {
	assignment, err := s.carrierAssignmentRepo.GetActiveByMoveID(ctx, tenantInfo, moveID)
	if err != nil {
		return pulid.Nil, err
	}
	if assignment != nil && assignment.ShipmentMove != nil {
		return assignment.ShipmentMove.ShipmentID, nil
	}
	if assignment != nil {
		full, gErr := s.carrierAssignmentRepo.GetByID(
			ctx,
			&shipmentports.GetCarrierAssignmentByIDRequest{
				TenantInfo:          tenantInfo,
				CarrierAssignmentID: assignment.ID,
			},
		)
		if gErr != nil {
			return pulid.Nil, gErr
		}
		if full.ShipmentMove != nil {
			return full.ShipmentMove.ShipmentID, nil
		}
	}

	return pulid.Nil, errortypes.NewBusinessError(
		"Shipment move has no active carrier assignment to confirm",
	).WithParam("shipmentMoveId", moveID.String())
}

func (s *Service) companyName(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) string {
	if s.orgRepo == nil {
		return ""
	}

	org, err := s.orgRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return ""
	}
	return org.Name
}

func (s *Service) applyLogo(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *documenttemplate.RateConfirmationContext,
) {
	if s.orgRepo == nil {
		return
	}

	org, err := s.orgRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return
	}

	dataURI, err := documenttemplateports.ResolveLogoDataURI(ctx, s.inliner, org.LogoURL)
	if err != nil {
		return
	}
	out.LogoDataURI = dataURI
}

// fileDocument stores the rendered PDF as a shipment document. Failures are
// logged rather than returned: the confirmation row exists and can be re-sent;
// a missing filing copy is a bookkeeping gap, not a lost agreement.
func (s *Service) fileDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *rateconfirmation.RateConfirmation,
	shipmentEntity *shipment.Shipment,
	rendered *documenttemplateports.RenderedDocument,
	actor *permissionports.RequestActor,
) {
	log := s.l.With(
		zap.String("operation", "fileDocument"),
		zap.String("rateConfirmationId", entity.ID.String()),
	)

	if s.uploadService == nil {
		log.Warn("document uploads are not configured; the rate confirmation PDF was not filed")
		return
	}

	docType, err := s.documentTypeRepo.GetByCode(
		ctx,
		documenttypeports.GetDocumentTypeByCodeRequest{
			Code:       documenttype.CodeRateConfirmation,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		log.Warn("failed to resolve the rate confirmation document type", zap.Error(err))
		return
	}

	requestActor := filingActor(tenantInfo, actor)

	session, err := s.uploadService.CreateSession(ctx, &documentuploadports.CreateSessionRequest{
		TenantInfo:     tenantInfo,
		Actor:          requestActor,
		ResourceID:     shipmentEntity.ID.String(),
		ResourceType:   "shipment",
		FileName:       fileName(entity),
		FileSize:       int64(len(rendered.PDF)),
		ContentType:    "application/pdf",
		Description:    "Carrier rate confirmation",
		Tags:           []string{"carrier", "rate-confirmation", "generated"},
		DocumentTypeID: docType.ID.String(),
	})
	if err != nil {
		log.Warn("failed to stage the rate confirmation PDF for storage", zap.Error(err))
		return
	}

	if _, err = s.uploadService.UploadPart(ctx, &documentuploadports.UploadPartRequest{
		TenantInfo: tenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(rendered.PDF),
		Size:       int64(len(rendered.PDF)),
	}); err != nil {
		log.Warn("failed to upload the rate confirmation PDF", zap.Error(err))
		return
	}

	completed, err := s.uploadService.Complete(ctx, &documentuploadports.CompletionRequest{
		TenantInfo: tenantInfo,
		Actor:      requestActor,
		SessionID:  session.ID,
	})
	if err != nil {
		log.Warn("failed to finalize the rate confirmation PDF upload", zap.Error(err))
		return
	}
	if completed.DocumentID == nil || completed.DocumentID.IsNil() {
		log.Warn("the rate confirmation PDF upload did not produce a document")
		return
	}

	entity.DocumentID = completed.DocumentID
	if _, err = s.repo.Update(ctx, entity); err != nil {
		log.Warn("failed to link the rate confirmation document", zap.Error(err))
	}

	provenance := *rendered
	provenance.PDF = nil
	provenance.HTML = ""
	if _, err = s.templates.RecordGeneratedDocument(
		ctx,
		&documenttemplateports.RecordGeneratedDocumentRequest{
			TenantInfo:    tenantInfo,
			Kind:          documenttemplate.KindRateConfirmationPDF,
			Rendered:      &provenance,
			ReferenceType: documentReferenceType,
			ReferenceID:   entity.CarrierAssignmentID,
			DocumentID:    completed.DocumentID,
			FileName:      fileName(entity),
			FileSize:      int64(len(rendered.PDF)),
			UserID:        tenantInfo.UserID,
		},
	); err != nil {
		log.Warn("could not record the generated rate confirmation PDF", zap.Error(err))
	}
}

func filingActor(
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
) permissionports.RequestActor {
	if actor != nil {
		return *actor
	}

	return permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		PrincipalID:    permissionports.SystemPrincipalID,
		UserID:         tenantInfo.UserID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

func contextSnapshot(
	templateContext *documenttemplate.RateConfirmationContext,
) (map[string]any, error) {
	raw, err := sonic.Marshal(templateContext)
	if err != nil {
		return nil, fmt.Errorf("snapshot rate confirmation context: %w", err)
	}

	var snapshot map[string]any
	if err = sonic.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("snapshot rate confirmation context: %w", err)
	}
	return snapshot, nil
}

func contextFromSnapshot(
	snapshot map[string]any,
) (*documenttemplate.RateConfirmationContext, error) {
	raw, err := sonic.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("decode rate confirmation snapshot: %w", err)
	}

	out := new(documenttemplate.RateConfirmationContext)
	if err = sonic.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("decode rate confirmation snapshot: %w", err)
	}
	return out, nil
}

func documentTitle(shipmentEntity *shipment.Shipment, revision int64) string {
	ref := shipmentEntity.ProNumber
	if ref == "" {
		ref = shipmentEntity.ID.String()
	}
	return fmt.Sprintf("Rate Confirmation %s Revision %d", ref, revision)
}

func fileName(entity *rateconfirmation.RateConfirmation) string {
	return fmt.Sprintf("rate-confirmation-%s-rev%d.pdf", entity.ID.String(), entity.Revision)
}
