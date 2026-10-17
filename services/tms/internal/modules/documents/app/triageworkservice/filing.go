package triageworkservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triageworkjobs/triageworkcontract"
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
	attachment, message, err := s.fileTarget(ctx, req)
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

	docType, err := s.fileAs(ctx, req, attachment)
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

type PlannedRange struct {
	FromPage       int
	ToPage         int
	DocumentTypeID pulid.ID
	DocumentType   string
	FileName       string
}

type FilePlan struct {
	Message        *inboundmessage.InboundMessage
	Attachment     *inboundmessage.InboundAttachment
	After          *inboundmessage.InboundAttachment
	DocumentType   *documenttype.DocumentType
	Ranges         []PlannedRange
	Replaces       pulid.ID
	PagesUnchecked bool
}

func (s *Service) PlanFile(ctx context.Context, req *FileRequest) (*FilePlan, error) {
	attachment, message, err := s.fileTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	after := *attachment
	plan := &FilePlan{Message: message, Attachment: attachment, After: &after}
	if len(req.Pages) == 0 {
		docType, typeErr := s.fileAs(ctx, req, attachment)
		if typeErr != nil {
			return nil, typeErr
		}
		applyFiling(&after, req.ShipmentID, docType.Name, req.UserID, s.now())
		plan.DocumentType = docType
		if conflict, ok := replacingConflict(message, attachment.ID); ok {
			plan.Replaces = conflict.ExistingDocumentID
		}

		return plan, nil
	}

	if err = requireSplittable(attachment); err != nil {
		return nil, err
	}
	pageCount := attachment.PageCount
	if pageCount <= 0 {
		plan.PagesUnchecked = true
		pageCount = lastPage(req.Pages)
	}
	ranges, err := s.validateRanges(ctx, req.TenantInfo, req.Pages, pageCount)
	if err != nil {
		return nil, err
	}

	staged := make([]triageworkcontract.SplitRange, 0, len(ranges))
	plan.Ranges = make([]PlannedRange, 0, len(ranges))
	for _, rng := range ranges {
		staged = append(staged, triageworkcontract.SplitRange{
			FromPage:       rng.FromPage,
			ToPage:         rng.ToPage,
			DocumentTypeID: rng.docType.ID,
			DocumentType:   rng.docType.Name,
		})
		plan.Ranges = append(plan.Ranges, PlannedRange{
			FromPage:       rng.FromPage,
			ToPage:         rng.ToPage,
			DocumentTypeID: rng.docType.ID,
			DocumentType:   rng.docType.Name,
			FileName:       rangeFileName(attachment.FileName, rng.PageRange),
		})
	}
	applyFiling(&after, req.ShipmentID, splitLabel(staged), req.UserID, s.now())

	return plan, nil
}

func lastPage(pages []PageRange) int {
	last := 0
	for _, page := range pages {
		last = max(last, page.ToPage)
	}

	return last
}

func (s *Service) fileTarget(
	ctx context.Context,
	req *FileRequest,
) (*inboundmessage.InboundAttachment, *inboundmessage.InboundMessage, error) {
	attachment, err := s.messages.GetAttachmentByID(
		ctx,
		inboundmessageports.GetInboundAttachmentRequest{
			ID:         req.AttachmentID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if err = requireFileable(attachment); err != nil {
		return nil, nil, err
	}
	if err = s.requireShipment(ctx, req.TenantInfo, req.ShipmentID); err != nil {
		return nil, nil, err
	}

	message, err := s.message(ctx, req.TenantInfo, attachment.MessageID)
	if err != nil {
		return nil, nil, err
	}

	return attachment, message, nil
}

func applyFiling(
	attachment *inboundmessage.InboundAttachment,
	shipmentID pulid.ID,
	filedAs string,
	userID pulid.ID,
	now int64,
) {
	attachment.FiledShipmentID = shipmentID
	attachment.FiledAs = filedAs
	attachment.FiledAt = now
	attachment.FiledBy = userID
	attachment.SplitDocuments = nil
}

func clearFilingFields(attachment *inboundmessage.InboundAttachment) {
	attachment.FiledShipmentID = pulid.Nil
	attachment.FiledAs = ""
	attachment.FiledAt = 0
	attachment.FiledBy = pulid.Nil
	attachment.SplitDocuments = nil
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

func (s *Service) fileAs(
	ctx context.Context,
	req *FileRequest,
	attachment *inboundmessage.InboundAttachment,
) (*documenttype.DocumentType, error) {
	if req.DocumentTypeID.IsNotNil() {
		return s.documentType(ctx, req.TenantInfo, req.DocumentTypeID, "documentTypeId")
	}

	docType, err := s.typeForKind(
		ctx,
		req.TenantInfo,
		attachment.Kind,
		make(map[inboundmessage.AttachmentKind]*documenttype.DocumentType, 1),
	)
	if err != nil {
		return nil, err
	}
	if docType == nil {
		return nil, errortypes.NewValidationError(
			"documentTypeId", errortypes.ErrRequired, "Choose what to file it as",
		)
	}

	return docType, nil
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

	applyFiling(f.attachment, f.shipmentID, f.docType.Name, f.userID, s.now())
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
	attachment, message, err := s.unfileTarget(ctx, req)
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

type UnfilePlan struct {
	Message    *inboundmessage.InboundMessage
	Attachment *inboundmessage.InboundAttachment
	After      *inboundmessage.InboundAttachment
	Removed    []inboundmessage.SplitDocument
	Restores   pulid.ID
}

func (s *Service) PlanUnfile(ctx context.Context, req *UnfileRequest) (*UnfilePlan, error) {
	attachment, message, err := s.unfileTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	after := *attachment
	clearFilingFields(&after)
	plan := &UnfilePlan{
		Message:    message,
		Attachment: attachment,
		After:      &after,
		Removed:    attachment.SplitDocuments,
	}
	if len(attachment.SplitDocuments) == 0 {
		if conflict, ok := replacingConflict(message, attachment.ID); ok {
			plan.Restores = conflict.ExistingDocumentID
		}
	}

	return plan, nil
}

func (s *Service) unfileTarget(
	ctx context.Context,
	req *UnfileRequest,
) (*inboundmessage.InboundAttachment, *inboundmessage.InboundMessage, error) {
	attachment, err := s.messages.GetAttachmentByID(
		ctx,
		inboundmessageports.GetInboundAttachmentRequest{
			ID:         req.AttachmentID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if !attachment.Filed() {
		return nil, nil, errortypes.NewBusinessError("This file is not filed")
	}

	message, err := s.message(ctx, req.TenantInfo, attachment.MessageID)
	if err != nil {
		return nil, nil, err
	}

	return attachment, message, nil
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
