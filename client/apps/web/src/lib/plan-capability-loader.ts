import { queryClient } from "@/lib/query-client";
import { queries } from "@/lib/queries";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { isCloudPlatform } from "@trenova/shared/types/platform";
import { isPlanRestricted, PLAN_USAGE_PATH, type PlanCapabilityType } from "@/lib/plan-capability";
import { redirect, type LoaderFunction } from "react-router";

/**
 * Keeps a route its plan withholds out of reach and sends the visitor to Plan &
 * usage instead. Like `createCapabilityLoader` this is decluttering, not
 * authorization: the API refuses restricted actions on its own. When the plan
 * cannot be loaded the route stays reachable rather than locking anyone out.
 */
export function createPlanCapabilityLoader(capability: PlanCapabilityType): LoaderFunction {
  return async () => {
    try {
      const config = await queryClient.ensureQueryData(publicConfigQueryOptions);
      if (!isCloudPlatform(config)) {
        return null;
      }

      const summary = await queryClient.ensureQueryData(queries.platformBilling.summary());
      if (!isPlanRestricted(summary.restrictions, capability)) {
        return null;
      }
    } catch {
      return null;
    }

    throw redirect(PLAN_USAGE_PATH);
  };
}
