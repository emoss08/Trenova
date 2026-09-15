package latechargerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/latecharge"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     ports.DBConnection
	Logger *zap.Logger
}

type repository struct {
	db ports.DBConnection
	l  *zap.Logger
}

func New(p Params) repositories.LateChargeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("repository.late-charge"),
	}
}

type candidateRecord struct {
	CustomerID       string          `bun:"customer_id"`
	CustomerName     string          `bun:"customer_name"`
	InvoiceID        string          `bun:"invoice_id"`
	InvoiceNumber    string          `bun:"invoice_number"`
	CurrencyCode     string          `bun:"currency_code"`
	DueDate          int64           `bun:"due_date"`
	GracePeriodDays  int             `bun:"grace_period_days"`
	RatePercent      decimal.Decimal `bun:"rate_percent"`
	OpenBalanceMinor int64           `bun:"open_balance_minor"`
	AssessedPeriods  []int64         `bun:"assessed_periods,array"`
}

const listCandidatesSQL = `
	SELECT
		cus.id AS customer_id,
		cus.name AS customer_name,
		inv.id AS invoice_id,
		inv.number AS invoice_number,
		inv.currency_code,
		inv.due_date,
		cbp.grace_period_days,
		cbp.late_charge_rate AS rate_percent,
		(inv.total_amount_minor - inv.applied_amount_minor) AS open_balance_minor,
		COALESCE((
			SELECT array_agg(lca.period_index ORDER BY lca.period_index)
			FROM late_charge_assessments lca
			WHERE lca.source_invoice_id = inv.id
			  AND lca.organization_id = inv.organization_id
			  AND lca.business_unit_id = inv.business_unit_id
		), '{}'::INT[]) AS assessed_periods
	FROM invoices inv
	JOIN customers cus
	  ON cus.id = inv.customer_id
	 AND cus.organization_id = inv.organization_id
	 AND cus.business_unit_id = inv.business_unit_id
	JOIN customer_billing_profiles cbp
	  ON cbp.customer_id = cus.id
	 AND cbp.organization_id = cus.organization_id
	 AND cbp.business_unit_id = cus.business_unit_id
	WHERE inv.organization_id = ?
	  AND inv.business_unit_id = ?
	  AND inv.status = 'Posted'
	  AND inv.bill_type IN ('Invoice', 'DebitMemo')
	  AND inv.total_amount_minor > inv.applied_amount_minor
	  AND inv.due_date IS NOT NULL
	  AND inv.due_date + (cbp.grace_period_days::BIGINT * 86400) <= ?
	  AND cbp.apply_late_charges = TRUE
	  AND cbp.late_charge_rate IS NOT NULL
	  AND cbp.late_charge_rate > 0
	  AND inv.dispute_status <> 'Disputed'
	  AND COALESCE(inv.memo_kind, '') <> 'LateCharge'
	  AND NOT EXISTS (
		SELECT 1
		FROM invoice_disputes idsp
		WHERE idsp.invoice_id = inv.id
		  AND idsp.organization_id = inv.organization_id
		  AND idsp.business_unit_id = inv.business_unit_id
		  AND idsp.status = 'Open'
	  )`

func (r *repository) ListCandidates(
	ctx context.Context,
	req *repositories.ListLateChargeCandidatesRequest,
) ([]*repositories.LateChargeCandidate, error) {
	query := listCandidatesSQL
	args := []any{req.TenantInfo.OrgID, req.TenantInfo.BuID, req.AsOfDate}
	if len(req.CustomerIDs) > 0 {
		query += "\n\t  AND inv.customer_id IN (?)"
		args = append(args, bun.In(req.CustomerIDs))
	}
	query += "\n\tORDER BY cus.name ASC, inv.due_date ASC, inv.number ASC"

	records := make([]*candidateRecord, 0)
	if err := r.db.DBForContext(ctx).NewRaw(query, args...).Scan(ctx, &records); err != nil {
		return nil, fmt.Errorf("list late charge candidates: %w", err)
	}

	candidates := make([]*repositories.LateChargeCandidate, 0, len(records))
	for _, rec := range records {
		assessed := make([]int, 0, len(rec.AssessedPeriods))
		for _, idx := range rec.AssessedPeriods {
			assessed = append(assessed, int(idx))
		}
		candidates = append(candidates, &repositories.LateChargeCandidate{
			CustomerID:       pulid.ID(rec.CustomerID),
			CustomerName:     rec.CustomerName,
			InvoiceID:        pulid.ID(rec.InvoiceID),
			InvoiceNumber:    rec.InvoiceNumber,
			CurrencyCode:     rec.CurrencyCode,
			DueDate:          rec.DueDate,
			GracePeriodDays:  rec.GracePeriodDays,
			RatePercent:      rec.RatePercent,
			OpenBalanceMinor: rec.OpenBalanceMinor,
			AssessedPeriods:  assessed,
		})
	}

	return candidates, nil
}

func (r *repository) InsertAssessments(
	ctx context.Context,
	assessments []*latecharge.LateChargeAssessment,
) ([]*latecharge.LateChargeAssessment, error) {
	if len(assessments) == 0 {
		return []*latecharge.LateChargeAssessment{}, nil
	}
	runKey := assessments[0].RunKey
	tenantInfo := pagination.TenantInfo{
		OrgID: assessments[0].OrganizationID,
		BuID:  assessments[0].BusinessUnitID,
	}

	if _, err := r.db.DBForContext(ctx).NewInsert().
		Model(&assessments).
		On("CONFLICT (organization_id, business_unit_id, source_invoice_id, period_index) DO NOTHING").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert late charge assessments: %w", err)
	}

	return r.listByRunKey(ctx, tenantInfo, runKey)
}

