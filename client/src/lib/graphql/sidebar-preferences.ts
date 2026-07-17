import {
  UpdateSidebarPreferencesDocument,
  type SidebarCustomizationOptionsQuery,
  type SidebarPreferencesInput,
  type SidebarPreferencesQuery,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";

export type EffectiveSidebarPreferences = SidebarPreferencesQuery["sidebarPreferences"];
export type SidebarSectionPreference = EffectiveSidebarPreferences["sections"][number];
export type SidebarCustomizationOptions =
  SidebarCustomizationOptionsQuery["sidebarCustomizationOptions"];

export const SIDEBAR_SECTION_KEYS = {
  attention: "attention",
  quickActions: "quickActions",
  favorites: "favorites",
  activity: "activity",
  browse: "browse",
} as const;

export type SidebarSectionKey = (typeof SIDEBAR_SECTION_KEYS)[keyof typeof SIDEBAR_SECTION_KEYS];

export const DEFAULT_SIDEBAR_PREFERENCES: EffectiveSidebarPreferences = {
  schemaVersion: 1,
  version: 0,
  sections: [
    { key: SIDEBAR_SECTION_KEYS.attention, hidden: false },
    { key: SIDEBAR_SECTION_KEYS.quickActions, hidden: false },
    { key: SIDEBAR_SECTION_KEYS.favorites, hidden: false },
    { key: SIDEBAR_SECTION_KEYS.activity, hidden: false },
    { key: SIDEBAR_SECTION_KEYS.browse, hidden: false },
  ],
  attentionMetrics: [
    "billingQueue",
    "pendingApprovals",
    "reconciliationExceptions",
    "serviceFailures",
    "ediAttention",
  ],
  quickActionIds: ["create-shipment", "create-worker", "create-location", "create-customer"],
  activity: { pageSize: 5, defaultOpen: true },
};

export const DEFAULT_SIDEBAR_PREFERENCES_QUERY: SidebarPreferencesQuery = {
  sidebarPreferences: DEFAULT_SIDEBAR_PREFERENCES,
};

export async function updateSidebarPreferences(
  input: SidebarPreferencesInput,
): Promise<EffectiveSidebarPreferences> {
  const data = await requestGraphQL({
    document: UpdateSidebarPreferencesDocument,
    operationName: "UpdateSidebarPreferences",
    variables: { input },
  });

  return data.updateSidebarPreferences;
}
