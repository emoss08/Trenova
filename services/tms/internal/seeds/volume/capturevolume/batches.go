package capturevolume

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturerequest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/google/uuid"
)

const (
	cvScanShare      = 0.82
	cvRequestShare   = 0.25
	cvPrintShare     = 0.15
	cvPrinterName    = "Trenova Printer"
	cvApplication    = "Trenova Capture"
	cvDriverVersion  = "3.1.4"
	cvReminderWindow = 7 * volumegen.Day
)

type cvKind struct {
	code     string
	detected string
	file     string
	low      int
	high     int
}

var (
	cvBOL     = cvKind{code: "BOL", detected: "bill_of_lading", file: "BOL", low: 1, high: 2}
	cvPOD     = cvKind{code: "POD", detected: "proof_of_delivery", file: "POD", low: 1, high: 2}
	cvLumper  = cvKind{detected: "lumper_receipt", file: "Lumper", low: 1, high: 1}
	cvScale   = cvKind{detected: "scale_ticket", file: "Scale", low: 1, high: 1}
	cvRateCon = cvKind{detected: "rate_confirmation", file: "RateConfirmation", low: 1, high: 3}

	cvPrintJobs = []string{
		"Microsoft Outlook - Lumper receipt.pdf",
		"Adobe Acrobat - Rate confirmation.pdf",
		"Microsoft Edge - Carrier packet",
		"Excel - Detention log.xlsx",
	}
	cvFailures = []string{
		"The scanner reported a paper jam before the stack finished.",
		"The pages could not be split into documents; the stack was left for review.",
		"The device lost its connection while uploading pages.",
	}
	cvReasons = map[string]string{
		"bill_of_lading":    "Read the PRO and shipper block on the bill of lading",
		"proof_of_delivery": "Read the PRO and a consignee signature on the delivery receipt",
		"lumper_receipt":    "Matched the receipt's date and facility to a delivery stop",
		"scale_ticket":      "Matched the scale ticket's weight and date to the load",
		"rate_confirmation": "Read the load number on the rate confirmation",
	}
)

type cvStack struct {
	source  capture.Source
	ships   []*volumegen.ShipmentRecord
	at      int64
	request bool
}

type cvDoc struct {
	kind  cvKind
	ship  *volumegen.ShipmentRecord
	pages []*capture.CapturePage
}

func (b *cvBuilder) planStacks() []*cvStack {
	f := b.g.Fake
	if len(b.devices) == 0 {
		return nil
	}
	from := max(b.now-b.retention, volumegen.Unix(b.g.Start))
	stacks := make([]*cvStack, 0, 256)
	var pending []*volumegen.ShipmentRecord
	var recent []*volumegen.ShipmentRecord
	want := 0
	flush := func() {
		if len(pending) == 0 {
			return
		}
		last := int64(0)
		for _, rec := range pending {
			last = max(last, rec.DeliveredAt)
		}
		at := last + f.Between64(2*volumegen.Hour, 3*volumegen.Day)
		if at > b.now {
			at = last + int64(float64(b.now-last)*f.Float())
		}
		if at >= from {
			stacks = append(stacks, &cvStack{
				source:  capture.SourceScan,
				ships:   pending,
				at:      at,
				request: len(pending) == 1 && f.Chance(cvRequestShare),
			})
		}
		pending, want = nil, 0
	}

	for _, rec := range b.g.Shipments {
		if rec.DeliveredAt <= 0 || rec.Status == shipmentvocab.StatusCanceled ||
			rec.DeliveredAt < from-3*volumegen.Day || rec.DeliveredAt > b.now {
			continue
		}
		recent = append(recent, rec)
		if !f.Chance(cvScanShare) {
			continue
		}
		if want == 0 {
			want = 1 + f.Weighted([]int{60, 25, 10, 5})
		}
		pending = append(pending, rec)
		if len(pending) >= want {
			flush()
		}
	}
	flush()

	prints := int(float64(len(stacks)) * cvPrintShare)
	for range prints {
		s := &cvStack{
			source: capture.SourcePrint,
			at:     b.now - f.Between64(volumegen.Minute, b.now-from),
		}
		if len(recent) > 0 && f.Chance(0.8) {
			s.ships = []*volumegen.ShipmentRecord{volumegen.Pick(f, recent)}
		}
		stacks = append(stacks, s)
	}

	sort.SliceStable(stacks, func(i, j int) bool { return stacks[i].at < stacks[j].at })
	return stacks
}

