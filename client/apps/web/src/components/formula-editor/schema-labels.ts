import { defineLabels } from "@trenova/shared/i18n/labels";
export const CATEGORY_LABELS: Record<string, string> = defineLabels({
  shipment: "Shipment fields",
  customer: "Customer",
  tractorType: "Tractor equipment",
  trailerType: "Trailer equipment",
  equipment: "Equipment",
  origin: "Origin",
  destination: "Destination",
  serviceType: "Service type",
  shipmentType: "Shipment type",
  computed: "Computed rollups",
  context: "Market data",
  custom: "Custom variables",
});

export const FUNCTION_CATEGORY_LABELS: Record<string, string> = defineLabels({
  math: "Math",
  rounding: "Rounding",
  aggregate: "Aggregates",
  conditional: "Conditionals",
  rateTable: "Rate tables",
  string: "Text",
});

export function categoryLabel(category: string, labels: Record<string, string>): string {
  return labels[category] ?? (category ? category : "Other");
}
