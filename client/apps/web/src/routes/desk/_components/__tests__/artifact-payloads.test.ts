import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  emailDraftFrom,
  entityCardFrom,
  planFrom,
  reportPreviewFrom,
  reportRunFrom,
  sortTableRows,
  tableViewFrom,
} from "../artifacts/artifact-payloads";

function artifact(kind: AssistantArtifact["kind"], payload: Record<string, unknown>) {
  return {
    id: "art_1",
    threadId: "athr_1",
    kind,
    status: "Ready",
    title: "Thing",
    payload,
    pinned: false,
    createdAt: 1,
    updatedAt: 1,
  } as AssistantArtifact;
}

/**
 * The server stores each artifact's payload in the shape the tool published
 * it. These readers turn that into what a renderer needs and refuse to
 * guess: a row the columns cannot name reads as empty, not as a crash.
 */
describe("reportPreviewFrom", () => {
  it("lines rows up under their columns by label", () => {
    const preview = reportPreviewFrom(
      artifact("report_preview", {
        columns: [
          { id: "customer", label: "Customer", type: "string" },
          { id: "revenue", label: "Revenue", type: "decimal", format: "currency" },
        ],
        rows: [{ Customer: "Acme", Revenue: "120.50" }, { Revenue: "9.00" }],
        totals: { Revenue: "129.50" },
        rowCount: 2,
        truncated: true,
      }),
    );
    expect(preview.columns.map((column) => column.id)).toEqual(["customer", "revenue"]);
    expect(preview.rows).toEqual([
      ["Acme", "120.50"],
      [null, "9.00"],
    ]);
    expect(preview.totals).toEqual([null, "129.50"]);
    expect(preview.truncated).toBe(true);
    expect(preview.rowCount).toBe(2);
  });

  it("reads an empty or malformed payload as an empty table", () => {
    const preview = reportPreviewFrom(artifact("report_preview", { columns: "nope", rows: 3 }));
    expect(preview.columns).toEqual([]);
    expect(preview.rows).toEqual([]);
    expect(preview.totals).toBeNull();
  });
});

describe("reportRunFrom", () => {
  it("names the run and what the tool knew about it", () => {
    const run = reportRunFrom(
      artifact("report_run", { runId: "rrun_1", reportKey: "revenue", reportName: "" }),
    );
    expect(run).toEqual({ runId: "rrun_1", reportKey: "revenue", reportName: "" });
    expect(reportRunFrom(artifact("report_run", {}))).toBeNull();
  });
});

describe("emailDraftFrom", () => {
  it("reads the subject, body and recipients", () => {
    const draft = emailDraftFrom(
      artifact("email_draft", {
        tool: "request_missing_docs",
        subject: "Missing POD",
        body: "Please send it.",
        to: ["ops@acme.test"],
        rationale: "Billing is blocked.",
      }),
    );
    expect(draft).toEqual({
      tool: "request_missing_docs",
      subject: "Missing POD",
      body: "Please send it.",
      to: ["ops@acme.test"],
      rationale: "Billing is blocked.",
    });
  });

  it("reads a single recipient and a missing one", () => {
    expect(
      emailDraftFrom(artifact("email_draft", { tool: "email_customer", to: "a@b.c" })).to,
    ).toEqual(["a@b.c"]);
    expect(emailDraftFrom(artifact("email_draft", { tool: "send_detention_notice" })).to).toEqual(
      [],
    );
  });
});

describe("entityCardFrom", () => {
  /*
   * A card stored before the projection holds the whole record the tool
   * returned. It is read by the same rules the server now applies: its name
   * first, its id, tenancy and version left out, nested structure left out
   * rather than shown as JSON, and a nested record read by its name.
   */
  it("reads a stored record by the projection's rules", () => {
    const card = entityCardFrom(
      artifact("entity_card", {
        entity: "shipment",
        record: {
          id: "shp_01M37R101VKZTB7TSKR30FJ0AT",
          organizationId: "org_01M37R101VKZTB7TSKR30FJ0AT",
          status: "InTransit",
          proNumber: "PRO-1",
          moves: [{ id: "mv_1" }],
          customer: { id: "cus_01M37R101VKZTB7TSKR30FJ0AT", name: "Acme" },
          createdAt: 1_790_187_600,
          version: 3,
        },
      }),
    );

    expect(card.entity).toBe("shipment");
    expect(card.fields.map((field) => [field.key, field.type, field.value])).toEqual([
      ["proNumber", "text", "PRO-1"],
      ["status", "status", "InTransit"],
      ["customer", "text", "Acme"],
      ["createdAt", "datetime", 1_790_187_600],
    ]);
    expect(card.fields.find((field) => field.key === "createdAt")?.label).toBe("Created");
  });

  it("reads the fields the server projected as they are", () => {
    const card = entityCardFrom(
      artifact("entity_card", {
        display: 1,
        entity: "insight",
        fields: [
          { key: "headline", label: "Headline", type: "text", value: "3 cards expire" },
          { key: "detectedOn", label: "Detected", type: "date", value: 1_790_187_600 },
          { key: "id", label: "ID", type: "text", value: "inst_01M37R101VKZTB7TSKR30FJ0AT" },
          { key: "severity", label: "Severity", type: "nonsense", value: "Critical" },
        ],
      }),
    );

    // An id or an unknown type is dropped even from a projected payload.
    expect(card.fields.map((field) => field.key)).toEqual(["headline", "detectedOn"]);
  });
});

