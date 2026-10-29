package factoringsubmissionservice

import (
	"bytes"
	"context"
	"encoding/csv"
	"html/template"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringaction"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/factoringtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/csvutils"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	pdfContentType       = "application/pdf"
	docResourceShipment  = "shipment"
	docResourceInvoice   = "invoice"
	maxPacketBytes       = 256 << 20
	csvContentType       = "text/csv"
	jpegContentType      = "image/jpeg"
	pngContentType       = "image/png"
	maxPacketSourceBytes = 50 << 20
	maxScheduleFileName  = 120
	scheduleFallbackName = "Factoring-Schedule"
	packetFallbackName   = "Invoice-Packet"
	submissionDocTag     = "factoring"
)

var scheduleCSVHeader = []string{
	"Schedule",
	"Invoice",
	"Invoice date",
	"Due date",
	"Debtor",
	"Debtor code",
	"Debtor address",
	"PRO",
	"BOL",
	"Currency",
	"Amount",
	"Expected advance",
	"Expected fee",
	"Expected reserve",
}

type scheduleSources struct {
	org       *tenant.Organization
	factor    *factoringcompany.FactoringCompany
	agreement *factoring.Agreement
	invoices  map[pulid.ID]*invoice.Invoice
	customers map[pulid.ID]*customer.Customer
	logo      template.URL
}

func (s *SubmissionService) PrepareDocuments(
	ctx context.Context,
	req *factoringsubmissionports.DeliverSubmissionRequest,
) error {
	sub, err := s.repo.GetByID(ctx, factoringsubmissionports.GetSubmissionRequest{
		ID:           req.SubmissionID,
		TenantInfo:   req.TenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return err
	}
	if sub.Status != factoringsubmission.SubmissionStatusSending || !documentsMissing(sub) {
		return nil
	}

	sources, err := s.scheduleSources(ctx, req.TenantInfo, sub)
	if err != nil {
		return err
	}
	update := &factoringsubmissionports.SetSubmissionDocumentsRequest{
		TenantInfo:            req.TenantInfo,
		SubmissionID:          sub.ID,
		SchedulePDFDocumentID: sub.SchedulePDFDocumentID,
		ScheduleCSVDocumentID: sub.ScheduleCSVDocumentID,
		Packets:               make(map[pulid.ID]pulid.ID, len(sub.Items)),
	}

	if sub.SchedulePDFDocumentID.IsNil() {
		rendered, renderErr := s.renderSchedule(ctx, req.TenantInfo, sub, sources)
		if renderErr != nil {
			return renderErr
		}
		stored, storeErr := s.storeFile(ctx, req.TenantInfo, sub, &storedFile{
			name:        scheduleFileName(sub, ".pdf"),
			contentType: pdfContentType,
			content:     rendered.PDF,
			description: "Schedule of accounts " + sub.Number,
		})
		if storeErr != nil {
			return storeErr
		}
		update.SchedulePDFDocumentID = stored
	}
	if sub.ScheduleCSVDocumentID.IsNil() {
		content, csvErr := scheduleCSV(sub, sources)
		if csvErr != nil {
			return csvErr
		}
		stored, storeErr := s.storeFile(ctx, req.TenantInfo, sub, &storedFile{
			name:        scheduleFileName(sub, ".csv"),
			contentType: csvContentType,
			content:     content,
			description: "Schedule of accounts " + sub.Number + " (CSV)",
		})
		if storeErr != nil {
			return storeErr
		}
		update.ScheduleCSVDocumentID = stored
	}
	for _, item := range sub.Items {
		if item == nil || !item.PacketDocumentID.IsNil() {
			continue
		}
		packet, packetErr := s.buildPacket(ctx, req.TenantInfo, item, sources)
		if packetErr != nil {
			return packetErr
		}
		stored, storeErr := s.storeFile(ctx, req.TenantInfo, sub, &storedFile{
			name:        packetFileName(item),
			contentType: pdfContentType,
			content:     packet,
			description: "Invoice " + item.InvoiceNumber + " packet on schedule " + sub.Number,
		})
		if storeErr != nil {
			return storeErr
		}
		update.Packets[item.ID] = stored
	}

	return s.repo.SetDocuments(ctx, update)
}

func documentsMissing(sub *factoringsubmission.Submission) bool {
	if sub.SchedulePDFDocumentID.IsNil() || sub.ScheduleCSVDocumentID.IsNil() {
		return true
	}
	for _, item := range sub.Items {
		if item != nil && item.PacketDocumentID.IsNil() {
			return true
		}
	}
	return false
}

func (s *SubmissionService) scheduleSources(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
) (*scheduleSources, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		return nil, err
	}
	factor, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:           sub.FactoringCompanyID,
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		return nil, err
	}
	agreement, err := s.agreements.GetByID(ctx, factoringports.GetAgreementRequest{
		ID:         sub.AgreementID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	invoices, err := s.loadInvoices(ctx, tenantInfo, sub.InvoiceIDs())
	if err != nil {
		return nil, err
	}
	customers, err := s.loadCustomers(ctx, tenantInfo, sub)
	if err != nil {
		return nil, err
	}
	logo, err := documenttemplateports.ResolveLogoDataURI(ctx, s.inliner, org.LogoURL)
	if err != nil {
		s.l.Warn("could not inline the organization logo on a factoring schedule", zap.Error(err))
		logo = ""
	}
	return &scheduleSources{
		org:       org,
		factor:    factor,
		agreement: agreement,
		invoices:  invoices,
		customers: customers,
		logo:      logo,
	}, nil
}

func (s *SubmissionService) loadCustomers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
) (map[pulid.ID]*customer.Customer, error) {
	ids := make([]pulid.ID, 0, len(sub.Items))
	seen := make(map[pulid.ID]struct{}, len(sub.Items))
	for _, item := range sub.Items {
		if _, dup := seen[item.CustomerID]; dup {
			continue
		}
		seen[item.CustomerID] = struct{}{}
		ids = append(ids, item.CustomerID)
	}
	out := make(map[pulid.ID]*customer.Customer, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: ids,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeState:          true,
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}
	for _, cus := range found {
		if cus != nil {
			out[cus.ID] = cus
		}
	}
	return out, nil
}

