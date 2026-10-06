import { edition } from "@/lib/edition";

const usePlanRestrictionsFromEdition = edition.plan.useRestrictions;

/**
 * The capabilities the organization's plan withholds, as the edition reports them.
 * Empty without an edition that has plans, and until its plan loads, so nothing is
 * hidden on a guess.
 */
export function usePlanRestrictions(): readonly string[] {
  return usePlanRestrictionsFromEdition();
}
