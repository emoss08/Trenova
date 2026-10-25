package invoicerepository

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var invoiceErrors = dberror.Mapping{
	Entity: "Invoice",
	Unique: []dberror.Constraint{
		{
			Name:    "uk_invoices_number",
			Field:   "number",
			Message: "Invoice with this number already exists in your organization",
		},
	},
}

var invoiceLineErrors = dberror.Mapping{
	Entity: "Invoice line",
	Unique: []dberror.Constraint{
		{
			Name:    "uk_invoice_lines_number",
			Field:   "lines",
			Message: "Invoice line numbers must be unique within the invoice",
		},
	},
}

var attachmentErrors = dberror.Mapping{
	Entity: "Invoice attachment",
	Unique: []dberror.Constraint{
		{
			Name:    "uk_invoice_attachments_document",
			Field:   "documentIds",
			Message: "A document can be attached to the invoice only once",
		},
	},
}

var emailAttemptErrors = dberror.Mapping{Entity: "Invoice email attempt"}

var shareTokenErrors = dberror.Mapping{
	Entity: "Document share token",
	Unique: []dberror.Constraint{
		{
			Name:    "uk_invoice_document_share_tokens_hash",
			Field:   "tokenHash",
			Message: "Document share token already exists",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) invoiceports.InvoiceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *invoiceports.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*invoice.Invoice], error) {
			inv := invoicedb.Invoices
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				inv.Query(),
				&listquery.Request[invoice.Invoice]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[invoice.Invoice]) *keel.Query[invoice.Invoice] {
						return q.Load(inv.Customer(), inv.Shipment())
					},
				},
			)
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *invoiceports.ListInvoiceConnectionRequest,
) (*pagination.CursorListResult[*invoice.Invoice], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*invoice.Invoice], error) {
			inv := invoicedb.Invoices
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				inv.Query(),
				&listquery.Request[invoice.Invoice]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[invoice.Invoice]) *keel.Query[invoice.Invoice] {
						return q.Load(inv.Customer())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list invoices", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func invoiceDetail() *keel.Query[invoice.Invoice] {
	inv := invoicedb.Invoices
	lines := inv.Lines()

	return inv.Load(
		inv.Customer(),
		inv.ShipperCustomer(),
		inv.FactoringCompany(),
		inv.Shipment(),
		inv.BillingQueueItem(),
		inv.PDFDocument().DocumentType(),
		lines.Order(lines.LineNumber.Asc()),
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*invoice.Invoice, error) {
		entity, err := invoiceDetail().
			Where(invoicedb.Invoices.ID.Eq(req.ID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invoiceErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req invoiceports.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*invoice.Invoice, error) {
		if len(req.InvoiceIDs) == 0 {
			return []*invoice.Invoice{}, nil
		}

		entities, err := invoiceDetail().
			Where(invoicedb.Invoices.ID.In(req.InvoiceIDs...)).
			Limit(len(req.InvoiceIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get invoices by ids", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) ListLineCharges(
	ctx context.Context,
	req *invoiceports.ListInvoiceLineChargesRequest,
) ([]invoiceports.InvoiceLineCharge, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]invoiceports.InvoiceLineCharge, error) {
			if req == nil || len(req.InvoiceIDs) == 0 {
				return []invoiceports.InvoiceLineCharge{}, nil
			}

			il := invoicedb.InvoiceLines
			charges, err := keel.Into[invoiceports.InvoiceLineCharge](ctx, r.db.Keel(ctx), il.
				Pick(il.AdditionalChargeID, il.ShipmentID).
				Where(
					il.InvoiceID.In(req.InvoiceIDs...),
					il.AdditionalChargeID.IsNotNull(),
					il.ShipmentID.IsNotNull(),
				).
				Group(il.AdditionalChargeID, il.ShipmentID))
			if err != nil {
				r.l.Error("failed to list invoice line charges", zap.Error(err))
				return nil, err
			}
			if charges == nil {
				return []invoiceports.InvoiceLineCharge{}, nil
			}

			return charges, nil
		},
	)
}

type chargeNet struct {
	ChargeID pulid.ID
	Net      decimal.Decimal
}

func (r *repository) NetBilledByCharge(
	ctx context.Context,
	req *invoiceports.NetBilledByChargeRequest,
) (map[pulid.ID]decimal.Decimal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]decimal.Decimal, error) {
		if req == nil || len(req.ChargeIDs) == 0 {
			return map[pulid.ID]decimal.Decimal{}, nil
		}

		il := invoicedb.InvoiceLines
		inv := invoicedb.Invoices
		adj := invoiceadjustmentdb.InvoiceAdjustments
		rows, err := keel.Into[chargeNet](ctx, r.db.Keel(ctx), il.
			Pick(il.AdditionalChargeID, il.Amount.Sum()).
			Join(inv, inv.ID.EqCol(il.InvoiceID)).
			LeftJoin(adj, adj.ID.EqCol(inv.SourceInvoiceAdjustmentID)).
			Where(
				il.AdditionalChargeID.In(req.ChargeIDs...),
				inv.Status.Ne(invoice.StatusVoided),
				keel.Or(
					inv.BillType.Ne(billingqueue.BillTypeCreditMemo),
					adj.Kind.In(
						invoiceadjustment.KindCreditOnly,
						invoiceadjustment.KindCreditRebill,
					),
				),
			).
			Group(il.AdditionalChargeID))
		if err != nil {
			r.l.Error("failed to total billed charges", zap.Error(err))
			return nil, err
		}

		net := make(map[pulid.ID]decimal.Decimal, len(rows))
		for _, row := range rows {
			net[row.ChargeID] = row.Net
		}

		return net, nil
	})
}

