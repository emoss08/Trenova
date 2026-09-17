export type CarrierPanelTab = "identity" | "intelligence";

export function carrierPanelPath(carrierId: string, tab?: CarrierPanelTab): string {
  const params = new URLSearchParams({ panelType: "edit", panelEntityId: carrierId });
  if (tab) {
    params.set("tab", tab);
  }
  return `/dispatch/carriers?${params.toString()}`;
}

export const CARRIER_INTEL_INTEGRATIONS_PATH = "/admin/integrations?category=CarrierCompliance";
