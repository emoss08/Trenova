import type { ReportRunDeskCard } from "@/lib/graphql/report-desk-card";
import type { AssistantArtifact } from "@/types/assistant";

const DISPLAY = {
  style: "text",
  decimals: 0,
  grouping: true,
  currency: "",
  negative: "minus",
  notation: "standard",
  prefix: "",
  suffix: "",
  dateStyle: "",
  boolStyle: "",
  durationUnit: "",
  durationStyle: "",
  nullText: "",
  rules: [],
  band: null,
};

const CUSTOMERS: [string, number, string][] = [
  ["Acme Foods", 162, "241880.50"],
  ["Harborline Logistics", 148, "198412.00"],
  ["Northwind Grocers", 121, "164209.75"],
  ["Blue Ridge Paper", 96, "128730.00"],
  ["Cascade Building Supply", 84, "112455.25"],
  ["Lone Star Beverage", 77, "96102.00"],
  ["Pioneer Plastics", 61, "74318.40"],
  ["Delta Farm Co-op", 52, "61988.00"],
  ["Granite State Mills", 44, "52101.10"],
  ["Sunbelt Produce", 40, "48000.00"],
];

/** A finished run of the built-in revenue report, the agent's, with the run before it. */
export function finishedCard(): ReportRunDeskCard {
  return {
    run: {
      __typename: "ReportRun",
      id: "rrun_1",
      definitionId: null,
      revisionId: null,
      cannedKey: "revenue-by-customer",
      cannedVersion: "1.0.0",
      requestedById: "usr_1",
      trigger: "manual",
      params: { windowDays: 30 },
      format: "xlsx",
      status: "succeeded",
      rowCount: 214,
      byteSize: 49152,
      durationMs: 3200,
      truncated: false,
      error: null,
      artifactExpiresAt: 1_893_456_000,
      cacheHit: false,
      queuedAt: 1_793_114_042,
      startedAt: 1_793_114_043,
      completedAt: 1_793_114_046,
      version: 2,
      createdAt: 1_793_114_042,
    } as ReportRunDeskCard["run"],
    report: {
      name: "Revenue by customer",
      description: "Freight, accessorial and total charges per customer.",
      category: "Billing",
      definitionId: null,
      cannedKey: "revenue-by-customer",
    },
    parameters: [
      {
        name: "windowDays",
        label: "Window (days)",
        type: "int",
        value: 30,
        isDefault: false,
        refEntity: null,
        windowStart: 1_790_398_800,
        windowEnd: 1_792_990_800,
      },
    ],
    timezone: "America/Chicago",
    sample: {
      columns: [
        { id: "customer", label: "Customer", type: "string", format: null, display: DISPLAY },
        {
          id: "shipments",
          label: "Shipments",
          type: "int",
          format: "count",
          display: { ...DISPLAY, style: "number" },
        },
        {
          id: "revenue",
          label: "Total revenue",
          type: "decimal",
          format: "money",
          display: { ...DISPLAY, style: "currency", currency: "USD", decimals: 2 },
        },
      ],
      rows: CUSTOMERS.map((row) => [...row]),
      totals: [null, 1142, "1284310.42"],
      more: true,
    },
    withheld: null,
    previous: {
      runId: "rrun_0",
      completedAt: 1_792_509_246,
      rowCount: 208,
      totals: new Map<string, unknown>([
        ["shipments", 1108],
        ["revenue", "1187000.00"],
      ]),
    },
    schedule: null,
    chart: null,
    sort: [{ columnId: "revenue", direction: "desc" }],
  };
}

export function runArtifact(payload: Record<string, unknown> = {}): AssistantArtifact {
  return {
    id: "art_1",
    threadId: "athr_1",
    messageId: "amsg_1",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "report_run",
    status: "ready",
    title: "Revenue by customer",
    payload: {
      runId: "rrun_1",
      reportKey: "revenue-by-customer",
      readBy: { id: "agdef_1", name: "Dispatch Desk", icon: "", accent: "teal" },
      ...payload,
    },
    sourceToolCallId: "call_1",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: "revenue-by-customer",
    turn: "",
    createdAt: 1_793_114_042,
    updatedAt: 1_793_114_046,
  } as AssistantArtifact;
}
