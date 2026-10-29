package accountingfixtures

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol/accountingcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/pricingfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	ledgerAccountOperatingCash   = "1010"
	ledgerAccountTradeReceivable = "1110"
	ledgerAccountLinehaulRevenue = "4010"
	ledgerAccountUnappliedCash   = "2160"
	receivableSeedDay            = int64(86400)
	receivableFuelChargeCode     = "FUEL"
)

type seedReceivable struct {
	number   string
	pro      string
	customer string
	from     string
	to       string
	ageDays  int64
	freight  string
}

type seedReceiptApplication struct {
	invoice string
	amount  string
}

type seedReceipt struct {
	reference    string
	customer     string
	method       customerpayment.Method
	ageDays      int64
	applications []seedReceiptApplication
	unapplied    string
	memo         string
}

type seedCreditMemo struct {
	number  string
	against string
	ageDays int64
	amount  string
	reason  string
}

var seedReceivables = []seedReceivable{
	{
		number:   "SEED-INV-0001",
		pro:      "SEED-AR-001",
		customer: "ACME",
		from:     "WH-DAL",
		to:       "DC-CHI",
		ageDays:  12,
		freight:  "2950.00",
	},
	{
		number:   "SEED-INV-0002",
		pro:      "SEED-AR-002",
		customer: "ACME",
		from:     "TERM-LA",
		to:       "WH-DAL",
		ageDays:  40,
		freight:  "2600.00",
	},
	{
		number:   "SEED-INV-0003",
		pro:      "SEED-AR-003",
		customer: "ACME",
		from:     "WH-DAL",
		to:       "COLD-PHX",
		ageDays:  75,
		freight:  "3800.00",
	},
	{
		number:   "SEED-INV-0004",
		pro:      "SEED-AR-004",
		customer: "ACME",
		from:     "TERM-LA",
		to:       "DC-CHI",
		ageDays:  55,
		freight:  "2400.00",
	},
	{
		number:   "SEED-INV-0005",
		pro:      "SEED-AR-005",
		customer: "PEAK",
		from:     "WH-DAL",
		to:       "DC-CHI",
		ageDays:  20,
		freight:  "1850.00",
	},
	{
		number:   "SEED-INV-0006",
		pro:      "SEED-AR-006",
		customer: "PEAK",
		from:     "CUST-DEN",
		to:       "DC-CHI",
		ageDays:  95,
		freight:  "2250.00",
	},
	{
		number:   "SEED-INV-0007",
		pro:      "SEED-AR-007",
		customer: "PEAK",
		from:     "WH-DAL",
		to:       "CUST-DEN",
		ageDays:  130,
		freight:  "1620.00",
	},
	{
		number:   "SEED-INV-0008",
		pro:      "SEED-AR-008",
		customer: "FRSH",
		from:     "COLD-PHX",
		to:       "TERM-LA",
		ageDays:  8,
		freight:  "3400.00",
	},
	{
		number:   "SEED-INV-0009",
		pro:      "SEED-AR-009",
		customer: "FRSH",
		from:     "COLD-PHX",
		to:       "WH-DAL",
		ageDays:  50,
		freight:  "2050.00",
	},
	{
		number:   "SEED-INV-0010",
		pro:      "SEED-AR-010",
		customer: "FRSH",
		from:     "TERM-LA",
		to:       "COLD-PHX",
		ageDays:  160,
		freight:  "900.00",
	},
}

var seedReceipts = []seedReceipt{
	{
		reference: "SEED-RCPT-0001",
		customer:  "ACME",
		method:    customerpayment.MethodACH,
		ageDays:   15,
		applications: []seedReceiptApplication{
			{invoice: "SEED-INV-0004"},
			{invoice: "SEED-INV-0002", amount: "1000.00"},
		},
		memo: "ACH remittance: SEED-INV-0004 in full, SEED-INV-0002 in part",
	},
	{
		reference: "SEED-RCPT-0002",
		customer:  "FRSH",
		method:    customerpayment.MethodCheck,
		ageDays:   10,
		applications: []seedReceiptApplication{
			{invoice: "SEED-INV-0009", amount: "1200.00"},
		},
		unapplied: "500.00",
		memo:      "Check 40418: $1,200.00 against SEED-INV-0009, $500.00 with no remittance",
	},
}

