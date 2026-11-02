package captureprocessing

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	BarcodeReason       = "A barcode on these pages names this shipment"
	// barcodeConfidence ranks a barcode above the text: it was printed to be
	// read by a machine, and it names exactly one shipment or none.
	barcodeConfidence = 0.95
	// maxReadCodes and maxReadCodeLength bound the codes kept from a page.
	maxReadCodes      = 8
	maxReadCodeLength = 200
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
		if suggestion.IsAssistantThread() && suggestion.HasRecord() &&
			item.SuggestionSource == capture.SuggestionRequest && batch.RequestID != nil {
			ready = append(ready, item.ID)
			continue
		}
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

func (s *Service) ReadPages(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pages []*capture.CapturePage,
	pending []int,
	priority kernelports.ImagingPriority,
	done func(i int),
) error {
	var unreadableMu sync.Mutex
	unreadable := make(map[int]error)
	_, err := s.imaging.InspectCapture(ctx, &kernelports.ImagingCaptureRequest{
		PageCount: len(pending),
		Priority:  priority,
		Pages: func() iter.Seq2[kernelports.ImagingCapturePage, error] {
			return func(yield func(kernelports.ImagingCapturePage, error) bool) {
				for position, i := range pending {
					page := pages[i]
					data, readErr := s.support.GetObject(ctx, tenantInfo, page.StoragePath)
					if readErr != nil {
						unreadableMu.Lock()
						unreadable[position] = readErr
						unreadableMu.Unlock()
						data = nil
					}
					if !yield(kernelports.ImagingCapturePage{
						Index:  position,
						Source: kernelports.ImagingBytes("application/pdf", data),
					}, nil) {
						return
					}
				}
			}
		},
	}, func(result *kernelports.ImagingCaptureResult) error {
		if result.Index < 0 || result.Index >= len(pending) {
			return fmt.Errorf(
				"the imaging service answered for page %d of %d",
				result.Index,
				len(pending),
			)
		}
		i := pending[result.Index]
		unreadableMu.Lock()
		readErr, ok := unreadable[result.Index]
		unreadableMu.Unlock()
		if ok {
			s.failPage(pages[i], "The page could not be read back from storage.", readErr)
		} else {
			s.applyInspection(ctx, tenantInfo, pages[i], result)
		}
		done(i)
		return nil
	})
	if err != nil {
		return fmt.Errorf("inspect captured pages: %w", err)
	}
	return nil
}

func (s *Service) failPage(page *capture.CapturePage, message string, err error) {
	s.l.Warn("could not read a captured page",
		zap.String("pageId", page.ID.String()), zap.Error(err))
	page.Status = capture.PageFailed
	page.FailureMessage = stringutils.TruncateRunes(message, maxPageFailureLength)
}

func (s *Service) applyInspection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	page *capture.CapturePage,
	result *kernelports.ImagingCaptureResult,
) {
	if result.Err != nil {
		s.failPage(page, "The page could not be rendered.", result.Err)
		return
	}

	thumb := capturesupport.ThumbnailKey(page.StoragePath)
	if err := s.support.PutObject(
		ctx,
		tenantInfo,
		thumb,
		capturesupport.ThumbnailContentType,
		result.Thumbnail,
	); err != nil {
		s.l.Warn(
			"could not store a page thumbnail",
			zap.String("pageId", page.ID.String()),
			zap.Error(err),
		)
	} else {
		page.ThumbnailPath = thumb
	}

	coverage := result.InkCoverage
	page.BlankScore = &coverage
	page.WidthPx = result.Width
	page.HeightPx = result.Height
	codes := make([]string, 0, len(result.Codes))
	for _, code := range result.Codes {
		codes = append(codes, code.Text)
	}
	s.readCodes(ctx, tenantInfo, page, codes)
	page.Status = capture.PageProcessed
}

// readCodes looks each cover-sheet code up in this tenant. A code that does
// not resolve, or has expired, still marks the page as a divider.
func (s *Service) readCodes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	page *capture.CapturePage,
	codes []string,
) {
	page.Markers.ReadCodes = nil
	for _, code := range codes {
		token, ok := capturecoversheet.CoverSheetToken(code)
		if !ok {
			if len(page.Markers.ReadCodes) < maxReadCodes {
				page.Markers.ReadCodes = append(
					page.Markers.ReadCodes,
					stringutils.TruncateRunes(strings.TrimSpace(code), maxReadCodeLength),
				)
			}

			continue
		}
		if page.Markers.CoverSheetID != nil {
			continue
		}

		sheet, err := s.coverSheets.GetByTokenHash(
			ctx,
			captureports.GetCaptureCoverSheetByTokenRequest{
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
