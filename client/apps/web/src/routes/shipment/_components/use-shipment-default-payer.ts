import { useT } from "@trenova/shared/i18n/use-t";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The customer billed for a charge nobody split: the shipment's bill-to when it
 * has one, otherwise the customer who ordered it.
 */
export function useShipmentDefaultPayer(): { id: string; label: string } | null {
  const t = useT();
  const { control } = useFormContext<Shipment>();
  const customerId = useWatch({ control, name: "customerId" });
  const billToCustomerId = useWatch({ control, name: "billToCustomerId" });
  const customer = useWatch({ control, name: "customer" });
  const billToCustomer = useWatch({ control, name: "billToCustomer" });

  const id = billToCustomerId || customerId;
  if (!id) return null;
  const label = (billToCustomerId ? billToCustomer?.name : customer?.name) ?? t("customer");

  return { id, label };
}