var seedCreditMemos = []seedCreditMemo{
	{
		number:  "SEED-CM-0001",
		against: "SEED-INV-0007",
		ageDays: 30,
		amount:  "260.00",
		reason:  "Allowance for two pallets damaged in transit on SEED-AR-007",
	},
}

type ReceivablesSeed struct {
}

type receivableSeedRefs struct {
	orgID        pulid.ID
	buID         pulid.ID
	adminID      pulid.ID
	now          int64
	ledgerFloor  int64
	accounts     map[string]pulid.ID
	customers    map[string]*customer.Customer
	locations    map[string]pulid.ID
	serviceType  pulid.ID
	shipmentType pulid.ID
	template     pulid.ID
	fuel         *accessorialcharge.AccessorialCharge
	invoices     map[string]*invoice.Invoice
}

func (s *ReceivablesSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	refs, err := s.loadRefs(ctx, db, sr)
	if err != nil {
		return err
	}

	created := 0
	for idx := range seedReceivables {
		fresh, invErr := s.ensureInvoice(ctx, db, sr, refs, &seedReceivables[idx])
		if invErr != nil {
			return fmt.Errorf("invoice %s: %w", seedReceivables[idx].number, invErr)
		}
		if fresh {
			created++
		}
	}
	for idx := range seedCreditMemos {
		if err = s.ensureCreditMemo(ctx, db, sr, refs, &seedCreditMemos[idx]); err != nil {
			return fmt.Errorf("credit memo %s: %w", seedCreditMemos[idx].number, err)
		}
	}
	for idx := range seedReceipts {
		if err = s.ensureReceipt(ctx, db, sr, refs, &seedReceipts[idx]); err != nil {
			return fmt.Errorf("receipt %s: %w", seedReceipts[idx].reference, err)
		}
	}

	if created > 0 {
	}

	return nil
}

func (s *ReceivablesSeed) loadRefs(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
) (*receivableSeedRefs, error) {
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return nil, err
	}
	admin, err := tenantfixtures.UserByUsername(ctx, db, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}

	refs := &receivableSeedRefs{
		orgID:    org.ID,
		buID:     org.BusinessUnitID,
		adminID:  admin.ID,
		now:      timeutils.NowUnix(),
		invoices: make(map[string]*invoice.Invoice, len(seedReceivables)+len(seedCreditMemos)),
	}

	accounts, _, err := ensureGLAccounts(ctx, db, &seedGLAccountsRequest{
		orgID: org.ID,
		buID:  org.BusinessUnitID,
		accounts: []seedGLAccount{{
			code:        ledgerAccountUnappliedCash,
			name:        "Unapplied Customer Cash",
			description: "Customer receipts held until they are applied to an invoice",
			category:    "Liability",
		}},
	})
	if err != nil {
		return nil, err
	}
	for _, code := range []string{
		ledgerAccountOperatingCash,
		ledgerAccountTradeReceivable,
		ledgerAccountLinehaulRevenue,
		ledgerAccountUnappliedCash,
	} {
		if _, ok := accounts[code]; !ok {
			return nil, fmt.Errorf("required gl account %s not found", code)
		}
	}
	refs.accounts = accounts

	if err = s.configureReceivableAccounts(ctx, db, refs); err != nil {
		return nil, err
	}
	if refs.ledgerFloor, err = s.firstOpenPeriodStart(ctx, db, refs); err != nil {
		return nil, err
	}
	if err = s.loadLookups(ctx, db, refs); err != nil {
		return nil, err
	}

	return refs, nil
}

func (s *ReceivablesSeed) configureReceivableAccounts(
	ctx context.Context,
	db *keel.DB,
	refs *receivableSeedRefs,
) error {
	ac := accountingcontroldb.AccountingControls
	defaults := []struct {
		column keelkinds.PulidIDCol
		code   string
	}{
		{column: ac.DefaultARAccountID, code: ledgerAccountTradeReceivable},
		{column: ac.DefaultRevenueAccountID, code: ledgerAccountLinehaulRevenue},
		{column: ac.DefaultCashAccountID, code: ledgerAccountOperatingCash},
		{column: ac.DefaultUnappliedCashAccountID, code: ledgerAccountUnappliedCash},
	}
	for _, def := range defaults {
		if _, err := ac.AllTenants().Where(
			ac.OrganizationID.Eq(refs.orgID),
			ac.BusinessUnitID.Eq(refs.buID),
			def.column.IsNull(),
		).Update(ctx, db, def.column.To(refs.accounts[def.code])); err != nil {
			return fmt.Errorf("set default account %s: %w", def.code, err)
		}
	}

	return nil
}

