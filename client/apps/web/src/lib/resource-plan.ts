import { queryClient } from "@/lib/query-client";
import type { ResourceCategory } from "@/lib/role-api";
import { planFeatureOf } from "@trenova/shared/hooks/use-plan-feature";
import type { PlanRestriction } from "@trenova/shared/lib/plan-limit";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";

/** The feature a permission resource belongs to, from the server's permission catalogue. */
export function featureOfResource(
  categories: readonly ResourceCategory[],
  resource: string,
): string | undefined {
  for (const category of categories) {
    const definition = category.resources.find((entry) => entry.resource === resource);
    if (definition) return definition.feature;
  }
  return undefined;
}

/**
 * Whether a permission resource is missing from the manifest because the organization's plan
 * leaves its feature out, rather than because the person's roles do not grant it. The manifest
 * drops an off-plan resource from `permissions` either way, so the answer comes from the
 * resource's feature, which the server's permission catalogue (`GET /permissions/resources`)
 * declares, checked against the manifest's `unavailableFeatures`. Nothing is fetched when the
 * plan leaves nothing out, and a catalogue that cannot be read answers null: the person then
 * sees the permission refusal, which is still true.
 */
export async function planRestrictionOfResource(resource: string): Promise<PlanRestriction | null> {
  const { manifest } = usePermissionStore.getState();
  if (!manifest || manifest.unavailableFeatures.length === 0) {
    return null;
  }

  let feature: string | undefined;
  try {
    const { PERMISSION_CATALOG_STALE_TIME, permissionCatalog } =
      await import("@/lib/queries/permission-catalog");
    const categories = await queryClient.fetchQuery({
      ...permissionCatalog.resources(),
      staleTime: PERMISSION_CATALOG_STALE_TIME,
    });
    feature = featureOfResource(categories, resource);
  } catch {
    return null;
  }

  if (!feature) {
    return null;
  }

  const plan = planFeatureOf(manifest, feature);
  return plan.available ? null : { feature, addOn: plan.addOnName };
}