type shipmentInvoice struct {
	ShipmentID pulid.ID
	InvoiceID  pulid.ID
}

func (r *repository) ListByShipmentIDs(
	ctx context.Context,
	req invoiceports.ListInvoicesByShipmentIDsRequest,
) (map[pulid.ID][]*invoice.Invoice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*invoice.Invoice, error) {
			result := make(map[pulid.ID][]*invoice.Invoice, len(req.ShipmentIDs))
			if len(req.ShipmentIDs) == 0 {
				return result, nil
			}

			il := invoicedb.InvoiceLines
			inv := invoicedb.Invoices
			pairs, err := keel.Into[shipmentInvoice](ctx, r.db.Keel(ctx), keel.UnionAll(
				il.Pick(il.ShipmentID, il.InvoiceID).
					Where(il.ShipmentID.In(req.ShipmentIDs...)).
					Group(il.ShipmentID, il.InvoiceID),
				inv.Pick(inv.ShipmentID, inv.ID).
					Where(inv.ShipmentID.In(req.ShipmentIDs...)),
			))
			if err != nil {
				r.l.Error("failed to list invoice ids by shipment", zap.Error(err))
				return nil, err
			}
			if len(pairs) == 0 {
				return result, nil
			}

			invoiceIDs := make([]pulid.ID, 0, len(pairs))
			seen := make(map[pulid.ID]struct{}, len(pairs))
			for _, pair := range pairs {
				if _, ok := seen[pair.InvoiceID]; ok {
					continue
				}
				seen[pair.InvoiceID] = struct{}{}
				invoiceIDs = append(invoiceIDs, pair.InvoiceID)
			}

			invoices, err := r.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
				TenantInfo: req.TenantInfo,
				InvoiceIDs: invoiceIDs,
			})
			if err != nil {
				return nil, err
			}
			byID := make(map[pulid.ID]*invoice.Invoice, len(invoices))
			for _, entity := range invoices {
				byID[entity.ID] = entity
			}

			linked := make(map[pulid.ID]map[pulid.ID]struct{}, len(req.ShipmentIDs))
			for _, pair := range pairs {
				entity, ok := byID[pair.InvoiceID]
				if !ok {
					continue
				}
				if linked[pair.ShipmentID] == nil {
					linked[pair.ShipmentID] = make(map[pulid.ID]struct{})
				}
				if _, dup := linked[pair.ShipmentID][pair.InvoiceID]; dup {
					continue
				}
				linked[pair.ShipmentID][pair.InvoiceID] = struct{}{}
				result[pair.ShipmentID] = append(result[pair.ShipmentID], entity)
			}

			return result, nil
		},
	)
}

