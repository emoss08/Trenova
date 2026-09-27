package captureservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const (
	maxPageFailureLength = 500
	// maxReferencesTried bounds how many numbers from a document are looked
	// up as shipment references. The first few are the ones printed large.
	maxReferencesTried  = 8
	classifierReason    = "Read from the document's text"
	coverSheetReason    = "Routed by the cover sheet in front of these pages"
	requestReason       = "Scanned from this record"
	referenceConfidence = 0.9
)

// Progress is told how many pages have been read, so a long batch keeps its
// worker's heartbeat alive.
type Progress func(pagesRead int)

// ProcessResult is what reading a batch decided.
type ProcessResult struct {
	BatchID pulid.ID `json:"batchId"`
	// AutoFile are the items that may be filed without a person: the only
	// document of a scan somebody started from a record with a known type, and
	// cover-sheet routes when the organization allows them.
	AutoFile []pulid.ID `json:"autoFile"`
	// Skipped is set when the batch was not in a state to read, which a
	// retried run finds after the first one finished.
	Skipped bool `json:"skipped"`
}

// ProcessBatch reads a sealed batch and divides it into items. It is safe to
// run again: pages already read are not read twice, and a batch past
// processing is left alone.
func (s *Service) ProcessBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID pulid.ID,
	progress Progress,
) (*ProcessResult, error) {
	batch, err := s.batches.GetByID(ctx, &repositories.GetCaptureBatchByIDRequest{
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
		if batch, err = s.batches.Update(ctx, batch); err != nil {
			return nil, err
		}
		s.publishBatch(ctx, batch, batchActionUpdated)
	}

	tenantInfo.UserID = batch.UserID
	for i, page := range pages {
		if page.Status == capture.PageReceived {
			pages[i] = s.inspectPage(ctx, tenantInfo, page)
		}
		if progress != nil {
			progress(i + 1)
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
	}

	profile := s.batchProfile(ctx, tenantInfo, batch)
	split := capture.SplitPages(pages, capture.RulesFor(profile))
	if err = s.markSeparators(ctx, pages, split.Separators); err != nil {
		return nil, err
	}

	items, err := s.proposeItems(ctx, tenantInfo, batch, pages, split.Proposals)
	if err != nil {
		return nil, err
	}

	if err = s.items.ReplaceOpen(ctx, &repositories.ReplaceOpenCaptureItemsRequest{
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
	if batch, err = s.batches.Update(ctx, batch); err != nil {
		return nil, err
	}
	s.publishBatch(ctx, batch, batchActionUpdated)

	control, err := s.control(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &ProcessResult{
		BatchID:  batch.ID,
		AutoFile: autoFileable(batch, items, control.CaptureAutoFileCoverSheets),
	}, nil
}

// autoFileable picks the items nobody needs to look at. A scan started from a
// record produced one document of a known type: the person chose where it
// goes by pressing Scan on it. A cover sheet named the record and the type,
// and the organization trusts its cover sheets. Anything the classifier
// guessed waits for a person.
func autoFileable(
	batch *capture.CaptureBatch,
	items []*capture.CaptureItem,
	trustCoverSheets bool,
) []pulid.ID {
	ready := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		suggestion := item.Suggestion()
		if !suggestion.HasRecord() || suggestion.DocumentTypeID == nil {
			continue
		}

		switch item.SuggestionSource {
		case capture.SuggestionRequest:
			if len(items) == 1 && batch.RequestID != nil {
				ready = append(ready, item.ID)
			}
		case capture.SuggestionCoverSheet:
			if trustCoverSheets {
				ready = append(ready, item.ID)
			}
		case capture.SuggestionClassifier, capture.SuggestionPerson:
		}
	}

	return ready
}

// inspectPage renders a page and records what it found. A page that cannot be
// read is kept and marked, never dropped: it is still paper somebody scanned.
func (s *Service) inspectPage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	page *capture.CapturePage,
) *capture.CapturePage {
	fail := func(message string, err error) *capture.CapturePage {
		s.l.Warn("could not read a captured page",
			zap.String("pageId", page.ID.String()), zap.Error(err))
		page.Status = capture.PageFailed
		page.FailureMessage = stringutils.TruncateRunes(message, maxPageFailureLength)

		return s.savePage(ctx, page)
	}

	data, err := s.getObject(ctx, tenantInfo, page.StoragePath)
	if err != nil {
		return fail("The page could not be read back from storage.", err)
	}

	inspection, err := s.inspector.Inspect(ctx, data)
	if errors.Is(err, services.ErrPageInspectionUnavailable) {
		page.Status = capture.PageProcessed

		return s.savePage(ctx, page)
	}
	if err != nil {
		return fail("The page could not be rendered.", err)
	}

	thumb := thumbnailKey(page.StoragePath)
	if err = s.putObject(
		ctx,
		tenantInfo,
		thumb,
		thumbnailContentType,
		inspection.Thumbnail,
	); err != nil {
		s.l.Warn(
			"could not store a page thumbnail",
			zap.String("pageId", page.ID.String()),
			zap.Error(err),
		)
	} else {
		page.ThumbnailPath = thumb
	}

	coverage := inspection.InkCoverage
	page.BlankScore = &coverage
	page.WidthPx = inspection.WidthPx
	page.HeightPx = inspection.HeightPx
	s.readCodes(ctx, tenantInfo, page, inspection.Codes)
	page.Status = capture.PageProcessed

	return s.savePage(ctx, page)
}

// readCodes looks each cover-sheet code up in this tenant. A code that does
// not resolve, or has expired, still marks the page as a divider.
func (s *Service) readCodes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	page *capture.CapturePage,
	codes []string,
) {
	for _, code := range codes {
		token, ok := capture.CoverSheetToken(code)
		if !ok {
			continue
		}

		sheet, err := s.coverSheets.GetByTokenHash(
			ctx,
			repositories.GetCaptureCoverSheetByTokenRequest{
				TenantInfo: tenantInfo,
				TokenHash:  tokenutils.Hash(token),
			},
		)
		if err != nil || sheet.IsExpired(timeutils.NowUnix()) {
			if err != nil && !errortypes.IsNotFoundError(err) {
				s.l.Warn("could not look up a cover sheet", zap.Error(err))
			}
			page.Markers.UnrecognizedCoverSheet = true

			continue
		}

		page.Markers.CoverSheetID = &sheet.ID
		page.Markers.UnrecognizedCoverSheet = false

		return
	}
}

// savePage writes what inspection found. If the write fails the page is
// returned as it is, so the batch still divides on what was learned; the
// next run reads the page again.
func (s *Service) savePage(ctx context.Context, page *capture.CapturePage) *capture.CapturePage {
	updated, err := s.pages.Update(ctx, page)
	if err != nil {
		s.l.Warn(
			"could not record a page's inspection",
			zap.String("pageId", page.ID.String()),
			zap.Error(err),
		)

		return page
	}

	return updated
}

func (s *Service) markSeparators(
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
		if _, err := s.pages.Update(ctx, page); err != nil {
			return fmt.Errorf("mark page %d: %w", page.Sequence, err)
		}
	}

	return nil
}

func (s *Service) batchProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batch *capture.CaptureBatch,
) *capture.CaptureProfile {
	if batch.ProfileID == nil {
		return nil
	}

	profile, err := s.profiles.GetByID(ctx, repositories.GetCaptureProfileByIDRequest{
		ID:         *batch.ProfileID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("could not read a batch's profile; splitting on patch codes only",
			zap.String("batchId", batch.ID.String()), zap.Error(err))

		return nil
	}

	return profile
}

// proposeItems turns the split into items and suggests where each goes, most
// trusted source first: a cover sheet, then the record the scan was started
// from, then what the document itself says.
func (s *Service) proposeItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batch *capture.CaptureBatch,
	pages []*capture.CapturePage,
	proposals []capture.Proposal,
) ([]*capture.CaptureItem, error) {
	sheets := make(map[pulid.ID]*capture.CaptureCoverSheet)
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
			sheet, err := s.coverSheet(ctx, tenantInfo, *proposal.CoverSheetID, sheets)
			if err != nil {
				return nil, err
			}
			if sheet != nil && sheet.Target().HasRecord() {
				item.Suggest(sheet.Target(), capture.SuggestionCoverSheet, 1, coverSheetReason)
			}
		case batch.Target().HasRecord():
			item.Suggest(batch.Target(), capture.SuggestionRequest, 1, requestReason)
		}

		if item.SuggestedDocTypeID == nil || !item.Suggestion().HasRecord() {
			s.suggestFromContent(ctx, tenantInfo, item, pages)
		}

		items = append(items, item)
	}

	return items, nil
}