func (s *ReceivablesSeed) firstOpenPeriodStart(
	ctx context.Context,
	db *keel.DB,
	refs *receivableSeedRefs,
) (int64, error) {
	fp := fiscalperioddb.FiscalPeriods
	start, err := keel.IntoOne[*int64](ctx, db, fp.AllTenants().Pick(fp.StartDate.Min()).Where(
		fp.OrganizationID.Eq(refs.orgID),
		fp.BusinessUnitID.Eq(refs.buID),
		fp.Status.Eq("Open"),
	))
	if err != nil {
		return 0, fmt.Errorf("get first open fiscal period: %w", err)
	}
	if start == nil {
		return 0, errors.New("no open fiscal period to post receivables into")
	}

	return *start, nil
}

func (s *ReceivablesSeed) loadLookups(
	ctx context.Context,
	db *keel.DB,
	refs *receivableSeedRefs,
) error {
	customers, err := customerdb.Customers.AllTenants().Where(
		customerdb.Customers.OrganizationID.Eq(refs.orgID),
		customerdb.Customers.BusinessUnitID.Eq(refs.buID),
		customerdb.Customers.Code.In("ACME", "PEAK", "FRSH"),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("get customers: %w", err)
	}
	refs.customers = make(map[string]*customer.Customer, len(customers))
	for _, cus := range customers {
		refs.customers[cus.Code] = cus
	}

	l := locationdb.Locations
	locations, err := l.AllTenants().Pick(l.ID, l.Code).Where(
		l.OrganizationID.Eq(refs.orgID),
		l.BusinessUnitID.Eq(refs.buID),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("get locations: %w", err)
	}
	refs.locations = make(map[string]pulid.ID, len(locations))
	for _, loc := range locations {
		refs.locations[loc.Code] = loc.ID
	}

	st := servicetypedb.ServiceTypes
	serviceType, err := st.AllTenants().Pick(st.ID).Where(
		st.OrganizationID.Eq(refs.orgID),
		st.BusinessUnitID.Eq(refs.buID),
		st.Code.Eq("STD"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get service type STD: %w", err)
	}
	refs.serviceType = serviceType.ID

	sht := shipmenttypedb.ShipmentTypes
	shipmentType, err := sht.AllTenants().Pick(sht.ID).Where(
		sht.OrganizationID.Eq(refs.orgID),
		sht.BusinessUnitID.Eq(refs.buID),
		sht.Code.Eq("FTL"),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get shipment type FTL: %w", err)
	}
	refs.shipmentType = shipmentType.ID

	template, err := formulatemplatedb.FormulaTemplates.AllTenants().
		Pick(formulatemplatedb.FormulaTemplates.ID).
		Where(
			formulatemplatedb.FormulaTemplates.OrganizationID.Eq(refs.orgID),
			formulatemplatedb.FormulaTemplates.BusinessUnitID.Eq(refs.buID),
			formulatemplatedb.FormulaTemplates.Name.Eq("Flat Rate"),
		).
		First(ctx, db)
	if err != nil {
		return fmt.Errorf("get flat rate formula template: %w", err)
	}
	refs.template = template.ID

	fuel, err := accessorialchargedb.AccessorialCharges.AllTenants().Where(
		accessorialchargedb.AccessorialCharges.OrganizationID.Eq(refs.orgID),
		accessorialchargedb.AccessorialCharges.BusinessUnitID.Eq(refs.buID),
		accessorialchargedb.AccessorialCharges.Code.Eq(receivableFuelChargeCode),
	).First(ctx, db)
	if err != nil {
		return fmt.Errorf("get fuel surcharge: %w", err)
	}
	refs.fuel = fuel

	return nil
}

func (s *ReceivablesSeed) ensureInvoice(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *receivableSeedRefs,
	spec *seedReceivable,
) (bool, error) {
	existing, err := s.findInvoice(ctx, db, refs, spec.number)
	if err != nil {
		return false, err
	}
	if existing != nil {
		refs.invoices[spec.number] = existing
		return false, nil
	}

	cus, ok := refs.customers[spec.customer]
	if !ok {
		return false, fmt.Errorf("customer %s not found", spec.customer)
	}
	invoiceDate := refs.now - spec.ageDays*receivableSeedDay
	shp, err := s.createDeliveredShipment(ctx, db, sr, refs, spec, cus, invoiceDate)
	if err != nil {
		return false, err
	}

	freight := decimal.RequireFromString(spec.freight)
	fuel := freight.Mul(refs.fuel.Amount).Div(decimal.NewFromInt(100)).Round(2)
	total := freight.Add(fuel)

	queue := &billingqueue.BillingQueueItem{
		OrganizationID:    refs.orgID,
		BusinessUnitID:    refs.buID,
		ShipmentID:        shp.ID,
		BillToCustomerID:  cus.ID,
		Number:            spec.number,
		Status:            billingqueue.StatusPosted,
		BillType:          billingqueue.BillTypeInvoice,
		AdjustmentContext: map[string]any{},
	}
	if err = billingqueuedb.BillingQueueItems.Create(ctx, db, queue); err != nil {
		return false, fmt.Errorf("insert billing queue item: %w", err)
	}

	entity := &invoice.Invoice{
		OrganizationID:     refs.orgID,
		BusinessUnitID:     refs.buID,
		BillingQueueItemID: queue.ID,
		ShipmentID:         shp.ID,
		CustomerID:         cus.ID,
		Number:             spec.number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusPosted,
		Scope:              invoice.ScopeShipment,
		ShipmentCount:      1,
		PostedAt:           &invoiceDate,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       money.DefaultCurrencyCode,
		InvoiceDate:        invoiceDate,
		DueDate:            invoice.DueDateFromPaymentTerm(invoiceDate, invoice.PaymentTermNet30),
		ShipmentProNumber:  spec.pro,
		ShipmentBOL:        shp.BOL,
		BillToName:         cus.Name,
		SubtotalAmount:     total,
		OtherAmount:        decimal.Zero,
		TotalAmount:        total,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines: []*invoice.InvoiceLine{
			{
				LineNumber:  1,
				Type:        invoice.InvoiceLineTypeFreight,
				Description: "Linehaul",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   freight,
				Amount:      freight,
			},
			{
				LineNumber:          2,
				Type:                invoice.InvoiceLineTypeAccessorial,
				Description:         refs.fuel.Description,
				ChargeCode:          refs.fuel.Code,
				AccessorialChargeID: refs.fuel.ID,
				Quantity:            decimal.NewFromInt(1),
				UnitPrice:           fuel,
				Amount:              fuel,
			},
		},
		Detail:            customer.InvoiceDetailDetailed,
		SectionBy:         customer.InvoiceSectionKeyShipment,
		SendStatus:        invoice.SendStatusNotSent,
		FactoringOverride: invoice.FactoringOverrideDefault,
		EDISendStatus:     invoice.EDISendStatusNotSent,
	}
	if err = s.insertInvoice(ctx, db, sr, refs, entity); err != nil {
		return false, err
	}
	if err = s.postInvoice(ctx, db, sr, refs, entity, pulid.Nil); err != nil {
		return false, err
	}
	refs.invoices[spec.number] = entity

	return true, nil
}

func (s *ReceivablesSeed) findInvoice(
	ctx context.Context,
	db *keel.DB,
	refs *receivableSeedRefs,
	number string,
) (*invoice.Invoice, error) {
	entity, err := invoicedb.Invoices.AllTenants().Where(
		invoicedb.Invoices.OrganizationID.Eq(refs.orgID),
		invoicedb.Invoices.BusinessUnitID.Eq(refs.buID),
		invoicedb.Invoices.Number.Eq(number),
	).First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find invoice %s: %w", number, err)
	}

	return entity, nil
}

