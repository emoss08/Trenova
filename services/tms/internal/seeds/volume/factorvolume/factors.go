package factorvolume

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance/factoringremittancedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	facAccountDueFrom   = "1150"
	facAccountLiability = "2420"
	facAccountFee       = "6840"
)

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
	style       volumegen.FactorStyle
}

var facStyles = []volumegen.FactorStyle{
	{
		Headers: []string{
			"Invoice Number", "Schedule", "Debtor", "Type", "Face Amount",
			"Advance", "Fee", "Reserve", "Net Paid", "Date",
		},
		Ext: ".csv", Purchase: "Purchase", Release: "Reserve Release",
		Decline: "Declined", Chargeback: "Chargeback", Other: "Wire Fee",
	},
	{
		Headers: []string{
			"Inv No", "Batch Number", "Customer Name", "Transaction Type", "Gross Amount",
			"Advance Amount", "Discount Fee", "Reserve Amount", "Amount Paid", "Transaction Date",
		},
		Ext: ".xlsx", Purchase: "Funded", Release: "Rebate",
		Decline: "Non-Purchase", Chargeback: "Recourse", Other: "ACH Fee",
	},
	{
		Headers: []string{
			"Client Invoice", "Reference", "Account Debtor", "Activity", "Purchased Amount",
			"Amount Advanced", "Factor Fee", "Reserve Held", "Disbursed", "Funding Date",
		},
		Ext: ".csv", Purchase: "Advance", Release: "Escrow Release",
		Decline: "Rejected", Chargeback: "Repurchase", Other: "Wire Fee",
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
		advance: 9300, reserve: 300, minimumFee: 1_000,
		recourse: factoring.RecourseFull, treatment: factoring.AccountingTreatmentSecuredBorrowing,
		fundingDays: 1, style: facStyles[2],
	},
}

// facMasterData sets up the factors, their agreements, delivery settings and
// notices of assignment, and puts a minority of customers on factoring.
func (g *factorPhase) facMasterData(ctx context.Context, tx *keel.DB) error {
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
	st := &volumegen.FactorState{
		Factors: factors,
		ByID:    make(map[pulid.ID]*volumegen.Factor, len(factors)),
	}
	for _, f := range factors {
		st.ByID[f.Company.ID] = f
	}
	g.Factoring = st

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

func (g *factorPhase) facAccounts(ctx context.Context, tx *keel.DB) (map[string]pulid.ID, error) {
	codes := []string{"1100", "2400", "6800"}
	for _, def := range facAccountDefs {
		codes = append(codes, def.code)
	}
	byCode, err := g.GLAccountsByCode(ctx, codes...)
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
			BusinessUnitID: g.Refs.BuID,
			OrganizationID: g.Refs.OrgID,
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
		if err := volumegen.InsertRows(ctx, tx, glaccountdb.GLAccounts.Handle, created); err != nil {
			return nil, fmt.Errorf("insert factoring gl accounts: %w", err)
		}
		g.Report.Add("GL accounts", len(created))
	}

	return byCode, nil
}

func facCity(name, state string) *volumegen.CityRecord {
	for i := range volumegen.Cities {
		if volumegen.Cities[i].Name == name && volumegen.Cities[i].State == state {
			return &volumegen.Cities[i]
		}
	}
	return &volumegen.Cities[0]
}

