package captureprocessing

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturesplit"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// ProcessBatch reads a sealed batch and divides it into items. It is safe to
// run again: pages already read are not read twice, and a batch past
// processing is left alone.
func (p *BatchProcessor) ProcessBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
	progress Progress,
) (*ProcessResult, error) {
	batch, err := p.batches.GetByID(ctx, &captureports.GetCaptureBatchByIDRequest{
		ID:           batchID,
		TenantInfo:   tenantInfo,
		IncludePages: true,
	})
	if err != nil {
		return nil, err
	}
	if batch.Status != capture.BatchSealed && batch.Status != capture.BatchProcessing {
		return &ProcessResult{BatchID: batch.ID, Skipped: true}, nil
	}

	pages := batch.Pages
	if batch.Status == capture.BatchSealed {
		batch.Pages = nil
		batch.Status = capture.BatchProcessing
		if batch, err = p.batches.Update(ctx, batch); err != nil {
			return nil, err
		}
		p.support.PublishBatch(ctx, batch, capturesupport.BatchActionUpdated)
	}

	tenantInfo.UserID = batch.UserID
	if err = p.inspectPages(ctx, tenantInfo, pages, progress); err != nil {
		return nil, err
	}

	profile := p.batchProfile(ctx, tenantInfo, batch)
	split := capturesplit.SplitPages(pages, capturesplit.RulesFor(profile))
	if err = p.markSeparators(ctx, pages, split.Separators); err != nil {
		return nil, err
	}

	items, err := p.proposeItems(ctx, tenantInfo, batch, pages, split.Proposals)
	if err != nil {
		return nil, err
	}

	if err = p.items.ReplaceOpen(ctx, &captureports.ReplaceOpenCaptureItemsRequest{
		BatchID:    batch.ID,
		TenantInfo: tenantInfo,
		Items:      items,
	}); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	batch.ItemCount = len(items)
	batch.FiledItemCount = 0
	batch.ProcessedAt = &now
	batch.Status = capture.BatchReady
	if len(items) == 0 {
		batch.Status = capture.BatchDiscarded
	}
	if batch, err = p.batches.Update(ctx, batch); err != nil {
		return nil, err
	}
	p.support.PublishBatch(ctx, batch, capturesupport.BatchActionUpdated)

	control, err := p.support.Control(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &ProcessResult{
		BatchID:  batch.ID,
		AutoFile: autoFileable(batch, items, control.CaptureAutoFileCoverSheets),
	}, nil
}

func (p *BatchProcessor) inspectPages(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pages []*capture.CapturePage,
	progress Progress,
) error {
	read := 0
	report := func() {
		read++
		if progress != nil {
			progress(read)
		}
	}
	pending := make([]int, 0, len(pages))
	for i, page := range pages {
		if page.Status != capture.PageReceived {
			report()
			continue
		}
		pending = append(pending, i)
	}
	if len(pending) == 0 {
		return ctx.Err()
	}

	err := p.service.ReadPages(ctx, tenantInfo, pages, pending, kernelports.ImagingBatch,
		func(i int) {
			pages[i] = p.service.savePage(ctx, pages[i])
			report()
		})
	if err != nil {
		return err
	}
	return ctx.Err()
}

func (p *BatchProcessor) markSeparators(
	ctx context.Context,
	pages []*capture.CapturePage,
	separators []pulid.ID,
) error {
	for _, page := range pages {
		separator := slices.Contains(separators, page.ID)
		if page.IsSeparator == separator {
			continue
		}
		page.IsSeparator = separator
		if _, err := p.pages.Update(ctx, page); err != nil {
			return fmt.Errorf("mark page %d: %w", page.Sequence, err)
		}
	}

	return nil
}

func (p *BatchProcessor) batchProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batch *capture.CaptureBatch,
) *captureprofile.CaptureProfile {
	if batch.ProfileID == nil {
		return nil
	}

	profile, err := p.profileRepo.GetByID(ctx, captureports.GetCaptureProfileByIDRequest{
		ID:         *batch.ProfileID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		p.l.Warn("could not read a batch's profile; splitting on patch codes only",
			zap.String("batchId", batch.ID.String()), zap.Error(err))

		return nil
	}

	return profile
}

// proposeItems turns the split into items and suggests where each goes, most
// trusted source first: a cover sheet, then the record the scan was started
// from, then what the document itself says.
func (p *BatchProcessor) proposeItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batch *capture.CaptureBatch,
	pages []*capture.CapturePage,
	proposals []capturesplit.Proposal,
) ([]*capture.CaptureItem, error) {
	sheets := make(map[pulid.ID]*capturecoversheet.CaptureCoverSheet)
	items := make([]*capture.CaptureItem, 0, len(proposals))

	for i, proposal := range proposals {
		item := &capture.CaptureItem{
			ID:             pulid.MustNew("citm_"),
			OrganizationID: batch.OrganizationID,
			BusinessUnitID: batch.BusinessUnitID,
			BatchID:        batch.ID,
			Position:       i + 1,
			Status:         capture.ItemProposed,
			PageIDs:        proposal.PageIDs,
			CoverSheetID:   proposal.CoverSheetID,
		}

		switch {
		case proposal.CoverSheetID != nil:
			sheet, err := p.coverSheet(ctx, tenantInfo, *proposal.CoverSheetID, sheets)
			if err != nil {
				return nil, err
			}
			if sheet != nil && sheet.Target().HasRecord() {
				item.Suggest(sheet.Target(), capture.SuggestionCoverSheet, 1, coverSheetReason)
			}
		case batch.Target().HasRecord():
			item.Suggest(batch.Target(), capture.SuggestionRequest, 1, requestReason)
		}

		if !item.Suggestion().HasRecord() {
			p.suggestFromBarcodes(ctx, tenantInfo, item, pages)
		}

		if item.SuggestedDocTypeID == nil || !item.Suggestion().HasRecord() {
			p.suggestFromContent(ctx, tenantInfo, item, pages)
		}

		items = append(items, item)
	}

	return items, nil
}