func (b *cvBuilder) build(s *cvStack) {
	f := b.g.Fake
	device := volumegen.Pick(f, b.devices)
	created := s.at
	id := volumegen.IDAt("cbat_", created)
	batch := &capture.CaptureBatch{
		ID:             id,
		BusinessUnitID: b.g.Refs.BuID,
		OrganizationID: b.g.Refs.OrgID,
		UserID:         device.entity.UserID,
		DeviceID:       device.entity.ID,
		ClientKey:      uuid.NewSHA1(uuid.NameSpaceOID, []byte(id.String())).String(),
		Source:         s.source,
		RetainUntil:    created + b.retention,
		Version:        1,
		CreatedAt:      created,
	}

	var profile *captureprofile.CaptureProfile
	if s.source == capture.SourceScan {
		batch.SourceName = device.scanner
		if len(b.profiles) > 0 {
			profile = b.profiles[0]
			if f.Chance(0.3) {
				profile = volumegen.Pick(f, b.profiles)
			}
			batch.ProfileID = volumegen.Ptr(profile.ID)
		}
		batch.Settings = b.settings(profile)
	} else {
		batch.SourceName = cvPrinterName
		batch.JobName = volumegen.Pick(f, cvPrintJobs)
		batch.Settings = capture.Settings{Application: cvApplication}
	}
	if len(s.ships) == 1 && (s.request || f.Chance(0.4)) {
		batch.TargetType = string(permission.ResourceShipment)
		batch.TargetID = volumegen.Ptr(s.ships[0].ID)
	}

	docs := b.documents(s)
	separators := profile != nil &&
		slices.Contains(profile.SeparatorStrategies, captureprofile.SeparatorPatchCode) &&
		f.Chance(0.5)
	dpi := 200
	if profile != nil {
		dpi = profile.DPI
	}
	seq := 0
	var pages []*capture.CapturePage
	for i, doc := range docs {
		if separators && i > 0 {
			seq++
			sep := b.page(batch, seq, dpi)
			sep.IsSeparator = true
			sep.BlankScore = volumegen.Ptr(0.001)
			sep.Markers = capture.PageMarkers{PatchCode: "PatchT"}
			pages = append(pages, sep)
		}
		for range f.Between(doc.kind.low, doc.kind.high) {
			seq++
			page := b.page(batch, seq, dpi)
			if doc.ship != nil && f.Chance(0.4) {
				page.Markers = capture.PageMarkers{ReadCodes: []string{doc.ship.ProNumber}}
			}
			doc.pages = append(doc.pages, page)
			pages = append(pages, page)
		}
	}
	batch.ExpectedPageCount = len(pages)
	batch.ReceivedPageCount = len(pages)

	status := b.status(created, s.source)
	updated := b.settle(batch, status, pages, docs, s)
	if s.request {
		b.request(batch, device, s.ships[0], updated)
	}

	b.statuses[batch.Status]++
	b.batch.batches = append(b.batch.batches, batch)
	b.batch.stamp(cvTableBatches, batch.ID, created, updated)
}

func (b *cvBuilder) documents(s *cvStack) []*cvDoc {
	f := b.g.Fake
	if s.source == capture.SourcePrint {
		var ship *volumegen.ShipmentRecord
		if len(s.ships) > 0 {
			ship = s.ships[0]
		}
		kind := cvRateCon
		if f.Chance(0.5) {
			kind = cvLumper
		}
		return []*cvDoc{{kind: kind, ship: ship}}
	}
	docs := make([]*cvDoc, 0, len(s.ships)*3)
	for _, ship := range s.ships {
		docs = append(docs, &cvDoc{kind: cvBOL, ship: ship}, &cvDoc{kind: cvPOD, ship: ship})
		if f.Chance(0.3) {
			docs = append(docs, &cvDoc{kind: cvLumper, ship: ship})
		}
		if f.Chance(0.12) {
			docs = append(docs, &cvDoc{kind: cvScale, ship: ship})
		}
	}

	return docs
}

