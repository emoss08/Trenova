package rateconfirmationservice

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const documentReferenceType = "carrier_assignment"

type Params struct {
	fx.In

	Logger                *zap.Logger
	DB                    ports.DBConnection
	Repo                  repositories.RateConfirmationRepository
	CarrierAssignmentRepo repositories.CarrierAssignmentRepository
	CarrierRepo           repositories.CarrierRepository
	ShipmentRepo          repositories.ShipmentRepository
	OrgRepo               repositories.OrganizationRepository
	DocumentTypeRepo      repositories.DocumentTypeRepository
	Templates             services.DocumentTemplateResolver
	EmailService          services.EmailService
	UploadService         services.DocumentUploadService
	Inliner               services.AssetInliner
}

type Service struct {
	l                     *zap.Logger
	db                    ports.DBConnection
	repo                  repositories.RateConfirmationRepository
	carrierAssignmentRepo repositories.CarrierAssignmentRepository
	carrierRepo           repositories.CarrierRepository
	shipmentRepo          repositories.ShipmentRepository
	orgRepo               repositories.OrganizationRepository
	documentTypeRepo      repositories.DocumentTypeRepository
	templates             services.DocumentTemplateResolver
	emailService          services.EmailService
	uploadService         services.DocumentUploadService
	inliner               services.AssetInliner
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
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ListByMove(
	ctx context.Context,
	req *repositories.ListRateConfirmationsByMoveRequest,
) ([]*rateconfirmation.RateConfirmation, error) {
	return s.repo.ListByMoveID(ctx, req)
}

// Generate renders a new rate confirmation revision for the move's active
// carrier assignment, files the PDF as a shipment document, and voids the prior
// active revision so exactly one agreement stands at a time.
func (s *Service) Generate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	actor *services.RequestActor,
) (*rateconfirmation.RateConfirmation, error) {
	if s.templates == nil {
		return nil, errortypes.NewBusinessError(
			"Document templates are not configured; the rate confirmation cannot be rendered",
		)
	}

	assignment, err := s.carrierAssignmentRepo.GetActiveByMoveID(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return nil, errortypes.NewBusinessError(
			"Shipment move has no active carrier assignment to confirm",
		).WithParam("shipmentMoveId", moveID.String())
	}

	carrierEntity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         assignment.CarrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeContacts: true,
		},
	})
	if err != nil {
		return nil, err
	}

	shipmentEntity, moveEntity, err := s.loadShipmentAndMove(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}

	maxRevision, err := s.repo.MaxRevisionForAssignment(ctx, tenantInfo, assignment.ID)
	if err != nil {
		return nil, err
	}
	revision := maxRevision + 1

	templateContext := buildContext(payloadParams{
		Shipment:    shipmentEntity,
		Move:        moveEntity,
		Assignment:  assignment,
		Carrier:     carrierEntity,
		Revision:    revision,
		CompanyName: s.companyName(ctx, tenantInfo),
	})
	s.applyLogo(ctx, tenantInfo, templateContext)

	rendered, err := s.templates.RenderDocument(ctx, &services.RenderDocumentRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindRateConfirmationPDF,
		Data:        templateContext,
		ReferenceID: assignment.ID,
		UserID:      tenantInfo.UserID,
		Title:       documentTitle(shipmentEntity, revision),
	})
	if err != nil {
		return nil, err
	}
	if len(rendered.PDF) == 0 {
		return nil, errortypes.NewBusinessError("The rate confirmation PDF rendered no content")
	}

	snapshot, err := contextSnapshot(templateContext)
	if err != nil {
		return nil, err
	}

	var created *rateconfirmation.RateConfirmation
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if txErr := s.voidActiveRevision(
			txCtx, tenantInfo, assignment.ID, "Superseded by a new revision",
		); txErr != nil {
			return txErr
		}

		entity := &rateconfirmation.RateConfirmation{
			OrganizationID:      tenantInfo.OrgID,
			BusinessUnitID:      tenantInfo.BuID,
			CarrierAssignmentID: assignment.ID,
			CarrierID:           assignment.CarrierID,
			ShipmentID:          shipmentEntity.ID,
			ShipmentMoveID:      moveID,
			Revision:            revision,
			Status:              rateconfirmation.StatusGenerated,
			PayloadSnapshot:     snapshot,
		}
		if actor != nil && !actor.UserID.IsNil() {
			userID := actor.UserID
			entity.GeneratedByID = &userID
		}

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		var txErr error
		created, txErr = s.repo.Create(txCtx, entity)
		return txErr
	})
	if err != nil {
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
	entity, err := s.repo.GetByID(ctx, &repositories.GetRateConfirmationByIDRequest{
		TenantInfo:         tenantInfo,
		RateConfirmationID: rateConfirmationID,
	})
	if err != nil {
		return nil, err
	}
	if !entity.CanSend() {
		return nil, errortypes.NewBusinessError(
			fmt.Sprintf("A %s rate confirmation cannot be sent", entity.Status),
		)
	}
	if s.emailService == nil {
		return nil, errortypes.NewBusinessError(
			"No email service is configured. Download the rate confirmation and deliver it manually",
		)
	}
	if len(entity.PayloadSnapshot) == 0 {
		return nil, errortypes.NewBusinessError(
			"The rate confirmation has no payload snapshot to render from. Regenerate it first",
		)
	}

	recipients, err := s.recipients(ctx, tenantInfo, entity)
	if err != nil {
		return nil, err
	}

	message, err := s.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindRateConfirmationEmail,
		Data:        entity.PayloadSnapshot,
		ReferenceID: entity.CarrierAssignmentID,
	})
	if err != nil {
		return nil, err
	}

	rendered, err := s.templates.RenderDocument(ctx, &services.RenderDocumentRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindRateConfirmationPDF,
		Data:        entity.PayloadSnapshot,
		ReferenceID: entity.CarrierAssignmentID,
		Title:       fmt.Sprintf("Rate Confirmation Revision %d", entity.Revision),
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.emailService.Send(ctx, &services.SendEmailRequest{
		TenantInfo: tenantInfo,
		Purpose:    email.PurposeNotifications,
		To:         recipients,
		Subject:    message.Subject,
		Text:       message.Text,
		HTML:       message.HTML,
		Attachments: []services.EmailAttachment{
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

	now := timeutils.NowUnix()
	entity.Status = rateconfirmation.StatusSent
	entity.SentAt = &now
	entity.SentToEmails = strings.Join(recipients, ", ")

	return s.repo.Update(ctx, entity)
}

func (s *Service) MarkConfirmed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	confirmedByName string,
) (*rateconfirmation.RateConfirmation, error) {
	entity, err := s.repo.GetByID(ctx, &repositories.GetRateConfirmationByIDRequest{
		TenantInfo:         tenantInfo,
		RateConfirmationID: rateConfirmationID,
	})
	if err != nil {
		return nil, err
	}
	if !entity.CanConfirm() {
		return nil, errortypes.NewBusinessError(
			fmt.Sprintf("A %s rate confirmation cannot be confirmed", entity.Status),
		)
	}
	if confirmedByName == "" {
		return nil, errortypes.NewValidationError(
			"confirmedByName", errortypes.ErrRequired,
			"The confirming party's name is required")
	}

	now := timeutils.NowUnix()
	entity.Status = rateconfirmation.StatusConfirmed
	entity.ConfirmedAt = &now
	entity.ConfirmedByName = confirmedByName

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.confirmAssignment(ctx, tenantInfo, entity.CarrierAssignmentID, now)

	return updated, nil
}

func (s *Service) Void(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	reason string,
) (*rateconfirmation.RateConfirmation, error) {
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason", errortypes.ErrRequired, "A void reason is required")
	}

	entity, err := s.repo.GetByID(ctx, &repositories.GetRateConfirmationByIDRequest{
		TenantInfo:         tenantInfo,
		RateConfirmationID: rateConfirmationID,
	})
	if err != nil {
		return nil, err
	}
	if entity.Status == rateconfirmation.StatusVoided {
		return entity, nil
	}

	now := timeutils.NowUnix()
	entity.Status = rateconfirmation.StatusVoided
	entity.VoidedAt = &now
	entity.VoidReason = reason

	return s.repo.Update(ctx, entity)
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

	now := timeutils.NowUnix()
	active.Status = rateconfirmation.StatusVoided
	active.VoidedAt = &now
	active.VoidReason = reason
	_, err = s.repo.Update(ctx, active)
	return err
}

// confirmAssignment flips the carrier assignment to Confirmed alongside its
// confirmed rate confirmation. A failure is logged rather than returned: the
// confirmation record itself is the agreement.
func (s *Service) confirmAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
	now int64,
) {
	assignment, err := s.carrierAssignmentRepo.GetByID(
		ctx,
		&repositories.GetCarrierAssignmentByIDRequest{
			TenantInfo:          tenantInfo,
			CarrierAssignmentID: assignmentID,
		},
	)
	if err != nil {
		s.l.Warn("failed to load the carrier assignment for confirmation", zap.Error(err))
		return
	}
	if assignment.Status != shipment.CarrierAssignmentStatusPending {
		return
	}

	assignment.Status = shipment.CarrierAssignmentStatusConfirmed
	assignment.ConfirmedAt = &now
	if _, err = s.carrierAssignmentRepo.Update(ctx, assignment); err != nil {
		s.l.Warn("failed to confirm the carrier assignment", zap.Error(err))
	}
}