func (s *SubmissionService) renderSchedule(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
	sources *scheduleSources,
) (*documenttemplateports.RenderedDocument, error) {
	return s.templates.RenderDocument(ctx, &documenttemplateports.RenderDocumentRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindFactoringSchedulePDF,
		Data:        scheduleContext(sub, sources),
		ReferenceID: sub.ID,
		UserID:      sub.SubmittedByID,
		Title:       strings.TrimSuffix(scheduleFileName(sub, ".pdf"), ".pdf"),
	})
}

func scheduleContext(
	sub *factoringsubmission.Submission,
	sources *scheduleSources,
) *factoringtemplate.FactoringScheduleContext {
	tz := sources.org.Timezone
	format := func(minor int64) string {
		return money.FormatDecimal(sub.CurrencyCode, money.DecimalFromMinor(minor))
	}
	out := &factoringtemplate.FactoringScheduleContext{
		Number:       sub.Number,
		ScheduleDate: formatDate(sub.CreatedAt, tz),
		CurrencyCode: sub.CurrencyCode,
		Organization: templatecatalog.AddressBlock{
			Name:  sources.org.Name,
			Lines: sources.org.MailingAddress().Lines(),
		},
		Factor: templatecatalog.AddressBlock{
			Name:  sources.factor.Name,
			Lines: sources.factor.MailingAddress().Lines(),
		},
		LogoDataURI:     sources.logo,
		AdvanceRate:     basisPointsPercent(sources.agreement.AdvanceRateBP),
		ReserveRate:     basisPointsPercent(sources.agreement.ReserveRateBP),
		Recourse:        recourseLabel(sources.agreement.Recourse),
		Items:           make([]factoringtemplate.FactoringScheduleItem, 0, len(sub.Items)),
		InvoiceCount:    sub.InvoiceCount,
		TotalFace:       format(sub.FaceMinor),
		TotalAdvance:    format(sub.ExpectedAdvanceMinor),
		TotalFee:        format(sub.ExpectedFeeMinor),
		TotalReserve:    format(sub.ExpectedReserveMinor),
		ExpectedPayment: format(sub.ExpectedAdvanceMinor),
	}
	for _, item := range sub.Items {
		row := scheduleRowFor(item, sources)
		out.Items = append(out.Items, factoringtemplate.FactoringScheduleItem{
			InvoiceNumber: item.InvoiceNumber,
			InvoiceDate:   formatDate(row.invoiceDate, tz),
			DueDate:       formatOptionalDate(row.dueDate, tz),
			Debtor:        row.debtor,
			DebtorCode:    row.debtorCode,
			DebtorAddress: row.debtorAddress,
			Reference:     row.reference(),
			Amount:        format(item.FaceMinor),
			Advance:       format(item.ExpectedAdvanceMinor),
			Fee:           format(item.ExpectedFeeMinor),
			Reserve:       format(item.ExpectedReserveMinor),
		})
	}
	return out
}

type scheduleRow struct {
	invoiceDate   int64
	dueDate       *int64
	debtor        string
	debtorCode    string
	debtorAddress string
	pro           string
	bol           string
}

func (r *scheduleRow) reference() string {
	parts := make([]string, 0, 2)
	if r.pro != "" {
		parts = append(parts, "PRO "+r.pro)
	}
	if r.bol != "" {
		parts = append(parts, "BOL "+r.bol)
	}
	return strings.Join(parts, " · ")
}

