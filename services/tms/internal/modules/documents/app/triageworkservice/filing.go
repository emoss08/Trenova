package triageworkservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

const (
	inboundResourceType = "inbound_message"
	maxBulkFileMessages = inboundmessageports.MaxBulkResultMessages
)

type PageRange struct {
	FromPage       int
	ToPage         int
	DocumentTypeID pulid.ID
}

type FileRequest struct {
	TenantInfo     pagination.TenantInfo
	UserID         pulid.ID
	AttachmentID   pulid.ID
	ShipmentID     pulid.ID
	DocumentTypeID pulid.ID
	Pages          []PageRange
}

func (s *Service) File(
	ctx context.Context,
	req *FileRequest,
) (*inboundmessage.InboundMessage, error) {
	attachment, err := s.messages.GetAttachmentByID(
		ctx,
		inboundmessageports.GetInboundAttachmentRequest{
			ID:         req.AttachmentID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if err = requireFileable(attachment); err != nil {
		return nil, err
	}
	if err = s.requireShipment(ctx, req.TenantInfo, req.ShipmentID); err != nil {
		return nil, err
	}

	message, err := s.message(ctx, req.TenantInfo, attachment.MessageID)
	if err != nil {
		return nil, err
	}

	if len(req.Pages) > 0 {
		return s.split(ctx, &splitRequest{
			tenantInfo: req.TenantInfo,
			userID:     req.UserID,
			message:    message,
			attachment: attachment,
			shipmentID: req.ShipmentID,
			pages:      req.Pages,
		})
	}

	if req.DocumentTypeID.IsNil() {
		return nil, errortypes.NewValidationError(
			"documentTypeId", errortypes.ErrRequired, "Choose what to file it as",
		)
	}
	docType, err := s.documentType(ctx, req.TenantInfo, req.DocumentTypeID, "documentTypeId")
	if err != nil {
		return nil, err
	}

	if err = s.fileWhole(ctx, &wholeFiling{
		tenantInfo: req.TenantInfo,
		userID:     req.UserID,
		message:    message,
		attachment: attachment,
		shipmentID: req.ShipmentID,
		docType:    docType,
	}); err != nil {
		return nil, err
	}

	return s.refreshed(ctx, req.TenantInfo, message.ID)
}

func requireFileable(attachment *inboundmessage.InboundAttachment) error {
	if !attachment.Readable() {
		return errortypes.NewBusinessError(
			"This file was not read, so it cannot be filed. Ask the sender for it again.",
		)
	}
	if attachment.Filed() {
		return errortypes.NewBusinessError(
			"This file is already filed. Undo that first to file it somewhere else.",
		)
	}

	return nil
}

func (s *Service) requireShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) error {
	if shipmentID.IsNil() {
		return errortypes.NewValidationError(
			"shipmentId", errortypes.ErrRequired, "Choose the shipment to file it on",
		)
	}

	exists, err := s.shipments.ShipmentExists(ctx, tenantInfo, shipmentID)
	if err != nil {
		return err
	}
	if !exists {
		return errortypes.NewValidationError(
			"shipmentId", errortypes.ErrInvalid, "No shipment with that id exists",
		)
	}

	return nil
}

func (s *Service) documentType(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	field string,
) (*documenttype.DocumentType, error) {
	docType, err := s.documentTypes.GetByID(ctx, documenttypeports.GetDocumentTypeByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				field, errortypes.ErrInvalid, "That document type does not exist",
			)
		}

		return nil, err
	}

	return docType, nil
}

type wholeFiling struct {
	tenantInfo pagination.TenantInfo
	userID     pulid.ID
	message    *inboundmessage.InboundMessage
	attachment *inboundmessage.InboundAttachment
	shipmentID pulid.ID
	docType    *documenttype.DocumentType
}

func (s *Service) fileWhole(ctx context.Context, f *wholeFiling) error {
	attached, err := s.documents.AttachLineageToResource(
		ctx,
		f.attachment.DocumentID,
		inboundmessageports.DocumentResourceType,
		f.shipmentID.String(),
		f.tenantInfo,
		f.userID,
	)
	if err != nil {
		return err
	}

	typeID := f.docType.ID
	if attached.DocumentTypeID == nil || *attached.DocumentTypeID != typeID {
		attached.DocumentTypeID = &typeID
		if _, err = s.documentRepo.Update(ctx, attached); err != nil {
			return err
		}
	}

	replaced := s.replaceExisting(ctx, f, attached.ID)

	f.attachment.FiledShipmentID = f.shipmentID
	f.attachment.FiledAs = f.docType.Name
	f.attachment.FiledAt = s.now()
	f.attachment.FiledBy = f.userID
	f.attachment.SplitDocuments = nil
	if _, err = s.messages.UpdateAttachment(ctx, f.attachment); err != nil {
		return err
	}

	detail := f.docType.Name
	if replaced > 0 {
		detail = fmt.Sprintf("%s, replacing the one on file", f.docType.Name)
	}
	activity := personActivity(f.message, triage.ActivityFiled, f.userID).
		About(f.attachment.FileName, detail)
	activity.RecordID = f.shipmentID
	s.record(ctx, activity)
	s.documentAttached(ctx, f.tenantInfo, f.shipmentID, typeID)

	return nil
}

