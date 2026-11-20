import type { CostCategory, CostingControl } from "@/lib/graphql/cost-control";

/** One of the categories every organization starts with (costingcontrol/defaults.go). */
function seeded(
  index: number,
  category: CostCategory["category"],
  name: string,
  costBehavior: CostCategory["costBehavior"],
  benchmarkRatePerMile: string,
): CostCategory {
  return {
    id: `ccat_0${index}`,
    category,
    name,
    costBehavior,
    rateSource: "Benchmark",
    benchmarkRatePerMile,
    overrideRatePerMile: null,
    isActive: true,
    sortOrder: index,
    version: 1,
    glAccounts: [],
  };
}

/**
 * A new organization's cost control as the GraphQL query returns it: live fuel price and
 * GL actuals off, 6.5 mpg, a three-month window, no target margin, and the ten seeded
 * categories on their benchmarks in sort order.
 */
export const savedCostControl: CostingControl = {
  id: "cstc_01",
  businessUnitId: "bu_01",
  organizationId: "org_01",
  fuelIndexId: null,
  fuelIndex: null,
  useLiveFuelPrice: false,
  milesPerGallon: "6.5",
  includeDeadheadMiles: true,
  glActualsEnabled: false,
  glRollingMonths: 3,
  plannedMonthlyMiles: null,
  targetMarginPercent: null,
  version: 2,
  createdAt: 1_793_577_600,
  updatedAt: 1_793_577_600,
  categories: [
    seeded(0, "DriverWages", "Driver Wages", "Variable", "0.82"),
    seeded(1, "DriverBenefits", "Driver Benefits", "Variable", "0.21"),
    seeded(2, "Fuel", "Fuel", "Variable", "0.48"),
    seeded(3, "EquipmentPayments", "Truck & Trailer Payments", "Fixed", "0.404"),
    seeded(4, "Maintenance", "Repair & Maintenance", "Variable", "0.215"),
    seeded(5, "Insurance", "Insurance Premiums", "Fixed", "0.106"),
    seeded(6, "Tires", "Tires", "Variable", "0.05"),
    seeded(7, "Tolls", "Tolls", "Variable", "0.04"),
    seeded(8, "PermitsLicenses", "Permits & Licenses", "Fixed", "0.01"),
    seeded(9, "Overhead", "Overhead & Admin", "Fixed", "0.10"),
  ],
};