func (s *ReceivablesSeed) createDeliveredShipment(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *receivableSeedRefs,
	spec *seedReceivable,
	cus *customer.Customer,
	invoiceDate int64,
) (*shipment.Shipment, error) {
	origin, ok := refs.locations[spec.from]
	if !ok {
		return nil, fmt.Errorf("location %s not found", spec.from)
	}
	destination, ok := refs.locations[spec.to]
	if !ok {
		return nil, fmt.Errorf("location %s not found", spec.to)
	}

	pickedUp := invoiceDate - 3*receivableSeedDay
	delivered := invoiceDate - receivableSeedDay
	charge := decimal.NullDecimal{Decimal: decimal.RequireFromString(spec.freight), Valid: true}
	shp := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		BusinessUnitID:    refs.buID,
		OrganizationID:    refs.orgID,
		CustomerID:        cus.ID,
		ServiceTypeID:     refs.serviceType,
		ShipmentTypeID:    refs.shipmentType,
		FormulaTemplateID: refs.template,
		Status:            shipmentvocab.StatusInvoiced,
		ProNumber:         spec.pro,
		BOL:               "BOL-" + spec.pro,
		Pieces:            new(int64(22)),
		Weight:            new(int64(38000)),
		TariffInputs: pricingfixtures.SeededTariffInputs(
			formulatemplate.StandardFlatRate,
			charge.Decimal,
		),
		FreightChargeAmount:    charge,
		TotalChargeAmount:      charge,
		ActualShipDate:         &pickedUp,
		ActualDeliveryDate:     &delivered,
		BillingTransferStatus:  shipmentvocab.BillingTransferPosted,
		MarkedReadyToBillAt:    &delivered,
		TransferredToBillingAt: &delivered,
		BilledAt:               &invoiceDate,
		EntryMethod:            shipmentvocab.EntryMethodManual,
		FreightTerms:           shipmentvocab.FreightTermsPrepaid,
		FreightLines:           []shipment.FreightLine{},
		OtherChargeAmount:      decimal.NewNullDecimal(decimal.RequireFromString("0")),
	}
	shp.FreightLines = pricingfixtures.SeededFreightLines(shp)
	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: refs.buID,
		OrganizationID: refs.orgID,
		ShipmentID:     shp.ID,
		Status:         shipmentvocab.MoveStatusCompleted,
		Loaded:         true,
		Sequence:       0,
		CoverageType:   shipmentvocab.MoveCoverageTypeUnassigned,
	}
	stops := []*shipment.Stop{
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       refs.buID,
			OrganizationID:       refs.orgID,
			ShipmentMoveID:       move.ID,
			LocationID:           origin,
			Type:                 shipmentvocab.StopTypePickup,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             0,
			ScheduledWindowStart: pickedUp - 3600,
			ScheduledWindowEnd:   new(pickedUp + 3600),
			ActualArrival:        new(pickedUp - 1800),
			ActualDeparture:      &pickedUp,

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       refs.buID,
			OrganizationID:       refs.orgID,
			ShipmentMoveID:       move.ID,
			LocationID:           destination,
			Type:                 shipmentvocab.StopTypeDelivery,
			Status:               shipmentvocab.StopStatusCompleted,
			Sequence:             1,
			ScheduledWindowStart: delivered - 3600,
			ScheduledWindowEnd:   new(delivered + 3600),
			ActualArrival:        new(delivered - 2700),
			ActualDeparture:      &delivered,

			ScheduleType: shipmentvocab.StopScheduleTypeOpen,
		},
	}

	if err := shipmentdb.Shipments.Create(ctx, db, shp); err != nil {
		return nil, fmt.Errorf("insert shipment: %w", err)
	}
	if err := shipmentdb.ShipmentMoves.Create(ctx, db, move); err != nil {
		return nil, fmt.Errorf("insert shipment move: %w", err)
	}
	if err := shipmentdb.Stops.CreateMany(ctx, db, stops); err != nil {
		return nil, fmt.Errorf("insert stops: %w", err)
	}

	return shp, nil
}

