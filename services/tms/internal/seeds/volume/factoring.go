package volume

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/factoringremit"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	facDay    = int64(86_400)
	facHour   = int64(3_600)
	facMinute = int64(60)

	facAccountDueFrom   = "1150"
	facAccountLiability = "2420"
	facAccountFee       = "6840"

	facSourceObject = "FactoringSubmissionItem"
	facNoFile       = "Created by the volume seed; no file was stored for this document"
)

// facPeriod is a span in which a customer's invoices were assigned to one
// factor: billing stamps an invoice with the factor whose period covers the
// moment it posted, as invoicepostingservice.assignFactoring does.
type facPeriod struct {
	factor *facFactor
	from   int64
	to     int64
}

func (p *facPeriod) covers(at int64) bool {
	return at >= p.from && (p.to == 0 || at < p.to)
}

type facStyle struct {
	headers    []string
	ext        string
	purchase   string
	release    string
	decline    string
	chargeback string
	other      string
}

type facFactor struct {
	company   *factoringcompany.FactoringCompany
	agreement *factoring.Agreement
	transport *factoringsubmission.Transport
	noas      map[pulid.ID]*factoring.Noa
	style     facStyle
	slug      string
	profiled  bool
}

type facState struct {
	factors []*facFactor
	byID    map[pulid.ID]*facFactor
}

type facProfile struct {
	name        string
	city        string
	state       string
	advance     int
	reserve     int
	minimumFee  int64
	recourse    factoring.Recourse
	treatment   factoring.AccountingTreatment
	fundingDays int
	style       facStyle
}

var facStyles = []facStyle{
	{
		headers: []string{
			"Invoice Number", "Schedule", "Debtor", "Type", "Face Amount",
			"Advance", "Fee", "Reserve", "Net Paid", "Date",
		},
		ext: ".csv", purchase: "Purchase", release: "Reserve Release",
		decline: "Declined", chargeback: "Chargeback", other: "Wire Fee",
	},
	{
		headers: []string{
			"Inv No", "Batch Number", "Customer Name", "Transaction Type", "Gross Amount",
			"Advance Amount", "Discount Fee", "Reserve Amount", "Amount Paid", "Transaction Date",
		},
		ext: ".xlsx", purchase: "Funded", release: "Rebate",
		decline: "Non-Purchase", chargeback: "Recourse", other: "ACH Fee",
	},
	{
		headers: []string{
			"Client Invoice", "Reference", "Account Debtor", "Activity", "Purchased Amount",
			"Amount Advanced", "Factor Fee", "Reserve Held", "Disbursed", "Funding Date",
		},
		ext: ".csv", purchase: "Advance", release: "Escrow Release",
		decline: "Rejected", chargeback: "Repurchase", other: "Wire Fee",
	},
}

var facCatalog = []facProfile{
	{
		name: "Lonestar Freight Funding", city: "Dallas", state: "TX",
		advance: 9000, reserve: 700, minimumFee: 1_500,
		recourse: factoring.RecourseFull, treatment: factoring.AccountingTreatmentSale,
		fundingDays: 1, style: facStyles[0],
	},
	{
		name: "Keystone Transport Capital", city: "Harrisburg", state: "PA",
		advance: 8500, reserve: 1000, minimumFee: 2_500,
		recourse: factoring.RecourseNone, treatment: factoring.AccountingTreatmentSale,
		fundingDays: 2, style: facStyles[1],
	},
	{
		name: "Bluewater Receivables Finance", city: "Jacksonville", state: "FL",
		advance: 9500, reserve: 300, minimumFee: 1_000,
		recourse: factoring.RecourseFull, treatment: factoring.AccountingTreatmentSecuredBorrowing,
		fundingDays: 1, style: facStyles[2],
	},
}

var facDeclineReasons = []string{
	"Debtor over credit limit",
	"Debtor not approved",
	"Duplicate of an invoice already purchased",
	"Proof of delivery illegible",
	"Invoice older than purchase terms",
}

var facExcludeReasons = []string{
	"Customer pays this lane directly under a separate PO",
	"Spot load billed outside the factoring agreement",
	"Customer asked to pay this invoice to us directly",
}

var facFailures = []string{
	"550 5.7.1 Message rejected: attachment size exceeds the recipient's limit",
	"421 4.7.0 Temporary server error at the factor's mail gateway; retries exhausted",
	"Connection to the factor's mail server timed out",
}

// facMasterData sets up the factors, their agreements, delivery settings and
// notices of assignment, and puts a minority of customers on factoring.
func (g *generator) facMasterData(ctx context.Context, tx *keel.DB) error {
	accounts, err := g.facAccounts(ctx, tx)
	if err != nil {
		return err
	}
	factors, err := g.facFactors(ctx, tx, accounts)
	if err != nil {
		return err
	}
	if len(factors) == 0 {
		return nil
	}
	st := &facState{factors: factors, byID: make(map[pulid.ID]*facFactor, len(factors))}
	for _, f := range factors {
		st.byID[f.company.ID] = f
	}
	g.factoring = st

	return g.facAssign(ctx, tx, st)
}

type facAccountDef struct {
	code        string
	name        string
	description string
	parent      string
	category    accounttype.Category
}

var facAccountDefs = []facAccountDef{
	{
		code: facAccountDueFrom, name: "Due From Factor", parent: "1100",
		description: "Reserves a factor holds until the customer pays",
		category:    accounttype.CategoryAsset,
	},
	{
		code: facAccountLiability, name: "Factoring Advances Payable", parent: "2400",
		description: "Advances received under secured-borrowing factoring",
		category:    accounttype.CategoryLiability,
	},
	{
		code: facAccountFee, name: "Factoring Fees", parent: "6800",
		description: "Discount and service fees charged by factors",
		category:    accounttype.CategoryExpense,
	},
}

func (g *generator) facAccounts(ctx context.Context, tx *keel.DB) (map[string]pulid.ID, error) {
	codes := []string{"1100", "2400", "6800"}
	for _, def := range facAccountDefs {
		codes = append(codes, def.code)
	}
	byCode, err := g.glAccountsByCode(ctx, codes...)
	if err != nil {
		return nil, err
	}

	at := accounttypedb.AccountTypes
	types, err := at.Pick(at.ID, at.Category).Order(at.ID.Asc()).All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load account types: %w", err)
	}
	typeOf := make(map[accounttype.Category]pulid.ID, len(types))
	for _, t := range types {
		if _, ok := typeOf[t.Category]; !ok {
			typeOf[t.Category] = t.ID
		}
	}

	created := make([]*glaccount.GLAccount, 0, len(facAccountDefs))
	for _, def := range facAccountDefs {
		if _, ok := byCode[def.code]; ok {
			continue
		}
		typeID, ok := typeOf[def.category]
		if !ok {
			continue
		}
		account := &glaccount.GLAccount{
			ID:             pulid.MustNew("gla_"),
			BusinessUnitID: g.refs.buID,
			OrganizationID: g.refs.orgID,
			Status:         domaintypes.StatusActive,
			AccountTypeID:  typeID,
			ParentID:       byCode[def.parent],
			AccountCode:    def.code,
			Name:           def.name,
			Description:    def.description,
			AllowManualJE:  true,
		}
		created = append(created, account)
		byCode[def.code] = account.ID
	}
	if len(created) > 0 {
		if err := insertRows(ctx, tx, glaccountdb.GLAccounts.Handle, created); err != nil {
			return nil, fmt.Errorf("insert factoring gl accounts: %w", err)
		}
		g.report.add("GL accounts", len(created))
	}

	return byCode, nil
}

func facCity(name, state string) *cityRecord {
	for i := range cities {
		if cities[i].Name == name && cities[i].State == state {
			return &cities[i]
		}
	}
	return &cities[0]
}