func (r *repository) LockForUpdate(
	ctx context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoice.Invoice, error) {
		inv := invoicedb.Invoices
		lines := inv.Lines()
		entity, err := inv.Where(inv.ID.Eq(req.ID)).
			Load(lines.Order(lines.LineNumber.Asc())).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invoiceErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateEDISendStatus(
	ctx context.Context,
	req invoiceports.UpdateInvoiceEDISendStatusRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		inv := invoicedb.Invoices
		sets := []keel.Assignment{
			inv.EDISendStatus.To(req.Status),
			inv.LastEDIError.To(req.Error),
			inv.Version.ToExpr(inv.Version),
		}
		if req.MessageID.IsNotNil() {
			sets = append(sets, inv.LastEDIMessageID.To(req.MessageID))
		}
		if req.SentAt != nil {
			sets = append(sets, inv.EDISentAt.To(*req.SentAt))
		}

		_, err := inv.Where(inv.ID.Eq(req.InvoiceID)).Update(ctx, r.db.Keel(ctx), sets...)
		return invoiceErrors.Map(err, req.InvoiceID.String())
	})
}

func (r *repository) GetByBillingQueueItemID(
	ctx context.Context,
	req invoiceports.GetInvoiceByBillingQueueItemIDRequest,
) (*invoice.Invoice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*invoice.Invoice, error) {
		inv := invoicedb.Invoices
		entity, err := invoiceDetail().
			Where(
				inv.BillingQueueItemID.Eq(req.BillingQueueItemID),
				inv.Status.Ne(invoice.StatusVoided),
			).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invoiceErrors.Map(err, req.BillingQueueItemID.String())
		}

		return entity, nil
	})
}

func (r *repository) CountPostedReconciliationDiscrepancies(
	ctx context.Context,
	req invoiceports.CountPostedInvoiceReconciliationDiscrepanciesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		n, err := reconciliationDiscrepancies(&req).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, err
		}

		return int(n), nil
	})
}

