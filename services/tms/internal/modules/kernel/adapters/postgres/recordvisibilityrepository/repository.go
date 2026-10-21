package recordvisibilityrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/buncolgen/ordercol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

var ErrUnscopedKind = errors.New("record kind has no database read scope")

type scopedTable struct {
	table          buncolgen.TableInfo
	id             buncolgen.Column
	organizationID buncolgen.Column
	businessUnitID buncolgen.Column
}

var scopedTables = map[string]scopedTable{
	"invoice": {
		table:          invoicecol.InvoiceTable,
		id:             invoicecol.InvoiceColumns.ID,
		organizationID: invoicecol.InvoiceColumns.OrganizationID,
		businessUnitID: invoicecol.InvoiceColumns.BusinessUnitID,
	},
	"order": {
		table:          ordercol.OrderTable,
		id:             ordercol.OrderColumns.ID,
		organizationID: ordercol.OrderColumns.OrganizationID,
		businessUnitID: ordercol.OrderColumns.BusinessUnitID,
	},
	"shipment": {
		table:          shipmentcol.ShipmentTable,
		id:             shipmentcol.ShipmentColumns.ID,
		organizationID: shipmentcol.ShipmentColumns.OrganizationID,
		businessUnitID: shipmentcol.ShipmentColumns.BusinessUnitID,
	},
	"worker": {
		table:          workercol.WorkerTable,
		id:             workercol.WorkerColumns.ID,
		organizationID: workercol.WorkerColumns.OrganizationID,
		businessUnitID: workercol.WorkerColumns.BusinessUnitID,
	},
}

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) kernelports.RecordVisibility {
	return &repository{db: p.DB}
}

func (r *repository) Scoped(kind string) bool {
	_, ok := scopedTables[kind]
	return ok
}

func (r *repository) VisibleIDs(
	ctx context.Context,
	req *kernelports.ListVisibleRecordsRequest,
) ([]pulid.ID, error) {
	scoped, ok := scopedTables[req.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnscopedKind, req.Kind)
	}
	if len(req.IDs) == 0 {
		return []pulid.ID{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		visible := make([]pulid.ID, 0, len(req.IDs))
		if err := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("? AS ?", bun.Ident(scoped.table.Name), bun.Ident(scoped.table.Alias)).
			Column(scoped.id.Qualified()).
			Where(scoped.organizationID.Eq(), req.TenantInfo.OrgID).
			Where(scoped.businessUnitID.Eq(), req.TenantInfo.BuID).
			Where(scoped.id.In(), bun.List(req.IDs)).
			Scan(ctx, &visible); err != nil {
			return nil, fmt.Errorf("check which %s records are visible: %w", req.Kind, err)
		}
		return visible, nil
	})
}
