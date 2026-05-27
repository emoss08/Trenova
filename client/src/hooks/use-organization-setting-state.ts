import { parseAsString, parseAsStringLiteral } from "nuqs";

export const organizationSettingsTabValues = ["general", "security", "billing-usage"] as const;
export const securityTabValues = ["sign-in", "provisioning", "policies", "activity"] as const;
export const activityViewValues = ["auth", "risk", "identities", "mfa"] as const;

export type SecurityTabValue = (typeof securityTabValues)[number];
export type ActivityViewValue = (typeof activityViewValues)[number];

export const searchParamsParser = {
  tab: parseAsStringLiteral(organizationSettingsTabValues).withDefault("general"),
  securityTab: parseAsStringLiteral(securityTabValues)
    .withDefault("sign-in")
    .withOptions({ history: "push" }),
  activityView: parseAsStringLiteral(activityViewValues)
    .withDefault("auth")
    .withOptions({ history: "push" }),
  directoryId: parseAsString.withOptions({ history: "replace" }),
};
