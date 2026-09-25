package aiauditrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	pruneSetting      = "trenova.ai_audit_prune"
	defaultPruneBatch = 5000
)

// GetChainHead reads a tenant's chain head, or an empty one for a tenant
// whose trail has not started.
func (r *repository) GetChainHead(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*aiaudit.AIAuditChainHead, error) {
	head := new(aiaudit.AIAuditChainHead)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(head).
		Apply(buncolgen.AIAuditChainHeadApplyTenant(tenantInfo)).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &aiaudit.AIAuditChainHead{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read AI audit chain head: %w", err)
	}

	return head, nil
}

func (r *repository) ListChainHeads(ctx context.Context) ([]*aiaudit.AIAuditChainHead, error) {
	cols := buncolgen.AIAuditChainHeadColumns
	heads := make([]*aiaudit.AIAuditChainHead, 0, 16)

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&heads).
		Where(cols.LastSeq.Gt(), 0).
		Order(cols.OrganizationID.OrderAsc(), cols.BusinessUnitID.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list AI audit chain heads: %w", err)
	}

	return heads, nil
}

// RecordVerification writes what a check of the chain found. It touches only
// the verification columns, so it never races the projector's advance of the
// head it shares a row with.
func (r *repository) RecordVerification(
	ctx context.Context,
	req *repositories.RecordAIAuditVerificationRequest,
) (*aiaudit.AIAuditChainHead, error) {
	cols := buncolgen.AIAuditChainHeadColumns
	head := new(aiaudit.AIAuditChainHead)

	q := r.db.DBForContext(ctx).NewUpdate().
		Model(head).
		Set(cols.LastVerificationStatus.Set(), req.Status).
		Set(cols.LastVerifiedAt.Set(), req.At).
		Set(cols.LastVerificationDetail.Set(), nullableText(req.Detail)).
		Set(cols.UpdatedAt.Set(), req.At).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AIAuditChainHeadScopeTenantUpdate(uq, req.TenantInfo)
		}).
		Returning("*")
	if req.Status == aiaudit.VerificationVerified {
		q = q.Set(cols.LastVerifiedSeq.Set(), req.VerifiedSeq).
			Set(cols.LastVerificationFailedSeq.SetNull())
	} else {
		q = q.Set(cols.LastVerificationFailedSeq.Set(), req.FailedSeq)
	}

	if _, err := q.Exec(ctx); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("record AI audit verification: %w", err)
	}
	if head.OrganizationID.IsNil() {
		return r.GetChainHead(ctx, req.TenantInfo)
	}

	return head, nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func (r *repository) GetSealEndingAt(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	toSeq int64,
) (*aiaudit.AIAuditSeal, error) {
	cols := buncolgen.AIAuditSealColumns
	seal := new(aiaudit.AIAuditSeal)

	err := r.db.DBForContext(ctx).NewSelect().
		Model(seal).
		Apply(buncolgen.AIAuditSealApplyTenant(tenantInfo)).
		Where(cols.ToSeq.Eq(), toSeq).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no seal ends there
	}
	if err != nil {
		return nil, fmt.Errorf("read AI audit seal: %w", err)
	}

	return seal, nil
}

func (r *repository) ListSeals(
	ctx context.Context,
	req repositories.ListAIAuditSealsRequest,
) ([]*aiaudit.AIAuditSeal, error) {
	cols := buncolgen.AIAuditSealColumns
	seals := make([]*aiaudit.AIAuditSeal, 0, req.Limit)

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&seals).
		Apply(buncolgen.AIAuditSealApplyTenant(req.TenantInfo)).
		Where(cols.ToSeq.Gte(), req.FromSeq).
		Order(cols.ToSeq.OrderAsc()).
		Limit(req.Limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list AI audit seals: %w", err)
	}

	return seals, nil
}