func (s *ReceivablesSeed) insertInvoice(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *receivableSeedRefs,
	entity *invoice.Invoice,
) error {
	entity.SyncMinorAmounts()
	if err := invoicedb.Invoices.Create(ctx, db, entity); err != nil {
		return fmt.Errorf("insert invoice: %w", err)
	}
	for _, line := range entity.Lines {
		line.ID = pulid.MustNew("invl_")
		line.OrganizationID = refs.orgID
		line.BusinessUnitID = refs.buID
		line.InvoiceID = entity.ID
	}
	if err := invoicedb.InvoiceLines.CreateMany(ctx, db, entity.Lines); err != nil {
		return fmt.Errorf("insert invoice lines: %w", err)
	}

	return nil
}

func (s *ReceivablesSeed) postInvoice(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *receivableSeedRefs,
	entity *invoice.Invoice,
	relatedInvoiceID pulid.ID,
) error {
	event := accountingcontrol.JournalSourceEventInvoicePosted
	debit, credit := refs.accounts[ledgerAccountTradeReceivable],
		refs.accounts[ledgerAccountLinehaulRevenue]
	ledgerAmount := entity.TotalAmountMinor
	if entity.IsCreditMemo() {
		event = accountingcontrol.JournalSourceEventCreditMemoPosted
		debit, credit = credit, debit
		ledgerAmount = -entity.CreditTotalMinor()
	}
	amount := ledgerAmount
	if amount < 0 {
		amount = -amount
	}

	ledgerDate := max(entity.InvoiceDate, refs.ledgerFloor)
	if _, err := postSeedJournal(ctx, db, &seedJournalRequest{
		orgID:          refs.orgID,
		buID:           refs.buID,
		actorID:        refs.adminID,
		accountingDate: ledgerDate,
		batchNumber:    "JB-" + entity.Number,
		entryNumber:    "JE-" + entity.Number,
		description:    "Invoice posted for " + entity.Number,
		sourceType:     "Invoice",
		sourceID:       entity.ID.String(),
		documentNumber: entity.Number,
		event:          event.String(),
		idempotencyKey: "invoice-posted:" + entity.ID.String(),
		autoGenerated:  true,
		legs: []seedJournalLeg{
			{AccountID: debit, Debit: amount, CustomerID: entity.CustomerID},
			{AccountID: credit, Credit: amount, CustomerID: entity.CustomerID},
		},
	}); err != nil {
		return err
	}

	return s.appendLedger(ctx, db, refs, &customerledger.CustomerLedgerEntry{
		CustomerID:       entity.CustomerID,
		SourceObjectType: "Invoice",
		SourceObjectID:   entity.ID.String(),
		SourceEventType:  event.String(),
		RelatedInvoiceID: relatedInvoiceID,
		DocumentNumber:   entity.Number,
		TransactionDate:  ledgerDate,
		LineNumber:       1,
		AmountMinor:      ledgerAmount,
	})
}

