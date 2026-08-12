import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  ORGANIZATION_CAPABILITIES_ENABLED,
  type OrganizationCapabilities,
} from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { useMemo } from "react";

export { ORGANIZATION_CAPABILITIES_ENABLED };

/**
 * The session payload carries the whole organization inside the membership that
 * matches `currentOrganizationId` — the same lookup `metadata.tsx` uses for the
 * organization name — so capabilities need no extra request.
 */
export function resolveOrgCapabilities(user: User | null): OrganizationCapabilities {
  const organization = user?.memberships?.find(
    (membership) => membership.organizationId === user.currentOrganizationId,
  )?.organization;

  if (!organization) {
    return ORGANIZATION_CAPABILITIES_ENABLED;
  }

  return {
    brokerageEnabled: organization.brokerageEnabled,
    assetOperationsEnabled: organization.assetOperationsEnabled,
  };
}

/** Reads the current organization's capabilities from the persisted session. */
export function useOrgCapabilities(): OrganizationCapabilities {
  const user = useAuthStore((state) => state.user);

  return useMemo(() => resolveOrgCapabilities(user), [user]);
}

/** Non-reactive read for route loaders and other non-React call sites. */
export function getOrgCapabilities(): OrganizationCapabilities {
  return resolveOrgCapabilities(useAuthStore.getState().user);
}
