import { queryClient } from "@/lib/query-client";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { isCloudPlatform } from "@trenova/shared/types/platform";
import { platformBilling } from "./queries/platform-billing";

const NO_RESTRICTIONS: readonly string[] = [];

/**
 * The capabilities the organization's plan withholds, for a route loader. Nothing off
 * Trenova Cloud. A failed read rejects, and the host then keeps the route reachable
 * rather than locking anyone out: the API refuses restricted actions on its own.
 */
export async function loadPlanRestrictions(): Promise<readonly string[]> {
  const config = await queryClient.ensureQueryData(publicConfigQueryOptions);
  if (!isCloudPlatform(config)) {
    return NO_RESTRICTIONS;
  }

  const summary = await queryClient.ensureQueryData(platformBilling.summary());
  return summary.restrictions;
}
