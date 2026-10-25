package editenderrepairrepository

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultLimit = 200
	maxLimit     = 1000
	fileHeadSize = 256

	shipmentIDKey    = "externalShipmentId"
	paymentMethodKey = "paymentMethod"

	transferReferencesPath = "{ratingDetail}"
	messageReferencesPath  = "{loadTender,ratingDetail}"
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
	MessageID      pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	TransferID     pulid.ID
	RawX12         string
	FileHead       string
	ShipmentID     string
	PaymentMethod  string
}

func limitOf(limit int) int {
	if limit <= 0 || limit > maxLimit {
		return defaultLimit
	}
	return limit
}

func inboundTenders(orgID pulid.ID) []keel.Pred {
	m := edidb.EDIMessages
	return []keel.Pred{
		m.Direction.Eq(edi.DocumentDirectionInbound),
		m.TransactionSet.Eq(edi.TransactionSet204),
		m.PayloadSnapshot.HasKey("loadTender"),
		keel.When(orgID.IsNotNil(), m.OrganizationID.Eq(orgID)),
	}
}

func ratingDetailText(key string) keel.Expr {
	m := edidb.EDIMessages
	return keel.DynOf(m.PayloadSnapshot.Text("loadTender", "ratingDetail", key)).Coalesce("")
}

func candidatesQuery(
	req *ediports.ListTenderReferenceCandidatesRequest,
) *keel.Query[edi.EDIMessage] {
	m := edidb.EDIMessages
	f := edidb.EDIInboundFiles

	return m.AllTenants().
		Pick(
			m.ID,
			m.OrganizationID,
			m.BusinessUnitID,
			m.TransferID.Coalesce(""),
			m.RawX12,
			keel.Raw("COALESCE(SUBSTRING(? FROM 1 FOR ?), '')", f.RawContent, fileHeadSize),
			ratingDetailText(shipmentIDKey),
			ratingDetailText(paymentMethodKey),
		).
		LeftJoin(
			f,
			f.ID.EqCol(m.InboundFileID),
			f.OrganizationID.EqCol(m.OrganizationID),
			f.BusinessUnitID.EqCol(m.BusinessUnitID),
		).
		Where(inboundTenders(req.OrganizationID)...).
		Where(
			keel.DynFunc("btrim", m.RawX12).NullIf("").IsNotNull(),
			keel.When(req.AfterID.IsNotNil(), m.ID.Gt(req.AfterID)),
		).
		Order(m.ID.Asc()).
		Limit(limitOf(req.Limit))
}

func (r *repository) ListTenderReferenceCandidates(
	ctx context.Context,
	req *ediports.ListTenderReferenceCandidatesRequest,
) ([]*ediports.TenderReferenceCandidate, error) {
	ctx = dbscope.WithSystem(ctx, "re-read inbound EDI tenders across every organization")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediports.TenderReferenceCandidate, error) {
			rows, err := keel.Into[candidateRow](ctx, r.db.Keel(ctx), candidatesQuery(req))
			if err != nil {
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
	ctx = dbscope.WithSystem(ctx, "count inbound EDI tenders whose X12 was purged")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		m := edidb.EDIMessages
		count, err := m.AllTenants().
			Where(inboundTenders(req.OrganizationID)...).
			Where(
				keel.DynFunc("btrim", m.RawX12).NullIf("").IsNull(),
				keel.DynOf(ratingDetailText(shipmentIDKey)).Ne(""),
			).
			Count(ctx, r.db.Keel(ctx))
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
	ctx = dbscope.WithSystem(ctx, "repair the references of a re-read inbound EDI tender")
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*ediports.RepairTenderReferenceResult, error) {
			references, err := referencesJSON(req.ShipmentID, req.PaymentMethod)
			if err != nil {
				return nil, err
			}

			db := r.db.Keel(ctx)
			shipments, err := repairShipments(ctx, db, req)
			if err != nil {
				return nil, err
			}
			transfers, err := repairTransfers(ctx, db, req, references)
			if err != nil {
				return nil, err
			}
			if err = repairMessage(ctx, db, req, references); err != nil {
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

func withReferences(column keel.JSONBCol, path, references string) keel.Assignment {
	return column.ToExpr(keel.Raw(
		"jsonb_set(?, ?::text[], (COALESCE(? #> ?::text[], jsonb_build_object()) - ? - ?) || ?::jsonb)",
		column,
		path,
		column,
		path,
		shipmentIDKey,
		paymentMethodKey,
		references,
	))
}

func transferMatch(req *ediports.RepairTenderReferenceRequest) keel.Pred {
	t := edidb.EDITransfers
	candidate := req.Candidate
	return keel.And(
		keel.Or(
			t.InboundMessageID.Eq(candidate.MessageID),
			keel.When(candidate.TransferID.IsNotNil(), t.ID.Eq(candidate.TransferID)),
		),
		keel.DynOf(t.TenderPayload.Text("ratingDetail", shipmentIDKey)).Coalesce("").Eq(candidate.ShipmentID),
	)
}

func repairShipments(
	ctx context.Context,
	db *keel.DB,
	req *ediports.RepairTenderReferenceRequest,
) (int, error) {
	if req.Candidate.ShipmentID == "" {
		return 0, nil
	}

	sp := shipmentdb.Shipments
	t := edidb.EDITransfers
	targets := t.Pick(t.TargetShipmentID).Where(
		t.TargetShipmentID.IsNotNull(),
		t.TargetOrganizationID.EqCol(sp.OrganizationID),
		t.TargetBusinessUnitID.EqCol(sp.BusinessUnitID),
		transferMatch(req),
	)

	n, err := sp.AllTenants().
		Where(sp.ID.InQuery(targets), sp.ExternalReference.Eq(req.Candidate.ShipmentID)).
		Update(ctx, db, sp.ExternalReference.To(req.ShipmentID))
	if err != nil {
		return 0, fmt.Errorf(
			"repair the external reference of shipments built from tender: %w",
			err,
		)
	}

	return int(n), nil
}

func repairTransfers(
	ctx context.Context,
	db *keel.DB,
	req *ediports.RepairTenderReferenceRequest,
	references string,
) (int, error) {
	t := edidb.EDITransfers
	n, err := t.Where(transferMatch(req)).
		Update(ctx, db, withReferences(t.TenderPayload, transferReferencesPath, references))
	if err != nil {
		return 0, fmt.Errorf("repair the tender references of transfers: %w", err)
	}

	return int(n), nil
}

func repairMessage(
	ctx context.Context,
	db *keel.DB,
	req *ediports.RepairTenderReferenceRequest,
	references string,
) error {
	m := edidb.EDIMessages
	candidate := req.Candidate
	_, err := m.AllTenants().
		Where(
			m.ID.Eq(candidate.MessageID),
			m.OrganizationID.Eq(candidate.OrganizationID),
			m.BusinessUnitID.Eq(candidate.BusinessUnitID),
		).
		Update(ctx, db, withReferences(m.PayloadSnapshot, messageReferencesPath, references))
	if err != nil {
		return fmt.Errorf(
			"repair the tender references of message %s: %w",
			candidate.MessageID,
			err,
		)
	}

	return nil
}
