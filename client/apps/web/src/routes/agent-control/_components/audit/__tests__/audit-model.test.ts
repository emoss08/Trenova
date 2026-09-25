import type {
  AiAuditEventKind,
  AiAuditEventOutcome,
  AiAuditExportStatus,
} from "@trenova/graphql/generated/graphql";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { describe, expect, it } from "vitest";
import {
  AUDIT_MAX_EXPORT_RANGE_SECONDS,
  DEFAULT_AUDIT_SCOPE,
  auditEventRecordPath,
  auditKindLabel,
  auditOutcomeAttrs,
  auditScopeFilters,
  auditScopeRange,
  auditExportStatusAttrs,
  buildAuditExportRequest,
  parseAuditScope,
  verificationTone,
  type AuditTrailScope,
} from "../audit-model";

const t = (text: string | null | undefined, ...args: unknown[]) =>
  (text ?? "").replace(/\{(\d+)\}/g, (_m, i) => String(args[Number(i)]));

/*
Every value the schema's enums can take, written out from
services/tms/internal/api/graphql/schema/aiaudit.graphqls rather than read
from the model, so a value the model forgets fails here instead of rendering
as a blank badge.
*/
const OUTCOMES: AiAuditEventOutcome[] = [
  "Started",
  "Completed",
  "Failed",
  "Refused",
  "Stopped",
  "Succeeded",
  "Ran",
  "Proposed",
  "Simulated",
  "Denied",
  "Unknown",
  "Filed",
  "Accepted",
  "Modified",
  "Rejected",
  "Expired",
  "Exhausted",
  "Declined",
];

const KINDS: AiAuditEventKind[] = [
  "RunStarted",
  "RunEnded",
  "ModelCall",
  "ToolCall",
  "ToolRefused",
  "ProposalFiled",
  "ProposalDecided",
  "ProposalExecuted",
  "ProposalExecutionFailed",
  "ProposalSimulated",
  "ProposalExpired",
  "DelegationStarted",
  "DelegationEnded",
];

const EXPORT_STATUSES: AiAuditExportStatus[] = [
  "Pending",
  "Running",
  "Succeeded",
  "Failed",
  "Expired",
];

describe("auditOutcomeAttrs", () => {
  const attrs = auditOutcomeAttrs(t);

  it("names a phase and a label for every outcome the schema allows", () => {
    for (const outcome of OUTCOMES) {
      expect(attrs[outcome], outcome).toBeDefined();
      expect(attrs[outcome].text, outcome).not.toBe("");
    }
  });

  // The tone follows the phase: an outcome waiting on a person is a warning,
  // a write that happened is a success, and a refusal, a rejection or an
  // expiry is a failure. An outcome that cannot be known is something to
  // look at, not a success.
  it("puts each outcome in the phase that says what a person should do", () => {
    expect(attrs.Proposed.phase).toBe("awaiting");
    expect(attrs.Filed.phase).toBe("awaiting");
    expect(attrs.Started.phase).toBe("active");
    expect(attrs.Ran.phase).toBe("complete");
    expect(attrs.Succeeded.phase).toBe("complete");
    expect(attrs.Completed.phase).toBe("complete");
    expect(attrs.Accepted.phase).toBe("complete");
    expect(attrs.Modified.phase).toBe("complete");
    expect(attrs.Failed.phase).toBe("failed");
    expect(attrs.Denied.phase).toBe("failed");
    expect(attrs.Rejected.phase).toBe("failed");
    expect(attrs.Expired.phase).toBe("failed");
    expect(attrs.Refused.phase).toBe("attention");
    expect(attrs.Unknown.phase).toBe("attention");
    expect(attrs.Exhausted.phase).toBe("attention");
    expect(attrs.Simulated.phase).toBe("closed");
    expect(attrs.Stopped.phase).toBe("closed");
    expect(attrs.Declined.phase).toBe("closed");
    expect(phaseTone(attrs.Unknown.phase)).toBe("warning");
    expect(phaseTone(attrs.Denied.phase)).toBe("danger");
  });

  it("labels every kind", () => {
    for (const kind of KINDS) {
      expect(auditKindLabel(t, kind), kind).not.toBe(kind);
      expect(auditKindLabel(t, kind), kind).not.toBe("");
    }
    expect(auditKindLabel(t, "ProposalExecutionFailed")).toBe("Proposal failed to run");
  });

  it("names a phase for every export status", () => {
    const exportAttrs = auditExportStatusAttrs(t);
    for (const status of EXPORT_STATUSES) {
      expect(exportAttrs[status], status).toBeDefined();
    }
    expect(exportAttrs.Running.phase).toBe("active");
    expect(exportAttrs.Succeeded.phase).toBe("complete");
    expect(exportAttrs.Failed.phase).toBe("failed");
    expect(exportAttrs.Expired.phase).toBe("closed");
  });

  it("tones the chain's verification by how bad it is", () => {
    expect(verificationTone("Verified")).toBe("success");
    expect(verificationTone("KeyMissing")).toBe("warning");
    expect(verificationTone("Mismatch")).toBe("danger");
    expect(verificationTone(null)).toBe("muted");
  });
});