//nolint:funlen // reuse and creation of the factor catalogue read best together
func (g *generator) facFactors(
	ctx context.Context,
	tx *keel.DB,
	accounts map[string]pulid.ID,
) ([]*facFactor, error) {
	companies, err := factoringcompanydb.FactoringCompanies.Query().All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load factoring companies: %w", err)
	}
	fa := factoringdb.Agreements
	agreements, err := fa.Load(fa.FeeTiers()).
		Where(fa.Status.Eq(factoring.AgreementStatusActive)).
		All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load factoring agreements: %w", err)
	}
	transports, err := factoringdb.Transports.Query().All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load factoring transports: %w", err)
	}
	rp := factoringdb.RemittanceProfiles
	profiles, err := rp.Pick(rp.FactoringCompanyID).All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load remittance profiles: %w", err)
	}
	profiled := make([]pulid.ID, 0, len(profiles))
	for _, p := range profiles {
		profiled = append(profiled, p.FactoringCompanyID)
	}

	start, now := unix(g.start), unix(g.now)
	codes := make(map[string]struct{}, len(companies))
	names := make(map[string]struct{}, len(companies))
	for _, c := range companies {
		codes[strings.ToLower(c.Code)] = struct{}{}
		names[strings.ToLower(c.Name)] = struct{}{}
	}

	factors := make([]*facFactor, 0, len(facCatalog))
	for _, c := range companies {
		if !c.IsActive() || len(factors) == len(facCatalog) {
			continue
		}
		var agreement *factoring.Agreement
		for _, a := range agreements {
			if a.FactoringCompanyID == c.ID && a.InForce(start) && a.InForce(now) &&
				len(a.FeeTiers) > 0 {
				agreement = a
			}
		}
		var transport *factoringsubmission.Transport
		for _, t := range transports {
			if t.FactoringCompanyID == c.ID {
				transport = t
			}
		}
		if agreement == nil || transport == nil {
			continue
		}
		factors = append(factors, &facFactor{
			company:   c,
			agreement: agreement,
			transport: transport,
			noas:      make(map[pulid.ID]*factoring.Noa),
			style:     facStyles[len(factors)%len(facStyles)],
			slug:      mdSlug(c.Name),
			profiled:  slices.Contains(profiled, c.ID),
		})
	}

	var newCompanies []*factoringcompany.FactoringCompany
	var newAgreements []*factoring.Agreement
	var newTiers []*factoring.FeeTier
	var newTransports []*factoringsubmission.Transport
	var companyStamps, agreementStamps, transportStamps []stamp
	for _, p := range facCatalog {
		if len(factors) == len(facCatalog) {
			break
		}
		if _, taken := names[strings.ToLower(p.name)]; taken {
			continue
		}
		created := start - g.fake.between64(420, 900)*facDay
		city := facCity(p.city, p.state)
		slug := mdSlug(p.name)
		company := &factoringcompany.FactoringCompany{
			ID:             idAt("fac_", created),
			BusinessUnitID: g.refs.buID,
			OrganizationID: g.refs.orgID,
			Status:         domaintypes.StatusActive,
			Code:           uniqueCode(codeFrom(p.name, 4), codes),
			Name:           p.name,
			AddressLine1:   g.fake.street(),
			City:           city.Name,
			StateID:        g.refs.stateIDs[city.State],
			PostalCode:     city.Zip,
			Email:          "funding@" + slug + ".example.com",
			PhoneNumber:    g.fake.phone(),
			RemittanceInstructions: fmt.Sprintf(
				"Remit payment to %s, PO Box %d, %s, %s %s. ACH routing %s, account ending %s. "+
					"Reference the invoice number on every payment.",
				p.name, g.fake.between(1000, 99999), city.Name, city.State, city.Zip,
				g.fake.digits(9), g.fake.digits(4),
			),
			NoticeOfAssignment: fmt.Sprintf(
				"The accounts receivable of %s have been sold and assigned to %s. "+
					"Payment of this invoice must be made only to %s at the remittance address shown.",
				g.refs.orgName, p.name, p.name,
			),
			NonRecourse: p.recourse == factoring.RecourseNone,
		}
		effective := created + g.fake.between64(10, 40)*facDay
		agreement := &factoring.Agreement{
			ID:                     idAt("facagr_", effective),
			OrganizationID:         g.refs.orgID,
			BusinessUnitID:         g.refs.buID,
			FactoringCompanyID:     company.ID,
			Status:                 factoring.AgreementStatusActive,
			EffectiveFrom:          effective,
			AdvanceRateBP:          p.advance,
			ReserveRateBP:          p.reserve,
			MinimumFeeMinor:        p.minimumFee,
			ChargebackWindowDays:   factoring.DefaultChargebackWindow,
			Recourse:               p.recourse,
			AccountingTreatment:    p.treatment,
			FundingDays:            p.fundingDays,
			SubmissionTime:         600,
			Notes:                  "Signed master purchase agreement; schedules sent daily by 10:00.",
			DueFromFactorAccountID: accounts[facAccountDueFrom],
			FeeExpenseAccountID:    accounts[facAccountFee],
			LiabilityAccountID:     accounts[facAccountLiability],
			ActivatedAt:            ptr(effective),
		}
		for _, tier := range []struct {
			from int
			to   *int
			rate int
		}{
			{from: 0, to: ptr(30), rate: 250},
			{from: 30, to: ptr(45), rate: 325},
			{from: 45, to: ptr(75), rate: 425},
			{from: 75, rate: 600},
		} {
			fee := &factoring.FeeTier{
				ID:             pulid.MustNew("facfee_"),
				OrganizationID: g.refs.orgID,
				BusinessUnitID: g.refs.buID,
				AgreementID:    agreement.ID,
				DaysFrom:       tier.from,
				DaysTo:         tier.to,
				RateBP:         tier.rate,
			}
			agreement.FeeTiers = append(agreement.FeeTiers, fee)
			newTiers = append(newTiers, fee)
		}
		transport := &factoringsubmission.Transport{
			ID:                 idAt("factrn_", effective),
			OrganizationID:     g.refs.orgID,
			BusinessUnitID:     g.refs.buID,
			FactoringCompanyID: company.ID,
			Method:             factoringsubmission.TransportMethodEmail,
			EmailRecipients:    []string{"schedules@" + slug + ".example.com"},
			AttachPackets:      true,
			LastTestedAt:       ptr(effective + facHour),
		}

		newCompanies = append(newCompanies, company)
		newAgreements = append(newAgreements, agreement)
		newTransports = append(newTransports, transport)
		companyStamps = append(
			companyStamps,
			stamp{id: company.ID, created: created, updated: created},
		)
		agreementStamps = append(agreementStamps, stamp{
			id: agreement.ID, created: created + facDay, updated: effective,
		})
		transportStamps = append(transportStamps, stamp{
			id: transport.ID, created: effective, updated: effective + facHour,
		})
		factors = append(factors, &facFactor{
			company:   company,
			agreement: agreement,
			transport: transport,
			noas:      make(map[pulid.ID]*factoring.Noa),
			style:     p.style,
			slug:      slug,
		})
	}

	steps := []struct {
		table string
		rows  int
		run   func() error
		marks []stamp
	}{
		{"factoring_companies", len(newCompanies), func() error {
			return insertRows(ctx, tx, factoringcompanydb.FactoringCompanies.Handle, newCompanies)
		}, companyStamps},
		{"factoring_agreements", len(newAgreements), func() error {
			return insertRows(ctx, tx, factoringdb.Agreements.Handle, newAgreements)
		}, agreementStamps},
		{"factoring_fee_tiers", len(newTiers), func() error {
			return insertRows(ctx, tx, factoringdb.FeeTiers.Handle, newTiers)
		}, nil},
		{"factoring_transports", len(newTransports), func() error {
			return insertRows(ctx, tx, factoringdb.Transports.Handle, newTransports)
		}, transportStamps},
	}
	for _, step := range steps {
		if step.rows == 0 {
			continue
		}
		if err := step.run(); err != nil {
			return nil, fmt.Errorf("insert %s: %w", step.table, err)
		}
		if step.marks != nil {
			if err := backdateTable(ctx, tx, step.table, step.marks); err != nil {
				return nil, err
			}
		}
	}
	g.report.add("Factoring companies", len(newCompanies))
	g.report.add("Factoring agreements", len(newAgreements))

	return factors, nil
}

type facPlanKind int

const (
	facPlanFull facPlanKind = iota
	facPlanMidway
	facPlanSwitched
	facPlanDraftNotice
	facPlanSentNotice
	facPlanExpiring
)

//nolint:gocognit,funlen // each customer's factoring history is decided in one place
func (g *generator) facAssign(ctx context.Context, tx *keel.DB, st *facState) error {
	candidates := make([]*customerRecord, 0, len(g.customers))
	weights := make([]int, 0, len(g.customers))
	for i, c := range g.customers {
		if i < 2 || c.Weight <= 1 {
			continue
		}
		candidates = append(candidates, c)
		weights = append(weights, c.Weight)
	}
	if len(candidates) == 0 {
		return nil
	}
	n := clamp(len(g.customers)*8/100, min(3, len(candidates)), min(60, len(candidates)))

	chosen := make([]*customerRecord, 0, n)
	used := make(map[pulid.ID]bool, n)
	for tries := 0; len(chosen) < n && tries < n*20; tries++ {
		c := candidates[g.fake.weighted(weights)]
		if used[c.ID] {
			continue
		}
		used[c.ID] = true
		chosen = append(chosen, c)
	}

	start, now := unix(g.start), unix(g.now)
	span := now - start
	var noas []*factoring.Noa
	var docs []*document.Document
	var noaStamps []stamp
	profileCustomers := make([]string, 0, len(chosen))
	profileFactors := make([]string, 0, len(chosen))

	for i, c := range chosen {
		kind := facPlanFull
		switch {
		case i == 0 && len(chosen) >= 5 && len(st.factors) > 1:
			kind = facPlanSwitched
		case i == 1 && len(chosen) >= 5:
			kind = facPlanDraftNotice
		case i == 2 && len(chosen) >= 5:
			kind = facPlanSentNotice
		case i == 3 && len(chosen) >= 6:
			kind = facPlanExpiring
		case g.fake.chance(0.25):
			kind = facPlanMidway
		}

		factor := st.factors[i%len(st.factors)]
		switch kind {
		case facPlanSwitched:
			previous := st.factors[0]
			factor = st.factors[1]
			switchAt := start + int64(float64(span)*(0.35+g.fake.float()*0.2))
			first := &facPeriod{
				factor: previous,
				from:   start - g.fake.between64(60, 300)*facDay,
				to:     switchAt,
			}
			second := &facPeriod{factor: factor, from: switchAt}
			c.Factoring = []*facPeriod{first, second}

			old := g.facNoa(previous, c, factoring.NoaStatusAcknowledged, first.from, &docs)
			released := switchAt - g.fake.between64(1, 3)*facDay
			old.Status = factoring.NoaStatusReleased
			old.ReleasedAt = ptr(released)
			old.ReleaseReason = "Release letter from " + previous.company.Name +
				" received; the customer moved to " + factor.company.Name
			noas = append(noas, old)
			noaStamps = append(
				noaStamps,
				stamp{id: old.ID, created: old.CreatedAt, updated: released},
			)
			previous.noas[c.ID] = old

			next := g.facNoa(factor, c, factoring.NoaStatusAcknowledged, switchAt, &docs)
			noas = append(noas, next)
			noaStamps = append(
				noaStamps,
				stamp{id: next.ID, created: next.CreatedAt, updated: next.UpdatedAt},
			)
			factor.noas[c.ID] = next
		default:
			from := start - g.fake.between64(30, 300)*facDay
			status := factoring.NoaStatusAcknowledged
			switch kind {
			case facPlanMidway:
				from = start + int64(float64(span)*(0.2+g.fake.float()*0.5))
			case facPlanDraftNotice:
				from = now - g.fake.between64(1, 4)*facDay
				status = factoring.NoaStatusDraft
			case facPlanSentNotice:
				from = now - g.fake.between64(6, 18)*facDay
				status = factoring.NoaStatusSent
			case facPlanFull, facPlanExpiring, facPlanSwitched:
			}
			c.Factoring = []*facPeriod{{factor: factor, from: from}}
			noa := g.facNoa(factor, c, status, from, &docs)
			if kind == facPlanExpiring && noa.AcknowledgedAt != nil {
				noa.ExpiresAt = ptr(now + g.fake.between64(8, 25)*facDay)
				noa.ExpiryWarnedAt = ptr(now - g.fake.between64(1, 4)*facDay)
				noa.UpdatedAt = *noa.ExpiryWarnedAt
			}
			noas = append(noas, noa)
			noaStamps = append(
				noaStamps,
				stamp{id: noa.ID, created: noa.CreatedAt, updated: noa.UpdatedAt},
			)
			factor.noas[c.ID] = noa
		}

		current := c.Factoring[len(c.Factoring)-1].factor
		profileCustomers = append(profileCustomers, c.ID.String())
		profileFactors = append(profileFactors, current.company.ID.String())
	}

	if len(docs) > 0 {
		docStamps := make([]stamp, 0, len(docs))
		for _, d := range docs {
			docStamps = append(
				docStamps,
				stamp{id: d.ID, created: d.CreatedAt, updated: d.CreatedAt},
			)
		}
		if err := insertRows(ctx, tx, documentdb.Documents.Handle, docs); err != nil {
			return fmt.Errorf("insert signed notice documents: %w", err)
		}
		if err := backdate(ctx, tx, documentdb.Documents.Handle, docStamps); err != nil {
			return err
		}
	}
	if err := insertRows(ctx, tx, factoringdb.Noas.Handle, noas); err != nil {
		return fmt.Errorf("insert notices of assignment: %w", err)
	}
	if err := backdate(ctx, tx, factoringdb.Noas.Handle, noaStamps); err != nil {
		return err
	}
	cbp := customerdb.CustomerBillingProfiles
	assigned := keel.Unnest(
		"d",
		[]string{"customer_id", "factoring_company_id"},
		profileCustomers,
		profileFactors,
	)
	if _, err := cbp.With(assigned).Join(assigned, assigned.Col("customer_id").EqCol(cbp.CustomerID)).
		Where(keel.True()).
		Update(ctx, tx,
			cbp.FactoringCompanyID.ToExpr(assigned.Col("factoring_company_id")),
			keepUpdated(cbp.Handle),
		); err != nil {
		return fmt.Errorf("assign factoring companies to billing profiles: %w", err)
	}

	g.report.add("Factored customers", len(chosen))
	g.report.add("Notices of assignment", len(noas))
	g.report.add("Signed notice documents", len(docs))
	return nil
}

