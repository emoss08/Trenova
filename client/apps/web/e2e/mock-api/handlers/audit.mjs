// The audit log page and Data retention: entries (filtered by the table's field filters and
// search), one entry's detail, the same request's entries, a field's history, the seal status,
// legal holds, exports, alert policies, and the retention policy the page reads and saves.
import { createAuditState } from "../fixtures/audit.mjs";
import { connection } from "./formula.mjs";

const DAY = 86400;
const MAX_EXPORT_DAYS = 400;

function audit(state) {
  state.audit ??= createAuditState(state.anchor);
  return state.audit;
}

function validation(field, message) {
  const error = new Error(message);
  error.extensions = { code: "VALIDATION_ERROR", field };
  throw error;
}

const DETAIL_FIELDS = new Set([
  "changes",
  "previousState",
  "currentState",
  "metadata",
  "userAgent",
  "traceId",
  "apiKeyId",
]);

function summary(entry) {
  return Object.fromEntries(Object.entries(entry).filter(([key]) => !DETAIL_FIELDS.has(key)));
}

function matchesFilter(entry, filter) {
  const value = entry[filter.field];
  const wanted = Array.isArray(filter.value) ? filter.value.map(String) : [String(filter.value)];
  switch (filter.operator) {
    case "eq":
      return String(value) === wanted[0];
    case "in":
      return wanted.includes(String(value));
    case "contains":
      return String(value ?? "")
        .toLowerCase()
        .includes(wanted[0].toLowerCase());
    default:
      return true;
  }
}

function filtered(state, input = {}) {
  const query = (input.query ?? "").trim().toLowerCase();
  return audit(state)
    .entries.filter((entry) => (input.fieldFilters ?? []).every((f) => matchesFilter(entry, f)))
    .filter(
      (entry) =>
        !query ||
        [entry.entityRef, entry.comment, entry.resourceId].some((text) =>
          String(text ?? "")
            .toLowerCase()
            .includes(query),
        ),
    )
    .toSorted((a, b) => b.timestamp - a.timestamp);
}

function placeHold(state, { input }) {
  if (!input.name?.trim()) validation("name", "Name is required");
  if (!input.reason?.trim()) validation("reason", "Reason is required");
  const hold = {
    id: `alh_${audit(state).holds.length + 1}`,
    name: input.name.trim(),
    reason: input.reason.trim(),
    resource: input.resource ?? null,
    resourceId: input.resourceId ?? null,
    placedAt: state.now(),
    releasedAt: null,
    releaseReason: null,
    active: true,
    placedBy: { id: "usr_mock", name: "System Administrator" },
    releasedBy: null,
  };
  audit(state).holds.unshift(hold);
  return { placeAuditLegalHold: hold };
}

function releaseHold(state, { id, reason }) {
  const hold = audit(state).holds.find((candidate) => candidate.id === id);
  if (!hold) throw new Error("legal hold not found");
  if (!reason?.trim()) validation("reason", "Reason is required");
  Object.assign(hold, {
    active: false,
    releasedAt: state.now(),
    releaseReason: reason.trim(),
    releasedBy: { id: "usr_mock", name: "System Administrator" },
  });
  return { releaseAuditLegalHold: hold };
}

function requestExport(state, { input }) {
  if (input.rangeTo < input.rangeFrom) validation("rangeTo", "The last day is before the first");
  if ((input.rangeTo - input.rangeFrom) / DAY > MAX_EXPORT_DAYS) {
    validation("rangeTo", "An export covers at most 400 days");
  }
  const created = {
    id: `aex_${audit(state).exports.length + 1}`,
    format: input.format,
    rangeFrom: input.rangeFrom,
    rangeTo: input.rangeTo,
    includeSecurity: !!input.includeSecurity,
    status: "Pending",
    rowCount: 0,
    byteSize: 0,
    sha256: null,
    artifactExpiresAt: null,
    errorMessage: null,
    completedAt: null,
    createdAt: state.now(),
    downloadable: false,
  };
  audit(state).exports.unshift(created);
  return { requestAuditExport: created };
}

function updatePolicy(state, { input }) {
  const policy = audit(state).policies.find((candidate) => candidate.kind === input.kind);
  if (!policy) throw new Error("no such alert");
  if (policy.counted && (input.threshold < 2 || input.threshold > 100000)) {
    validation("threshold", "A count between 2 and 100,000 raises it");
  }
  Object.assign(policy, {
    enabled: input.enabled,
    threshold: policy.counted ? input.threshold : 1,
    windowMinutes: policy.counted ? input.windowMinutes : 1,
  });
  return { updateAuditAlertPolicy: policy };
}

export const AUDIT_HANDLERS = {
  AuditLogTable: (state, v) => ({
    auditEntries: { ...connection(filtered(state, v.input), v, summary), totalCountExact: true },
  }),
  AuditEntryDetail: (state, v) => ({
    auditEntry: audit(state).entries.find((entry) => entry.id === v.id) ?? null,
  }),
  AuditRequestEntries: (state, v) => ({
    auditEntries: connection(filtered(state, v.input), v, summary),
  }),
  AuditFieldHistory: (state, v) => ({
    auditEntriesByResourceId: connection(
      filtered(state, v.input).filter(
        (entry) => entry.resourceId === v.resourceId && entry.changes && v.field in entry.changes,
      ),
      v,
      (entry) => entry,
    ),
  }),
  AuditChainStatus: (state) => ({ auditChainStatus: audit(state).chain }),
  AuditLegalHoldTable: (state, v) => ({
    auditLegalHolds: {
      ...connection(audit(state).holds, v, (hold) => hold),
      totalCountExact: true,
    },
  }),
  PlaceAuditLegalHold: placeHold,
  ReleaseAuditLegalHold: releaseHold,
  AuditExportTable: (state, v) => ({
    auditExports: { ...connection(audit(state).exports, v, (row) => row), totalCountExact: true },
  }),
  RequestAuditExport: requestExport,
  AuditExportDownload: (state, { id }) => ({
    auditExportDownload: {
      url: `http://localhost:8080/__mock/files/${id}`,
      fileName: `audit-log-${id}.jsonl`,
      expiresAt: state.now() + 300,
      sha256: audit(state).exports.find((row) => row.id === id)?.sha256 ?? null,
    },
  }),
  AuditAlertPolicies: (state) => ({ auditAlertPolicies: audit(state).policies }),
  UpdateAuditAlertPolicy: updatePolicy,
};

function saveRetention(state, _path, body = {}) {
  const retention = audit(state).retention;
  const floors = {
    auditAccessRetentionDays: [90, "Reads and downloads are kept for at least 90 days"],
    auditOperationalRetentionDays: [730, "Operational changes are kept for at least 730 days"],
    auditFinancialRetentionDays: [2555, "Financial changes are kept for at least 2,555 days"],
    auditSecurityRetentionDays: [730, "Security entries are kept for at least 730 days"],
  };
  for (const [field, [floor, message]] of Object.entries(floors)) {
    if (field in body && (body[field] < floor || body[field] > 3650)) {
      return {
        __status: 422,
        body: {
          type: "validation-error",
          title: "Validation Failed",
          status: 422,
          detail: message,
          errors: [{ field, code: "INVALID", message, location: "body" }],
        },
      };
    }
  }
  Object.assign(retention, body, { version: retention.version + 1 });
  return retention;
}

export const AUDIT_ROUTES = [
  ["GET", /^\/api\/v1\/data-retention\/?$/, (state) => audit(state).retention],
  ["PUT", /^\/api\/v1\/data-retention\/?$/, saveRetention],
];
