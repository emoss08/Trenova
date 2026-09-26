/**
 * Where a record opens.
 *
 * Every link to a single record is built from this table: the app's own links,
 * the assistant's page context, and — through the generated product guide —
 * every link the server hands out. It used to be built in some forty places
 * with three different query conventions, and the copies drifted: the
 * assistant's links pointed at `/shipments`, which is not a page.
 *
 * `{id}` stands for the record's id wherever it appears, in the path or in a
 * parameter's value. The product guide generator reads this file statically,
 * so entries stay plain literals.
 */
export type RecordLink = {
  /** What the record is called, for a link's label. */
  label: string;
  /** The page it opens on; `{id}` marks a path segment carrying the id. */
  path: string;
  /** Query parameters that open the record there. */
  params?: Readonly<Record<string, string>>;
};

export const RECORD_LINKS = {
  shipment: {
    label: "Shipment",
    path: "/shipment-management/shipments",
    params: { expanded: "{id}", panelType: "edit", panelEntityId: "{id}" },
  },
  order: {
    label: "Order",
    path: "/shipment-management/orders",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  service_failure: {
    label: "Service failure",
    path: "/shipment-management/service-failures",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  worker: {
    label: "Worker",
    path: "/hr/workers",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  tractor: {
    label: "Tractor",
    path: "/equipment/tractors",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  trailer: {
    label: "Trailer",
    path: "/equipment/trailers",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  customer: {
    label: "Customer",
    path: "/billing/configuration-files/customers",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  carrier: {
    label: "Carrier",
    path: "/dispatch/carriers",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  location: {
    label: "Location",
    path: "/dispatch/locations",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  rate_matrix: {
    label: "Rate matrix",
    path: "/billing/configuration-files/rate-matrices",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  billing_queue_item: {
    label: "Billing queue item",
    path: "/billing/queue",
    params: { item: "{id}" },
  },
  carrier_settlement: {
    label: "Carrier settlement",
    path: "/carrier-settlements/settlements",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  driver_settlement: {
    label: "Driver settlement",
    path: "/payroll/settlements",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  invoice: {
    label: "Invoice",
    path: "/billing/invoices",
    params: { item: "{id}" },
  },
  detention_occurrence: {
    label: "Detention charge",
    path: "/detention/desk",
    params: { stop: "{id}" },
  },
  edi_inbound_file: {
    label: "EDI inbound file",
    path: "/edi/inbound-files",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  report: {
    label: "Report",
    path: "/reports/explore/{id}",
  },
  dashboard: {
    label: "Dashboard",
    path: "/reports/dashboards/{id}",
  },
  assistant_thread: {
    label: "Conversation",
    path: "/desk/t/{id}",
  },
  accounting_sync_record: {
    label: "Sync record",
    path: "/accounting/sync",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  accounting_inbound_change: {
    label: "Payment from the books",
    path: "/accounting/sync/inbound",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  accounting_drift_finding: {
    label: "Drift finding",
    path: "/accounting/sync/drift",
    params: { panelType: "edit", panelEntityId: "{id}" },
  },
  journal_entry: {
    label: "Journal entry",
    path: "/accounting/journal-entries/{id}",
  },
  agent_run: {
    label: "Agent run",
    path: "/admin/agent-control",
    params: {
      tab: "activity",
      activity: "runs",
      fieldFilters: '[{"field":"id","operator":"eq","value":"{id}"}]',
    },
  },
  ai_audit_event: {
    label: "AI audit event",
    path: "/admin/agent-control",
    params: { tab: "audit", audit: "trail", panelType: "edit", panelEntityId: "{id}" },
  },
  ai_audit_export: {
    label: "AI audit trail export",
    path: "/admin/agent-control",
    params: {
      tab: "audit",
      audit: "exports",
      fieldFilters: '[{"field":"id","operator":"eq","value":"{id}"}]',
    },
  },
} as const satisfies Record<string, RecordLink>;

export type RecordEntityType = keyof typeof RECORD_LINKS;

const ID = "{id}";

export function isRecordEntityType(value: string): value is RecordEntityType {
  return Object.hasOwn(RECORD_LINKS, value);
}

/** The address that opens one record. */
export function recordPath(
  entity: RecordEntityType,
  id: string,
  extra?: Readonly<Record<string, string>>,
): string {
  const link: RecordLink = RECORD_LINKS[entity];
  const path = link.path.replaceAll(ID, encodeURIComponent(id));
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(link.params ?? {})) {
    params.set(key, value.replaceAll(ID, id));
  }
  for (const [key, value] of Object.entries(extra ?? {})) {
    params.set(key, value);
  }

  const query = params.toString();
  return query === "" ? path : `${path}?${query}`;
}

export type RecordAtLocation = {
  entityType: RecordEntityType;
  /** Empty when the page is the record's page but no record is open. */
  entityId: string;
};

function templatePattern(template: string, capture: string): string {
  return template
    .split(ID)
    .map((part) => part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
    .join(capture);
}

function pathPattern(path: string): RegExp {
  return new RegExp(`^${templatePattern(path, "([^/]+)")}/?$`);
}

function valuePattern(template: string): RegExp {
  return new RegExp(`^${templatePattern(template, "(.+?)")}$`);
}

const LOCATORS = (Object.keys(RECORD_LINKS) as RecordEntityType[])
  .map((entityType) => {
    const link: RecordLink = RECORD_LINKS[entityType];
    const entries = Object.entries(link.params ?? {});
    const idParams = entries
      .filter(([, value]) => value.includes(ID))
      .map(([key, value]) => ({ key, pattern: valuePattern(value) }));
    const fixedParams = entries.filter(([, value]) => !value.includes(ID));
    return {
      entityType,
      pattern: pathPattern(link.path),
      idParams,
      fixedParams,
      specificity: link.path.length,
    };
  })
  // Longest path first, so a page nested under another is not claimed by it.
  .sort((a, b) => b.specificity - a.specificity);

type Locator = (typeof LOCATORS)[number];

/**
 * The record a location shows, when it is one of the pages records open on:
 * the reverse of `recordPath`. Several kinds of record can open on one page
 * (AI Control holds agent runs and the audit trail's events and exports);
 * the one whose fixed parameters the address carries is the one it shows,
 * and an address naming none of them reads as the first.
 */
export function recordAtLocation(pathname: string, search: string): RecordAtLocation | null {
  const params = new URLSearchParams(search);
  let chosen: { locator: Locator; match: RegExpExecArray } | null = null;
  for (const locator of LOCATORS) {
    const match = locator.pattern.exec(pathname);
    if (!match) {
      continue;
    }
    if (locator.fixedParams.every(([key, value]) => params.get(key) === value)) {
      chosen = { locator, match };
      break;
    }
    chosen ??= { locator, match };
  }
  if (chosen === null) {
    return null;
  }

  const { locator, match } = chosen;
  const fromPath = match[1] === undefined ? "" : decodeURIComponent(match[1]);
  let fromQuery = "";
  for (const idParam of locator.idParams) {
    const found = idParam.pattern.exec(params.get(idParam.key) ?? "");
    if (found?.[1]) {
      fromQuery = found[1];
      break;
    }
  }
  return { entityType: locator.entityType, entityId: (fromPath || fromQuery).trim() };
}