func (b *cvBuilder) settings(profile *captureprofile.CaptureProfile) capture.Settings {
	s := capture.Settings{
		Protocol:      capture.ProtocolTWAIN,
		Bitness:       64,
		DPI:           300,
		PixelType:     capture.PixelBlackWhite,
		Duplex:        true,
		Feeder:        true,
		BlankDiscard:  true,
		DriverVersion: cvDriverVersion,
		Application:   cvApplication,
	}
	if profile != nil {
		s.DPI = profile.DPI
		s.PixelType = profile.PixelType
		s.Duplex = profile.Duplex
		s.Feeder = profile.UseFeeder
		s.BlankDiscard = profile.DiscardBlankPages
		s.ShowDriverUI = profile.ShowDriverUI
	}

	return s
}

func (b *cvBuilder) page(batch *capture.CaptureBatch, seq, dpi int) *capture.CapturePage {
	f := b.g.Fake
	path := fmt.Sprintf("%s/capture/%s/%04d.pdf", batch.OrganizationID, batch.ID, seq)
	page := &capture.CapturePage{
		ID:             volumegen.IDAt("cpg_", batch.CreatedAt),
		BusinessUnitID: batch.BusinessUnitID,
		OrganizationID: batch.OrganizationID,
		BatchID:        batch.ID,
		Sequence:       seq,
		Status:         capture.PageReceived,
		StoragePath:    path,
		ChecksumSHA256: digest(path),
		ByteSize:       f.Between64(60_000, 900_000),
		ContentType:    capture.PageContentType,
		WidthPx:        dpi * 17 / 2,
		HeightPx:       dpi * 11,
		DPI:            dpi,
		BlankScore:     volumegen.Ptr(0.02 + f.Float()*0.3),
		CreatedAt:      batch.CreatedAt,
	}
	switch roll := f.Intn(100); {
	case roll < 3:
		page.Rotation = 180
	case roll < 5:
		page.Rotation = 90
	}

	return page
}

func (b *cvBuilder) status(created int64, source capture.Source) capture.BatchStatus {
	f := b.g.Fake
	age := b.now - created
	switch {
	case age < 3*volumegen.Minute:
		return volumegen.Pick(f, []capture.BatchStatus{
			capture.BatchReceiving, capture.BatchSealed, capture.BatchProcessing,
		})
	case age < 2*volumegen.Day:
		weights := []int{38, 14, 43, 3, 2}
		if source == capture.SourcePrint {
			weights = []int{30, 0, 64, 4, 2}
		}
		return []capture.BatchStatus{
			capture.BatchReady, capture.BatchPartiallyFiled, capture.BatchFiled,
			capture.BatchDiscarded, capture.BatchFailed,
		}[f.Weighted(weights)]
	default:
		return []capture.BatchStatus{
			capture.BatchReady, capture.BatchPartiallyFiled, capture.BatchFiled,
			capture.BatchDiscarded, capture.BatchFailed,
		}[f.Weighted([]int{4, 6, 85, 4, 1})]
	}
}