func (r *repository) listByRunKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runKey string,
) ([]*latecharge.LateChargeAssessment, error) {
	cols := buncolgen.LateChargeAssessmentColumns
	rows := make([]*latecharge.LateChargeAssessment, 0)
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.LateChargeAssessmentScopeTenant(sq, tenantInfo).
				Where(cols.RunKey.Eq(), runKey)
		}).
		Order(cols.SourceInvoiceID.OrderAsc(), cols.PeriodIndex.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list late charge assessments by run: %w", err)
	}

	return rows, nil
}

func (r *repository) SetDebitMemoLines(
	ctx context.Context,
	assessments []*latecharge.LateChargeAssessment,
) error {
	cols := buncolgen.LateChargeAssessmentColumns
	for _, assessment := range assessments {
		if assessment == nil || assessment.DebitMemoLineID.IsNil() {
			continue
		}
		if _, err := r.db.DBForContext(ctx).NewUpdate().
			Model(assessment).
			Column(cols.DebitMemoInvoiceID.Bare(), cols.DebitMemoLineID.Bare(), cols.UpdatedAt.Bare()).
			WherePK().
			Exec(ctx); err != nil {
			return fmt.Errorf("stamp late charge assessment memo line: %w", err)
		}
	}

	return nil
}

func (r *repository) DeleteByRunKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runKey string,
) (int64, error) {
	cols := buncolgen.LateChargeAssessmentColumns
	res, err := r.db.DBForContext(ctx).NewDelete().
		Model((*latecharge.LateChargeAssessment)(nil)).
		Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(cols.RunKey.Eq(), runKey).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete late charge assessments by run: %w", err)
	}

	return res.RowsAffected()
}

func (r *repository) ListBySourceInvoiceIDs(
	ctx context.Context,
	req *repositories.ListLateChargeAssessmentsByInvoiceIDsRequest,
) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
	cols := buncolgen.LateChargeAssessmentColumns
	return r.listGrouped(ctx, req, cols.SourceInvoiceID, func(a *latecharge.LateChargeAssessment) pulid.ID {
		return a.SourceInvoiceID
	})
}

func (r *repository) ListByDebitMemoIDs(
	ctx context.Context,
	req *repositories.ListLateChargeAssessmentsByInvoiceIDsRequest,
) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
	cols := buncolgen.LateChargeAssessmentColumns
	return r.listGrouped(ctx, req, cols.DebitMemoInvoiceID, func(a *latecharge.LateChargeAssessment) pulid.ID {
		return a.DebitMemoInvoiceID
	})
}

func (r *repository) listGrouped(
	ctx context.Context,
	req *repositories.ListLateChargeAssessmentsByInvoiceIDsRequest,
	column buncolgen.Column,
	keyOf func(*latecharge.LateChargeAssessment) pulid.ID,
) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
	result := make(map[pulid.ID][]*latecharge.LateChargeAssessment, len(req.InvoiceIDs))
	if len(req.InvoiceIDs) == 0 {
		return result, nil
	}

	cols := buncolgen.LateChargeAssessmentColumns
	rows := make([]*latecharge.LateChargeAssessment, 0, len(req.InvoiceIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.LateChargeAssessmentScopeTenant(sq, req.TenantInfo).
				Where(column.In(), bun.List(req.InvoiceIDs))
		}).
		Order(cols.AsOfDate.OrderDesc(), cols.PeriodIndex.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list late charge assessments: %w", err)
	}

	for _, row := range rows {
		key := keyOf(row)
		result[key] = append(result[key], row)
	}

	return result, nil
}

func (r *repository) CountBySourceInvoiceID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID pulid.ID,
) (int64, error) {
	cols := buncolgen.LateChargeAssessmentColumns
	count, err := r.db.DBForContext(ctx).NewSelect().
		Model((*latecharge.LateChargeAssessment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.LateChargeAssessmentScopeTenant(sq, tenantInfo).
				Where(cols.SourceInvoiceID.Eq(), invoiceID)
		}).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count late charge assessments: %w", err)
	}

	return int64(count), nil
}

type tenantRecord struct {
	OrganizationID string `bun:"organization_id"`
	BusinessUnitID string `bun:"business_unit_id"`
}

func (r *repository) ListLateChargeTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	if limit <= 0 {
		limit = 1000
	}
	records := make([]*tenantRecord, 0)
	if err := r.db.DBForContext(ctx).NewRaw(`
		SELECT bc.organization_id, bc.business_unit_id
		FROM billing_controls bc
		WHERE bc.late_charge_assessment_mode <> 'Disabled'
		ORDER BY bc.organization_id ASC
		LIMIT ?`, limit).Scan(ctx, &records); err != nil {
		return nil, fmt.Errorf("list late charge tenants: %w", err)
	}

	tenants := make([]pagination.TenantInfo, 0, len(records))
	for _, rec := range records {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: pulid.ID(rec.OrganizationID),
			BuID:  pulid.ID(rec.BusinessUnitID),
		})
	}

	return tenants, nil
}
