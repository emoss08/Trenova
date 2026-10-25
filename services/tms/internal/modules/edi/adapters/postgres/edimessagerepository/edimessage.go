package edimessagerepository

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	serviceFailure214ReviewedTrigger = "Reviewed"
	serviceFailure214ResolvedTrigger = "Resolved"
	serviceFailure214ReviewedAlias   = "sf_reviewed"
	serviceFailure214ResolvedAlias   = "sf_resolved"

	defaultRecentFailuresLimit = 10
	defaultBucketSeconds       = 3600
)

var messageErrors = dberror.Mapping{Entity: "EDIMessage"}

var (
	errDeliveryRequestRequired = errors.New("EDI message delivery update request is required")
	errAckRequestRequired      = errors.New("EDI message acknowledgment update request is required")
)

var pendingDeliveryStatuses = []edi.MessageDeliveryStatus{
	edi.MessageDeliveryStatusQueued,
	edi.MessageDeliveryStatusSending,
	edi.MessageDeliveryStatusFailed,
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

func New(p Params) ediports.EDIMessageRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-message-repository"),
	}
}

func (r *repository) ListMessages(
	ctx context.Context,
	req *ediports.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIMessage], error) {
			return listquery.Offset(ctx, r.db.Keel(ctx), archiveQuery(req), req.Filter.Pagination)
		},
	)
}

func archiveQuery(req *ediports.ListEDIMessagesRequest) *keel.Query[*edi.EDIMessage] {
	m := edidb.EDIMessages
	ve := edidb.EDIMessageValidationErrors
	search := strings.TrimSpace(req.Query)

	return m.Where(messageListFilters(req)...).
		Where(
			keel.When(req.GeneratedFrom > 0, m.GeneratedAt.Gte(req.GeneratedFrom)),
			keel.When(req.GeneratedTo > 0, m.GeneratedAt.Lte(req.GeneratedTo)),
			keel.When(search != "", keel.Or(
				m.ID.ContainsFold(search),
				m.ShipmentID.ContainsFold(search),
				m.TransferID.ContainsFold(search),
				m.InterchangeControlNumber.ContainsFold(search),
				m.GroupControlNumber.ContainsFold(search),
				m.TransactionControlNumber.ContainsFold(search),
			)),
		).
		Compute(m.DiagnosticCount, keel.Scalar(ve.Pick(keel.Count()).Where(ve.MessageID.EqCol(m.ID)))).
		Load(m.Partner(), m.PartnerDocumentProfile(), m.Template()).
		Order(m.GeneratedAt.Desc())
}

func (r *repository) ListMessagesCursor(
	ctx context.Context,
	req *ediports.ListEDIMessagesRequest,
) (*pagination.CursorListResult[*edi.EDIMessage], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDIMessage], error) {
			m := edidb.EDIMessages
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				m.Omit(m.RawX12, m.PayloadSnapshot).Where(messageListFilters(req)...),
				&listquery.Request[edi.EDIMessage]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*edi.EDIMessage]) *keel.Query[*edi.EDIMessage] {
						return q.Load(m.Partner())
					},
				},
			)
		},
	)
}

func messageListFilters(req *ediports.ListEDIMessagesRequest) []keel.Pred {
	m := edidb.EDIMessages
	return []keel.Pred{
		keel.When(req.TransactionSet != "", m.TransactionSet.Eq(req.TransactionSet)),
		keel.When(req.Direction != "", m.Direction.Eq(req.Direction)),
		keel.When(req.PartnerID.IsNotNil(), m.EDIPartnerID.Eq(req.PartnerID)),
		keel.When(req.Status != "", m.Status.Eq(req.Status)),
		keel.When(req.AckStatus != "", m.AckStatus.Eq(req.AckStatus)),
		keel.When(req.ShipmentID.IsNotNil(), m.ShipmentID.Eq(req.ShipmentID)),
		keel.When(req.InvoiceID.IsNotNil(), m.InvoiceID.Eq(req.InvoiceID)),
		keel.When(req.CreatedAfter > 0, m.CreatedAt.Gte(req.CreatedAfter)),
		keel.When(req.CreatedBefore > 0, m.CreatedAt.Lt(req.CreatedBefore)),
	}
}