func (s *Service) recipients(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *rateconfirmation.RateConfirmation,
) ([]string, error) {
	carrierEntity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         entity.CarrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
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

	shipmentEntity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
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
			&repositories.GetCarrierAssignmentByIDRequest{
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

	org, err := s.orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
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

	org, err := s.orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return
	}

	dataURI, err := services.ResolveLogoDataURI(ctx, s.inliner, org.LogoURL)
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
	rendered *services.RenderedDocument,
	actor *services.RequestActor,
) {
	log := s.l.With(
		zap.String("operation", "fileDocument"),
		zap.String("rateConfirmationId", entity.ID.String()),
	)

	if s.uploadService == nil {
		log.Warn("document uploads are not configured; the rate confirmation PDF was not filed")
		return
	}

	docType, err := s.documentTypeRepo.GetByCode(ctx, repositories.GetDocumentTypeByCodeRequest{
		Code:       documenttype.CodeRateConfirmation,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		log.Warn("failed to resolve the rate confirmation document type", zap.Error(err))
		return
	}

	requestActor := services.RequestActor{
		PrincipalType:  services.PrincipalTypeSystem,
		PrincipalID:    services.SystemPrincipalID,
		UserID:         tenantInfo.UserID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
	if actor != nil {
		requestActor = *actor
	}

	session, err := s.uploadService.CreateSession(ctx, &services.CreateSessionRequest{
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

	if _, err = s.uploadService.UploadPart(ctx, &services.UploadPartRequest{
		TenantInfo: tenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(rendered.PDF),
		Size:       int64(len(rendered.PDF)),
	}); err != nil {
		log.Warn("failed to upload the rate confirmation PDF", zap.Error(err))
		return
	}

	completed, err := s.uploadService.Complete(ctx, &services.CompletionRequest{
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
	if _, err = s.templates.RecordGeneratedDocument(ctx, &services.RecordGeneratedDocumentRequest{
		TenantInfo:    tenantInfo,
		Kind:          documenttemplate.KindRateConfirmationPDF,
		Rendered:      &provenance,
		ReferenceType: documentReferenceType,
		ReferenceID:   entity.CarrierAssignmentID,
		DocumentID:    completed.DocumentID,
		FileName:      fileName(entity),
		FileSize:      int64(len(rendered.PDF)),
		UserID:        tenantInfo.UserID,
	}); err != nil {
		log.Warn("could not record the generated rate confirmation PDF", zap.Error(err))
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
