package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// GlAccountSeed Creates gl account data
type GlAccountSeed struct {
	seedhelpers.BaseSeed
}

// NewGlAccountSeed creates a new gl_account seed
func NewGlAccountSeed() *GlAccountSeed {
	seed := &GlAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"GlAccount",
		"1.0.0",
		"Creates gl account data",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	return seed
}

// Run executes the seed
func (s *GlAccountSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			// Get default organization and business unit
			defaultOrg, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return fmt.Errorf("get default organization: %w", err)
			}

			defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return fmt.Errorf("get default business unit: %w", err)
			}

			// Create default account types and get their IDs
			accountTypeIDs, err := s.createDefaultAccountTypes(ctx, tx, defaultOrg.ID, defaultBU.ID)
			if err != nil {
				return fmt.Errorf("create default account types: %w", err)
			}

			// Create default chart of accounts
			accountCount, err := s.createDefaultCOA(
				ctx,
				tx,
				defaultOrg.ID,
				defaultBU.ID,
				accountTypeIDs,
			)
			if err != nil {
				return fmt.Errorf("create default COA: %w", err)
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

func (s *GlAccountSeed) createDefaultAccountTypes(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) (map[accounting.Category]pulid.ID, error) {
	accountTypes := []accounting.AccountType{
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domain.StatusActive,
			Code:           "ASSET",
			Name:           "Assets",
			Category:       accounting.CategoryAsset,
			Color:          "#3b82f6", // Blue
			Description:    "Resources owned by the company",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domain.StatusActive,
			Code:           "LIAB",
			Name:           "Liabilities",
			Category:       accounting.CategoryLiability,
			Color:          "#ef4444", // Red
			Description:    "Obligations owed by the company",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domain.StatusActive,
			Code:           "EQUITY",
			Name:           "Equity",
			Category:       accounting.CategoryEquity,
			Color:          "#8b5cf6", // Purple
			Description:    "Owner's stake in the company",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domain.StatusActive,
			Code:           "REV",
			Name:           "Revenue",
			Category:       accounting.CategoryRevenue,
			Color:          "#10b981", // Green
			Description:    "Income from operations",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domain.StatusActive,
			Code:           "COR",
			Name:           "Cost of Revenue",
			Category:       accounting.CategoryCostOfRevenue,
			Color:          "#f59e0b", // Amber
			Description:    "Direct costs of providing service",
			IsSystem:       true,
		},
		{
			ID:             pulid.MustNew("at_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domain.StatusActive,
			Code:           "EXP",
			Name:           "Expenses",
			Category:       accounting.CategoryExpense,
			Color:          "#ec4899", // Pink
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

	// Build a map of category to account type ID for easy lookup
	accountTypeMap := make(map[accounting.Category]pulid.ID)
	for _, at := range accountTypes {
		accountTypeMap[at.Category] = at.ID
	}

	return accountTypeMap, nil
}

// glAccountSeedData represents the seed data for a GL account
type glAccountSeedData struct {
	Code        string
	Name        string
	Description string
	Category    accounting.Category
	Parent      string // Account code of parent
	IsSystem    bool
}

// getDefaultTruckingCOA returns the complete default chart of accounts for trucking operations
func getDefaultTruckingCOA() []glAccountSeedData {
	return []glAccountSeedData{
		// ASSETS (1000-1999)
		{
			Code:        "1000",
			Name:        "Cash and Cash Equivalents",
			Category:    accounting.CategoryAsset,
			Description: "Liquid assets including bank accounts and cash on hand",
			IsSystem:    true,
		},
		{
			Code:        "1010",
			Name:        "Operating Cash",
			Category:    accounting.CategoryAsset,
			Description: "Primary operating bank account",
			Parent:      "1000",
		},
		{
			Code:        "1020",
			Name:        "Payroll Cash",
			Category:    accounting.CategoryAsset,
			Description: "Dedicated payroll bank account",
			Parent:      "1000",
		},
		{
			Code:        "1030",
			Name:        "Fuel Card Cash",
			Category:    accounting.CategoryAsset,
			Description: "Fuel card deposits and balances",
			Parent:      "1000",
		},

		{
			Code:        "1100",
			Name:        "Accounts Receivable",
			Category:    accounting.CategoryAsset,
			Description: "Money owed to the company by customers",
			IsSystem:    true,
		},
		{
			Code:        "1110",
			Name:        "AR - Trade",
			Category:    accounting.CategoryAsset,
			Description: "Standard freight receivables",
			Parent:      "1100",
		},
		{
			Code:        "1120",
			Name:        "AR - Fuel Surcharge",
			Category:    accounting.CategoryAsset,
			Description: "Fuel surcharge receivables",
			Parent:      "1100",
		},
		{
			Code:        "1130",
			Name:        "AR - Accessorial",
			Category:    accounting.CategoryAsset,
			Description: "Accessorial charge receivables",
			Parent:      "1100",
		},
		{
			Code:        "1190",
			Name:        "Allowance for Doubtful Accounts",
			Category:    accounting.CategoryAsset,
			Description: "Reserve for uncollectible receivables",
			Parent:      "1100",
		},

		{
			Code:        "1200",
			Name:        "Prepaid Expenses",
			Category:    accounting.CategoryAsset,
			Description: "Expenses paid in advance",
			IsSystem:    true,
		},
		{
			Code:        "1210",
			Name:        "Prepaid Insurance",
			Category:    accounting.CategoryAsset,
			Description: "Insurance premiums paid in advance",
			Parent:      "1200",
		},
		{
			Code:        "1220",
			Name:        "Prepaid Licenses",
			Category:    accounting.CategoryAsset,
			Description: "License fees paid in advance",
			Parent:      "1200",
		},
		{
			Code:        "1230",
			Name:        "Prepaid Fuel",
			Category:    accounting.CategoryAsset,
			Description: "Fuel purchased in advance",
			Parent:      "1200",
		},

		{
			Code:        "1500",
			Name:        "Fixed Assets",
			Category:    accounting.CategoryAsset,
			Description: "Long-term tangible assets",
			IsSystem:    true,
		},
		{
			Code:        "1510",
			Name:        "Tractors",
			Category:    accounting.CategoryAsset,
			Description: "Tractor units",
			Parent:      "1500",
		},
		{
			Code:        "1520",
			Name:        "Trailers",
			Category:    accounting.CategoryAsset,
			Description: "Trailer units",
			Parent:      "1500",
		},
		{
			Code:        "1530",
			Name:        "Service Vehicles",
			Category:    accounting.CategoryAsset,
			Description: "Maintenance and support vehicles",
			Parent:      "1500",
		},
		{
			Code:        "1540",
			Name:        "Office Equipment",
			Category:    accounting.CategoryAsset,
			Description: "Office furniture and equipment",
			Parent:      "1500",
		},
		{
			Code:        "1550",
			Name:        "Computer Equipment",
			Category:    accounting.CategoryAsset,
			Description: "Computers and IT equipment",
			Parent:      "1500",
		},
		{
			Code:        "1560",
			Name:        "Buildings",
			Category:    accounting.CategoryAsset,
			Description: "Owned real estate",
			Parent:      "1500",
		},
		{
			Code:        "1570",
			Name:        "Land",
			Category:    accounting.CategoryAsset,
			Description: "Owned land",
			Parent:      "1500",
		},

		{
			Code:        "1600",
			Name:        "Accumulated Depreciation",
			Category:    accounting.CategoryAsset,
			Description: "Accumulated depreciation on fixed assets",
			IsSystem:    true,
		},
		{
			Code:        "1610",
			Name:        "Accumulated Depreciation - Tractors",
			Category:    accounting.CategoryAsset,
			Description: "Depreciation on tractors",
			Parent:      "1600",
		},
		{
			Code:        "1620",
			Name:        "Accumulated Depreciation - Trailers",
			Category:    accounting.CategoryAsset,
			Description: "Depreciation on trailers",
			Parent:      "1600",
		},
		{
			Code:        "1630",
			Name:        "Accumulated Depreciation - Service Vehicles",
			Category:    accounting.CategoryAsset,
			Description: "Depreciation on service vehicles",
			Parent:      "1600",
		},
		{
			Code:        "1640",
			Name:        "Accumulated Depreciation - Office Equipment",
			Category:    accounting.CategoryAsset,
			Description: "Depreciation on office equipment",
			Parent:      "1600",
		},
		{
			Code:        "1650",
			Name:        "Accumulated Depreciation - Computer Equipment",
			Category:    accounting.CategoryAsset,
			Description: "Depreciation on computer equipment",
			Parent:      "1600",
		},
		{
			Code:        "1660",
			Name:        "Accumulated Depreciation - Buildings",
			Category:    accounting.CategoryAsset,
			Description: "Depreciation on buildings",
			Parent:      "1600",
		},

		// LIABILITIES (2000-2999)
		{
			Code:        "2000",
			Name:        "Accounts Payable",
			Category:    accounting.CategoryLiability,
			Description: "Money owed to vendors and suppliers",
			IsSystem:    true,
		},
		{
			Code:        "2010",
			Name:        "AP - Trade",
			Category:    accounting.CategoryLiability,
			Description: "Standard vendor payables",
			Parent:      "2000",
		},
		{
			Code:        "2020",
			Name:        "AP - Fuel",
			Category:    accounting.CategoryLiability,
			Description: "Fuel vendor payables",
			Parent:      "2000",
		},
		{
			Code:        "2030",
			Name:        "AP - Maintenance",
			Category:    accounting.CategoryLiability,
			Description: "Maintenance and repair payables",
			Parent:      "2000",
		},

		{
			Code:        "2100",
			Name:        "Accrued Expenses",
			Category:    accounting.CategoryLiability,
			Description: "Expenses incurred but not yet paid",
			IsSystem:    true,
		},
		{
			Code:        "2110",
			Name:        "Accrued Payroll",
			Category:    accounting.CategoryLiability,
			Description: "Wages earned but not yet paid",
			Parent:      "2100",
		},
		{
			Code:        "2120",
			Name:        "Accrued Driver Settlement",
			Category:    accounting.CategoryLiability,
			Description: "Driver pay earned but not settled",
			Parent:      "2100",
		},
		{
			Code:        "2130",
			Name:        "Accrued Taxes",
			Category:    accounting.CategoryLiability,
			Description: "Taxes owed but not yet paid",
			Parent:      "2100",
		},
		{
			Code:        "2140",
			Name:        "Accrued Insurance",
			Category:    accounting.CategoryLiability,
			Description: "Insurance premiums owed",
			Parent:      "2100",
		},

		{
			Code:        "2200",
			Name:        "Payroll Liabilities",
			Category:    accounting.CategoryLiability,
			Description: "Employee-related payables",
			IsSystem:    true,
		},
		{
			Code:        "2210",
			Name:        "Federal Income Tax Withheld",
			Category:    accounting.CategoryLiability,
			Description: "Federal income tax withheld from employees",
			Parent:      "2200",
		},
		{
			Code:        "2220",
			Name:        "State Income Tax Withheld",
			Category:    accounting.CategoryLiability,
			Description: "State income tax withheld from employees",
			Parent:      "2200",
		},
		{
			Code:        "2230",
			Name:        "FICA Tax Payable",
			Category:    accounting.CategoryLiability,
			Description: "Social Security and Medicare taxes payable",
			Parent:      "2200",
		},
		{
			Code:        "2240",
			Name:        "401(k) Contributions Payable",
			Category:    accounting.CategoryLiability,
			Description: "Employee 401(k) contributions payable",
			Parent:      "2200",
		},
		{
			Code:        "2250",
			Name:        "Health Insurance Payable",
			Category:    accounting.CategoryLiability,
			Description: "Employee health insurance premiums payable",
			Parent:      "2200",
		},

		{
			Code:        "2300",
			Name:        "Sales Tax Payable",
			Category:    accounting.CategoryLiability,
			Description: "Sales tax collected and owed",
			IsSystem:    true,
		},

		{
			Code:        "2400",
			Name:        "Short-Term Notes Payable",
			Category:    accounting.CategoryLiability,
			Description: "Notes due within one year",
			IsSystem:    true,
		},
		{
			Code:        "2410",
			Name:        "Equipment Loans - Current",
			Category:    accounting.CategoryLiability,
			Description: "Current portion of equipment loans",
			Parent:      "2400",
		},
		{
			Code:        "2420",
			Name:        "Line of Credit",
			Category:    accounting.CategoryLiability,
			Description: "Operating line of credit",
			Parent:      "2400",
		},

		{
			Code:        "2500",
			Name:        "Long-Term Notes Payable",
			Category:    accounting.CategoryLiability,
			Description: "Notes due after one year",
			IsSystem:    true,
		},
		{
			Code:        "2510",
			Name:        "Equipment Loans - Long Term",
			Category:    accounting.CategoryLiability,
			Description: "Long-term portion of equipment loans",
			Parent:      "2500",
		},
		{
			Code:        "2520",
			Name:        "Real Estate Loans",
			Category:    accounting.CategoryLiability,
			Description: "Mortgages on real property",
			Parent:      "2500",
		},

		// EQUITY (3000-3999)
		{
			Code:        "3000",
			Name:        "Owner's Equity",
			Category:    accounting.CategoryEquity,
			Description: "Owner's investment and retained earnings",
			IsSystem:    true,
		},
		{
			Code:        "3010",
			Name:        "Owner's Capital",
			Category:    accounting.CategoryEquity,
			Description: "Initial and additional owner investments",
			Parent:      "3000",
		},
		{
			Code:        "3020",
			Name:        "Owner's Draws",
			Category:    accounting.CategoryEquity,
			Description: "Distributions to owners",
			Parent:      "3000",
		},
		{
			Code:        "3030",
			Name:        "Retained Earnings",
			Category:    accounting.CategoryEquity,
			Description: "Accumulated profits retained in business",
			Parent:      "3000",
		},
		{
			Code:        "3040",
			Name:        "Current Year Earnings",
			Category:    accounting.CategoryEquity,
			Description: "Profit or loss for current year",
			Parent:      "3000",
		},

		// REVENUE (4000-4999)
		{
			Code:        "4000",
			Name:        "Freight Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Income from freight operations",
			IsSystem:    true,
		},
		{
			Code:        "4010",
			Name:        "Linehaul Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Base freight revenue",
			Parent:      "4000",
		},
		{
			Code:        "4020",
			Name:        "Fuel Surcharge Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Fuel surcharge income",
			Parent:      "4000",
		},
		{
			Code:        "4030",
			Name:        "Accessorial Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Additional service charges",
			Parent:      "4000",
		},
		{
			Code:        "4031",
			Name:        "Detention Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Detention time charges",
			Parent:      "4030",
		},
		{
			Code:        "4032",
			Name:        "Layover Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Layover charges",
			Parent:      "4030",
		},
		{
			Code:        "4033",
			Name:        "Stop-Off Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Additional stop charges",
			Parent:      "4030",
		},
		{
			Code:        "4034",
			Name:        "Loading/Unloading Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Driver assist charges",
			Parent:      "4030",
		},

		{
			Code:        "4100",
			Name:        "Other Operating Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Other revenue from operations",
			IsSystem:    true,
		},
		{
			Code:        "4110",
			Name:        "Brokerage Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Revenue from brokered loads",
			Parent:      "4100",
		},
		{
			Code:        "4120",
			Name:        "Warehouse Revenue",
			Category:    accounting.CategoryRevenue,
			Description: "Storage and warehouse fees",
			Parent:      "4100",
		},

		// COST OF REVENUE (5000-5999)
		{
			Code:        "5000",
			Name:        "Driver Costs",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Direct driver compensation",
			IsSystem:    true,
		},
		{
			Code:        "5010",
			Name:        "Driver Wages - Company",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Company driver wages",
			Parent:      "5000",
		},
		{
			Code:        "5020",
			Name:        "Driver Wages - Owner Operator",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Owner operator settlements",
			Parent:      "5000",
		},
		{
			Code:        "5030",
			Name:        "Driver Bonuses",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Performance and safety bonuses",
			Parent:      "5000",
		},
		{
			Code:        "5040",
			Name:        "Driver Benefits",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Driver health and benefits",
			Parent:      "5000",
		},

		{
			Code:        "5100",
			Name:        "Fuel Costs",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Fuel and related expenses",
			IsSystem:    true,
		},
		{
			Code:        "5110",
			Name:        "Diesel Fuel",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Diesel fuel purchases",
			Parent:      "5100",
		},
		{
			Code:        "5120",
			Name:        "DEF (Diesel Exhaust Fluid)",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "DEF purchases",
			Parent:      "5100",
		},
		{
			Code:        "5130",
			Name:        "Fuel Taxes",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "IFTA and fuel taxes",
			Parent:      "5100",
		},

		{
			Code:        "5200",
			Name:        "Vehicle Maintenance",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Maintenance and repairs",
			IsSystem:    true,
		},
		{
			Code:        "5210",
			Name:        "Preventive Maintenance",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Scheduled maintenance",
			Parent:      "5200",
		},
		{
			Code:        "5220",
			Name:        "Repairs - Tractors",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Tractor repairs",
			Parent:      "5200",
		},
		{
			Code:        "5230",
			Name:        "Repairs - Trailers",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Trailer repairs",
			Parent:      "5200",
		},
		{
			Code:        "5240",
			Name:        "Tires",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Tire purchases and repairs",
			Parent:      "5200",
		},
		{
			Code:        "5250",
			Name:        "Parts and Supplies",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Maintenance parts and supplies",
			Parent:      "5200",
		},

		{
			Code:        "5300",
			Name:        "Insurance Costs",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Operating insurance",
			IsSystem:    true,
		},
		{
			Code:        "5310",
			Name:        "Liability Insurance",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "General and auto liability",
			Parent:      "5300",
		},
		{
			Code:        "5320",
			Name:        "Cargo Insurance",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Cargo coverage",
			Parent:      "5300",
		},
		{
			Code:        "5330",
			Name:        "Physical Damage Insurance",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Vehicle physical damage",
			Parent:      "5300",
		},
		{
			Code:        "5340",
			Name:        "Workers Compensation",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Workers comp insurance",
			Parent:      "5300",
		},

		{
			Code:        "5400",
			Name:        "Permits and Licenses",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Operating permits and licenses",
			IsSystem:    true,
		},
		{
			Code:        "5410",
			Name:        "Vehicle Registrations",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Vehicle registration fees",
			Parent:      "5400",
		},
		{
			Code:        "5420",
			Name:        "IRP Fees",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "International Registration Plan fees",
			Parent:      "5400",
		},
		{
			Code:        "5430",
			Name:        "UCR Fees",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Unified Carrier Registration",
			Parent:      "5400",
		},
		{
			Code:        "5440",
			Name:        "Oversize/Overweight Permits",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Special permits",
			Parent:      "5400",
		},

		{
			Code:        "5500",
			Name:        "Tolls and Road Fees",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Toll roads and fees",
			IsSystem:    true,
		},
		{
			Code:        "5510",
			Name:        "Highway Tolls",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Toll road charges",
			Parent:      "5500",
		},
		{
			Code:        "5520",
			Name:        "Scale Fees",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Weigh station fees",
			Parent:      "5500",
		},

		{
			Code:        "5600",
			Name:        "Subcontractor Costs",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Independent contractor expenses",
			IsSystem:    true,
		},
		{
			Code:        "5610",
			Name:        "Brokered Loads",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Cost of brokered loads",
			Parent:      "5600",
		},
		{
			Code:        "5620",
			Name:        "Owner Operator Lease",
			Category:    accounting.CategoryCostOfRevenue,
			Description: "Owner operator settlements",
			Parent:      "5600",
		},

		// EXPENSES (6000-6999)
		{
			Code:        "6000",
			Name:        "Administrative Expenses",
			Category:    accounting.CategoryExpense,
			Description: "General administrative costs",
			IsSystem:    true,
		},
		{
			Code:        "6010",
			Name:        "Salaries - Office",
			Category:    accounting.CategoryExpense,
			Description: "Office staff salaries",
			Parent:      "6000",
		},
		{
			Code:        "6020",
			Name:        "Salaries - Management",
			Category:    accounting.CategoryExpense,
			Description: "Management salaries",
			Parent:      "6000",
		},
		{
			Code:        "6030",
			Name:        "Payroll Taxes - Office",
			Category:    accounting.CategoryExpense,
			Description: "Employer payroll taxes for office staff",
			Parent:      "6000",
		},
		{
			Code:        "6040",
			Name:        "Employee Benefits - Office",
			Category:    accounting.CategoryExpense,
			Description: "Office staff benefits",
			Parent:      "6000",
		},

		{
			Code:        "6100",
			Name:        "Facility Expenses",
			Category:    accounting.CategoryExpense,
			Description: "Facility and occupancy costs",
			IsSystem:    true,
		},
		{
			Code:        "6110",
			Name:        "Rent - Office",
			Category:    accounting.CategoryExpense,
			Description: "Office rent",
			Parent:      "6100",
		},
		{
			Code:        "6120",
			Name:        "Rent - Yard",
			Category:    accounting.CategoryExpense,
			Description: "Yard and parking rent",
			Parent:      "6100",
		},
		{
			Code:        "6130",
			Name:        "Utilities",
			Category:    accounting.CategoryExpense,
			Description: "Electric, water, gas",
			Parent:      "6100",
		},
		{
			Code:        "6140",
			Name:        "Property Taxes",
			Category:    accounting.CategoryExpense,
			Description: "Real estate taxes",
			Parent:      "6100",
		},
		{
			Code:        "6150",
			Name:        "Building Maintenance",
			Category:    accounting.CategoryExpense,
			Description: "Building repairs and maintenance",
			Parent:      "6100",
		},
		{
			Code:        "6160",
			Name:        "Security",
			Category:    accounting.CategoryExpense,
			Description: "Security services and systems",
			Parent:      "6100",
		},

		{
			Code:        "6200",
			Name:        "Technology Expenses",
			Category:    accounting.CategoryExpense,
			Description: "IT and software costs",
			IsSystem:    true,
		},
		{
			Code:        "6210",
			Name:        "Software Subscriptions",
			Category:    accounting.CategoryExpense,
			Description: "SaaS and software licenses",
			Parent:      "6200",
		},
		{
			Code:        "6220",
			Name:        "IT Support",
			Category:    accounting.CategoryExpense,
			Description: "IT consulting and support",
			Parent:      "6200",
		},
		{
			Code:        "6230",
			Name:        "Internet and Phone",
			Category:    accounting.CategoryExpense,
			Description: "Communication services",
			Parent:      "6200",
		},
		{
			Code:        "6240",
			Name:        "Computer Equipment",
			Category:    accounting.CategoryExpense,
			Description: "Computer purchases under capitalization threshold",
			Parent:      "6200",
		},

		{
			Code:        "6300",
			Name:        "Professional Services",
			Category:    accounting.CategoryExpense,
			Description: "Professional fees",
			IsSystem:    true,
		},
		{
			Code:        "6310",
			Name:        "Legal Fees",
			Category:    accounting.CategoryExpense,
			Description: "Attorney fees",
			Parent:      "6300",
		},
		{
			Code:        "6320",
			Name:        "Accounting Fees",
			Category:    accounting.CategoryExpense,
			Description: "Accounting and bookkeeping",
			Parent:      "6300",
		},
		{
			Code:        "6330",
			Name:        "Consulting Fees",
			Category:    accounting.CategoryExpense,
			Description: "Business consulting",
			Parent:      "6300",
		},
		{
			Code:        "6340",
			Name:        "Audit Fees",
			Category:    accounting.CategoryExpense,
			Description: "Financial audit fees",
			Parent:      "6300",
		},

		{
			Code:        "6400",
			Name:        "Marketing and Sales",
			Category:    accounting.CategoryExpense,
			Description: "Marketing and business development",
			IsSystem:    true,
		},
		{
			Code:        "6410",
			Name:        "Advertising",
			Category:    accounting.CategoryExpense,
			Description: "Advertising expenses",
			Parent:      "6400",
		},
		{
			Code:        "6420",
			Name:        "Website and Digital Marketing",
			Category:    accounting.CategoryExpense,
			Description: "Online marketing",
			Parent:      "6400",
		},
		{
			Code:        "6430",
			Name:        "Trade Shows",
			Category:    accounting.CategoryExpense,
			Description: "Trade show expenses",
			Parent:      "6400",
		},
		{
			Code:        "6440",
			Name:        "Customer Entertainment",
			Category:    accounting.CategoryExpense,
			Description: "Client entertainment",
			Parent:      "6400",
		},

		{
			Code:        "6500",
			Name:        "Office Expenses",
			Category:    accounting.CategoryExpense,
			Description: "Office supplies and expenses",
			IsSystem:    true,
		},
		{
			Code:        "6510",
			Name:        "Office Supplies",
			Category:    accounting.CategoryExpense,
			Description: "General office supplies",
			Parent:      "6500",
		},
		{
			Code:        "6520",
			Name:        "Postage and Shipping",
			Category:    accounting.CategoryExpense,
			Description: "Mailing and shipping costs",
			Parent:      "6500",
		},
		{
			Code:        "6530",
			Name:        "Printing and Copying",
			Category:    accounting.CategoryExpense,
			Description: "Printing services",
			Parent:      "6500",
		},

		{
			Code:        "6600",
			Name:        "Travel and Entertainment",
			Category:    accounting.CategoryExpense,
			Description: "Business travel expenses",
			IsSystem:    true,
		},
		{
			Code:        "6610",
			Name:        "Airfare",
			Category:    accounting.CategoryExpense,
			Description: "Air travel",
			Parent:      "6600",
		},
		{
			Code:        "6620",
			Name:        "Lodging",
			Category:    accounting.CategoryExpense,
			Description: "Hotel expenses",
			Parent:      "6600",
		},
		{
			Code:        "6630",
			Name:        "Meals",
			Category:    accounting.CategoryExpense,
			Description: "Business meals",
			Parent:      "6600",
		},
		{
			Code:        "6640",
			Name:        "Auto Rental",
			Category:    accounting.CategoryExpense,
			Description: "Vehicle rentals",
			Parent:      "6600",
		},

		{
			Code:        "6700",
			Name:        "Depreciation and Amortization",
			Category:    accounting.CategoryExpense,
			Description: "Non-cash depreciation expense",
			IsSystem:    true,
		},
		{
			Code:        "6710",
			Name:        "Depreciation Expense",
			Category:    accounting.CategoryExpense,
			Description: "Depreciation of fixed assets",
			Parent:      "6700",
		},
		{
			Code:        "6720",
			Name:        "Amortization Expense",
			Category:    accounting.CategoryExpense,
			Description: "Amortization of intangibles",
			Parent:      "6700",
		},

		{
			Code:        "6800",
			Name:        "Interest and Bank Charges",
			Category:    accounting.CategoryExpense,
			Description: "Financial costs",
			IsSystem:    true,
		},
		{
			Code:        "6810",
			Name:        "Interest Expense",
			Category:    accounting.CategoryExpense,
			Description: "Loan interest",
			Parent:      "6800",
		},
		{
			Code:        "6820",
			Name:        "Bank Service Charges",
			Category:    accounting.CategoryExpense,
			Description: "Bank fees",
			Parent:      "6800",
		},
		{
			Code:        "6830",
			Name:        "Credit Card Fees",
			Category:    accounting.CategoryExpense,
			Description: "Merchant and processing fees",
			Parent:      "6800",
		},

		{
			Code:        "6900",
			Name:        "Other Expenses",
			Category:    accounting.CategoryExpense,
			Description: "Miscellaneous expenses",
			IsSystem:    true,
		},
		{
			Code:        "6910",
			Name:        "Training and Education",
			Category:    accounting.CategoryExpense,
			Description: "Employee training",
			Parent:      "6900",
		},
		{
			Code:        "6920",
			Name:        "Dues and Subscriptions",
			Category:    accounting.CategoryExpense,
			Description: "Industry memberships",
			Parent:      "6900",
		},
		{
			Code:        "6930",
			Name:        "Charitable Contributions",
			Category:    accounting.CategoryExpense,
			Description: "Donations",
			Parent:      "6900",
		},
		{
			Code:        "6940",
			Name:        "Bad Debt Expense",
			Category:    accounting.CategoryExpense,
			Description: "Write-offs of uncollectible receivables",
			Parent:      "6900",
		},
		{
			Code:        "6950",
			Name:        "Penalties and Fines",
			Category:    accounting.CategoryExpense,
			Description: "DOT and other fines",
			Parent:      "6900",
		},
	}
}

func (s *GlAccountSeed) createDefaultCOA(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
	accountTypeIDs map[accounting.Category]pulid.ID,
) (int, error) {
	coaData := getDefaultTruckingCOA()

	// Build a map to track created accounts by code for parent lookup
	accountsByCode := make(map[string]*accounting.GLAccount)

	// First pass: create all accounts without parent relationships
	var accounts []*accounting.GLAccount
	for _, seed := range coaData {
		account := &accounting.GLAccount{
			ID:             pulid.MustNew("gla_"),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			Status:         domain.StatusActive,
			AccountTypeID:  accountTypeIDs[seed.Category],
			AccountCode:    seed.Code,
			Name:           seed.Name,
			Description:    seed.Description,
			IsActive:       true,
			IsSystem:       seed.IsSystem,
			AllowManualJE:  true,
			RequireProject: false,
		}
		accounts = append(accounts, account)
		accountsByCode[seed.Code] = account
	}

	// Insert all accounts first
	_, err := tx.NewInsert().
		Model(&accounts).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("insert GL accounts: %w", err)
	}

	// Second pass: update parent relationships
	for i, seed := range coaData {
		if seed.Parent != "" {
			if parent, ok := accountsByCode[seed.Parent]; ok {
				accounts[i].ParentID = &parent.ID
			}
		}
	}

	// Update accounts with parent relationships
	for _, account := range accounts {
		if account.ParentID != nil {
			_, err := tx.NewUpdate().
				Model(account).
				Column("parent_id").
				Where("id = ?", account.ID).
				Where("organization_id = ?", orgID).
				Where("business_unit_id = ?", buID).
				Exec(ctx)
			if err != nil {
				return 0, fmt.Errorf("update parent relationships: %w", err)
			}
		}
	}

	return len(accounts), nil
}
