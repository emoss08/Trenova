package captureservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	maxFileNameLength   = 200
	filingFailedMessage = "The document could not be filed. Try again, or file it from intake."
	itemActionFiled     = "item.filed"
	itemActionEdited    = "items.edited"
)

// coverSheetByID reads a cover sheet, reporting whether it still exists.
func (s *Service) coverSheetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*capture.CaptureCoverSheet, bool, error) {
	sheet, err := s.coverSheets.GetByID(ctx, repositories.GetCaptureCoverSheetByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return sheet, true, nil
}

// assembleItem joins an item's pages, rotated as a person left them, into
// one PDF.
func (s *Service) assembleItem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *capture.CaptureItem,
	pages []*capture.CapturePage,
) ([]byte, error) {
	byID := make(map[pulid.ID]*capture.CapturePage, len(pages))
	for _, page := range pages {
		byID[page.ID] = page
	}

	parts := make([]services.CaptureAssemblyPage, 0, len(item.PageIDs))
	for _, id := range item.PageIDs {
		page, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("page %s is not in the batch", id)
		}

		data, err := s.getObject(ctx, tenantInfo, page.StoragePath)
		if err != nil {
			return nil, err
		}
		parts = append(parts, services.CaptureAssemblyPage{PDF: data, Rotation: page.Rotation})
	}

	return s.assembler.Assemble(ctx, parts)
}

// visibleBatch is a batch the person may act on: their own, or anybody's when
// their data scope reaches past their own records.
func (s *Service) visibleBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	op permission.Operation,
	req *repositories.GetCaptureBatchByIDRequest,
) (*capture.CaptureBatch, error) {
	result, err := s.require(ctx, tenantInfo, permission.ResourceCaptureBatch, op)
	if err != nil {
		return nil, err
	}

	req.TenantInfo = tenantInfo
	batch, err := s.batches.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if batch.UserID != tenantInfo.UserID && !seesEveryone(result) {
		return nil, errortypes.NewNotFoundError("Capture batch not found")
	}

	return batch, nil
}

// GetBatch is one batch with its pages and items, for the intake queue.
func (s *Service) GetBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
) (*capture.CaptureBatch, error) {
	return s.visibleBatch(
		ctx,
		tenantInfo,
		permission.OpRead,
		&repositories.GetCaptureBatchByIDRequest{
			ID:           batchID,
			IncludePages: true,
			IncludeItems: true,
		},
	)
}

type ListBatchesInput struct {
	TenantInfo pagination.TenantInfo    `json:"-"`
	Filter     *pagination.QueryOptions `json:"-"`
	Cursor     pagination.CursorInfo    `json:"-"`
	Statuses   []capture.BatchStatus    `json:"statuses"`
	Source     capture.Source           `json:"source"`
	// Mine narrows to the caller's own batches even when they could see
	// everybody's.
	Mine       bool     `json:"mine"`
	TargetType string   `json:"targetType"`
	TargetID   pulid.ID `json:"targetId"`
}

// ListBatches is the intake queue. A person whose data scope is their own
// records sees only their own batches, whatever they ask for.
func (s *Service) ListBatches(
	ctx context.Context,
	in *ListBatchesInput,
) (*pagination.CursorListResult[*capture.CaptureBatch], error) {
	result, err := s.require(ctx, in.TenantInfo, permission.ResourceCaptureBatch, permission.OpRead)
	if err != nil {
		return nil, err
	}

	req := &repositories.ListCaptureBatchesRequest{
		Filter:     in.Filter,
		Cursor:     in.Cursor,
		Statuses:   in.Statuses,
		Source:     in.Source,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
	}
	if in.Mine || !seesEveryone(result) {
		req.UserID = in.TenantInfo.UserID
	}

	return s.batches.ListCursor(ctx, req)
}

type PageContentKind string

const (
	PageContentPDF       = PageContentKind("pdf")
	PageContentThumbnail = PageContentKind("thumbnail")
)