func (r *repository) GetMessageByID(
	ctx context.Context,
	req ediports.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		m := edidb.EDIMessages
		tv := m.TemplateVersion()
		libraries := tv.ScriptLibraries()
		diagnostics := m.ValidationErrors()

		entity, err := m.Where(m.ID.Eq(req.ID)).
			Load(
				m.Partner(),
				m.DocumentType(),
				m.PartnerDocumentProfile(),
				m.Template(),
				tv,
				libraries.Order(libraries.Name.Lower().Asc()),
				diagnostics.Order(diagnostics.CreatedAt.Asc(), diagnostics.ID.Asc()),
			).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetMessageByIdempotencyKey(
	ctx context.Context,
	req ediports.GetEDIMessageByIdempotencyKeyRequest,
) (*edi.EDIMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		m := edidb.EDIMessages
		entity, err := m.Where(m.IdempotencyKey.Eq(req.IdempotencyKey)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.IdempotencyKey)
		}

		return entity, nil
	})
}

func (r *repository) CreateMessageWithDiagnostics(
	ctx context.Context,
	req ediports.CreateEDIMessageWithDiagnosticsRequest,
) (*edi.EDIMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		db := r.db.Keel(ctx)
		if err := edidb.EDIMessages.Create(ctx, db, req.Message); err != nil {
			return nil, messageErrors.Map(err, req.Message.ID.String())
		}

		for _, diagnostic := range req.Diagnostics {
			diagnostic.MessageID = req.Message.ID
			diagnostic.BusinessUnitID = req.Message.BusinessUnitID
			diagnostic.OrganizationID = req.Message.OrganizationID
		}
		if len(req.Diagnostics) > 0 {
			if err := edidb.EDIMessageValidationErrors.CreateMany(
				ctx,
				db,
				req.Diagnostics,
			); err != nil {
				return nil, messageErrors.Map(err, req.Message.ID.String())
			}
		}

		req.Message.ValidationErrors = req.Diagnostics
		return req.Message, nil
	})
}

func (r *repository) GetServiceFailure214LifecycleMessage(
	ctx context.Context,
	req ediports.GetServiceFailure214LifecycleMessageRequest,
) (*edi.EDIMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		entity, err := lifecycleMessageQuery(&req).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.ServiceFailureID.String())
		}

		return entity, nil
	})
}

func lifecycleMessageQuery(
	req *ediports.GetServiceFailure214LifecycleMessageRequest,
) *keel.Query[*edi.EDIMessage] {
	m := edidb.EDIMessages
	failureID := req.ServiceFailureID.String()

	return m.Where(serviceFailure214Lifecycle(m, failureID)...).
		Where(serviceFailure214Trigger(m).Eq(req.Trigger)).
		Load(m.Partner(), m.PartnerDocumentProfile()).
		Order(m.GeneratedAt.Desc())
}

func serviceFailure214Lifecycle(m *edidb.EDIMessagesTable, failureID string) []keel.Pred {
	return []keel.Pred{
		m.TransactionSet.Eq(edi.TransactionSet214),
		m.Direction.Eq(edi.DocumentDirectionOutbound),
		m.PayloadSnapshot.Text("shipmentStatus", "serviceFailureId").Eq(failureID),
		m.PayloadSnapshot.Text("shipmentStatus", "references", "serviceFailureId").Eq(failureID),
	}
}

func serviceFailure214Trigger(m *edidb.EDIMessagesTable) keel.StringExpr {
	return m.PayloadSnapshot.Text("shipmentStatus", "references", "serviceFailure214Trigger")
}

func serviceFailure214LatestFor(alias, failureID, trigger string) keel.StringExpr {
	m := edidb.EDIMessages.As(alias)

	return keel.StringOf(keel.Scalar(m.
		Pick(m.ID).
		Where(serviceFailure214Lifecycle(m, failureID)...).
		Where(serviceFailure214Trigger(m).Eq(trigger)).
		Order(m.GeneratedAt.Desc(), m.ID.Desc()).
		Limit(1))).Coalesce("")
}