func reconciliationDiscrepancies(
	req *invoiceports.CountPostedInvoiceReconciliationDiscrepanciesRequest,
) *keel.Query[invoice.Invoice] {
	inv := invoicedb.Invoices
	bqi := billingqueuedb.BillingQueueItems
	legs := shipmentdb.Shipments.As("leg")
	legLines := invoicedb.InvoiceLines.As("legl")
	shp := shipmentdb.Shipments.As("shp")
	orderLines := invoicedb.InvoiceLines.As("ordl")

	splitTotal := keel.Scalar(bqi.Pick(bqi.AllocatedTotalAmount.Sum()).Where(
		bqi.BillToCustomerID.EqCol(inv.CustomerID),
		keel.Or(bqi.InvoiceID.EqCol(inv.ID), bqi.ID.EqCol(inv.BillingQueueItemID)),
		inv.IsSplitBill.IsTrue(),
	))
	legTotal := keel.Scalar(legs.Pick(legs.TotalChargeAmount.Sum()).Where(
		legs.ID.InQuery(legLines.Pick(legLines.ShipmentID).Where(
			legLines.InvoiceID.EqCol(inv.ID),
			legLines.ShipmentID.IsNotNull(),
		)),
	))
	shipmentTotal := keel.Scalar(
		shp.Pick(shp.TotalChargeAmount).Where(shp.ID.EqCol(inv.ShipmentID)),
	)
	orderTotal := keel.Scalar(orderLines.Pick(orderLines.Amount.Sum()).Where(
		orderLines.InvoiceID.EqCol(inv.ID),
		orderLines.ShipmentID.IsNull(),
	))

	sign := keel.DynCase().
		When(keel.DynOf(inv.BillType).Eq(billingqueue.BillTypeCreditMemo), signed(-1)).
		Else(signed(1))
	orderPart := keel.DynCase().
		When(inv.OrderID.IsNotNull(), keel.DynOf(orderTotal).Coalesce(0)).
		Else(signed(0))
	expected := sign.Mul(keel.DynOf(splitTotal).Coalesce(legTotal, shipmentTotal, 0)).Add(orderPart)

	return inv.Where(
		inv.Status.Eq(invoice.StatusPosted),
		inv.PostedAt.IsNotNull(),
		inv.PostedAt.Gte(req.PeriodStartDate),
		inv.PostedAt.Lte(req.PeriodEndDate),
		keel.DynFunc("abs", keel.DynOf(inv.TotalAmount).Sub(expected)).Gt(req.ToleranceAmount),
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoice.Invoice, error) {
		entity.SyncMinorAmounts()
		db := r.db.Keel(ctx)

		if err := invoicedb.Invoices.Create(ctx, db, entity); err != nil {
			r.l.Error("failed to create invoice", zap.Error(err))
			return nil, invoiceErrors.Map(err, entity.ID.String())
		}

		if len(entity.Lines) > 0 {
			for _, line := range entity.Lines {
				line.InvoiceID = entity.ID
			}
			if err := invoicedb.InvoiceLines.CreateMany(ctx, db, entity.Lines); err != nil {
				r.l.Error("failed to create invoice lines", zap.Error(err))
				return nil, invoiceLineErrors.Map(err, entity.ID.String())
			}
		}

		return r.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoice.Invoice, error) {
		entity.SyncMinorAmounts()
		entity.EDISendStatus = ediSendStatusOrDefault(entity.EDISendStatus)

		inv := invoicedb.Invoices
		err := inv.Save(ctx, r.db.Keel(ctx), entity,
			inv.Status,
			inv.PostedAt,
			inv.DueDate,
			inv.AppliedAmount,
			inv.AppliedAmountMinor,
			inv.SettlementStatus,
			inv.DisputeStatus,
			inv.PDFDocumentID,
			inv.SendStatus,
			inv.SentAt,
			inv.SentByID,
			inv.LastSendError,
			inv.LastSendWarning,
			inv.Memo,
			inv.RemittanceInstructions,
			inv.FactoringCompanyID,
			inv.FactoringAgreementID,
			inv.FactoringOverride,
			inv.FactoringOverrideReason,
			inv.EmailSubjectSnapshot,
			inv.EmailBodySnapshot,
			inv.EmailToSnapshot,
			inv.EmailCCSnapshot,
			inv.EmailBCCSnapshot,
			inv.CorrectionGroupID,
			inv.SupersedesInvoiceID,
			inv.SupersededByInvoiceID,
			inv.SourceInvoiceAdjustmentID,
			inv.IsAdjustmentArtifact,
			inv.VoidedAt,
			inv.VoidedByID,
			inv.VoidReason,
			inv.VoidDisposition,
			inv.VoidedByAdjustmentID,
			inv.ReferenceInvoiceID,
			inv.MemoReason,
			inv.MemoKind,
			inv.EDISendStatus,
			inv.LastEDIMessageID,
			inv.EDISentAt,
			inv.LastEDIError,
		)
		if err != nil {
			r.l.Error("failed to update invoice", zap.Error(err))
			return nil, invoiceErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantInfo(entity),
		})
	})
}

func (r *repository) UpsertAttachments(
	ctx context.Context,
	req invoiceports.UpsertInvoiceAttachmentsRequest,
) ([]*invoice.Attachment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*invoice.Attachment, error) {
		db := r.db.Keel(ctx)
		att := invoicedb.Attachments
		if _, err := att.Where(att.InvoiceID.Eq(req.InvoiceID)).Delete(ctx, db); err != nil {
			return nil, attachmentErrors.Map(err, req.InvoiceID.String())
		}

		if len(req.DocumentIDs) > 0 {
			entities := make([]*invoice.Attachment, 0, len(req.DocumentIDs))
			for idx, documentID := range req.DocumentIDs {
				entities = append(entities, &invoice.Attachment{
					OrganizationID: req.OrganizationID,
					BusinessUnitID: req.BusinessUnitID,
					InvoiceID:      req.InvoiceID,
					DocumentID:     documentID,
					Selected:       true,
					SortOrder:      idx + 1,
				})
			}

			if err := att.CreateMany(ctx, db, entities); err != nil {
				return nil, attachmentErrors.Map(err, req.InvoiceID.String())
			}
		}

		return r.ListAttachments(ctx, invoiceports.ListInvoiceEmailAttemptsRequest{
			InvoiceID:  req.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
	})
}

