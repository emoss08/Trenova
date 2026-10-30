import type { SelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaBoolean, selectOptionMetaString } from "@/lib/select-option-meta";
import type { ShipmentCommodity, ShipmentLineCommodity } from "@trenova/shared/types/shipment";

/**
 * The commodity a line names once someone picks it, from what the picker carries: its name,
 * description and the handling flags the line shows (hazardous, stackable, fragile), so the row
 * reads the same before the shipment is saved as after it is read back.
 */
export function commodityFromOption(option: SelectOption): ShipmentLineCommodity {
  return {
    id: option.id,
    name: option.label,
    description: option.description ?? "",
    hazardousMaterialId: selectOptionMetaString(option, "hazardousMaterialId") || null,
    stackable: selectOptionMetaBoolean(option, "stackable"),
    fragile: selectOptionMetaBoolean(option, "fragile"),
  };
}

/** What a shipment's lines add up to; a line still being typed counts as nothing. */
export function commodityTotals(lines: readonly Pick<ShipmentCommodity, "pieces" | "weight">[]): {
  pieces: number;
  weight: number;
} {
  let pieces = 0;
  let weight = 0;
  for (const line of lines) {
    if (typeof line.pieces === "number") pieces += line.pieces;
    if (typeof line.weight === "number") weight += line.weight;
  }
  return { pieces, weight };
}