// PageContent is one page's bytes, for showing it. Pages are encrypted at
// rest, so they are served through the API rather than by presigned link.
type PageContent struct {
	ContentType string
	Data        []byte
}

func (s *Service) PageContent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pageID pulid.ID,
	kind PageContentKind,
) (*PageContent, error) {
	page, err := s.pages.GetByID(
		ctx,
		repositories.GetCapturePageByIDRequest{ID: pageID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if _, err = s.visibleBatch(
		ctx,
		tenantInfo,
		permission.OpRead,
		&repositories.GetCaptureBatchByIDRequest{
			ID: page.BatchID,
		},
	); err != nil {
		return nil, err
	}

	key, contentType := page.StoragePath, capture.PageContentType
	if kind == PageContentThumbnail {
		if page.ThumbnailPath == "" {
			return nil, errortypes.NewNotFoundError("This page has no thumbnail yet")
		}
		key, contentType = page.ThumbnailPath, thumbnailContentType
	}

	data, err := s.getObject(ctx, tenantInfo, key)
	if err != nil {
		return nil, err
	}

	return &PageContent{ContentType: contentType, Data: data}, nil
}

// ItemLayout is one document in a person's edit of a batch.
type ItemLayout struct {
	PageIDs []pulid.ID `json:"pageIds"`
}

type EditItemsInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	BatchID    pulid.ID              `json:"-"`
	// Version is the batch version the person was looking at. An edit made
	// against an older split is refused rather than merged.
	Version int64        `json:"version"`
	Items   []ItemLayout `json:"items"`
	// Rotations set each page's clockwise rotation in degrees.
	Rotations map[pulid.ID]int `json:"rotations"`
}

// EditItems replaces how a batch's open pages divide into documents: split,
// merge, reorder, drop. Filed items are fixed and their pages cannot move.
func (s *Service) EditItems(
	ctx context.Context,
	in *EditItemsInput,
) (*capture.CaptureBatch, error) {
	batch, err := s.visibleBatch(
		ctx,
		in.TenantInfo,
		permission.OpUpdate,
		&repositories.GetCaptureBatchByIDRequest{
			ID:           in.BatchID,
			IncludePages: true,
			IncludeItems: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if !batch.Status.Editable() {
		return nil, errortypes.NewConflictError("This batch can no longer be edited")
	}
	if batch.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this batch; reload it and try again")
	}

	items, err := layoutItems(batch, in.Items)
	if err != nil {
		return nil, err
	}

	pages, err := applyRotations(batch.Pages, in.Rotations)
	if err != nil {
		return nil, err
	}

	tenantInfo := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
	var updated *capture.CaptureBatch
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		for _, page := range pages {
			if _, pageErr := s.pages.Update(txCtx, page); pageErr != nil {
				return pageErr
			}
		}
		if replaceErr := s.items.ReplaceOpen(txCtx, &repositories.ReplaceOpenCaptureItemsRequest{
			BatchID:    batch.ID,
			TenantInfo: tenantInfo,
			Items:      items,
		}); replaceErr != nil {
			return replaceErr
		}

		all, listErr := s.items.ListByBatch(txCtx, repositories.ListCaptureItemsRequest{
			BatchID:    batch.ID,
			TenantInfo: tenantInfo,
		})
		if listErr != nil {
			return listErr
		}

		batch.Pages, batch.Items = nil, nil
		recount(batch, all)
		var updateErr error
		updated, updateErr = s.batches.Update(txCtx, batch)

		return updateErr
	})
	if err != nil {
		return nil, err
	}

	s.publishBatch(ctx, updated, itemActionEdited)

	return s.GetBatch(ctx, in.TenantInfo, updated.ID)
}

