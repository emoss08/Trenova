package auditsealrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/auditdigest"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const importedThroughSQL = `SELECT COALESCE(max(seq), 0)::bigint FROM audit_seals
WHERE organization_id = $1 AND business_unit_id = $2 AND hash_version IN (3, 4)`

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
		digest, err := auditdigest.Compute(ctx, r.db.Keel(ctx), auditdigest.Window{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
			From:           from,
			To:             to,
		})
		if err != nil {
			return nil, err
		}
		return &auditports.SealComputation{
			TenantKey:    tenant,
			FromTS:       from,
			ToTS:         to,
			RowCount:     digest.RowCount,
			ClassDigests: digest.Classes,
		}, nil
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

func (r *repository) ImportedThrough(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (int64, error) {
	ctx = tenantContext(ctx, auditports.TenantKey{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	})
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		seq, err := keel.QueryValue[int64](
			ctx,
			r.db.Keel(ctx),
			importedThroughSQL,
			tenant.OrgID.String(),
			tenant.BuID.String(),
		)
		if err != nil {
			return 0, fmt.Errorf("read the imported audit seals: %w", err)
		}
		return seq, nil
	})
}

func (r *repository) DeleteSealsBefore(ctx context.Context, before int64) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "delete audit seals older than every entry they could cover")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		s := auditdb.Seals
		return s.AllTenants().Where(s.ToTS.Lt(before)).Delete(ctx, r.db.Keel(ctx))
	})
}