func (r *repository) ListAttachments(
	ctx context.Context,
	req invoiceports.ListInvoiceEmailAttemptsRequest,
) ([]*invoice.Attachment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*invoice.Attachment, error) {
		att := invoicedb.Attachments
		entities, err := att.Where(att.InvoiceID.Eq(req.InvoiceID)).
			Load(att.Document()).
			Order(att.SortOrder.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if entities == nil {
			return []*invoice.Attachment{}, nil
		}

		return entities, nil
	})
}

func emailAttemptDetail() *keel.Query[invoice.EmailAttempt] {
	ea := invoicedb.EmailAttempts
	attachments := ea.Attachments()

	return ea.Load(
		ea.Email(),
		attachments.Order(attachments.CreatedAt.Asc()).
			Load(attachments.Document(), attachments.ShareToken()),
	)
}

func (r *repository) CreateEmailAttempt(
	ctx context.Context,
	attempt *invoice.EmailAttempt,
	attachments []*invoice.EmailAttemptAttachment,
) (*invoice.EmailAttempt, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoice.EmailAttempt, error) {
		db := r.db.Keel(ctx)
		existingID, err := r.emailAttemptIDForMessage(ctx, attempt)
		if err != nil {
			return nil, err
		}

		if existingID.IsNil() {
			if err = insertEmailAttempt(ctx, db, attempt, attachments); err != nil {
				return nil, emailAttemptErrors.Map(err, attempt.ID.String())
			}
		} else {
			attempt.ID = existingID
		}

		ea := invoicedb.EmailAttempts
		entity, err := emailAttemptDetail().
			Where(ea.ID.Eq(attempt.ID), ea.InvoiceID.Eq(attempt.InvoiceID)).
			One(ctx, db)
		if err != nil {
			return nil, emailAttemptErrors.Map(err, attempt.ID.String())
		}

		return entity, nil
	})
}

func insertEmailAttempt(
	ctx context.Context,
	db *keel.DB,
	attempt *invoice.EmailAttempt,
	attachments []*invoice.EmailAttemptAttachment,
) error {
	if err := invoicedb.EmailAttempts.Create(ctx, db, attempt); err != nil {
		return err
	}
	if len(attachments) == 0 {
		return nil
	}
	for _, attachment := range attachments {
		attachment.AttemptID = attempt.ID
	}

	return invoicedb.EmailAttemptAttachments.CreateMany(ctx, db, attachments)
}

func (r *repository) ListEmailAttempts(
	ctx context.Context,
	req invoiceports.ListInvoiceEmailAttemptsRequest,
) (*pagination.ListResult[*invoice.EmailAttempt], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*invoice.EmailAttempt], error) {
			ea := invoicedb.EmailAttempts
			page := pagination.SelectQueryRequest{
				Pagination: pagination.Info{
					Limit:  pagination.DefaultLimit,
					Offset: pagination.DefaultOffset,
				},
			}
			if req.Filter != nil {
				page.Pagination = req.Filter.Pagination
			}

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				emailAttemptDetail().
					Where(ea.InvoiceID.Eq(req.InvoiceID)).
					Order(ea.CreatedAt.Desc()),
				&page,
				nil,
			)
		},
	)
}

func (r *repository) emailAttemptIDForMessage(
	ctx context.Context,
	attempt *invoice.EmailAttempt,
) (pulid.ID, error) {
	if attempt.EmailMessageID.IsNil() {
		return pulid.Nil, nil
	}

	ea := invoicedb.EmailAttempts
	existing, err := ea.Pick(ea.ID).
		Where(ea.EmailMessageID.Eq(attempt.EmailMessageID), ea.InvoiceID.Eq(attempt.InvoiceID)).
		First(ctx, r.db.Keel(ctx))
	if errors.Is(err, keel.ErrNotFound) {
		return pulid.Nil, nil
	}
	if err != nil {
		return pulid.Nil, err
	}

	return existing.ID, nil
}

