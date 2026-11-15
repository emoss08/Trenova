import { isFeatureOnPlan } from "@trenova/shared/hooks/use-plan-feature";
import type { UnavailableFeature } from "@trenova/shared/types/permission";

type PlanColumn = { meta?: { planFeature?: string } };

const NONE_UNAVAILABLE: readonly UnavailableFeature[] = [];

/**
 * The columns the organization's plan includes. A column whose `meta.planFeature` names an
 * add-on the plan leaves out is not part of the table at all: no header, no cells, no filter,
 * no export and no entry in the column picker. With every feature on (self-hosted) nothing is
 * removed and the same array is returned.
 */
export function columnsOnPlan<T extends PlanColumn>(
  columns: readonly T[],
  unavailable: readonly UnavailableFeature[] = NONE_UNAVAILABLE,
): T[] {
  if (unavailable.length === 0) return columns as T[];
  return columns.filter((column) => isFeatureOnPlan(unavailable, column.meta?.planFeature));
}
