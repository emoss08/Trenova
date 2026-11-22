package emailpreview

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/invoicetemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/reporttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templateregistry"
	"github.com/emoss08/trenova/pkg/emaildesign"
)

type Case struct {
	Group string
	Kind  documenttemplate.Kind
	Name  string
	Slug  string
	Data  any
	Brand emaildesign.Brand
}

type variant struct {
	name    string
	fields  map[string]any
	noLogo  bool
	fieldFn func() map[string]any
}

type entry struct {
	group    string
	kind     documenttemplate.Kind
	variants []variant
}

const (
	groupAccount   = "Account and access"
	groupNotices   = "Notifications and reports"
	groupTracking  = "Tracking, tendering and detention"
	groupBilling   = "Billing and factoring"
	groupAgents    = "Agents and Triage"
	groupMail      = "Mail checks"
	defaultVariant = "Default"
	detentionStage = "Kind"
)

func entries() []entry {
	plain := []variant{{name: defaultVariant}}
	return []entry{
		{groupAccount, documenttemplate.KindAccountInviteEmail, []variant{
			{name: defaultVariant},
			{name: "Without a logo", noLogo: true},
			{name: "No username or inviter, expiring in hours", fields: map[string]any{
				"Username": "", "InvitedByName": "", "ExpiresInDays": 0,
			}},
		}},
		{groupAccount, documenttemplate.KindPasswordResetEmail, plain},
		{groupAccount, documenttemplate.KindDriverPortalInvitationEmail, plain},

		{groupNotices, documenttemplate.KindNotificationEmail, []variant{
			{name: defaultVariant},
			{name: "Without a link", fields: map[string]any{"Link": ""}},
		}},
		{groupNotices, documenttemplate.KindNotificationDigestEmail, []variant{
			{name: defaultVariant},
			{name: "A long day", fieldFn: longDigest},
		}},
		{groupNotices, documenttemplate.KindRecordShareEmail, []variant{
			{name: defaultVariant},
			{name: "Without a note", fields: map[string]any{"Note": ""}},
		}},
		{groupNotices, documenttemplate.KindReportDeliveryEmail, []variant{
			{name: defaultVariant},
			{name: "Download link, results cut short", fields: map[string]any{
				"Attached":     false,
				"Truncated":    true,
				"AttachedNote": "The workbook is too large to attach, so it is ready to download instead.",
			}},
			{name: "A long table", fieldFn: longReport},
		}},

		{groupTracking, documenttemplate.KindCustomerTrackingLinkEmail, plain},
		{groupTracking, documenttemplate.KindCustomerStatusEmail, []variant{
			{name: "Picked up", fields: statusFields("IsPickedUp")},
			{name: "Running late", fields: statusFields("IsDelayed")},
			{name: "Delivered", fields: statusFields("IsDelivered")},
			{name: "Proof of delivery", fields: statusFields("IsPodReady")},
		}},
		{groupTracking, documenttemplate.KindCarrierTrackingLinkEmail, plain},
		{groupTracking, documenttemplate.KindTenderOfferEmail, plain},
		{groupTracking, documenttemplate.KindRateConfirmationEmail, []variant{
			{name: "With a sign link"},
			{name: "Without a sign link", fields: map[string]any{
				"SignURL": "", "SignExpiresLabel": "",
			}},
		}},
		{groupTracking, documenttemplate.KindDetentionNoticeEmail, []variant{
			{name: "Warning", fields: map[string]any{detentionStage: "Warning"}},
			{name: "Started", fields: map[string]any{detentionStage: "Started"}},
			{name: "Update", fields: map[string]any{detentionStage: "Update"}},
			{name: "Final"},
		}},

		{groupBilling, documenttemplate.KindInvoiceEmail, []variant{
			{name: defaultVariant},
			{name: "Written message, part 2 of 3", fields: map[string]any{
				"Message": "Hi Morgan,\n\nHere is the invoice for last week's Pharr to Fort Worth load. " +
					"The detention line matches the gate times we sent on Friday.\n\nThanks,\nAvery",
				"PartLabel":     "2 of 3",
				"DocumentLinks": []invoicetemplate.InvoiceEmailLink(nil),
			}},
			{name: "Without shipment detail or a logo", noLogo: true, fields: map[string]any{
				"Charges":              []invoicetemplate.InvoiceEmailCharge(nil),
				"DocumentLinks":        []invoicetemplate.InvoiceEmailLink(nil),
				"ShipmentPickupDate":   "",
				"ShipmentDeliveryDate": "",
				"ShipmentCommodities":  "",
				"ShipmentPieces":       "",
				"ShipmentWeight":       "",
			}},
		}},
		{groupBilling, documenttemplate.KindPaymentReminderEmail, []variant{
			{name: defaultVariant},
			{name: "Final reminder, many invoices", fieldFn: longReminder},
		}},
		{groupBilling, documenttemplate.KindCustomerStatementEmail, plain},
		{groupBilling, documenttemplate.KindFactoringNoaEmail, plain},
		{groupBilling, documenttemplate.KindFactoringScheduleEmail, []variant{
			{name: defaultVariant},
			{name: "Split across messages", fields: map[string]any{"Parts": 3, "Part": 2}},
		}},

		{groupAgents, documenttemplate.KindAgentRequestMissingDocsEmail, plain},
		{groupAgents, documenttemplate.KindAgentStatusRequestEmail, plain},
		{groupAgents, documenttemplate.KindAgentCustomerUpdateEmail, plain},
		{groupAgents, documenttemplate.KindAgentInboundReplyEmail, plain},
		{groupAgents, documenttemplate.KindAgentProposalReminderEmail, plain},
		{groupAgents, documenttemplate.KindTriageReplyEmail, plain},

		{groupMail, documenttemplate.KindMailboxTestEmail, plain},
		{groupMail, documenttemplate.KindEmailProfileTestEmail, plain},
	}
}

