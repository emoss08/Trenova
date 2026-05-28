import { parseAsBoolean, parseAsString, parseAsStringLiteral } from "nuqs";

export const organizationSettingsTabValues = ["general", "security", "billing-usage"] as const;
export const securityTabValues = ["sign-in", "provisioning", "policies", "activity"] as const;
export const activityViewValues = ["auth", "risk", "identities", "mfa"] as const;
export const identityProviderPanelModeValues = ["create", "edit"] as const;

export type SecurityTabValue = (typeof securityTabValues)[number];
export type OrganizationSettingsTabValue = (typeof organizationSettingsTabValues)[number];
export type ActivityViewValue = (typeof activityViewValues)[number];
export type IdentityProviderPanelModeValue = (typeof identityProviderPanelModeValues)[number];

export const organizationSettingsTabParser = parseAsStringLiteral(
  organizationSettingsTabValues,
).withDefault("general");
export const securityTabParser = parseAsStringLiteral(securityTabValues)
  .withDefault("sign-in")
  .withOptions({ history: "push" });
export const activityViewParser = parseAsStringLiteral(activityViewValues)
  .withDefault("auth")
  .withOptions({ history: "push" });
export const directoryIdParser = parseAsString.withOptions({ history: "replace" });
export const identityProviderSearchParser = parseAsString
  .withDefault("")
  .withOptions({ history: "replace" });
export const editingProviderParser = parseAsString.withOptions({ history: "replace" });
export const identityProviderPanelModeParser = parseAsStringLiteral(
  identityProviderPanelModeValues,
)
  .withDefault("create")
  .withOptions({ history: "replace" });
export const identityProviderPanelOpenParser = parseAsBoolean
  .withDefault(false)
  .withOptions({ history: "replace" });

export const organizationSettingsTabSearchParamsParser = {
  tab: organizationSettingsTabParser,
};

export const securityTabSearchParamsParser = {
  securityTab: securityTabParser,
};

export const activityViewSearchParamsParser = {
  activityView: activityViewParser,
};

export const directorySearchParamsParser = {
  directoryId: directoryIdParser,
};

export const identityProviderSearchParamsParser = {
  search: identityProviderSearchParser,
};

export const identityProviderPanelSearchParamsParser = {
  editingProvider: editingProviderParser,
  panelMode: identityProviderPanelModeParser,
  panelOpen: identityProviderPanelOpenParser,
};

export const searchParamsParser = {
  ...organizationSettingsTabSearchParamsParser,
  ...securityTabSearchParamsParser,
  ...activityViewSearchParamsParser,
  ...directorySearchParamsParser,
  ...identityProviderSearchParamsParser,
  ...identityProviderPanelSearchParamsParser,
};