func (s *ReceivablesSeed) appendLedger(
	ctx context.Context,
	db *keel.DB,
	refs *receivableSeedRefs,
	entries ...*customerledger.CustomerLedgerEntry,
) error {
	for _, entry := range entries {
		entry.ID = pulid.MustNew("cledg_")
		entry.OrganizationID = refs.orgID
		entry.BusinessUnitID = refs.buID
		entry.CreatedByID = refs.adminID
	}
	if err := customerledgerdb.CustomerLedgerEntries.CreateMany(ctx, db, entries); err != nil {
		return fmt.Errorf("insert customer ledger entries: %w", err)
	}

	return nil
}

func (s *ReceivablesSeed) ensureCreditMemo(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *receivableSeedRefs,
	spec *seedCreditMemo,
) error {
	existing, err := s.findInvoice(ctx, db, refs, spec.number)
	if err != nil || existing != nil {
		return err
	}
	target, ok := refs.invoices[spec.against]
	if !ok {
		return fmt.Errorf("invoice %s not seeded", spec.against)
	}

	memoDate := refs.now - spec.ageDays*receivableSeedDay
	amount := decimal.RequireFromString(spec.amount)
	queue := &billingqueue.BillingQueueItem{
		OrganizationID:   refs.orgID,
		BusinessUnitID:   refs.buID,
		BillToCustomerID: target.CustomerID,
		Number:           spec.number,
		Status:           billingqueue.StatusPosted,
		BillType:         billingqueue.BillTypeCreditMemo,
		AdjustmentContext: map[string]any{
			"memo":     true,
			"memoKind": string(invoice.MemoKindManual),
		},
	}
	if err = billingqueuedb.BillingQueueItems.Create(ctx, db, queue); err != nil {
		return fmt.Errorf("insert credit memo queue item: %w", err)
	}

	memo := &invoice.Invoice{
		OrganizationID:     refs.orgID,
		BusinessUnitID:     refs.buID,
		BillingQueueItemID: queue.ID,
		CustomerID:         target.CustomerID,
		Number:             spec.number,
		BillType:           billingqueue.BillTypeCreditMemo,
		Status:             invoice.StatusPosted,
		Scope:              invoice.ScopeMemo,
		PostedAt:           &memoDate,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       money.DefaultCurrencyCode,
		InvoiceDate:        memoDate,
		DueDate:            &memoDate,
		BillToName:         target.BillToName,
		SubtotalAmount:     amount,
		OtherAmount:        decimal.Zero,
		TotalAmount:        amount,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		ReferenceInvoiceID: target.ID,
		MemoReason:         spec.reason,
		MemoKind:           invoice.MemoKindManual,
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeMemo,
			Description: spec.reason,
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   amount,
			Amount:      amount,
		}},
		Detail:            customer.InvoiceDetailDetailed,
		SectionBy:         customer.InvoiceSectionKeyShipment,
		SendStatus:        invoice.SendStatusNotSent,
		FactoringOverride: invoice.FactoringOverrideDefault,
		EDISendStatus:     invoice.EDISendStatusNotSent,
	}
	if err = s.insertInvoice(ctx, db, sr, refs, memo); err != nil {
		return err
	}
	if err = s.postInvoice(ctx, db, sr, refs, memo, target.ID); err != nil {
		return err
	}

	applied := memo.CreditRemainingMinor()
	target.ApplyPaymentMinor(applied)
	memo.ApplyCreditMinor(applied)
	application := &customerpayment.CreditMemoApplication{
		ID:                  pulid.MustNew("cma_"),
		OrganizationID:      refs.orgID,
		BusinessUnitID:      refs.buID,
		CreditMemoInvoiceID: memo.ID,
		InvoiceID:           target.ID,
		AppliedAmountMinor:  applied,
		AccountingDate:      max(memoDate, refs.ledgerFloor),
		LineNumber:          1,
		Status:              customerpayment.CreditApplicationStatusApplied,
		CreatedByID:         refs.adminID,
	}
	if err = customerpaymentdb.CreditMemoApplications.Create(ctx, db, application); err != nil {
		return fmt.Errorf("insert credit memo application: %w", err)
	}

	return s.saveSettlement(ctx, db, refs, target, memo)
}

