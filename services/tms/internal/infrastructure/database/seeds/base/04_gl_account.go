package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type GLAccountSeed struct {
	seedhelpers.BaseSeed
}

func NewGLAccountSeed() *GLAccountSeed {
	seed := &GLAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"GLAccount",
		"1.0.0",
		"Creates default account types and chart of accounts for trucking operations",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *GLAccountSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get default organization: %w", err)
				}
			}

			count, err := tx.NewSelect().
				Model((*accounttype.AccountType)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing account types: %w", err)
			}

			if count > 0 {
				return nil
			}

			accountTypeIDs, err := s.createDefaultAccountTypes(
				ctx,
				tx,
				sc,
				org.ID,
				org.BusinessUnitID,
			)
			if err != nil {
				return fmt.Errorf("create default account types: %w", err)
			}

			accountCount, err := s.createDefaultCOA(
				ctx,
				tx,
				sc,
				org.ID,
				org.BusinessUnitID,
				accountTypeIDs,
			)
			if err != nil {
				return fmt.Errorf("create default COA: %w", err)
			}
			if err = s.applyAccountingDefaults(ctx, tx, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("apply accounting control defaults: %w", err)
			}

			seedhelpers.LogSuccess(
				"Created GL account fixtures",
				"- Created 6 default account types",
				fmt.Sprintf("- Created %d GL accounts for trucking operations", accountCount),
			)

			return nil
		},
	)
}

func (s *GLAccountSeed) applyAccountingDefaults(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) error {
	type accountRow struct {
		ID   pulid.ID `bun:"id"`
		Code string   `bun:"account_code"`
	}

	rows := make([]accountRow, 0, 2)
	if err := tx.NewSelect().
		Model((*glaccount.GLAccount)(nil)).
		Column("id", "account_code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("account_code IN (?)", bun.In([]string{"1110", "6940"})).
		Scan(ctx, &rows); err != nil {
		return err
	}

	var arAccountID pulid.ID
	var writeOffAccountID pulid.ID
	for i := range rows {
		switch rows[i].Code {
		case "1110":
			arAccountID = rows[i].ID
		case "6940":
			writeOffAccountID = rows[i].ID
		}
	}
	if arAccountID.IsNil() || writeOffAccountID.IsNil() {
		return fmt.Errorf("required accounting default accounts were not created")
	}

	_, err := tx.NewUpdate().
		Model((*tenant.AccountingControl)(nil)).
		Set("default_ar_account_id = ?", arAccountID).
		Set("default_write_off_account_id = ?", writeOffAccountID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	return err
}

func (s *GLAccountSeed) createDefaultAccountTypes(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) (map[accounttype.Category]pulid.ID, error) {
	accountTypes := []accounttype.AccountType{
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Code:           "ASSET",
			Name:           "Assets",
			Category:       accounttype.CategoryAsset,
			Color:          "#3b82f6",
			Description:    "Resources owned by the company",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Code:           "LIAB",
			Name:           "Liabilities",
			Category:       accounttype.CategoryLiability,
			Color:          "#ef4444",
			Description:    "Obligations owed by the company",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Code:           "EQUITY",
			Name:           "Equity",
			Category:       accounttype.CategoryEquity,
			Color:          "#8b5cf6",
			Description:    "Owner's stake in the company",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Code:           "REV",
			Name:           "Revenue",
			Category:       accounttype.CategoryRevenue,
			Color:          "#10b981",
			Description:    "Income from operations",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Code:           "COR",
			Name:           "Cost of Revenue",
			Category:       accounttype.CategoryCostOfRevenue,
			Color:          "#f59e0b",
			Description:    "Direct costs of providing service",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			Code:           "EXP",
			Name:           "Expenses",
			Category:       accounttype.CategoryExpense,
			Color:          "#ec4899",
			Description:    "Operating expenses",
			IsSystem:       true,
		},
	}

	_, err := tx.NewInsert().
		Model(&accountTypes).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("insert account types: %w", err)
	}

	accountTypeMap := make(map[accounttype.Category]pulid.ID)
	for i := range accountTypes {
		at := &accountTypes[i]
		accountTypeMap[at.Category] = at.ID

		if err = sc.TrackCreated(ctx, "account_types", at.ID, s.Name()); err != nil {
			return nil, fmt.Errorf("track account type: %w", err)
		}
	}

	return accountTypeMap, nil
}