func serviceFailure214StatusQuery(failureID string) *keel.Query[*edi.EDIMessage] {
	m := edidb.EDIMessages
	e := edidb.EDIMessageValidationErrors

	return m.
		Pick(
			keel.Text(failureID),
			serviceFailure214LatestFor(
				serviceFailure214ReviewedAlias,
				failureID,
				serviceFailure214ReviewedTrigger,
			),
			serviceFailure214LatestFor(
				serviceFailure214ResolvedAlias,
				failureID,
				serviceFailure214ResolvedTrigger,
			),
			m.ID,
			keel.CastText(m.Status).Coalesce(""),
			keel.CastText(m.DeliveryStatus).Coalesce(""),
			keel.CastText(m.AckStatus).Coalesce(""),
			keel.StringOf(keel.Scalar(e.
				Pick(e.Message).
				Where(e.MessageID.EqCol(m.ID), e.Severity.Eq(edi.ValidationSeverityError)).
				Order(e.CreatedAt.Desc(), e.ID.Desc()).
				Limit(1))).Coalesce(""),
			m.GeneratedAt.Coalesce(0),
		).
		Where(serviceFailure214Lifecycle(m, failureID)...).
		Order(m.GeneratedAt.Desc(), m.ID.Desc()).
		Limit(1)
}

func (r *repository) GetServiceFailure214Status(
	ctx context.Context,
	req ediports.GetServiceFailure214StatusRequest,
) (*ediports.ServiceFailure214StatusRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*ediports.ServiceFailure214StatusRow, error) {
			rows, err := keel.Into[ediports.ServiceFailure214StatusRow](
				ctx,
				r.db.Keel(ctx),
				serviceFailure214StatusQuery(req.ServiceFailureID.String()),
			)
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				return &ediports.ServiceFailure214StatusRow{
					ServiceFailureID: req.ServiceFailureID,
				}, nil
			}

			return &rows[0], nil
		},
	)
}

func (r *repository) UpdateMessageDelivery(
	ctx context.Context,
	req *ediports.UpdateEDIMessageDeliveryRequest,
) (*edi.EDIMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		if req == nil {
			return nil, errDeliveryRequestRequired
		}

		m := edidb.EDIMessages
		sets := []keel.Assignment{
			m.DeliveryStatus.To(req.DeliveryStatus),
			m.DeliveryRemotePath.To(req.DeliveryRemotePath),
			setOptional(m.DeliveryLastAttemptAt, req.DeliveryLastAttemptAt),
			setOptional(m.DeliverySentAt, req.DeliverySentAt),
			m.DeliveryLastError.To(req.DeliveryLastError),
			m.Version.ToExpr(m.Version),
		}
		if req.AS2MessageID != "" {
			sets = append(sets, m.AS2MessageID.To(req.AS2MessageID))
		}
		if req.AS2MIC != "" {
			sets = append(sets, m.AS2MIC.To(req.AS2MIC))
		}
		if req.IncrementAttempts {
			sets = append(sets, m.DeliveryAttempts.Add(1))
		}

		return updateOne(ctx, r.db.Keel(ctx), req.ID, sets)
	})
}

func (r *repository) UpdateMessageAcknowledgment(
	ctx context.Context,
	req *ediports.UpdateEDIMessageAcknowledgmentRequest,
) (*edi.EDIMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		if req == nil {
			return nil, errAckRequestRequired
		}

		m := edidb.EDIMessages
		return updateOne(ctx, r.db.Keel(ctx), req.ID, []keel.Assignment{
			m.AckStatus.To(req.AckStatus),
			m.AckMessageID.To(req.AckMessageID),
			setOptional(m.AckReceivedAt, req.AckReceivedAt),
			m.AckLastError.To(req.AckLastError),
			m.Version.ToExpr(m.Version),
		})
	})
}