func (r *repository) SyncEmailAttemptsForMessage(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*invoice.Invoice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*invoice.Invoice, error) {
		db := r.db.Keel(ctx)
		msg, err := emaildb.Messages.Get(ctx, db, messageID)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "EmailMessage")
		}

		ea := invoicedb.EmailAttempts
		attempts, err := ea.Pick(ea.ID, ea.InvoiceID).
			Where(ea.EmailMessageID.Eq(messageID)).
			All(ctx, db)
		if err != nil {
			return nil, err
		}
		if len(attempts) == 0 {
			return []*invoice.Invoice{}, nil
		}

		status := invoiceSendStatusForEmailMessage(msg)
		var sentAt *int64
		if status == invoice.SendStatusSent && msg.SentAt > 0 {
			sentAt = &msg.SentAt
		}

		_, err = ea.Where(ea.EmailMessageID.Eq(messageID)).Update(ctx, db,
			ea.Status.To(status),
			ea.ProviderMessageID.To(msg.ProviderMessageID),
			ea.Error.To(invoiceEmailMessageError(msg)),
			ea.SentAt.ToExpr(keel.Val(sentAt)),
		)
		if err != nil {
			return nil, emailAttemptErrors.Map(err, messageID.String())
		}

		invoiceIDs := make([]pulid.ID, 0, len(attempts))
		seen := make(map[pulid.ID]struct{}, len(attempts))
		for _, attempt := range attempts {
			if _, ok := seen[attempt.InvoiceID]; ok {
				continue
			}
			seen[attempt.InvoiceID] = struct{}{}
			invoiceIDs = append(invoiceIDs, attempt.InvoiceID)
		}

		updated := make([]*invoice.Invoice, 0, len(invoiceIDs))
		for _, invoiceID := range invoiceIDs {
			entity, syncErr := r.syncInvoiceSendStatus(ctx, invoiceID)
			if syncErr != nil {
				return nil, syncErr
			}
			if entity != nil {
				updated = append(updated, entity)
			}
		}

		return updated, nil
	})
}

func (r *repository) syncInvoiceSendStatus(
	ctx context.Context,
	invoiceID pulid.ID,
) (*invoice.Invoice, error) {
	db := r.db.Keel(ctx)
	ea := invoicedb.EmailAttempts
	attempts, err := ea.Where(ea.InvoiceID.Eq(invoiceID)).All(ctx, db)
	if err != nil {
		return nil, err
	}
	if len(attempts) == 0 {
		return nil, nil //nolint:nilnil // an invoice never sent has no send status to record
	}

	status, sentAt, lastError := invoiceSendStatusFromAttempts(
		invoice.LatestSendAttempts(attempts),
	)
	inv := invoicedb.Invoices
	rows, err := inv.Where(inv.ID.Eq(invoiceID)).
		UpdateReturning(ctx, db, inv.SendStatus.To(status), inv.SentAt.ToExpr(keel.Val(sentAt)), inv.LastSendError.To(lastError), inv.Version.ToExpr(inv.Version))
	if err != nil {
		return nil, invoiceErrors.Map(err, invoiceID.String())
	}
	if len(rows) == 0 {
		return nil, nil //nolint:nilnil // the invoice is gone, so there is no row to report
	}

	return rows[0], nil
}

func invoiceSendStatusForEmailMessage(msg *email.Message) invoice.SendStatus {
	switch msg.Status {
	case email.MessageStatusSent,
		email.MessageStatusDelivered,
		email.MessageStatusOpened,
		email.MessageStatusClicked:
		return invoice.SendStatusSent
	case email.MessageStatusFailed,
		email.MessageStatusBounced,
		email.MessageStatusComplained,
		email.MessageStatusSuppressed:
		return invoice.SendStatusFailed
	default:
		return invoice.SendStatusSending
	}
}

func invoiceEmailMessageError(msg *email.Message) string {
	status := invoiceSendStatusForEmailMessage(msg)
	if status != invoice.SendStatusFailed {
		return ""
	}
	if strings.TrimSpace(msg.LastError) != "" {
		return msg.LastError
	}
	return "Email provider reported " + string(msg.Status)
}

func invoiceSendStatusFromAttempts(
	attempts []*invoice.EmailAttempt,
) (invoice.SendStatus, *int64, string) {
	failed := 0
	sending := 0
	var sentAt *int64
	failures := make([]string, 0)
	for _, attempt := range attempts {
		switch attempt.Status {
		case invoice.SendStatusFailed:
			failed++
			if strings.TrimSpace(attempt.Error) != "" {
				failures = append(failures, attempt.Error)
			}
		case invoice.SendStatusSending:
			sending++
		case invoice.SendStatusSent:
			if attempt.SentAt != nil && (sentAt == nil || *attempt.SentAt > *sentAt) {
				sentAt = attempt.SentAt
			}
		}
	}

	lastError := strings.Join(sliceutils.DedupeStrings(failures), "; ")
	switch {
	case sending > 0:
		return invoice.SendStatusSending, sentAt, lastError
	case failed == 0:
		return invoice.SendStatusSent, sentAt, ""
	case failed < len(attempts):
		return invoice.SendStatusPartiallySent, sentAt, lastError
	default:
		return invoice.SendStatusFailed, nil, lastError
	}
}

