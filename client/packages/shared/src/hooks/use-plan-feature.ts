import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest, UnavailableFeature } from "@trenova/shared/types/permission";

/**
 * Whether the organization's plan includes a feature, and, when it does not, the names the
 * server gave it and the add-on it belongs to. The permission manifest's
 * `unavailableFeatures` is the one source of truth: a feature not listed there is
 * available, and so is every feature before the manifest has loaded (the server still
 * refuses what the plan withholds, so nothing is shown that should not be).
 */
export type PlanFeatureState = {
  available: boolean;
  featureName: string;
  addOnName: string;
};

const AVAILABLE: PlanFeatureState = Object.freeze({
  available: true,
  featureName: "",
  addOnName: "",
});

/** The manifest's entry for a feature the plan does not include, or undefined. */
export function unavailablePlanFeature(
  manifest: Pick<PermissionManifest, "unavailableFeatures"> | null | undefined,
  feature: string,
): UnavailableFeature | undefined {
  if (!manifest || !feature) return undefined;
  return manifest.unavailableFeatures.find((entry) => entry.feature === feature);
}

function stateOf(entry: UnavailableFeature | undefined): PlanFeatureState {
  if (!entry) return AVAILABLE;
  return { available: false, featureName: entry.featureName, addOnName: entry.addOnName };
}

/** The same answer as `usePlanFeature`, for a loader or any code outside React. */
export function planFeatureOf(
  manifest: Pick<PermissionManifest, "unavailableFeatures"> | null | undefined,
  feature: string,
): PlanFeatureState {
  return stateOf(unavailablePlanFeature(manifest, feature));
}

export function usePlanFeature(feature: string): PlanFeatureState {
  const entry = usePermissionStore((state) => unavailablePlanFeature(state.manifest, feature));
  return stateOf(entry);
}

const NONE_UNAVAILABLE: readonly UnavailableFeature[] = Object.freeze([]);

/**
 * Every feature the plan leaves out, for code that filters a list (navigation, pickers) by
 * each entry's feature. Empty until the manifest loads and on a self-hosted install.
 */
export function useUnavailableFeatures(): readonly UnavailableFeature[] {
  return usePermissionStore((state) => state.manifest?.unavailableFeatures ?? NONE_UNAVAILABLE);
}

/** Whether a feature is on the plan, given the manifest's `unavailableFeatures`. No feature is. */
export function isFeatureOnPlan(
  unavailable: readonly UnavailableFeature[],
  feature: string | undefined,
): boolean {
  if (!feature) return true;
  return !unavailable.some((entry) => entry.feature === feature);
}