func (s *ReceivablesSeed) ensureReceipt(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *receivableSeedRefs,
	spec *seedReceipt,
) error {
	cus, ok := refs.customers[spec.customer]
	if !ok {
		return fmt.Errorf("customer %s not found", spec.customer)
	}

	paymentDate := refs.now - spec.ageDays*receivableSeedDay
	payment := &customerpayment.Payment{
		ID:              pulid.MustNew("cpay_"),
		OrganizationID:  refs.orgID,
		BusinessUnitID:  refs.buID,
		CustomerID:      cus.ID,
		PaymentDate:     paymentDate,
		AccountingDate:  max(paymentDate, refs.ledgerFloor),
		Status:          customerpayment.StatusPosted,
		PaymentMethod:   spec.method,
		ReferenceNumber: spec.reference,
		Memo:            spec.memo,
		CurrencyCode:    money.DefaultCurrencyCode,
		CreatedByID:     refs.adminID,
		UpdatedByID:     refs.adminID,
	}

	settled := make([]*invoice.Invoice, 0, len(spec.applications))
	for idx, app := range spec.applications {
		target, found := refs.invoices[app.invoice]
		if !found {
			return fmt.Errorf("invoice %s not seeded", app.invoice)
		}
		amount := target.OpenBalanceMinor()
		if app.amount != "" {
			amount = money.MinorUnits(decimal.RequireFromString(app.amount))
		}
		target.ApplyPaymentMinor(amount)
		settled = append(settled, target)
		payment.Applications = append(payment.Applications, &customerpayment.Application{
			ID:                 pulid.MustNew("cpapp_"),
			OrganizationID:     refs.orgID,
			BusinessUnitID:     refs.buID,
			CustomerPaymentID:  payment.ID,
			InvoiceID:          target.ID,
			AppliedAmountMinor: amount,
			LineNumber:         idx + 1,
		})
		payment.AmountMinor += amount
	}
	if spec.unapplied != "" {
		payment.AmountMinor += money.MinorUnits(decimal.RequireFromString(spec.unapplied))
	}
	payment.SyncAmounts()

	description := "Customer payment " + payment.DocumentLabel()
	legs := []seedJournalLeg{{
		AccountID:  refs.accounts[ledgerAccountOperatingCash],
		Debit:      payment.AmountMinor,
		CustomerID: cus.ID,
	}}
	for _, app := range payment.Applications {
		legs = append(legs, seedJournalLeg{
			AccountID:  refs.accounts[ledgerAccountTradeReceivable],
			Credit:     app.AppliedAmountMinor,
			CustomerID: cus.ID,
		})
	}
	if payment.UnappliedAmountMinor > 0 {
		legs = append(legs, seedJournalLeg{
			AccountID:  refs.accounts[ledgerAccountUnappliedCash],
			Credit:     payment.UnappliedAmountMinor,
			CustomerID: cus.ID,
		})
	}
	event := accountingcontrol.JournalSourceEventCustomerPaymentPosted.String()
	journal, err := postSeedJournal(ctx, db, &seedJournalRequest{
		orgID:          refs.orgID,
		buID:           refs.buID,
		actorID:        refs.adminID,
		accountingDate: payment.AccountingDate,
		batchNumber:    "JB-" + spec.reference,
		entryNumber:    "JE-" + spec.reference,
		description:    description,
		sourceType:     "CustomerPayment",
		sourceID:       payment.ID.String(),
		documentNumber: spec.reference,
		event:          event,
		idempotencyKey: "customer-payment-posted:" + payment.ID.String(),
		legs:           legs,
	})
	if err != nil {
		return err
	}
	payment.PostedBatchID = journal.batchID

	if err = customerpaymentdb.Payments.Create(ctx, db, payment); err != nil {
		return fmt.Errorf("insert customer payment: %w", err)
	}
	ledger := make([]*customerledger.CustomerLedgerEntry, 0, len(payment.Applications))
	for _, app := range payment.Applications {
		if err = customerpaymentdb.Applications.Create(ctx, db, app); err != nil {
			return fmt.Errorf("insert customer payment application: %w", err)
		}
		ledger = append(ledger, &customerledger.CustomerLedgerEntry{
			CustomerID:       cus.ID,
			SourceObjectType: "CustomerPayment",
			SourceObjectID:   payment.ID.String(),
			SourceEventType:  event,
			RelatedInvoiceID: app.InvoiceID,
			DocumentNumber:   spec.reference,
			TransactionDate:  payment.AccountingDate,
			LineNumber:       len(ledger) + 1,
			AmountMinor:      -app.AppliedAmountMinor,
		})
	}
	if err = s.appendLedger(ctx, db, refs, ledger...); err != nil {
		return err
	}

	return s.saveSettlement(ctx, db, refs, settled...)
}

func (s *ReceivablesSeed) saveSettlement(
	ctx context.Context,
	db *keel.DB,
	refs *receivableSeedRefs,
	invoices ...*invoice.Invoice,
) error {
	inv := invoicedb.Invoices
	for _, entity := range invoices {
		if _, err := inv.AllTenants().Where(
			inv.ID.Eq(entity.ID),
			inv.OrganizationID.Eq(refs.orgID),
			inv.BusinessUnitID.Eq(refs.buID),
		).Update(
			ctx,
			db,
			inv.AppliedAmount.To(entity.AppliedAmount),
			inv.AppliedAmountMinor.To(entity.AppliedAmountMinor),
			inv.SettlementStatus.To(entity.SettlementStatus),
		); err != nil {
			return fmt.Errorf("settle invoice %s: %w", entity.Number, err)
		}
	}

	return nil
}