func (p *BatchProcessor) coverSheet(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	cache map[pulid.ID]*capturecoversheet.CaptureCoverSheet,
) (*capturecoversheet.CaptureCoverSheet, error) {
	if sheet, ok := cache[id]; ok {
		return sheet, nil
	}

	sheet, found, err := p.service.coverSheetByID(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}
	if !found {
		sheet = nil
	}
	cache[id] = sheet

	return sheet, nil
}

// suggestFromContent asks the document pipeline what the item is and whose it
// is, filling in only what a more trusted source left empty. It never fails
// the batch: an item with no suggestion is one a person routes by hand.
func (p *BatchProcessor) suggestFromContent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *capture.CaptureItem,
	pages []*capture.CapturePage,
) {
	pdf, err := p.service.AssembleItem(ctx, tenantInfo, item, pages)
	if err != nil {
		p.l.Warn(
			"could not assemble an item to read it",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)

		return
	}

	analysis, err := p.analyzer.Analyze(ctx, &captureports.CaptureAnalysisRequest{
		TenantInfo: tenantInfo,
		FileName:   item.ID.String() + ".pdf",
		PDF:        pdf,
	})
	if err != nil {
		p.l.Warn(
			"could not read an item's content",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)

		return
	}

	item.DetectedKind = analysis.Kind
	target := item.Suggestion()
	source := item.SuggestionSource
	confidence := analysis.Confidence
	if item.SuggestionConfidence != nil && source != "" {
		confidence = *item.SuggestionConfidence
	}

	if target.DocumentTypeID == nil && analysis.DocumentTypeCode != "" {
		if docTypeID := p.documentTypeByCode(
			ctx,
			tenantInfo,
			analysis.DocumentTypeCode,
		); docTypeID != nil {
			target.DocumentTypeID = docTypeID
			if source == "" {
				source = capture.SuggestionClassifier
			}
		}
	}

	if !target.HasRecord() {
		if shipmentID, ok := p.shipmentFromReferences(ctx, tenantInfo, analysis.References); ok {
			target.ResourceType = permission.ResourceShipment.String()
			target.ResourceID = &shipmentID
			source = capture.SuggestionClassifier
			confidence = min(analysis.Confidence, referenceConfidence)
		}
	}

	if source == "" {
		return
	}

	reason := item.SuggestionReason
	if source == capture.SuggestionClassifier && reason == "" {
		reason = classifierReason
	}
	item.Suggest(target, source, confidence, reason)
}

// suggestFromBarcodes routes an item to the shipment a code on its pages
// names: a PRO or bill of lading number the scanner decoded, or a QR code the
// server read. It is a reading of the document, so it is offered for a person
// to confirm, never filed on its own.
func (p *BatchProcessor) suggestFromBarcodes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *capture.CaptureItem,
	pages []*capture.CapturePage,
) {
	byID := make(map[pulid.ID]*capture.CapturePage, len(pages))
	for _, page := range pages {
		byID[page.ID] = page
	}
	seen := make(map[string]struct{})
	codes := make([]string, 0, maxReferencesTried)
	for _, id := range item.PageIDs {
		page, ok := byID[id]
		if !ok {
			continue
		}
		for _, code := range page.Markers.Codes() {
			code = strings.TrimSpace(code)
			if _, dup := seen[code]; dup || code == "" {
				continue
			}
			seen[code] = struct{}{}
			codes = append(codes, code)
		}
	}

	shipmentID, ok := p.shipmentFromReferences(ctx, tenantInfo, codes)
	if !ok {
		return
	}
	target := item.Suggestion()
	target.ResourceType = permission.ResourceShipment.String()
	target.ResourceID = &shipmentID
	item.Suggest(target, capture.SuggestionClassifier, barcodeConfidence, BarcodeReason)
}

func (p *BatchProcessor) documentTypeByCode(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	code string,
) *pulid.ID {
	docType, err := p.documentTypes.GetByCode(ctx, documenttypeports.GetDocumentTypeByCodeRequest{
		Code:       code,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil
	}

	return &docType.ID
}

// shipmentFromReferences resolves the first reference that names exactly one
// shipment. The finder already refuses a reference shared by two, so a match
// here is unambiguous.
func (p *BatchProcessor) shipmentFromReferences(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	references []string,
) (pulid.ID, bool) {
	for i, reference := range references {
		if i >= maxReferencesTried {
			break
		}

		id, ok, err := p.shipments.FindByReference(ctx, tenantInfo, reference)
		if err != nil {
			p.l.Warn("could not look up a shipment reference", zap.Error(err))

			continue
		}
		if ok {
			return id, true
		}
	}

	return pulid.Nil, false
}
