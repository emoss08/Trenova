package aiauditrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit/aiauditdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrChainRace is an append that inserted fewer rows than it chained. Under
// the chain head's lock that cannot happen unless something else wrote the
// tenant's trail, so the whole batch is rolled back rather than leave a gap
// in the chain.
var ErrChainRace = errors.New("the AI audit chain moved while a batch was being appended")

var eventErrors = dberror.Mapping{Entity: "AI audit event"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) aiauditports.AIAuditRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiaudit-repository"),
	}
}

// Append chains a tenant's new rows under the chain head's row lock: rows the
// tenant already holds are dropped, the rest take the next seqs in the order
// given and are signed one after another, and the batch is sealed.
func (r *repository) Append(
	ctx context.Context,
	req *aiauditports.AppendAIAuditEventsRequest,
) (*aiauditports.AppendAIAuditEventsResult, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"append projected events to each organization's AI audit chain in one pass",
	)
	result := &aiauditports.AppendAIAuditEventsResult{}
	if len(req.Events) == 0 {
		return result, nil
	}
	if req.Sign == nil {
		return nil, errors.New("an AI audit append needs a signer")
	}

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*aiauditports.AppendAIAuditEventsResult, error) {
			if err := r.appendChain(ctx, req, result); err != nil {
				r.l.Error("failed to append AI audit events",
					zap.String("organizationId", req.TenantInfo.OrgID.String()),
					zap.Int("events", len(req.Events)),
					zap.Error(err),
				)

				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) appendChain(
	ctx context.Context,
	req *aiauditports.AppendAIAuditEventsRequest,
	result *aiauditports.AppendAIAuditEventsResult,
) error {
	db := r.db.Keel(ctx)
	head, err := lockHead(ctx, db, req.TenantInfo, req.Now)
	if err != nil {
		return err
	}

	fresh, err := dropExisting(ctx, db, req.TenantInfo, req.Events)
	if err != nil || len(fresh) == 0 {
		return err
	}

	prev := head.LastHash
	if head.LastSeq == 0 || prev == "" {
		prev = aiaudit.GenesisHash
	}
	seq := head.LastSeq
	for _, event := range fresh {
		seq++
		event.Seq = seq
		event.OrganizationID = req.TenantInfo.OrgID
		event.BusinessUnitID = req.TenantInfo.BuID
		if event.ID.IsNil() {
			event.ID = pulid.MustNew(aiaudit.EventIDPrefix)
		}
		if event.RecordedAt == 0 {
			event.RecordedAt = req.Now
		}
		if err = req.Sign(event, prev); err != nil {
			return fmt.Errorf("sign AI audit event %s: %w", event.SourceKey, err)
		}
		prev = event.Hash
	}

	inserted, err := aiauditdb.AIAuditEvents.CreateOrSkip(ctx, db, fresh)
	if err != nil {
		return fmt.Errorf("insert AI audit events: %w", err)
	}
	if int(inserted) != len(fresh) {
		return ErrChainRace
	}

	fromSeq := head.LastSeq + 1
	if err = advanceHead(ctx, db, head, seq, prev, req.KeyID); err != nil {
		return err
	}

	if err = aiauditdb.AIAuditSeals.Create(ctx, db, &aiaudit.AIAuditSeal{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		FromSeq:        fromSeq,
		ToSeq:          seq,
		HeadHash:       prev,
		HashKeyID:      req.KeyID,
		RowCount:       len(fresh),
		SealedAt:       req.Now,
	}); err != nil {
		return fmt.Errorf("seal AI audit batch: %w", err)
	}

	result.Inserted = len(fresh)
	result.FromSeq = fromSeq
	result.ToSeq = seq
	result.HeadHash = prev

	return nil
}

func lockHead(
	ctx context.Context,
	db *keel.DB,
	tenantInfo pagination.TenantInfo,
	now int64,
) (*aiaudit.AIAuditChainHead, error) {
	h := aiauditdb.AIAuditChainHeads
	seed := &aiaudit.AIAuditChainHead{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if _, err := h.CreateOrSkip(ctx, db, []*aiaudit.AIAuditChainHead{seed}); err != nil {
		return nil, fmt.Errorf("open AI audit chain head: %w", err)
	}

	head, err := h.AllTenants().
		Where(
			h.OrganizationID.Eq(tenantInfo.OrgID),
			h.BusinessUnitID.Eq(tenantInfo.BuID),
		).
		Lock(keel.ForUpdate).
		One(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("lock AI audit chain head: %w", err)
	}

	return head, nil
}

func advanceHead(
	ctx context.Context,
	db *keel.DB,
	head *aiaudit.AIAuditChainHead,
	lastSeq int64,
	lastHash, keyID string,
) error {
	h := aiauditdb.AIAuditChainHeads
	sets := []keel.Assignment{
		h.LastSeq.To(lastSeq),
		h.LastHash.To(lastHash),
		h.UpdatedAt.To(head.UpdatedAt),
	}
	if keyID != "" {
		sets = append(sets, h.HashKeyID.To(keyID))
	}

	affected, err := h.AllTenants().
		Where(
			h.OrganizationID.Eq(head.OrganizationID),
			h.BusinessUnitID.Eq(head.BusinessUnitID),
			h.LastSeq.Eq(head.LastSeq),
		).
		Update(ctx, db, sets...)
	if err != nil {
		return fmt.Errorf("advance AI audit chain head: %w", err)
	}
	if affected != 1 {
		return ErrChainRace
	}

	return nil
}

func dropExisting(
	ctx context.Context,
	db *keel.DB,
	tenantInfo pagination.TenantInfo,
	events []*aiaudit.AIAuditEvent,
) ([]*aiaudit.AIAuditEvent, error) {
	keys := make([]string, 0, len(events))
	for _, event := range events {
		keys = append(keys, event.SourceKey)
	}

	existing, err := existingSourceKeys(ctx, db, tenantInfo, keys)
	if err != nil {
		return nil, err
	}

	fresh := make([]*aiaudit.AIAuditEvent, 0, len(events))
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		if _, held := existing[event.SourceKey]; held {
			continue
		}
		if _, dup := seen[event.SourceKey]; dup {
			continue
		}
		seen[event.SourceKey] = struct{}{}
		fresh = append(fresh, event)
	}

	return fresh, nil
}

func (r *repository) ExistingSourceKeys(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	keys []string,
) (map[string]struct{}, error) {
	ctx = dbscope.WithSystem(ctx, "deduplicate projected AI audit events across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (map[string]struct{}, error) {
		return existingSourceKeys(ctx, r.db.Keel(ctx), tenantInfo, keys)
	})
}

