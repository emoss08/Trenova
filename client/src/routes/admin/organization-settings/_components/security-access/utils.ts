import type { BadgeVariant } from "@/components/ui/badge";
import type {
  AccessPolicy,
  IdentityProvider,
  IdentityProviderFormValues,
} from "@/types/iam";
import type { API_ENDPOINTS } from "@/types/server";

export type ConditionRow = { id: string; key: string; value: string };

export const emptyProvider: IdentityProvider = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  name: "",
  slug: "",
  protocol: "OIDC",
  enabled: true,
  enforceSso: false,
  autoProvision: false,
  allowFederatedMfa: true,
  allowedDomains: [],
  attributeMap: { email: "email" },
  oidcIssuerUrl: "",
  oidcClientId: "",
  oidcClientSecret: "",
  oidcRedirectUrl: "",
  oidcScopes: ["openid", "email", "profile"],
  version: 0,
  createdAt: 0,
  updatedAt: 0,
};

export const emptyPolicy: AccessPolicy = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  name: "",
  resource: "",
  operation: "read",
  effect: "deny",
  priority: 100,
  conditions: {},
  enabled: true,
  createdAt: 0,
  updatedAt: 0,
};

export const providerPresets = [
  {
    label: "Entra ID",
    name: "Microsoft Entra ID",
    slug: "entra-id",
    issuer: "https://login.microsoftonline.com/{tenant-id}/v2.0",
    scopes: ["openid", "email", "profile"],
  },
  {
    label: "Okta",
    name: "Okta",
    slug: "okta",
    issuer: "https://{yourOktaDomain}/oauth2/default",
    scopes: ["openid", "email", "profile"],
  },
];

export function identityProviderQueryKey(organizationId: string) {
  return `identity-provider-list:${organizationId}`;
}

export function identityProviderEndpoint(organizationId: string) {
  return `/organizations/${organizationId}/iam/identity-providers/` as API_ENDPOINTS;
}

export function recordToConditionRows(value: Record<string, string>) {
  return Object.entries(value).map(([key, mapValue], index) => ({
    id: `${key}-${index}`,
    key,
    value: mapValue,
  }));
}

export function conditionRowsToRecord(rows: ConditionRow[]) {
  return Object.fromEntries(
    rows
      .map((row) => [row.key.trim(), row.value.trim()] as const)
      .filter(([key]) => key),
  );
}

export function toIdentityProvider(values: IdentityProviderFormValues): IdentityProvider {
  return {
    ...emptyProvider,
    ...values,
    protocol: "OIDC",
    allowedDomains: values.allowedDomains ?? [],
    attributeMap: values.attributeMap ?? { email: "email" },
    oidcScopes: values.oidcScopes ?? [],
  };
}

export function riskVariant(value: string): BadgeVariant {
  switch (value) {
    case "allow":
    case "success":
    case "active":
    case "created":
    case "updated":
    case "completed":
      return "active";
    case "challenge":
    case "pending":
      return "warning";
    case "deny":
    case "denied":
    case "failed":
    case "error":
    case "revoked":
      return "inactive";
    default:
      return "outline";
  }
}

export function outcomeVariant(value: string): BadgeVariant {
  switch (value) {
    case "success":
      return "active";
    case "challenge":
      return "warning";
    case "denied":
    case "failed":
      return "inactive";
    default:
      return "outline";
  }
}