describe("planFrom", () => {
  it("keeps the steps in order", () => {
    const plan = planFrom(
      artifact("plan", {
        title: "Cover the move",
        summary: "Assign then tell the customer.",
        stepCount: 2,
        steps: [
          { step: 2, proposalId: "ap_2", toolName: "email_customer", rationale: "" },
          { step: 1, proposalId: "ap_1", toolName: "assign_move", rationale: "Closest driver" },
        ],
      }),
    );
    expect(plan.steps.map((step) => step.step)).toEqual([1, 2]);
    expect(plan.stepCount).toBe(2);
  });
});

/*
A list or search result is the turn that most often earns a table. Its columns
arrive in the order the server's row projection declares them, because a JSON
object has none of its own, and every cell is read against its column, so a
row missing a field leaves a hole rather than shifting every value one column
left.

What a person reads of it is the projection: the columns they can reason with,
typed and labelled. A payload stored before the server projected it holds the
tool's own columns — ids, raw JSON and epoch seconds — and is projected here by
the same rules, so an old table reads as a new one does.
*/
describe("tableViewFrom", () => {
  const view = (payload: Record<string, unknown>) => tableViewFrom(artifact("table_view", payload));

  // The table the owner saw, stored before the projection existed.
  const storedInsights = {
    entity: "insights",
    columns: [
      "id",
      "category",
      "severity",
      "status",
      "subject",
      "headline",
      "narrative",
      "recommendation",
      "metrics",
      "links",
      "windowStart",
      "windowEnd",
      "detectedOn",
      "stale",
      "dismissReason",
    ],
    rows: [
      {
        id: "inst_01M37R101VKZTB7TSKR30FJ0AT",
        category: "Compliance",
        severity: "Critical",
        status: "Active",
        subject: "",
        headline: "3 workers' medical cards expire within 14 days",
        narrative: "Three drivers hold medical certificates that lapse this month.",
        recommendation: "Schedule DOT physicals this week.",
        metrics: [
          { direction: "HigherIsWorse", label: "Workers affected", unit: "Count", value: "3" },
          { direction: "LowerIsWorse", label: "First expiry in", unit: "Days", value: "0" },
        ],
        links: [{ count: 3, label: "Workers", path: "/hr/workers" }],
        windowStart: "2026-08-24 11:20 PDT (30 days ago)",
        windowEnd: "2026-09-23 11:20 PDT (today)",
        detectedOn: 1_790_187_600,
        stale: false,
      },
    ],
    rowCount: 1,
  };

  it("reads a stored finding without its id, its flags or its raw JSON", () => {
    const table = view(storedInsights);

    expect(table.columns.map((column) => [column.key, column.type])).toEqual([
      ["headline", "text"],
      ["category", "enum"],
      ["severity", "status"],
      ["status", "status"],
      ["narrative", "longText"],
      ["recommendation", "longText"],
      ["metrics", "metrics"],
      ["links", "links"],
      ["windowStart", "datetime"],
      ["windowEnd", "datetime"],
      ["detectedOn", "date"],
    ]);
    expect(table.columns.find((column) => column.key === "detectedOn")?.label).toBe("Detected");

    const [row] = table.rows;
    expect(row.values).not.toHaveProperty("id");
    expect(row.values.metrics).toEqual([
      { label: "Workers affected", value: "3", unit: "Count" },
      { label: "First expiry in", value: "0", unit: "Days" },
    ]);
    expect(JSON.stringify(row.values)).not.toContain("HigherIsWorse");
    // An insight has no page of its own, so its row opens nothing.
    expect(row.path).toBe("");
  });

  it("reads the columns the server projected as they are", () => {
    const table = view({
      display: 1,
      entity: "shipments",
      recordEntity: "shipment",
      columns: [
        { key: "proNumber", label: "Pro number", type: "text" },
        { key: "status", label: "Status", type: "status" },
      ],
      rows: [{ id: "shp_01M37R101VKZTB7TSKR30FJ0AT", proNumber: "P1", status: "InTransit" }],
    });

    expect(table.columns.map((column) => column.label)).toEqual(["Pro number", "Status"]);
    expect(table.rows[0].values).toEqual({ proNumber: "P1", status: "InTransit" });
    // The id is kept only to open the record, through the registry.
    expect(table.rows[0].path).toBe(
      "/shipment-management/shipments?expanded=shp_01M37R101VKZTB7TSKR30FJ0AT&panelType=edit&panelEntityId=shp_01M37R101VKZTB7TSKR30FJ0AT",
    );
  });

  it("opens nothing when the server said the rows have no page", () => {
    const table = view({
      display: 1,
      entity: "shipments",
      columns: [{ key: "proNumber", label: "Pro number", type: "text" }],
      rows: [{ id: "shp_01M37R101VKZTB7TSKR30FJ0AT", proNumber: "P1" }],
    });

    expect(table.rows[0].path).toBe("");
  });

  it("finds a stored list's record page from the list it came from", () => {
    const table = view({
      entity: "shipments",
      columns: ["id", "proNumber"],
      rows: [{ id: "shp_01M37R101VKZTB7TSKR30FJ0AT", proNumber: "P1" }],
    });

    expect(table.columns.map((column) => column.key)).toEqual(["proNumber"]);
    expect(table.rows[0].path).toContain("panelEntityId=shp_01M37R101VKZTB7TSKR30FJ0AT");
  });

  it("leaves a hole where a row has no value for a column", () => {
    const table = view({
      columns: ["proNumber", "customer", "status"],
      rows: [
        { proNumber: "P1", status: "InTransit" },
        { proNumber: "P2", customer: "Acme", status: "Delivered" },
      ],
    });

    expect(table.rows[0].values).toEqual({ proNumber: "P1", status: "InTransit" });
    expect(table.rows[1].values.customer).toBe("Acme");
  });

  // A projection can carry nested records. A cell has no layout for them and a
  // person does not need their structure, so the column is left out rather
  // than drawn as JSON.
  it("leaves out a column of nested records instead of printing their JSON", () => {
    const table = view({
      columns: ["proNumber", "stops"],
      rows: [{ proNumber: "P1", stops: [{ city: "Reno" }] }],
    });

    expect(table.columns.map((column) => column.key)).toEqual(["proNumber"]);
  });

  it("keeps false and zero, which are values and not absences", () => {
    const table = view({
      columns: ["billed", "weight"],
      rows: [{ billed: false, weight: 0 }],
    });

    expect(table.rows[0].values).toEqual({ billed: false, weight: 0 });
  });

  // The count is what the search found; the rows are what fitted in the
  // payload. A table showing 200 of 400 has to say so.
  it("knows it is showing less than was found", () => {
    const table = view({
      columns: ["proNumber"],
      rows: [{ proNumber: "P1" }, { proNumber: "P2" }],
      rowCount: 400,
    });

    expect(table.rowCount).toBe(400);
    expect(table.truncated).toBe(true);
  });

  it("is not truncated when every row came back", () => {
    const table = view({
      columns: ["proNumber"],
      rows: [{ proNumber: "P1" }],
      rowCount: 1,
    });

    expect(table.truncated).toBe(false);
  });

  // Reading a filtered page as the whole fleet is the mistake the footer
  // exists to prevent, so the terms come through with the rows.
  it("carries the terms the search applied", () => {
    const table = view({
      columns: ["proNumber"],
      rows: [{ proNumber: "P1" }],
      searchedFor: ["status is InTransit", "", 7],
    });

    expect(table.searchedFor).toEqual(["status is InTransit"]);
  });

  it("reads a payload that promises nothing as an empty table", () => {
    const table = view({});

    expect(table.columns).toEqual([]);
    expect(table.rows).toEqual([]);
    expect(table.rowCount).toBe(0);
    expect(table.truncated).toBe(false);
  });
});

