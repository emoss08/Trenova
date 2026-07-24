export const RESOURCE_EVENT_NAME = "resource.invalidation";

export const RESOURCE_QUERY_KEY_MAP: Record<string, string[]> = {
  shipments: ["shipment-list"],
  orders: ["order-list", "order-detail"],
  users: ["user-list"],
  customers: ["customer-list"],
  tractors: ["tractor-list"],
  trailers: ["trailer-list"],
  workers: ["worker-list"],
  "audit-logs": ["audit-entry-list"],
  billing_queue: ["billing-queue-list", "billingQueue"],
  shipmentEvents: ["shipment-events"],
  "report-run": ["report-run-list"],
  shipment_comment: ["shipment-comments", "shipment-comment-count"],
  driver_settlement: [
    "driver-settlement-list",
    "driver-settlement-detail",
    "settlement-workspace-summary",
    "settlement-workspace-settlements",
  ],
  driver_pay_event: ["driver-pay-event-list"],
  settlement_dispute: ["settlement-dispute-list", "settlement-dispute-detail"],
  driver_expense: ["driver-expense-list", "driver-expense-detail", "pending-driver-expense-count"],
  worker_pto: ["worker-pto-list", "worker"],
  dash_control: ["dash-control"],
  vehiclePosition: ["telematics"],
  workerHosState: ["telematics"],
  workerHosViolation: ["telematics"],
  vehicleInspection: ["telematics"],
  telematicsEvent: ["telematics"],
};

export const PATCHABLE_FIELDS_BY_RESOURCE: Record<string, Set<string>> = {
  shipments: new Set(["status", "proNumber", "moves", "updatedAt"]),
  users: new Set([
    "status",
    "name",
    "emailAddress",
    "username",
    "thumbnailUrl",
    "lastLoginAt",
    "updatedAt",
  ]),
  customers: new Set(["status", "name", "code", "emailAddress", "updatedAt"]),
  tractors: new Set(["status", "code", "updatedAt"]),
  trailers: new Set(["status", "code", "updatedAt"]),
  workers: new Set(["status", "firstName", "lastName", "updatedAt"]),
};

export const CORE_QUERY_KEYS = Array.from(new Set(Object.values(RESOURCE_QUERY_KEY_MAP).flat()));

export interface ResourceInvalidationEvent {
  type?: string;
  organizationId: string;
  businessUnitId: string;
  resource: string;
  action?: string;
  fields?: string[];
  entityId?: string;
  recordId?: string;
  entity?: Record<string, unknown>;
}

export function parseInvalidationEvent(payload: unknown): ResourceInvalidationEvent | null {
  if (!payload) return null;

  let data: unknown = payload;
  if (typeof payload === "string") {
    try {
      data = JSON.parse(payload);
    } catch {
      return null;
    }
  }

  if (
    typeof data !== "object" ||
    data === null ||
    !("organizationId" in data) ||
    !("businessUnitId" in data) ||
    !("resource" in data)
  ) {
    return null;
  }

  return data as ResourceInvalidationEvent;
}

export function isBulkAction(action: string) {
  return action.startsWith("bulk_");
}

export function resolveEntityID(event: ResourceInvalidationEvent) {
  const fromEvent = event.entityId || event.recordId;
  if (fromEvent) return fromEvent;

  const entity = event.entity;
  if (!entity || typeof entity !== "object") return "";

  return typeof entity.id === "string" ? entity.id : "";
}

export function shouldPatchEvent(event: ResourceInvalidationEvent) {
  const action = event.action ?? "";
  const entityID = resolveEntityID(event);
  if (action !== "updated" || !entityID || !event.entity) return false;

  const patchableFields = PATCHABLE_FIELDS_BY_RESOURCE[event.resource];
  if (!patchableFields) return false;

  if (!event.fields || event.fields.length === 0) {
    return true;
  }

  return event.fields.every((field) => patchableFields.has(field));
}

export function patchEntityInListRows(
  current: unknown,
  event: ResourceInvalidationEvent,
): { data: unknown; patched: boolean } {
  const entityID = resolveEntityID(event);
  const entity = event.entity;
  if (!entityID || !entity || !hasRowsShape(current)) {
    return { data: current, patched: false };
  }

  const index = current.results.findIndex((row) => row.id === entityID);
  if (index < 0) {
    return { data: current, patched: false };
  }

  const nextResults = [...current.results];
  nextResults[index] = {
    ...nextResults[index],
    ...entity,
  };

  return {
    data: {
      ...current,
      results: nextResults,
    },
    patched: true,
  };
}

function hasRowsShape(value: unknown): value is { results: Record<string, unknown>[] } {
  return (
    !!value &&
    typeof value === "object" &&
    "results" in value &&
    Array.isArray((value as { results: unknown[] }).results)
  );
}
