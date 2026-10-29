package querytooltest

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func DraftEditOf(t *testing.T, result any) pagedraft.Edit {
	t.Helper()

	edited, ok := result.(pagedraft.EditResult)
	require.True(t, ok, "a draft tool answers with a draft edit")
	assert.NotEmpty(t, edited.Note)

	return edited.Draft
}

func MarginsReader() *FakePermissions {
	return &FakePermissions{Readable: map[string]*permissionports.ResourcePermissionDetail{
		permission.ResourceProtectedMargins.String(): {
			Resource:       permission.ResourceProtectedMargins.String(),
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityConfidential,
		},
	}}
}

type FakeInvoiceRepo struct {
	invoiceports.InvoiceRepository

	Items    []*invoice.Invoice
	Captured *invoiceports.ListInvoicesRequest
}

func (f *FakeInvoiceRepo) List(
	_ context.Context,
	req *invoiceports.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	f.Captured = req

	return &pagination.ListResult[*invoice.Invoice]{
		Items:      f.Items,
		Total:      len(f.Items),
		TotalExact: true,
	}, nil
}

func (f *FakeInvoiceRepo) GetByID(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	for _, item := range f.Items {
		if item.ID == req.ID {
			return item, nil
		}
	}

	return nil, assert.AnError
}

func SensitiveInvoice() *invoice.Invoice {
	return &invoice.Invoice{
		ID:                     pulid.MustNew("inv_"),
		CustomerID:             pulid.MustNew("cus_"),
		Number:                 "INV-1042",
		Status:                 invoice.StatusPosted,
		SettlementStatus:       invoice.SettlementStatusUnpaid,
		BillToName:             "Acme",
		CurrencyCode:           "USD",
		TotalAmount:            decimal.RequireFromString("1250.00"),
		TotalAmountMinor:       125000,
		SubtotalAmount:         decimal.RequireFromString("1200.00"),
		OtherAmount:            decimal.RequireFromString("50.00"),
		Memo:                   "Discount agreed with the buyer on the phone",
		RemittanceInstructions: "Wire to account 12345",
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Description: "Linehaul",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.RequireFromString("1200.00"),
			Amount:      decimal.RequireFromString("1200.00"),
		}},
	}
}

type FakeBoard struct {
	Board *dispatchconsoleservice.Board
	Req   *dispatchconsoleservice.GetBoardRequest
}

func (f *FakeBoard) GetBoard(
	_ context.Context,
	req *dispatchconsoleservice.GetBoardRequest,
) (*dispatchconsoleservice.Board, error) {
	f.Req = req

	return f.Board, nil
}

type FakeWeather struct {
	Features []*weatheralertports.WeatherAlertFeature
}

func (f *FakeWeather) GetActiveAlerts(
	_ context.Context,
	_ pagination.TenantInfo,
) (*weatheralertports.WeatherAlertFeatureCollection, error) {
	return &weatheralertports.WeatherAlertFeatureCollection{Features: f.Features}, nil
}

func Readable(
	resources ...permission.Resource,
) map[string]*permissionports.ResourcePermissionDetail {
	out := make(map[string]*permissionports.ResourcePermissionDetail, len(resources))
	for _, resource := range resources {
		out[resource.String()] = &permissionports.ResourcePermissionDetail{
			Resource:   resource.String(),
			Operations: []permission.Operation{permission.OpRead},
		}
	}

	return out
}

type FakeReporting struct {
	Entries     []*canned.Entry
	Definitions []*report.ReportDefinition
	Run         *report.ReportRun
	runErr      error
	// settled is what GetRun answers after RunReport, when a test wants the
	// run to have moved on; nil answers run again.
	Settled       *report.ReportRun
	Polls         int
	PreviewResult *reporting.PreviewResult
	PreviewErr    error

	// runs are the runs GetRun finds by id, and rows maps a run id to the
	// stored rows a comparison reads back. A run absent from rows is a run
	// whose rows were never stored, which is a state the tool has to report
	// rather than crash on.
	Runs    []*report.ReportRun
	Rows    map[pulid.ID]*export.Envelope
	RowsErr map[pulid.ID]error

	Submitted *reporting.RunReportRequest
	Fetched   *reporting.GetRunRequest
	Listed    *reporting.ListDefinitionsRequest
	Previewed *reporting.PreviewRequest

	Kinds *reportaccess.Resolver
}

func (f *FakeReporting) Catalog(
	ctx context.Context,
	req *reporting.Request,
) (*reporting.Catalog, error) {
	return reporting.CatalogFor(ctx, f.Kinds, req.TenantInfo)
}

func (f *FakeReporting) ListCanned() []*canned.Entry { return f.Entries }

func (f *FakeReporting) GetCanned(key string) (*canned.Entry, error) {
	for _, entry := range f.Entries {
		if entry.Key == key {
			return entry, nil
		}
	}

	return nil, errortypes.NewNotFoundError("no such report")
}

