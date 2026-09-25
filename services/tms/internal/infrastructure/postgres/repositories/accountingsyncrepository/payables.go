package accountingsyncrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type PayablesParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type payablesRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayablesRepository(p PayablesParams) repositories.AccountingPayablesSource {
	return &payablesRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-payables"),
	}
}

func (r *payablesRepository) GetSettlement(
	ctx context.Context,
	req *repositories.GetPayableSettlementRequest,
) (*repositories.PayableSettlement, error) {
	switch req.Kind {
	case repositories.PayableCarrier:
		return r.carrierSettlement(ctx, req)
	case repositories.PayableDriver:
		return r.driverSettlement(ctx, req)
	default:
		return nil, fmt.Errorf("accounting payables: unknown settlement kind %q", req.Kind)
	}
}

func (r *payablesRepository) carrierSettlement(
	ctx context.Context,
	req *repositories.GetPayableSettlementRequest,
) (*repositories.PayableSettlement, error) {
	entity := new(carriersettlement.CarrierSettlement)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.CarrierSettlementRelations.Carrier).
		Apply(buncolgen.CarrierSettlementApplyTenant(req.TenantInfo)).
		Where(buncolgen.CarrierSettlementColumns.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Carrier settlement")
	}

	out := &repositories.PayableSettlement{
		Kind:             repositories.PayableCarrier,
		ID:               entity.ID,
		Number:           entity.SettlementNumber,
		PartyID:          entity.CarrierID,
		PeriodStart:      entity.PeriodStart,
		PeriodEnd:        entity.PeriodEnd,
		PayDate:          entity.PayDate,
		PostedAt:         entity.PostedAt,
		PaidAt:           entity.PaidAt,
		NetMinor:         entity.NetPayableMinor,
		ShipmentCount:    entity.ShipmentCount,
		CurrencyCode:     entity.CurrencyCode,
		PaymentMethod:    entity.PaymentMethod,
		PaymentReference: entity.PaymentReference,
		PayableAccountID: pulid.ConvertFromPtr(entity.PostedAPAccountID),
	}
	if entity.Carrier != nil {
		out.PartyName = entity.Carrier.Name
	}

	var err error
	if out.Lines, err = r.journalLines(
		ctx,
		req.TenantInfo,
		entity.PostedJournalBatchID,
	); err != nil {
		return nil, err
	}
	if out.BankAccountID, err = r.carrierBankAccount(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}
	if out.InvoiceNumbers, err = r.invoiceNumbers(ctx, req.TenantInfo, entity.ID); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *payablesRepository) driverSettlement(
	ctx context.Context,
	req *repositories.GetPayableSettlementRequest,
) (*repositories.PayableSettlement, error) {
	entity := new(driversettlement.Settlement)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.SettlementRelations.Worker).
		Apply(buncolgen.SettlementApplyTenant(req.TenantInfo)).
		Where(buncolgen.SettlementColumns.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Driver settlement")
	}

	control, err := r.accountingControl(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	out := &repositories.PayableSettlement{
		Kind:             repositories.PayableDriver,
		ID:               entity.ID,
		Number:           entity.SettlementNumber,
		PartyID:          entity.WorkerID,
		OwnerOperator:    entity.Classification == driverpay.PayeeClassificationOwnerOperator,
		PeriodStart:      entity.PeriodStart,
		PeriodEnd:        entity.PeriodEnd,
		PayDate:          entity.PayDate,
		PostedAt:         entity.PostedAt,
		PaidAt:           entity.PaidAt,
		NetMinor:         entity.NetPayMinor,
		ShipmentCount:    entity.ShipmentCount,
		CurrencyCode:     entity.CurrencyCode,
		PaymentMethod:    entity.PaymentMethod,
		PaymentReference: entity.PaymentReference,
		PayableAccountID: pulid.ConvertFromPtr(entity.PostedPayableAccountID),
	}
	if entity.Worker != nil {
		out.PartyName = strings.TrimSpace(
			strings.TrimSpace(
				entity.Worker.FirstName,
			) + " " + strings.TrimSpace(
				entity.Worker.LastName,
			),
		)
	}
	if control != nil {
		if out.PayableAccountID.IsNil() {
			out.PayableAccountID = control.DefaultSettlementsPayableAccountID
		}
		out.BankAccountID = control.DefaultCashAccountID
	}

	if out.Lines, err = r.journalLines(
		ctx,
		req.TenantInfo,
		entity.PostedJournalBatchID,
	); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *payablesRepository) accountingControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AccountingControl, error) {
	control := new(tenant.AccountingControl)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(control).
		Where(buncolgen.AccountingControlColumns.OrganizationID.Eq(), tenantInfo.OrgID).
		Limit(1).
		Scan(ctx); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // a tenant without an accounting control has no default accounts
		}
		return nil, fmt.Errorf("load accounting control: %w", err)
	}
	return control, nil
}