func scheduleRowFor(
	item *factoringsubmission.SubmissionItem,
	sources *scheduleSources,
) scheduleRow {
	row := scheduleRow{}
	if cus := sources.customers[item.CustomerID]; cus != nil {
		row.debtor = cus.DisplayName()
		row.debtorCode = cus.Code
		row.debtorAddress = strings.Join(cus.MailingAddress().Lines(), ", ")
	}
	inv := sources.invoices[item.InvoiceID]
	if inv == nil {
		return row
	}
	row.invoiceDate = inv.InvoiceDate
	row.dueDate = inv.DueDate
	row.pro = inv.ShipmentProNumber
	row.bol = inv.ShipmentBOL
	if inv.BillToName != "" {
		row.debtor = inv.BillToName
	}
	if inv.BillToCode != "" {
		row.debtorCode = inv.BillToCode
	}
	if address := billToAddress(inv); address != "" {
		row.debtorAddress = address
	}
	return row
}

func billToAddress(inv *invoice.Invoice) string {
	cityLine := strings.TrimSpace(strings.Join(nonEmpty(
		inv.BillToCity,
		strings.TrimSpace(inv.BillToState+" "+inv.BillToPostalCode),
	), ", "))
	return strings.Join(nonEmpty(inv.BillToAddressLine1, inv.BillToAddressLine2, cityLine), ", ")
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func scheduleCSV(sub *factoringsubmission.Submission, sources *scheduleSources) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	if err := writer.Write(scheduleCSVHeader); err != nil {
		return nil, err
	}
	tz := sources.org.Timezone
	plain := func(minor int64) string { return money.DecimalFromMinor(minor).StringFixed(2) }
	for _, item := range sub.Items {
		row := scheduleRowFor(item, sources)
		if err := writer.Write(csvutils.SafeRow([]string{
			sub.Number,
			item.InvoiceNumber,
			formatDate(row.invoiceDate, tz),
			formatOptionalDate(row.dueDate, tz),
			row.debtor,
			row.debtorCode,
			row.debtorAddress,
			row.pro,
			row.bol,
			sub.CurrencyCode,
			plain(item.FaceMinor),
			plain(item.ExpectedAdvanceMinor),
			plain(item.ExpectedFeeMinor),
			plain(item.ExpectedReserveMinor),
		})); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *SubmissionService) buildPacket(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *factoringsubmission.SubmissionItem,
	sources *scheduleSources,
) ([]byte, error) {
	inv, err := s.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         item.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	invoicePDF, err := s.invoicePDF(ctx, tenantInfo, inv)
	if err != nil {
		return nil, err
	}
	pages := []imagingkit.Part{{Source: kernelports.ImagingBytes(pdfContentType, invoicePDF)}}

	supporting, err := s.supportingDocuments(ctx, tenantInfo, inv, sources)
	if err != nil {
		return nil, err
	}
	for _, doc := range supporting {
		page, pageErr := s.documentPart(ctx, tenantInfo, doc)
		if pageErr != nil {
			return nil, errortypes.NewBusinessError(
				"{0} on invoice {1} could not be added to its packet: {2}",
				doc.OriginalName,
				inv.Number,
				pageErr.Error(),
			)
		}
		if page != nil {
			pages = append(pages, *page)
		}
	}
	packet, _, err := imagingkit.Merge(
		ctx,
		s.imaging,
		pages,
		maxPacketBytes,
		kernelports.ImagingBatch,
	)
	return packet, err
}

func (s *SubmissionService) invoicePDF(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inv *invoice.Invoice,
) ([]byte, error) {
	if !inv.PDFDocumentID.IsNil() {
		content, err := s.readDocument(ctx, tenantInfo, inv.PDFDocumentID)
		if err == nil {
			return content, nil
		}
		s.l.Warn("could not read an invoice's stored PDF; rendering it again",
			zap.String("invoiceId", inv.ID.String()), zap.Error(err))
	}
	if s.invoiceService == nil {
		return nil, errortypes.NewBusinessError("Invoice {0} has no PDF yet", inv.Number)
	}
	rendered, err := s.invoiceService.RenderPreview(ctx, &invoiceports.InvoicePreviewRequest{
		InvoiceID:  inv.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	return rendered.Content, nil
}

func (s *SubmissionService) supportingDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inv *invoice.Invoice,
	sources *scheduleSources,
) ([]*document.Document, error) {
	order := packetTypeOrder(sources.customers[inv.CustomerID], sources.agreement)
	if len(order) == 0 {
		return nil, nil
	}
	byType := make(map[pulid.ID][]*document.Document, len(order))
	now := s.now()
	collect := func(resourceType string, ids []pulid.ID) error {
		if len(ids) == 0 {
			return nil
		}
		resourceIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			resourceIDs = append(resourceIDs, id.String())
		}
		docs, err := s.documents.GetByResourceIDs(
			ctx,
			&documentports.GetDocumentsByResourceIDsRequest{
				TenantInfo:   tenantInfo,
				ResourceType: resourceType,
				ResourceIDs:  resourceIDs,
			},
		)
		if err != nil {
			return err
		}
		for _, doc := range docs {
			if doc == nil || doc.DocumentTypeID == nil ||
				doc.StandingAt(now) != document.StandingAccepted {
				continue
			}
			byType[*doc.DocumentTypeID] = append(byType[*doc.DocumentTypeID], doc)
		}
		return nil
	}
	if err := collect(docResourceShipment, inv.LegShipmentIDs()); err != nil {
		return nil, err
	}
	if err := collect(docResourceInvoice, []pulid.ID{inv.ID}); err != nil {
		return nil, err
	}

	out := make([]*document.Document, 0, len(order))
	seen := make(map[pulid.ID]struct{}, len(order))
	for _, typeID := range order {
		for _, doc := range byType[typeID] {
			if _, dup := seen[doc.ID]; dup {
				continue
			}
			seen[doc.ID] = struct{}{}
			out = append(out, doc)
		}
	}
	return out, nil
}

func packetTypeOrder(cus *customer.Customer, agreement *factoring.Agreement) []pulid.ID {
	out := make([]pulid.ID, 0, 8)
	seen := make(map[pulid.ID]struct{}, 8)
	add := func(id pulid.ID) {
		if id.IsNil() {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if cus != nil && cus.BillingProfile != nil {
		for _, dt := range cus.BillingProfile.DocumentTypes {
			if dt != nil {
				add(dt.ID)
			}
		}
	}
	if agreement != nil {
		for _, id := range agreement.DocumentTypeIDs() {
			add(id)
		}
	}
	return out
}

func (s *SubmissionService) documentPart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	doc *document.Document,
) (*imagingkit.Part, error) {
	contentType := strings.ToLower(strings.TrimSpace(doc.FileType))
	switch contentType {
	case pdfContentType, jpegContentType, pngContentType:
	default:
		s.l.Info("left a document out of a factoring packet because it is not a PDF or image",
			zap.String("documentId", doc.ID.String()), zap.String("fileType", doc.FileType))
		return nil, nil
	}
	content, err := s.readDocument(ctx, tenantInfo, doc.ID)
	if err != nil {
		return nil, err
	}
	return &imagingkit.Part{Source: kernelports.ImagingBytes(contentType, content)}, nil
}

func (s *SubmissionService) readDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentID pulid.ID,
) ([]byte, error) {
	content, err := s.invoiceDocuments.GetDownloadContent(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	defer content.Body.Close()
	body, err := io.ReadAll(io.LimitReader(content.Body, maxPacketSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxPacketSourceBytes {
		return nil, errortypes.NewBusinessError("A document is larger than 50 MB")
	}
	return body, nil
}

type storedFile struct {
	name        string
	contentType string
	content     []byte
	description string
}

func (s *SubmissionService) storeFile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
	file *storedFile,
) (pulid.ID, error) {
	stored, err := s.store.Store(ctx, &documentports.StoreDocumentRequest{
		TenantInfo:   tenantInfo,
		Actor:        factoringaction.SystemActor(tenantInfo),
		Content:      file.content,
		FileName:     file.name,
		ContentType:  file.contentType,
		ResourceType: factoringsubmission.SubmissionDocumentResourceType,
		ResourceID:   sub.ID.String(),
		Description:  file.description,
		Tags:         []string{submissionDocTag, sub.Number},
	})
	if err != nil {
		return pulid.Nil, err
	}
	return stored.ID, nil
}

func scheduleFileName(sub *factoringsubmission.Submission, ext string) string {
	return fileutils.SanitizeDisplayFilename(
		sub.Number+ext,
		scheduleFallbackName+ext,
		maxScheduleFileName,
	)
}

func packetFileName(item *factoringsubmission.SubmissionItem) string {
	return fileutils.SanitizeDisplayFilename(
		item.InvoiceNumber+".pdf",
		packetFallbackName+".pdf",
		maxScheduleFileName,
	)
}

func formatDate(ts int64, tz string) string {
	return timeutils.FormatUnixDateIn(ts, tz)
}

func formatOptionalDate(ts *int64, tz string) string {
	if ts == nil {
		return ""
	}
	return timeutils.FormatUnixDateIn(*ts, tz)
}

func basisPointsPercent(bp int) string {
	return floatutils.FormatPercent(float64(bp)/100, 2)
}

func recourseLabel(r factoring.Recourse) string {
	if r == factoring.RecourseNone {
		return "Without recourse"
	}
	return "With recourse"
}