func (s *Service) replaceExisting(ctx context.Context, f *wholeFiling, filedID pulid.ID) int {
	conflict, ok := replacingConflict(f.message, f.attachment.ID)
	if !ok {
		return 0
	}

	existing := []pulid.ID{conflict.ExistingDocumentID}
	if conflict.ExistingDocumentID.IsNil() {
		current, err := s.work.CurrentShipmentDocuments(
			ctx,
			f.tenantInfo,
			f.shipmentID,
			f.docType.ID,
		)
		if err != nil {
			s.l.Warn("could not find the document a filing replaces",
				zap.String("attachmentId", f.attachment.ID.String()), zap.Error(err))

			return 0
		}
		existing = current
	}

	replaced := 0
	for _, id := range existing {
		if id.IsNil() || id == filedID {
			continue
		}
		if err := s.setDocumentStatus(ctx, f.tenantInfo, id, document.StatusArchived); err != nil {
			s.l.Warn("could not archive the document a filing replaces",
				zap.String("documentId", id.String()), zap.Error(err))

			continue
		}
		replaced++
	}

	return replaced
}

func replacingConflict(
	message *inboundmessage.InboundMessage,
	attachmentID pulid.ID,
) (inboundmessage.Conflict, bool) {
	for i := range message.Conflicts {
		conflict := &message.Conflicts[i]
		if conflict.Kind == inboundmessage.ConflictDuplicateDocument &&
			conflict.AttachmentID == attachmentID &&
			conflict.Resolution == inboundmessage.ConflictReplaceExisting {
			return *conflict, true
		}
	}

	return inboundmessage.Conflict{}, false
}

func (s *Service) setDocumentStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	status document.Status,
) error {
	doc, err := s.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if doc.Status == status {
		return nil
	}

	doc.Status = status
	_, err = s.documentRepo.Update(ctx, doc)

	return err
}

func (s *Service) documentAttached(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	documentTypeID pulid.ID,
) {
	if s.events == nil {
		return
	}
	s.events.DocumentAttached(ctx, tenantInfo, shipmentID, documentTypeID)
}

func (s *Service) refreshed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundmessage.InboundMessage, error) {
	message, err := s.message(ctx, tenantInfo, messageID)
	if err != nil {
		return nil, err
	}
	s.changed(ctx, message)

	return message, nil
}

type UnfileRequest struct {
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
	AttachmentID pulid.ID
}

func (s *Service) Unfile(
	ctx context.Context,
	req *UnfileRequest,
) (*inboundmessage.InboundMessage, error) {
	attachment, err := s.messages.GetAttachmentByID(
		ctx,
		inboundmessageports.GetInboundAttachmentRequest{
			ID:         req.AttachmentID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !attachment.Filed() {
		return nil, errortypes.NewBusinessError("This file is not filed")
	}

	message, err := s.message(ctx, req.TenantInfo, attachment.MessageID)
	if err != nil {
		return nil, err
	}

	shipmentID := attachment.FiledShipmentID
	if len(attachment.SplitDocuments) > 0 {
		for _, split := range attachment.SplitDocuments {
			if err = s.documents.Delete(ctx, documentports.DeleteDocumentRequest{
				ID:         split.DocumentID,
				TenantInfo: req.TenantInfo,
			}, req.UserID); err != nil && !errortypes.IsNotFoundError(err) {
				return nil, err
			}
		}
	} else if attachment.DocumentID.IsNotNil() {
		if _, err = s.documents.AttachLineageToResource(
			ctx,
			attachment.DocumentID,
			inboundResourceType,
			message.ID.String(),
			req.TenantInfo,
			req.UserID,
		); err != nil {
			return nil, err
		}
		s.restoreReplaced(ctx, req.TenantInfo, message, attachment.ID)
	}

	if err = s.clearFiling(ctx, attachment); err != nil {
		return nil, err
	}

	activity := personActivity(message, triage.ActivityUnfiled, req.UserID).
		About(attachment.FileName, "")
	activity.RecordID = shipmentID
	s.record(ctx, activity)

	return s.refreshed(ctx, req.TenantInfo, message.ID)
}

func (s *Service) restoreReplaced(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	message *inboundmessage.InboundMessage,
	attachmentID pulid.ID,
) {
	conflict, ok := replacingConflict(message, attachmentID)
	if !ok || conflict.ExistingDocumentID.IsNil() {
		return
	}
	if err := s.setDocumentStatus(
		ctx,
		tenantInfo,
		conflict.ExistingDocumentID,
		document.StatusActive,
	); err != nil {
		s.l.Warn("could not restore the document an undone filing had replaced",
			zap.String("documentId", conflict.ExistingDocumentID.String()), zap.Error(err))
	}
}

type FileManyRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MessageIDs []pulid.ID
}

func (s *Service) FileMany(
	ctx context.Context,
	req *FileManyRequest,
) (*inboundmessageports.BulkResult, error) {
	if len(req.MessageIDs) > inboundmessageports.MaxBulkSelection {
		return nil, errortypes.NewValidationError(
			"messageIds", errortypes.ErrInvalid,
			"File at most {0} messages at once", inboundmessageports.MaxBulkSelection,
		)
	}

	ids := sliceutils.Dedupe(req.MessageIDs)
	result := &inboundmessageports.BulkResult{
		Messages: make([]*inboundmessage.InboundMessage, 0, min(len(ids), maxBulkFileMessages)),
	}
	changed := make([]*inboundmessage.InboundMessage, 0, len(ids))
	for _, id := range ids {
		message, err := s.message(ctx, req.TenantInfo, id)
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				continue
			}

			return nil, err
		}

		filed, err := s.fileOnMatched(ctx, &matchedFiling{
			tenantInfo: req.TenantInfo,
			userID:     req.UserID,
			message:    message,
		})
		if err != nil {
			return nil, err
		}
		if filed == 0 {
			continue
		}

		refreshed, err := s.message(ctx, req.TenantInfo, id)
		if err != nil {
			return nil, err
		}
		result.Affected++
		changed = append(changed, refreshed)
		if len(result.Messages) < maxBulkFileMessages {
			result.Messages = append(result.Messages, refreshed)
		}
	}
	s.changed(ctx, changed...)

	return result, nil
}

