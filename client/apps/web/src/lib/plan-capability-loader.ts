import { edition } from "@/lib/edition";
import { isPlanRestricted, type PlanCapabilityType } from "@/lib/plan-capability";
import { redirect, type LoaderFunction } from "react-router";

/**
 * Keeps a route its plan withholds out of reach and sends the visitor where the
 * edition says instead. Like `createCapabilityLoader` this is decluttering, not
 * authorization: the API refuses restricted actions on its own. Without an edition
 * that has plans nothing is withheld, and when the plan cannot be loaded the route
 * stays reachable rather than locking anyone out.
 */
export function createPlanCapabilityLoader(capability: PlanCapabilityType): LoaderFunction {
  return async () => {
    try {
      const restrictions = await edition.plan.loadRestrictions();
      if (!isPlanRestricted(restrictions, capability)) {
        return null;
      }
    } catch {
      return null;
    }

    throw redirect(edition.plan.restrictedRedirect);
  };
}
