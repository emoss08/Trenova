package recordvisibilityrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
)

var ErrUnscopedKind = errors.New("record kind has no database read scope")

const visibleIDsTemplate = " WHERE organization_id = $1 AND business_unit_id = $2 AND id = ANY($3::text[])"

var visibleIDsSQL = map[string]string{
	"invoice":  "SELECT id FROM invoices" + visibleIDsTemplate,
	"order":    "SELECT id FROM orders" + visibleIDsTemplate,
	"shipment": "SELECT id FROM shipments" + visibleIDsTemplate,
	"worker":   "SELECT id FROM workers" + visibleIDsTemplate,
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
	_, ok := visibleIDsSQL[kind]
	return ok
}

func (r *repository) VisibleIDs(
	ctx context.Context,
	req *kernelports.ListVisibleRecordsRequest,
) ([]pulid.ID, error) {
	statement, ok := visibleIDsSQL[req.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnscopedKind, req.Kind)
	}
	if len(req.IDs) == 0 {
		return []pulid.ID{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ids := make([]string, len(req.IDs))
		for i, id := range req.IDs {
			ids[i] = id.String()
		}

		visible, err := keel.QueryAll(
			ctx,
			r.db.Keel(ctx),
			pgx.RowTo[pulid.ID],
			statement,
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
			ids,
		)
		if err != nil {
			return nil, fmt.Errorf("check which %s records are visible: %w", req.Kind, err)
		}
		if visible == nil {
			return []pulid.ID{}, nil
		}
		return visible, nil
	})
}