describe("auditScopeFilters", () => {
  // Evaluations replay a run against the agent as it is now; they are not
  // live work, so the trail leaves them out until asked.
  it("shows the last seven days of live work by default", () => {
    expect(auditScopeFilters(DEFAULT_AUDIT_SCOPE)).toEqual([
      { field: "occurredAt", operator: "lastndays", value: 7 },
      { field: "purpose", operator: "eq", value: "Live" },
    ]);
  });

  // The server rewrites personId into "for this person or decided by this
  // person"; the client names the person once and never spells the pair.
  it("narrows to an agent and to a person by the fields the server reads", () => {
    const scope: AuditTrailScope = {
      ...DEFAULT_AUDIT_SCOPE,
      range: { kind: "last", days: 30 },
      agent: { id: "agdef_1", name: "Billing desk" },
      personId: "usr_1",
      includeEvaluations: true,
    };

    expect(auditScopeFilters(scope)).toEqual([
      { field: "occurredAt", operator: "lastndays", value: 30 },
      { field: "agentDefinitionId", operator: "eq", value: "agdef_1" },
      { field: "personId", operator: "eq", value: "usr_1" },
    ]);
  });

  // A picked range is whole days in the reader's zone: its first second to
  // its last, both inclusive, which is how the server reads gte and lte.
  it("sends a picked range as its first and last second", () => {
    const scope: AuditTrailScope = {
      ...DEFAULT_AUDIT_SCOPE,
      range: { kind: "between", from: 1_760_000_000, to: 1_760_604_799 },
    };

    expect(auditScopeFilters(scope)).toEqual([
      { field: "occurredAt", operator: "gte", value: 1_760_000_000 },
      { field: "occurredAt", operator: "lte", value: 1_760_604_799 },
      { field: "purpose", operator: "eq", value: "Live" },
    ]);
  });

  it("resolves a relative range against now, and a picked one as it is", () => {
    const now = 1_760_000_000;
    expect(auditScopeRange(DEFAULT_AUDIT_SCOPE, now)).toEqual({
      from: now - 7 * 86_400,
      to: now,
    });
    expect(
      auditScopeRange(
        { ...DEFAULT_AUDIT_SCOPE, range: { kind: "between", from: 10, to: 20 } },
        now,
      ),
    ).toEqual({ from: 10, to: 20 });
  });
});