func (s *Service) FileOnMatchedShipment(
	ctx context.Context,
	req inboundmessageports.FileOnMatchedRequest,
) (int, error) {
	userID := req.ActorID
	if req.ByNova {
		userID = pulid.Nil
	}

	filed, err := s.fileOnMatched(ctx, &matchedFiling{
		tenantInfo:  tenantOf(req.Message),
		userID:      userID,
		message:     req.Message,
		queueForPay: req.QueueForPay,
	})
	if err != nil {
		return filed, err
	}
	if filed > 0 {
		refreshed, readErr := s.message(ctx, tenantOf(req.Message), req.Message.ID)
		if readErr != nil {
			return filed, readErr
		}
		s.changed(ctx, refreshed)
	}

	return filed, nil
}

type matchedFiling struct {
	tenantInfo  pagination.TenantInfo
	userID      pulid.ID
	message     *inboundmessage.InboundMessage
	queueForPay bool
}

func (s *Service) fileOnMatched(ctx context.Context, f *matchedFiling) (int, error) {
	message := f.message
	if message.MatchedShipmentID.IsNil() || !triage.FilesAttachments(message.Classification) {
		return 0, nil
	}

	attachments := message.Attachments
	if attachments == nil {
		listed, err := s.messages.ListAttachments(ctx, message.ID, f.tenantInfo)
		if err != nil {
			return 0, err
		}
		attachments = listed
	}

	filed := 0
	typeCache := make(map[inboundmessage.AttachmentKind]*documenttype.DocumentType, 2)
	for _, attachment := range attachments {
		if !attachment.Readable() || attachment.Filed() {
			continue
		}

		docType, err := s.typeForKind(ctx, f.tenantInfo, attachment.Kind, typeCache)
		if err != nil {
			return filed, err
		}
		if docType == nil {
			continue
		}

		if err = s.fileWhole(ctx, &wholeFiling{
			tenantInfo: f.tenantInfo,
			userID:     f.userID,
			message:    message,
			attachment: attachment,
			shipmentID: message.MatchedShipmentID,
			docType:    docType,
		}); err != nil {
			return filed, err
		}
		filed++

		if f.queueForPay && attachment.Kind == inboundmessage.AttachmentInvoice {
			s.queueForPay(ctx, f, attachment)
		}
	}

	return filed, nil
}

func (s *Service) typeForKind(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	kind inboundmessage.AttachmentKind,
	cache map[inboundmessage.AttachmentKind]*documenttype.DocumentType,
) (*documenttype.DocumentType, error) {
	if cached, ok := cache[kind]; ok {
		return cached, nil
	}

	code, ok := kind.DocumentTypeCode()
	if !ok {
		cache[kind] = nil

		return nil, nil //nolint:nilnil // this kind has no document type to file as
	}

	docType, err := s.documentTypes.GetByCode(ctx, documenttypeports.GetDocumentTypeByCodeRequest{
		Code:       code,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			cache[kind] = nil

			return nil, nil //nolint:nilnil // the tenant has no document type with this code
		}

		return nil, err
	}
	cache[kind] = docType

	return docType, nil
}