// layoutItems validates a person's split and builds the items it describes.
// An item whose pages are exactly an existing open item's keeps that item's
// suggestion; one carved out of an item keeps the suggestion of the item its
// first page came from, so splitting a routed stack does not lose the route.
func layoutItems(
	batch *capture.CaptureBatch,
	layouts []ItemLayout,
) ([]*capture.CaptureItem, error) {
	onBatch := make(map[pulid.ID]bool, len(batch.Pages))
	for _, page := range batch.Pages {
		onBatch[page.ID] = true
	}

	locked := make(map[pulid.ID]bool)
	origin := make(map[pulid.ID]*capture.CaptureItem)
	position := 0
	for _, item := range batch.Items {
		position = max(position, item.Position)
		if item.Status.Open() {
			for _, id := range item.PageIDs {
				origin[id] = item
			}

			continue
		}
		if item.Status == capture.ItemDiscarded {
			continue
		}
		for _, id := range item.PageIDs {
			locked[id] = true
		}
	}

	seen := make(map[pulid.ID]bool)
	items := make([]*capture.CaptureItem, 0, len(layouts))
	for i, layout := range layouts {
		if len(layout.PageIDs) == 0 {
			continue
		}
		for _, id := range layout.PageIDs {
			switch {
			case !onBatch[id]:
				return nil, errortypes.NewValidationError(fmt.Sprintf("items[%d].pageIds", i),
					errortypes.ErrInvalid, "A page is not part of this batch")
			case locked[id]:
				return nil, errortypes.NewValidationError(fmt.Sprintf("items[%d].pageIds", i),
					errortypes.ErrInvalid, "A page already filed cannot be moved")
			case seen[id]:
				return nil, errortypes.NewValidationError(fmt.Sprintf("items[%d].pageIds", i),
					errortypes.ErrInvalid, "A page can be in only one document")
			}
			seen[id] = true
		}

		position++
		item := &capture.CaptureItem{
			ID:             pulid.MustNew("citm_"),
			OrganizationID: batch.OrganizationID,
			BusinessUnitID: batch.BusinessUnitID,
			BatchID:        batch.ID,
			Position:       position,
			Status:         capture.ItemProposed,
			PageIDs:        slices.Clone(layout.PageIDs),
		}
		if from := origin[layout.PageIDs[0]]; from != nil {
			inherit(item, from)
		}
		items = append(items, item)
	}

	return items, nil
}

func inherit(item, from *capture.CaptureItem) {
	if from.SuggestionSource != "" {
		confidence := 0.0
		if from.SuggestionConfidence != nil {
			confidence = *from.SuggestionConfidence
		}
		item.Suggest(from.Suggestion(), from.SuggestionSource, confidence, from.SuggestionReason)
	}
	item.CoverSheetID = from.CoverSheetID
	item.DetectedKind = from.DetectedKind
	if slices.Equal(item.PageIDs, from.PageIDs) {
		item.ID = from.ID
		item.Version = from.Version
	}
}

func applyRotations(
	pages []*capture.CapturePage,
	rotations map[pulid.ID]int,
) ([]*capture.CapturePage, error) {
	changed := make([]*capture.CapturePage, 0, len(rotations))
	for id, degrees := range rotations {
		if degrees%90 != 0 {
			return nil, errortypes.NewValidationError("rotations", errortypes.ErrInvalid,
				"Pages rotate in quarter turns")
		}
		idx := slices.IndexFunc(pages, func(p *capture.CapturePage) bool { return p.ID == id })
		if idx < 0 {
			return nil, errortypes.NewValidationError("rotations", errortypes.ErrInvalid,
				"A rotated page is not part of this batch")
		}
		rotation := capture.NormalizeRotation(degrees)
		if pages[idx].Rotation == rotation {
			continue
		}
		pages[idx].Rotation = rotation
		changed = append(changed, pages[idx])
	}

	return changed, nil
}

