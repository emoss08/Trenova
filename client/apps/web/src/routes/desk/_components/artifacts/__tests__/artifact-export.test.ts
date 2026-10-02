import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { tableViewCsv } from "../artifact-export";
import { tableViewFrom } from "../artifact-payloads";

const t = (message: string | null | undefined) => message ?? "";

function tableArtifact(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "aart_1",
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: "Shipments",
    payload,
    sourceToolCallId: "call_1",
    pinned: false,
    createdAt: 1,
    updatedAt: 1,
  } as AssistantArtifact;
}

describe("a table copied as CSV", () => {
  const artifact = tableArtifact({
    display: 1,
    tool: "list_shipments",
    entity: "shipments",
    recordEntity: "shipment",
    columns: [
      { key: "proNumber", label: "PRO number", type: "text" },
      { key: "status", label: "Status", type: "status" },
      { key: "weight", label: "Weight", type: "number" },
      { key: "hazmat", label: "Hazmat", type: "boolean" },
      { key: "notes", label: "Notes", type: "longText" },
    ],
    rows: [
      {
        id: "shp_01HX5",
        proNumber: "PRO-1",
        status: "InTransit",
        weight: 1240.5,
        hazmat: true,
        notes: 'Call before delivery, ask for "Sam"',
      },
      { id: "shp_01HX6", proNumber: "=PRO-2", status: "Delivered", weight: 7, hazmat: false },
    ],
    rowCount: 2,
    searchedFor: ["status is InTransit"],
  });

  it("writes the column labels and every row's values as a person reads them", () => {
    const csv = tableViewCsv(tableViewFrom(artifact), t);
    const lines = csv.split("\r\n");

    expect(lines[0]).toBe("PRO number,Status,Weight,Hazmat,Notes");
    expect(lines[1]).toBe('PRO-1,In transit,"1,240.5",Yes,"Call before delivery, ask for ""Sam"""');
    expect(lines).toHaveLength(3);
  });

  // A cell that a spreadsheet would run as a formula is text, and stays text.
  it("keeps text that starts like a formula from being evaluated, and leaves a hole where a row has no value", () => {
    const lines = tableViewCsv(tableViewFrom(artifact), t).split("\r\n");

    expect(lines[2]).toBe("'=PRO-2,Delivered,7,No,");
  });

  it("never writes a record's id, which is how the row links and not something to read", () => {
    const csv = tableViewCsv(tableViewFrom(artifact), t);

    expect(csv).not.toContain("shp_01HX5");
    expect(csv).not.toContain("id");
  });

  it("copies the rows it is given, so a filtered table copies what is shown", () => {
    const view = tableViewFrom(artifact);
    const csv = tableViewCsv({ ...view, rows: view.rows.slice(1) }, t);

    expect(csv.split("\r\n")).toHaveLength(2);
  });
});