// LastSealBefore is the newest seal written before a time, which is where a
// retention sweep may cut the chain: pruning at a seal's end leaves the next
// row's prev_hash checkable against the seal.
func (r *repository) LastSealBefore(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sealedBefore int64,
) (*aiaudit.AIAuditSeal, error) {
	cols := buncolgen.AIAuditSealColumns
	seal := new(aiaudit.AIAuditSeal)

	err := r.db.DBForContext(ctx).NewSelect().
		Model(seal).
		Apply(buncolgen.AIAuditSealApplyTenant(tenantInfo)).
		Where(cols.SealedAt.Lt(), sealedBefore).
		Order(cols.ToSeq.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // nothing is old enough
	}
	if err != nil {
		return nil, fmt.Errorf("read the last AI audit seal before a time: %w", err)
	}

	return seal, nil
}

// Prune removes a tenant's chain up to a seq, oldest first, a batch per
// transaction, each under the setting the append-only trigger admits.
func (r *repository) Prune(
	ctx context.Context,
	req repositories.PruneAIAuditEventsRequest,
) (int, error) {
	batch := req.BatchSize
	if batch <= 0 {
		batch = defaultPruneBatch
	}
	cols := buncolgen.AIAuditEventColumns

	total := 0
	for {
		deleted := 0
		err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if isPostgres(tx) {
				if _, err := tx.NewSelect().
					ColumnExpr("set_config(?, 'on', true)", pruneSetting).
					Exec(txCtx); err != nil {
					return fmt.Errorf("admit the AI audit prune: %w", err)
				}
			}

			doomed := tx.NewSelect().
				Model((*aiaudit.AIAuditEvent)(nil)).
				Column(cols.ID.String()).
				Apply(buncolgen.AIAuditEventApplyTenant(req.TenantInfo)).
				Where(cols.Seq.Lte(), req.ThroughSeq).
				Order(cols.Seq.OrderAsc()).
				Limit(batch)

			res, err := tx.NewDelete().
				Model((*aiaudit.AIAuditEvent)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return buncolgen.AIAuditEventScopeTenantDelete(dq, req.TenantInfo).
						Where(cols.ID.In(), doomed)
				}).
				Exec(txCtx)
			if err != nil {
				return fmt.Errorf("prune AI audit events: %w", err)
			}
			affected, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("prune AI audit events: %w", err)
			}
			deleted = int(affected)

			return nil
		})
		if err != nil {
			r.l.Error("failed to prune AI audit events",
				zap.String("organizationId", req.TenantInfo.OrgID.String()),
				zap.Error(err),
			)

			return total, err
		}

		total += deleted
		if deleted < batch {
			return total, nil
		}
		if err = ctx.Err(); err != nil {
			return total, err
		}
	}
}

func (r *repository) GetWatermarks(
	ctx context.Context,
) (map[aiaudit.Source]*aiaudit.AIAuditProjectorState, error) {
	states := make([]*aiaudit.AIAuditProjectorState, 0, len(aiaudit.AllSources()))
	if err := r.db.DBForContext(ctx).NewSelect().Model(&states).Scan(ctx); err != nil {
		return nil, fmt.Errorf("read AI audit projector watermarks: %w", err)
	}

	byName := make(map[aiaudit.Source]*aiaudit.AIAuditProjectorState, len(states))
	for _, state := range states {
		byName[state.Source] = state
	}

	return byName, nil
}

func (r *repository) SaveWatermark(
	ctx context.Context,
	state *aiaudit.AIAuditProjectorState,
) error {
	cols := buncolgen.AIAuditProjectorStateColumns

	if _, err := r.db.DBForContext(ctx).NewInsert().
		Model(state).
		On("CONFLICT (" + cols.Source.Bare() + ") DO UPDATE").
		Set(cols.WatermarkTS.SetExcluded()).
		Set(cols.WatermarkID.SetExcluded()).
		Set(cols.UpdatedAt.SetExcluded()).
		Exec(ctx); err != nil {
		return fmt.Errorf("save AI audit projector watermark: %w", err)
	}

	return nil
}
