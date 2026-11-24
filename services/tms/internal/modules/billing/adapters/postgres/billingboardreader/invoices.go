package billingboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const slot = "{0}"

var (
	invoiceKind = recordkind.Kind(permission.ResourceInvoice)
	disputeKind = recordkind.Kind(permission.ResourceInvoiceDispute)
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Invoices struct{ db *postgres.Connection }

type Disputes struct{ db *postgres.Connection }

type AmountDue struct{ db *postgres.Connection }

type Total struct{ db *postgres.Connection }

type Age struct{ db *postgres.Connection }

func NewInvoices(p Params) *Invoices { return &Invoices{db: p.DB} }

func NewDisputes(p Params) *Disputes { return &Disputes{db: p.DB} }

func NewAmountDue(p Params) *AmountDue { return &AmountDue{db: p.DB} }

func NewTotal(p Params) *Total { return &Total{db: p.DB} }

func NewAge(p Params) *Age { return &Age{db: p.DB} }

func (*Invoices) Kind() recordkind.Kind  { return invoiceKind }
func (*Disputes) Kind() recordkind.Kind  { return disputeKind }
func (*AmountDue) Kind() recordkind.Kind { return invoiceKind }
func (*AmountDue) Measure() string       { return billingboard.MeasureAmountDue }
func (*Total) Kind() recordkind.Kind     { return invoiceKind }
func (*Total) Measure() string           { return billingboard.MeasureTotal }
func (*Age) Kind() recordkind.Kind       { return invoiceKind }
func (*Age) Measure() string             { return billingboard.MeasureAge }

type invoiceRow struct {
	ID          pulid.ID
	Label       string
	Subtitle    string
	Status      string
	Currency    string
	TotalMinor  int64
	BalanceDue  int64
	InvoiceDate int64
	DueDate     *int64
}

type summaryRow struct {
	ID       pulid.ID
	Label    string
	Subtitle string
	Status   string
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func readInvoices(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
) ([]invoiceRow, error) {
	t := invoicedb.Invoices

	return keel.Into[invoiceRow](ctx, db.Keel(ctx), t.Pick(
		t.ID, text(t.Number), text(t.BillToName), text(t.Status), text(t.CurrencyCode),
		t.TotalAmountMinor, t.BalanceDueMinor, t.InvoiceDate, t.DueDate,
	).Where(t.ID.In(ids...)))
}

func readEach(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
	each func(row *invoiceRow) (boardreading.Reading, bool),
) (map[pulid.ID]boardreading.Reading, error) {
	if len(ids) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}

	rows, err := readInvoices(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Reading, len(rows))
	for idx := range rows {
		if reading, ok := each(&rows[idx]); ok {
			out[rows[idx].ID] = reading
		}
	}

	return out, nil
}

func (r *Invoices) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Summary{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			rows, err := readInvoices(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for idx := range rows {
				status, tone := invoiceStatus(ctx, rows[idx].Status)
				out[rows[idx].ID] = boardreading.Summary{
					Label:      strings.TrimSpace(rows[idx].Label),
					Subtitle:   strings.TrimSpace(rows[idx].Subtitle),
					Status:     status,
					StatusTone: tone,
				}
			}

			return out, nil
		},
	)
}

func (r *Disputes) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Summary{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			t := invoicedisputedb.InvoiceDisputes
			inv := invoicedb.Invoices
			rows, err := keel.Into[summaryRow](ctx, r.db.Keel(ctx),
				t.Join(inv, inv.ID.EqCol(t.InvoiceID)).Pick(
					t.ID, text(inv.Number), text(t.ReasonCode), text(t.Status),
				).Where(t.ID.In(req.IDs...)))
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for _, row := range rows {
				status, tone := disputeStatus(ctx, row.Status)
				out[row.ID] = boardreading.Summary{
					Label:      i18n.T(ctx, "Dispute on {0}", strings.TrimSpace(row.Label)),
					Subtitle:   strings.TrimSpace(row.Subtitle),
					Status:     status,
					StatusTone: tone,
				}
			}

			return out, nil
		},
	)
}

func (r *AmountDue) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			return readEach(ctx, r.db, req.IDs, func(row *invoiceRow) (boardreading.Reading, bool) {
				value := boardcatalog.Money(row.BalanceDue, row.Currency)
				reading := boardreading.Reading{
					Value:    value,
					Standing: boardreading.Words{Text: i18n.T(ctx, "{0} due"), Value: &value},
				}
				switch {
				case row.BalanceDue <= 0:
					reading.Standing = boardreading.Words{Text: i18n.T(ctx, "Paid")}
					reading.Tone = boardcatalog.ToneOK
				case row.DueDate != nil && *row.DueDate > 0:
					reading.Fact = &boardreading.Words{
						Text: i18n.T(ctx, "Due {0}"),
						At:   *row.DueDate,
						Day:  true,
					}
					if *row.DueDate < req.Now {
						reading.Tone = boardcatalog.ToneWarn
					}
				}

				return reading, true
			})
		},
	)
}

func (r *Total) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			return readEach(ctx, r.db, req.IDs, func(row *invoiceRow) (boardreading.Reading, bool) {
				value := boardcatalog.Money(row.TotalMinor, row.Currency)

				return boardreading.Reading{
					Value:    value,
					Standing: boardreading.Words{Text: slot, Value: &value},
				}, true
			})
		},
	)
}

func (r *Age) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			return readEach(ctx, r.db, req.IDs, func(row *invoiceRow) (boardreading.Reading, bool) {
				if row.InvoiceDate <= 0 {
					return boardreading.Reading{}, false
				}
				age := max(req.Now-row.InvoiceDate, 0)
				value := boardcatalog.Duration(age)

				return boardreading.Reading{
					Value:    value,
					Standing: boardreading.Words{Text: i18n.T(ctx, "{0} old"), Value: &value},
					Fact: &boardreading.Words{
						Text: i18n.T(ctx, "Dated {0}"),
						At:   row.InvoiceDate,
						Day:  true,
					},
					Track: &boardreading.TrackFacts{Behind: age},
				}, true
			})
		},
	)
}
