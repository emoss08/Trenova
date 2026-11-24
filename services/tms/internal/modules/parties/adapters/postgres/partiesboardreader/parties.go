package partiesboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardwording"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Carriers struct{ db *postgres.Connection }

type Customers struct{ db *postgres.Connection }

func NewCarriers(p Params) *Carriers { return &Carriers{db: p.DB} }

func NewCustomers(p Params) *Customers { return &Customers{db: p.DB} }

func (*Carriers) Kind() recordkind.Kind  { return recordkind.Kind(permission.ResourceCarrier) }
func (*Customers) Kind() recordkind.Kind { return recordkind.Kind(permission.ResourceCustomer) }

type summaryRow struct {
	ID       pulid.ID
	Label    string
	Subtitle string
	Status   string
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func summaries(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
	query func(ids []pulid.ID) keel.Projection,
	word func(ctx context.Context, status string) (string, boardcatalog.Tone),
) (map[pulid.ID]boardreading.Summary, error) {
	if len(ids) == 0 {
		return map[pulid.ID]boardreading.Summary{}, nil
	}

	rows, err := keel.Into[summaryRow](ctx, db.Keel(ctx), query(ids))
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Summary, len(rows))
	for _, row := range rows {
		status, tone := word(ctx, row.Status)
		out[row.ID] = boardreading.Summary{
			Label:      strings.TrimSpace(row.Label),
			Subtitle:   strings.TrimSpace(row.Subtitle),
			Status:     status,
			StatusTone: tone,
		}
	}

	return out, nil
}

func (r *Carriers) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			return summaries(ctx, r.db, req.IDs, func(ids []pulid.ID) keel.Projection {
				t := carrierdb.Carriers
				mc := keel.Text("MC ").Cat(t.MCNumber.NullIf(""))
				dot := keel.Text("DOT ").Cat(t.DOTNumber.NullIf(""))

				return t.Pick(t.ID, text(t.Name), text(mc.Coalesce(dot)), text(t.ComplianceStatus)).
					Where(t.ID.In(ids...))
			}, complianceStatus)
		},
	)
}

func (r *Customers) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			return summaries(ctx, r.db, req.IDs, func(ids []pulid.ID) keel.Projection {
				t := customerdb.Customers

				return t.Pick(t.ID, text(t.Name), text(t.Code), text(t.Status)).
					Where(t.ID.In(ids...))
			}, boardwording.ActiveStatus)
		},
	)
}
