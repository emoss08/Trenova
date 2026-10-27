import type {
  ReportCatalog,
  ReportCatalogEdge,
  ReportCatalogEntity,
  ReportCatalogField,
} from "@/lib/graphql/reports";

/**
 * A catalog as the reportCatalog query returns it for a dispatcher who reads
 * shipments but not customers or workers, and holds the Personal field group
 * but not Margins.
 */
export function field(
  overrides: Partial<ReportCatalogField> & { key: string },
): ReportCatalogField {
  return {
    label: overrides.key,
    description: null,
    type: "string",
    format: null,
    measure: false,
    enumValues: [],
    aggregations: ["count"],
    operators: ["eq", "ne", "is_null", "is_not_null"],
    filterable: true,
    groupable: true,
    access: "Available",
    inReferenceView: false,
    protection: null,
    sensitivity: "internal",
    ...overrides,
  };
}

export function edge(overrides: Partial<ReportCatalogEdge> & { name: string; target: string }) {
  return {
    label: overrides.name,
    description: null,
    toMany: false,
    pick: false,
    access: "Available",
    inReferenceView: true,
    needsReadOf: null,
    ...overrides,
  } satisfies ReportCatalogEdge;
}

function entity(
  overrides: Partial<ReportCatalogEntity> & { key: string; label: string },
): ReportCatalogEntity {
  return {
    resource: overrides.key,
    pluralLabel: `${overrides.label}s`,
    description: null,
    category: "Operations",
    ownScopeSupported: false,
    readable: true,
    exportable: true,
    fields: [],
    edges: [],
    ...overrides,
  };
}

export const shipment = entity({
  key: "shipment",
  label: "Shipment",
  fields: [
    field({ key: "proNumber", label: "PRO Number", inReferenceView: true }),
    field({ key: "status", label: "Status", access: "WithheldByCeiling" }),
    field({
      key: "marginFloorPercent",
      label: "Margin Floor",
      access: "NeedsFieldGroup",
      protection: { group: "Margins", held: false },
    }),
    field({
      key: "driverLicense",
      label: "Driver License",
      protection: { group: "Personal", held: true },
    }),
  ],
  edges: [
    edge({ name: "customer", label: "Customer", target: "customer" }),
    edge({
      name: "driverProfile",
      label: "Driver Profile",
      target: "worker_profile",
      access: "NeedsKindRead",
      inReferenceView: false,
      needsReadOf: { resource: "worker", label: "Workers" },
    }),
  ],
});

export const customer = entity({
  key: "customer",
  label: "Customer",
  readable: false,
  exportable: false,
  fields: [
    field({ key: "name", label: "Name", inReferenceView: true }),
    field({ key: "code", label: "Code", inReferenceView: true, access: "WithheldByCeiling" }),
    field({ key: "dotNumber", label: "DOT Number", access: "NeedsKindRead" }),
  ],
});

export const catalog: ReportCatalog = { version: "1", rules: {}, entities: [shipment, customer] };
