package accountingpayablesrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingpayablesports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	carrierSettlementErrors = dberror.Mapping{Entity: "Carrier settlement"}
	driverSettlementErrors  = dberror.Mapping{Entity: "Driver settlement"}
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

func NewPayablesRepository(p PayablesParams) accountingpayablesports.AccountingPayablesSource {
	return &payablesRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-payables"),
	}
}

func (r *payablesRepository) GetSettlement(
	ctx context.Context,
	req *accountingpayablesports.GetPayableSettlementRequest,
) (*accountingpayablesports.PayableSettlement, error) {
	switch req.Kind {
	case accountingpayablesports.PayableCarrier:
		return r.carrierSettlement(ctx, req)
	case accountingpayablesports.PayableDriver:
		return r.driverSettlement(ctx, req)
	default:
		return nil, fmt.Errorf("accounting payables: unknown settlement kind %q", req.Kind)
	}
}

func (r *payablesRepository) carrierSettlement(
	ctx context.Context,
	req *accountingpayablesports.GetPayableSettlementRequest,
) (*accountingpayablesports.PayableSettlement, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingpayablesports.PayableSettlement, error) {
			cs := carriersettlementdb.CarrierSettlements
			entity, err := cs.Where(cs.ID.Eq(req.ID)).Load(cs.Carrier()).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, carrierSettlementErrors.Map(err, req.ID.String())
			}

			out := &accountingpayablesports.PayableSettlement{
				Kind:                 accountingpayablesports.PayableCarrier,
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
			if out.InvoiceNumbers, err = r.invoiceNumbers(
				ctx,
				req.TenantInfo,
				entity.ID,
			); err != nil {
				return nil, err
			}
			return out, nil
		},
	)
}

func (r *payablesRepository) driverSettlement(
	ctx context.Context,
	req *accountingpayablesports.GetPayableSettlementRequest,
) (*accountingpayablesports.PayableSettlement, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingpayablesports.PayableSettlement, error) {
			ds := driversettlementdb.Settlements
			entity, err := ds.Where(ds.ID.Eq(req.ID)).Load(ds.Worker()).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, driverSettlementErrors.Map(err, req.ID.String())
			}

			control, err := r.accountingControl(ctx, req.TenantInfo)
			if err != nil {
				return nil, err
			}

			out := &accountingpayablesports.PayableSettlement{
				Kind:                 accountingpayablesports.PayableDriver,
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
		},
	)
}

func (r *payablesRepository) accountingControl(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*accountingcontrol.AccountingControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingcontrol.AccountingControl, error) {
			control, err := tenantdb.AccountingControls.Query().First(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil //nolint:nilnil // a tenant without an accounting control has no default accounts
				}
				return nil, fmt.Errorf("load accounting control: %w", err)
			}
			return control, nil
		},
	)
}

func (r *payablesRepository) journalLines(
	ctx context.Context,
	_ pagination.TenantInfo,
	batchID *pulid.ID,
) ([]accountingpayablesports.PayableJournalLine, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingpayablesports.PayableJournalLine, error) {
			if batchID == nil || batchID.IsNil() {
				return nil, nil
			}

			je := journalentrydb.JournalEntries
			jel := journalentrydb.JournalEntryLines
			lines, err := jel.Where(
				jel.JournalEntryID.InQuery(je.Pick(je.ID).Where(je.BatchID.Eq(*batchID))),
			).
				Load(jel.GLAccount()).
				Order(jel.LineNumber.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to load settlement journal lines", zap.Error(err))
				return nil, fmt.Errorf("load settlement journal lines: %w", err)
			}

			byAccount := make(map[pulid.ID]int, len(lines))
			out := make([]accountingpayablesports.PayableJournalLine, 0, len(lines))
			for _, line := range lines {
				idx, seen := byAccount[line.GLAccountID]
				if !seen {
					entry := accountingpayablesports.PayableJournalLine{AccountID: line.GLAccountID}
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
		},
	)
}

func carrierBankAccount(
	paid []accountingpayablesports.PayableJournalLine,
	entity *carriersettlement.CarrierSettlement,
	control *accountingcontrol.AccountingControl,
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

func defaultAccounts(
	control *accountingcontrol.AccountingControl,
) accountingpayablesports.PayableDefaultAccounts {
	if control == nil {
		return accountingpayablesports.PayableDefaultAccounts{}
	}
	return accountingpayablesports.PayableDefaultAccounts{
		Payable:                 control.DefaultAPAccountID,
		PurchasedTransportation: control.DefaultPurchasedTransportationAccountID,
		Cash:                    control.DefaultCashAccountID,
	}
}

func (r *payablesRepository) invoiceNumbers(
	ctx context.Context,
	_ pagination.TenantInfo,
	settlementID pulid.ID,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		im := carriersettlementdb.InvoiceMatches
		numbers, err := keel.Into[string](ctx, r.db.Keel(ctx), im.Pick(im.InvoiceNumber).Where(
			im.CarrierSettlementID.Eq(settlementID),
			im.Status.In(
				carriersettlement.InvoiceMatchStatusMatched,
				carriersettlement.InvoiceMatchStatusResolved,
			),
			im.InvoiceNumber.IsNotNull(),
		))
		if err != nil {
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
	req *accountingpayablesports.StampPayableExchangeRateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rate := decimal.NullDecimal{Decimal: req.Rate, Valid: true}
		var (
			affected int64
			err      error
			label    string
		)
		switch req.Kind {
		case accountingpayablesports.PayableCarrier:
			cs := carriersettlementdb.CarrierSettlements
			rateCol, dateCol := cs.ExchangeRate, cs.ExchangeRateDate
			if req.Paid {
				rateCol, dateCol = cs.PaidExchangeRate, cs.PaidExchangeRateDate
			}
			label = "Carrier settlement"
			affected, err = cs.Where(cs.ID.Eq(req.ID)).Update(
				ctx,
				r.db.Keel(ctx),
				rateCol.To(rate),
				dateCol.To(req.Date),
				cs.Version.ToExpr(cs.Version),
			)
		case accountingpayablesports.PayableDriver:
			ds := driversettlementdb.Settlements
			rateCol, dateCol := ds.ExchangeRate, ds.ExchangeRateDate
			if req.Paid {
				rateCol, dateCol = ds.PaidExchangeRate, ds.PaidExchangeRateDate
			}
			label = "Driver settlement"
			affected, err = ds.Where(ds.ID.Eq(req.ID)).Update(
				ctx,
				r.db.Keel(ctx),
				rateCol.To(rate),
				dateCol.To(req.Date),
				ds.Version.ToExpr(ds.Version),
			)
		default:
			return fmt.Errorf("stamp payable exchange rate: unknown payable kind %q", req.Kind)
		}
		if err != nil {
			return fmt.Errorf("stamp %s exchange rate: %w", strings.ToLower(label), err)
		}

		return dberror.CheckFoundCount(affected, label)
	})
}
