import type { PlanCapabilityType } from "@/lib/plan-capability";
import type { OrganizationCapabilityType } from "@trenova/shared/types/organization-capability";
import type { PlatformMode } from "@trenova/shared/types/platform";
import type { OperationType } from "@trenova/shared/types/permission";

export type SidebarLink = {
  href: string;
  title: string;
  group?: string;
  disabled?: boolean;
  includeBetaTag?: boolean;
  resource?: string;
  requiredOperation?: OperationType;
  capability?: OrganizationCapabilityType;
  /** Hides the entry when the organization's plan withholds it. */
  planCapability?: PlanCapabilityType;
  /**
   * Shows the entry only on an install running in this platform mode, for an edition
   * page that means nothing elsewhere.
   */
  platformMode?: PlatformMode;
};
