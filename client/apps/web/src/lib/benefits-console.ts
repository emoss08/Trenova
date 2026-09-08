import { BENEFIT_PLAN_TYPE_ORDER, benefitPlanTypeLabel } from "@trenova/shared/lib/benefits";

const SECONDS_IN_DAY = 86_400;

// Structural shapes rather than the generated types, so the same maths serves
// the page and a fixture written from the schema by hand.
export type PlanLike = {
  id: string;
  status: string;
  planType: string;
  planYear: number;
  employeeCostMinor: number;
  employerCostMinor: number;
};

export type CostLike = {
  planId: string;
  enrolled: number;
  waived: number;
  employeeCostMinor: number;
  employerCostMinor: number;
};

export type EnrollmentLike = {
  id: string;
  status: string;
  effectiveFrom: number;
  effectiveTo?: number | null;
};

/** The plan years on file, newest first. */
export function planYearsOf(plans: readonly Pick<PlanLike, "planYear">[]): number[] {
  const years = new Set<number>();
  for (const plan of plans) years.add(plan.planYear);
  return Array.from(years).sort((a, b) => b - a);
}

/**
 * The year the page opens on: this year if there is a plan for it, otherwise
 * the newest year there is. A carrier that has priced next year already
 * still administers this one.
 */
export function defaultPlanYear(years: readonly number[], thisYear: number): number | null {
  if (years.includes(thisYear)) return thisYear;
  return years[0] ?? null;
}

export type CostTotals = {
  enrolled: number;
  waived: number;
  employeeMinor: number;
  employerMinor: number;
  totalMinor: number;
};

export function costTotals(costs: readonly CostLike[]): CostTotals {
  const totals: CostTotals = {
    enrolled: 0,
    waived: 0,
    employeeMinor: 0,
    employerMinor: 0,
    totalMinor: 0,
  };
  for (const row of costs) {
    totals.enrolled += row.enrolled;
    totals.waived += row.waived;
    totals.employeeMinor += row.employeeCostMinor;
    totals.employerMinor += row.employerCostMinor;
  }
  totals.totalMinor = totals.employeeMinor + totals.employerMinor;
  return totals;
}

export type PlanWithCost<P extends PlanLike, C extends CostLike> = {
  plan: P;
  cost: C | null;
};

export type PlanTypeGroup<P extends PlanLike, C extends CostLike> = {
  type: string;
  label: string;
  plans: PlanWithCost<P, C>[];
  enrolled: number;
  waived: number;
  employeeMinor: number;
  employerMinor: number;
};

/**
 * Plans by kind, in the order a benefits administrator lists them: medical
 * first, then the rest. Inside a kind, active plans come before archived
 * ones and the fuller plan first, so the plan most people are on leads.
 */
export function groupPlansByType<P extends PlanLike, C extends CostLike>(
  plans: readonly P[],
  costs: readonly C[],
): PlanTypeGroup<P, C>[] {
  const costByPlan = new Map(costs.map((row) => [row.planId, row]));
  const groups = new Map<string, PlanTypeGroup<P, C>>();
  for (const plan of plans) {
    let group = groups.get(plan.planType);
    if (!group) {
      group = {
        type: plan.planType,
        label: benefitPlanTypeLabel(plan.planType),
        plans: [],
        enrolled: 0,
        waived: 0,
        employeeMinor: 0,
        employerMinor: 0,
      };
      groups.set(plan.planType, group);
    }
    const cost = costByPlan.get(plan.id) ?? null;
    group.plans.push({ plan, cost });
    if (cost) {
      group.enrolled += cost.enrolled;
      group.waived += cost.waived;
      group.employeeMinor += cost.employeeCostMinor;
      group.employerMinor += cost.employerCostMinor;
    }
  }
  const order = (type: string) => {
    const index = (BENEFIT_PLAN_TYPE_ORDER as readonly string[]).indexOf(type);
    return index === -1 ? BENEFIT_PLAN_TYPE_ORDER.length : index;
  };
  const result = Array.from(groups.values()).sort((a, b) => order(a.type) - order(b.type));
  for (const group of result) {
    group.plans.sort(
      (a, b) =>
        Number(b.plan.status === "Active") - Number(a.plan.status === "Active") ||
        (b.cost?.enrolled ?? 0) - (a.cost?.enrolled ?? 0),
    );
  }
  return result;
}

/** Plans in the year still open to new enrollments that nobody has joined. */
export function emptyActivePlans<P extends PlanLike, C extends CostLike>(
  plans: readonly P[],
  costs: readonly C[],
): P[] {
  const enrolled = new Map(costs.map((row) => [row.planId, row.enrolled]));
  return plans.filter((plan) => plan.status === "Active" && (enrolled.get(plan.id) ?? 0) === 0);
}

export type EnrollmentStanding = "starting" | "covered" | "ending" | "declined" | "ended";

const ENDING_SOON_DAYS = 30;

/**
 * Where an enrollment stands today, which the stored status alone does not
 * say: an Active enrollment dated next month has not started, and one with
 * an end date this month is on its way out.
 */
export function enrollmentStanding(enrollment: EnrollmentLike, now: number): EnrollmentStanding {
  switch (enrollment.status) {
    case "Waived":
      return "declined";
    case "Ended":
      return "ended";
    default:
      break;
  }
  if (enrollment.effectiveFrom > now) return "starting";
  if (enrollment.effectiveTo != null) {
    if (enrollment.effectiveTo < now) return "ended";
    if (enrollment.effectiveTo <= now + ENDING_SOON_DAYS * SECONDS_IN_DAY) return "ending";
  }
  return "covered";
}

export const ENROLLMENT_STANDING_LABELS: Record<EnrollmentStanding, string> = {
  starting: "Starts soon",
  covered: "Covered",
  ending: "Ending soon",
  declined: "Declined",
  ended: "Ended",
};

export function enrollmentStandingTone(
  standing: EnrollmentStanding,
): "active" | "inactive" | "warning" | "secondary" {
  switch (standing) {
    case "covered":
      return "active";
    case "ended":
      return "inactive";
    case "starting":
    case "ending":
      return "warning";
    default:
      return "secondary";
  }
}

/** Cover that has been arranged but has not begun, soonest first. */
export function startingSoon<T extends EnrollmentLike>(entries: readonly T[], now: number): T[] {
  return entries
    .filter((entry) => enrollmentStanding(entry, now) === "starting")
    .sort((a, b) => a.effectiveFrom - b.effectiveFrom);
}

/** Cover with an end date inside the next thirty days, soonest first. */
export function endingSoon<T extends EnrollmentLike>(entries: readonly T[], now: number): T[] {
  return entries
    .filter((entry) => enrollmentStanding(entry, now) === "ending")
    .sort((a, b) => (a.effectiveTo ?? 0) - (b.effectiveTo ?? 0));
}

/** Declines, most recent decision first. */
export function recentDeclines<T extends EnrollmentLike>(entries: readonly T[]): T[] {
  return entries
    .filter((entry) => entry.status === "Waived")
    .sort((a, b) => b.effectiveFrom - a.effectiveFrom);
}

/** Name or terminal, matched loosely enough for a half-typed name. */
export function matchesEnrollmentSearch(
  entry: {
    worker?: { firstName: string; lastName: string; fleetCode?: { code: string } | null } | null;
  },
  query: string,
): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) return true;
  const worker = entry.worker;
  if (!worker) return false;
  return [`${worker.firstName} ${worker.lastName}`, worker.fleetCode?.code ?? ""].some((field) =>
    field.toLowerCase().includes(needle),
  );
}
