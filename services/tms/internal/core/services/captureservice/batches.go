package captureservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	secondsPerDay       = 24 * 60 * 60
	maxPatchCodeLength  = 2
	maxDeviceBarcodes   = 16
	maxBarcodeLength    = 512
	maxPrintJobBytes    = 200 << 20
	pdfMagic            = "%PDF-"
	batchActionCreated  = "created"
	batchActionUpdated  = "updated"
	batchActionReceived = "page.received"
)

// validPatchCodes are the patch sheets scanners recognise (patch 1 to 4, 6
// and T). Patch 5 does not exist.
var validPatchCodes = []string{"1", "2", "3", "4", "6", "T"}

type OpenBatchInput struct {
	// ClientKey is the device's own name for the batch. Opening twice with
	// the same key returns the same batch.
	ClientKey  string           `json:"clientKey"`
	Source     capture.Source   `json:"source"`
	RequestID  *pulid.ID        `json:"requestId"`
	ProfileID  *pulid.ID        `json:"profileId"`
	SourceName string           `json:"sourceName"`
	JobName    string           `json:"jobName"`
	Settings   capture.Settings `json:"settings"`
}

// OpenBatch starts an acquisition. A batch opened for a request inherits the
// request's destination; a print opened with no request picks up the print
// destination the person armed in the web app, if there is one.
func (s *Service) OpenBatch(
	ctx context.Context,
	principal *DevicePrincipal,
	in *OpenBatchInput,
) (*capture.CaptureBatch, error) {
	tenantInfo := principal.TenantInfo()
	control, err := s.requireEnabled(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.require(
		ctx,
		tenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpCreate,
	); err != nil {
		return nil, err
	}

	clientKey := strings.TrimSpace(in.ClientKey)
	existing, err := s.batches.GetByClientKey(ctx, repositories.GetCaptureBatchByClientKeyRequest{
		TenantInfo: tenantInfo,
		DeviceID:   principal.Device.ID,
		ClientKey:  clientKey,
	})
	if err == nil {
		return existing, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	now := timeutils.NowUnix()
	batch := &capture.CaptureBatch{
		ID:             pulid.MustNew("cbat_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		UserID:         tenantInfo.UserID,
		DeviceID:       principal.Device.ID,
		ClientKey:      clientKey,
		Source:         in.Source,
		Status:         capture.BatchReceiving,
		SourceName:     strings.TrimSpace(in.SourceName),
		JobName:        strings.TrimSpace(in.JobName),
		Settings:       in.Settings,
		RetainUntil:    now + int64(control.CaptureRetentionDays)*secondsPerDay,
	}

	request, fulfils, err := s.requestForBatch(ctx, principal, in)
	if err != nil {
		return nil, err
	}
	if fulfils {
		batch.RequestID = &request.ID
		batch.ProfileID = request.ProfileID
		batch.TargetType = request.TargetType
		targetID := request.TargetID
		batch.TargetID = &targetID
		batch.DocumentTypeID = request.DocumentTypeID
	} else if in.Source == capture.SourceScan && in.ProfileID != nil {
		profileID, found, profileErr := s.resolveProfile(ctx, tenantInfo, in.ProfileID)
		if profileErr != nil {
			return nil, profileErr
		}
		if found {
			batch.ProfileID = &profileID
		}
	}

	multiErr := errortypes.NewMultiError()
	batch.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.batches.Create(ctx, batch)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return s.batches.GetByClientKey(ctx, repositories.GetCaptureBatchByClientKeyRequest{
				TenantInfo: tenantInfo,
				DeviceID:   principal.Device.ID,
				ClientKey:  clientKey,
			})
		}

		return nil, err
	}

	if fulfils {
		s.attachRequest(ctx, request, created.ID)
	}
	s.publishBatch(ctx, created, batchActionCreated)

	return created, nil
}

// requestForBatch finds the request a new batch fulfils, reporting whether
// there is one: the request the device named, or for a print with none named,
// the destination armed for it.
func (s *Service) requestForBatch(
	ctx context.Context,
	principal *DevicePrincipal,
	in *OpenBatchInput,
) (*capture.CaptureRequest, bool, error) {
	if in.RequestID != nil {
		request, err := s.deviceRequest(ctx, principal, *in.RequestID)
		if err != nil {
			return nil, false, err
		}
		if request.Status.Terminal() {
			return nil, false, errortypes.NewConflictError("That capture request is no longer open")
		}
		if request.Mode.Source() != in.Source {
			return nil, false, errortypes.NewValidationError("source", errortypes.ErrInvalid,
				"The batch source does not match the request")
		}
		if request.BatchID != nil {
			return nil, false, errortypes.NewConflictError(
				"That capture request already has a batch",
			)
		}

		return request, true, nil
	}

	if in.Source != capture.SourcePrint {
		return nil, false, nil
	}

	open, err := s.OpenRequests(ctx, principal)
	if err != nil {
		return nil, false, err
	}
	for _, request := range open {
		if request.Mode == capture.RequestModePrint && request.BatchID == nil {
			return request, true, nil
		}
	}

	return nil, false, nil
}

// attachRequest marks a request as being fulfilled by a batch. A failure is
// logged, not returned: the pages are safe in the batch either way.
func (s *Service) attachRequest(
	ctx context.Context,
	request *capture.CaptureRequest,
	batchID pulid.ID,
) {
	request.BatchID = &batchID
	request.Transition(capture.RequestInProgress, timeutils.NowUnix())

	updated, err := s.requests.Update(ctx, request)
	if err != nil {
		s.l.Warn("could not attach a batch to its request",
			zap.String("requestId", request.ID.String()),
			zap.String("batchId", batchID.String()),
			zap.Error(err))

		return
	}
	s.signalRequest(ctx, updated, requestActionUpdated)
}

// deviceBatch is a batch this device opened.
func (s *Service) deviceBatch(
	ctx context.Context,
	principal *DevicePrincipal,
	batchID pulid.ID,
) (*capture.CaptureBatch, error) {
	batch, err := s.batches.GetByID(ctx, &repositories.GetCaptureBatchByIDRequest{
		ID:         batchID,
		TenantInfo: principal.TenantInfo(),
	})
	if err != nil {
		return nil, err
	}
	if batch.DeviceID != principal.Device.ID {
		return nil, errortypes.NewNotFoundError("Capture batch not found")
	}

	return batch, nil
}

type PutPageInput struct {
	BatchID  pulid.ID  `json:"-"`
	Sequence int       `json:"-"`
	Body     io.Reader `json:"-"`
	// DPI is the resolution the page was scanned at, as the device reports it.
	DPI            int      `json:"dpi"`
	PatchCode      string   `json:"patchCode"`
	DeviceBarcodes []string `json:"deviceBarcodes"`
}

// PutPage stores one scanned page. It is idempotent on the page's sequence:
// the same page sent again returns the page already stored, and a different
// page sent for a sequence already taken is refused rather than overwriting
// it.
func (s *Service) PutPage(
	ctx context.Context,
	principal *DevicePrincipal,
	in *PutPageInput,
) (*capture.CapturePage, error) {
	batch, err := s.deviceBatch(ctx, principal, in.BatchID)
	if err != nil {
		return nil, err
	}
	if batch.Source != capture.SourceScan {
		return nil, errortypes.NewValidationError("source", errortypes.ErrInvalid,
			"A print job is uploaded whole, not a page at a time")
	}
	if in.Sequence < 1 || in.Sequence > capture.MaxBatchPages {
		return nil, errortypes.NewValidationError("sequence", errortypes.ErrInvalid,
			"Page sequence must be between 1 and {0}", capture.MaxBatchPages)
	}

	markers, err := deviceMarkers(in.PatchCode, in.DeviceBarcodes)
	if err != nil {
		return nil, err
	}

	data, err := readBounded(in.Body, capture.MaxPageBytes)
	if err != nil {
		return nil, err
	}
	if err = s.checkPDF(ctx, data, 1); err != nil {
		return nil, err
	}

	tenantInfo := principal.TenantInfo()
	existing, err := s.pages.GetBySequence(ctx, repositories.GetCapturePageBySequenceRequest{
		BatchID:    batch.ID,
		TenantInfo: tenantInfo,
		Sequence:   in.Sequence,
	})
	checksum := hashutils.SHA256BytesHex(data)
	switch {
	case err == nil:
		return samePage(existing, checksum)
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}

	if !batch.AcceptsPages() {
		return nil, errortypes.NewConflictError("This batch is sealed and takes no more pages")
	}

	page, err := s.storePage(ctx, batch, &storePageInput{
		sequence: in.Sequence,
		data:     data,
		checksum: checksum,
		dpi:      in.DPI,
		markers:  markers,
	})
	if err != nil {
		return nil, err
	}

	s.publishBatch(ctx, batch, batchActionReceived)

	return page, nil
}

func samePage(existing *capture.CapturePage, checksum string) (*capture.CapturePage, error) {
	if existing.ChecksumSHA256 != checksum {
		return nil, errortypes.NewConflictError(
			"A different page is already stored at sequence {0}", existing.Sequence)
	}

	return existing, nil
}

type storePageInput struct {
	sequence int
	data     []byte
	checksum string
	dpi      int
	markers  capture.PageMarkers
}

// storePage writes a page's bytes and its row, counting it on the batch when
// it is new.
func (s *Service) storePage(
	ctx context.Context,
	batch *capture.CaptureBatch,
	in *storePageInput,
) (*capture.CapturePage, error) {
	tenantInfo := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
	key := pageKey(batch.OrganizationID, batch.ID, in.sequence, in.checksum)
	if err := s.putObject(ctx, tenantInfo, key, capture.PageContentType, in.data); err != nil {
		return nil, err
	}

	page, inserted, err := s.pages.Insert(ctx, &capture.CapturePage{
		OrganizationID: batch.OrganizationID,
		BusinessUnitID: batch.BusinessUnitID,
		BatchID:        batch.ID,
		Sequence:       in.sequence,
		Status:         capture.PageReceived,
		StoragePath:    key,
		ChecksumSHA256: in.checksum,
		ByteSize:       int64(len(in.data)),
		ContentType:    capture.PageContentType,
		DPI:            in.dpi,
		Markers:        in.markers,
	})
	if err != nil {
		return nil, err
	}
	if !inserted {
		return samePage(page, in.checksum)
	}

	if err = s.batches.IncrementReceived(ctx, repositories.IncrementCaptureBatchPagesRequest{
		ID:         batch.ID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	return page, nil
}

func deviceMarkers(patchCode string, barcodes []string) (capture.PageMarkers, error) {
	markers := capture.PageMarkers{}

	code := strings.ToUpper(strings.TrimSpace(patchCode))
	if code != "" {
		if len(code) > maxPatchCodeLength || !slices.Contains(validPatchCodes, code) {
			return markers, errortypes.NewValidationError("patchCode", errortypes.ErrInvalid,
				"Patch code must be 1, 2, 3, 4, 6 or T")
		}
		markers.PatchCode = code
	}

	if len(barcodes) > maxDeviceBarcodes {
		return markers, errortypes.NewValidationError("deviceBarcodes", errortypes.ErrInvalid,
			"A page may carry at most {0} barcodes", maxDeviceBarcodes)
	}
	for _, barcode := range barcodes {
		value := strings.TrimSpace(barcode)
		if value == "" {
			continue
		}
		if len(value) > maxBarcodeLength {
			return markers, errortypes.NewValidationError("deviceBarcodes", errortypes.ErrInvalid,
				"A barcode may be at most {0} characters", maxBarcodeLength)
		}
		markers.DeviceBarcodes = append(markers.DeviceBarcodes, value)
	}

	return markers, nil
}

// readBounded reads a request body, refusing one over the limit rather than
// silently cutting it short.
func readBounded(body io.Reader, limit int64) ([]byte, error) {
	if body == nil {
		return nil, errortypes.NewValidationError(
			"body",
			errortypes.ErrRequired,
			"The page is empty",
		)
	}

	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errortypes.NewValidationError("body", errortypes.ErrInvalid,
			"The upload is larger than {0} bytes", limit)
	}
	if len(data) == 0 {
		return nil, errortypes.NewValidationError(
			"body",
			errortypes.ErrRequired,
			"The page is empty",
		)
	}

	return data, nil
}

// checkPDF confirms bytes are a readable PDF, and when pages is positive, one
// with exactly that many pages. Content is sniffed, never taken from a header.
func (s *Service) checkPDF(ctx context.Context, data []byte, pages int) error {
	if !bytes.HasPrefix(data, []byte(pdfMagic)) {
		return errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The upload is not a PDF",
		)
	}

	count, err := s.assembler.PageCount(ctx, data)
	if err != nil {
		s.l.Info("refused an unreadable capture PDF", zap.Error(err))

		return errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The PDF could not be read",
		)
	}
	if pages > 0 && count != pages {
		return errortypes.NewValidationError("body", errortypes.ErrInvalid,
			"A scanned page must be a PDF of exactly one page")
	}
	if count < 1 || count > capture.MaxBatchPages {
		return errortypes.NewValidationError("body", errortypes.ErrInvalid,
			"A PDF must have between 1 and {0} pages", capture.MaxBatchPages)
	}

	return nil
}

