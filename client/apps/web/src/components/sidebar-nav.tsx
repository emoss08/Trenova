import type { PlanFeatureKey } from "@/lib/plan-feature";
import type { OrganizationCapabilityType } from "@trenova/shared/types/organization-capability";
import type { PlatformMode } from "@trenova/shared/types/platform";
import type { OperationType } from "@trenova/shared/types/permission";

export type SidebarLink = {
  href: string;
  title: string;
  /** Other words people use for the page, which the command palette also matches. */
  keywords?: readonly string[];
  group?: string;
  disabled?: boolean;
  includeBetaTag?: boolean;
  resource?: string;
  requiredOperation?: OperationType;
  /**
   * Shows the entry to someone who may read any one of these, for a page whose parts answer to
   * different permissions and which shows each person the parts they may read. Used in place
   * of `resource`.
   */
  anyResources?: readonly string[];
  capability?: OrganizationCapabilityType;
  /** Hides the entry when the organization's plan withholds it. */
  planFeature?: PlanFeatureKey;
  /**
   * Shows the entry only on an install running in this platform mode, for an edition
   * page that means nothing elsewhere.
   */
  platformMode?: PlatformMode;
  /** Shows the entry only on a Managed Trenova instance (the public config's `managed`). */
  managedOnly?: boolean;
};
