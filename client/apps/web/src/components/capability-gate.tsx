import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import {
  hasOrganizationCapability,
  type OrganizationCapabilityType,
} from "@trenova/shared/types/organization-capability";
import type { ReactNode } from "react";

interface CapabilityGateProps {
  capability: OrganizationCapabilityType;
  children: ReactNode;
  fallback?: ReactNode;
}

/**
 * Hides a surface the organization has switched off. It is the in-page twin of
 * the navigation and route capability filters, and it composes with
 * `PermissionGate` rather than replacing it: the capability says whether the
 * organization uses this half of the product, the permission says whether this
 * user may act on it.
 */
export function CapabilityGate({ capability, children, fallback = null }: CapabilityGateProps) {
  const capabilities = useOrgCapabilities();

  if (hasOrganizationCapability(capabilities, capability)) {
    return <>{children}</>;
  }

  return <>{fallback}</>;
}
