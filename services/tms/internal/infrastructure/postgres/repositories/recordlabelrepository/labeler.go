// Package recordlabelrepository reads the words records are known by, for a
// preview that names a customer rather than its id. One query per resource,
// inside the tenant, through the caller's transaction when there is one.
package recordlabelrepository

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// MaxIDsPerResource bounds one resource's query: every record the longest
// record subset may offer a person to untick. A preview names a few dozen;
// anything past this is left unlabelled.
const MaxIDsPerResource = services.MaxRecordLabelsPerResource

const labelAlias = " AS label"

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Labeler struct {
	db *postgres.Connection
}

func New(p Params) services.RecordLabeler {
	return &Labeler{db: p.DB}
}

type labelRow struct {
	ID    pulid.ID `bun:"id"`
	Label string   `bun:"label"`
}

// source is one resource's label: which table, how it is scoped to the
// tenant, which column is the id and what expression names a row.
type source struct {
	model func() any
	scope func(*bun.SelectQuery, pagination.TenantInfo) *bun.SelectQuery
	id    buncolgen.Column
	label string
}

//nolint:exhaustive // only the resources whose records a preview names by label are listed
var sources = map[permission.Resource]source{
	permission.ResourceShipment: {
		model: func() any { return (*shipment.Shipment)(nil) },
		scope: buncolgen.ShipmentScopeTenant,
		id:    buncolgen.ShipmentColumns.ID,
		label: buncolgen.ShipmentColumns.ProNumber.Qualified(),
	},
	permission.ResourceCustomer: {
		model: func() any { return (*customer.Customer)(nil) },
		scope: buncolgen.CustomerScopeTenant,
		id:    buncolgen.CustomerColumns.ID,
		label: buncolgen.CustomerColumns.Name.Qualified(),
	},
	permission.ResourceLocation: {
		model: func() any { return (*location.Location)(nil) },
		scope: buncolgen.LocationScopeTenant,
		id:    buncolgen.LocationColumns.ID,
		label: buncolgen.LocationColumns.Name.Qualified(),
	},
	permission.ResourceWorker: {
		model: func() any { return (*worker.Worker)(nil) },
		scope: buncolgen.WorkerScopeTenant,
		id:    buncolgen.WorkerColumns.ID,
		label: buncolgen.Expr(
			"CONCAT_WS(' ', {0}, {1})",
			buncolgen.WorkerColumns.FirstName,
			buncolgen.WorkerColumns.LastName,
		),
	},
	permission.ResourceTractor: {
		model: func() any { return (*tractor.Tractor)(nil) },
		scope: buncolgen.TractorScopeTenant,
		id:    buncolgen.TractorColumns.ID,
		label: buncolgen.TractorColumns.Code.Qualified(),
	},
	permission.ResourceTrailer: {
		model: func() any { return (*trailer.Trailer)(nil) },
		scope: buncolgen.TrailerScopeTenant,
		id:    buncolgen.TrailerColumns.ID,
		label: buncolgen.TrailerColumns.Code.Qualified(),
	},
	permission.ResourceCarrier: {
		model: func() any { return (*carrier.Carrier)(nil) },
		scope: buncolgen.CarrierScopeTenant,
		id:    buncolgen.CarrierColumns.ID,
		label: buncolgen.CarrierColumns.Name.Qualified(),
	},
	permission.ResourceCommodity: {
		model: func() any { return (*commodity.Commodity)(nil) },
		scope: buncolgen.CommodityScopeTenant,
		id:    buncolgen.CommodityColumns.ID,
		label: buncolgen.CommodityColumns.Name.Qualified(),
	},
	permission.ResourceEquipmentType: {
		model: func() any { return (*equipmenttype.EquipmentType)(nil) },
		scope: buncolgen.EquipmentTypeScopeTenant,
		id:    buncolgen.EquipmentTypeColumns.ID,
		label: buncolgen.EquipmentTypeColumns.Code.Qualified(),
	},
	permission.ResourceFleetCode: {
		model: func() any { return (*fleetcode.FleetCode)(nil) },
		scope: buncolgen.FleetCodeScopeTenant,
		id:    buncolgen.FleetCodeColumns.ID,
		label: buncolgen.FleetCodeColumns.Code.Qualified(),
	},
	permission.ResourceAccessorialCharge: {
		model: func() any { return (*accessorialcharge.AccessorialCharge)(nil) },
		scope: buncolgen.AccessorialChargeScopeTenant,
		id:    buncolgen.AccessorialChargeColumns.ID,
		label: buncolgen.AccessorialChargeColumns.Code.Qualified(),
	},
	permission.ResourceInvoice: {
		model: func() any { return (*invoice.Invoice)(nil) },
		scope: buncolgen.InvoiceScopeTenant,
		id:    buncolgen.InvoiceColumns.ID,
		label: buncolgen.InvoiceColumns.Number.Qualified(),
	},
	permission.ResourceDocument: {
		model: func() any { return (*document.Document)(nil) },
		scope: buncolgen.DocumentScopeTenant,
		id:    buncolgen.DocumentColumns.ID,
		label: buncolgen.Expr(
			"COALESCE(NULLIF({0}, ''), {1})",
			buncolgen.DocumentColumns.OriginalName,
			buncolgen.DocumentColumns.FileName,
		),
	},
	permission.ResourceDashboard: {
		model: func() any { return (*report.Dashboard)(nil) },
		scope: buncolgen.DashboardScopeTenant,
		id:    buncolgen.DashboardColumns.ID,
		label: buncolgen.DashboardColumns.Name.Qualified(),
	},
	permission.ResourceReport: {
		model: func() any { return (*report.ReportDefinition)(nil) },
		scope: buncolgen.ReportDefinitionScopeTenant,
		id:    buncolgen.ReportDefinitionColumns.ID,
		label: buncolgen.ReportDefinitionColumns.Name.Qualified(),
	},
	permission.ResourceBillingQueue: {
		model: func() any { return (*billingqueue.BillingQueueItem)(nil) },
		scope: buncolgen.BillingQueueItemScopeTenant,
		id:    buncolgen.BillingQueueItemColumns.ID,
		label: buncolgen.BillingQueueItemColumns.Number.Qualified(),
	},
	permission.ResourceBankReceipt: {
		model: func() any { return (*bankreceipt.BankReceipt)(nil) },
		scope: buncolgen.BankReceiptScopeTenant,
		id:    buncolgen.BankReceiptColumns.ID,
		label: buncolgen.BankReceiptColumns.ReferenceNumber.Qualified(),
	},
	permission.ResourceInsight: {
		model: func() any { return (*insight.Insight)(nil) },
		scope: buncolgen.InsightScopeTenant,
		id:    buncolgen.InsightColumns.ID,
		label: buncolgen.InsightColumns.Headline.Qualified(),
	},
	permission.ResourceInboundMessage: {
		model: func() any { return (*inboundmessage.InboundMessage)(nil) },
		scope: buncolgen.InboundMessageScopeTenant,
		id:    buncolgen.InboundMessageColumns.ID,
		label: buncolgen.InboundMessageColumns.Subject.Qualified(),
	},
	permission.ResourceServiceFailure: {
		model: func() any { return (*servicefailure.ServiceFailure)(nil) },
		scope: buncolgen.ServiceFailureScopeTenant,
		id:    buncolgen.ServiceFailureColumns.ID,
		label: buncolgen.ServiceFailureColumns.Number.Qualified(),
	},
	permission.ResourceHoldReason: {
		model: func() any { return (*holdreason.HoldReason)(nil) },
		scope: buncolgen.HoldReasonScopeTenant,
		id:    buncolgen.HoldReasonColumns.ID,
		label: buncolgen.HoldReasonColumns.Label.Qualified(),
	},
}