func updateOne(
	ctx context.Context,
	db *keel.DB,
	id pulid.ID,
	sets []keel.Assignment,
) (*edi.EDIMessage, error) {
	m := edidb.EDIMessages
	rows, err := m.Where(m.ID.Eq(id)).UpdateReturning(ctx, db, sets...)
	if err != nil {
		return nil, messageErrors.Map(err, id.String())
	}
	if err = dberror.CheckAffected(int64(len(rows)), "EDIMessage", id.String()); err != nil {
		return nil, err
	}

	return rows[0], nil
}

func setOptional(col keel.Int64Col, value *int64) keel.Assignment {
	if value == nil {
		return col.ToNull()
	}

	return col.To(*value)
}

func (r *repository) GetOutboundMessageByAS2MessageID(
	ctx context.Context,
	as2MessageID string,
) (*edi.EDIMessage, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve the outbound message an inbound AS2 receipt names before its tenant is known",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		m := edidb.EDIMessages
		entity, err := m.AllTenants().
			Where(m.AS2MessageID.Eq(as2MessageID), m.Direction.Eq(edi.DocumentDirectionOutbound)).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, as2MessageID)
		}

		return entity, nil
	})
}

type deliveryStatusCount struct {
	Status edi.MessageDeliveryStatus
	Count  int64
}

func (r *repository) GetDeliveryStatusCounts(
	ctx context.Context,
	req ediports.GetEDIMessageStatusCountsRequest,
) (map[edi.MessageDeliveryStatus]int, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[edi.MessageDeliveryStatus]int, error) {
			m := edidb.EDIMessages
			rows, err := keel.Into[deliveryStatusCount](
				ctx,
				r.db.Keel(ctx),
				m.Where(
					m.DeliveryStatus.IsNotNull(),
					keel.When(req.Since > 0, m.GeneratedAt.Gte(req.Since)),
					keel.When(req.PartnerID.IsNotNil(), m.EDIPartnerID.Eq(req.PartnerID)),
				).
					Group(m.DeliveryStatus).
					Pick(m.DeliveryStatus, keel.Count()),
			)
			if err != nil {
				return nil, err
			}

			counts := make(map[edi.MessageDeliveryStatus]int, len(rows))
			for _, row := range rows {
				counts[row.Status] = int(row.Count)
			}

			return counts, nil
		},
	)
}

type ackStatusCount struct {
	Status edi.MessageAcknowledgmentStatus
	Count  int64
}

func ackStatusFilters(req *ediports.GetEDIMessageStatusCountsRequest) []keel.Pred {
	m := edidb.EDIMessages
	return []keel.Pred{
		m.AckStatus.IsNotNull(),
		keel.When(req.Since > 0, m.GeneratedAt.Gte(req.Since)),
		keel.When(req.PartnerID.IsNotNil(), m.EDIPartnerID.Eq(req.PartnerID)),
		keel.When(req.TransactionSet != "", m.TransactionSet.Eq(req.TransactionSet)),
	}
}

func (r *repository) GetAckStatusCounts(
	ctx context.Context,
	req ediports.GetEDIMessageStatusCountsRequest,
) (map[edi.MessageAcknowledgmentStatus]int, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[edi.MessageAcknowledgmentStatus]int, error) {
			m := edidb.EDIMessages
			rows, err := keel.Into[ackStatusCount](
				ctx,
				r.db.Keel(ctx),
				m.Where(ackStatusFilters(&req)...).
					Group(m.AckStatus).
					Pick(m.AckStatus, keel.Count()),
			)
			if err != nil {
				return nil, err
			}

			counts := make(map[edi.MessageAcknowledgmentStatus]int, len(rows))
			for _, row := range rows {
				counts[row.Status] = int(row.Count)
			}

			return counts, nil
		},
	)
}

