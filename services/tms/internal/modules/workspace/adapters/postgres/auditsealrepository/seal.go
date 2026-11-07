package auditsealrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const digestSQL = `
SELECT r.retention_class,
	count(*)::bigint,
	encode(sha256(convert_to(string_agg(r.row_hash, '' ORDER BY r."timestamp", r.id), 'UTF8')), 'hex')
FROM (
	SELECT e.retention_class, e."timestamp", e.id,
		encode(sha256(convert_to(jsonb_build_array(
			e.id, e.organization_id, e.business_unit_id, e."timestamp", e.resource, e.resource_id,
			e.operation, e.action, e.category, e.retention_class, e.channel, e.principal_type,
			e.principal_id, e.user_id, e.api_key_id, e.actor_name, e.ai_run_id, e.entity_ref,
			e.correlation_id, e.trace_id, e.ip_address, e.user_agent, e.comment, e.changes,
			e.previous_state, e.current_state, e.metadata, e.sensitive_data, e.critical
		)::text, 'UTF8')), 'hex') AS row_hash
	FROM audit_entries e
	WHERE e.organization_id = $1
		AND e.business_unit_id = $2
		AND e."timestamp" >= $3
		AND e."timestamp" < $4
) r
GROUP BY r.retention_class`

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) auditports.SealRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.audit-seal-repository")}
}

func tenantContext(ctx context.Context, tenant auditports.TenantKey) context.Context {
	return dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
	})
}

func (r *repository) GetHead(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*audit.ChainHead, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*audit.ChainHead, error) {
		head, err := auditdb.ChainHeads.Query().One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return &audit.ChainHead{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read audit chain head: %w", err)
		}
		return head, nil
	})
}

func (r *repository) ListHeads(ctx context.Context) ([]*audit.ChainHead, error) {
	ctx = dbscope.WithSystem(ctx, "list every organization's audit chain head to verify")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.ChainHead, error) {
		h := auditdb.ChainHeads
		return h.AllTenants().
			Where(h.LastSeq.Gt(0)).
			Order(h.OrganizationID.Asc(), h.BusinessUnitID.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Digest(
	ctx context.Context,
	tenant auditports.TenantKey,
	from, to int64,
) (*auditports.SealComputation, error) {
	ctx = tenantContext(ctx, tenant)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*auditports.SealComputation, error) {
		out := &auditports.SealComputation{
			TenantKey:    tenant,
			FromTS:       from,
			ToTS:         to,
			ClassDigests: make(map[string]audit.ClassDigest, len(audit.RetentionClasses)),
		}
		err := keel.QueryRows(ctx, r.db.Keel(ctx), func(row pgx.CollectableRow) error {
			var class, digest string
			var count int64
			if err := row.Scan(&class, &count, &digest); err != nil {
				return err
			}
			out.ClassDigests[class] = audit.ClassDigest{Count: count, Digest: digest}
			out.RowCount += count
			return nil
		}, digestSQL, tenant.OrganizationID.String(), tenant.BusinessUnitID.String(), from, to)
		if err != nil {
			return nil, fmt.Errorf("digest audit entries: %w", err)
		}
		return out, nil
	})
}

func (r *repository) AppendSeal(
	ctx context.Context,
	tenant auditports.TenantKey,
	seal func(head *audit.ChainHead) (*audit.Seal, error),
) (*audit.Seal, error) {
	ctx = tenantContext(ctx, tenant)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*audit.Seal, error) {
		h := auditdb.ChainHeads
		if _, err := h.Upsert(ctx, r.db.Keel(ctx), []*audit.ChainHead{{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}}, keel.OnConflict(h.OrganizationID, h.BusinessUnitID).DoNothing()); err != nil {
			return nil, fmt.Errorf("open audit chain head: %w", err)
		}

		head, err := h.Query().Lock(keel.ForUpdate).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("lock audit chain head: %w", err)
		}

		next, err := seal(head)
		if err != nil || next == nil {
			return nil, err
		}

		if err = auditdb.Seals.Create(ctx, r.db.Keel(ctx), next); err != nil {
			return nil, fmt.Errorf("write audit seal: %w", err)
		}

		head.LastSeq = next.Seq
		head.LastHash = next.Hash
		head.SealedThrough = next.ToTS
		head.HashKeyID = next.HashKeyID
		if err = h.Save(ctx, r.db.Keel(ctx), head); err != nil {
			return nil, fmt.Errorf("advance audit chain head: %w", err)
		}

		return next, nil
	})
}

func (r *repository) ListSeals(
	ctx context.Context,
	_ pagination.TenantInfo,
	afterSeq int64,
	limit int,
) ([]*audit.Seal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.Seal, error) {
		s := auditdb.Seals
		return s.Where(s.Seq.Gt(afterSeq)).Order(s.Seq.Asc()).Limit(limit).List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) RecordVerification(ctx context.Context, head *audit.ChainHead) error {
	ctx = tenantContext(ctx, auditports.TenantKey{
		OrganizationID: head.OrganizationID,
		BusinessUnitID: head.BusinessUnitID,
	})
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		h := auditdb.ChainHeads
		sets := []keel.Assignment{
			h.LastVerifiedSeq.To(head.LastVerifiedSeq),
			h.LastVerificationStatus.To(head.LastVerificationStatus),
			h.LastVerificationDetail.To(head.LastVerificationDetail),
			h.VerifyCursorSeq.To(head.VerifyCursorSeq),
		}
		if head.LastVerifiedAt != nil {
			sets = append(sets, h.LastVerifiedAt.To(*head.LastVerifiedAt))
		}
		if head.LastVerificationFailedSeq != nil {
			sets = append(sets, h.LastVerificationFailedSeq.To(*head.LastVerificationFailedSeq))
		} else {
			sets = append(sets, h.LastVerificationFailedSeq.ToNull())
		}
		_, err := h.Where(keel.True()).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return fmt.Errorf("record audit chain verification: %w", err)
		}
		return nil
	})
}

func (r *repository) DeleteSealsBefore(ctx context.Context, before int64) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "delete audit seals older than every entry they could cover")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		s := auditdb.Seals
		return s.AllTenants().Where(s.ToTS.Lt(before)).Delete(ctx, r.db.Keel(ctx))
	})
}
