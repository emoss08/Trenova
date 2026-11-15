import { PlanFeature } from "@/lib/plan-feature";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";

/**
 * Whether Nova works in the formula and library studios: drafting, explaining, suggesting
 * tests, answering. Nova is Trenova AI's, so a plan without it has a studio with no Nova
 * tab, no Nova buttons and nothing that asks it.
 */
export function useStudioNova(): boolean {
  return usePlanFeature(PlanFeature.AIDesk).available;
}
