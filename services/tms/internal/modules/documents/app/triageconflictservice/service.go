package triageconflictservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	duplicatePrefix = "dup:"
	overlapPrefix   = "load:"
	moneyPlaces     = 2
	saveAttempts    = 3
)

var resolutionOrder = []inboundmessage.ConflictResolution{
	inboundmessage.ConflictReplaceExisting,
	inboundmessage.ConflictKeepBoth,
	inboundmessage.ConflictSkipDuplicate,
	inboundmessage.ConflictConfirmNew,
}

func Options(kind inboundmessage.ConflictKind) []inboundmessage.ConflictResolution {
	options := make([]inboundmessage.ConflictResolution, 0, 2)
	for _, resolution := range resolutionOrder {
		if resolution.FitsKind(kind) {
			options = append(options, resolution)
		}
	}

	return options
}

type shipmentDocuments interface {
	GetByResourceID(
		ctx context.Context,
		req *documentports.GetDocumentsByResourceRequest,
	) ([]*document.Document, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Messages      inboundmessageports.InboundMessageRepository
	Store         triageports.TriageMessageRepository
	Documents     documentports.DocumentRepository
	Organizations tenantports.OrganizationRepository
	Activities    inboundmessageports.TriageActivityRecorder
	Notifier      inboundmessageports.TriageMessageNotifier
	Contents      documentports.DocumentContentService `optional:"true"`
	Overlaps      triageports.TenderOverlapFinder      `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	messages   inboundmessageports.InboundMessageRepository
	store      triageports.TriageMessageRepository
	documents  shipmentDocuments
	activities inboundmessageports.TriageActivityRecorder
	notifier   inboundmessageports.TriageMessageNotifier
	contents   documentports.DocumentContentService
	overlaps   triageports.TenderOverlapFinder
	clock      *triagesupport.Clock
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:          p.Logger.Named("service.triage-conflict"),
		messages:   p.Messages,
		store:      p.Store,
		documents:  p.Documents,
		activities: p.Activities,
		notifier:   p.Notifier,
		contents:   p.Contents,
		overlaps:   p.Overlaps,
		clock:      triagesupport.NewClock(p.Organizations, nil),
	}
}

var _ triageports.TriageConflictChecker = (*Service)(nil)

func tenantOf(message *inboundmessage.InboundMessage) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID}
}

func (s *Service) CheckConflicts(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) ([]inboundmessage.Conflict, error) {
	detected, err := s.detect(ctx, message)
	if err != nil {
		return nil, err
	}

	merged := Merge(message.Conflicts, detected)
	if SameConflicts(message.Conflicts, merged) {
		return merged, nil
	}

	if err = s.store.SaveTriageFields(ctx, &triageports.SaveTriageFieldsRequest{
		TenantInfo: tenantOf(message),
		MessageID:  message.ID,
		Version:    message.Version,
		Conflicts:  &merged,
	}); err != nil {
		return nil, err
	}
	message.Version++
	message.Conflicts = merged

	return merged, nil
}

type CheckMessageRequest struct {
	TenantInfo pagination.TenantInfo
	MessageID  pulid.ID
}

func (s *Service) CheckMessage(ctx context.Context, req CheckMessageRequest) error {
	for attempt := range saveAttempts {
		message, err := s.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
			ID:                 req.MessageID,
			TenantInfo:         req.TenantInfo,
			IncludeAttachments: true,
		})
		if err != nil {
			return err
		}
		if !message.NeedsReview() {
			return nil
		}

		previous := message.Version
		if _, err = s.CheckConflicts(ctx, message); err != nil {
			if errors.Is(err, triageports.ErrMessageChanged) && attempt < saveAttempts-1 {
				continue
			}

			return err
		}
		if message.Version != previous {
			s.notifier.MessagesChanged(ctx, message)
		}

		return nil
	}

	return triageports.ErrMessageChanged
}

func (s *Service) detect(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) ([]inboundmessage.Conflict, error) {
	conflicts := make([]inboundmessage.Conflict, 0, 2)

	duplicates, err := s.duplicates(ctx, message)
	if err != nil {
		return nil, err
	}
	conflicts = append(conflicts, duplicates...)

	overlaps, err := s.overlapping(ctx, message)
	if err != nil {
		return nil, err
	}

	return append(conflicts, overlaps...), nil
}

func (s *Service) duplicates(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) ([]inboundmessage.Conflict, error) {
	if message.MatchedShipmentID.IsNil() || !triage.FilesAttachments(message.Classification) {
		return nil, nil
	}

	pending := make([]*inboundmessage.InboundAttachment, 0, len(message.Attachments))
	for _, attachment := range message.Attachments {
		if _, ok := attachment.Kind.DocumentTypeCode(); ok && attachment.Readable() &&
			!attachment.Filed() {
			pending = append(pending, attachment)
		}
	}
	if len(pending) == 0 {
		return nil, nil
	}

	onFile, err := s.documents.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo:          tenantOf(message),
		ResourceID:          message.MatchedShipmentID.String(),
		ResourceType:        inboundmessageports.DocumentResourceType,
		IncludeDocumentType: true,
	})
	if err != nil {
		return nil, fmt.Errorf("read the shipment's documents: %w", err)
	}

	conflicts := make([]inboundmessage.Conflict, 0, len(pending))
	for _, attachment := range pending {
		existing := existingOfKind(onFile, attachment)
		if existing == nil {
			continue
		}

		conflict := inboundmessage.Conflict{
			ID:                 duplicatePrefix + attachment.ID.String(),
			Kind:               inboundmessage.ConflictDuplicateDocument,
			AttachmentID:       attachment.ID,
			ExistingDocumentID: existing.ID,
			ExistingShipmentID: message.MatchedShipmentID,
			ExistingFiledAt:    existing.CreatedAt,
		}
		s.attachAmounts(ctx, tenantOf(message), &conflict, attachment.DocumentID)
		conflicts = append(conflicts, conflict)
	}

	return conflicts, nil
}

func existingOfKind(
	onFile []*document.Document,
	attachment *inboundmessage.InboundAttachment,
) *document.Document {
	code, _ := attachment.Kind.DocumentTypeCode()
	for _, doc := range onFile {
		if doc.ID == attachment.DocumentID || doc.LineageID == attachment.DocumentID ||
			!doc.IsCurrentVersion || !standing(doc.Status) {
			continue
		}
		if doc.DocumentType != nil && doc.DocumentType.Code == code {
			return doc
		}
	}

	return nil
}

func standing(status document.Status) bool {
	switch status {
	case document.StatusDraft, document.StatusActive, document.StatusPending,
		document.StatusPendingApproval:
		return true
	case document.StatusArchived, document.StatusExpired, document.StatusRejected:
		return false
	default:
		return true
	}
}

func (s *Service) attachAmounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	conflict *inboundmessage.Conflict,
	newDocumentID pulid.ID,
) {
	if s.contents == nil {
		return
	}

	existing, okExisting := s.amountOf(ctx, tenantInfo, conflict.ExistingDocumentID)
	incoming, okIncoming := s.amountOf(ctx, tenantInfo, newDocumentID)
	if okExisting && okIncoming {
		conflict.ExistingAmount = existing
		conflict.NewAmount = incoming
	}
}

func (s *Service) amountOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentID pulid.ID,
) (string, bool) {
	draft, err := s.contents.GetShipmentDraft(ctx, documentID, tenantInfo)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("could not read what was extracted from a document",
				zap.String("documentId", documentID.String()), zap.Error(err))
		}

		return "", false
	}
	if draft == nil {
		return "", false
	}

	return AmountIn(draft.DraftData)
}

func AmountIn(data map[string]any) (string, bool) {
	fields, ok := data["fields"].(map[string]any)
	if !ok {
		return "", false
	}
	rate, ok := fields["rate"].(map[string]any)
	if !ok {
		return "", false
	}
	raw, ok := rate["value"].(string)
	if !ok {
		return "", false
	}

	parsed, err := decimalutils.ParseMoneyText(raw)
	if err != nil || !parsed.Valid {
		return "", false
	}

	return parsed.Decimal.StringFixed(moneyPlaces), true
}

func (s *Service) overlapping(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) ([]inboundmessage.Conflict, error) {
	if s.overlaps == nil || message.Classification != inboundmessage.ClassificationTender ||
		message.LoadDraft == nil || message.MatchedCustomerID.IsNil() ||
		message.ActionState.CreatedShipmentID.IsNotNil() {
		return nil, nil
	}

	loc, err := s.clock.Location(ctx, tenantOf(message))
	if err != nil {
		return nil, err
	}
	loads := tenderLoads(message.LoadDraft, loc, s.clock.Now())
	if len(loads) == 0 {
		return nil, nil
	}

	found, err := s.overlaps.FindOverlappingLoads(ctx, &triageports.FindOverlappingLoadsRequest{
		TenantInfo: tenantOf(message),
		CustomerID: message.MatchedCustomerID,
		Loads:      loads,
	})
	if err != nil {
		return nil, fmt.Errorf("look for loads the tender repeats: %w", err)
	}

	conflicts := make([]inboundmessage.Conflict, 0, len(found))
	for _, overlap := range found {
		conflicts = append(conflicts, inboundmessage.Conflict{
			ID:                 overlapPrefix + strconv.Itoa(overlap.LoadIndex),
			Kind:               inboundmessage.ConflictOverlappingLoad,
			ExistingShipmentID: overlap.ShipmentID,
			LoadIndex:          overlap.LoadIndex,
		})
	}

	return conflicts, nil
}

func Merge(current, detected []inboundmessage.Conflict) []inboundmessage.Conflict {
	byID := make(map[string]int, len(current))
	for i := range current {
		byID[current[i].ID] = i
	}

	merged := make([]inboundmessage.Conflict, 0, len(detected)+len(current))
	seen := make(map[string]struct{}, len(detected))
	for i := range detected {
		conflict := detected[i]
		if previous, ok := byID[conflict.ID]; ok {
			conflict.Resolution = current[previous].Resolution
			conflict.ResolvedAt = current[previous].ResolvedAt
			conflict.ResolvedBy = current[previous].ResolvedBy
		}
		merged = append(merged, conflict)
		seen[conflict.ID] = struct{}{}
	}
	for i := range current {
		if _, ok := seen[current[i].ID]; ok || current[i].Resolution == "" {
			continue
		}
		merged = append(merged, current[i])
	}

	return merged
}

func SameConflicts(a, b []inboundmessage.Conflict) bool {
	return slices.Equal(a, b)
}
