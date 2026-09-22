import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  emailDraftFrom,
  entityCardFrom,
  planFrom,
  reportPreviewFrom,
  reportRunFrom,
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
  it("lists the record's readable values and skips the structure", () => {
    const card = entityCardFrom(
      artifact("entity_card", {
        entity: "shipment",
        record: {
          id: "shp_1",
          proNumber: "PRO-1",
          status: "InTransit",
          moves: [{ id: "mv_1" }],
          customer: { name: "Acme" },
          createdAt: 1_700_000_000,
          version: 3,
        },
      }),
    );
    expect(card.entity).toBe("shipment");
    expect(card.id).toBe("shp_1");
    expect(card.facts.map((fact) => fact.key)).toEqual(["proNumber", "status", "createdAt"]);
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
A list or search result is the turn that most often earns a table, and for a
while it produced nothing at all: only a report preview, a report run and a
single-record get were mapped, so the pane promised a table and then sat empty
through "which shipments are in transit".

The columns arrive as names in the order the server's row projection declares
them, because a JSON object has none of its own. Everything here turns on
that: the cells are read against the declared order, so a row missing a field
leaves a hole rather than shifting every value one column left.
*/
describe("tableViewFrom", () => {
  const view = (payload: Record<string, unknown>) => tableViewFrom(artifact("table_view", payload));

  it("labels each column from its field name and keeps the declared order", () => {
    const table = view({
      columns: ["proNumber", "customerName", "status"],
      rows: [{ status: "InTransit", proNumber: "P1", customerName: "Acme" }],
    });

    expect(table.columns.map((column) => column.id)).toEqual([
      "proNumber",
      "customerName",
      "status",
    ]);
    expect(table.columns.map((column) => column.label)).toEqual([
      "Pro Number",
      "Customer Name",
      "Status",
    ]);
    // Read against the columns, not against the row's own key order.
    expect(table.rows).toEqual([["P1", "Acme", "InTransit"]]);
  });

  it("leaves a hole where a row has no value for a column", () => {
    const table = view({
      columns: ["proNumber", "customer", "status"],
      rows: [{ proNumber: "P1", status: "InTransit" }],
    });

    expect(table.rows).toEqual([["P1", null, "InTransit"]]);
  });

  // A projection can carry a nested value. The grid has no layout for one, so
  // it is drawn as its JSON rather than as "[object Object]".
  it("renders a nested value as text", () => {
    const table = view({
      columns: ["stops"],
      rows: [{ stops: [{ city: "Reno" }] }],
    });

    expect(table.rows).toEqual([['[{"city":"Reno"}]']]);
  });

  it("keeps false and zero, which are values and not absences", () => {
    const table = view({
      columns: ["billed", "weight"],
      rows: [{ billed: false, weight: 0 }],
    });

    expect(table.rows).toEqual([[false, 0]]);
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