// facNoa drafts a notice for the customer's factoring period starting at from
// and walks it to status: sent before the period, and for an acknowledged
// notice signed and returned with the signed copy attached, as the app
// requires before it records an acknowledgement.
func (g *generator) facNoa(
	f *facFactor,
	c *customerRecord,
	status factoring.NoaStatus,
	from int64,
	docs *[]*document.Document,
) *factoring.Noa {
	now := unix(g.now)
	created := min(from-g.fake.between64(12, 25)*facDay, now-2*facHour)
	if status == factoring.NoaStatusDraft {
		created = min(from+g.fake.between64(2, 20)*facHour, now-facHour)
	}
	noa := &factoring.Noa{
		ID:                 idAt("facnoa_", created),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		CustomerID:         c.ID,
		FactoringCompanyID: f.company.ID,
		Status:             factoring.NoaStatusDraft,
		CreatedAt:          created,
		UpdatedAt:          created,
	}
	if status == factoring.NoaStatusDraft {
		return noa
	}

	sentAt := min(created+g.fake.between64(2*facHour, 3*facDay), now-facHour)
	recipient := c.BillingEmail
	if recipient == "" {
		recipient = "accounts.payable@" + mdSlug(c.Name) + ".example.com"
	}
	noa.Status = factoring.NoaStatusSent
	noa.SentAt = ptr(sentAt)
	noa.SentTo = recipient
	noa.UpdatedAt = sentAt
	if status == factoring.NoaStatusSent {
		return noa
	}

	acknowledged := min(sentAt+g.fake.between64(2, 9)*facDay, from-facHour)
	acknowledged = max(acknowledged, sentAt+facHour)
	uploader := g.bilPick(g.refs.userIDs)
	fileName := fmt.Sprintf("NOA-%s-%s-signed.pdf", c.Code, f.company.Code)
	doc := g.facDocument(factoring.NoaDocumentResourceType, noa.ID, fileName, "application/pdf",
		document.ProcessingProfileNone, acknowledged+facHour, uploader,
		g.fake.between64(90_000, 420_000))
	doc.Description = "Signed notice of assignment returned by " + c.Name
	*docs = append(*docs, doc)

	noa.Status = factoring.NoaStatusAcknowledged
	noa.AcknowledgedAt = ptr(acknowledged)
	noa.SignedDocumentID = doc.ID
	noa.UpdatedAt = acknowledged + facHour
	if g.fake.chance(0.3) {
		noa.ExpiresAt = ptr(acknowledged + g.fake.between64(540, 900)*facDay)
	}
	return noa
}

// facDocument is the metadata row of a file the app would have stored. The
// seed stores no file, so the row says so instead of pretending a preview
// or extracted text exists.
func (g *generator) facDocument(
	resourceType string,
	resourceID pulid.ID,
	fileName, fileType string,
	profile document.ProcessingProfile,
	at int64,
	by pulid.ID,
	size int64,
) *document.Document {
	id := idAt("doc_", at)
	storagePath := fileutils.GenerateStoragePath(g.refs.orgID.String(), resourceType, fileName)
	return &document.Document{
		ID:                  id,
		OrganizationID:      g.refs.orgID,
		BusinessUnitID:      g.refs.buID,
		LineageID:           id,
		VersionNumber:       1,
		IsCurrentVersion:    true,
		FileName:            path.Base(storagePath),
		OriginalName:        fileName,
		FileSize:            size,
		FileType:            fileType,
		StoragePath:         storagePath,
		Status:              document.StatusActive,
		ResourceID:          resourceID.String(),
		ResourceType:        resourceType,
		ProcessingProfile:   profile,
		UploadedByID:        by,
		PreviewStatus:       document.PreviewStatusUnsupported,
		ContentStatus:       document.ContentStatusFailed,
		ContentError:        facNoFile,
		CreatedAt:           at,
		UpdatedAt:           at,
		CryptoMode:          "blob_v1",
		CryptoVersion:       2,
		ShipmentDraftStatus: document.ShipmentDraftStatusUnavailable,
		SignatureStatus:     document.SignatureStatusUnknown,
	}
}

// facStamp records on a posting invoice the factor its customer was assigned
// to at that moment and the agreement then in force, unless the invoice is
// kept out of factoring.
func (g *generator) facStamp(inv *invoice.Invoice, cus *customerRecord) {
	if cus == nil || inv.PostedAt == nil {
		return
	}
	at := *inv.PostedAt
	for _, p := range cus.Factoring {
		if !p.covers(at) {
			continue
		}
		if g.fake.chance(0.025) {
			inv.FactoringOverride = invoice.FactoringOverrideExclude
			inv.FactoringOverrideReason = pick(g.fake, facExcludeReasons)
			return
		}
		factorID := p.factor.company.ID
		inv.FactoringCompanyID = &factorID
		if p.factor.agreement.InForce(at) {
			agreementID := p.factor.agreement.ID
			inv.FactoringAgreementID = &agreementID
		}
		return
	}
}

// -- Factoring phase --

type facInvoice struct {
	rec        *shipmentRecord
	inv        *invoice.Invoice
	factor     *facFactor
	readyAt    int64
	face       int64
	items      []*facItem
	rejectedAt int64
	updatedAt  int64
	touched    bool
}

type facItem struct {
	fi           *facInvoice
	sub          *facSub
	item         *factoringsubmission.SubmissionItem
	fundOn       int64
	payOn        int64
	chargebackOn int64
	reversalOn   int64
	decline      string
	extraFee     int64
	chargeback   *factoringledger.Movement
	chargebackID pulid.ID
}

type facSub struct {
	factor *facFactor
	sub    *factoringsubmission.Submission
	items  []*facItem
	slot   int64
	keys   []string
}

type facEvent struct {
	at   int64
	kind factoringremittance.RemittanceLineKind
	item *facItem
	cash int64
}

type facAppliedLine struct {
	line    *factoringremittance.RemittanceLine
	entryID pulid.ID
}

type facRun struct {
	setup       *bilSetup
	rows        *bilRows
	accounts    map[pulid.ID]factoring.PostingAccounts
	bookable    map[pulid.ID]bool
	invoices    []*facInvoice
	subs        []*facSub
	entries     []*factoringledger.ReserveEntry
	remittances []*factoringremittance.Remittance
	lines       []*factoringremittance.RemittanceLine
	applied     []facAppliedLine
	profiles    []*factoringremittance.RemittanceProfile
	docs        []*document.Document
	states      []*factoringreadiness.InvoiceState
	remStamps   []stamp
	lineStamps  []stamp
	excluded    int
}

//nolint:gocognit,funlen // the phase is a sequence of steps over one timeline
func (g *generator) seedFactoring(ctx context.Context) error {
	st := g.factoring
	if st == nil || len(st.factors) == 0 {
		return nil
	}
	setup, err := g.bilLoadSetup(ctx)
	if err != nil {
		return err
	}
	run := &facRun{
		setup:    setup,
		rows:     &bilRows{balances: make(map[bilBalanceKey]*bilBalance)},
		accounts: make(map[pulid.ID]factoring.PostingAccounts, len(st.factors)),
		bookable: make(map[pulid.ID]bool, len(st.factors)),
	}
	for _, f := range st.factors {
		accounts := f.agreement.PostingAccounts(setup.ar, setup.cash)
		run.accounts[f.company.ID] = accounts
		run.bookable[f.company.ID] = len(setup.periods) > 0 &&
			len(accounts.Missing(f.agreement.AccountingTreatment)) == 0
		if !run.bookable[f.company.ID] {
			g.opts.Logf("  ! %s: no open period or posting accounts; funding is not booked",
				f.company.Name)
		}
	}

	g.facCollect(st, run)
	if len(run.invoices) == 0 {
		return nil
	}
	if err = g.facSubmit(ctx, run); err != nil {
		return err
	}
	g.facPlanEvents(run)
	if err = g.facRemit(run); err != nil {
		return err
	}
	if err = g.facExclude(run); err != nil {
		return err
	}
	g.facStates(run)

	if len(run.rows.journals) > 0 {
		if err = g.bilMaterialize(ctx, run.rows); err != nil {
			return err
		}
	}
	if err = g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		return g.facWrite(ctx, tx, run)
	}); err != nil {
		return err
	}
	g.facCount(run)

	return nil
}

func (g *generator) facCollect(st *facState, run *facRun) {
	now := unix(g.now)
	for _, r := range g.shipments {
		inv := r.Invoice
		if inv == nil || inv.Status == invoice.StatusDraft || inv.IsCreditMemo() {
			continue
		}
		var factor *facFactor
		if inv.FactoringCompanyID != nil {
			factor = st.byID[*inv.FactoringCompanyID]
		}
		if factor == nil && !inv.FactoringOverride.Excludes() {
			continue
		}
		fi := &facInvoice{rec: r, inv: inv, factor: factor, face: inv.OpenBalanceMinor()}
		run.invoices = append(run.invoices, fi)
		if factor == nil || inv.Status != invoice.StatusPosted || fi.face <= 0 ||
			inv.DisputeStatus == invoice.DisputeStatusDisputed {
			continue
		}
		noa := factor.noas[inv.CustomerID]
		if noa == nil || noa.AcknowledgedAt == nil {
			continue
		}
		ready := max(*inv.PostedAt+g.fake.between64(5*facMinute, 2*facHour), *noa.AcknowledgedAt)
		if !factor.agreement.InForce(ready) ||
			(noa.ExpiresAt != nil && ready >= *noa.ExpiresAt) ||
			(noa.ReleasedAt != nil && ready >= *noa.ReleasedAt) || ready > now {
			continue
		}
		fi.readyAt = ready
	}
}