func (r *repository) GetAckStatusCountsByPartner(
	ctx context.Context,
	req ediports.GetEDIMessageStatusCountsRequest,
) ([]*ediports.EDIPartnerAckStatusCount, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediports.EDIPartnerAckStatusCount, error) {
			m := edidb.EDIMessages
			partner := m.Partner()
			rows, err := keel.Into[ediports.EDIPartnerAckStatusCount](
				ctx,
				r.db.Keel(ctx),
				m.Where(ackStatusFilters(&req)...).
					Group(m.EDIPartnerID, m.AckStatus).
					Pick(m.EDIPartnerID, partner.Name.Max(), partner.Code.Max(), m.AckStatus, keel.Count()).
					Order(partner.Name.Max().Asc(), m.EDIPartnerID.Asc(), m.AckStatus.Asc()),
			)
			if err != nil {
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

func (r *repository) GetPartnerScorecards(
	ctx context.Context,
	req *ediports.GetEDIPartnerScorecardsRequest,
) ([]*ediports.EDIPartnerScorecardRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediports.EDIPartnerScorecardRow, error) {
			rows, err := keel.Into[ediports.EDIPartnerScorecardRow](
				ctx,
				r.db.Keel(ctx),
				scorecardQuery(req),
			)
			if err != nil {
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

func scorecardQuery(req *ediports.GetEDIPartnerScorecardsRequest) *keel.Query[*edi.EDIMessage] {
	m := edidb.EDIMessages
	partner := m.Partner()
	outbound := m.Direction.Eq(edi.DocumentDirectionOutbound)
	delivered := func(status edi.MessageDeliveryStatus) keel.Pred {
		return keel.And(outbound, m.DeliveryStatus.Eq(status))
	}
	pendingDelivery := keel.And(outbound, m.DeliveryStatus.In(pendingDeliveryStatuses...))
	acknowledged := keel.And(m.AckReceivedAt.IsNotNull(), m.DeliverySentAt.IsNotNull())
	latency := keel.DynOf(m.AckReceivedAt).Sub(m.DeliverySentAt)

	return m.Where(
		m.EDIPartnerID.IsNotNull(),
		keel.When(req.Since > 0, m.GeneratedAt.Gte(req.Since)),
		keel.When(req.PartnerID.IsNotNil(), m.EDIPartnerID.Eq(req.PartnerID)),
	).
		Group(m.EDIPartnerID).
		Pick(
			m.EDIPartnerID,
			partner.Name.Max(),
			partner.Code.Max(),
			keel.Count().Filter(outbound),
			keel.Count().Filter(delivered(edi.MessageDeliveryStatusSent)),
			keel.Count().Filter(delivered(edi.MessageDeliveryStatusFailed)),
			keel.Count().Filter(delivered(edi.MessageDeliveryStatusDeadLettered)),
			keel.Count().Filter(m.Direction.Eq(edi.DocumentDirectionInbound)),
			latency.Avg().Filter(acknowledged).Cast(keel.TypeFloat64),
			keel.CastFloat64(
				keel.Float64Of(keel.PercentileCont(0.95, latency.Asc()).Filter(acknowledged)),
			),
			keel.Count().Filter(keel.And(
				m.AckStatus.Eq(edi.MessageAcknowledgmentStatusPending),
				m.GeneratedAt.Lt(req.OverdueAckPendingSince),
			)),
			keel.Count().
				Filter(keel.And(pendingDelivery, m.GeneratedAt.Lt(req.PendingOver4hBefore))),
			keel.Count().
				Filter(keel.And(pendingDelivery, m.GeneratedAt.Lt(req.PendingOver24hBefore))),
			m.GeneratedAt.Min().Filter(pendingDelivery),
		).
		Order(partner.Name.Max().Asc())
}

func (r *repository) GetVolumeSeries(
	ctx context.Context,
	req ediports.GetEDIVolumeSeriesRequest,
) ([]*ediports.EDIVolumePoint, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ediports.EDIVolumePoint, error) {
		points, err := keel.Into[ediports.EDIVolumePoint](
			ctx,
			r.db.Keel(ctx),
			volumeQuery(&req),
		)
		if err != nil {
			return nil, err
		}

		return pointers(points), nil
	})
}

func volumeQuery(req *ediports.GetEDIVolumeSeriesRequest) *keel.Query[*edi.EDIMessage] {
	m := edidb.EDIMessages
	bucket := req.BucketSeconds
	if bucket <= 0 {
		bucket = defaultBucketSeconds
	}
	bucketStart := keel.DynOf(m.GeneratedAt).Div(bucket).Mul(bucket)
	outbound := m.Direction.Eq(edi.DocumentDirectionOutbound)

	return m.Where(
		keel.When(req.Since > 0, m.GeneratedAt.Gte(req.Since)),
		keel.When(req.PartnerID.IsNotNil(), m.EDIPartnerID.Eq(req.PartnerID)),
	).
		Group(bucketStart).
		Pick(
			bucketStart,
			keel.Count().Filter(outbound),
			keel.Count().
				Filter(keel.And(outbound, m.DeliveryStatus.Eq(edi.MessageDeliveryStatusSent))),
			keel.Count().Filter(keel.And(outbound, m.DeliveryStatus.In(
				edi.MessageDeliveryStatusFailed,
				edi.MessageDeliveryStatusDeadLettered,
			))),
			keel.Count().Filter(m.Direction.Eq(edi.DocumentDirectionInbound)),
		).
		Order(keel.Asc(bucketStart))
}

func (r *repository) GetPartnerSetActivity(
	ctx context.Context,
	req *ediports.GetEDIPartnerSetActivityRequest,
) ([]*ediports.EDIPartnerSetActivityRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*ediports.EDIPartnerSetActivityRow, error) {
			m := edidb.EDIMessages
			outbound := m.Direction.Eq(edi.DocumentDirectionOutbound)
			rows, err := keel.Into[ediports.EDIPartnerSetActivityRow](
				ctx,
				r.db.Keel(ctx),
				m.Where(
					m.EDIPartnerID.Eq(req.PartnerID),
					keel.When(req.Since > 0, m.GeneratedAt.Gte(req.Since)),
				).
					Group(m.TransactionSet, m.Direction).
					Pick(
						m.TransactionSet,
						m.Direction,
						keel.Count(),
						keel.Count().Filter(keel.And(
							outbound,
							m.DeliveryStatus.Eq(edi.MessageDeliveryStatusFailed),
						)),
						keel.Count().Filter(keel.And(
							outbound,
							m.DeliveryStatus.Eq(edi.MessageDeliveryStatusDeadLettered),
						)),
						m.GeneratedAt.Max(),
					).
					Order(m.Direction.Desc(), m.TransactionSet.Asc()),
			)
			if err != nil {
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

type overdueAckFilter struct {
	pendingSince   int64
	partnerID      pulid.ID
	transactionSet edi.TransactionSet
}

func (f *overdueAckFilter) where() []keel.Pred {
	m := edidb.EDIMessages
	return []keel.Pred{
		m.AckStatus.Eq(edi.MessageAcknowledgmentStatusPending),
		m.GeneratedAt.Lt(f.pendingSince),
		keel.When(f.partnerID.IsNotNil(), m.EDIPartnerID.Eq(f.partnerID)),
		keel.When(f.transactionSet != "", m.TransactionSet.Eq(f.transactionSet)),
	}
}

func (r *repository) GetOverdueAckCount(
	ctx context.Context,
	req ediports.GetEDIOverdueAckCountRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		filter := overdueAckFilter{
			pendingSince:   req.PendingSince,
			partnerID:      req.PartnerID,
			transactionSet: req.TransactionSet,
		}
		n, err := edidb.EDIMessages.Where(filter.where()...).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, err
		}

		return int(n), nil
	})
}

func (r *repository) ListOverdueAcks(
	ctx context.Context,
	req *ediports.ListOverdueEDIAcksRequest,
) (*ediports.OverdueEDIAcks, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ediports.OverdueEDIAcks, error) {
		db := r.db.Keel(ctx)
		m := edidb.EDIMessages
		partner := m.Partner()
		filter := overdueAckFilter{
			pendingSince:   req.PendingSince,
			partnerID:      req.PartnerID,
			transactionSet: req.TransactionSet,
		}

		groups, err := keel.Into[ediports.OverdueEDIAckPartner](
			ctx,
			db,
			m.Where(filter.where()...).
				Group(m.EDIPartnerID).
				Pick(
					m.EDIPartnerID,
					partner.Name.Max(),
					partner.Code.Max(),
					keel.Count(),
					m.GeneratedAt.Min(),
				).
				Order(m.GeneratedAt.Min().Asc(), m.EDIPartnerID.Asc()),
		)
		if err != nil {
			return nil, err
		}

		partners := pointers(groups)
		total := 0
		for _, group := range partners {
			total += group.Count
		}
		overdue := &ediports.OverdueEDIAcks{
			Messages: make([]*edi.EDIMessage, 0, min(max(req.Limit, 0), total)),
			Partners: partners,
		}
		if total == 0 || req.Limit <= 0 {
			return overdue, nil
		}

		messages, err := m.Omit(m.RawX12, m.PayloadSnapshot).
			Where(filter.where()...).
			Load(m.Partner()).
			Order(m.GeneratedAt.Asc(), m.ID.Asc()).
			Limit(req.Limit).
			List(ctx, db)
		if err != nil {
			return nil, err
		}
		overdue.Messages = append(overdue.Messages, messages...)

		return overdue, nil
	})
}

