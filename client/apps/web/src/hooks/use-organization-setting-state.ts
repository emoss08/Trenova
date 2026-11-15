import { useSearchState } from "@trenova/shared/hooks/use-search-state";
import { parseAsBoolean, parseAsString, parseAsStringLiteral } from "nuqs";
import { useCallback } from "react";

const organizationSettingsTabValues = ["general", "security"] as const;
export const securityTabValues = [
  "overview",
  "sign-in",
  "provisioning",
  "policies",
  "activity",
] as const;
export const activityViewValues = ["auth", "risk", "identities", "mfa"] as const;

export type SecurityTabValue = (typeof securityTabValues)[number];
export type OrganizationSettingsTabValue = (typeof organizationSettingsTabValues)[number];
export type ActivityViewValue = (typeof activityViewValues)[number];

export const activityViewParser = parseAsStringLiteral(activityViewValues)
  .withDefault("auth")
  .withOptions({ history: "push" });
export const directoryIdParser = parseAsString.withOptions({ history: "replace" });
export const identityProviderSearchParser = parseAsString
  .withDefault("")
  .withOptions({ history: "replace" });

export const identityProviderPanelSearchParamsParser = {
  editingProvider: parseAsString.withOptions({ history: "replace" }),
  panelMode: parseAsStringLiteral(["create", "edit"] as const)
    .withDefault("create")
    .withOptions({ history: "replace" }),
  panelOpen: parseAsBoolean.withDefault(false).withOptions({ history: "replace" }),
};

/** Every key the organization settings page keeps in the address. */
const searchParamsParser = {
  tab: parseAsStringLiteral(organizationSettingsTabValues).withDefault("general"),
  securityTab: parseAsStringLiteral(securityTabValues)
    .withDefault("overview")
    .withOptions({ history: "push" }),
  activityView: activityViewParser,
  directoryId: directoryIdParser,
  search: identityProviderSearchParser,
  ...identityProviderPanelSearchParamsParser,
};

/** The keys a security sub-tab reads; the others' keys leave the address when it opens. */
const SECURITY_TAB_KEYS: Record<SecurityTabValue, readonly SecurityKey[]> = {
  overview: [],
  "sign-in": ["search", "editingProvider", "panelMode", "panelOpen"],
  provisioning: ["directoryId"],
  policies: [],
  activity: ["activityView"],
};

type SecurityKey =
  | "activityView"
  | "directoryId"
  | "search"
  | "editingProvider"
  | "panelMode"
  | "panelOpen";

const SECURITY_KEYS = Object.values(SECURITY_TAB_KEYS).flat();

function cleared(keys: readonly SecurityKey[]) {
  return Object.fromEntries(keys.map((key) => [key, null])) as Partial<Record<SecurityKey, null>>;
}

/**
 * The organization settings tab, and the way to open another. Leaving Security takes its
 * sub-tab and every key a sub-tab read out of the address in the same write, so coming back
 * opens Security as it first opens.
 */
export function useOrganizationSettingsTab() {
  const [{ tab }, setSettings] = useSearchState(searchParamsParser);
  const setTab = useCallback(
    (next: OrganizationSettingsTabValue) =>
      setSettings(
        next === "security"
          ? { tab: next }
          : { tab: next, securityTab: null, ...cleared(SECURITY_KEYS) },
      ),
    [setSettings],
  );
  return [tab, setTab] as const;
}

/**
 * The open Security sub-tab, and the way to open another. Opening one takes the keys only the
 * others read (a directory, a provider being edited, an activity view) out of the address in
 * the same write.
 */
export function useSecurityTab() {
  const [{ securityTab }, setSettings] = useSearchState(searchParamsParser);
  const setSecurityTab = useCallback(
    (next: SecurityTabValue) =>
      setSettings({
        securityTab: next,
        ...cleared(SECURITY_KEYS.filter((key) => !SECURITY_TAB_KEYS[next].includes(key))),
      }),
    [setSettings],
  );
  return [securityTab, setSecurityTab] as const;
}
