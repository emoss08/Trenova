package edimessagerepository

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	serviceFailure214ReviewedTrigger = "Reviewed"
	serviceFailure214ResolvedTrigger = "Resolved"
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

func New(p Params) repositories.EDIMessageRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-message-repository"),
	}
}

func (r *repository) ListMessages(
	ctx context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	entities := make([]*edi.EDIMessage, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDIMessageColumns
	rel := buncolgen.EDIMessageRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ColumnExpr(buncolgen.EDIMessageTable.All()).
		ColumnExpr(`(
			SELECT COUNT(*)
			FROM edi_message_validation_errors AS emve
			WHERE emve.message_id = emsg.id
				AND emve.organization_id = emsg.organization_id
				AND emve.business_unit_id = emsg.business_unit_id
		) AS diagnostic_count`).
		Relation(rel.Partner).
		Relation(rel.PartnerDocumentProfile).
		Relation(rel.Template).
		Apply(buncolgen.EDIMessageApplyTenant(req.Filter.TenantInfo))
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if !req.PartnerID.IsNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.GeneratedFrom > 0 {
		query = query.Where(cols.GeneratedAt.Gte(), req.GeneratedFrom)
	}
	if req.GeneratedTo > 0 {
		query = query.Where(cols.GeneratedAt.Lte(), req.GeneratedTo)
	}
	query = applyMessageArchiveSearch(query, req.Query)
	total, err := query.
		Order(cols.GeneratedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDIMessage]{Items: entities, Total: total}, nil
}

func (r *repository) GetMessageByID(
	ctx context.Context,
	req repositories.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	entity := new(edi.EDIMessage)
	cols := buncolgen.EDIMessageColumns
	rel := buncolgen.EDIMessageRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.Partner).
		Relation(rel.DocumentType).
		Relation(rel.PartnerDocumentProfile).
		Relation(rel.Template).
		Relation(rel.TemplateVersion).
		Relation(buncolgen.Rel(rel.TemplateVersion, buncolgen.EDITemplateVersionRelations.ScriptLibraries), func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr(buncolgen.EDITemplateScriptLibraryColumns.Name.Expr("lower({}) ASC"))
		}).
		Relation(rel.ValidationErrors, func(q *bun.SelectQuery) *bun.SelectQuery {
			errCols := buncolgen.EDIMessageValidationErrorColumns
			return q.Order(errCols.CreatedAt.OrderAsc(), errCols.ID.OrderAsc())
		}).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.EDIMessageApplyTenant(req.TenantInfo)).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIMessage")
	}
	return entity, nil
}

func (r *repository) CreateMessageWithDiagnostics(
	ctx context.Context,
	req repositories.CreateEDIMessageWithDiagnosticsRequest,
) (*edi.EDIMessage, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(req.Message).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		for _, diagnostic := range req.Diagnostics {
			diagnostic.MessageID = req.Message.ID
			diagnostic.BusinessUnitID = req.Message.BusinessUnitID
			diagnostic.OrganizationID = req.Message.OrganizationID
		}
		if len(req.Diagnostics) > 0 {
			if _, err := r.db.DBForContext(c).
				NewInsert().
				Model(&req.Diagnostics).
				Exec(c); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	req.Message.ValidationErrors = req.Diagnostics
	return req.Message, nil
}

func (r *repository) GetServiceFailure214LifecycleMessage(
	ctx context.Context,
	req repositories.GetServiceFailure214LifecycleMessageRequest,
) (*edi.EDIMessage, error) {
	entity := new(edi.EDIMessage)
	cols := buncolgen.EDIMessageColumns
	rel := buncolgen.EDIMessageRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.Partner).
		Relation(rel.PartnerDocumentProfile).
		Where(cols.TransactionSet.Eq(), edi.TransactionSet214).
		Where(cols.Direction.Eq(), edi.DocumentDirectionOutbound).
		Where("payload_snapshot->'shipmentStatus'->>'serviceFailureId' = ?", req.ServiceFailureID.String()).
		Where(
			"payload_snapshot->'shipmentStatus'->'references'->>'serviceFailureId' = ?",
			req.ServiceFailureID.String(),
		).
		Where(
			"payload_snapshot->'shipmentStatus'->'references'->>'serviceFailure214Trigger' = ?",
			req.Trigger,
		).
		Apply(buncolgen.EDIMessageApplyTenant(req.TenantInfo)).
		Order(cols.GeneratedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIMessage")
	}
	return entity, nil
}

