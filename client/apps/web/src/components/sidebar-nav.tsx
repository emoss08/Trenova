import type { OrganizationCapabilityType } from "@trenova/shared/types/organization-capability";
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
};
