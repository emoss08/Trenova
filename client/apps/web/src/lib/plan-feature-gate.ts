import type { PlanFeatureKey } from "@/lib/plan-feature";
import { ensurePermissionManifest } from "@/lib/route-permission";
import { planFeatureOf } from "@trenova/shared/hooks/use-plan-feature";
import { planRestrictedRefusal } from "@trenova/shared/lib/plan-limit";
import type { RouteGate } from "@trenova/shared/lib/router/gates";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";

/**
 * Guards a page that belongs to a feature with no permission resource of its own (the freight
 * outlook, document intelligence). When the organization's plan leaves the feature out the
 * route throws the plan refusal, and its error component explains which add-on the page is
 * part of. The API refuses the page's reads on its own; this is so a saved link explains
 * itself instead of opening a page that cannot load. Without a manifest (the session is gone)
 * the route stays reachable and the session gate sends the person to sign in.
 */
export function requirePlanFeature(feature: PlanFeatureKey): RouteGate {
  return async () => {
    if (!(await ensurePermissionManifest())) {
      return;
    }

    const plan = planFeatureOf(usePermissionStore.getState().manifest, feature);
    if (plan.available) {
      return;
    }

    throw planRestrictedRefusal({ feature, addOn: plan.addOnName });
  };
}