func (r *repository) CreateDocumentShareToken(
	ctx context.Context,
	token *invoice.DocumentShareToken,
) (*invoice.DocumentShareToken, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoice.DocumentShareToken, error) {
		db := r.db.Keel(ctx)
		if err := invoicedb.DocumentShareTokens.Create(ctx, db, token); err != nil {
			return nil, shareTokenErrors.Map(err, token.ID.String())
		}

		return shareTokenWithDocument(ctx, db, token.ID)
	})
}

func shareTokenWithDocument(
	ctx context.Context,
	db *keel.DB,
	id pulid.ID,
) (*invoice.DocumentShareToken, error) {
	dt := invoicedb.DocumentShareTokens
	entity, err := dt.Where(dt.ID.Eq(id)).Load(dt.Document()).One(ctx, db)
	if err != nil {
		return nil, shareTokenErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) GetDocumentShareToken(
	ctx context.Context,
	req invoiceports.GetInvoiceDocumentShareTokenRequest,
) (*invoice.DocumentShareToken, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve a public invoice document link before its tenant is known",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*invoice.DocumentShareToken, error) {
		dt := invoicedb.DocumentShareTokens
		entity, err := dt.AllTenants().
			Where(dt.TokenHash.Eq(req.TokenHash)).
			Load(dt.Document()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shareTokenErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) RevokeDocumentShareTokens(
	ctx context.Context,
	req invoiceports.RevokeInvoiceDocumentShareTokensRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		dt := invoicedb.DocumentShareTokens
		n, err := dt.Where(dt.InvoiceID.Eq(req.InvoiceID), dt.RevokedAt.IsNull()).
			Update(ctx, r.db.Keel(ctx), dt.RevokedAt.To(timeutils.NowUnix()))
		if err != nil {
			return 0, shareTokenErrors.Map(err, req.InvoiceID.String())
		}

		return n, nil
	})
}

func (r *repository) UpdateDocumentShareToken(
	ctx context.Context,
	token *invoice.DocumentShareToken,
) (*invoice.DocumentShareToken, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoice.DocumentShareToken, error) {
		db := r.db.Keel(ctx)
		dt := invoicedb.DocumentShareTokens
		n, err := dt.Where(dt.ID.Eq(token.ID)).Update(ctx, db,
			dt.DownloadedAt.ToExpr(keel.Val(token.DownloadedAt)),
			dt.RevokedAt.ToExpr(keel.Val(token.RevokedAt)),
		)
		if err != nil {
			return nil, shareTokenErrors.Map(err, token.ID.String())
		}
		if err = dberror.CheckAffected(n, "Document share token", token.ID.String()); err != nil {
			return nil, err
		}

		return shareTokenWithDocument(ctx, db, token.ID)
	})
}

func tenantInfo(entity *invoice.Invoice) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}

func ediSendStatusOrDefault(status invoice.EDISendStatus) invoice.EDISendStatus {
	if status == "" {
		return invoice.EDISendStatusNotSent
	}

	return status
}

func (r *repository) StampExchangeRate(
	ctx context.Context,
	req *kernelports.StampExchangeRateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		inv := invoicedb.Invoices
		n, err := inv.Where(inv.ID.Eq(req.ID)).Update(ctx, r.db.Keel(ctx),
			inv.ExchangeRate.ToExpr(keel.Val(req.Rate)),
			inv.ExchangeRateDate.To(req.Date),
			inv.Version.ToExpr(inv.Version),
		)
		if err != nil {
			return invoiceErrors.Map(err, req.ID.String())
		}

		return dberror.CheckFoundCount(n, "Invoice")
	})
}

func signed(n int64) keel.DynExpr {
	return keel.DynLit(n).Cast(keel.TypeInt64)
}