func (r *repository) ListRecentDeadLettered(
	ctx context.Context,
	req *ediports.ListRecentEDIMessageFailuresRequest,
) ([]*edi.EDIMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIMessage, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultRecentFailuresLimit
		}

		m := edidb.EDIMessages
		return m.Omit(m.RawX12, m.PayloadSnapshot).
			Where(
				m.DeliveryStatus.Eq(edi.MessageDeliveryStatusDeadLettered),
				keel.When(req.PartnerID.IsNotNil(), m.EDIPartnerID.Eq(req.PartnerID)),
				keel.When(req.TransactionSet != "", m.TransactionSet.Eq(req.TransactionSet)),
				keel.When(req.Since > 0, m.UpdatedAt.Gte(req.Since)),
			).
			Load(m.Partner()).
			Order(m.UpdatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) PurgeRawX12Before(
	ctx context.Context,
	req ediports.PurgeEDIRawPayloadsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		m := edidb.EDIMessages
		due := edidb.EDIMessages.As("due")
		batch := due.Pick(due.ID).
			Where(
				due.CreatedAt.Lt(req.Before),
				due.RawPurgedAt.IsNull(),
				keel.Or(
					due.Direction.Eq(edi.DocumentDirectionInbound),
					due.DeliveryStatus.Eq(edi.MessageDeliveryStatusSent),
				),
			).
			Limit(req.Limit)

		return m.Where(m.ID.InQuery(batch)).Update(
			ctx,
			r.db.Keel(ctx),
			m.RawX12.To(""),
			m.PayloadSnapshot.To(map[string]any{}),
			m.RawPurgedAt.To(req.PurgedAt),
			m.Version.ToExpr(m.Version),
			m.UpdatedAt.ToExpr(m.UpdatedAt),
		)
	})
}

