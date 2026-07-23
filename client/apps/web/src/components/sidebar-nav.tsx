import type { OperationType } from "@/types/permission";

export type SidebarLink = {
  href: string;
  title: string;
  group?: string;
  disabled?: boolean;
  includeBetaTag?: boolean;
  resource?: string;
  requiredOperation?: OperationType;
};