//nolint:funlen // reuse and creation of the factor catalogue read best together
func (g *factorPhase) facFactors(
	ctx context.Context,
	tx *keel.DB,
	accounts map[string]pulid.ID,
) ([]*volumegen.Factor, error) {
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
	transports, err := factoringsubmissiondb.Transports.Query().All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load factoring transports: %w", err)
	}
	rp := factoringremittancedb.RemittanceProfiles
	profiles, err := rp.Pick(rp.FactoringCompanyID).All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load remittance profiles: %w", err)
	}
	profiled := make([]pulid.ID, 0, len(profiles))
	for _, p := range profiles {
		profiled = append(profiled, p.FactoringCompanyID)
	}

	start, now := volumegen.Unix(g.Start), volumegen.Unix(g.Now)
	codes := make(map[string]struct{}, len(companies))
	names := make(map[string]struct{}, len(companies))
	for _, c := range companies {
		codes[strings.ToLower(c.Code)] = struct{}{}
		names[strings.ToLower(c.Name)] = struct{}{}
	}

	factors := make([]*volumegen.Factor, 0, len(facCatalog))
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
		factors = append(factors, &volumegen.Factor{
			Company:   c,
			Agreement: agreement,
			Transport: transport,
			Noas:      make(map[pulid.ID]*factoring.Noa),
			Style:     facStyles[len(factors)%len(facStyles)],
			Slug:      volumegen.Slug(c.Name),
			Profiled:  slices.Contains(profiled, c.ID),
		})
	}

	var newCompanies []*factoringcompany.FactoringCompany
	var newAgreements []*factoring.Agreement
	var newTiers []*factoring.FeeTier
	var newTransports []*factoringsubmission.Transport
	var companyStamps, agreementStamps, transportStamps []volumegen.Stamp
	for _, p := range facCatalog {
		if len(factors) == len(facCatalog) {
			break
		}
		if _, taken := names[strings.ToLower(p.name)]; taken {
			continue
		}
		created := start - g.Fake.Between64(420, 900)*volumegen.Day
		city := facCity(p.city, p.state)
		slug := volumegen.Slug(p.name)
		company := &factoringcompany.FactoringCompany{
			ID:             volumegen.IDAt("fac_", created),
			BusinessUnitID: g.Refs.BuID,
			OrganizationID: g.Refs.OrgID,
			Status:         domaintypes.StatusActive,
			Code:           volumegen.UniqueCode(volumegen.CodeFrom(p.name, 4), codes),
			Name:           p.name,
			AddressLine1:   g.Fake.Street(),
			City:           city.Name,
			StateID:        g.Refs.StateIDs[city.State],
			PostalCode:     city.Zip,
			Email:          "funding@" + slug + ".example.com",
			PhoneNumber:    g.Fake.Phone(),
			RemittanceInstructions: fmt.Sprintf(
				"Remit payment to %s, PO Box %d, %s, %s %s. ACH routing %s, account ending %s. "+
					"Reference the invoice number on every payment.",
				p.name, g.Fake.Between(1000, 99999), city.Name, city.State, city.Zip,
				g.Fake.Digits(9), g.Fake.Digits(4),
			),
			NoticeOfAssignment: fmt.Sprintf(
				"The accounts receivable of %s have been sold and assigned to %s. "+
					"Payment of this invoice must be made only to %s at the remittance address shown.",
				g.Refs.OrgName, p.name, p.name,
			),
			NonRecourse: p.recourse == factoring.RecourseNone,
		}
		effective := created + g.Fake.Between64(10, 40)*volumegen.Day
		agreement := &factoring.Agreement{
			ID:                     volumegen.IDAt("facagr_", effective),
			OrganizationID:         g.Refs.OrgID,
			BusinessUnitID:         g.Refs.BuID,
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
			ActivatedAt:            volumegen.Ptr(effective),
		}
		for _, tier := range []struct {
			from int
			to   *int
			rate int
		}{
			{from: 0, to: volumegen.Ptr(30), rate: 250},
			{from: 30, to: volumegen.Ptr(45), rate: 325},
			{from: 45, to: volumegen.Ptr(75), rate: 425},
			{from: 75, rate: 600},
		} {
			fee := &factoring.FeeTier{
				ID:             pulid.MustNew("facfee_"),
				OrganizationID: g.Refs.OrgID,
				BusinessUnitID: g.Refs.BuID,
				AgreementID:    agreement.ID,
				DaysFrom:       tier.from,
				DaysTo:         tier.to,
				RateBP:         tier.rate,
			}
			agreement.FeeTiers = append(agreement.FeeTiers, fee)
			newTiers = append(newTiers, fee)
		}
		transport := &factoringsubmission.Transport{
			ID:                 volumegen.IDAt("factrn_", effective),
			OrganizationID:     g.Refs.OrgID,
			BusinessUnitID:     g.Refs.BuID,
			FactoringCompanyID: company.ID,
			Method:             factoringsubmission.TransportMethodEmail,
			EmailRecipients:    []string{"schedules@" + slug + ".example.com"},
			AttachPackets:      true,
			LastTestedAt:       volumegen.Ptr(effective + volumegen.Hour),
		}

		newCompanies = append(newCompanies, company)
		newAgreements = append(newAgreements, agreement)
		newTransports = append(newTransports, transport)
		companyStamps = append(
			companyStamps,
			volumegen.Stamp{Id: company.ID, Created: created, Updated: created},
		)
		agreementStamps = append(agreementStamps, volumegen.Stamp{
			Id: agreement.ID, Created: created + volumegen.Day, Updated: effective,
		})
		transportStamps = append(transportStamps, volumegen.Stamp{
			Id: transport.ID, Created: effective, Updated: effective + volumegen.Hour,
		})
		factors = append(factors, &volumegen.Factor{
			Company:   company,
			Agreement: agreement,
			Transport: transport,
			Noas:      make(map[pulid.ID]*factoring.Noa),
			Style:     p.style,
			Slug:      slug,
		})
	}

	steps := []struct {
		table string
		rows  int
		run   func() error
		marks []volumegen.Stamp
	}{
		{"factoring_companies", len(newCompanies), func() error {
			return volumegen.InsertRows(
				ctx,
				tx,
				factoringcompanydb.FactoringCompanies.Handle,
				newCompanies,
			)
		}, companyStamps},
		{"factoring_agreements", len(newAgreements), func() error {
			return volumegen.InsertRows(ctx, tx, factoringdb.Agreements.Handle, newAgreements)
		}, agreementStamps},
		{"factoring_fee_tiers", len(newTiers), func() error {
			return volumegen.InsertRows(ctx, tx, factoringdb.FeeTiers.Handle, newTiers)
		}, nil},
		{"factoring_transports", len(newTransports), func() error {
			return volumegen.InsertRows(
				ctx,
				tx,
				factoringsubmissiondb.Transports.Handle,
				newTransports,
			)
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
			if err := volumegen.BackdateTable(ctx, tx, step.table, step.marks); err != nil {
				return nil, err
			}
		}
	}
	g.Report.Add("Factoring companies", len(newCompanies))
	g.Report.Add("Factoring agreements", len(newAgreements))

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
func (g *factorPhase) facAssign(ctx context.Context, tx *keel.DB, st *volumegen.FactorState) error {
	candidates := make([]*volumegen.CustomerRecord, 0, len(g.Customers))
	weights := make([]int, 0, len(g.Customers))
	for i, c := range g.Customers {
		if i < 2 || c.Weight <= 1 {
			continue
		}
		candidates = append(candidates, c)
		weights = append(weights, c.Weight)
	}
	if len(candidates) == 0 {
		return nil
	}
	n := volumegen.Clamp(len(g.Customers)*8/100, min(3, len(candidates)), min(60, len(candidates)))

	chosen := make([]*volumegen.CustomerRecord, 0, n)
	used := make(map[pulid.ID]bool, n)
	for tries := 0; len(chosen) < n && tries < n*20; tries++ {
		c := candidates[g.Fake.Weighted(weights)]
		if used[c.ID] {
			continue
		}
		used[c.ID] = true
		chosen = append(chosen, c)
	}

	start, now := volumegen.Unix(g.Start), volumegen.Unix(g.Now)
	span := now - start
	var noas []*factoring.Noa
	var docs []*document.Document
	var noaStamps []volumegen.Stamp
	profileCustomers := make([]string, 0, len(chosen))
	profileFactors := make([]string, 0, len(chosen))

	for i, c := range chosen {
		kind := facPlanFull
		switch {
		case i == 0 && len(chosen) >= 5 && len(st.Factors) > 1:
			kind = facPlanSwitched
		case i == 1 && len(chosen) >= 5:
			kind = facPlanDraftNotice
		case i == 2 && len(chosen) >= 5:
			kind = facPlanSentNotice
		case i == 3 && len(chosen) >= 6:
			kind = facPlanExpiring
		case g.Fake.Chance(0.25):
			kind = facPlanMidway
		}

		factor := st.Factors[i%len(st.Factors)]
		switch kind {
		case facPlanSwitched:
			previous := st.Factors[0]
			factor = st.Factors[1]
			switchAt := start + int64(float64(span)*(0.35+g.Fake.Float()*0.2))
			first := &volumegen.FactorPeriod{
				Factor: previous,
				From:   start - g.Fake.Between64(60, 300)*volumegen.Day,
				To:     switchAt,
			}
			second := &volumegen.FactorPeriod{Factor: factor, From: switchAt}
			c.Factoring = []*volumegen.FactorPeriod{first, second}

			old := g.facNoa(previous, c, factoring.NoaStatusAcknowledged, first.From, &docs)
			released := switchAt - g.Fake.Between64(1, 3)*volumegen.Day
			old.Status = factoring.NoaStatusReleased
			old.ReleasedAt = volumegen.Ptr(released)
			old.ReleaseReason = "Release letter from " + previous.Company.Name +
				" received; the customer moved to " + factor.Company.Name
			noas = append(noas, old)
			noaStamps = append(
				noaStamps,
				volumegen.Stamp{Id: old.ID, Created: old.CreatedAt, Updated: released},
			)
			previous.Noas[c.ID] = old

			next := g.facNoa(factor, c, factoring.NoaStatusAcknowledged, switchAt, &docs)
			noas = append(noas, next)
			noaStamps = append(
				noaStamps,
				volumegen.Stamp{Id: next.ID, Created: next.CreatedAt, Updated: next.UpdatedAt},
			)
			factor.Noas[c.ID] = next
		default:
			from := start - g.Fake.Between64(30, 300)*volumegen.Day
			status := factoring.NoaStatusAcknowledged
			switch kind {
			case facPlanMidway:
				from = start + int64(float64(span)*(0.2+g.Fake.Float()*0.5))
			case facPlanDraftNotice:
				from = now - g.Fake.Between64(1, 4)*volumegen.Day
				status = factoring.NoaStatusDraft
			case facPlanSentNotice:
				from = now - g.Fake.Between64(6, 18)*volumegen.Day
				status = factoring.NoaStatusSent
			case facPlanFull, facPlanExpiring, facPlanSwitched:
			}
			c.Factoring = []*volumegen.FactorPeriod{{Factor: factor, From: from}}
			noa := g.facNoa(factor, c, status, from, &docs)
			if kind == facPlanExpiring && noa.AcknowledgedAt != nil {
				noa.ExpiresAt = volumegen.Ptr(now + g.Fake.Between64(8, 25)*volumegen.Day)
				noa.ExpiryWarnedAt = volumegen.Ptr(now - g.Fake.Between64(1, 4)*volumegen.Day)
				noa.UpdatedAt = *noa.ExpiryWarnedAt
			}
			noas = append(noas, noa)
			noaStamps = append(
				noaStamps,
				volumegen.Stamp{Id: noa.ID, Created: noa.CreatedAt, Updated: noa.UpdatedAt},
			)
			factor.Noas[c.ID] = noa
		}

		current := c.Factoring[len(c.Factoring)-1].Factor
		profileCustomers = append(profileCustomers, c.ID.String())
		profileFactors = append(profileFactors, current.Company.ID.String())
	}

	if len(docs) > 0 {
		docStamps := make([]volumegen.Stamp, 0, len(docs))
		for _, d := range docs {
			docStamps = append(
				docStamps,
				volumegen.Stamp{Id: d.ID, Created: d.CreatedAt, Updated: d.CreatedAt},
			)
		}
		if err := volumegen.InsertRows(ctx, tx, documentdb.Documents.Handle, docs); err != nil {
			return fmt.Errorf("insert signed notice documents: %w", err)
		}
		if err := volumegen.Backdate(ctx, tx, documentdb.Documents.Handle, docStamps); err != nil {
			return err
		}
	}
	if err := volumegen.InsertRows(ctx, tx, factoringdb.Noas.Handle, noas); err != nil {
		return fmt.Errorf("insert notices of assignment: %w", err)
	}
	if err := volumegen.Backdate(ctx, tx, factoringdb.Noas.Handle, noaStamps); err != nil {
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
			volumegen.KeepUpdated(cbp.Handle),
		); err != nil {
		return fmt.Errorf("assign factoring companies to billing profiles: %w", err)
	}

	g.Report.Add("Factored customers", len(chosen))
	g.Report.Add("Notices of assignment", len(noas))
	g.Report.Add("Signed notice documents", len(docs))
	return nil
}

// facNoa drafts a notice for the customer's factoring period starting at from
// and walks it to status: sent before the period, and for an acknowledged
// notice signed and returned with the signed copy attached, as the app
// requires before it records an acknowledgement.
func (g *factorPhase) facNoa(
	f *volumegen.Factor,
	c *volumegen.CustomerRecord,
	status factoring.NoaStatus,
	from int64,
	docs *[]*document.Document,
) *factoring.Noa {
	now := volumegen.Unix(g.Now)
	created := min(from-g.Fake.Between64(12, 25)*volumegen.Day, now-2*volumegen.Hour)
	if status == factoring.NoaStatusDraft {
		created = min(from+g.Fake.Between64(2, 20)*volumegen.Hour, now-volumegen.Hour)
	}
	noa := &factoring.Noa{
		ID:                 volumegen.IDAt("facnoa_", created),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
		CustomerID:         c.ID,
		FactoringCompanyID: f.Company.ID,
		Status:             factoring.NoaStatusDraft,
		CreatedAt:          created,
		UpdatedAt:          created,
	}
	if status == factoring.NoaStatusDraft {
		return noa
	}

	sentAt := min(created+g.Fake.Between64(2*volumegen.Hour, 3*volumegen.Day), now-volumegen.Hour)
	recipient := c.BillingEmail
	if recipient == "" {
		recipient = "accounts.payable@" + volumegen.Slug(c.Name) + ".example.com"
	}
	noa.Status = factoring.NoaStatusSent
	noa.SentAt = volumegen.Ptr(sentAt)
	noa.SentTo = recipient
	noa.UpdatedAt = sentAt
	if status == factoring.NoaStatusSent {
		return noa
	}

	acknowledged := min(sentAt+g.Fake.Between64(2, 9)*volumegen.Day, from-volumegen.Hour)
	acknowledged = max(acknowledged, sentAt+volumegen.Hour)
	uploader := g.PickUser(g.Refs.UserIDs)
	fileName := fmt.Sprintf("NOA-%s-%s-signed.pdf", c.Code, f.Company.Code)
	doc := g.DocumentRow(factoring.NoaDocumentResourceType, noa.ID, fileName, "application/pdf",
		document.ProcessingProfileNone, acknowledged+volumegen.Hour, uploader,
		g.Fake.Between64(90_000, 420_000))
	doc.Description = "Signed notice of assignment returned by " + c.Name
	*docs = append(*docs, doc)

	noa.Status = factoring.NoaStatusAcknowledged
	noa.AcknowledgedAt = volumegen.Ptr(acknowledged)
	noa.SignedDocumentID = doc.ID
	noa.UpdatedAt = acknowledged + volumegen.Hour
	if g.Fake.Chance(0.3) {
		noa.ExpiresAt = volumegen.Ptr(acknowledged + g.Fake.Between64(540, 900)*volumegen.Day)
	}
	return noa
}

type factorPhase struct {
	*volumegen.Generator
}

func Seed(ctx context.Context, tx *keel.DB, g *volumegen.Generator) error {
	return (&factorPhase{Generator: g}).facMasterData(ctx, tx)
}