// facSlot is the next time the organization sends schedules: the agreement's
// submission time on the next business day at or after at.
func facSlot(at int64, minuteOfDay int) int64 {
	t := time.Unix(at, 0).UTC()
	slot := time.Date(t.Year(), t.Month(), t.Day(), 0, minuteOfDay, 0, 0, time.UTC)
	if !slot.After(t) {
		slot = slot.AddDate(0, 0, 1)
	}
	for slot.Weekday() == time.Saturday || slot.Weekday() == time.Sunday {
		slot = slot.AddDate(0, 0, 1)
	}
	return slot.Unix()
}

func facBusinessDays(at int64, days int) int64 {
	t := time.Unix(at, 0).UTC()
	for days > 0 {
		t = t.AddDate(0, 0, 1)
		if t.Weekday() != time.Saturday && t.Weekday() != time.Sunday {
			days--
		}
	}
	return t.Unix()
}

func facWeekStart(at int64) int64 {
	t := time.Unix(at, 0).UTC()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset).Unix()
}

//nolint:gocognit,funlen // scheduling, delivery and resubmission share one timeline
func (g *generator) facSubmit(ctx context.Context, run *facRun) error {
	now := unix(g.now)
	type slotKey struct {
		factor pulid.ID
		slot   int64
	}
	groups := make(map[slotKey][]*facInvoice)
	for _, fi := range run.invoices {
		if fi.readyAt == 0 {
			continue
		}
		slot := facSlot(fi.readyAt, fi.factor.agreement.SubmissionTime)
		switch g.fake.weighted([]int{70, 20, 10}) {
		case 1:
			slot = facSlot(slot, fi.factor.agreement.SubmissionTime)
		case 2:
			slot = facSlot(
				facBusinessDays(slot, g.fake.between(1, 2)),
				fi.factor.agreement.SubmissionTime,
			)
		}
		noa := fi.factor.noas[fi.inv.CustomerID]
		if slot+10*facMinute > now || (noa.ReleasedAt != nil && slot >= *noa.ReleasedAt) {
			continue
		}
		key := slotKey{factor: fi.factor.company.ID, slot: slot}
		groups[key] = append(groups[key], fi)
	}

	keys := make([]slotKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].slot != keys[j].slot {
			return keys[i].slot < keys[j].slot
		}
		return keys[i].factor < keys[j].factor
	})

	byFactor := make(map[pulid.ID]*facFactor, len(run.bookable))
	for _, fi := range run.invoices {
		if fi.factor != nil {
			byFactor[fi.factor.company.ID] = fi.factor
		}
	}
	var resubmit []*facInvoice
	var resubmitAt []int64
	for _, key := range keys {
		members := groups[key]
		for begin := 0; begin < len(members); begin += factoringsubmission.MaxSubmissionInvoices {
			batch := members[begin:min(begin+factoringsubmission.MaxSubmissionInvoices, len(members))]
			fs := g.facNewSub(byFactor[key.factor], key.slot, batch)
			run.subs = append(run.subs, fs)
			if cancelled := g.facDeliver(fs, now); cancelled > 0 {
				for _, it := range fs.items {
					resubmit = append(resubmit, it.fi)
					resubmitAt = append(resubmitAt, cancelled)
				}
			}
		}
	}

	type retryKey struct {
		factor pulid.ID
		slot   int64
	}
	retries := make(map[retryKey][]*facInvoice)
	for i, fi := range resubmit {
		// Readiness clears ready_since while an invoice is on a schedule and
		// sets it afresh on release, so a resubmission keys on the release time.
		fi.readyAt = resubmitAt[i]
		slot := facSlot(resubmitAt[i], fi.factor.agreement.SubmissionTime)
		if slot+10*facMinute > now {
			continue
		}
		key := retryKey{factor: fi.factor.company.ID, slot: slot}
		retries[key] = append(retries[key], fi)
	}
	retryKeys := make([]retryKey, 0, len(retries))
	for key := range retries {
		retryKeys = append(retryKeys, key)
	}
	sort.Slice(retryKeys, func(i, j int) bool {
		if retryKeys[i].slot != retryKeys[j].slot {
			return retryKeys[i].slot < retryKeys[j].slot
		}
		return retryKeys[i].factor < retryKeys[j].factor
	})
	for _, key := range retryKeys {
		members := retries[key]
		fs := g.facNewSub(byFactor[key.factor], key.slot, members)
		fs.sub.Status = factoringsubmission.SubmissionStatusSent
		fs.sub.Attempts = 1
		fs.sub.SentAt = ptr(key.slot + g.fake.between64(facMinute, 8*facMinute))
		fs.sub.TransportReference = g.facMessageID(fs.factor)
		run.subs = append(run.subs, fs)
	}
	sort.SliceStable(run.subs, func(i, j int) bool { return run.subs[i].slot < run.subs[j].slot })

	slots := make([]numberSlot, len(run.subs))
	for i, fs := range run.subs {
		slots[i] = numberSlot{At: fs.slot}
	}
	numbers, err := g.numbers.allocate(ctx, sequenceconfig.SequenceTypeFactoringSubmission, slots)
	if err != nil {
		return err
	}
	for i, fs := range run.subs {
		fs.sub.Number = numbers[i]
	}
	return nil
}

func (g *generator) facNewSub(f *facFactor, slot int64, members []*facInvoice) *facSub {
	submitter := g.bilPick(g.refs.userIDs)
	fs := &facSub{factor: f, slot: slot, keys: []string{f.company.ID.String()}}
	fs.sub = &factoringsubmission.Submission{
		ID:                 idAt("facsub_", slot),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		FactoringCompanyID: f.company.ID,
		AgreementID:        f.agreement.ID,
		Status:             factoringsubmission.SubmissionStatusSending,
		CurrencyCode:       money.DefaultCurrencyCode,
		TransportMethod:    f.transport.Method,
		TransportTarget:    f.transport.Target(),
		SendAttemptGroup:   1,
		SubmittedByID:      submitter,
		CreatedAt:          slot,
		UpdatedAt:          slot,
	}
	for i, fi := range members {
		funding := factoring.ExpectedFunding(
			fi.face,
			f.agreement,
			factoring.DaysToRun(fi.inv.DueDate, slot),
		)
		item := &factoringsubmission.SubmissionItem{
			ID:                   idAt("facsit_", slot),
			OrganizationID:       g.refs.orgID,
			BusinessUnitID:       g.refs.buID,
			SubmissionID:         fs.sub.ID,
			InvoiceID:            fi.inv.ID,
			InvoiceNumber:        fi.inv.Number,
			CustomerID:           fi.inv.CustomerID,
			Position:             i + 1,
			FaceMinor:            funding.FaceMinor,
			ExpectedAdvanceMinor: funding.AdvanceMinor,
			ExpectedFeeMinor:     funding.FeeMinor,
			ExpectedReserveMinor: funding.ReserveMinor,
			ExpectedDays:         funding.Days,
			Status:               factoringsubmission.SubmissionItemStatusPending,
			CreatedAt:            slot,
			UpdatedAt:            slot,
		}
		it := &facItem{fi: fi, sub: fs, item: item}
		fi.items = append(fi.items, it)
		fs.items = append(fs.items, it)
		fs.sub.Items = append(fs.sub.Items, item)
		fs.keys = append(fs.keys, fi.inv.ID.String()+"@"+strconv.FormatInt(fi.readyAt, 10))
	}
	fs.sub.Totalize()
	fs.sub.IdempotencyKey = facSubmissionKey(fs.keys)
	return fs
}

// facSubmissionKey is factoringsubmissionservice.submissionKey: the factor, then the
// sorted invoices with the moment each became ready.
func facSubmissionKey(parts []string) string {
	sorted := slices.Clone(parts)
	slices.Sort(sorted[1:])
	sum := sha256.Sum256([]byte(strings.Join(sorted, "|")))
	return "facsub-" + hex.EncodeToString(sum[:16])
}

// facDeliver decides how sending the schedule went and returns the time it
// was cancelled, if it was.
func (g *generator) facDeliver(fs *facSub, now int64) int64 {
	sub := fs.sub
	sentAt := fs.slot + g.fake.between64(facMinute, 8*facMinute)
	if !g.fake.chance(0.04) {
		sub.Status = factoringsubmission.SubmissionStatusSent
		sub.Attempts = 1
		sub.SentAt = ptr(sentAt)
		sub.TransportReference = g.facMessageID(fs.factor)
		sub.UpdatedAt = sentAt
		return 0
	}

	failedAt := fs.slot + g.fake.between64(2*facMinute, 20*facMinute)
	sub.Status = factoringsubmission.SubmissionStatusFailed
	sub.Attempts = 3
	sub.FailedAt = ptr(failedAt)
	sub.LastError = pick(g.fake, facFailures)
	sub.UpdatedAt = failedAt
	if now-fs.slot < 2*facDay {
		return 0
	}
	if g.fake.chance(0.75) {
		retried := min(failedAt+g.fake.between64(facHour, 20*facHour), now-facMinute)
		sub.Status = factoringsubmission.SubmissionStatusSent
		sub.SendAttemptGroup = 2
		sub.Attempts = 1
		sub.SentAt = ptr(retried + 2*facMinute)
		sub.FailedAt = nil
		sub.TransportReference = g.facMessageID(fs.factor)
		sub.UpdatedAt = retried + 2*facMinute
		return 0
	}
	cancelled := min(failedAt+g.fake.between64(2*facHour, 30*facHour), now-facMinute)
	sub.Status = factoringsubmission.SubmissionStatusCancelled
	sub.CancelledAt = ptr(cancelled)
	sub.CancelReason = "Factor asked for the invoices on a fresh schedule after the failed delivery"
	sub.UpdatedAt = cancelled
	for _, it := range fs.items {
		it.item.Status = factoringsubmission.SubmissionItemStatusReleased
		it.item.ReleasedAt = ptr(cancelled)
		it.item.UpdatedAt = cancelled
	}
	return cancelled
}

func (g *generator) facMessageID(f *facFactor) string {
	return fmt.Sprintf(
		"<%s.%s@mail.%s.example.com>",
		g.fake.digits(12),
		strings.ToLower(g.fake.letters(8)),
		f.slug,
	)
}