func (f *FakeReporting) RunReport(
	_ context.Context,
	req *reporting.RunReportRequest,
) (*report.ReportRun, error) {
	f.Submitted = req
	if f.runErr != nil {
		return nil, f.runErr
	}

	return f.Run, nil
}

func (f *FakeReporting) GetRun(
	_ context.Context,
	req *reporting.GetRunRequest,
) (*report.ReportRun, error) {
	f.Fetched = req
	f.Polls++

	// A test that seeded several runs is asking about one of them by id; the
	// single-run tests never seed runs and keep the old behaviour.
	for _, run := range f.Runs {
		if run.ID == req.RunID {
			return run, nil
		}
	}

	if f.Settled != nil {
		return f.Settled, nil
	}

	return f.Run, nil
}

func (f *FakeReporting) ReadRunRows(
	_ context.Context,
	req *reporting.GetRunRequest,
) (*export.Envelope, error) {
	if err, failed := f.RowsErr[req.RunID]; failed {
		return nil, err
	}
	rows, stored := f.Rows[req.RunID]
	if !stored {
		return nil, errortypes.NewBusinessError(
			"This report run was generated before its rows were stored, " +
				"so it cannot be compared",
		)
	}

	return rows, nil
}

func (f *FakeReporting) GetDefinition(
	_ context.Context,
	req *reporting.GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	for _, definition := range f.Definitions {
		if definition.ID != req.DefinitionID {
			continue
		}
		if definition.Visibility == report.VisibilityShared ||
			definition.OwnerID == req.TenantInfo.UserID {
			return definition, nil
		}
	}

	return nil, errortypes.NewNotFoundError("ReportDefinition not found")
}

func (f *FakeReporting) ListDefinitions(
	_ context.Context,
	req *reporting.ListDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	f.Listed = req

	visible := make([]*report.ReportDefinition, 0, len(f.Definitions))
	for _, definition := range f.Definitions {
		if definition.Visibility == report.VisibilityShared ||
			definition.OwnerID == req.TenantInfo.UserID {
			visible = append(visible, definition)
		}
	}

	return visible, nil
}

func (f *FakeReporting) Preview(
	_ context.Context,
	req *reporting.PreviewRequest,
) (*reporting.PreviewResult, error) {
	f.Previewed = req
	if f.PreviewErr != nil {
		return nil, f.PreviewErr
	}
	if f.PreviewResult == nil {
		return &reporting.PreviewResult{}, nil
	}

	return f.PreviewResult, nil
}

func TestEnvelope(
	t *testing.T,
	title string,
	columns []table.Column,
	rows ...[]any,
) *export.Envelope {
	t.Helper()
	builder := table.NewBuilder(columns)
	for _, row := range rows {
		require.NoError(t, builder.Append(row...))
	}
	return &export.Envelope{
		Meta:     export.Meta{Title: title, GeneratedAt: time.Unix(1784131200, 0)},
		Location: time.UTC,
		Table:    builder.Table(),
		RowCount: int64(len(rows)),
	}
}

func TestColumn(id, label string, kind value.Type) table.Column {
	return table.Column{
		ID:      id,
		Label:   label,
		Type:    kind,
		Display: format.Resolve(kind, format.Hint{}, nil),
	}
}

type FakeRetrievalSearcher struct {
	Documents *retrievalports.DocumentSearchResult
	Messages  *retrievalports.InboundMessageSearchResult
	Requests  []retrievalports.RetrievalSearchRequest
}

func (f *FakeRetrievalSearcher) SearchDocuments(
	_ context.Context,
	req *retrievalports.RetrievalSearchRequest,
) (*retrievalports.DocumentSearchResult, error) {
	f.Requests = append(f.Requests, *req)

	return f.Documents, nil
}

func (f *FakeRetrievalSearcher) SearchInboundMessages(
	_ context.Context,
	req *retrievalports.RetrievalSearchRequest,
) (*retrievalports.InboundMessageSearchResult, error) {
	f.Requests = append(f.Requests, *req)

	return f.Messages, nil
}

// FakeWorkerRepo records the request a tool built and returns what it is told.
// It embeds the interface so the methods no tool calls stay unimplemented and
// a future addition to WorkerRepository does not break this file.
type FakeWorkerRepo struct {
	workerports.WorkerRepository

	Captured *workerports.ListWorkersRequest
	Items    []*worker.Worker
	err      error
}

func (f *FakeWorkerRepo) List(
	_ context.Context,
	req *workerports.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	f.Captured = req
	if f.err != nil {
		return nil, f.err
	}

	return &pagination.CursorListResult[*worker.Worker]{Items: f.Items}, nil
}

func (f *FakeWorkerRepo) GetByID(
	_ context.Context,
	_ workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.Items[0], nil
}