func (r *repository) CountDeadLetteredSince(ctx context.Context, since int64) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"count dead-lettered EDI messages instance-wide for the health check",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		m := edidb.EDIMessages
		return m.AllTenants().
			Where(
				m.DeliveryStatus.Eq(edi.MessageDeliveryStatusDeadLettered),
				m.UpdatedAt.Gte(since),
			).
			Count(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetOutboundMessageForAck(
	ctx context.Context,
	req ediports.GetEDIOutboundMessageForAckRequest,
) (*edi.EDIMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIMessage, error) {
		m := edidb.EDIMessages
		entity, err := m.Where(
			m.EDIPartnerID.Eq(req.PartnerID),
			m.Direction.Eq(edi.DocumentDirectionOutbound),
			m.GroupControlNumber.LTrimChars("0").
				EqExpr(keel.Text(req.GroupControlNumber).LTrimChars("0")),
			m.TransactionControlNumber.LTrimChars("0").
				EqExpr(keel.Text(req.TransactionControlNumber).LTrimChars("0")),
			keel.When(req.TransactionSet != "", m.TransactionSet.Eq(req.TransactionSet)),
		).
			Order(m.GeneratedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.TransactionControlNumber)
		}

		return entity, nil
	})
}

func pointers[T any](rows []T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out
}