func existingSourceKeys(
	ctx context.Context,
	db *keel.DB,
	tenantInfo pagination.TenantInfo,
	keys []string,
) (map[string]struct{}, error) {
	found := make(map[string]struct{}, len(keys))
	if len(keys) == 0 {
		return found, nil
	}

	e := aiauditdb.AIAuditEvents
	held, err := keel.Into[string](ctx, db, e.AllTenants().
		Where(
			e.OrganizationID.Eq(tenantInfo.OrgID),
			e.BusinessUnitID.Eq(tenantInfo.BuID),
			e.SourceKey.In(keys...),
		).
		Pick(e.SourceKey))
	if err != nil {
		return nil, fmt.Errorf("read existing AI audit source keys: %w", err)
	}

	for _, key := range held {
		found[key] = struct{}{}
	}

	return found, nil
}

// ListConnection pages the trail newest first by default, keyed on
// (occurred_at, id).
func (r *repository) ListConnection(
	ctx context.Context,
	req *aiauditports.ListAIAuditEventsQuery,
) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error) {
			if len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{{
					Field:     "occurredAt",
					Direction: dbtype.SortDirectionDesc,
				}}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				aiauditdb.AIAuditEvents.Query(),
				&listquery.Request[aiaudit.AIAuditEvent]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list AI audit events", zap.Error(err))

				return nil, fmt.Errorf("list AI audit events: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req aiauditports.GetAIAuditEventRequest,
) (*aiaudit.AIAuditEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditEvent, error) {
		entity, err := aiauditdb.AIAuditEvents.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, eventErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*aiaudit.AIAuditEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiaudit.AIAuditEvent, error) {
		if len(ids) == 0 {
			return []*aiaudit.AIAuditEvent{}, nil
		}

		e := aiauditdb.AIAuditEvents
		events, err := e.Where(e.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list AI audit events by id: %w", err)
		}

		return events, nil
	})
}