func (r *repository) GetServiceFailure214Status(
	ctx context.Context,
	req repositories.GetServiceFailure214StatusRequest,
) (*repositories.ServiceFailure214Status, error) {
	status := &repositories.ServiceFailure214Status{ServiceFailureID: req.ServiceFailureID}
	err := r.db.DBForContext(ctx).NewRaw(`
		WITH lifecycle_messages AS (
			SELECT
				emsg.id,
				emsg.status,
				emsg.delivery_status,
				emsg.ack_status,
				emsg.generated_at,
				emsg.payload_snapshot->'shipmentStatus'->'references'->>'serviceFailure214Trigger' AS trigger
			FROM edi_messages AS emsg
			WHERE emsg.organization_id = ?
				AND emsg.business_unit_id = ?
				AND emsg.transaction_set = ?
				AND emsg.direction = ?
				AND emsg.payload_snapshot->'shipmentStatus'->>'serviceFailureId' = ?
				AND emsg.payload_snapshot->'shipmentStatus'->'references'->>'serviceFailureId' = ?
		),
		last_message AS (
			SELECT *
			FROM lifecycle_messages
			ORDER BY generated_at DESC, id DESC
			LIMIT 1
		),
		last_diagnostic AS (
			SELECT emve.message
			FROM edi_message_validation_errors AS emve
			JOIN last_message lm ON lm.id = emve.message_id
			WHERE emve.organization_id = ?
				AND emve.business_unit_id = ?
				AND emve.severity = ?
			ORDER BY emve.created_at DESC, emve.id DESC
			LIMIT 1
		)
		SELECT
			? AS service_failure_id,
			COALESCE((
				SELECT id FROM lifecycle_messages
				WHERE trigger = ?
				ORDER BY generated_at DESC, id DESC
				LIMIT 1
			), '') AS reviewed_message_id,
			COALESCE((
				SELECT id FROM lifecycle_messages
				WHERE trigger = ?
				ORDER BY generated_at DESC, id DESC
				LIMIT 1
			), '') AS resolved_message_id,
			COALESCE((SELECT id FROM last_message), '') AS last_message_id,
			COALESCE((SELECT status FROM last_message), '') AS generated_status,
			COALESCE((SELECT delivery_status FROM last_message), '') AS delivery_status,
			COALESCE((SELECT ack_status FROM last_message), '') AS ack_status,
			COALESCE((SELECT message FROM last_diagnostic), '') AS last_diagnostic,
			COALESCE((SELECT generated_at FROM last_message), 0) AS last_generated_at
	`,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		edi.TransactionSet214,
		edi.DocumentDirectionOutbound,
		req.ServiceFailureID.String(),
		req.ServiceFailureID.String(),
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		edi.ValidationSeverityError,
		req.ServiceFailureID,
		serviceFailure214ReviewedTrigger,
		serviceFailure214ResolvedTrigger,
	).Scan(ctx, status)
	if err != nil {
		return nil, err
	}
	return status, nil
}

func (r *repository) UpdateMessageDelivery(
	ctx context.Context,
	req *repositories.UpdateEDIMessageDeliveryRequest,
) (*edi.EDIMessage, error) {
	if req == nil {
		return nil, errors.New("EDI message delivery update request is required")
	}
	entity := new(edi.EDIMessage)
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("delivery_status = ?", req.DeliveryStatus).
		Set("delivery_remote_path = ?", req.DeliveryRemotePath).
		Set("delivery_last_attempt_at = ?", req.DeliveryLastAttemptAt).
		Set("delivery_sent_at = ?", req.DeliverySentAt).
		Set("delivery_last_error = ?", req.DeliveryLastError).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Returning("*")
	if req.AS2MessageID != "" {
		query = query.Set("as2_message_id = ?", req.AS2MessageID)
	}
	if req.AS2MIC != "" {
		query = query.Set("as2_mic = ?", req.AS2MIC)
	}
	if req.IncrementAttempts {
		query = query.Set("delivery_attempts = delivery_attempts + 1")
	}
	results, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDIMessage", req.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) GetOutboundMessageByAS2MessageID(
	ctx context.Context,
	as2MessageID string,
) (*edi.EDIMessage, error) {
	entity := new(edi.EDIMessage)
	cols := buncolgen.EDIMessageColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.AS2MessageID.Eq(), as2MessageID).
		Where(cols.Direction.Eq(), edi.DocumentDirectionOutbound).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIMessage")
	}
	return entity, nil
}

func (r *repository) GetDeliveryStatusCounts(
	ctx context.Context,
	req repositories.GetEDIMessageStatusCountsRequest,
) (map[edi.MessageDeliveryStatus]int, error) {
	cols := buncolgen.EDIMessageColumns
	var rows []struct {
		DeliveryStatus edi.MessageDeliveryStatus `bun:"delivery_status"`
		Count          int                       `bun:"count"`
	}
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIMessage)(nil)).
		ColumnExpr(cols.DeliveryStatus.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Where(cols.DeliveryStatus.IsNotNull()).
		Apply(buncolgen.EDIMessageApplyTenant(req.TenantInfo)).
		GroupExpr(cols.DeliveryStatus.Qualified())
	if req.Since > 0 {
		query = query.Where(cols.GeneratedAt.Gte(), req.Since)
	}
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, err
	}
	counts := make(map[edi.MessageDeliveryStatus]int, len(rows))
	for _, row := range rows {
		counts[row.DeliveryStatus] = row.Count
	}
	return counts, nil
}

