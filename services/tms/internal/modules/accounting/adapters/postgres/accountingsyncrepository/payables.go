package accountingsyncrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/carriersettlementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/driversettlementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/journalentrycol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func NewPayablesRepository(p PayablesParams) accountingsyncports.AccountingPayablesSource {
	return &payablesRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-payables"),
	}
}

func (r *payablesRepository) GetSettlement(
	ctx context.Context,
	req *accountingsyncports.GetPayableSettlementRequest,
) (*accountingsyncports.PayableSettlement, error) {
	switch req.Kind {
	case accountingsyncports.PayableCarrier:
		return r.carrierSettlement(ctx, req)
	case accountingsyncports.PayableDriver:
		return r.driverSettlement(ctx, req)
	default:
		return nil, fmt.Errorf("accounting payables: unknown settlement kind %q", req.Kind)
	}
}

func (r *payablesRepository) carrierSettlement(
	ctx context.Context,
	req *accountingsyncports.GetPayableSettlementRequest,
) (*accountingsyncports.PayableSettlement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*accountingsyncports.PayableSettlement, error) {
		entity := new(carriersettlement.CarrierSettlement)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Relation(carriersettlementcol.CarrierSettlementRelations.Carrier).
			Apply(carriersettlementcol.CarrierSettlementApplyTenant(req.TenantInfo)).
			Where(carriersettlementcol.CarrierSettlementColumns.ID.Eq(), req.ID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Carrier settlement")
		}

		out := &accountingsyncports.PayableSettlement{
			Kind:                 accountingsyncports.PayableCarrier,
			ID:                   entity.ID,
			Number:               entity.SettlementNumber,
			PartyID:              entity.CarrierID,
			PeriodStart:          entity.PeriodStart,
			PeriodEnd:            entity.PeriodEnd,
			PayDate:              entity.PayDate,
			PostedAt:             entity.PostedAt,
			PaidAt:               entity.PaidAt,
			Voided:               entity.Status == carriersettlement.StatusVoided,
			NetMinor:             entity.NetPayableMinor,
			ShipmentCount:        entity.ShipmentCount,
			CurrencyCode:         entity.CurrencyCode,
			ExchangeRate:         entity.ExchangeRate,
			ExchangeRateDate:     entity.ExchangeRateDate,
			PaidExchangeRate:     entity.PaidExchangeRate,
			PaidExchangeRateDate: entity.PaidExchangeRateDate,
			PaymentMethod:        entity.PaymentMethod,
			PaymentReference:     entity.PaymentReference,
			PayableAccountID:     pulid.ConvertFromPtr(entity.PostedAPAccountID),
		}
		if entity.Carrier != nil {
			out.PartyName = entity.Carrier.Name
		}

		control, err := r.accountingControl(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		out.Defaults = defaultAccounts(control)

		if out.Lines, err = r.journalLines(
			ctx,
			req.TenantInfo,
			entity.PostedJournalBatchID,
		); err != nil {
			return nil, err
		}
		paid, err := r.journalLines(ctx, req.TenantInfo, entity.PaidJournalBatchID)
		if err != nil {
			return nil, err
		}
		out.BankAccountID = carrierBankAccount(paid, entity, control)
		if out.InvoiceNumbers, err = r.invoiceNumbers(ctx, req.TenantInfo, entity.ID); err != nil {
			return nil, err
		}
		return out, nil
	})
}

func (r *payablesRepository) driverSettlement(
	ctx context.Context,
	req *accountingsyncports.GetPayableSettlementRequest,
) (*accountingsyncports.PayableSettlement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*accountingsyncports.PayableSettlement, error) {
		entity := new(driversettlement.Settlement)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Relation(driversettlementcol.SettlementRelations.Worker).
			Apply(driversettlementcol.SettlementApplyTenant(req.TenantInfo)).
			Where(driversettlementcol.SettlementColumns.ID.Eq(), req.ID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Driver settlement")
		}

		control, err := r.accountingControl(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}

		out := &accountingsyncports.PayableSettlement{
			Kind:                 accountingsyncports.PayableDriver,
			ID:                   entity.ID,
			Number:               entity.SettlementNumber,
			PartyID:              entity.WorkerID,
			OwnerOperator:        entity.Classification == driverpay.PayeeClassificationOwnerOperator,
			PeriodStart:          entity.PeriodStart,
			PeriodEnd:            entity.PeriodEnd,
			PayDate:              entity.PayDate,
			PostedAt:             entity.PostedAt,
			PaidAt:               entity.PaidAt,
			Voided:               entity.Status == driversettlement.StatusVoided,
			NetMinor:             entity.NetPayMinor,
			ShipmentCount:        entity.ShipmentCount,
			CurrencyCode:         entity.CurrencyCode,
			ExchangeRate:         entity.ExchangeRate,
			ExchangeRateDate:     entity.ExchangeRateDate,
			PaidExchangeRate:     entity.PaidExchangeRate,
			PaidExchangeRateDate: entity.PaidExchangeRateDate,
			PaymentMethod:        entity.PaymentMethod,
			PaymentReference:     entity.PaymentReference,
			PayableAccountID:     pulid.ConvertFromPtr(entity.PostedPayableAccountID),
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
		out.Defaults = defaultAccounts(control)
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
	})
}