func (b *cvBuilder) settle(
	batch *capture.CaptureBatch,
	status capture.BatchStatus,
	pages []*capture.CapturePage,
	docs []*cvDoc,
	s *cvStack,
) int64 {
	f := b.g.Fake
	created := batch.CreatedAt
	updated := created
	if status == capture.BatchReceiving {
		pages = pages[:f.Between(0, max(len(pages)-1, 0))]
		batch.ReceivedPageCount = len(pages)
	}
	b.batch.pages = append(b.batch.pages, pages...)
	defer func() {
		for _, page := range pages {
			b.batch.stamp(cvTablePages, page.ID, created, min(updated, b.now))
		}
	}()
	if status == capture.BatchReceiving {
		batch.Status = status
		return created
	}

	sealed := min(created+f.Between64(20, 4*volumegen.Minute), b.now)
	batch.SealedAt = volumegen.Ptr(sealed)
	batch.ManifestDigest = digest("manifest:" + batch.ID.String())
	updated = sealed
	if status == capture.BatchSealed {
		batch.Status = status
		return updated
	}
	for _, page := range pages {
		page.Status = capture.PageProcessed
		page.ThumbnailPath = strings.TrimSuffix(page.StoragePath, ".pdf") + ".thumb.jpg"
	}
	if status == capture.BatchProcessing {
		batch.Status = status
		return updated
	}

	processed := min(sealed+f.Between64(20, 6*volumegen.Minute), b.now)
	batch.ProcessedAt = volumegen.Ptr(processed)
	updated = processed
	if status == capture.BatchFailed {
		batch.Status = status
		batch.FailureMessage = volumegen.Pick(f, cvFailures)
		return updated
	}

	filedCount := 0
	for i, doc := range docs {
		item := b.item(batch, doc, i, processed, s)
		switch status {
		case capture.BatchFiled:
			b.file(item, doc, processed)
		case capture.BatchPartiallyFiled:
			if i == 0 || (i < len(docs)-1 && f.Chance(0.5)) {
				b.file(item, doc, processed)
			}
		case capture.BatchDiscarded:
			item.Status = capture.ItemDiscarded
		case capture.BatchReceiving, capture.BatchSealed, capture.BatchProcessing,
			capture.BatchReady, capture.BatchExpired, capture.BatchFailed:
		}
		itemUpdated := processed
		if item.Status == capture.ItemFiled {
			filedCount++
			itemUpdated = *item.FiledAt
			updated = max(updated, itemUpdated)
		}
		b.batch.items = append(b.batch.items, item)
		b.batch.stamp(cvTableItems, item.ID, processed, itemUpdated)
	}
	batch.ItemCount = len(docs)
	batch.FiledItemCount = filedCount
	batch.Status = status
	if status == capture.BatchFiled || status == capture.BatchPartiallyFiled {
		switch {
		case filedCount == len(docs):
			batch.Status = capture.BatchFiled
		case filedCount == 0:
			batch.Status = capture.BatchReady
		default:
			batch.Status = capture.BatchPartiallyFiled
		}
	}
	if status == capture.BatchDiscarded {
		updated = min(processed+f.Between64(volumegen.Minute, volumegen.Day), b.now)
	}
	if (batch.Status == capture.BatchReady || batch.Status == capture.BatchPartiallyFiled) &&
		batch.RetainUntil-b.now <= cvReminderWindow {
		reminded := max(batch.RetainUntil-cvReminderWindow, created)
		batch.RetentionRemindedAt = volumegen.Ptr(min(reminded, b.now))
	}

	return min(updated, b.now)
}

func (b *cvBuilder) item(
	batch *capture.CaptureBatch,
	doc *cvDoc,
	position int,
	processed int64,
	s *cvStack,
) *capture.CaptureItem {
	f := b.g.Fake
	ids := make([]pulid.ID, len(doc.pages))
	for i, page := range doc.pages {
		ids[i] = page.ID
	}
	item := &capture.CaptureItem{
		ID:             volumegen.IDAt("citm_", processed),
		BusinessUnitID: batch.BusinessUnitID,
		OrganizationID: batch.OrganizationID,
		BatchID:        batch.ID,
		Position:       position + 1,
		Status:         capture.ItemProposed,
		PageIDs:        ids,
		DetectedKind:   doc.kind.detected,
		Version:        1,
		CreatedAt:      processed,
	}
	if doc.ship != nil {
		target := capture.Target{
			ResourceType: string(permission.ResourceShipment),
			ResourceID:   volumegen.Ptr(doc.ship.ID),
		}
		if docType, ok := b.docTypes[doc.kind.code]; ok {
			target.DocumentTypeID = volumegen.Ptr(docType)
		}
		source := capture.SuggestionClassifier
		confidence := 0.62 + f.Float()*0.37
		switch {
		case s.request:
			source, confidence = capture.SuggestionRequest, 1
		case batch.Source == capture.SourcePrint:
			source = capture.SuggestionPerson
		}
		item.Suggest(
			target,
			source,
			float64(int(confidence*1000))/1000,
			cvReasons[doc.kind.detected],
		)
	}

	return item
}

