package billingboardreader

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
)

var receiptKind = recordkind.Kind(permission.ResourceBankReceipt)

type BankReceipts struct{ db *postgres.Connection }

type Received struct{ db *postgres.Connection }

func NewBankReceipts(p Params) *BankReceipts { return &BankReceipts{db: p.DB} }

func NewReceived(p Params) *Received { return &Received{db: p.DB} }

func (*BankReceipts) Kind() recordkind.Kind { return receiptKind }
func (*Received) Kind() recordkind.Kind     { return receiptKind }
func (*Received) Measure() string           { return billingboard.MeasureReceived }

type receiptRow struct {
	ID              pulid.ID
	Reference       string
	Status          string
	AmountMinor     int64
	ReceiptDate     int64
	ExceptionReason string
	MatchedAt       *int64
	Customer        string
}

func readReceipts(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
) ([]receiptRow, error) {
	br := bankreceiptdb.BankReceipts
	cp := customerpaymentdb.Payments
	cus := customerdb.Customers

	return keel.Into[receiptRow](ctx, db.Keel(ctx), br.
		LeftJoin(cp, cp.ID.EqCol(br.MatchedCustomerPaymentID)).
		LeftJoin(cus, cus.ID.EqCol(cp.CustomerID)).
		Pick(
			br.ID, text(br.ReferenceNumber), text(br.Status), br.AmountMinor, br.ReceiptDate,
			text(br.ExceptionReason), br.MatchedAt, text(cus.Name),
		).
		Where(br.ID.In(ids...)))
}

func (r *BankReceipts) Summaries(
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
			rows, err := readReceipts(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for idx := range rows {
				out[rows[idx].ID] = receiptSummary(ctx, &rows[idx])
			}

			return out, nil
		},
	)
}

func receiptSummary(ctx context.Context, row *receiptRow) boardreading.Summary {
	label := strings.TrimSpace(row.Reference)
	if label == "" {
		label = i18n.T(ctx, "Receipt with no reference")
	}
	subtitle := ""
	if customer := strings.TrimSpace(row.Customer); customer != "" &&
		bankreceipt.Status(row.Status) == bankreceipt.StatusMatched {
		subtitle = customer
	}
	status, tone := receiptStatus(ctx, row.Status)

	return boardreading.Summary{
		Label:      label,
		Subtitle:   subtitle,
		Status:     status,
		StatusTone: tone,
	}
}

func dayText(ctx context.Context, at int64, loc *time.Location, now int64) string {
	if loc == nil {
		loc = time.UTC
	}

	return i18n.Day(
		i18n.FromContext(ctx),
		time.Unix(at, 0).In(loc),
		time.Unix(now, 0).In(loc),
	)
}

func (r *Received) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := readReceipts(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(rows))
			for idx := range rows {
				out[rows[idx].ID] = receivedReading(ctx, &rows[idx])
			}

			return out, nil
		},
	)
}

func receivedReading(ctx context.Context, row *receiptRow) boardreading.Reading {
	value := boardcatalog.Money(row.AmountMinor, "")
	reading := boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: slot, Value: &value},
		Fact: &boardreading.Words{
			Text: i18n.T(ctx, "Received {0}"),
			At:   row.ReceiptDate,
			Day:  true,
		},
	}
	switch bankreceipt.Status(row.Status) {
	case bankreceipt.StatusException:
		reading.Tone = boardcatalog.ToneWarn
		if reason := strings.TrimSpace(row.ExceptionReason); reason != "" {
			reading.Fact = &boardreading.Words{
				Text:  slot,
				Value: &boardcatalog.Value{Kind: boardcatalog.ValueText, Text: reason},
			}
		}
	case bankreceipt.StatusMatched:
		if row.MatchedAt != nil && *row.MatchedAt > 0 {
			reading.Fact = &boardreading.Words{
				Text: i18n.T(ctx, "Matched {0}"),
				At:   *row.MatchedAt,
				Day:  true,
			}
		}
	case bankreceipt.StatusImported:
	}

	return reading
}