// facPlanEvents decides what the factor does with each schedule's invoices:
// fund or decline them, and later release the reserve when the customer
// pays, or charge the invoice back when it does not.
func (g *generator) facPlanEvents(run *facRun) {
	for _, fs := range run.subs {
		if fs.sub.Status != factoringsubmission.SubmissionStatusSent || fs.sub.SentAt == nil {
			continue
		}
		agreement := fs.factor.agreement
		fundOn := facBusinessDays(*fs.sub.SentAt, max(agreement.FundingDays, 0)) +
			g.fake.between64(2*facHour, 6*facHour)
		recourse := agreement.Recourse == factoring.RecourseFull
		for _, it := range fs.items {
			it.fundOn = fundOn
			if g.fake.chance(0.06) {
				it.fundOn = facBusinessDays(fundOn, g.fake.between(3, 6))
			}
			if g.fake.chance(0.03) {
				it.decline = pick(g.fake, facDeclineReasons)
				continue
			}
			due := it.fi.inv.InvoiceDate + 30*facDay
			if it.fi.inv.DueDate != nil {
				due = *it.fi.inv.DueDate
			}
			floor := facWeekStart(it.fundOn) + 8*facDay
			if recourse && g.fake.chance(0.025) {
				it.chargebackOn = max(it.fundOn+int64(agreement.ChargebackWindowDays)*facDay, floor)
				if g.fake.chance(0.2) {
					it.reversalOn = facWeekStart(it.chargebackOn) + 10*facDay +
						g.fake.between64(2, 20)*facDay
					it.payOn = max(it.reversalOn+g.fake.between64(2, 10)*facDay,
						facWeekStart(it.reversalOn)+8*facDay)
				}
			} else {
				it.payOn = max(due+g.fake.between64(-12, 25)*facDay, floor)
			}
			if it.payOn > 0 && it.payOn > due+30*facDay {
				it.extraFee = int64(math.Round(float64(it.item.FaceMinor) * 0.01))
			}
		}
	}
}

type facBucket struct {
	factor *facFactor
	week   int64
	events []*facEvent
}

//nolint:gocognit,funlen // staging, matching and booking happen in the factor's own order
func (g *generator) facRemit(run *facRun) error {
	now := unix(g.now)
	buckets := make(map[string]*facBucket)
	add := func(f *facFactor, ev *facEvent) {
		week := facWeekStart(ev.at)
		key := f.company.ID.String() + "|" + strconv.FormatInt(week, 10)
		b, ok := buckets[key]
		if !ok {
			b = &facBucket{factor: f, week: week}
			buckets[key] = b
		}
		b.events = append(b.events, ev)
	}
	reversals := make([]*facItem, 0)
	for _, fs := range run.subs {
		for _, it := range fs.items {
			if it.fundOn == 0 {
				continue
			}
			if it.decline != "" {
				add(
					fs.factor,
					&facEvent{
						at:   it.fundOn,
						kind: factoringremittance.RemittanceLineRejection,
						item: it,
					},
				)
				continue
			}
			add(
				fs.factor,
				&facEvent{at: it.fundOn, kind: factoringremittance.RemittanceLineFunding, item: it},
			)
			if it.chargebackOn > 0 {
				add(
					fs.factor,
					&facEvent{
						at:   it.chargebackOn,
						kind: factoringremittance.RemittanceLineChargeback,
						item: it,
					},
				)
			}
			if it.reversalOn > 0 {
				reversals = append(reversals, it)
			}
			if it.payOn > 0 {
				add(
					fs.factor,
					&facEvent{
						at:   it.payOn,
						kind: factoringremittance.RemittanceLineReserveRelease,
						item: it,
					},
				)
			}
		}
	}

	type action struct {
		at     int64
		bucket *facBucket
		item   *facItem
	}
	actions := make([]action, 0, len(buckets)+len(reversals))
	for _, b := range buckets {
		received := b.week + 7*facDay + 6*facHour + g.fake.between64(0, 4*facHour)
		if received > now {
			continue
		}
		actions = append(actions, action{at: received, bucket: b})
	}
	for _, it := range reversals {
		if it.reversalOn <= now {
			actions = append(actions, action{at: it.reversalOn, item: it})
		}
	}
	sort.SliceStable(actions, func(i, j int) bool { return actions[i].at < actions[j].at })

	for _, a := range actions {
		if a.bucket != nil {
			if err := g.facApplyBucket(run, a.bucket, a.at); err != nil {
				return err
			}
			continue
		}
		if err := g.facReverse(run, a.item, a.at); err != nil {
			return err
		}
	}

	g.facOddRemittances(run, now)
	return nil
}

var facKindOrder = map[factoringremittance.RemittanceLineKind]int{
	factoringremittance.RemittanceLineFunding:        0,
	factoringremittance.RemittanceLineRejection:      1,
	factoringremittance.RemittanceLineChargeback:     2,
	factoringremittance.RemittanceLineReserveRelease: 3,
	factoringremittance.RemittanceLineOther:          4,
}

func facMoney(minor int64) string {
	return money.DecimalFromMinor(minor).StringFixed(2)
}

//nolint:gocognit,funlen // one remittance file from receipt to booking
func (g *generator) facApplyBucket(run *facRun, b *facBucket, received int64) error {
	now := unix(g.now)
	f := b.factor
	sort.SliceStable(b.events, func(i, j int) bool {
		if facKindOrder[b.events[i].kind] != facKindOrder[b.events[j].kind] {
			return facKindOrder[b.events[i].kind] < facKindOrder[b.events[j].kind]
		}
		return b.events[i].at < b.events[j].at
	})
	if g.fake.chance(0.4) {
		b.events = append(
			b.events,
			&facEvent{
				at:   b.week + 4*facDay,
				kind: factoringremittance.RemittanceLineOther,
				cash: -2500,
			},
		)
	}

	weekEnd := b.week + 7*facDay - facDay
	fileName := fmt.Sprintf("%s_remittance_%s%s", strings.ToUpper(f.company.Code),
		time.Unix(weekEnd, 0).UTC().Format("2006-01-02"), f.style.ext)
	applyAt := received + g.fake.between64(facHour, 30*facHour)
	applied := applyAt <= now && run.bookable[f.company.ID]
	uploader := g.bilPick(g.refs.userIDs)

	rows := make([][]string, 0, len(b.events))
	for _, ev := range b.events {
		rows = append(rows, g.facCells(f, ev))
	}
	sheet := &rateimport.Sheet{Headers: f.style.headers, Rows: rows, FirstDataRow: 2}
	mapping := factoringremit.GuessMapping(f.style.headers)
	parsed := factoringremit.ToDomain(factoringremit.Parse(sheet, mapping, factoringremit.Options{
		DefaultKind: factoringremittance.RemittanceLineFunding,
		Location:    time.UTC,
	}))
	if len(parsed) != len(b.events) {
		return fmt.Errorf(
			"remittance %s parsed %d lines from %d rows",
			fileName,
			len(parsed),
			len(b.events),
		)
	}

	rem := &factoringremittance.Remittance{
		ID:                 idAt("facrem_", received),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		FactoringCompanyID: f.company.ID,
		FileName:           fileName,
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       facFormat(f.style.ext),
		Status:             factoringremittance.RemittanceStatusParsed,
		DefaultKind:        factoringremittance.RemittanceLineFunding,
		RemittedOn:         ptr(weekEnd),
		CurrencyCode:       money.DefaultCurrencyCode,
		Mapping:            mapping.Strings(),
		Headers:            f.style.headers,
		UnmappedHeaders:    factoringremit.UnmappedHeaders(f.style.headers, mapping),
		StagedAt:           ptr(received + 2*facMinute),
		CreatedByID:        uploader,
	}
	doc := g.facDocument(factoringremittance.RemittanceDocumentResourceType, rem.ID, fileName,
		facContentType(f.style.ext), document.ProcessingProfileNone, received, uploader,
		g.fake.between64(4_000, 60_000))
	rem.DocumentID = doc.ID
	run.docs = append(run.docs, doc)
	for _, line := range parsed {
		line.ID = idAt("facrml_", received)
		line.OrganizationID = g.refs.orgID
		line.BusinessUnitID = g.refs.buID
		line.RemittanceID = rem.ID
	}
	rem.Lines = parsed

	factoringremittance.MatchRemittance(parsed, g.facCandidates(run, f))

	touched := make(map[*facSub]int64)
	if applied {
		for i, line := range parsed {
			ev := b.events[i]
			if ev.item == nil {
				if line.MatchStatus.IsException() {
					if err := line.Ignore("Bank fee on the factor's wire; booked with the month's bank charges"); err != nil {
						return err
					}
				}
				continue
			}
			if !facBookable(ev) {
				if line.MatchStatus == factoringremittance.RemittanceLineMatched {
					line.Suggest(nil, factoringremittance.MatchReasonNoOpenItem)
				}
				continue
			}
			if line.MatchStatus != factoringremittance.RemittanceLineMatched ||
				line.SubmissionItemID != ev.item.item.ID {
				score, _ := factoringremittance.ScoreRemittanceLine(
					line,
					&factoringremittance.RemittanceCandidate{
						Item:             ev.item.item,
						SubmissionNumber: ev.item.sub.sub.Number,
						SentAt:           ev.item.sub.sub.SentAt,
						Recourse:         f.agreement.Recourse == factoring.RecourseFull,
					},
				)
				line.MatchTo(
					ev.item.item,
					factoringremittance.RemittanceMatchedByPerson,
					max(score, 1),
					factoringremittance.MatchReasonChosenByPerson,
				)
			}
			entryID, err := g.facBookLine(run, rem, line, ev, applyAt, uploader)
			if err != nil {
				return err
			}
			line.MarkApplied(entryID)
			if ev.kind == factoringremittance.RemittanceLineFunding ||
				ev.kind == factoringremittance.RemittanceLineRejection {
				touched[ev.item.sub] = max(touched[ev.item.sub], ev.at)
			}
			if entryID.IsNotNil() {
				run.applied = append(run.applied, facAppliedLine{line: line, entryID: entryID})
			}
		}
		for fs, fundedOn := range touched {
			fs.sub.Resolve(fundedOn)
			fs.sub.UpdatedAt = applyAt
		}
		rem.MarkApplied(applyAt, uploader)
	} else {
		rem.Recount()
	}

	updated := received + 2*facMinute
	if rem.AppliedAt != nil {
		updated = *rem.AppliedAt
	}
	run.remittances = append(run.remittances, rem)
	run.remStamps = append(run.remStamps, stamp{id: rem.ID, created: received, updated: updated})
	for _, line := range parsed {
		run.lines = append(run.lines, line)
		run.lineStamps = append(
			run.lineStamps,
			stamp{id: line.ID, created: received, updated: updated},
		)
	}
	if !f.profiled {
		f.profiled = true
		run.profiles = append(run.profiles, &factoringremittance.RemittanceProfile{
			ID:                 idAt("facrmp_", received),
			OrganizationID:     g.refs.orgID,
			BusinessUnitID:     g.refs.buID,
			FactoringCompanyID: f.company.ID,
			Mapping:            mapping.Strings(),
			HeaderSignature:    factoringremit.HeaderSignature(f.style.headers),
			CreatedAt:          received,
			UpdatedAt:          received,
		})
	}
	return nil
}

