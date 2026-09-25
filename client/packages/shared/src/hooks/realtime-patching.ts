export const RESOURCE_EVENT_NAME = "resource.invalidation";

/**
 * Where a resource's rows are cached: either a plain key root, or the prefix a
 * query factory puts in front of one.
 *
 * `createQueryKeys("assistant", { proposals })` does not cache under the root
 * it is handed — it prepends its scope and the method name, so a component
 * reading `queries.assistant.proposals(id)` reads
 * `["assistant", "proposals", "assistant-proposals", id]`. Naming the root on
 * its own matches nothing, because TanStack matches from the start of the key,
 * and nothing reports the miss: the event arrives, the invalidation runs, and
 * the screen does not move. Five resources here were addressing rows that way.
 *
 * So a factory-backed entry spells its prefix as an array. The prefixes are
 * literals because this package cannot see the app's factories; a test in the
 * app compares each one against the live `_def` and fails if they drift.
 */
export type QueryKeyRoot = string | readonly string[];

/** One root as a key TanStack can match a cached query against. */
export function queryKeyPrefix(root: QueryKeyRoot): readonly string[] {
  return typeof root === "string" ? [root] : root;
}

/** A stable identity for a root, so a coalescing pass can deduplicate them. */
export function queryKeyRootId(root: QueryKeyRoot): string {
  return queryKeyPrefix(root).join("\u0000");
}

export const RESOURCE_QUERY_KEY_MAP: Record<string, QueryKeyRoot[]> = {
  // A proposal or plan moving can change what another one would do (a step
  // after it on the same record, one decided elsewhere now reading as
  // recorded), so the previews go with them.
  agent_proposal: [
    ["assistant", "proposals"],
    "agent-proposal-list",
    "pending-decisions",
    "pending-decision-summary",
    "attention",
    "agentPreview",
  ],
  agent_plan: [
    ["assistant", "plans"],
    "agent-plan-list",
    "pending-decisions",
    "pending-decision-summary",
    "attention",
    "agentPreview",
  ],
  agent_run: ["agent-run-list", ["assistant", "agents"]],
  // aiauditservice announces an export to its requester as it is written,
  // expires or fails, and the chain once a check has stored its result.
  "ai-audit-export": ["ai-audit-export-list"],
  "ai-audit-chain": [["aiAudit", "chainStatus"]],
  accounting_integration: ["accountingSync"],
  assistant_artifact: [["assistant", "artifacts"]],
  // A reply starting or closing moves the "writing" markers and the
  // conversation's place in the list together.
  assistant_turns: [
    ["assistant", "activeTurns"],
    ["assistant", "threads"],
  ],
  // The feed and its counts live under one key root from the query factory
  // (createQueryKeys("watchtower")), so invalidating the root catches both the
  // list and every filtered variant of it.
  watchtower: ["watchtower", "attention"],
  briefings: ["briefing"],
  // Every inbox write: the lanes, their counts and the open message move
  // together. Mailbox administration is not a message and is left alone.
  inbound_message: [
    ["inbox", "messages"],
    ["inbox", "counts"],
    ["inbox", "message"],
  ],
  shipments: [
    "shipment-list",
    "dispatch-board",
    "dispatch-live-tender",
    "dispatch-shipment-tenders",
  ],
  orders: ["order-list", "order-detail"],
  users: ["user-list"],
  customers: ["customer-list"],
  tractors: ["tractor-list"],
  trailers: ["trailer-list"],
  workers: ["worker-list", "dispatch-board"],
  "audit-logs": ["audit-entry-list"],
  billing_queue: ["billing-queue-list", "billingQueue"],
  "billing-transfer-run": [
    "billing-transfer-run",
    "billing-transfer-active-run",
    "billing-transfer-run-items",
  ],
  shipmentEvents: ["shipment-events"],
  "report-run": ["report-run-list"],
  routing_guides: ["routing-guide-list"],
  driver_settlement: [
    "driver-settlement-list",
    "driver-settlement-detail",
    "settlement-workspace-summary",
    "settlement-workspace-settlements",
  ],
  driver_pay_event: ["driver-pay-event-list"],
  settlement_dispute: ["settlement-dispute-list", "settlement-dispute-detail"],
  driver_expense: ["driver-expense-list", "driver-expense-detail", "pending-driver-expense-count"],
  worker_pto: ["worker-pto-list", "worker", "dispatch-board", "worker-pto-balances"],
  worker_pto_balance: ["worker-pto-balances", "worker-pto-ledger", "worker", "pto-balance-summary"],
  pto_policy: ["pto-policy-list", "pto-policy-options", "worker"],
  org_holiday: ["org-holidays"],
  worker_training: ["worker-training", "worker-training-summary", "worker", "dash-training"],
  training_course: ["training-course-list", "training-courses", "worker-training-summary"],
  worker_safety_event: ["worker-safety-events", "worker-safety-scorecard", "dash-safety"],
  worker_disciplinary_action: [
    "worker-disciplinary-actions",
    "worker-disciplinary-ladder",
    "worker-safety-scorecard",
    "dash-discipline",
    "dash-safety",
  ],
  worker_recognition: ["worker-recognitions", "worker-safety-scorecard", "dash-recognitions"],
  performance_review: ["worker-reviews", "dash-reviews"],
  performance_review_template: ["review-template-list", "review-templates"],
  worker_credential: [
    "worker-credentials",
    "worker-credential-summary",
    "credential-expiry-forecast",
    "worker",
    "dash-credentials",
  ],
  worker_employment_event: ["worker-employment-events", "worker", "worker-list"],
  worker_checklist: ["worker-checklists", "worker", "worker-list"],
  worker_checklist_template: ["worker-checklist-template-list", "worker-checklist-templates"],
  worker_credential_type: [
    "worker-credential-type-list",
    "worker-credential-types",
    "worker-credential-summary",
  ],
  dash_control: ["dash-control"],
  carrier_intel_events: [
    "carrier-intel-event-list",
    "carrier-intel-events",
    "carrier-intelligence",
    "carrierIntelSettings",
  ],
  carrier_monitoring_enrollments: [
    "carrier-monitoring-enrollment-list",
    "carrier-intelligence",
    "carrierIntelSettings",
  ],
  carrier_intelligence: [
    "carrier-intel-review-queue",
    "carrier-intelligence",
    "carrier-intel-history",
    "carrierIntelSettings",
  ],
  carrier_intelligence_control: ["carrierIntelSettings"],
  vehiclePosition: ["telematics", "dispatch-board"],
  workerHosState: ["telematics", "dispatch-board"],
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

export const CORE_QUERY_KEYS: QueryKeyRoot[] = Array.from(
  new Map(
    Object.values(RESOURCE_QUERY_KEY_MAP)
      .flat()
      .map((root) => [queryKeyRootId(root), root] as const),
  ).values(),
);

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
