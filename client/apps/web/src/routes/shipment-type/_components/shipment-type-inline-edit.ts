import type { ShipmentTypeRow } from "@/lib/graphql/shipment-type-table";
import type { ShipmentType } from "@/types/shipment-type";
import { translate } from "@trenova/shared/i18n/runtime";

const INLINE_EDITABLE_FIELDS = new Set<string>(["code", "description", "color"]);

/**
 * What a change made in the shipment types table saves, or null for a column the table does
 * not edit. A colour taken away is saved as none: a JSON null would leave the old one in place.
 */
export function shipmentTypeInlinePatch(
  field: string,
  value: unknown,
): Partial<ShipmentType> | null {
  if (!INLINE_EDITABLE_FIELDS.has(field)) return null;
  if (field === "code" && (value === null || value === "")) {
    throw new Error(translate("Code is required"));
  }
  const saved = field === "color" && value === null ? "" : value;
  return { [field as keyof ShipmentTypeRow]: saved } as Partial<ShipmentType>;
}