describe("parseAuditScope", () => {
  it("reads back what it wrote", () => {
    const scope: AuditTrailScope = {
      range: { kind: "between", from: 100, to: 200 },
      agent: { id: "agdef_1", name: "Billing desk" },
      personId: "usr_1",
      includeEvaluations: true,
    };

    expect(parseAuditScope(JSON.parse(JSON.stringify(scope)))).toEqual(scope);
  });

  // The address is typed by hand and shared; a value the model does not
  // know is dropped rather than sent to the server as a filter.
  it("refuses what the address cannot mean", () => {
    expect(parseAuditScope({ range: { kind: "last", days: 12 } })).toBeNull();
    expect(parseAuditScope({ range: { kind: "between", from: 200, to: 100 } })).toBeNull();
    expect(parseAuditScope("nonsense")).toBeNull();
  });
});

describe("buildAuditExportRequest", () => {
  const table = {
    query: "reassign",
    fieldFilters: [{ field: "kind", operator: "eq" as const, value: "ToolCall" }],
    filterGroups: [
      {
        filters: [
          { field: "outcome", operator: "eq" as const, value: "Denied" },
          { field: "outcome", operator: "eq" as const, value: "Failed" },
        ],
      },
    ],
    sort: [{ field: "occurredAt", direction: "asc" as const }],
  };
  const scope: AuditTrailScope = {
    ...DEFAULT_AUDIT_SCOPE,
    agent: { id: "agdef_1", name: "Billing desk" },
    personId: "usr_1",
  };

  // Unfiltered, the file holds every row in the range, which is the only
  // file whose chain an auditor can check end to end.
  it("sends only the range and format when current filters are not used", () => {
    expect(
      buildAuditExportRequest({
        format: "CSV",
        from: 100,
        to: 200,
        useCurrentFilters: false,
        scope,
        table,
      }),
    ).toEqual({ format: "CSV", from: 100, to: 200 });
  });

  // The dialog's range replaces the scope's, so the scope's own date filter
  // is not sent twice; everything else the reader narrowed by is.
  it("sends the scope and the table's own filters, search and sort when asked", () => {
    expect(
      buildAuditExportRequest({
        format: "JSON",
        from: 100,
        to: 200,
        useCurrentFilters: true,
        scope,
        table,
      }),
    ).toEqual({
      format: "JSON",
      from: 100,
      to: 200,
      query: "reassign",
      fieldFilters: [
        { field: "agentDefinitionId", operator: "eq", value: "agdef_1" },
        { field: "personId", operator: "eq", value: "usr_1" },
        { field: "purpose", operator: "eq", value: "Live" },
        { field: "kind", operator: "eq", value: "ToolCall" },
      ],
      filterGroups: table.filterGroups,
      sort: table.sort,
    });
  });

  it("leaves out an empty search", () => {
    const request = buildAuditExportRequest({
      format: "CSV",
      from: 1,
      to: 2,
      useCurrentFilters: true,
      scope: DEFAULT_AUDIT_SCOPE,
      table: { query: "  ", fieldFilters: [], filterGroups: [], sort: [] },
    });

    expect(request).toEqual({
      format: "CSV",
      from: 1,
      to: 2,
      fieldFilters: [{ field: "purpose", operator: "eq", value: "Live" }],
      filterGroups: [],
      sort: [],
    });
  });

  it("caps the range at the server's seven years", () => {
    expect(AUDIT_MAX_EXPORT_RANGE_SECONDS).toBe(2556 * 86_400);
  });
});

describe("auditEventRecordPath", () => {
  // The trail records an agent's subject by its subject type and a tool's
  // target by its permission resource; both open where the record lives.
  it("opens a record named either way on its own page", () => {
    expect(auditEventRecordPath("Shipment", "shp_1")).toBe(
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
    expect(auditEventRecordPath("worker", "wrk_1")).toBe(
      "/hr/workers?panelType=edit&panelEntityId=wrk_1",
    );
  });

  it("has no link for a record without a page or without an id", () => {
    expect(auditEventRecordPath("Insight", "ins_1")).toBeNull();
    expect(auditEventRecordPath("shipment", "")).toBeNull();
    expect(auditEventRecordPath(null, "shp_1")).toBeNull();
  });
});
