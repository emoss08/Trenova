package invoicerepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.InvoiceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListInvoicesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"inv",
		req.Filter,
		(*invoice.Invoice)(nil),
	)

	q = q.Relation("Customer").Relation("Shipment")

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	entities := make([]*invoice.Invoice, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*invoice.Invoice]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListInvoiceConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.InvoiceTable.Alias,
		req.Filter,
		req.Cursor,
		(*invoice.Invoice)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListInvoiceConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.InvoiceTable.Alias,
		req.Filter,
		(*invoice.Invoice)(nil),
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListInvoiceConnectionRequest,
) (*pagination.CursorListResult[*invoice.Invoice], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*invoice.Invoice)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count invoices", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*invoice.Invoice]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*invoice.Invoice) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.InvoiceTable.All()).
					Relation("Customer")
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan invoices", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	entity := new(invoice.Invoice)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("inv.id = ?", req.ID).
		Where("inv.organization_id = ?", req.TenantInfo.OrgID).
		Where("inv.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Customer").
		Relation("Shipment").
		Relation("BillingQueueItem").
		Relation("PDFDocument").
		Relation("PDFDocument.DocumentType").
		Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("invl.line_number ASC")
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Invoice")
	}

	return entity, nil
}

func (r *repository) GetByBillingQueueItemID(
	ctx context.Context,
	req repositories.GetInvoiceByBillingQueueItemIDRequest,
) (*invoice.Invoice, error) {
	var entity invoice.Invoice
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entity).
		Where("inv.billing_queue_item_id = ?", req.BillingQueueItemID).
		Where("inv.organization_id = ?", req.TenantInfo.OrgID).
		Where("inv.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Invoice")
	}

	return r.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.ID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) CountPostedReconciliationDiscrepancies(
	ctx context.Context,
	req repositories.CountPostedInvoiceReconciliationDiscrepanciesRequest,
) (int, error) {
	return r.db.DBForContext(ctx).
		NewSelect().
		Model((*invoice.Invoice)(nil)).
		Join("JOIN shipments AS shp ON shp.id = inv.shipment_id AND shp.organization_id = inv.organization_id AND shp.business_unit_id = inv.business_unit_id").
		Where("inv.organization_id = ?", req.OrgID).
		Where("inv.business_unit_id = ?", req.BuID).
		Where("inv.status = ?", invoice.StatusPosted).
		Where("inv.posted_at IS NOT NULL").
		Where("inv.posted_at >= ?", req.PeriodStartDate).
		Where("inv.posted_at <= ?", req.PeriodEndDate).
		Where(
			`ABS(
				inv.total_amount - CASE
					WHEN inv.bill_type = ? THEN COALESCE(shp.total_charge_amount, 0) * -1
					ELSE COALESCE(shp.total_charge_amount, 0)
				END
			) > ?`,
			billingqueue.BillTypeCreditMemo,
			req.ToleranceAmount,
		).
		Count(ctx)
}

