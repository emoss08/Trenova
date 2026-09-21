import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  emailDraftFrom,
  entityCardFrom,
  planFrom,
  reportPreviewFrom,
  reportRunFrom,
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
        rows: [
          { Customer: "Acme", Revenue: "120.50" },
          { Revenue: "9.00" },
        ],
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
    expect(emailDraftFrom(artifact("email_draft", { tool: "email_customer", to: "a@b.c" })).to).toEqual([
      "a@b.c",
    ]);
    expect(emailDraftFrom(artifact("email_draft", { tool: "send_detention_notice" })).to).toEqual([]);
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
