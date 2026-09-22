export const CATEGORY_LABELS: Record<string, string> = {
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
};

export const FUNCTION_CATEGORY_LABELS: Record<string, string> = {
  math: "Math",
  rounding: "Rounding",
  aggregate: "Aggregates",
  conditional: "Conditionals",
  rateTable: "Rate tables",
  string: "Text",
};

export function categoryLabel(category: string, labels: Record<string, string>): string {
  return labels[category] ?? (category ? category : "Other");
}

/** Which control a sample-data field should render for a schema type. */
export type SampleInputKind = "number" | "boolean" | "enum" | "text";

export function sampleInputKind(type: string, enumValues?: string[] | null): SampleInputKind {
  if (enumValues && enumValues.length > 0) return "enum";
  switch (type.toLowerCase()) {
    case "number":
    case "integer":
      return "number";
    case "boolean":
      return "boolean";
    default:
      return "text";
  }
}