// recount sets a batch's counts from its items and rolls its status. A
// discarded item is not counted: it is neither waiting nor filed.
func recount(batch *capture.CaptureBatch, items []*capture.CaptureItem) {
	batch.ItemCount, batch.FiledItemCount = 0, 0
	for _, item := range items {
		switch item.Status {
		case capture.ItemDiscarded:
			continue
		case capture.ItemFiled:
			batch.FiledItemCount++
		case capture.ItemProposed, capture.ItemFiling, capture.ItemFailed:
		}
		batch.ItemCount++
	}

	switch {
	case batch.ItemCount == 0:
		batch.Status = capture.BatchDiscarded
	case batch.FiledItemCount == 0:
		batch.Status = capture.BatchReady
	default:
		batch.SettleFiling()
	}
}

type FileItemInput struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ItemID         pulid.ID              `json:"-"`
	TargetType     string                `json:"targetType"`
	TargetID       pulid.ID              `json:"targetId"`
	DocumentTypeID *pulid.ID             `json:"documentTypeId"`
	// Version is the item version the person was looking at.
	Version int64 `json:"version"`
	// Automatic marks a filing the system made on the person's behalf, which
	// takes whatever version the item is at.
	Automatic bool `json:"-"`
}

// FileItem makes an item a document on a record, as the person filing it.
// The document goes through the same upload pipeline as any other, so it is
// encrypted, checksummed, thumbnailed, read and counted toward the record's
// required paperwork exactly as an uploaded file would be.
func (s *Service) FileItem(ctx context.Context, in *FileItemInput) (*capture.CaptureItem, error) {
	item, err := s.items.GetByID(
		ctx,
		repositories.GetCaptureItemByIDRequest{ID: in.ItemID, TenantInfo: in.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	batch, err := s.visibleBatch(
		ctx,
		in.TenantInfo,
		permission.OpUpdate,
		&repositories.GetCaptureBatchByIDRequest{
			ID:           item.BatchID,
			IncludePages: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if item.Status == capture.ItemFiled || item.Status == capture.ItemFiling {
		return item, nil
	}
	if !item.Status.Open() || !batch.Status.Editable() {
		return nil, errortypes.NewConflictError("This document can no longer be filed")
	}
	if !in.Automatic && item.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this document; reload it and try again")
	}

	target := capture.Target{
		ResourceType:   in.TargetType,
		ResourceID:     &in.TargetID,
		DocumentTypeID: in.DocumentTypeID,
	}
	if err = s.checkTarget(ctx, in.TenantInfo, target, "targetType", "targetId"); err != nil {
		return nil, err
	}

	filer := in.TenantInfo.UserID
	now := timeutils.NowUnix()
	item.Status = capture.ItemFiling
	item.FiledType = target.ResourceType
	item.FiledID = target.ResourceID
	item.FiledDocTypeID = target.DocumentTypeID
	item.FiledByID = &filer
	item.FiledAt = &now
	item.FailureMessage = ""
	if item, err = s.items.Update(ctx, item); err != nil {
		return nil, err
	}

	sessionID, err := s.stageFiling(ctx, in.TenantInfo, batch, item)
	if err != nil {
		s.l.Error(
			"could not stage a capture filing",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)

		return s.RecordFilingFailed(ctx, in.TenantInfo, item.ID, filingFailedMessage)
	}

	item.UploadSessionID = &sessionID
	if item, err = s.items.Update(ctx, item); err != nil {
		return nil, err
	}

	if s.workflows != nil && s.workflows.Enabled() {
		if err = s.startFiling(ctx, in.TenantInfo, item, sessionID); err != nil {
			s.l.Error(
				"could not start a capture filing",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)

			return s.RecordFilingFailed(ctx, in.TenantInfo, item.ID, filingFailedMessage)
		}
		s.publishBatch(ctx, batch, batchActionUpdated)

		return item, nil
	}

	session, err := s.uploads.Complete(ctx, &services.CompletionRequest{
		TenantInfo: in.TenantInfo,
		Actor:      *services.UserActor(in.TenantInfo),
		SessionID:  sessionID,
	})
	if err != nil || session.DocumentID == nil {
		s.l.Error(
			"could not finalize a capture filing",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)

		return s.RecordFilingFailed(ctx, in.TenantInfo, item.ID, filingFailedMessage)
	}

	return s.RecordFiled(ctx, in.TenantInfo, item.ID, *session.DocumentID)
}

// stageFiling assembles the item and uploads it into a document session
// owned by the filer.
func (s *Service) stageFiling(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batch *capture.CaptureBatch,
	item *capture.CaptureItem,
) (pulid.ID, error) {
	pdf, err := s.assembleItem(ctx, tenantInfo, item, batch.Pages)
	if err != nil {
		return pulid.Nil, err
	}

	docTypeID := ""
	if item.FiledDocTypeID != nil {
		docTypeID = item.FiledDocTypeID.String()
	}

	session, err := s.uploads.CreateSession(ctx, &services.CreateSessionRequest{
		TenantInfo:        tenantInfo,
		Actor:             *services.UserActor(tenantInfo),
		ResourceID:        item.FiledID.String(),
		ResourceType:      item.FiledType,
		ProcessingProfile: string(document.ProcessingProfileCapture),
		FileName:          itemFileName(batch, item),
		FileSize:          int64(len(pdf)),
		ContentType:       capture.PageContentType,
		Description:       itemDescription(batch),
		Tags:              []string{"capture", strings.ToLower(string(batch.Source))},
		DocumentTypeID:    docTypeID,
	})
	if err != nil {
		return pulid.Nil, err
	}

	if _, err = s.uploads.UploadPart(ctx, &services.UploadPartRequest{
		TenantInfo: tenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(pdf),
		Size:       int64(len(pdf)),
	}); err != nil {
		return pulid.Nil, err
	}

	return session.ID, nil
}

// itemFileName names a filed capture the way a person would: the print job's
// own name, or when it was scanned, and which part of the stack it was.
func itemFileName(batch *capture.CaptureBatch, item *capture.CaptureItem) string {
	base := strings.TrimSpace(batch.JobName)
	if base == "" {
		base = "Scan " + time.Unix(batch.CreatedAt, 0).UTC().Format("2006-01-02 15.04")
	}
	base = strings.TrimSuffix(base, ".pdf")
	if batch.ItemCount > 1 {
		base = fmt.Sprintf("%s (%d of %d)", base, item.Position, batch.ItemCount)
	}

	return stringutils.TruncateRunes(base, maxFileNameLength) + ".pdf"
}

func itemDescription(batch *capture.CaptureBatch) string {
	if batch.Source == capture.SourcePrint {
		return "Printed to Trenova"
	}
	if batch.SourceName != "" {
		return "Scanned on " + batch.SourceName
	}

	return "Scanned into Trenova"
}

// FileWorkflowID names an item's filing run, so a retried filing finds the
// run already started.
func FileWorkflowID(itemID pulid.ID) string {
	return "capture-item-file-" + itemID.String()
}

// FileItemPayload starts the filing workflow.
type FileItemPayload struct {
	temporaltype.BasePayload
	ItemID    pulid.ID `json:"itemId"`
	SessionID pulid.ID `json:"sessionId"`
}

func (s *Service) startFiling(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *capture.CaptureItem,
	sessionID pulid.ID,
) error {
	_, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                                       FileWorkflowID(item.ID),
			TaskQueue:                                temporaltype.CaptureTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary:                            "Filing captured document " + item.ID.String(),
		},
		temporaltype.FileCaptureItemWorkflowName,
		&FileItemPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				UserID:         tenantInfo.UserID,
				Timestamp:      timeutils.NowUnix(),
			},
			ItemID:    item.ID,
			SessionID: sessionID,
		},
	)

	var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &alreadyStarted) {
		return nil
	}

	return err
}

