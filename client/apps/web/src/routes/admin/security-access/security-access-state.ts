import { useSearchState } from "@trenova/shared/hooks/use-search-state";
import {
  searchBoolean,
  searchLiteral,
  searchString,
} from "@trenova/shared/lib/router/search-codec";
import { useCallback } from "react";
import { securityTabValues, type SecurityTabValue } from "./security-address";

export { securityTabValues, type SecurityTabValue };

export const activityViewValues = ["auth", "risk", "identities", "mfa"] as const;

export type ActivityViewValue = (typeof activityViewValues)[number];

export const activityViewParser = searchLiteral(activityViewValues)
  .withDefault("auth")
  .withOptions({ history: "push" });
export const directoryIdParser = searchString.withOptions({ history: "replace" });
export const identityProviderSearchParser = searchString
  .withDefault("")
  .withOptions({ history: "replace" });

export const identityProviderPanelSearchParamsParser = {
  editingProvider: searchString.withOptions({ history: "replace" }),
  panelMode: searchLiteral(["create", "edit"] as const)
    .withDefault("create")
    .withOptions({ history: "replace" }),
  panelOpen: searchBoolean.withDefault(false).withOptions({ history: "replace" }),
};

/** Every key the security and access page keeps in the address. */
const searchParamsParser = {
  // The page's own tab row.
  tab: searchLiteral(securityTabValues).withDefault("overview").withOptions({ history: "push" }),
  activityView: activityViewParser,
  directoryId: directoryIdParser,
  search: identityProviderSearchParser,
  ...identityProviderPanelSearchParamsParser,
};

type SecurityKey =
  | "activityView"
  | "directoryId"
  | "search"
  | "editingProvider"
  | "panelMode"
  | "panelOpen";

/** The keys a tab reads; the others' keys leave the address when it opens. */
const SECURITY_TAB_KEYS: Record<SecurityTabValue, readonly SecurityKey[]> = {
  overview: [],
  "sign-in": ["search", "editingProvider", "panelMode", "panelOpen"],
  provisioning: ["directoryId"],
  policies: [],
  activity: ["activityView"],
};

const SECURITY_KEYS = Object.values(SECURITY_TAB_KEYS).flat();

function cleared(keys: readonly SecurityKey[]) {
  return Object.fromEntries(keys.map((key) => [key, null])) as Partial<Record<SecurityKey, null>>;
}

/**
 * The open tab of the security and access page, and the way to open another. Opening one
 * takes the keys only the others read (a directory, a provider being edited, an activity
 * view) out of the address in the same write.
 */
export function useSecurityTab() {
  const [{ tab: securityTab }, setSettings] = useSearchState(searchParamsParser);
  const setSecurityTab = useCallback(
    (next: SecurityTabValue) =>
      setSettings({
        tab: next,
        ...cleared(SECURITY_KEYS.filter((key) => !SECURITY_TAB_KEYS[next].includes(key))),
      }),
    [setSettings],
  );
  return [securityTab, setSecurityTab] as const;
}