func (r *payablesRepository) journalLines(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID *pulid.ID,
) ([]repositories.PayableJournalLine, error) {
	if batchID == nil || batchID.IsNil() {
		return nil, nil
	}

	entries := r.db.DBForContext(ctx).
		NewSelect().
		Model((*journalentry.JournalEntry)(nil)).
		ColumnExpr(buncolgen.JournalEntryColumns.ID.Qualified()).
		Apply(buncolgen.JournalEntryApplyTenant(tenantInfo)).
		Where(buncolgen.JournalEntryColumns.BatchID.Eq(), *batchID)

	lines := make([]*journalentry.JournalEntryLine, 0, 8)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&lines).
		Relation(buncolgen.JournalEntryLineRelations.GLAccount).
		Apply(buncolgen.JournalEntryLineApplyTenant(tenantInfo)).
		Where(buncolgen.Expr("{0} IN (?)", buncolgen.JournalEntryLineColumns.JournalEntryID), entries).
		Order(buncolgen.JournalEntryLineColumns.LineNumber.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to load settlement journal lines", zap.Error(err))
		return nil, fmt.Errorf("load settlement journal lines: %w", err)
	}

	byAccount := make(map[pulid.ID]int, len(lines))
	out := make([]repositories.PayableJournalLine, 0, len(lines))
	for _, line := range lines {
		idx, seen := byAccount[line.GLAccountID]
		if !seen {
			entry := repositories.PayableJournalLine{AccountID: line.GLAccountID}
			if line.GLAccount != nil {
				entry.AccountCode = line.GLAccount.AccountCode
				entry.AccountName = line.GLAccount.Name
			}
			out = append(out, entry)
			idx = len(out) - 1
			byAccount[line.GLAccountID] = idx
		}
		out[idx].DebitMinor += line.DebitAmount
		out[idx].CreditMinor += line.CreditAmount
	}
	return out, nil
}

func (r *payablesRepository) carrierBankAccount(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *carriersettlement.CarrierSettlement,
) (pulid.ID, error) {
	paid, err := r.journalLines(ctx, tenantInfo, entity.PaidJournalBatchID)
	if err != nil {
		return pulid.Nil, err
	}
	payable := pulid.ConvertFromPtr(entity.PostedAPAccountID)
	for idx := range paid {
		if paid[idx].AccountID != payable {
			return paid[idx].AccountID, nil
		}
	}

	control, err := r.accountingControl(ctx, tenantInfo)
	if err != nil || control == nil {
		return pulid.Nil, err
	}
	return control.DefaultCashAccountID, nil
}

func (r *payablesRepository) invoiceNumbers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) ([]string, error) {
	cols := buncolgen.InvoiceMatchColumns
	numbers := make([]string, 0, 4)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carriersettlement.InvoiceMatch)(nil)).
		ColumnExpr(cols.InvoiceNumber.Qualified()).
		Apply(buncolgen.InvoiceMatchApplyTenant(tenantInfo)).
		Where(cols.CarrierSettlementID.Eq(), settlementID).
		Where(cols.Status.In(), bun.List([]carriersettlement.InvoiceMatchStatus{
			carriersettlement.InvoiceMatchStatusMatched,
			carriersettlement.InvoiceMatchStatusResolved,
		})).
		Where(cols.InvoiceNumber.IsNotNull()).
		Scan(ctx, &numbers); err != nil {
		r.l.Error("failed to load carrier invoice numbers", zap.Error(err))
		return nil, fmt.Errorf("load carrier invoice numbers: %w", err)
	}

	distinct := make([]string, 0, len(numbers))
	for _, number := range numbers {
		if trimmed := strings.TrimSpace(
			number,
		); trimmed != "" &&
			!slices.Contains(distinct, trimmed) {
			distinct = append(distinct, trimmed)
		}
	}
	slices.Sort(distinct)
	return distinct, nil
}