// RecordFiled marks an item filed as the document it became and rolls the
// batch forward. Recording the same document twice is harmless.
func (s *Service) RecordFiled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
	documentID pulid.ID,
) (*capture.CaptureItem, error) {
	scope := pagination.TenantInfo{OrgID: tenantInfo.OrgID, BuID: tenantInfo.BuID}
	var filed *capture.CaptureItem
	var batch *capture.CaptureBatch

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		item, err := s.items.GetByID(
			txCtx,
			repositories.GetCaptureItemByIDRequest{ID: itemID, TenantInfo: scope},
		)
		if err != nil {
			return err
		}
		if item.Status == capture.ItemFiled {
			filed = item

			return nil
		}

		now := timeutils.NowUnix()
		item.Status = capture.ItemFiled
		item.DocumentID = &documentID
		item.FiledAt = &now
		item.FailureMessage = ""
		if filed, err = s.items.Update(txCtx, item); err != nil {
			return err
		}

		batch, err = s.settleBatch(txCtx, scope, item.BatchID)

		return err
	})
	if err != nil {
		return nil, err
	}
	if batch == nil {
		return filed, nil
	}

	if filed.CoverSheetID != nil && filed.SuggestionSource == capture.SuggestionCoverSheet {
		if markErr := s.coverSheets.MarkUsed(ctx, repositories.MarkCaptureCoverSheetUsedRequest{
			ID:         *filed.CoverSheetID,
			TenantInfo: scope,
			UsedAt:     timeutils.NowUnix(),
		}); markErr != nil {
			s.l.Warn("could not record a cover sheet's use", zap.Error(markErr))
		}
	}

	filer := tenantInfo.UserID
	if filed.FiledByID != nil {
		filer = *filed.FiledByID
	}
	s.logAudit(&services.LogActionParams{
		Resource:       permission.ResourceCaptureBatch,
		ResourceID:     batch.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         filer,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    filer,
		OrganizationID: batch.OrganizationID,
		BusinessUnitID: batch.BusinessUnitID,
	}, fmt.Sprintf("Filed captured pages as document %s on %s %s",
		documentID, filed.FiledType, filed.FiledID))
	s.publishBatch(ctx, batch, itemActionFiled)
	if filed.FiledID != nil {
		s.publish(ctx, scope, document.OwnerResource(filed.FiledType), *filed.FiledID,
			"document.created", pulid.Nil, nil)
	}

	return filed, nil
}

