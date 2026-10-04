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
  /**
   * Shows the entry only on an install running in this platform mode — the plan page
   * means nothing on a self-hosted install, where every organization is unlimited.
   */
  platformMode?: PlatformMode;
};