type PutPrintJobInput struct {
	BatchID pulid.ID  `json:"-"`
	Body    io.Reader `json:"-"`
}

// PutPrintJob stores a whole print job and seals the batch. The job is split
// into its pages here so a printed document is edited, split and filed like a
// scanned one, and each page keeps its own text.
func (s *Service) PutPrintJob(
	ctx context.Context,
	principal *DevicePrincipal,
	in *PutPrintJobInput,
) (*capture.CaptureBatch, error) {
	batch, err := s.deviceBatch(ctx, principal, in.BatchID)
	if err != nil {
		return nil, err
	}
	if batch.Source != capture.SourcePrint {
		return nil, errortypes.NewValidationError("source", errortypes.ErrInvalid,
			"Only a print batch takes a print job")
	}
	if !batch.AcceptsPages() {
		return batch, nil
	}

	data, err := readBounded(in.Body, maxPrintJobBytes)
	if err != nil {
		return nil, err
	}
	if err = s.checkPDF(ctx, data, 0); err != nil {
		return nil, err
	}

	pages, err := s.assembler.Split(ctx, data)
	if err != nil {
		s.l.Info("refused a print job that would not split", zap.Error(err))

		return nil, errortypes.NewValidationError("body", errortypes.ErrInvalid,
			"The print job could not be split into pages")
	}

	checksums := make([]string, 0, len(pages))
	for i, pageData := range pages {
		checksum := hashutils.SHA256BytesHex(pageData)
		if _, err = s.storePage(ctx, batch, &storePageInput{
			sequence: i + 1,
			data:     pageData,
			checksum: checksum,
		}); err != nil {
			return nil, fmt.Errorf("store printed page %d: %w", i+1, err)
		}
		checksums = append(checksums, checksum)
	}

	return s.SealBatch(ctx, principal, &SealBatchInput{
		BatchID:        batch.ID,
		PageCount:      len(pages),
		ManifestDigest: ManifestDigest(checksums),
	})
}