/*
A sorted column puts what needs a person first and never leads with a blank:
statuses by how far off the happy path they are, figures and dates by value,
empty cells last in either direction.
*/
describe("sortTableRows", () => {
  const columns = [
    { key: "severity", label: "Severity", type: "status" as const },
    { key: "amount", label: "Amount", type: "money" as const },
  ];
  const rows = [
    { key: "0", path: "", values: { severity: "Info", amount: "10.00" } },
    { key: "1", path: "", values: { severity: "Critical" } },
    { key: "2", path: "", values: { severity: "Warning", amount: "250.50" } },
  ];

  it("orders a status by urgency", () => {
    const sorted = sortTableRows(rows, columns, { key: "severity", direction: "asc" });

    expect(sorted.map((row) => row.values.severity)).toEqual(["Critical", "Warning", "Info"]);
  });

  it("orders money by value and keeps an empty cell last either way", () => {
    expect(
      sortTableRows(rows, columns, { key: "amount", direction: "desc" }).map((row) => row.key),
    ).toEqual(["2", "0", "1"]);
    expect(
      sortTableRows(rows, columns, { key: "amount", direction: "asc" }).map((row) => row.key),
    ).toEqual(["0", "2", "1"]);
  });

  it("leaves the rows as found without a sort", () => {
    expect(sortTableRows(rows, columns, null).map((row) => row.key)).toEqual(["0", "1", "2"]);
  });
});