func (r *repository) GetAckStatusCounts(
	ctx context.Context,
	req repositories.GetEDIMessageStatusCountsRequest,
) (map[edi.MessageAcknowledgmentStatus]int, error) {
	cols := buncolgen.EDIMessageColumns
	var rows []struct {
		AckStatus edi.MessageAcknowledgmentStatus `bun:"ack_status"`
		Count     int                             `bun:"count"`
	}
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIMessage)(nil)).
		ColumnExpr(cols.AckStatus.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Where(cols.AckStatus.IsNotNull()).
		Apply(buncolgen.EDIMessageApplyTenant(req.TenantInfo)).
		GroupExpr(cols.AckStatus.Qualified())
	if req.Since > 0 {
		query = query.Where(cols.GeneratedAt.Gte(), req.Since)
	}
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, err
	}
	counts := make(map[edi.MessageAcknowledgmentStatus]int, len(rows))
	for _, row := range rows {
		counts[row.AckStatus] = row.Count
	}
	return counts, nil
}

func (r *repository) GetOverdueAckCount(
	ctx context.Context,
	req repositories.GetEDIOverdueAckCountRequest,
) (int, error) {
	cols := buncolgen.EDIMessageColumns
	return r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIMessage)(nil)).
		Where(cols.AckStatus.Eq(), edi.MessageAcknowledgmentStatusPending).
		Where(cols.GeneratedAt.Lt(), req.PendingSince).
		Apply(buncolgen.EDIMessageApplyTenant(req.TenantInfo)).
		Count(ctx)
}

func (r *repository) ListRecentDeadLettered(
	ctx context.Context,
	req *repositories.ListRecentEDIMessageFailuresRequest,
) ([]*edi.EDIMessage, error) {
	cols := buncolgen.EDIMessageColumns
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	entities := make([]*edi.EDIMessage, 0, limit)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ExcludeColumn("raw_x12", "payload_snapshot").
		Relation("Partner").
		Where(cols.DeliveryStatus.Eq(), edi.MessageDeliveryStatusDeadLettered).
		Apply(buncolgen.EDIMessageApplyTenant(req.TenantInfo)).
		Order(cols.UpdatedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) GetOutboundMessageForAck(
	ctx context.Context,
	req repositories.GetEDIOutboundMessageForAckRequest,
) (*edi.EDIMessage, error) {
	entity := new(edi.EDIMessage)
	cols := buncolgen.EDIMessageColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.EDIPartnerID.Eq(), req.PartnerID).
		Where(cols.Direction.Eq(), edi.DocumentDirectionOutbound).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(
			"ltrim(emsg.group_control_number, '0') = ltrim(?, '0')",
			req.GroupControlNumber,
		).
		Where(
			"ltrim(emsg.transaction_control_number, '0') = ltrim(?, '0')",
			req.TransactionControlNumber,
		)
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	err := query.
		Order(cols.GeneratedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIMessage")
	}
	return entity, nil
}

func (r *repository) UpdateMessageAcknowledgment(
	ctx context.Context,
	req *repositories.UpdateEDIMessageAcknowledgmentRequest,
) (*edi.EDIMessage, error) {
	if req == nil {
		return nil, errors.New("EDI message acknowledgment update request is required")
	}
	entity := new(edi.EDIMessage)
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("ack_status = ?", req.AckStatus).
		Set("ack_message_id = ?", req.AckMessageID).
		Set("ack_received_at = ?", req.AckReceivedAt).
		Set("ack_last_error = ?", req.AckLastError).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDIMessage", req.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func applyMessageArchiveSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	cols := buncolgen.EDIMessageColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.ID.LowerLike(), term).
			WhereOr(cols.ShipmentID.LowerLike(), term).
			WhereOr(cols.TransferID.LowerLike(), term).
			WhereOr(cols.InterchangeControlNumber.LowerLike(), term).
			WhereOr(cols.GroupControlNumber.LowerLike(), term).
			WhereOr(cols.TransactionControlNumber.LowerLike(), term)
	})
}

func (r *repository) ListMessagesCursor(
	ctx context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.CursorListResult[*edi.EDIMessage], error) {
	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*edi.EDIMessage)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFiltersWithoutSort(
				sq,
				"emsg",
				req.Filter,
				(*edi.EDIMessage)(nil),
			)
			return applyMessageListFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*edi.EDIMessage]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*edi.EDIMessage) *bun.SelectQuery {
			return dba.
				NewSelect().
				Model(entities).
				ExcludeColumn("raw_x12", "payload_snapshot").
				Relation(buncolgen.EDIMessageRelations.Partner)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq,
				"emsg",
				req.Filter,
				req.Cursor,
				(*edi.EDIMessage)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			return applyMessageListFilters(sq, req), nil
		},
	})
}

func applyMessageListFilters(
	q *bun.SelectQuery,
	req *repositories.ListEDIMessagesRequest,
) *bun.SelectQuery {
	cols := buncolgen.EDIMessageColumns
	if req.TransactionSet != "" {
		q = q.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		q = q.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.PartnerID.IsNotNil() {
		q = q.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}
	return q
}