func (r *payablesRepository) accountingControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AccountingControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.AccountingControl, error) {
		control := new(tenant.AccountingControl)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(control).
			Where(tenantcol.AccountingControlColumns.OrganizationID.Eq(), tenantInfo.OrgID).
			Limit(1).
			Scan(ctx); err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // a tenant without an accounting control has no default accounts
			}
			return nil, fmt.Errorf("load accounting control: %w", err)
		}
		return control, nil
	})
}

func (r *payablesRepository) journalLines(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID *pulid.ID,
) ([]accountingsyncports.PayableJournalLine, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]accountingsyncports.PayableJournalLine, error) {
		if batchID == nil || batchID.IsNil() {
			return nil, nil
		}

		entries := r.db.DBForContext(ctx).
			NewSelect().
			Model((*journalentry.JournalEntry)(nil)).
			ColumnExpr(journalentrycol.JournalEntryColumns.ID.Qualified()).
			Apply(journalentrycol.JournalEntryApplyTenant(tenantInfo)).
			Where(journalentrycol.JournalEntryColumns.BatchID.Eq(), *batchID)

		lines := make([]*journalentry.JournalEntryLine, 0, 8)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&lines).
			Relation(journalentrycol.JournalEntryLineRelations.GLAccount).
			Apply(journalentrycol.JournalEntryLineApplyTenant(tenantInfo)).
			Where(buncolgen.Expr("{0} IN (?)", journalentrycol.JournalEntryLineColumns.JournalEntryID), entries).
			Order(journalentrycol.JournalEntryLineColumns.LineNumber.OrderAsc()).
			Scan(ctx); err != nil {
			r.l.Error("failed to load settlement journal lines", zap.Error(err))
			return nil, fmt.Errorf("load settlement journal lines: %w", err)
		}

		byAccount := make(map[pulid.ID]int, len(lines))
		out := make([]accountingsyncports.PayableJournalLine, 0, len(lines))
		for _, line := range lines {
			idx, seen := byAccount[line.GLAccountID]
			if !seen {
				entry := accountingsyncports.PayableJournalLine{AccountID: line.GLAccountID}
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
	})
}

func carrierBankAccount(
	paid []accountingsyncports.PayableJournalLine,
	entity *carriersettlement.CarrierSettlement,
	control *tenant.AccountingControl,
) pulid.ID {
	payable := pulid.ConvertFromPtr(entity.PostedAPAccountID)
	for idx := range paid {
		if paid[idx].AccountID != payable {
			return paid[idx].AccountID
		}
	}
	if control == nil {
		return pulid.Nil
	}
	return control.DefaultCashAccountID
}

func defaultAccounts(control *tenant.AccountingControl) accountingsyncports.PayableDefaultAccounts {
	if control == nil {
		return accountingsyncports.PayableDefaultAccounts{}
	}
	return accountingsyncports.PayableDefaultAccounts{
		Payable:                 control.DefaultAPAccountID,
		PurchasedTransportation: control.DefaultPurchasedTransportationAccountID,
		Cash:                    control.DefaultCashAccountID,
	}
}

func (r *payablesRepository) invoiceNumbers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	settlementID pulid.ID,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		cols := carriersettlementcol.InvoiceMatchColumns
		numbers := make([]string, 0, 4)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*carriersettlement.InvoiceMatch)(nil)).
			ColumnExpr(cols.InvoiceNumber.Qualified()).
			Apply(carriersettlementcol.InvoiceMatchApplyTenant(tenantInfo)).
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
	})
}

func (r *payablesRepository) StampExchangeRate(
	ctx context.Context,
	req *accountingsyncports.StampPayableExchangeRateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		var (
			q     *bun.UpdateQuery
			label string
		)
		switch req.Kind {
		case accountingsyncports.PayableCarrier:
			cols := carriersettlementcol.CarrierSettlementColumns
			rate, date := cols.ExchangeRate, cols.ExchangeRateDate
			if req.Paid {
				rate, date = cols.PaidExchangeRate, cols.PaidExchangeRateDate
			}
			label = "Carrier settlement"
			q = r.db.DBForContext(ctx).NewUpdate().
				Model((*carriersettlement.CarrierSettlement)(nil)).
				Set(rate.Set(), req.Rate).
				Set(date.Set(), req.Date).
				Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return carriersettlementcol.CarrierSettlementScopeTenantUpdate(uq, req.TenantInfo).
						Where(cols.ID.Eq(), req.ID)
				})
		case accountingsyncports.PayableDriver:
			cols := driversettlementcol.SettlementColumns
			rate, date := cols.ExchangeRate, cols.ExchangeRateDate
			if req.Paid {
				rate, date = cols.PaidExchangeRate, cols.PaidExchangeRateDate
			}
			label = "Driver settlement"
			q = r.db.DBForContext(ctx).NewUpdate().
				Model((*driversettlement.Settlement)(nil)).
				Set(rate.Set(), req.Rate).
				Set(date.Set(), req.Date).
				Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return driversettlementcol.SettlementScopeTenantUpdate(uq, req.TenantInfo).
						Where(cols.ID.Eq(), req.ID)
				})
		default:
			return fmt.Errorf("stamp payable exchange rate: unknown payable kind %q", req.Kind)
		}
		result, err := q.Exec(ctx)
		if err != nil {
			return fmt.Errorf("stamp %s exchange rate: %w", strings.ToLower(label), err)
		}

		return dberror.CheckFound(result, label)
	})
}
