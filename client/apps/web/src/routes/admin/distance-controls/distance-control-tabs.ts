import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { DistanceControl } from "@/types/distance-control";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type DistanceControlTab = "mileage";

export type DistanceControlField = keyof DistanceControl & string;

export type DistanceControlTabSpec = SettingsTabSpec<DistanceControlField> & {
  value: DistanceControlTab;
};

const TAB_LABELS = defineLabels({
  mileage: "Mileage",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<DistanceControlTab, readonly DistanceControlField[]> = {
  mileage: [
    "storeMileage",
    "autoCreateStoredMileage",
    "postalCodeFallbackToCity",
    "captureJurisdictionMiles",
    "ratingDistanceUnit",
    "loadedMoveDistanceProfileId",
    "emptyMoveDistanceProfileId",
    "etaOutOfRouteDistanceProfileId",
    "payDistanceProfileId",
    "billingDistanceProfileId",
    "fuelDistanceProfileId",
  ],
};

const TAB_ORDER: readonly DistanceControlTab[] = ["mileage"];

/**
 * The page's tabs. It has one job, how Trenova measures distance (which profile routes each
 * kind of mile, what is kept and reused, what rating reads), so it has one tab and draws no
 * tab row; every organization measures distance, so every organization sees it.
 */
export function distanceControlTabs(): DistanceControlTabSpec[] {
  return TAB_ORDER.map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs this organization sees, for the form and its loading state alike. */
export function useDistanceControlTabs(): DistanceControlTabSpec[] {
  return distanceControlTabs();
}
