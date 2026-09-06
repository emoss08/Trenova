/**
 * Labels and money shaping for benefits and compensation.
 *
 * Every amount crossing the wire is in minor units — cents — because a total
 * compensation figure assembled from floats is a figure that disagrees with
 * itself by a penny somewhere.
 */

export const BENEFIT_PLAN_TYPE_ORDER = [
  "Medical",
  "Dental",
  "Vision",
  "Life",
  "Disability",
  "Retirement",
  "Other",
] as const;

export type BenefitPlanTypeValue = (typeof BENEFIT_PLAN_TYPE_ORDER)[number];

export const BENEFIT_PLAN_TYPE_LABELS: Record<string, string> = {
  Medical: "Medical",
  Dental: "Dental",
  Vision: "Vision",
  Life: "Life insurance",
  Disability: "Disability",
  Retirement: "Retirement",
  Other: "Other",
};

export function benefitPlanTypeLabel(value: string): string {
  return BENEFIT_PLAN_TYPE_LABELS[value] ?? value;
}

export const COVERAGE_TIER_ORDER = [
  "Employee",
  "EmployeeSpouse",
  "EmployeeChildren",
  "Family",
] as const;

export type CoverageTierValue = (typeof COVERAGE_TIER_ORDER)[number];

export const COVERAGE_TIER_LABELS: Record<string, string> = {
  Employee: "Employee only",
  EmployeeSpouse: "Employee and spouse",
  EmployeeChildren: "Employee and children",
  Family: "Family",
};

export function coverageTierLabel(value: string): string {
  return COVERAGE_TIER_LABELS[value] ?? value;
}

export const ENROLLMENT_STATUS_LABELS: Record<string, string> = {
  Pending: "Not started",
  Active: "Covered",
  Waived: "Declined",
  Ended: "Ended",
};

export function enrollmentStatusLabel(value: string): string {
  return ENROLLMENT_STATUS_LABELS[value] ?? value;
}

export function enrollmentStatusTone(
  value: string,
): "active" | "inactive" | "warning" | "secondary" {
  switch (value) {
    case "Active":
      return "active";
    case "Ended":
      return "inactive";
    case "Pending":
      return "warning";
    default:
      return "secondary";
  }
}

export const DEDUCTION_KIND_LABELS: Record<string, string> = {
  Standard: "Deduction",
  Garnishment: "Garnishment",
  Benefit: "Benefit",
};

export function deductionKindLabel(value: string): string {
  return DEDUCTION_KIND_LABELS[value] ?? value;
}

/**
 * Renders a minor-unit amount as money. Integer arithmetic all the way to the
 * formatter: dividing by 100 is the last thing that happens, not the first.
 */
export function formatMinor(minor: number, currency = "USD"): string {
  return (minor / 100).toLocaleString("en-US", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

/**
 * What one enrollment costs over a year, given how many settlements a year
 * carries it. Reported per period as well, because a driver reads the number
 * that comes off their settlement and HR reads the annual one.
 */
export function annualCostMinor(perPeriodMinor: number, periodsPerYear: number): number {
  if (perPeriodMinor <= 0 || periodsPerYear <= 0) return 0;
  return perPeriodMinor * periodsPerYear;
}

/**
 * The employer's share of a total compensation figure. Zero when the total is
 * zero — an empty statement has no split, not an even one.
 */
export function employerSharePercent(employerBenefitMinor: number, totalMinor: number): number {
  if (employerBenefitMinor <= 0 || totalMinor <= 0) return 0;
  return Math.min(100, Math.round((employerBenefitMinor / totalMinor) * 1000) / 10);
}