// RecordFilingFailed returns an item to the person with the reason, keeping
// its pages. Its upload session is abandoned; the upload service's own sweep
// expires it.
func (s *Service) RecordFilingFailed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
	message string,
) (*capture.CaptureItem, error) {
	scope := pagination.TenantInfo{OrgID: tenantInfo.OrgID, BuID: tenantInfo.BuID}
	item, err := s.items.GetByID(
		ctx,
		repositories.GetCaptureItemByIDRequest{ID: itemID, TenantInfo: scope},
	)
	if err != nil {
		return nil, err
	}
	if item.Status == capture.ItemFiled {
		return item, nil
	}

	item.Status = capture.ItemFailed
	item.FailureMessage = stringutils.TruncateRunes(message, maxFailureMessageLen)
	item.UploadSessionID = nil
	updated, err := s.items.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	if batch, settleErr := s.settleBatch(ctx, scope, item.BatchID); settleErr == nil {
		s.publishBatch(ctx, batch, batchActionUpdated)
	}

	return updated, nil
}

// settleBatch recounts a batch from its items.
func (s *Service) settleBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
) (*capture.CaptureBatch, error) {
	batch, err := s.batches.GetByID(
		ctx,
		&repositories.GetCaptureBatchByIDRequest{ID: batchID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	items, err := s.items.ListByBatch(
		ctx,
		repositories.ListCaptureItemsRequest{BatchID: batchID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}

	recount(batch, items)

	return s.batches.Update(ctx, batch)
}

type DiscardItemInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ItemID     pulid.ID              `json:"-"`
	Version    int64                 `json:"version"`
}

// DiscardItem throws away one proposed document without filing it.
func (s *Service) DiscardItem(
	ctx context.Context,
	in *DiscardItemInput,
) (*capture.CaptureBatch, error) {
	item, err := s.items.GetByID(
		ctx,
		repositories.GetCaptureItemByIDRequest{ID: in.ItemID, TenantInfo: in.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	batch, err := s.visibleBatch(
		ctx,
		in.TenantInfo,
		permission.OpDelete,
		&repositories.GetCaptureBatchByIDRequest{
			ID: item.BatchID,
		},
	)
	if err != nil {
		return nil, err
	}
	if !item.Status.Open() {
		return nil, errortypes.NewConflictError(
			"Only a document waiting to be filed can be discarded",
		)
	}
	if item.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this document; reload it and try again")
	}

	item.Status = capture.ItemDiscarded
	scope := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
	var settled *capture.CaptureBatch
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, updateErr := s.items.Update(txCtx, item); updateErr != nil {
			return updateErr
		}
		var settleErr error
		settled, settleErr = s.settleBatch(txCtx, scope, batch.ID)

		return settleErr
	})
	if err != nil {
		return nil, err
	}

	s.publishBatch(ctx, settled, batchActionUpdated)

	return settled, nil
}

type DiscardBatchInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	BatchID    pulid.ID              `json:"-"`
	Version    int64                 `json:"version"`
}