func (s *Service) coverSheet(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	cache map[pulid.ID]*capture.CaptureCoverSheet,
) (*capture.CaptureCoverSheet, error) {
	if sheet, ok := cache[id]; ok {
		return sheet, nil
	}

	sheet, found, err := s.coverSheetByID(ctx, tenantInfo, id)
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
func (s *Service) suggestFromContent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *capture.CaptureItem,
	pages []*capture.CapturePage,
) {
	if s.analyzer == nil {
		return
	}

	pdf, err := s.assembleItem(ctx, tenantInfo, item, pages)
	if err != nil {
		s.l.Warn(
			"could not assemble an item to read it",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)

		return
	}

	analysis, err := s.analyzer.Analyze(ctx, &services.CaptureAnalysisRequest{
		TenantInfo: tenantInfo,
		FileName:   item.ID.String() + ".pdf",
		PDF:        pdf,
	})
	if err != nil {
		s.l.Warn(
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
		if docTypeID := s.documentTypeByCode(
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
		if shipmentID, ok := s.shipmentFromReferences(ctx, tenantInfo, analysis.References); ok {
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
	if source == capture.SuggestionClassifier {
		reason = classifierReason
	}
	item.Suggest(target, source, confidence, reason)
}

func (s *Service) documentTypeByCode(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	code string,
) *pulid.ID {
	docType, err := s.documentTypes.GetByCode(ctx, repositories.GetDocumentTypeByCodeRequest{
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
func (s *Service) shipmentFromReferences(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	references []string,
) (pulid.ID, bool) {
	if s.shipments == nil {
		return pulid.Nil, false
	}

	for i, reference := range references {
		if i >= maxReferencesTried {
			break
		}

		id, ok, err := s.shipments.FindByReference(ctx, tenantInfo, reference)
		if err != nil {
			s.l.Warn("could not look up a shipment reference", zap.Error(err))

			continue
		}
		if ok {
			return id, true
		}
	}

	return pulid.Nil, false
}