func (b *cvBuilder) file(item *capture.CaptureItem, doc *cvDoc, processed int64) {
	f := b.g.Fake
	if doc.ship == nil {
		return
	}
	at := processed + f.Between64(
		volumegen.Minute,
		max(min(2*volumegen.Day, b.now-processed), volumegen.Minute),
	)
	at = min(at, b.now)
	by := b.g.PickUser(b.g.Refs.UserIDs)
	name := fmt.Sprintf("%s_%s.pdf", doc.kind.file, doc.ship.ProNumber)
	size := int64(0)
	for _, page := range doc.pages {
		size += page.ByteSize
	}
	row := b.g.DocumentRow(
		string(permission.ResourceShipment),
		doc.ship.ID,
		name,
		capture.PageContentType,
		document.ProcessingProfileCapture,
		at,
		by,
		size,
	)
	row.DetectedKind = doc.kind.detected
	row.DocumentTypeID = item.SuggestedDocTypeID
	b.batch.docs = append(b.batch.docs, row)
	b.batch.stamp(cvTableDocs, row.ID, at, at)

	item.Status = capture.ItemFiled
	item.FiledType = string(permission.ResourceShipment)
	item.FiledID = volumegen.Ptr(doc.ship.ID)
	item.FiledDocTypeID = item.SuggestedDocTypeID
	item.DocumentID = volumegen.Ptr(row.ID)
	item.FiledByID = volumegen.Ptr(by)
	item.FiledAt = volumegen.Ptr(at)
}

func (b *cvBuilder) request(
	batch *capture.CaptureBatch,
	device *cvDevice,
	ship *volumegen.ShipmentRecord,
	updated int64,
) {
	f := b.g.Fake
	created := batch.CreatedAt - f.Between64(20, 4*volumegen.Minute)
	req := &capturerequest.CaptureRequest{
		ID:             volumegen.IDAt("creq_", created),
		BusinessUnitID: batch.BusinessUnitID,
		OrganizationID: batch.OrganizationID,
		UserID:         device.entity.UserID,
		DeviceID:       device.entity.ID,
		Mode:           capturerequest.RequestModeScan,
		Status:         capturerequest.RequestCompleted,
		TargetType:     string(permission.ResourceShipment),
		TargetID:       ship.ID,
		ProfileID:      batch.ProfileID,
		SourceName:     batch.SourceName,
		BatchID:        volumegen.Ptr(batch.ID),
		ExpiresAt:      created + 15*volumegen.Minute,
		DeliveredAt:    volumegen.Ptr(created + f.Between64(1, 20)),
		Version:        1,
		CreatedAt:      created,
	}
	if docType, ok := b.docTypes[cvPOD.code]; ok {
		req.DocumentTypeID = volumegen.Ptr(docType)
	}
	batch.RequestID = volumegen.Ptr(req.ID)
	switch batch.Status {
	case capture.BatchReceiving, capture.BatchSealed, capture.BatchProcessing:
		req.Status = capturerequest.RequestInProgress
	case capture.BatchFailed:
		req.Status = capturerequest.RequestFailed
		req.FailureCode = capturerequest.FailurePaperJam
		req.FailureMessage = batch.FailureMessage
		req.CompletedAt = batch.ProcessedAt
	default:
		req.CompletedAt = batch.ProcessedAt
	}
	reqUpdated := created
	if req.CompletedAt != nil {
		reqUpdated = *req.CompletedAt
	}
	b.batch.requests = append(b.batch.requests, req)
	b.batch.stamp(cvTableRequests, req.ID, created, min(max(reqUpdated, updated), b.now))
}