// facBookable says whether the line's item is still where the factor's report
// expects it; one that is not stays on the remittance for someone to resolve.
func facBookable(ev *facEvent) bool {
	switch ev.kind {
	case factoringremittance.RemittanceLineFunding, factoringremittance.RemittanceLineRejection:
		return ev.item.item.Status == factoringsubmission.SubmissionItemStatusPending
	case factoringremittance.RemittanceLineReserveRelease,
		factoringremittance.RemittanceLineChargeback:
		return ev.item.item.Status == factoringsubmission.SubmissionItemStatusFunded
	case factoringremittance.RemittanceLineOther:
	}
	return false
}

func facFormat(ext string) factoringremittance.RemittanceFormat {
	switch ext {
	case ".xlsx":
		return factoringremittance.RemittanceFormatXlsx
	case ".pdf":
		return factoringremittance.RemittanceFormatPdf
	default:
		return factoringremittance.RemittanceFormatCsv
	}
}

func facContentType(ext string) string {
	switch ext {
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pdf":
		return "application/pdf"
	default:
		return "text/csv"
	}
}

// facCells writes one row of the factor's report in its own column layout.
// A few invoice numbers lose their prefix, as factors' systems often print
// them, so those lines land as suggestions for someone to confirm.
func (g *generator) facCells(f *facFactor, ev *facEvent) []string {
	date := time.Unix(ev.at, 0).UTC().Format("2006-01-02")
	if ev.item == nil {
		return []string{
			"", "WIRE-" + g.fake.digits(6), f.company.Name, f.style.other,
			"", "", "", "", facMoney(ev.cash), date,
		}
	}
	it := ev.item
	number := it.item.InvoiceNumber
	if g.fake.chance(0.03) {
		number = strings.TrimLeft(number, "ABCDEFGHIJKLMNOPQRSTUVWXYZ-")
	}
	debtor := it.fi.rec.Customer.Name
	face := facMoney(it.item.FaceMinor)
	switch ev.kind {
	case factoringremittance.RemittanceLineFunding:
		reserve := it.item.FaceMinor - it.item.ExpectedAdvanceMinor - it.item.ExpectedFeeMinor
		return []string{
			number, it.sub.sub.Number, debtor, f.style.purchase, face,
			facMoney(it.item.ExpectedAdvanceMinor), facMoney(it.item.ExpectedFeeMinor),
			facMoney(reserve), facMoney(it.item.ExpectedAdvanceMinor), date,
		}
	case factoringremittance.RemittanceLineRejection:
		return []string{
			number, it.sub.sub.Number, debtor, f.style.decline + " - " + it.decline, face,
			"", "", "", "", date,
		}
	case factoringremittance.RemittanceLineChargeback:
		repurchase := it.item.ExpectedAdvanceMinor + it.item.ExpectedFeeMinor
		return []string{
			number, it.sub.sub.Number, debtor, f.style.chargeback, face,
			"", "", "", facMoney(-repurchase), date,
		}
	default:
		reserve := it.item.FaceMinor - it.item.ExpectedAdvanceMinor - it.item.ExpectedFeeMinor
		extra := min(it.extraFee, reserve)
		fee := ""
		if extra > 0 {
			fee = facMoney(extra)
		}
		return []string{
			number, it.sub.sub.Number, debtor, f.style.release, face,
			"", fee, facMoney(reserve), facMoney(reserve - extra), date,
		}
	}
}

func (g *generator) facCandidates(
	run *facRun,
	f *facFactor,
) []*factoringremittance.RemittanceCandidate {
	recourse := f.agreement.Recourse == factoring.RecourseFull
	out := make([]*factoringremittance.RemittanceCandidate, 0)
	for _, fs := range run.subs {
		if fs.factor != f {
			continue
		}
		for _, it := range fs.items {
			if !it.item.Status.IsLive() {
				continue
			}
			out = append(out, &factoringremittance.RemittanceCandidate{
				Item:             it.item,
				SubmissionNumber: fs.sub.Number,
				SentAt:           fs.sub.SentAt,
				Recourse:         recourse,
			})
		}
	}
	return out
}

// facBookLine books one matched line the way remittance apply does: funding
// and rejections through RecordFunding, releases and chargebacks through the
// reserve ledger.
func (g *generator) facBookLine(
	run *facRun,
	rem *factoringremittance.Remittance,
	line *factoringremittance.RemittanceLine,
	ev *facEvent,
	at int64,
	actor pulid.ID,
) (pulid.ID, error) {
	it := ev.item
	agreement := it.sub.factor.agreement
	row := "row " + strconv.Itoa(line.RowNumber)
	switch ev.kind {
	case factoringremittance.RemittanceLineFunding:
		if err := it.item.Fund(ev.at, it.item.ExpectedFunding()); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		movement, err := factoringledger.PlanFundingMovement(it.item, agreement.AccountingTreatment)
		if err != nil {
			return pulid.Nil, err
		}
		return g.facBook(run, &facBooking{
			item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringFunded,
			key: "factoring-funded:" + it.item.ID.String(), on: ev.at, at: at, actor: actor,
			memo: "Factor remittance " + rem.FileName, remitLine: line.ID,
		})
	case factoringremittance.RemittanceLineRejection:
		if err := it.item.Reject("Declined on the factor's remittance " + rem.FileName + ", " + row); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		it.fi.rejectedAt = at
		return pulid.Nil, nil
	case factoringremittance.RemittanceLineChargeback:
		movement, err := factoringledger.PlanChargebackMovement(
			it.item,
			agreement.AccountingTreatment,
			factoringledger.ChargebackRepurchaseMinor(it.item),
		)
		if err != nil {
			return pulid.Nil, err
		}
		if err = it.item.ChargeBack(ev.at, "Charged back on the factor's remittance "+rem.FileName+", "+row); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		id, err := g.facBook(run, &facBooking{
			item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringChargedBack,
			key: "factoring-charged-back:" + it.item.ID.String() + ":1", on: ev.at, at: at,
			actor: actor, memo: it.item.ChargebackReason, remitLine: line.ID,
		})
		it.chargeback = &movement
		it.chargebackID = id
		return id, err
	default:
		_, _, reserve := it.item.Actuals()
		cash := max(reserve-min(it.extraFee, reserve), 0)
		movement, err := factoringledger.PlanReleaseMovement(
			it.item,
			agreement.AccountingTreatment,
			cash,
		)
		if err != nil {
			return pulid.Nil, err
		}
		if err = it.item.Settle(ev.at); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		return g.facBook(run, &facBooking{
			item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringReserveReleased,
			key: "factoring-reserve-released:" + it.item.ID.String(), on: ev.at, at: at, actor: actor,
			memo: "Released on the factor's remittance " + rem.FileName + ", " + row, remitLine: line.ID,
		})
	}
}

// facReverse undoes a chargeback by hand once the customer paid the factor
// after all, as LedgerService.ReverseChargeback does.
func (g *generator) facReverse(run *facRun, it *facItem, at int64) error {
	if it.chargeback == nil ||
		it.item.Status != factoringsubmission.SubmissionItemStatusChargedBack {
		it.payOn = 0
		return nil
	}
	movement := factoringledger.ReverseMovement(it.chargeback)
	if err := it.item.ReverseChargeback(); err != nil {
		return err
	}
	it.item.UpdatedAt = at
	_, err := g.facBook(run, &facBooking{
		item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringChargebackReversed,
		key: "factoring-chargeback-reversed:" + it.chargebackID.String(), on: at, at: at + 10*facMinute,
		actor: g.bilPick(g.refs.userIDs), memo: "The customer paid the factor after the chargeback",
		reverses: it.chargebackID,
	})
	return err
}

type facBooking struct {
	item      *facItem
	movement  factoringledger.Movement
	event     accountingcontrol.JournalSourceEventType
	key       string
	on        int64
	at        int64
	actor     pulid.ID
	memo      string
	remitLine pulid.ID
	reverses  pulid.ID
}

func facDescription(kind factoringledger.ReserveEntryKind, number, schedule string) string {
	switch kind {
	case factoringledger.ReserveEntryFunding:
		return "Factor advance on invoice " + number + ", schedule " + schedule
	case factoringledger.ReserveEntryRelease:
		return "Factor reserve released on invoice " + number + ", schedule " + schedule
	case factoringledger.ReserveEntryChargeback:
		return "Factor chargeback of invoice " + number + ", schedule " + schedule
	case factoringledger.ReserveEntryChargebackReversal:
		return "Factor chargeback reversed on invoice " + number + ", schedule " + schedule
	default:
		return "Factoring of invoice " + number + ", schedule " + schedule
	}
}