type SealBatchInput struct {
	BatchID   pulid.ID `json:"-"`
	PageCount int      `json:"pageCount"`
	// ManifestDigest is SHA-256 over the pages' own SHA-256 hex digests in
	// sequence order, one per line. It proves the server holds exactly the
	// pages the device captured, in the order it captured them.
	ManifestDigest string `json:"manifestDigest"`
}

// ManifestDigest is the digest a device sends when it seals a batch.
func ManifestDigest(pageChecksums []string) string {
	h := sha256.New()
	for _, checksum := range pageChecksums {
		h.Write([]byte(checksum))
		h.Write([]byte{'\n'})
	}

	return hex.EncodeToString(h.Sum(nil))
}

// SealBatch closes a batch to new pages and starts reading it. Sealing an
// already sealed batch returns it, so a retried seal is harmless.
func (s *Service) SealBatch(
	ctx context.Context,
	principal *DevicePrincipal,
	in *SealBatchInput,
) (*capture.CaptureBatch, error) {
	batch, err := s.deviceBatch(ctx, principal, in.BatchID)
	if err != nil {
		return nil, err
	}
	if !batch.AcceptsPages() {
		return batch, nil
	}
	if in.PageCount < 0 || in.PageCount > capture.MaxBatchPages {
		return nil, errortypes.NewValidationError("pageCount", errortypes.ErrInvalid,
			"Page count must be between 0 and {0}", capture.MaxBatchPages)
	}

	tenantInfo := principal.TenantInfo()
	pages, err := s.pages.ListByBatch(ctx, repositories.ListCapturePagesRequest{
		BatchID:    batch.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = verifyManifest(pages, in); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	batch.ExpectedPageCount = in.PageCount
	batch.ReceivedPageCount = len(pages)
	batch.ManifestDigest = in.ManifestDigest
	batch.SealedAt = &now
	batch.Status = capture.BatchSealed
	if in.PageCount == 0 {
		batch.Status = capture.BatchDiscarded
	}

	sealed, err := s.batches.Update(ctx, batch)
	if err != nil {
		return nil, err
	}

	s.completeRequest(ctx, sealed)
	s.publishBatch(ctx, sealed, batchActionUpdated)
	if sealed.Status == capture.BatchSealed {
		s.startProcessing(ctx, sealed)
	}

	return sealed, nil
}

// verifyManifest confirms the server holds pages 1 to N and nothing else, and
// that they are the pages the device says it captured.
func verifyManifest(pages []*capture.CapturePage, in *SealBatchInput) error {
	if len(pages) != in.PageCount {
		return errortypes.NewConflictError(
			"The batch has {0} of the {1} pages; upload the rest before sealing",
			len(pages), in.PageCount)
	}

	checksums := make([]string, 0, len(pages))
	for i, page := range pages {
		if page.Sequence != i+1 {
			return errortypes.NewConflictError("Page {0} is missing", i+1)
		}
		checksums = append(checksums, page.ChecksumSHA256)
	}

	if in.PageCount > 0 &&
		!strings.EqualFold(strings.TrimSpace(in.ManifestDigest), ManifestDigest(checksums)) {
		return errortypes.NewConflictError(
			"The pages stored do not match the pages the device captured")
	}

	return nil
}

// completeRequest closes the request a batch fulfilled.
func (s *Service) completeRequest(ctx context.Context, batch *capture.CaptureBatch) {
	if batch.RequestID == nil {
		return
	}

	request, err := s.requests.GetByID(ctx, repositories.GetCaptureRequestByIDRequest{
		ID:         *batch.RequestID,
		TenantInfo: pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID},
	})
	if err != nil {
		s.l.Warn("could not read the request a batch fulfilled",
			zap.String("batchId", batch.ID.String()), zap.Error(err))

		return
	}

	next := capture.RequestCompleted
	if batch.Status == capture.BatchDiscarded {
		next = capture.RequestFailed
		request.FailureCode = capture.FailureFeederEmpty
	}
	if !request.Transition(next, timeutils.NowUnix()) {
		return
	}

	updated, err := s.requests.Update(ctx, request)
	if err != nil {
		s.l.Warn("could not complete a capture request",
			zap.String("requestId", request.ID.String()), zap.Error(err))

		return
	}
	s.signalRequest(ctx, updated, requestActionUpdated)
}

// startProcessing hands a sealed batch to the worker. Without one, or when the
// start fails, the batch stays sealed and the reconcile sweep starts it later:
// the pages are already safe, so there is nothing to fail back to the device.
func (s *Service) startProcessing(ctx context.Context, batch *capture.CaptureBatch) {
	if s.workflows == nil || !s.workflows.Enabled() {
		return
	}

	_, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                                       ProcessWorkflowID(batch.ID),
			TaskQueue:                                temporaltype.CaptureTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary:                            "Reading captured batch " + batch.ID.String(),
		},
		temporaltype.ProcessCaptureBatchWorkflowName,
		&ProcessBatchPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: batch.OrganizationID,
				BusinessUnitID: batch.BusinessUnitID,
				UserID:         batch.UserID,
				Timestamp:      timeutils.NowUnix(),
			},
			BatchID: batch.ID,
		},
	)
	if err == nil {
		return
	}

	var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
	if !errors.As(err, &alreadyStarted) {
		s.l.Warn("could not start capture batch processing; the reconcile sweep will retry",
			zap.String("batchId", batch.ID.String()), zap.Error(err))
	}
}

// ProcessWorkflowID names a batch's processing run, so a second start for the
// same batch finds the first rather than reading it twice.
func ProcessWorkflowID(batchID pulid.ID) string {
	return "capture-batch-process-" + batchID.String()
}

// ProcessBatchPayload starts the processing workflow.
type ProcessBatchPayload struct {
	temporaltype.BasePayload
	BatchID pulid.ID `json:"batchId"`
}

// publishBatch tells the intake queue a batch changed. It carries no entity:
// who may see a batch depends on their data scope, which the queue's own
// query applies when it refetches.
func (s *Service) publishBatch(ctx context.Context, batch *capture.CaptureBatch, action string) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  batch.OrganizationID,
		BuID:   batch.BusinessUnitID,
		UserID: batch.UserID,
	}
	s.publish(ctx, tenantInfo, permission.ResourceCaptureBatch, batch.ID, action, pulid.Nil, nil)
}