type glAccountSeedData struct {
	Code        string
	Name        string
	Description string
	Category    accounttype.Category
	Parent      string
	IsSystem    bool
}

func getDefaultTruckingCOA() []glAccountSeedData {
	return []glAccountSeedData{
		// ASSETS (1000-1999)
		{
			Code:        "1000",
			Name:        "Cash and Cash Equivalents",
			Category:    accounttype.CategoryAsset,
			Description: "Liquid assets including bank accounts and cash on hand",
			IsSystem:    true,
		},
		{
			Code:        "1010",
			Name:        "Operating Cash",
			Category:    accounttype.CategoryAsset,
			Description: "Primary operating bank account",
			Parent:      "1000",
		},
		{
			Code:        "1020",
			Name:        "Payroll Cash",
			Category:    accounttype.CategoryAsset,
			Description: "Dedicated payroll bank account",
			Parent:      "1000",
		},
		{
			Code:        "1030",
			Name:        "Fuel Card Cash",
			Category:    accounttype.CategoryAsset,
			Description: "Fuel card deposits and balances",
			Parent:      "1000",
		},

		{
			Code:        "1100",
			Name:        "Accounts Receivable",
			Category:    accounttype.CategoryAsset,
			Description: "Money owed to the company by customers",
			IsSystem:    true,
		},
		{
			Code:        "1110",
			Name:        "AR - Trade",
			Category:    accounttype.CategoryAsset,
			Description: "Standard freight receivables",
			Parent:      "1100",
		},
		{
			Code:        "1120",
			Name:        "AR - Fuel Surcharge",
			Category:    accounttype.CategoryAsset,
			Description: "Fuel surcharge receivables",
			Parent:      "1100",
		},
		{
			Code:        "1130",
			Name:        "AR - Accessorial",
			Category:    accounttype.CategoryAsset,
			Description: "Accessorial charge receivables",
			Parent:      "1100",
		},
		{
			Code:        "1190",
			Name:        "Allowance for Doubtful Accounts",
			Category:    accounttype.CategoryAsset,
			Description: "Reserve for uncollectible receivables",
			Parent:      "1100",
		},

		{
			Code:        "1200",
			Name:        "Prepaid Expenses",
			Category:    accounttype.CategoryAsset,
			Description: "Expenses paid in advance",
			IsSystem:    true,
		},
		{
			Code:        "1210",
			Name:        "Prepaid Insurance",
			Category:    accounttype.CategoryAsset,
			Description: "Insurance premiums paid in advance",
			Parent:      "1200",
		},
		{
			Code:        "1220",
			Name:        "Prepaid Licenses",
			Category:    accounttype.CategoryAsset,
			Description: "License fees paid in advance",
			Parent:      "1200",
		},
		{
			Code:        "1230",
			Name:        "Prepaid Fuel",
			Category:    accounttype.CategoryAsset,
			Description: "Fuel purchased in advance",
			Parent:      "1200",
		},

		{
			Code:        "1500",
			Name:        "Fixed Assets",
			Category:    accounttype.CategoryAsset,
			Description: "Long-term tangible assets",
			IsSystem:    true,
		},
		{
			Code:        "1510",
			Name:        "Tractors",
			Category:    accounttype.CategoryAsset,
			Description: "Tractor units",
			Parent:      "1500",
		},
		{
			Code:        "1520",
			Name:        "Trailers",
			Category:    accounttype.CategoryAsset,
			Description: "Trailer units",
			Parent:      "1500",
		},
		{
			Code:        "1530",
			Name:        "Service Vehicles",
			Category:    accounttype.CategoryAsset,
			Description: "Maintenance and support vehicles",
			Parent:      "1500",
		},
		{
			Code:        "1540",
			Name:        "Office Equipment",
			Category:    accounttype.CategoryAsset,
			Description: "Office furniture and equipment",
			Parent:      "1500",
		},
		{
			Code:        "1550",
			Name:        "Computer Equipment",
			Category:    accounttype.CategoryAsset,
			Description: "Computers and IT equipment",
			Parent:      "1500",
		},
		{
			Code:        "1560",
			Name:        "Buildings",
			Category:    accounttype.CategoryAsset,
			Description: "Owned real estate",
			Parent:      "1500",
		},
		{
			Code:        "1570",
			Name:        "Land",
			Category:    accounttype.CategoryAsset,
			Description: "Owned land",
			Parent:      "1500",
		},

		{
			Code:        "1600",
			Name:        "Accumulated Depreciation",
			Category:    accounttype.CategoryAsset,
			Description: "Accumulated depreciation on fixed assets",
			IsSystem:    true,
		},
		{
			Code:        "1610",
			Name:        "Accumulated Depreciation - Tractors",
			Category:    accounttype.CategoryAsset,
			Description: "Depreciation on tractors",
			Parent:      "1600",
		},
		{
			Code:        "1620",
			Name:        "Accumulated Depreciation - Trailers",
			Category:    accounttype.CategoryAsset,
			Description: "Depreciation on trailers",
			Parent:      "1600",
		},
		{
			Code:        "1630",
			Name:        "Accumulated Depreciation - Service Vehicles",
			Category:    accounttype.CategoryAsset,
			Description: "Depreciation on service vehicles",
			Parent:      "1600",
		},
		{
			Code:        "1640",
			Name:        "Accumulated Depreciation - Office Equipment",
			Category:    accounttype.CategoryAsset,
			Description: "Depreciation on office equipment",
			Parent:      "1600",
		},
		{
			Code:        "1650",
			Name:        "Accumulated Depreciation - Computer Equipment",
			Category:    accounttype.CategoryAsset,
			Description: "Depreciation on computer equipment",
			Parent:      "1600",
		},
		{
			Code:        "1660",
			Name:        "Accumulated Depreciation - Buildings",
			Category:    accounttype.CategoryAsset,
			Description: "Depreciation on buildings",
			Parent:      "1600",
		},

		// LIABILITIES (2000-2999)
		{
			Code:        "2000",
			Name:        "Accounts Payable",
			Category:    accounttype.CategoryLiability,
			Description: "Money owed to vendors and suppliers",
			IsSystem:    true,
		},
		{
			Code:        "2010",
			Name:        "AP - Trade",
			Category:    accounttype.CategoryLiability,
			Description: "Standard vendor payables",
			Parent:      "2000",
		},
		{
			Code:        "2020",
			Name:        "AP - Fuel",
			Category:    accounttype.CategoryLiability,
			Description: "Fuel vendor payables",
			Parent:      "2000",
		},
		{
			Code:        "2030",
			Name:        "AP - Maintenance",
			Category:    accounttype.CategoryLiability,
			Description: "Maintenance and repair payables",
			Parent:      "2000",
		},

		{
			Code:        "2100",
			Name:        "Accrued Expenses",
			Category:    accounttype.CategoryLiability,
			Description: "Expenses incurred but not yet paid",
			IsSystem:    true,
		},
		{
			Code:        "2110",
			Name:        "Accrued Payroll",
			Category:    accounttype.CategoryLiability,
			Description: "Wages earned but not yet paid",
			Parent:      "2100",
		},
		{
			Code:        "2120",
			Name:        "Accrued Driver Settlement",
			Category:    accounttype.CategoryLiability,
			Description: "Driver pay earned but not settled",
			Parent:      "2100",
		},
		{
			Code:        "2130",
			Name:        "Accrued Taxes",
			Category:    accounttype.CategoryLiability,
			Description: "Taxes owed but not yet paid",
			Parent:      "2100",
		},
		{
			Code:        "2140",
			Name:        "Accrued Insurance",
			Category:    accounttype.CategoryLiability,
			Description: "Insurance premiums owed",
			Parent:      "2100",
		},

		{
			Code:        "2200",
			Name:        "Payroll Liabilities",
			Category:    accounttype.CategoryLiability,
			Description: "Employee-related payables",
			IsSystem:    true,
		},
		{
			Code:        "2210",
			Name:        "Federal Income Tax Withheld",
			Category:    accounttype.CategoryLiability,
			Description: "Federal income tax withheld from employees",
			Parent:      "2200",
		},
		{
			Code:        "2220",
			Name:        "State Income Tax Withheld",
			Category:    accounttype.CategoryLiability,
			Description: "State income tax withheld from employees",
			Parent:      "2200",
		},
		{
			Code:        "2230",
			Name:        "FICA Tax Payable",
			Category:    accounttype.CategoryLiability,
			Description: "Social Security and Medicare taxes payable",
			Parent:      "2200",
		},
		{
			Code:        "2240",
			Name:        "401(k) Contributions Payable",
			Category:    accounttype.CategoryLiability,
			Description: "Employee 401(k) contributions payable",
			Parent:      "2200",
		},
		{
			Code:        "2250",
			Name:        "Health Insurance Payable",
			Category:    accounttype.CategoryLiability,
			Description: "Employee health insurance premiums payable",
			Parent:      "2200",
		},

		{
			Code:        "2300",
			Name:        "Sales Tax Payable",
			Category:    accounttype.CategoryLiability,
			Description: "Sales tax collected and owed",
			IsSystem:    true,
		},

		{
			Code:        "2400",
			Name:        "Short-Term Notes Payable",
			Category:    accounttype.CategoryLiability,
			Description: "Notes due within one year",
			IsSystem:    true,
		},
		{
			Code:        "2410",
			Name:        "Equipment Loans - Current",
			Category:    accounttype.CategoryLiability,
			Description: "Current portion of equipment loans",
			Parent:      "2400",
		},
		{
			Code:        "2420",
			Name:        "Line of Credit",
			Category:    accounttype.CategoryLiability,
			Description: "Operating line of credit",
			Parent:      "2400",
		},

		{
			Code:        "2500",
			Name:        "Long-Term Notes Payable",
			Category:    accounttype.CategoryLiability,
			Description: "Notes due after one year",
			IsSystem:    true,
		},
		{
			Code:        "2510",
			Name:        "Equipment Loans - Long Term",
			Category:    accounttype.CategoryLiability,
			Description: "Long-term portion of equipment loans",
			Parent:      "2500",
		},
		{
			Code:        "2520",
			Name:        "Real Estate Loans",
			Category:    accounttype.CategoryLiability,
			Description: "Mortgages on real property",
			Parent:      "2500",
		},

		// EQUITY (3000-3999)
		{
			Code:        "3000",
			Name:        "Owner's Equity",
			Category:    accounttype.CategoryEquity,
			Description: "Owner's investment and retained earnings",
			IsSystem:    true,
		},
		{
			Code:        "3010",
			Name:        "Owner's Capital",
			Category:    accounttype.CategoryEquity,
			Description: "Initial and additional owner investments",
			Parent:      "3000",
		},
		{
			Code:        "3020",
			Name:        "Owner's Draws",
			Category:    accounttype.CategoryEquity,
			Description: "Distributions to owners",
			Parent:      "3000",
		},
		{
			Code:        "3030",
			Name:        "Retained Earnings",
			Category:    accounttype.CategoryEquity,
			Description: "Accumulated profits retained in business",
			Parent:      "3000",
		},
		{
			Code:        "3040",
			Name:        "Current Year Earnings",
			Category:    accounttype.CategoryEquity,
			Description: "Profit or loss for current year",
			Parent:      "3000",
		},

		// REVENUE (4000-4999)
		{
			Code:        "4000",
			Name:        "Freight Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Income from freight operations",
			IsSystem:    true,
		},
		{
			Code:        "4010",
			Name:        "Linehaul Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Base freight revenue",
			Parent:      "4000",
		},
		{
			Code:        "4020",
			Name:        "Fuel Surcharge Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Fuel surcharge income",
			Parent:      "4000",
		},
		{
			Code:        "4030",
			Name:        "Accessorial Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Additional service charges",
			Parent:      "4000",
		},
		{
			Code:        "4031",
			Name:        "Detention Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Detention time charges",
			Parent:      "4030",
		},
		{
			Code:        "4032",
			Name:        "Layover Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Layover charges",
			Parent:      "4030",
		},
		{
			Code:        "4033",
			Name:        "Stop-Off Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Additional stop charges",
			Parent:      "4030",
		},
		{
			Code:        "4034",
			Name:        "Loading/Unloading Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Driver assist charges",
			Parent:      "4030",
		},

		{
			Code:        "4100",
			Name:        "Other Operating Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Other revenue from operations",
			IsSystem:    true,
		},
		{
			Code:        "4110",
			Name:        "Brokerage Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Revenue from brokered loads",
			Parent:      "4100",
		},
		{
			Code:        "4120",
			Name:        "Warehouse Revenue",
			Category:    accounttype.CategoryRevenue,
			Description: "Storage and warehouse fees",
			Parent:      "4100",
		},

		// COST OF REVENUE (5000-5999)
		{
			Code:        "5000",
			Name:        "Driver Costs",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Direct driver compensation",
			IsSystem:    true,
		},
		{
			Code:        "5010",
			Name:        "Driver Wages - Company",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Company driver wages",
			Parent:      "5000",
		},
		{
			Code:        "5020",
			Name:        "Driver Wages - Owner Operator",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Owner operator settlements",
			Parent:      "5000",
		},
		{
			Code:        "5030",
			Name:        "Driver Bonuses",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Performance and safety bonuses",
			Parent:      "5000",
		},
		{
			Code:        "5040",
			Name:        "Driver Benefits",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Driver health and benefits",
			Parent:      "5000",
		},

		{
			Code:        "5100",
			Name:        "Fuel Costs",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Fuel and related expenses",
			IsSystem:    true,
		},
		{
			Code:        "5110",
			Name:        "Diesel Fuel",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Diesel fuel purchases",
			Parent:      "5100",
		},
		{
			Code:        "5120",
			Name:        "DEF (Diesel Exhaust Fluid)",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "DEF purchases",
			Parent:      "5100",
		},
		{
			Code:        "5130",
			Name:        "Fuel Taxes",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "IFTA and fuel taxes",
			Parent:      "5100",
		},

		{
			Code:        "5200",
			Name:        "Vehicle Maintenance",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Maintenance and repairs",
			IsSystem:    true,
		},
		{
			Code:        "5210",
			Name:        "Preventive Maintenance",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Scheduled maintenance",
			Parent:      "5200",
		},
		{
			Code:        "5220",
			Name:        "Repairs - Tractors",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Tractor repairs",
			Parent:      "5200",
		},
		{
			Code:        "5230",
			Name:        "Repairs - Trailers",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Trailer repairs",
			Parent:      "5200",
		},
		{
			Code:        "5240",
			Name:        "Tires",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Tire purchases and repairs",
			Parent:      "5200",
		},
		{
			Code:        "5250",
			Name:        "Parts and Supplies",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Maintenance parts and supplies",
			Parent:      "5200",
		},

		{
			Code:        "5300",
			Name:        "Insurance Costs",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Operating insurance",
			IsSystem:    true,
		},
		{
			Code:        "5310",
			Name:        "Liability Insurance",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "General and auto liability",
			Parent:      "5300",
		},
		{
			Code:        "5320",
			Name:        "Cargo Insurance",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Cargo coverage",
			Parent:      "5300",
		},
		{
			Code:        "5330",
			Name:        "Physical Damage Insurance",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Vehicle physical damage",
			Parent:      "5300",
		},
		{
			Code:        "5340",
			Name:        "Workers Compensation",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Workers comp insurance",
			Parent:      "5300",
		},

		{
			Code:        "5400",
			Name:        "Permits and Licenses",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Operating permits and licenses",
			IsSystem:    true,
		},
		{
			Code:        "5410",
			Name:        "Vehicle Registrations",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Vehicle registration fees",
			Parent:      "5400",
		},
		{
			Code:        "5420",
			Name:        "IRP Fees",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "International Registration Plan fees",
			Parent:      "5400",
		},
		{
			Code:        "5430",
			Name:        "UCR Fees",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Unified Carrier Registration",
			Parent:      "5400",
		},
		{
			Code:        "5440",
			Name:        "Oversize/Overweight Permits",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Special permits",
			Parent:      "5400",
		},

		{
			Code:        "5500",
			Name:        "Tolls and Road Fees",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Toll roads and fees",
			IsSystem:    true,
		},
		{
			Code:        "5510",
			Name:        "Highway Tolls",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Toll road charges",
			Parent:      "5500",
		},
		{
			Code:        "5520",
			Name:        "Scale Fees",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Weigh station fees",
			Parent:      "5500",
		},

		{
			Code:        "5600",
			Name:        "Subcontractor Costs",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Independent contractor expenses",
			IsSystem:    true,
		},
		{
			Code:        "5610",
			Name:        "Brokered Loads",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Cost of brokered loads",
			Parent:      "5600",
		},
		{
			Code:        "5620",
			Name:        "Owner Operator Lease",
			Category:    accounttype.CategoryCostOfRevenue,
			Description: "Owner operator settlements",
			Parent:      "5600",
		},

		// EXPENSES (6000-6999)
		{
			Code:        "6000",
			Name:        "Administrative Expenses",
			Category:    accounttype.CategoryExpense,
			Description: "General administrative costs",
			IsSystem:    true,
		},
		{
			Code:        "6010",
			Name:        "Salaries - Office",
			Category:    accounttype.CategoryExpense,
			Description: "Office staff salaries",
			Parent:      "6000",
		},
		{
			Code:        "6020",
			Name:        "Salaries - Management",
			Category:    accounttype.CategoryExpense,
			Description: "Management salaries",
			Parent:      "6000",
		},
		{
			Code:        "6030",
			Name:        "Payroll Taxes - Office",
			Category:    accounttype.CategoryExpense,
			Description: "Employer payroll taxes for office staff",
			Parent:      "6000",
		},
		{
			Code:        "6040",
			Name:        "Employee Benefits - Office",
			Category:    accounttype.CategoryExpense,
			Description: "Office staff benefits",
			Parent:      "6000",
		},

		{
			Code:        "6100",
			Name:        "Facility Expenses",
			Category:    accounttype.CategoryExpense,
			Description: "Facility and occupancy costs",
			IsSystem:    true,
		},
		{
			Code:        "6110",
			Name:        "Rent - Office",
			Category:    accounttype.CategoryExpense,
			Description: "Office rent",
			Parent:      "6100",
		},
		{
			Code:        "6120",
			Name:        "Rent - Yard",
			Category:    accounttype.CategoryExpense,
			Description: "Yard and parking rent",
			Parent:      "6100",
		},
		{
			Code:        "6130",
			Name:        "Utilities",
			Category:    accounttype.CategoryExpense,
			Description: "Electric, water, gas",
			Parent:      "6100",
		},
		{
			Code:        "6140",
			Name:        "Property Taxes",
			Category:    accounttype.CategoryExpense,
			Description: "Real estate taxes",
			Parent:      "6100",
		},
		{
			Code:        "6150",
			Name:        "Building Maintenance",
			Category:    accounttype.CategoryExpense,
			Description: "Building repairs and maintenance",
			Parent:      "6100",
		},
		{
			Code:        "6160",
			Name:        "Security",
			Category:    accounttype.CategoryExpense,
			Description: "Security services and systems",
			Parent:      "6100",
		},

		{
			Code:        "6200",
			Name:        "Technology Expenses",
			Category:    accounttype.CategoryExpense,
			Description: "IT and software costs",
			IsSystem:    true,
		},
		{
			Code:        "6210",
			Name:        "Software Subscriptions",
			Category:    accounttype.CategoryExpense,
			Description: "SaaS and software licenses",
			Parent:      "6200",
		},
		{
			Code:        "6220",
			Name:        "IT Support",
			Category:    accounttype.CategoryExpense,
			Description: "IT consulting and support",
			Parent:      "6200",
		},
		{
			Code:        "6230",
			Name:        "Internet and Phone",
			Category:    accounttype.CategoryExpense,
			Description: "Communication services",
			Parent:      "6200",
		},
		{
			Code:        "6240",
			Name:        "Computer Equipment",
			Category:    accounttype.CategoryExpense,
			Description: "Computer purchases under capitalization threshold",
			Parent:      "6200",
		},

		{
			Code:        "6300",
			Name:        "Professional Services",
			Category:    accounttype.CategoryExpense,
			Description: "Professional fees",
			IsSystem:    true,
		},
		{
			Code:        "6310",
			Name:        "Legal Fees",
			Category:    accounttype.CategoryExpense,
			Description: "Attorney fees",
			Parent:      "6300",
		},
		{
			Code:        "6320",
			Name:        "Accounting Fees",
			Category:    accounttype.CategoryExpense,
			Description: "Accounting and bookkeeping",
			Parent:      "6300",
		},
		{
			Code:        "6330",
			Name:        "Consulting Fees",
			Category:    accounttype.CategoryExpense,
			Description: "Business consulting",
			Parent:      "6300",
		},
		{
			Code:        "6340",
			Name:        "Audit Fees",
			Category:    accounttype.CategoryExpense,
			Description: "Financial audit fees",
			Parent:      "6300",
		},

		{
			Code:        "6400",
			Name:        "Marketing and Sales",
			Category:    accounttype.CategoryExpense,
			Description: "Marketing and business development",
			IsSystem:    true,
		},
		{
			Code:        "6410",
			Name:        "Advertising",
			Category:    accounttype.CategoryExpense,
			Description: "Advertising expenses",
			Parent:      "6400",
		},
		{
			Code:        "6420",
			Name:        "Website and Digital Marketing",
			Category:    accounttype.CategoryExpense,
			Description: "Online marketing",
			Parent:      "6400",
		},
		{
			Code:        "6430",
			Name:        "Trade Shows",
			Category:    accounttype.CategoryExpense,
			Description: "Trade show expenses",
			Parent:      "6400",
		},
		{
			Code:        "6440",
			Name:        "Customer Entertainment",
			Category:    accounttype.CategoryExpense,
			Description: "Client entertainment",
			Parent:      "6400",
		},

		{
			Code:        "6500",
			Name:        "Office Expenses",
			Category:    accounttype.CategoryExpense,
			Description: "Office supplies and expenses",
			IsSystem:    true,
		},
		{
			Code:        "6510",
			Name:        "Office Supplies",
			Category:    accounttype.CategoryExpense,
			Description: "General office supplies",
			Parent:      "6500",
		},
		{
			Code:        "6520",
			Name:        "Postage and Shipping",
			Category:    accounttype.CategoryExpense,
			Description: "Mailing and shipping costs",
			Parent:      "6500",
		},
		{
			Code:        "6530",
			Name:        "Printing and Copying",
			Category:    accounttype.CategoryExpense,
			Description: "Printing services",
			Parent:      "6500",
		},

		{
			Code:        "6600",
			Name:        "Travel and Entertainment",
			Category:    accounttype.CategoryExpense,
			Description: "Business travel expenses",
			IsSystem:    true,
		},
		{
			Code:        "6610",
			Name:        "Airfare",
			Category:    accounttype.CategoryExpense,
			Description: "Air travel",
			Parent:      "6600",
		},
		{
			Code:        "6620",
			Name:        "Lodging",
			Category:    accounttype.CategoryExpense,
			Description: "Hotel expenses",
			Parent:      "6600",
		},
		{
			Code:        "6630",
			Name:        "Meals",
			Category:    accounttype.CategoryExpense,
			Description: "Business meals",
			Parent:      "6600",
		},
		{
			Code:        "6640",
			Name:        "Auto Rental",
			Category:    accounttype.CategoryExpense,
			Description: "Vehicle rentals",
			Parent:      "6600",
		},

		{
			Code:        "6700",
			Name:        "Depreciation and Amortization",
			Category:    accounttype.CategoryExpense,
			Description: "Non-cash depreciation expense",
			IsSystem:    true,
		},
		{
			Code:        "6710",
			Name:        "Depreciation Expense",
			Category:    accounttype.CategoryExpense,
			Description: "Depreciation of fixed assets",
			Parent:      "6700",
		},
		{
			Code:        "6720",
			Name:        "Amortization Expense",
			Category:    accounttype.CategoryExpense,
			Description: "Amortization of intangibles",
			Parent:      "6700",
		},

		{
			Code:        "6800",
			Name:        "Interest and Bank Charges",
			Category:    accounttype.CategoryExpense,
			Description: "Financial costs",
			IsSystem:    true,
		},
		{
			Code:        "6810",
			Name:        "Interest Expense",
			Category:    accounttype.CategoryExpense,
			Description: "Loan interest",
			Parent:      "6800",
		},
		{
			Code:        "6820",
			Name:        "Bank Service Charges",
			Category:    accounttype.CategoryExpense,
			Description: "Bank fees",
			Parent:      "6800",
		},
		{
			Code:        "6830",
			Name:        "Credit Card Fees",
			Category:    accounttype.CategoryExpense,
			Description: "Merchant and processing fees",
			Parent:      "6800",
		},

		{
			Code:        "6900",
			Name:        "Other Expenses",
			Category:    accounttype.CategoryExpense,
			Description: "Miscellaneous expenses",
			IsSystem:    true,
		},
		{
			Code:        "6910",
			Name:        "Training and Education",
			Category:    accounttype.CategoryExpense,
			Description: "Employee training",
			Parent:      "6900",
		},
		{
			Code:        "6920",
			Name:        "Dues and Subscriptions",
			Category:    accounttype.CategoryExpense,
			Description: "Industry memberships",
			Parent:      "6900",
		},
		{
			Code:        "6930",
			Name:        "Charitable Contributions",
			Category:    accounttype.CategoryExpense,
			Description: "Donations",
			Parent:      "6900",
		},
		{
			Code:        "6940",
			Name:        "Bad Debt Expense",
			Category:    accounttype.CategoryExpense,
			Description: "Write-offs of uncollectible receivables",
			Parent:      "6900",
		},
		{
			Code:        "6950",
			Name:        "Penalties and Fines",
			Category:    accounttype.CategoryExpense,
			Description: "DOT and other fines",
			Parent:      "6900",
		},
	}
}