// facBook posts a reserve movement: the journal, the append-only reserve
// entry and, for a movement that settles or reopens the invoice, the invoice
// balance and its customer ledger line, as LedgerService.postMovement does.
func (g *generator) facBook(run *facRun, b *facBooking) (pulid.ID, error) {
	if b.movement.IsEmpty() {
		return pulid.Nil, nil
	}
	it := b.item
	sub := it.sub.sub
	journal, err := g.bilNewJournal(run.setup, b.on)
	if err != nil {
		return pulid.Nil, err
	}
	accounts := run.accounts[sub.FactoringCompanyID]
	description := facDescription(b.movement.Kind, it.item.InvoiceNumber, sub.Number)
	journal.actorID = b.actor
	journal.description = description
	journal.sourceType = facSourceObject
	journal.sourceID = it.item.ID
	journal.document = it.item.InvoiceNumber
	journal.event = b.event
	journal.referenceType = b.event.String()
	journal.idempotency = b.key
	journal.auto = true
	for _, line := range b.movement.Lines {
		journal.legs = append(journal.legs, bilLeg{
			account:  accounts.For(line.Role),
			debit:    line.Debit,
			credit:   line.Credit,
			customer: it.item.CustomerID,
		})
	}
	run.rows.journals = append(run.rows.journals, journal)

	entry := factoringledger.NewReserveEntry(it.item, sub, &b.movement, b.on)
	row := &factoringledger.ReserveEntry{
		ID:                 idAt("facres_", b.at),
		OrganizationID:     entry.OrganizationID,
		BusinessUnitID:     entry.BusinessUnitID,
		FactoringCompanyID: entry.FactoringCompanyID,
		AgreementID:        entry.AgreementID,
		SubmissionID:       entry.SubmissionID,
		SubmissionItemID:   entry.SubmissionItemID,
		InvoiceID:          entry.InvoiceID,
		CustomerID:         entry.CustomerID,
		Kind:               entry.Kind,
		Treatment:          entry.Treatment,
		CurrencyCode:       entry.CurrencyCode,
		CashMinor:          entry.CashMinor,
		FeeMinor:           entry.FeeMinor,
		ReserveMinor:       entry.ReserveMinor,
		LiabilityMinor:     entry.LiabilityMinor,
		ARMinor:            entry.ARMinor,
		JournalEntryID:     journal.entryID,
		JournalBatchID:     journal.batchID,
		AccountingDate:     journal.date,
		OccurredOn:         b.on,
		ReversesEntryID:    b.reverses,
		RemittanceLineID:   b.remitLine,
		Memo:               b.memo,
		CreatedByID:        b.actor,
		CreatedAt:          b.at,
	}
	if len([]rune(row.Memo)) > factoringledger.MaxReserveMemoLength {
		row.Memo = string([]rune(row.Memo)[:factoringledger.MaxReserveMemoLength])
	}
	if row.CurrencyCode == "" {
		row.CurrencyCode = money.DefaultCurrencyCode
	}
	run.entries = append(run.entries, row)

	inv := it.fi.inv
	signed := int64(0)
	switch b.movement.InvoiceEffect {
	case factoringledger.InvoiceEffectSettle:
		inv.ApplyPaymentMinor(b.movement.InvoiceMinor)
		signed = -b.movement.InvoiceMinor
	case factoringledger.InvoiceEffectReopen:
		inv.RemovePaymentMinor(b.movement.InvoiceMinor)
		signed = b.movement.InvoiceMinor
	case factoringledger.InvoiceEffectNone:
	}
	if signed != 0 {
		it.fi.touched = true
		it.fi.updatedAt = max(it.fi.updatedAt, b.at)
		run.rows.ledger = append(run.rows.ledger, &customerledger.CustomerLedgerEntry{
			ID:               idAt("cledg_", b.at),
			OrganizationID:   inv.OrganizationID,
			BusinessUnitID:   inv.BusinessUnitID,
			CustomerID:       inv.CustomerID,
			SourceObjectType: facSourceObject,
			SourceObjectID:   row.ID.String(),
			SourceEventType:  b.event.String(),
			RelatedInvoiceID: inv.ID,
			DocumentNumber:   sub.Number,
			TransactionDate:  journal.date,
			LineNumber:       1,
			AmountMinor:      signed,
			CreatedByID:      b.actor,
			CreatedAt:        b.at,
		})
	}
	return row.ID, nil
}

// facExclude takes declined invoices out of factoring, as a biller does from
// the factoring page, and records the customer paying the organization
// directly once it does.
func (g *generator) facExclude(run *facRun) error {
	now := unix(g.now)
	byCustomer := make(map[pulid.ID]*customerRecord, len(g.customers))
	for _, c := range g.customers {
		byCustomer[c.ID] = c
	}
	for _, fi := range run.invoices {
		if fi.rejectedAt == 0 {
			continue
		}
		at := fi.rejectedAt + g.fake.between64(4*facHour, 3*facDay)
		if at > now {
			continue
		}
		number := ""
		for _, it := range fi.items {
			if it.item.Status == factoringsubmission.SubmissionItemStatusRejected {
				number = it.sub.sub.Number
			}
		}
		fi.inv.FactoringOverride = invoice.FactoringOverrideExclude
		fi.inv.FactoringOverrideReason = "Declined by the factor on schedule " + number +
			"; collected directly"
		fi.inv.FactoringCompanyID = nil
		fi.inv.FactoringAgreementID = nil
		fi.touched = true
		fi.updatedAt = max(fi.updatedAt, at)
		run.excluded++

		open := fi.inv.OpenBalanceMinor()
		if !run.setup.postPayments || open <= 0 {
			continue
		}
		due := fi.inv.InvoiceDate + 30*facDay
		if fi.inv.DueDate != nil {
			due = *fi.inv.DueDate
		}
		paid := max(due+g.fake.between64(-5, 25)*facDay, at+2*facDay)
		if paid > now-facHour || !g.fake.chance(0.85) {
			continue
		}
		cus := byCustomer[fi.inv.CustomerID]
		if cus == nil {
			continue
		}
		if err := g.bilPayment(run.setup, run.rows, cus, paid, []*bilPaymentPlan{{
			invoice: fi.inv, amount: open, date: paid,
		}}); err != nil {
			return err
		}
		fi.updatedAt = max(fi.updatedAt, paid)
	}
	return nil
}

// facStates writes each invoice's readiness as the factoring watch last left
// it, using the domain's own evaluation.
func (g *generator) facStates(run *facRun) {
	now := unix(g.now)
	for _, fi := range run.invoices {
		inv := fi.inv
		evaluated := now - g.fake.between64(facMinute, 50*facMinute)
		facts := factoringreadiness.InvoiceFacts{
			Posted:           inv.Status == invoice.StatusPosted,
			Voided:           inv.Status == invoice.StatusVoided,
			Excluded:         inv.FactoringOverride.Excludes(),
			OpenBalanceMinor: inv.OpenBalanceMinor(),
		}
		if inv.FactoringCompanyID != nil {
			facts.FactoringCompanyID = *inv.FactoringCompanyID
		}
		in := &factoringreadiness.ReadinessInputs{At: evaluated, Invoice: facts}
		if f := fi.factor; f != nil && inv.FactoringCompanyID != nil {
			if f.agreement.InForce(evaluated) {
				in.Agreement = f.agreement
			}
			if noa := f.noas[inv.CustomerID]; noa != nil &&
				noa.Status != factoring.NoaStatusReleased {
				in.Noa = noa
			}
			in.OpenDispute = inv.DisputeStatus == invoice.DisputeStatusDisputed
			in.Shipments = []factoringreadiness.ShipmentFacts{{
				ShipmentID: fi.rec.ID,
				ProNumber:  fi.rec.ProNumber,
				Available:  true,
			}}
			for _, it := range fi.items {
				switch {
				case it.item.Status.IsLive():
					in.SubmissionNumber = it.sub.sub.Number
					in.SubmissionItemStatus = it.item.Status
				case it.item.Status == factoringsubmission.SubmissionItemStatusChargedBack:
					in.ChargedBack = true
				}
			}
		}
		readiness := factoringreadiness.EvaluateReadiness(in)
		posted := inv.InvoiceDate
		if inv.PostedAt != nil {
			posted = *inv.PostedAt
		}
		state := &factoringreadiness.InvoiceState{
			InvoiceID:          inv.ID,
			OrganizationID:     inv.OrganizationID,
			BusinessUnitID:     inv.BusinessUnitID,
			FactoringCompanyID: inv.FactoringCompanyID,
			CustomerID:         inv.CustomerID,
			Verdict:            readiness.Verdict,
			Checks:             readiness.Checks,
			FailedChecks:       readiness.FailedCount(),
			OpenBalanceMinor:   inv.OpenBalanceMinor(),
			EvaluatedAt:        evaluated,
			CreatedAt:          min(posted+facMinute, evaluated),
		}
		if readiness.Verdict == factoringreadiness.VerdictReady {
			since := fi.readyAt
			for _, it := range fi.items {
				if it.item.ReleasedAt != nil {
					since = max(since, *it.item.ReleasedAt)
				}
			}
			if since == 0 || since > evaluated {
				since = min(posted+facMinute, evaluated)
			}
			state.ReadySince = ptr(since)
			state.ReadyAnnouncedAt = ptr(
				min(since+g.fake.between64(facMinute, 10*facMinute), evaluated),
			)
		}
		run.states = append(run.states, state)
	}
}

// facOddRemittances adds the files that never book anything: one whose
// columns need mapping, one that could not be read and a duplicate that was
// discarded.
func (g *generator) facOddRemittances(run *facRun, now int64) {
	if len(run.remittances) == 0 {
		return
	}
	f := run.subs[0].factor
	uploader := g.bilPick(g.refs.userIDs)

	mappingAt := now - g.fake.between64(2*facHour, 2*facDay)
	headers := []string{"Client", "Doc #", "Purch Dt", "Gross", "Holdback", "Net Funds"}
	mapping := factoringremit.GuessMapping(headers)
	needs := &factoringremittance.Remittance{
		ID:                 idAt("facrem_", mappingAt),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		FactoringCompanyID: f.company.ID,
		FileName:           strings.ToUpper(f.company.Code) + "_month_end_statement.csv",
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       factoringremittance.RemittanceFormatCsv,
		Status:             factoringremittance.RemittanceStatusNeedsMapping,
		DefaultKind:        factoringremittance.RemittanceLineReserveRelease,
		CurrencyCode:       money.DefaultCurrencyCode,
		Mapping:            mapping.Strings(),
		Headers:            headers,
		UnmappedHeaders:    factoringremit.UnmappedHeaders(headers, mapping),
		StagedAt:           ptr(mappingAt + facMinute),
		CreatedByID:        uploader,
	}
	needsDoc := g.facDocument(
		factoringremittance.RemittanceDocumentResourceType,
		needs.ID,
		needs.FileName,
		"text/csv",
		document.ProcessingProfileNone,
		mappingAt,
		uploader,
		18_400,
	)
	needs.DocumentID = needsDoc.ID

	failedAt := now - g.fake.between64(3*facDay, 20*facDay)
	failed := &factoringremittance.Remittance{
		ID:                 idAt("facrem_", failedAt),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		FactoringCompanyID: f.company.ID,
		FileName:           strings.ToUpper(f.company.Code) + "_funding_report_scan.pdf",
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       factoringremittance.RemittanceFormatPdf,
		Status:             factoringremittance.RemittanceStatusFailed,
		DefaultKind:        factoringremittance.RemittanceLineFunding,
		CurrencyCode:       money.DefaultCurrencyCode,
		Error:              "The PDF is a scanned image with no readable table of invoices",
		StagedAt:           ptr(failedAt + 3*facMinute),
		CreatedByID:        uploader,
	}
	failedDoc := g.facDocument(
		factoringremittance.RemittanceDocumentResourceType,
		failed.ID,
		failed.FileName,
		"application/pdf",
		document.ProcessingProfileFactoringRemittance,
		failedAt,
		uploader,
		812_000,
	)
	failed.DocumentID = failedDoc.ID

	original := run.remittances[g.fake.intn(len(run.remittances))]
	dupAt := (*original.StagedAt) + g.fake.between64(facHour, 6*facHour)
	discarded := &factoringremittance.Remittance{
		ID:                 idAt("facrem_", dupAt),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		FactoringCompanyID: original.FactoringCompanyID,
		FileName:           original.FileName,
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       original.SourceFormat,
		Status:             factoringremittance.RemittanceStatusDiscarded,
		DefaultKind:        factoringremittance.RemittanceLineFunding,
		CurrencyCode:       money.DefaultCurrencyCode,
		DiscardedAt:        ptr(dupAt + 20*facMinute),
		DiscardReason:      "Duplicate upload of " + original.FileName,
		CreatedByID:        uploader,
	}

	run.remittances = append(run.remittances, needs, failed, discarded)
	run.docs = append(run.docs, needsDoc, failedDoc)
	run.remStamps = append(run.remStamps,
		stamp{id: needs.ID, created: mappingAt, updated: mappingAt + facMinute},
		stamp{id: failed.ID, created: failedAt, updated: failedAt + 3*facMinute},
		stamp{id: discarded.ID, created: dupAt, updated: dupAt + 20*facMinute},
	)
}

