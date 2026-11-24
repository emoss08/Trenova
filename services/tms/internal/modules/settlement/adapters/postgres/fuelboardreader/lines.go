package fuelboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Lines struct{ db *postgres.Connection }

type NetTax struct{ db *postgres.Connection }

func NewLines(p Params) *Lines { return &Lines{db: p.DB} }

func NewNetTax(p Params) *NetTax { return &NetTax{db: p.DB} }

func (*Lines) Kind() recordkind.Kind  { return recordkind.IFTAReturnLine }
func (*NetTax) Kind() recordkind.Kind { return recordkind.IFTAReturnLine }
func (*NetTax) Measure() string       { return fuelboard.MeasureNetTax }

type lineRow struct {
	ID             pulid.ID
	ReturnID       pulid.ID
	Jurisdiction   string
	FuelType       string
	TotalMiles     decimal.Decimal
	TaxPaidGallons decimal.Decimal
	RateMissing    bool
	LineTotalMinor int64
	Currency       string
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func readLines(ctx context.Context, db *postgres.Connection, ids []pulid.ID) ([]lineRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rl := iftadb.ReturnLines
	ifj := iftadb.Jurisdictions
	ifr := iftadb.Returns

	return keel.Into[lineRow](ctx, db.Keel(ctx), rl.
		Join(ifj, ifj.ID.EqCol(rl.JurisdictionID)).
		Join(ifr, ifr.ID.EqCol(rl.ReturnID)).
		Pick(
			rl.ID, rl.ReturnID, text(ifj.Name), text(rl.FuelType), rl.TotalMiles,
			rl.TaxPaidGallons, rl.RateMissing, rl.LineTotalMinor, text(ifr.CurrencyCode),
		).
		Where(rl.ID.In(ids...)))
}

func mapLines[T any](rows []lineRow, each func(row *lineRow) T) map[pulid.ID]T {
	out := make(map[pulid.ID]T, len(rows))
	for idx := range rows {
		out[rows[idx].ID] = each(&rows[idx])
	}

	return out
}

func (r *Lines) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			rows, err := readLines(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}

			return mapLines(rows, func(row *lineRow) boardreading.Summary {
				return lineSummary(ctx, row)
			}), nil
		},
	)
}

func lineSummary(ctx context.Context, row *lineRow) boardreading.Summary {
	return boardreading.Summary{
		Label: strings.TrimSpace(row.Jurisdiction),
		Subtitle: i18n.T(ctx, "{0}, {1} mi",
			i18n.T(ctx, domaintypes.IFTAFuelType(row.FuelType).Label()),
			i18n.FormatNumber(i18n.FromContext(ctx), row.TotalMiles.InexactFloat64()),
		),
		Opens: &boardcatalog.Ref{Kind: recordkind.IFTAReturn, ID: row.ReturnID},
	}
}

func (r *NetTax) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := readLines(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}

			return mapLines(rows, func(row *lineRow) boardreading.Reading {
				return netTaxReading(ctx, row)
			}), nil
		},
	)
}

func netTaxReading(ctx context.Context, row *lineRow) boardreading.Reading {
	gallons := boardcatalog.Number(row.TaxPaidGallons)
	fact := &boardreading.Words{Text: i18n.T(ctx, "{0} gal bought"), Value: &gallons}
	value := boardcatalog.Money(row.LineTotalMinor, row.Currency)
	if row.RateMissing {
		return boardreading.Reading{
			Value:    value,
			Standing: boardreading.Words{Text: i18n.T(ctx, "Rate missing")},
			Fact:     fact,
			Tone:     boardcatalog.ToneBad,
			Centred:  &boardreading.CentredFacts{Missing: true},
		}
	}
	reading := boardreading.Reading{
		Value:   value,
		Fact:    fact,
		Centred: &boardreading.CentredFacts{Signed: value.Number},
	}
	switch {
	case row.LineTotalMinor > 0:
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} owed"), Value: &value}
	case row.LineTotalMinor < 0:
		credit := boardcatalog.Money(-row.LineTotalMinor, row.Currency)
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} credit"), Value: &credit}
	default:
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "Nothing owed")}
	}

	return reading
}