func (s *GLAccountSeed) createDefaultCOA(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
	accountTypeIDs map[accounttype.Category]pulid.ID,
) (int, error) {
	coaData := getDefaultTruckingCOA()

	accountsByCode := make(map[string]*glaccount.GLAccount)

	var accounts []*glaccount.GLAccount
	for _, seed := range coaData {
		account := &glaccount.GLAccount{
			ID:             pulid.MustNew("gla_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domaintypes.StatusActive,
			AccountTypeID:  accountTypeIDs[seed.Category],
			AccountCode:    seed.Code,
			Name:           seed.Name,
			Description:    seed.Description,
			IsSystem:       seed.IsSystem,
			AllowManualJE:  true,
			RequireProject: false,
		}
		accounts = append(accounts, account)
		accountsByCode[seed.Code] = account
	}

	_, err := tx.NewInsert().
		Model(&accounts).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("insert GL accounts: %w", err)
	}

	for _, account := range accounts {
		if err = sc.TrackCreated(ctx, "gl_accounts", account.ID, s.Name()); err != nil {
			return 0, fmt.Errorf("track GL account: %w", err)
		}
	}

	for i, seed := range coaData {
		if seed.Parent != "" {
			if parent, ok := accountsByCode[seed.Parent]; ok {
				accounts[i].ParentID = parent.ID
			}
		}
	}

	for _, account := range accounts {
		if !account.ParentID.IsNil() {
			_, err = tx.NewUpdate().
				Model(account).
				Column("parent_id").
				Where("id = ?", account.ID).
				Where("organization_id = ?", orgID).
				Where("business_unit_id = ?", buID).
				Exec(ctx)
			if err != nil {
				return 0, fmt.Errorf(
					"update parent relationship for %s: %w",
					account.AccountCode,
					err,
				)
			}
		}
	}

	return len(accounts), nil
}

func (s *GLAccountSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *GLAccountSeed) CanRollback() bool {
	return true
}