func scoped(scope *aiauditports.AIAuditEventScope) *keel.Query[*aiaudit.AIAuditEvent] {
	e := aiauditdb.AIAuditEvents
	filter := scope.Filter
	if filter == nil {
		filter = &aiaudit.ExportFilter{}
	}

	q := listquery.Filter(e.Query(), &pagination.QueryOptions{
		TenantInfo:   scope.TenantInfo,
		Query:        filter.Query,
		FieldFilters: filter.FieldFilters,
		FilterGroups: filter.FilterGroups,
	})
	q.Where(e.OccurredAt.Between(scope.From, scope.To))
	if scope.SnapshotSeq > 0 {
		q.Where(e.Seq.Lte(scope.SnapshotSeq))
	}

	return q
}

func (r *repository) Summarize(
	ctx context.Context,
	scope *aiauditports.AIAuditEventScope,
) (*aiauditports.AIAuditEventSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*aiauditports.AIAuditEventSummary, error) {
			e := aiauditdb.AIAuditEvents
			summary, err := keel.IntoOne[aiauditports.AIAuditEventSummary](
				ctx,
				r.db.Keel(ctx),
				scoped(scope).Pick(
					keel.Count(),
					keel.DynOf(e.Seq).Min().Coalesce(0),
					keel.DynOf(e.Seq).Max().Coalesce(0),
				),
			)
			if err != nil && !errors.Is(err, keel.ErrNotFound) {
				return nil, fmt.Errorf("summarize AI audit events: %w", err)
			}

			return &summary, nil
		},
	)
}

func (r *repository) ListPage(
	ctx context.Context,
	req *aiauditports.ListAIAuditEventPageRequest,
) ([]*aiaudit.AIAuditEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiaudit.AIAuditEvent, error) {
		e := aiauditdb.AIAuditEvents
		q := scoped(&req.Scope)
		if req.HasAfterCursor {
			q.Where(keel.DynRow(e.OccurredAt, e.ID).Gt(keel.DynRow(req.AfterOccurred, req.AfterID)))
		}

		events, err := q.
			Order(e.OccurredAt.Asc(), e.ID.Asc()).
			Limit(req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list AI audit event page: %w", err)
		}

		return events, nil
	})
}

func (r *repository) ListChainRange(
	ctx context.Context,
	req aiauditports.ListAIAuditChainRangeRequest,
) ([]*aiaudit.AIAuditEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiaudit.AIAuditEvent, error) {
		e := aiauditdb.AIAuditEvents
		events, err := e.Where(e.Seq.Gt(req.AfterSeq)).
			Order(e.Seq.Asc()).
			Limit(req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list AI audit chain range: %w", err)
		}

		return events, nil
	})
}

func (r *repository) FirstSeq(
	ctx context.Context,
	_ pagination.TenantInfo,
) (int64, bool, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (int64, bool, error) {
		e := aiauditdb.AIAuditEvents
		first, err := keel.IntoOne[*int64](ctx, r.db.Keel(ctx), e.Pick(e.Seq.Min()))
		if err != nil {
			return 0, false, fmt.Errorf("read the first AI audit seq: %w", err)
		}
		if first == nil {
			return 0, false, nil
		}

		return *first, true, nil
	})
}
