import { recordPath } from "@/config/record-links";

export type CarrierPanelTab = "identity" | "intelligence";

export function carrierPanelPath(carrierId: string, tab?: CarrierPanelTab): string {
  return recordPath("carrier", carrierId, tab ? { tab } : undefined);
}

export const CARRIER_INTEL_INTEGRATIONS_PATH = "/admin/integrations?category=CarrierCompliance";