//nolint:gocognit,funlen // writes follow the foreign keys in one place
func (g *generator) facWrite(ctx context.Context, tx *keel.DB, run *facRun) error {
	if len(run.docs) > 0 {
		marks := make([]stamp, 0, len(run.docs))
		for _, d := range run.docs {
			marks = append(marks, stamp{id: d.ID, created: d.CreatedAt, updated: d.CreatedAt})
		}
		if err := insertRows(ctx, tx, documentdb.Documents.Handle, run.docs); err != nil {
			return fmt.Errorf("insert remittance documents: %w", err)
		}
		if err := backdate(ctx, tx, documentdb.Documents.Handle, marks); err != nil {
			return err
		}
	}

	subs := make([]*factoringsubmission.Submission, 0, len(run.subs))
	items := make([]*factoringsubmission.SubmissionItem, 0)
	subMarks := make([]stamp, 0, len(run.subs))
	itemMarks := make([]stamp, 0)
	for _, fs := range run.subs {
		subs = append(subs, fs.sub)
		subMarks = append(
			subMarks,
			stamp{id: fs.sub.ID, created: fs.slot, updated: fs.sub.UpdatedAt},
		)
		for _, it := range fs.items {
			items = append(items, it.item)
			itemMarks = append(
				itemMarks,
				stamp{id: it.item.ID, created: fs.slot, updated: it.item.UpdatedAt},
			)
		}
	}
	if err := insertRows(ctx, tx, factoringdb.Submissions.Handle, subs); err != nil {
		return fmt.Errorf("insert factoring submissions: %w", err)
	}
	if err := insertRows(ctx, tx, factoringdb.SubmissionItems.Handle, items); err != nil {
		return fmt.Errorf("insert factoring submission items: %w", err)
	}

	pending := make(map[pulid.ID]pulid.ID, len(run.applied))
	for _, a := range run.applied {
		pending[a.line.ID] = a.entryID
		a.line.MatchStatus = factoringremittance.RemittanceLineMatched
		a.line.ReserveEntryID = pulid.Nil
	}
	if err := insertRows(ctx, tx, factoringdb.Remittances.Handle, run.remittances); err != nil {
		return fmt.Errorf("insert factoring remittances: %w", err)
	}
	if err := insertRows(ctx, tx, factoringdb.RemittanceLines.Handle, run.lines); err != nil {
		return fmt.Errorf("insert factoring remittance lines: %w", err)
	}
	profileMarks := make([]stamp, 0, len(run.profiles))
	for _, p := range run.profiles {
		profileMarks = append(
			profileMarks,
			stamp{id: p.ID, created: p.CreatedAt, updated: p.CreatedAt},
		)
	}
	if err := insertRows(ctx, tx, factoringdb.RemittanceProfiles.Handle, run.profiles); err != nil {
		return fmt.Errorf("insert factoring remittance profiles: %w", err)
	}

	if err := g.bilWrite(ctx, tx, run.rows); err != nil {
		return err
	}
	if err := insertRows(ctx, tx, factoringdb.ReserveEntries.Handle, run.entries); err != nil {
		return fmt.Errorf("insert factoring reserve entries: %w", err)
	}

	if len(pending) > 0 {
		lineIDs := make([]string, 0, len(pending))
		entryIDs := make([]string, 0, len(pending))
		for lineID, entryID := range pending {
			lineIDs = append(lineIDs, lineID.String())
			entryIDs = append(entryIDs, entryID.String())
		}
		rl := factoringdb.RemittanceLines
		d := keel.Unnest("d", []string{"id", "reserve_entry_id"}, lineIDs, entryIDs)
		if _, err := rl.With(d).Join(d, d.Col("id").EqCol(rl.ID)).Where(keel.True()).Update(
			ctx,
			tx,
			rl.MatchStatus.To(factoringremittance.RemittanceLineApplied),
			rl.ReserveEntryID.ToExpr(d.Col("reserve_entry_id")),
			keepUpdated(rl.Handle),
		); err != nil {
			return fmt.Errorf("mark remittance lines applied: %w", err)
		}
	}

	if err := g.facUpdateInvoices(ctx, tx, run); err != nil {
		return err
	}
	if err := insertRows(ctx, tx, factoringdb.InvoiceStates.Handle, run.states); err != nil {
		return fmt.Errorf("insert factoring invoice states: %w", err)
	}

	for _, b := range []struct {
		table string
		marks []stamp
	}{
		{"factoring_submissions", subMarks},
		{"factoring_submission_items", itemMarks},
		{"factoring_remittances", run.remStamps},
		{"factoring_remittance_lines", run.lineStamps},
	} {
		if err := backdateTable(ctx, tx, b.table, b.marks); err != nil {
			return err
		}
	}
	if len(profileMarks) > 0 {
		if err := backdate(ctx, tx, factoringdb.RemittanceProfiles.Handle, profileMarks); err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) facUpdateInvoices(ctx context.Context, tx *keel.DB, run *facRun) error {
	var ids, settlements, overrides, reasons, factors, agreements []string
	var applied []string
	var appliedMinor, updated []int64
	for _, fi := range run.invoices {
		if !fi.touched {
			continue
		}
		inv := fi.inv
		ids = append(ids, inv.ID.String())
		applied = append(applied, inv.AppliedAmount.StringFixed(4))
		appliedMinor = append(appliedMinor, inv.AppliedAmountMinor)
		settlements = append(settlements, string(inv.SettlementStatus))
		overrides = append(overrides, string(inv.FactoringOverride))
		reasons = append(reasons, inv.FactoringOverrideReason)
		factor, agreement := "", ""
		if inv.FactoringCompanyID != nil {
			factor = inv.FactoringCompanyID.String()
		}
		if inv.FactoringAgreementID != nil {
			agreement = inv.FactoringAgreementID.String()
		}
		factors = append(factors, factor)
		agreements = append(agreements, agreement)
		updated = append(updated, max(fi.updatedAt, inv.UpdatedAt))
		inv.UpdatedAt = max(fi.updatedAt, inv.UpdatedAt)
	}
	for begin := 0; begin < len(ids); begin += backdateChunk {
		end := min(begin+backdateChunk, len(ids))
		d := keel.Unnest("d", []string{
			"id", "applied_amount", "applied_amount_minor", "settlement_status", "factoring_override",
			"factoring_override_reason", "factoring_company_id", "factoring_agreement_id", "updated_at",
		}, ids[begin:end], applied[begin:end], appliedMinor[begin:end], settlements[begin:end],
			overrides[begin:end], reasons[begin:end], factors[begin:end], agreements[begin:end],
			updated[begin:end])
		inv := invoicedb.Invoices
		if _, err := inv.With(d).Join(d, d.Col("id").EqCol(inv.ID)).Where(keel.True()).Update(
			ctx,
			tx,
			inv.AppliedAmount.ToExpr(keel.CastNumeric(d.Col("applied_amount"))),
			inv.AppliedAmountMinor.ToExpr(d.Col("applied_amount_minor")),
			inv.SettlementStatus.ToExpr(d.Col("settlement_status")),
			inv.FactoringOverride.ToExpr(d.Col("factoring_override")),
			inv.FactoringOverrideReason.ToExpr(nullIfEmpty(d.Col("factoring_override_reason"))),
			inv.FactoringCompanyID.ToExpr(nullIfEmpty(d.Col("factoring_company_id"))),
			inv.FactoringAgreementID.ToExpr(nullIfEmpty(d.Col("factoring_agreement_id"))),
			inv.UpdatedAt.ToExpr(d.Col("updated_at")),
		); err != nil {
			return fmt.Errorf("update factored invoices: %w", err)
		}
	}
	return nil
}

func (g *generator) facCount(run *facRun) {
	items, funded, rejected, settled, chargedBack := 0, 0, 0, 0, 0
	for _, fs := range run.subs {
		for _, it := range fs.items {
			items++
			switch it.item.Status {
			case factoringsubmission.SubmissionItemStatusFunded:
				funded++
			case factoringsubmission.SubmissionItemStatusRejected:
				rejected++
			case factoringsubmission.SubmissionItemStatusSettled:
				settled++
			case factoringsubmission.SubmissionItemStatusChargedBack:
				chargedBack++
			case factoringsubmission.SubmissionItemStatusPending,
				factoringsubmission.SubmissionItemStatusReleased:
			}
		}
	}
	verdicts := make(map[factoringreadiness.Verdict]int)
	for _, s := range run.states {
		verdicts[s.Verdict]++
	}
	g.report.add("Factoring schedules", len(run.subs))
	g.report.add("Factoring schedule items", items)
	g.report.add("Factoring reserve entries", len(run.entries))
	g.report.add("Factor remittances", len(run.remittances))
	g.report.add("Factor remittance lines", len(run.lines))
	g.report.add("Factoring invoice states", len(run.states))
	g.report.add("Factoring payments collected directly", len(run.rows.payments))
	g.report.add("Customer ledger entries", len(run.rows.ledger))
	g.report.add("Journal entries", len(run.rows.entries))
	g.opts.Logf(
		"  • factoring items: %d funded, %d settled, %d charged back, %d declined (%d excluded)",
		funded,
		settled,
		chargedBack,
		rejected,
		run.excluded,
	)
	g.opts.Logf(
		"  • factoring readiness: %d ready, %d not ready, %d submitted, %d funded, %d excluded",
		verdicts[factoringreadiness.VerdictReady],
		verdicts[factoringreadiness.VerdictNotReady],
		verdicts[factoringreadiness.VerdictSubmitted],
		verdicts[factoringreadiness.VerdictFunded],
		verdicts[factoringreadiness.VerdictExcluded],
	)
}