// DiscardBatch throws away everything in a batch that is not already filed.
// Filed documents are untouched: they are on their records now.
func (s *Service) DiscardBatch(
	ctx context.Context,
	in *DiscardBatchInput,
) (*capture.CaptureBatch, error) {
	batch, err := s.visibleBatch(
		ctx,
		in.TenantInfo,
		permission.OpDelete,
		&repositories.GetCaptureBatchByIDRequest{
			ID:           in.BatchID,
			IncludeItems: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if batch.Status.Terminal() {
		return batch, nil
	}
	if batch.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this batch; reload it and try again")
	}
	if slices.ContainsFunc(batch.Items, func(item *capture.CaptureItem) bool {
		return item.Status == capture.ItemFiling
	}) {
		return nil, errortypes.NewConflictError(
			"A document in this batch is being filed; wait for it to finish",
		)
	}

	discarded, err := s.discardBatch(ctx, batch, capture.BatchDiscarded)
	if err != nil {
		return nil, err
	}

	s.logAudit(&services.LogActionParams{
		Resource:       permission.ResourceCaptureBatch,
		ResourceID:     discarded.ID.String(),
		Operation:      permission.OpDelete,
		UserID:         in.TenantInfo.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    in.TenantInfo.UserID,
		OrganizationID: discarded.OrganizationID,
		BusinessUnitID: discarded.BusinessUnitID,
	}, "Discarded captured pages")

	return discarded, nil
}

// discardBatch closes every open item and settles the batch as discarded or
// expired, keeping the count of what was filed before.
func (s *Service) discardBatch(
	ctx context.Context,
	batch *capture.CaptureBatch,
	status capture.BatchStatus,
) (*capture.CaptureBatch, error) {
	scope := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
	var discarded *capture.CaptureBatch

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		items, err := s.items.ListByBatch(txCtx, repositories.ListCaptureItemsRequest{
			BatchID:    batch.ID,
			TenantInfo: scope,
		})
		if err != nil {
			return err
		}
		for _, item := range items {
			if !item.Status.Open() {
				continue
			}
			item.Status = capture.ItemDiscarded
			if _, err = s.items.Update(txCtx, item); err != nil {
				return err
			}
		}

		batch.Items, batch.Pages = nil, nil
		recount(batch, items)
		if batch.FiledItemCount == 0 || status == capture.BatchExpired {
			batch.Status = status
		} else {
			batch.Status = capture.BatchFiled
		}
		discarded, err = s.batches.Update(txCtx, batch)

		return err
	})
	if err != nil {
		return nil, err
	}

	s.publishBatch(ctx, discarded, batchActionUpdated)

	return discarded, nil
}