func (r *repository) Create(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	entity.SyncMinorAmounts()

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(txCtx).NewInsert().Model(entity).Exec(txCtx); err != nil {
			return fmt.Errorf("insert invoice: %w", err)
		}

		if len(entity.Lines) > 0 {
			for _, line := range entity.Lines {
				line.InvoiceID = entity.ID
				line.OrganizationID = entity.OrganizationID
				line.BusinessUnitID = entity.BusinessUnitID
			}

			if _, err := r.db.DBForContext(txCtx).
				NewInsert().
				Model(&entity.Lines).
				Exec(txCtx); err != nil {
				return fmt.Errorf("insert invoice lines: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	entity.SyncMinorAmounts()

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("inv.id = ?", entity.ID).
		Where("inv.organization_id = ?", entity.OrganizationID).
		Where("inv.business_unit_id = ?", entity.BusinessUnitID).
		Where("inv.version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("posted_at = ?", entity.PostedAt).
		Set("due_date = ?", entity.DueDate).
		Set("applied_amount = ?", entity.AppliedAmount).
		Set("applied_amount_minor = ?", entity.AppliedAmountMinor).
		Set("settlement_status = ?", entity.SettlementStatus).
		Set("dispute_status = ?", entity.DisputeStatus).
		Set("pdf_document_id = ?", entity.PDFDocumentID).
		Set("send_status = ?", entity.SendStatus).
		Set("sent_at = ?", entity.SentAt).
		Set("sent_by_id = ?", entity.SentByID).
		Set("last_send_error = ?", entity.LastSendError).
		Set("last_send_warning = ?", entity.LastSendWarning).
		Set("memo = ?", entity.Memo).
		Set("remittance_instructions = ?", entity.RemittanceInstructions).
		Set("email_subject_snapshot = ?", entity.EmailSubjectSnapshot).
		Set("email_body_snapshot = ?", entity.EmailBodySnapshot).
		Set("email_to_snapshot = ?", pgdialect.Array(entity.EmailToSnapshot)).
		Set("email_cc_snapshot = ?", pgdialect.Array(entity.EmailCCSnapshot)).
		Set("email_bcc_snapshot = ?", pgdialect.Array(entity.EmailBCCSnapshot)).
		Set("correction_group_id = ?", entity.CorrectionGroupID).
		Set("supersedes_invoice_id = ?", entity.SupersedesInvoiceID).
		Set("superseded_by_invoice_id = ?", entity.SupersededByInvoiceID).
		Set("source_invoice_adjustment_id = ?", entity.SourceInvoiceAdjustmentID).
		Set("is_adjustment_artifact = ?", entity.IsAdjustmentArtifact).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update invoice: %w", err)
	}

	if err = dberror.CheckRowsAffected(result, "Invoice", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) UpsertAttachments(
	ctx context.Context,
	req repositories.UpsertInvoiceAttachmentsRequest,
) ([]*invoice.Attachment, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, deleteErr := r.db.DBForContext(txCtx).
			NewDelete().
			Model((*invoice.Attachment)(nil)).
			Where("invoice_id = ?", req.InvoiceID).
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Exec(txCtx); deleteErr != nil {
			return fmt.Errorf("delete invoice attachments: %w", deleteErr)
		}

		if len(req.DocumentIDs) == 0 {
			return nil
		}

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

		if _, insertErr := r.db.DBForContext(txCtx).NewInsert().Model(&entities).Exec(txCtx); insertErr != nil {
			return fmt.Errorf("insert invoice attachments: %w", insertErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.ListAttachments(ctx, repositories.ListInvoiceEmailAttemptsRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) ListAttachments(
	ctx context.Context,
	req repositories.ListInvoiceEmailAttemptsRequest,
) ([]*invoice.Attachment, error) {
	entities := make([]*invoice.Attachment, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("inva.invoice_id = ?", req.InvoiceID).
		Where("inva.organization_id = ?", req.TenantInfo.OrgID).
		Where("inva.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Document").
		Order("inva.sort_order ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) CreateEmailAttempt(
	ctx context.Context,
	attempt *invoice.EmailAttempt,
	attachments []*invoice.EmailAttemptAttachment,
) (*invoice.EmailAttempt, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, insertErr := r.db.DBForContext(txCtx).NewInsert().Model(attempt).Exec(txCtx); insertErr != nil {
			return fmt.Errorf("insert invoice email attempt: %w", insertErr)
		}
		if len(attachments) == 0 {
			return nil
		}
		for _, attachment := range attachments {
			attachment.AttemptID = attempt.ID
			attachment.OrganizationID = attempt.OrganizationID
			attachment.BusinessUnitID = attempt.BusinessUnitID
		}
		if _, insertErr := r.db.DBForContext(txCtx).
			NewInsert().
			Model(&attachments).
			Exec(txCtx); insertErr != nil {
			return fmt.Errorf("insert invoice email attempt attachments: %w", insertErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	entity := new(invoice.EmailAttempt)
	err = r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("inea.id = ?", attempt.ID).
		Where("inea.invoice_id = ?", attempt.InvoiceID).
		Where("inea.organization_id = ?", attempt.OrganizationID).
		Where("inea.business_unit_id = ?", attempt.BusinessUnitID).
		Relation("Email").
		Relation("Attachments", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("ineaa.created_at ASC").Relation("Document").Relation("ShareToken")
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) ListEmailAttempts(
	ctx context.Context,
	req repositories.ListInvoiceEmailAttemptsRequest,
) (*pagination.ListResult[*invoice.EmailAttempt], error) {
	limit := pagination.DefaultLimit
	offset := pagination.DefaultOffset
	if req.Filter != nil {
		limit = req.Filter.Pagination.SafeLimit()
		offset = req.Filter.Pagination.SafeOffset()
	}

	entities := make([]*invoice.EmailAttempt, 0, limit)
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("inea.invoice_id = ?", req.InvoiceID).
		Where("inea.organization_id = ?", req.TenantInfo.OrgID).
		Where("inea.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Email").
		Relation("Attachments", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("ineaa.created_at ASC").Relation("Document").Relation("ShareToken")
		}).
		Order("inea.created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*invoice.EmailAttempt]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) SyncEmailAttemptsForMessage(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	msg := new(email.Message)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(msg).
		Where("em.id = ?", messageID).
		Where("em.organization_id = ?", tenantInfo.OrgID).
		Where("em.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx); err != nil {
		return dberror.HandleNotFoundError(err, "EmailMessage")
	}

	attempts := make([]*invoice.EmailAttempt, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&attempts).
		Where("inea.email_message_id = ?", messageID).
		Where("inea.organization_id = ?", tenantInfo.OrgID).
		Where("inea.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx); err != nil {
		return err
	}
	if len(attempts) == 0 {
		return nil
	}

	status := invoiceSendStatusForEmailMessage(msg)
	errorMessage := invoiceEmailMessageError(msg)
	var sentAt *int64
	if status == invoice.SendStatusSent && msg.SentAt > 0 {
		sentAt = &msg.SentAt
	}

	return r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(txCtx).
			NewUpdate().
			Model((*invoice.EmailAttempt)(nil)).
			Set("status = ?", status).
			Set("provider_message_id = ?", msg.ProviderMessageID).
			Set("error = ?", errorMessage).
			Set("sent_at = ?", sentAt).
			Where("email_message_id = ?", messageID).
			Where("organization_id = ?", tenantInfo.OrgID).
			Where("business_unit_id = ?", tenantInfo.BuID).
			Exec(txCtx); err != nil {
			return fmt.Errorf("sync invoice email attempts: %w", err)
		}

		invoiceIDs := make(map[pulid.ID]struct{}, len(attempts))
		for _, attempt := range attempts {
			invoiceIDs[attempt.InvoiceID] = struct{}{}
		}
		for invoiceID := range invoiceIDs {
			if err := r.syncInvoiceSendStatus(txCtx, invoiceID, tenantInfo); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) syncInvoiceSendStatus(
	ctx context.Context,
	invoiceID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	attempts := make([]*invoice.EmailAttempt, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&attempts).
		Where("inea.invoice_id = ?", invoiceID).
		Where("inea.organization_id = ?", tenantInfo.OrgID).
		Where("inea.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx); err != nil {
		return err
	}
	if len(attempts) == 0 {
		return nil
	}

	status, sentAt, lastError := invoiceSendStatusFromAttempts(attempts)
	if _, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*invoice.Invoice)(nil)).
		Set("send_status = ?", status).
		Set("sent_at = ?", sentAt).
		Set("last_send_error = ?", lastError).
		Where("id = ?", invoiceID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx); err != nil {
		return fmt.Errorf("sync invoice send status: %w", err)
	}
	return nil
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
	errors := make([]string, 0)
	for _, attempt := range attempts {
		switch attempt.Status {
		case invoice.SendStatusFailed:
			failed++
			if strings.TrimSpace(attempt.Error) != "" {
				errors = append(errors, attempt.Error)
			}
		case invoice.SendStatusSending:
			sending++
		case invoice.SendStatusSent:
			if attempt.SentAt != nil && (sentAt == nil || *attempt.SentAt > *sentAt) {
				sentAt = attempt.SentAt
			}
		}
	}

	lastError := strings.Join(errors, "; ")
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
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(token).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert invoice document share token: %w", err)
	}
	return r.GetDocumentShareToken(ctx, repositories.GetInvoiceDocumentShareTokenRequest{
		TokenHash: token.TokenHash,
	})
}

func (r *repository) GetDocumentShareToken(
	ctx context.Context,
	req repositories.GetInvoiceDocumentShareTokenRequest,
) (*invoice.DocumentShareToken, error) {
	entity := new(invoice.DocumentShareToken)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("indst.token_hash = ?", req.TokenHash).
		Relation("Document").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document share token")
	}
	return entity, nil
}

func (r *repository) UpdateDocumentShareToken(
	ctx context.Context,
	token *invoice.DocumentShareToken,
) (*invoice.DocumentShareToken, error) {
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(token).
		Where("indst.id = ?", token.ID).
		Where("indst.organization_id = ?", token.OrganizationID).
		Where("indst.business_unit_id = ?", token.BusinessUnitID).
		Set("downloaded_at = ?", token.DownloadedAt).
		Set("revoked_at = ?", token.RevokedAt).
		Set("updated_at = ?", timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update invoice document share token: %w", err)
	}
	if err = dberror.CheckRowsAffected(result, "Document share token", token.ID.String()); err != nil {
		return nil, err
	}
	return r.GetDocumentShareToken(ctx, repositories.GetInvoiceDocumentShareTokenRequest{
		TokenHash: token.TokenHash,
	})
}

func tenantInfo(entity *invoice.Invoice) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}
