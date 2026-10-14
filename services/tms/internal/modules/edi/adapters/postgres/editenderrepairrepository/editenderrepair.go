package editenderrepairrepository

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/edicol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultLimit = 200
	maxLimit     = 1000
	fileHeadSize = 256

	shipmentIDKey    = "externalShipmentId"
	paymentMethodKey = "paymentMethod"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.TenderReferenceRepairRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.edi-tender-repair-repository")}
}

type candidateRow struct {
	MessageID      pulid.ID `bun:"message_id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
	TransferID     pulid.ID `bun:"transfer_id"`
	RawX12         string   `bun:"raw_x12"`
	FileHead       string   `bun:"file_head"`
	ShipmentID     string   `bun:"shipment_id"`
	PaymentMethod  string   `bun:"payment_method"`
}

func limitOf(limit int) int {
	if limit <= 0 || limit > maxLimit {
		return defaultLimit
	}
	return limit
}

func inboundTenders(q *bun.SelectQuery, orgID pulid.ID) *bun.SelectQuery {
	cols := edicol.EDIMessageColumns
	q = q.
		Where(cols.Direction.Eq(), edi.DocumentDirectionInbound).
		Where(cols.TransactionSet.Eq(), edi.TransactionSet204).
		Where(cols.PayloadSnapshot.Expr("{} -> 'loadTender' IS NOT NULL"))
	if orgID.IsNotNil() {
		q = q.Where(cols.OrganizationID.Eq(), orgID)
	}
	return q
}

func ratingDetailText(key string) string {
	return edicol.EDIMessageColumns.PayloadSnapshot.Expr(
		"COALESCE({} -> 'loadTender' -> 'ratingDetail' ->> '" + key + "', '')",
	)
}

func (r *repository) ListTenderReferenceCandidates(
	ctx context.Context,
	req *ediports.ListTenderReferenceCandidatesRequest,
) ([]*ediports.TenderReferenceCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediports.TenderReferenceCandidate, error) {
			cols := edicol.EDIMessageColumns
			fileCols := edicol.EDIInboundFileColumns
			limit := limitOf(req.Limit)

			rows := make([]candidateRow, 0, limit)
			q := r.db.DBForContext(ctx).
				NewSelect().
				Model((*edi.EDIMessage)(nil)).
				ColumnExpr(cols.ID.As("message_id")).
				ColumnExpr(cols.OrganizationID.As("organization_id")).
				ColumnExpr(cols.BusinessUnitID.As("business_unit_id")).
				ColumnExpr(cols.TransferID.As("transfer_id")).
				ColumnExpr(cols.RawX12.As("raw_x12")).
				ColumnExpr(
					fmt.Sprintf(
						"COALESCE(SUBSTRING(%s FROM 1 FOR %d), '') AS file_head",
						fileCols.RawContent.Qualified(),
						fileHeadSize,
					),
				).
				ColumnExpr(ratingDetailText(shipmentIDKey) + " AS shipment_id").
				ColumnExpr(ratingDetailText(paymentMethodKey) + " AS payment_method").
				Join(
					"LEFT JOIN edi_inbound_files AS eif ON " +
						fileCols.ID.EqColumn(cols.InboundFileID) +
						" AND " + fileCols.OrganizationID.EqColumn(cols.OrganizationID) +
						" AND " + fileCols.BusinessUnitID.EqColumn(cols.BusinessUnitID),
				).
				Where(cols.RawX12.Expr("NULLIF(BTRIM({}), '') IS NOT NULL")).
				Order(cols.ID.OrderAsc()).
				Limit(int64(limit))
			q = inboundTenders(q, req.OrganizationID)
			if req.AfterID.IsNotNil() {
				q = q.Where(cols.ID.Gt(), req.AfterID)
			}
			if err := q.Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("list inbound tenders to re-read: %w", err)
			}

			candidates := make([]*ediports.TenderReferenceCandidate, 0, len(rows))
			for i := range rows {
				row := &rows[i]
				candidates = append(candidates, &ediports.TenderReferenceCandidate{
					MessageID:      row.MessageID,
					OrganizationID: row.OrganizationID,
					BusinessUnitID: row.BusinessUnitID,
					TransferID:     row.TransferID,
					RawX12:         row.RawX12,
					FileHead:       row.FileHead,
					ShipmentID:     row.ShipmentID,
					PaymentMethod:  row.PaymentMethod,
				})
			}
			return candidates, nil
		},
	)
}

func (r *repository) CountUnreadableTenders(
	ctx context.Context,
	req *ediports.CountUnreadableTendersRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := edicol.EDIMessageColumns
		q := r.db.DBForContext(ctx).
			NewSelect().
			Model((*edi.EDIMessage)(nil)).
			Where(cols.RawX12.Expr("NULLIF(BTRIM({}), '') IS NULL")).
			Where(ratingDetailText(shipmentIDKey) + " <> ''")
		count, err := inboundTenders(q, req.OrganizationID).Count(ctx)
		if err != nil {
			return 0, fmt.Errorf("count inbound tenders whose X12 is gone: %w", err)
		}
		return int(count), nil
	})
}

func (r *repository) RepairTenderReference(
	ctx context.Context,
	req *ediports.RepairTenderReferenceRequest,
) (*ediports.RepairTenderReferenceResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*ediports.RepairTenderReferenceResult, error) {
			references, err := referencesJSON(req.ShipmentID, req.PaymentMethod)
			if err != nil {
				return nil, err
			}
			dba := r.db.DBForContext(ctx)
			now := timeutils.NowUnix()

			shipments, err := r.repairShipments(ctx, dba, req, now)
			if err != nil {
				return nil, err
			}
			transfers, err := r.repairTransfers(ctx, dba, req, references, now)
			if err != nil {
				return nil, err
			}
			if err = r.repairMessage(ctx, dba, req, references, now); err != nil {
				return nil, err
			}
			return &ediports.RepairTenderReferenceResult{
				Transfers: transfers,
				Shipments: shipments,
			}, nil
		},
	)
}

func referencesJSON(shipmentID, paymentMethod string) (string, error) {
	references := make(map[string]string, 2)
	if shipmentID != "" {
		references[shipmentIDKey] = shipmentID
	}
	if paymentMethod != "" {
		references[paymentMethodKey] = paymentMethod
	}
	encoded, err := sonic.MarshalString(references)
	if err != nil {
		return "", fmt.Errorf("encode the re-read tender references: %w", err)
	}
	return encoded, nil
}

func withReferences(column *buncolgen.Column, path string) string {
	return column.SetExpr(
		"jsonb_set({}, '" + path + "', (COALESCE({} #> '" + path + "', jsonb_build_object()) - '" +
			shipmentIDKey + "' - '" + paymentMethodKey + "') || ?::jsonb)",
	)
}

func transferMatch(req *ediports.RepairTenderReferenceRequest) (fragment string, args []any) {
	cols := edicol.EDITransferColumns
	candidate := req.Candidate
	fragment = "(" + cols.InboundMessageID.Qualified() + " = ?"
	args = []any{candidate.MessageID}
	if candidate.TransferID.IsNotNil() {
		fragment += " OR " + cols.ID.Qualified() + " = ?"
		args = append(args, candidate.TransferID)
	}
	fragment += ") AND " +
		cols.TenderPayload.Expr("COALESCE({} -> 'ratingDetail' ->> '"+shipmentIDKey+"', '')") +
		" = ?"
	return fragment, append(args, candidate.ShipmentID)
}

func (r *repository) repairShipments(
	ctx context.Context,
	dba bun.IDB,
	req *ediports.RepairTenderReferenceRequest,
	now int64,
) (int, error) {
	if req.Candidate.ShipmentID == "" {
		return 0, nil
	}
	cols := shipmentcol.ShipmentColumns
	transferCols := edicol.EDITransferColumns

	match, args := transferMatch(req)
	targets := dba.NewSelect().
		Model((*edi.EDITransfer)(nil)).
		Column(transferCols.TargetShipmentID.Bare()).
		Where(transferCols.TargetShipmentID.IsNotNull()).
		Where(transferCols.TargetOrganizationID.EqColumn(cols.OrganizationID)).
		Where(transferCols.TargetBusinessUnitID.EqColumn(cols.BusinessUnitID)).
		Where(match, args...)

	result, err := dba.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set(cols.ExternalReference.Set(), nullIfEmpty(req.ShipmentID)).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), now).
		Where(cols.ID.In(), targets).
		Where(cols.ExternalReference.Eq(), req.Candidate.ShipmentID).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"repair the external reference of shipments built from tender: %w",
			err,
		)
	}
	return affected(result)
}

func (r *repository) repairTransfers(
	ctx context.Context,
	dba bun.IDB,
	req *ediports.RepairTenderReferenceRequest,
	references string,
	now int64,
) (int, error) {
	cols := edicol.EDITransferColumns
	match, args := transferMatch(req)
	result, err := dba.NewUpdate().
		Model((*edi.EDITransfer)(nil)).
		Set(withReferences(&cols.TenderPayload, "{ratingDetail}"), references).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), now).
		Where(match, args...).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("repair the tender references of transfers: %w", err)
	}
	return affected(result)
}

func (r *repository) repairMessage(
	ctx context.Context,
	dba bun.IDB,
	req *ediports.RepairTenderReferenceRequest,
	references string,
	now int64,
) error {
	cols := edicol.EDIMessageColumns
	candidate := req.Candidate
	_, err := dba.NewUpdate().
		Model((*edi.EDIMessage)(nil)).
		Set(withReferences(&cols.PayloadSnapshot, "{loadTender,ratingDetail}"), references).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), now).
		Where(cols.ID.Eq(), candidate.MessageID).
		Where(cols.OrganizationID.Eq(), candidate.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), candidate.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"repair the tender references of message %s: %w",
			candidate.MessageID,
			err,
		)
	}
	return nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func affected(result interface{ RowsAffected() (int64, error) }) (int, error) {
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count repaired rows: %w", err)
	}
	return int(rows), nil
}