func (l *Labeler) Labels(
	ctx context.Context,
	tenant pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (services.RecordLabels, error) {
	labels := make(services.RecordLabels, len(refs))

	resources := make([]permission.Resource, 0, len(refs))
	for resource := range refs {
		resources = append(resources, resource)
	}
	slices.Sort(resources)

	for _, resource := range resources {
		ids := distinctIDs(refs[resource])
		if len(ids) == 0 {
			continue
		}

		var (
			found map[pulid.ID]string
			err   error
		)
		if resource == permission.ResourceShipmentMove {
			found, err = l.moveLabels(ctx, tenant, ids)
		} else if src, ok := sources[resource]; ok {
			found, err = l.read(ctx, tenant, &src, ids)
		} else {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s labels: %w", resource, err)
		}
		if len(found) > 0 {
			labels[resource] = found
		}
	}

	return labels, nil
}

func (l *Labeler) read(
	ctx context.Context,
	tenant pagination.TenantInfo,
	src *source,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	rows := make([]labelRow, 0, len(ids))
	if err := labelQuery(l.db.DBForContext(ctx), tenant, src, ids).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	return collect(rows), nil
}

func labelQuery(
	db bun.IDB,
	tenant pagination.TenantInfo,
	src *source,
	ids []pulid.ID,
) *bun.SelectQuery {
	return db.NewSelect().
		Model(src.model()).
		ColumnExpr(src.id.As("id")).
		ColumnExpr(src.label+labelAlias).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return src.scope(q, tenant).Where(src.id.In(), bun.List(ids))
		})
}

// moveLabels names a move by its shipment and its place in it: "PRO-100
// move 2".
func (l *Labeler) moveLabels(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	moves := make([]*shipment.ShipmentMove, 0, len(ids))
	cols := buncolgen.ShipmentMoveColumns
	if err := l.db.DBForContext(ctx).NewSelect().
		Model(&moves).
		Column(cols.ID.String(), cols.ShipmentID.String(), cols.Sequence.String()).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentMoveScopeTenant(q, tenant).
				Where(cols.ID.In(), bun.List(ids))
		}).
		Scan(ctx); err != nil {
		return nil, err
	}
	if len(moves) == 0 {
		return map[pulid.ID]string{}, nil
	}

	shipmentIDs := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		shipmentIDs = append(shipmentIDs, move.ShipmentID)
	}
	shipmentSource := sources[permission.ResourceShipment]
	pros, err := l.read(ctx, tenant, &shipmentSource, distinctIDs(shipmentIDs))
	if err != nil {
		return nil, err
	}

	labels := make(map[pulid.ID]string, len(moves))
	for _, move := range moves {
		pro := pros[move.ShipmentID]
		if pro == "" {
			continue
		}
		labels[move.ID] = pro + " move " + strconv.FormatInt(move.Sequence+1, 10)
	}

	return labels, nil
}

func collect(rows []labelRow) map[pulid.ID]string {
	labels := make(map[pulid.ID]string, len(rows))
	for _, row := range rows {
		if label := strings.TrimSpace(row.Label); label != "" {
			labels[row.ID] = label
		}
	}

	return labels
}

func distinctIDs(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, min(len(ids), MaxIDsPerResource))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if len(out) == MaxIDsPerResource {
			break
		}
	}

	return out
}