func Catalog() ([]*Case, error) {
	registry := templateregistry.Default()
	brand := SampleBrand()
	bare := emaildesign.Brand{Name: brand.Name}

	out := make([]*Case, 0, 64)
	for _, e := range entries() {
		sample, err := registry.SampleContext(e.kind)
		if err != nil {
			return nil, err
		}
		for i, v := range e.variants {
			fields := v.fields
			if v.fieldFn != nil {
				fields = v.fieldFn()
			}
			data, setErr := with(sample, fields)
			if setErr != nil {
				return nil, fmt.Errorf("%s %q: %w", e.kind, v.name, setErr)
			}
			c := &Case{
				Group: e.group,
				Kind:  e.kind,
				Name:  v.name,
				Slug:  string(e.kind) + "-" + strconv.Itoa(i+1),
				Data:  data,
				Brand: brand,
			}
			if v.noLogo {
				c.Brand = bare
			}
			out = append(out, c)
		}
	}
	return out, nil
}

func with(sample any, fields map[string]any) (any, error) {
	if len(fields) == 0 {
		return sample, nil
	}
	value := reflect.New(reflect.TypeOf(sample)).Elem()
	value.Set(reflect.ValueOf(sample))
	for name, raw := range fields {
		field := value.FieldByName(name)
		if !field.IsValid() || !field.CanSet() {
			return nil, fmt.Errorf("the sample has no field %s", name)
		}
		next := reflect.ValueOf(raw)
		if !next.Type().ConvertibleTo(field.Type()) {
			return nil, fmt.Errorf("field %s cannot hold a %s", name, next.Type())
		}
		field.Set(next.Convert(field.Type()))
	}
	return value.Interface(), nil
}

func statusFields(set string) map[string]any {
	fields := map[string]any{
		"IsPickedUp":  false,
		"IsDelayed":   false,
		"IsDelivered": false,
		"IsPodReady":  false,
	}
	fields[set] = true
	return fields
}

var sampleCustomers = []string{
	"Halstead Grocery Group", "Rio Grande Produce", "Cordell Manufacturing", "Bramble Home Goods",
	"Kestrel Paper Mills", "Ostrander Beverage", "Pinecrest Building Supply", "Marlow Auto Parts",
	"Tidewater Foods", "Granite Peak Outfitters", "Sable Creek Dairy", "Hollis Glassworks",
}

func longDigest() map[string]any {
	items := make([]notificationtemplate.NotificationDigestItem, 0, 12)
	for i := range 12 {
		items = append(items, notificationtemplate.NotificationDigestItem{
			Title:   "Shipment TRV-8841" + strconv.Itoa(20+i) + " was delivered",
			Message: sampleCustomers[i] + " signed for it at the dock.",
			Link:    "https://app.example.com/shipments/management?item=shp_" + strconv.Itoa(i),
		})
	}
	return map[string]any{"Items": items, "ItemCount": len(items)}
}

func longReport() map[string]any {
	rows := make([]reporttemplate.ReportRow, 0, 24)
	for i := range 24 {
		rows = append(rows, reporttemplate.ReportRow{Cells: []reporttemplate.ReportCell{
			{
				Text: sampleCustomers[i%len(sampleCustomers)] + " " + strconv.Itoa(
					i/len(sampleCustomers)+1,
				),
			},
			{Text: strconv.Itoa(40 - i), Numeric: true},
			{Text: "$" + strconv.Itoa(98-i*3) + ",441.20", Numeric: true},
		}})
	}
	return map[string]any{
		"RowCount": int64(len(rows)),
		"Digest": reporttemplate.ReportDigest{
			Headers: []reporttemplate.ReportColumn{
				{Label: "Customer"},
				{Label: "Shipments", Numeric: true},
				{Label: "Revenue", Numeric: true},
			},
			Rows: rows,
			Totals: []reporttemplate.ReportCell{
				{Text: templatecatalog.SampleTotalsLabel},
				{Text: "684", Numeric: true},
				{Text: "$1,522,184.80", Numeric: true},
			},
		},
	}
}

func longReminder() map[string]any {
	invoices := make([]invoicetemplate.PaymentReminderInvoice, 0, 18)
	for i := range 18 {
		invoices = append(invoices, invoicetemplate.PaymentReminderInvoice{
			InvoiceNumber: "INV-2026-" + strconv.Itoa(1000+i),
			InvoiceDate:   "2026-06-" + strconv.Itoa(10+i),
			DueDate:       "2026-07-" + strconv.Itoa(10+i),
			DaysPastDue:   38 - i,
			OpenAmount:    "USD " + strconv.Itoa(1200+i*37) + ".50",
		})
	}
	return map[string]any{
		"IsFinalReminder": true,
		"Invoices":        invoices,
		"TotalOverdue":    "USD 27,951.00",
	}
}
