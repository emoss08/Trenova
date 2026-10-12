import { buildCommandHref } from "@/components/command-palette/route-command-data";

export const SHIPMENTS_PATH = "/shipment-management/shipments";

/** The query parameter that starts a new shipment for one customer. */
export const NEW_SHIPMENT_CUSTOMER_PARAM = "customerId";

/** Where a new shipment for a customer starts, with the customer already chosen. */
export function newShipmentForCustomerPath(customerId: string): string {
  return buildCommandHref(SHIPMENTS_PATH, {
    panelType: "create",
    [NEW_SHIPMENT_CUSTOMER_PARAM]: customerId,
  });
}
